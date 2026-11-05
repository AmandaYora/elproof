# ADR-0022: Access boundaries and the subscription lifecycle (Fase 1)

## Status
Accepted — implemented 2026-11-06 (`docs/plan/registrasi-wo-self-service/PLAN.md`, Fase 1).

## Context
An audit before opening self-service registration found that access rules were not enforced
where it matters:
- A `client` principal was limited only to GET, so it could read its project's vendor/venue
  payments, internal activity log and every document; `GET /clients?projectId=<any>` leaked other
  couples' contact data because the role check only special-cased `Staff`/`Sales`.
- `subscription_status` `expired`/`expiring_soon` were never written and `is_suspended` was never
  checked, so no tenant was ever restricted. Deactivated staff/clients could still log in.
- Login had no rate limit and the per-IP limiter trusted a spoofable `X-Forwarded-For`.

Opening self-service registration on top of that would have meant free, unlimited use.

## Decision
1. **Deny-by-default gates, composed once in `cmd/server`** (the only place that knows every
   module's routes), all inside the `authed` chain:
   `RequireAuth → AuditTrail(platform_admin) → RestrictPrincipalBy(platform_admin) →
   RestrictPrincipal(client) → TenantAccessGate → handler`.
   - `client_scope.go`: the complete list of GET routes the Client Portal may call.
   - `platform_scope.go`: Platform Console role policy + mandatory two-factor (ADR-0023).
   - `tenant_scope.go`: what staff may still do when access is restricted.
   Owning modules keep their own checks (defense in depth): `projects` allowlists client
   sub-resources, filters evidence per document and redacts internal fields; `clients` requires a
   staff principal.
2. **Subscription lifecycle is computed, not stored.** `platform/domain.ComputeLifecycle` derives
   status and access from dates and `DefaultSubscriptionPolicy` (trial 14 d, warn 7 d, grace 7 d,
   read-only 23 d): `trialing/active → expiring_soon → past_due (full) → read_only → suspended
   (billing only)`. Data is never deleted automatically (product decision K10). The
   `subscription_status` column is a cache refreshed hourly by a sweeper for list filters only.
3. **`TenantAccessGate`** resolves the level per request (platform cache, 1-minute TTL,
   invalidated on pay/activate/freeze): `read_only` blocks staff writes, `billing_only` leaves only
   subscription/plan/export routes, `blocked` (platform freeze) stops everyone. The Client Portal is
   never restricted by billing state (K4). Refusals carry `errors.code`
   (`subscription_read_only`, `subscription_suspended`, `tenant_frozen`) that the web client
   handles centrally (`http-client.ts`).
4. **Login hardening (identity):** per-IP and per-identifier (failures) limits; `ClientIP` prefers
   `X-Real-IP` (overwritten by host nginx); frozen tenants cannot log in/refresh (`TenantGuard`
   port implemented by `platform`); deactivation and password reset revoke refresh tokens; empty
   email stored as NULL.

## Consequences
- Adding a Client Portal call requires adding it to `client_scope.go` (a test lists the whole
  surface).
- Staff restrictions are visible in the WO Console through `GET /tenants/me/subscription`
  (banner and locked screen in `AppLayout`).
- The in-memory caches/limiters assume one instance (ADR-0011); scaling out needs shared state.

## Revision (2026-11-06): independent security review
- **Live sessions:** `middleware.RequireLiveSession` (first in the authed chain) re-checks every
  staff/client/admin token against its credential — active, same role, two-factor still on —
  through identity's `SessionChecker` (30 s cache, invalidated immediately on deactivate, role
  change, password reset and two-factor changes). A demoted or deactivated account loses access
  at once instead of after the 30-minute token lifetime; refused tokens get `session_revoked`.
- **Login limits per account:** failed passwords (10) and TOTP codes (5) are budgeted per
  credential id inside `AuthService` for 15 minutes, so spelling variants (case/accents, username
  vs email) and fresh challenges share one budget; the per-IP limit stays in the handler. Auth
  bodies are capped (16 KB), identifiers at 254 chars, and limiter keys are swept when expired.
- **Proxy headers:** `ClientIP` trusts `X-Real-IP`/`X-Forwarded-For` only when the direct peer is
  loopback or a private address (host nginx / Docker bridge).
- **Access cache:** entries carry an invalidation generation, so a read that began before a
  freeze can no longer re-cache the un-frozen state.
- **Dates:** `subscription_expires_at` is DATETIME (migration 000053; old dates become 23:59:59 of
  the same day); `DaysLeft` counts calendar days (0 = ends today).
- Time zone: Go reads and writes with `loc=Local` and all lifecycle math happens in Go; keep the
  app container's TZ fixed when deploying.

