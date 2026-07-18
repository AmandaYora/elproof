# MODULE_PAYMENT.md — the `payment` module: one merchant wallet, many consumers

> **Note on this document's history:** this spec is referenced by section number (`§3`, `§7.5`,
> etc.) from dozens of code comments across `apps/api/internal/modules/payment/**` and
> `apps/api/internal/modules/identity/**`, written while Fase 9 was implemented — but the file
> itself was never actually saved to the repo until now (reconstructed from those comments plus
> `PLAN.md`'s Fase 9/10 sections and the code's own final shape, which is the ground truth
> wherever a comment and the code could conceivably have drifted). Section numbers below were
> chosen to match what the existing comments already cite, not renumbered for a cleaner read.

## 1. Why this module exists

ElProof needs to accept real money (WO tenants paying for their subscription). Rather than wiring
a payment gateway directly into `billing`/`platform`, this module wraps **one merchant credential**
(today: Tripay) behind a small internal API, so:

- The gateway credential lives in exactly one place, encrypted at rest.
- Any number of **consumers** ("Apps") can create charges through that one credential without ever
  touching it directly.
- Fase 9 has exactly one consumer: `platform`, calling in-process as Go code (an "App internal").
- Fase 10 opens the same wallet to consumers outside this codebase entirely — other SaaS products,
  over HTTP, authenticated with their own `appId`+`secret` (an "App eksternal").

## 2. Scope

### 2.2 What this module is explicitly NOT

- **Not a business ledger.** It never records "who owes what" — that's `billing`'s
  `subscription_transactions` (or whatever ledger a future App internal owns). This module only
  ever answers "can I charge this amount through the active gateway, and what happened to charge
  X" — see §4.
- **Not aware of tenants, plans, or any other business concept.** `contextID` in `contracts.Client`
  is opaque — the module never inspects it, just threads it through for the caller's own use (Fase
  9 doesn't even populate it; `platform` doesn't need to).
- **Not a multi-provider abstraction the caller sees.** Callers (Apps) never pick "Tripay" vs
  "Midtrans" — there's exactly one active provider at a time, chosen by a platform_admin in
  Konfigurasi Gateway (§4). Apps just call `CreateCharge`; which gateway actually processes it is
  this module's own concern.

## 3. Structure and dependencies

```
apps/api/internal/modules/payment/
  contracts/       — the ONLY package other modules may import (Client, Dispatcher, WebhookConsumer)
  domain/          — provider-agnostic value objects (App, GatewayConfig, Channel, Charge, WebhookEvent)
  infrastructure/  — PaymentService (the one concrete type), MySQL repositories, Tripay adapter, crypto
  presentation/    — HTTP handlers: webhook, gateway-config admin CRUD, App registry admin CRUD,
                     /auth/app/token, /external/payments/*
  payment.module.go
```

**No `application/` layer.** Unlike every other module, `payment` has no business process to
separate from its infrastructure — `infrastructure.PaymentService` directly implements both
`contracts.Client` (what an App internal calls) and `contracts.Dispatcher` (what only this
module's own composition-root wiring and presentation layer call). There is no domain invariant
here complex enough to warrant a use-case layer between the two.

**Dependencies:** `payment` depends on `identity/contracts` only (Fase 10 — to mint bearer tokens
for external Apps; see §7.1). It depends on nothing else, and is built early in `main.go` so any
App internal (`platform`) can receive its `Client` at their own construction time. Every provider
adapter (`tripay.go` today; a future `midtrans.go`/`xendit.go`) implements the unexported `gateway`
interface in `infrastructure/gateway.go` — never part of the public `contracts` package, so
swapping/adding a provider never touches a caller.

## 4. Data model

None of the four tables are a ledger — see §2.2.

- **`payment_gateway_config`** — single row by application convention (`id=1`, upserted, not a DB
  constraint), holding the one active provider's credential, AES-256-GCM-encrypted
  (`tripay_api_key_encrypted`, `tripay_private_key_encrypted`) with the key derived from
  `PAYMENT_ENCRYPTION_KEY` (§8). `active_provider IS NULL` means "simulation mode" — every
  `Client` method returns a validation error rather than silently succeeding.
- **`payment_apps`** — the App registry. One row per consumer allowed to create charges.
  `kind = 'internal'` (in-process, e.g. `platform-billing`) or `'external'` (HTTP, Fase 10).
  External-only columns: `secret_hash` (bcrypt, verifies inbound token exchange) and
  `secret_encrypted` (AES-GCM via the same cryptor as gateway credentials, decrypted only to sign
  outbound webhook relays — see §7.5). An App internal is usually seeded once, idempotently, on
  every server startup (`EnsureInternalApp`) rather than through any endpoint — there is currently
  exactly one (`platform-billing`).
- **`payment_charge_dispatch`** — a thin `order_ref -> app_id` (+ `provider_ref`) index, primary
  keyed on `order_ref`. This is the **entire** idempotency mechanism for charge creation (§7.3):
  attempting to reuse an `order_ref` fails fast, before ever calling the gateway a second time.
  Also carries `expires_at` (the charge's own deadline, recorded at creation) and `resolved_at`
  (NULL until a terminal outcome — paid/expired/failed/refund — has been dispatched to the owning
  App, via webhook or the reconciliation sweep, see §6 step 6) — the one exception to "never a
  ledger": these are completion markers, not amounts or history.
- **`payment_webhook_events`** — `(provider, event_id)` uniqueness gives webhook processing
  idempotency: the same gateway callback delivered twice is a no-op the second time, not a double
  application of the same payment.

## 5. The internal-mode contract (`contracts.Client`, `contracts.Dispatcher`)

`Client` is what an App internal calls directly, in-process, at its own call sites:

```go
type Client interface {
    Enabled(ctx) (bool, error)
    QuoteFee(ctx, channel string, amount int64) (int64, error)
    CreateCharge(ctx, appID, contextID, orderRef string, amount int64) (*ChargeResult, error)
    CreateChannelCharge(ctx, appID, contextID, orderRef string, amount int64, channel string, opts ChargeOptions) (*ChargeResult, error)
    ListChannels(ctx) ([]Channel, error)
    CheckStatus(ctx, providerRef string) (*ChargeResult, error)
}
```

`Dispatcher` is used only by this module's own composition root and presentation layer — never by
an App:

```go
type Dispatcher interface {
    RegisterConsumer(appID string, consumer WebhookConsumer)
}
```

An App internal that wants to be notified when its charges are confirmed implements
`WebhookConsumer.ApplyWebhookEvent(ctx, orderRef string, event WebhookEvent) error` and registers
itself once, at composition-root time — see §6 and `knowledge/MODULE_MAP.md`'s "Same pattern reused
for `payment`'s webhook dispatch" note for exactly how `main.go` bridges `payment` and `platform`
without either importing the other's package.

## 6. The webhook flow (gateway → this module → the right App)

1. The gateway (Tripay) POSTs to the one, permanent `POST /webhooks/payment` route — this URL never
   changes as new Apps register, internal or external.
2. The route is deliberately unauthenticated (no JWT — a gateway can't carry a bearer token).
3. Trust comes entirely from `X-Callback-Signature`: HMAC-SHA256 of the raw request body, keyed
   with the merchant's private key, checked with a constant-time comparison (see §8).
4. Once the signature verifies, the event is parsed into the provider-agnostic `WebhookEvent` shape
   and logged into `payment_webhook_events` — a duplicate delivery of an already-seen
   `(provider, event_id)` is a no-op from here on (idempotent).
5. The event's `order_ref` is looked up in the Charge Dispatch Index to find which App owns it.
   Before dispatching, a non-`unpaid` status first has to win `ClaimUnresolved` — an atomic
   `UPDATE ... WHERE resolved_at IS NULL` on that dispatch row. Losing the claim (`resolved_at`
   already set) means the reconciliation sweep (step 6 below) — or an earlier delivery of this same
   webhook — already dispatched a terminal outcome for this charge, so this call is a no-op. Winning
   it dispatches exactly once:
   - `kind = internal` → the App's registered `WebhookConsumer.ApplyWebhookEvent` is invoked
     in-process, same request (Fase 9).
   - `kind = external` → the event is relayed over HTTP to the App's `callback_url`, HMAC-signed
     with that App's own secret (decrypted via `secret_encrypted`), fire-and-forget: one attempt,
     short timeout, no retry (Fase 10). A relay that fails (network blip, App down) is **not**
     retried and does not fail the webhook handler back to the gateway — the external App's
     fallback is polling `GET /external/payments/charges/{orderRef}/status` (§7.2), which always
     reflects the gateway's own source of truth regardless of whether the relay arrived. Note this
     specific failure mode (delivery attempted, network drop reaching the App) is **not** covered by
     step 6's reconciliation sweep either — the claim is already consumed by the time delivery is
     attempted. Only a genuinely *missed or never-received* webhook is covered.
6. **Reconciliation — the safety net for a webhook ElProof itself never received at all**
   (`PaymentService.ReconcilePending`, run on a ticker — `PAYMENT_RECONCILE_INTERVAL`, default 3m,
   see `StartReconciler`). Every dispatch still `resolved_at IS NULL` and older than a couple of
   minutes (no point checking something that was just created) is re-checked directly against the
   gateway (`CheckStatus`, the same `GET /transaction/detail` Tripay call the status endpoints
   already use), batched (50/tick) and oldest-first:
   - A definitive terminal answer (paid/expired/failed/refund) is dispatched exactly like a webhook
     would be — same `ClaimUnresolved` guard, same consumer/relay switch, so downstream behavior
     is identical regardless of whether the outcome was discovered via webhook or by polling.
   - Still `unpaid`, or the status check itself errored (gateway/network trouble) — left alone,
     *unless* it's more than an hour past its own `expires_at` (or, for the small number of rows
     that predate this column, 24 hours past `created_at`), in which case it's force-resolved as
     expired locally rather than left open forever. This bounds every dispatch to eventually reach a
     terminal state, regardless of the gateway's or the network's behavior.
   - Genuinely still-open charges (unpaid, not yet past grace) are simply left for the next tick.

   This exists because, before it, the *only* paths that ever updated a pending charge's status were
   the webhook itself and (for the internal subscription flow) a client-side poll in the open
   browser tab that stops the moment the tab/modal closes — a charge whose webhook silently never
   arrived, combined with the customer closing the tab, had no path back to a resolved state at all.

## 7. External mode (Fase 10) — Apps outside this codebase

### 7.1 Authentication: `POST /auth/app/token`

An external App exchanges `{appId, secret}` for a short-lived bearer access token:

- `appId` is looked up in the App registry; must be `kind=external`, `is_active=true`, and have a
  `secret_hash` set.
- `secret` is bcrypt-compared against `secret_hash`.
- On success, `payment` asks `identity` to sign an access token for principal type `app`,
  principal id = the App's `appId` — `identity.Contracts.IssueServiceToken(ctx, "app", appID, ttl)`.
  `identity` never sees the App's secret and never touches its own `credentials` table for this —
  it only ever signs the JWT (see ADR-0005: identity stays profile-agnostic). Token lifetime is
  `APP_TOKEN_TTL` (default 1h, `.env`).
- **No refresh token exists for this principal type.** An App simply repeats the appId+secret
  exchange once its access token expires — there is no rotation/revocation-list machinery to build
  for a credential-exchange flow this simple.
- Rate-limited per IP (in-memory sliding window, `shared/middleware/ratelimit.go` — no
  Redis/Memcache, per `.claude/rules/monorepo.md`) to blunt secret brute-forcing.

### 7.2 The three external routes

All three require a valid `app`-principal token AND a **live** check that the App is still active
(re-queried from `payment_apps` on every single request — never cached off the JWT, and never just
trusted because the token itself is still cryptographically valid). This is deliberate: deactivating
an App from Platform Console must sever its access immediately, not wait for its current token to
expire.

| Method | Path | Notes |
|---|---|---|
| POST | `/external/payments/charges` | body `{orderRef, amount, channel?, customerName?, customerEmail?, customerPhone?}`. `appId` always comes from the token, never the body. Duplicate `orderRef` → `409 conflict` (§7.3), no duplicate charge ever created upstream. |
| GET | `/external/payments/charges/{orderRef}/status` | Only returns a charge if the Charge Dispatch Index says this `orderRef` belongs to the calling App — another App's `orderRef` is `404 not_found`, never leaked as "exists but not yours". |
| GET | `/external/payments/channels` | Same live channel list `Client.ListChannels` returns internally. |

### 7.3 Idempotency

`payment_charge_dispatch.order_ref` is the primary key — the **entire** idempotency guarantee. A
create-charge call first checks whether `orderRef` is already dispatched; if so, it returns
`409 conflict` immediately, without ever calling the gateway a second time. The check-then-insert
has a race window (two concurrent requests for the same fresh `orderRef`), closed by the database's
own uniqueness constraint: a losing insert is translated to the same `409 conflict` rather than a
raw SQL error.

### 7.4 (reserved — fee quoting)

`QuoteFee`/`Channel.QuoteFee` compute Tripay's flat+percentage fee shape; not part of the critical
charge-creation path, available to any caller (internal or external) that wants to show a fee
estimate before charging.

### 7.5 Secret storage

An external App's secret is stored **two ways**, for two different purposes, and the plaintext
itself is shown to the platform_admin creating/resetting it exactly once, then discarded:

- `secret_hash` (bcrypt) — one-way, verifies an inbound `/auth/app/token` exchange. Can never be
  used to recover the plaintext.
- `secret_encrypted` (AES-256-GCM, same `cryptor` as gateway credentials, key from
  `PAYMENT_ENCRYPTION_KEY`) — reversible, decrypted only at the moment of signing an outbound
  webhook relay (§6 step 5), so the relay's `X-Webhook-Signature` can be verified by the receiving
  App using the same secret it was given at registration.

### 7.6 Error codes

External routes (`/auth/app/token`, the three `/external/payments/*` routes) return the usual
`.claude/rules/api-standard.md` envelope, with an **additional** machine-readable `code` inside
`errors` so another team's code can branch on it without parsing `message` — a fixed vocabulary:
`bad_request` / `unauthorized` / `forbidden` / `not_found` / `conflict` / `rate_limited` /
`internal`. This is purely additive: every other route in the app keeps returning `errors` exactly
as before (validation field-errors or `null`); only these external-facing handlers also populate
`{"code": "..."}`.

## 8. Security notes

- **Gateway credentials and external-App-secret-encrypted-copies share one AES-256-GCM cryptor**,
  keyed from `PAYMENT_ENCRYPTION_KEY` (SHA-256 of the raw env value → a stable 32-byte key
  regardless of the operator-supplied string's length) — the one secret this module accepts living
  outside the database. Never rotate this key without a plan to re-encrypt every stored value.
- **No caching on status/deactivation checks** — `loadGateway` re-reads `payment_gateway_config` on
  every call, and `IsAppActive` re-reads `payment_apps` on every external request. A platform_admin
  disabling the gateway or an App takes effect on the very next request, not eventually.
- **Webhook signature verification is constant-time** (`hmac.Equal`), guarding against timing
  attacks on the comparison.
- **Write-only credential fields.** `GetConfig`/`ListApps` never return a secret/API key/private
  key, encrypted or otherwise — only booleans (`hasTripayApiKey`, etc.) or nothing at all. Updating
  a credential with an empty string means "leave the existing stored value unchanged," never
  "clear it" — an admin re-submitting the form without retyping every secret can't accidentally
  wipe it.

## 9. Fase boundary

- **Fase 9 (internal mode)** — everything in §1–§6 except the `kind=external` branch of §6 step 5;
  exactly one App (`platform-billing`), called in-process. Webhook confirmation from a real Tripay
  sandbox/production account is the one checkpoint that needs real external credentials to verify
  (see `PLAN.md` Fase 9 notes) — everything reachable without them (charge creation attempting and
  failing honestly against dummy credentials, encrypted-at-rest storage, config CRUD) is verified.
- **Fase 10 (external mode)** — §7 in full, plus the `kind=external` relay branch of §6 step 5, plus
  Platform Console's "Manajemen Aplikasi" page (list Apps, register a new external App, reset its
  secret, toggle active/inactive — internal Apps can never be toggled off through this page, since
  that would sever ElProof's own subscription billing). Purely additive on top of Fase 9's
  infrastructure — registering an external App, or that App creating charges, never changes how
  `platform` (the internal App) behaves.
- **Reconciliation (post-Fase-10 hardening, no phase number of its own)** — §6 step 6 above. Added
  after production Tripay credentials were configured, in response to a gap found during an
  architecture review: nothing previously reconciled a charge whose webhook was never received —
  Tripay itself has no cancel/void endpoint (confirmed against its own public docs) and no
  `CANCELLED` status, so an abandoned charge's only ordinary path to resolution was its own
  `expires_at` plus a webhook ElProof might never actually receive. Verified end-to-end against a
  local test database with a deliberately-invalid gateway credential (so every status check fails,
  the worst case): a dispatch created hours in the past and long past its grace period is correctly
  force-resolved and dispatched to both an internal consumer and an external App's relay; a
  still-open dispatch is correctly left untouched across repeated sweep runs; and a legitimate
  webhook arriving for an order_ref the sweep already resolved is correctly treated as a no-op
  (confirmed via a captured HTTP relay log showing exactly one delivery, not two).
