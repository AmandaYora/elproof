# ADR-0024: Sign in with Google, self-service WO signup, free-trial caps

## Status
Accepted — implemented 2026-11-06 (PLAN.md Fase 2, decisions K1, K2, K5, K6, K9).

## Context
ElProof had no self-service registration: every WO tenant was typed in by ElProof's team, who
also chose the Owner's password. A SaaS meant for many WOs needs a signup a WO can finish alone,
without ElProof handling passwords or e-mail delivery, and a trial that cannot be abused.

## Decision
- **Sign in with Google** for WO Owners and staff (`identity`). The browser uses Google Identity
  Services (popup) and only ever holds an ID token; the API verifies it itself in
  `identity/infrastructure/google_verifier.go`: RS256 signature against Google's JWKS (cached per
  `Cache-Control`, unknown-kid refetch throttled to once a minute), `aud` = our client ID, issuer,
  expiry, and `email_verified`. No client secret, no redirect URI. `credentials.google_sub`
  (unique) links a login to a Google account; `password_hash` is NULL for Google-only logins.
- `POST /auth/google`: only a login already linked to this Google account signs in. A staff
  login that merely has the same email is **never linked silently** — staff emails are typed by a
  WO Owner and anyone can be an Owner, so a planted staff account could otherwise capture a
  stranger's Google sign-in. The response is `linkRequired`, and `POST /auth/google/link`
  links only after that login's ElProof password is confirmed (failure budget applies). Anything
  else — no login, or a client/platform-admin login with that email — is `signupRequired`.
  Fase 3 invitations replace password confirmation for new staff.
- **Workspace signup** (`POST /public/signup/workspace`, `platform`): the same ID token plus WO
  name and consent. An email held by a login *without* Google never blocks signup (no squatting);
  the new Owner login is then created without the email (Owners never sign in by email). Consent evidence (terms version `domain.TermsVersion`, time, IP, user agent,
  optional marketing opt-in) is stored in `signup_requests`. Provisioning crosses modules, so it is
  a **resumable saga**: the signup row is written first and records the tenant as soon as it
  exists (tenant insert and signup link in one transaction); seeds (vendor categories, Timeline
  Default) insert only what is missing, the Owner staff row and login are created only if absent
  (picking a fresh internal username if the saved one was taken meanwhile). The run is detached
  from the HTTP request (30 s budget), so a closed tab cannot stop it half-way. A retry — pressing the button again or signing
  in with Google again — finishes the same workspace. Runs per Google account are serialised by
  an in-process lock (single instance) backed by a guarded SQL update. Nothing is usable before
  the Owner's login exists. Rate limit: 10 attempts per IP per hour.
- **Free-trial caps** (K6): only trialing tenants are limited (staff besides the Owner, active
  projects, active client accounts — default 3/5/10, single row `trial_limits`, editable by Super
  Admin and Billing at `/trial-limits`). Paid plans are unlimited. Each owning module counts its
  own rows and asks `platform` for the cap through a local `TrialQuota` port wired in
  `cmd/server` (no cross-module import). Checked on create, duplicate, reactivate and unarchive
  (a soft cap: two simultaneous creates may both pass); refusals
  carry `errors.code = trial_limit_reached`.
- **Onboarding**: Owner-only `/tenants/me/profile` (WO name, WhatsApp normalised to `62…`, city),
  `/tenants/me/logo`, `/tenants/me/onboarding` (checklist + trial usage) and
  `/tenants/me/onboarding/dismiss`.
- `apperror.AppError` gained an optional machine-readable `Code`, sent as `errors.code`.
- Not done, by decision K15: no tenant slug — custom domains were discontinued, so a slug had no
  use.

## Consequences
- `GOOGLE_CLIENT_ID` defaults to ElProof's own (public) client; the Google Cloud consent screen
  must be "In production" before real WOs sign up. `GOOGLE_JWKS_URL` (tests sign their own
  tokens) is accepted only for a loopback URL outside production; anything else stops the server.
- Staff created before Fase 3 keep their passwords; invitation-based Google-only staff accounts
  arrive in Fase 3.
- The signup lock and rate limits are in memory (single instance, as ADR-0011 assumes).
