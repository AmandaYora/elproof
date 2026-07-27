# ADR-0005: Auth & session model

## Status
Accepted

## Context
The frontend mock has three distinct principal types — WO staff (Owner/Admin/Staff), client
(customer portal user), and platform admin (Super Admin/Support) — currently modeled as three
separate hardcoded credential lists with no token, no session, and no expiry. A real backend needs
one coherent session mechanism serving all three without merging their very different authorization
rules.

## Decision
- A single `identity` module owns authentication for **all** principal types. It does not own their
  profile data (name, role details, etc. — that stays in `staff`/`clients`/`platform` respectively);
  it owns only the `credentials` and `refresh_tokens` tables plus the login/refresh/logout use cases.
- `credentials` row shape: `id, tenant_id (nullable), principal_type (staff|client|platform_admin),
  principal_id (varchar — primitive reference, no FK), username, password_hash (bcrypt),
  role, display_name`. `principal_id`/`tenant_id` are stored as plain values, never as SQL foreign
  keys, per the modular-monolith no-cross-module-FK rule — `identity` does not know the internal
  schema of `staff`/`clients`/`platform`. (See "Update — email login" below: `email` was added to
  this row shape later.)
- Auth is JWT access + refresh: access token (short-lived, e.g. 15–30 min) carries
  `{principal_type, principal_id, tenant_id, role}` claims; refresh token is opaque, stored hashed
  in `refresh_tokens`, rotated on use, revocable on logout.
- A `shared/middleware` component verifies the access token's signature/expiry and injects claims
  into the request context — this is treated as a technical utility (crypto verification), not
  domain logic, so it lives in `shared/`, not inside `identity`. Issuing tokens (business rules:
  password check, lockout, rotation) stays inside `identity`.
- Passwords are bcrypt-hashed at rest; the plaintext demo arrays the frontend originally used
  (`shared/constants/demo-accounts.ts`) have been retired in favor of real, seeded, hashed
  credentials now that this module is live.

## Consequences
- Any module that needs "who is this staff member" beyond the JWT claims (e.g. their display name,
  full profile) must call the `staff`/`clients`/`platform` module's own contract — `identity` will
  not become a dumping ground for profile fields beyond the minimal `display_name` needed for
  immediate UI display right after login.
- Three principal types share one login endpoint (`POST /api/v1/auth/login`) that tries to resolve
  `username` against `credentials` regardless of type — this mirrors the frontend's existing
  sequential-check behavior in `LoginPage.tsx`, just moved server-side and made real.

## Update — Fase 10: a 4th principal type (`app`), issued without a `credentials` row

`payment`'s external mode (Payment Gateway as a Service — see `MODULE_PAYMENT.md` §7.1) introduces a
4th JWT principal type, `app`, for external SaaS consumers exchanging an `appId`+`secret` at
`POST /auth/app/token`. This principal deliberately does **not** fit the model this ADR originally
described:

- No `credentials` row is ever created or checked for it — `payment` verifies the appId+secret
  itself, against its own `payment_apps` registry (bcrypt hash), never touching `identity`'s
  `credentials` table at all.
- `identity.Contracts` gained one new method, `IssueServiceToken(ctx, principalType, principalID,
  ttl)`, which signs a bearer JWT for *any* caller-vouched-for principal — no password check, no
  `tenant_id`/`role` claims, and critically **no refresh token** (the App simply re-exchanges its
  secret once the token expires, rather than rotating a refresh token).
- This keeps `identity` "profile-agnostic" in spirit (it still never becomes a dumping ground for
  another module's domain data) but stretches "owns authentication for all principal types" to mean
  *signing*, not always *verifying a password against a stored credential* — a real, if narrow,
  widening of this ADR's original decision that's worth naming explicitly rather than leaving only
  implicit in `payment`'s own docs.
- `shared/middleware`'s verification side is unaffected — `Claims.PrincipalType` was always a plain
  string, so no schema change was needed there to accept `"app"` alongside the original three.

## Update — regular Admin/Staff `staff` principals didn't actually have a `credentials` row

This ADR describes `identity` as owning authentication for "all principal types," but until this
was found and fixed, that was only true in principle for `staff`: `staff.NewModule` never took an
`identity.Contracts` dependency at all, and `StaffService.Create` (the "Tambah Pengguna" flow) only
ever wrote the `staff_members` row — no `identity.CreateCredential` call. Only the tenant **Owner**
(provisioned separately by `platform`'s tenant-registration orchestration, which does call
`identity.CreateCredential`) could actually log in; every Admin/Staff account created afterward from
WO Console had no working credential at all.

Fixed by wiring `staff` to `identity.Contracts` (same plain one-way shape as `payment`→`identity`
above — see `MODULE_MAP.md`) and having `Create` provision a real credential the same way
`clients.Create` already did, including the same compensating rollback if credential creation fails
after the `staff_members` row committed. See `docs/API_CONTRACT.md`'s `staff` section for the
updated request/response shape.

## Update — email login (`credentials.email`, denormalized)

Login originally accepted only `username` (see the original Decision above). `credentials` gained
an `email VARCHAR(150) NULL` column (migration `000015`), populated at `CreateCredential` time by
every caller (mirrors how `username` itself was always denormalized here) and backfilled once for
accounts that existed before the column did — see the migration's own comment for why that one-time
backfill query is schema/data tooling, not an ongoing violation of the no-cross-module-join rule.

`Login`'s `username` request field now doubles as "username or email": tried as username first
(unambiguous, unique); if that doesn't resolve to a matching active credential whose password
checks out, tried again as an email. The JSON field name (`username`) and the generic
"Username/email atau password salah" error message were kept exactly this ambiguous on purpose —
same reasoning as usernames never being tenant-scoped: never reveal *which* identifier actually
existed.

### Update — email uniqueness (migration `000016`)

Shortly after shipping the above, a real production account was found with the same email shared
across three credentials (one deactivated+orphaned leftover from a client hard-delete, two genuinely
active accounts) — harmless for username (never happened, since username was always unique) but a
real problem for email once it can resolve a login: which of several matching accounts should
"win" is not something the system should ever have to guess. `CreateCredential` now rejects a
duplicate email up front (`apperror.Conflict`, mirrors the existing username check exactly), and
`credentials.email` itself gained a `UNIQUE KEY` (MySQL treats multiple `NULL`s as non-conflicting,
so accounts still missing a backfilled email aren't affected). `Login`'s `FindAllByEmail` +
try-every-candidate loop is kept as defensive code rather than simplified to "expect exactly one
row" — it's now effectively always ≤1 result, but there's no cost to leaving the safer, more
general implementation in place.

The three duplicate production accounts themselves were resolved by hand before this constraint
was added (one hard-deleted for being genuinely orphaned test data, one hard-deleted as unwanted
test data per the account owner, one kept as the sole holder of that email) — this migration adds
no automatic dedup logic of its own; if this ever recurs, it must be resolved the same
way (by hand) before the migration can apply, since a `UNIQUE KEY` cannot be added over existing
duplicates.
