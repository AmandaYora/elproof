# ADR-0005: Auth & session model

## Status
Accepted

## Context
The frontend mock has three distinct principal types — WO staff (Owner/Admin/Staff), client
(customer portal user), and platform admin (Super Admin/Support) — currently modeled as three
separate hardcoded credential lists with no token, no session, and no expiry (see PLAN.md §2.3).
A real backend needs one coherent session mechanism serving all three without merging their very
different authorization rules.

## Decision
- A single `identity` module owns authentication for **all** principal types. It does not own their
  profile data (name, role details, etc. — that stays in `staff`/`clients`/`platform` respectively);
  it owns only the `credentials` and `refresh_tokens` tables plus the login/refresh/logout use cases.
- `credentials` row shape: `id, tenant_id (nullable), principal_type (staff|client|platform_admin),
  principal_id (varchar — primitive reference, no FK), username, password_hash (bcrypt),
  role, display_name`. `principal_id`/`tenant_id` are stored as plain values, never as SQL foreign
  keys, per the modular-monolith no-cross-module-FK rule — `identity` does not know the internal
  schema of `staff`/`clients`/`platform`.
- Auth is JWT access + refresh: access token (short-lived, e.g. 15–30 min) carries
  `{principal_type, principal_id, tenant_id, role}` claims; refresh token is opaque, stored hashed
  in `refresh_tokens`, rotated on use, revocable on logout.
- A `shared/middleware` component verifies the access token's signature/expiry and injects claims
  into the request context — this is treated as a technical utility (crypto verification), not
  domain logic, so it lives in `shared/`, not inside `identity`. Issuing tokens (business rules:
  password check, lockout, rotation) stays inside `identity`.
- Passwords are bcrypt-hashed at rest; the plaintext demo arrays in the frontend
  (`shared/constants/demo-accounts.ts`) are retired in favor of seeded, hashed credentials once this
  module is live (tracked as a Fase 7 cleanup item in PLAN.md, so login keeps working in the interim
  via the same demo usernames/passwords, just checked server-side).

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
