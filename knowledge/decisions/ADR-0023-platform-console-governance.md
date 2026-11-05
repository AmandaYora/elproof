# ADR-0023: Platform Console governance — roles, two-factor, audit trail

## Status
Accepted — implemented 2026-11-06 (PLAN.md Fase 1, decision K14).

## Context
The Platform Console can see every tenant, grant free subscriptions and change the payment
gateway, yet "Super Admin" and "Support" had identical power, there was no second factor, no record
of who did what, and manual activation recorded neither a reason nor the acting admin.

## Decision
- **Roles** (`platform_admins.role`): Super Admin (everything), Billing (plans, plan↔App mapping,
  manual activation), Support (register/edit tenants, reset Owner credential), Read-only (view).
  Enforced centrally in `cmd/server/platform_scope.go` for every module's routes; reads are open
  to all roles, unknown writes default to Super Admin only.
- **Mandatory TOTP two-factor** (RFC 6238, SHA-1/30 s/6 digits, ±1 step) in `identity`: secret
  encrypted with AES-GCM (key derived from `JWT_SECRET`), replay blocked by the last accepted step.
  Password login returns an opaque in-memory challenge; `POST /auth/mfa/verify` completes it. The
  JWT carries `mfa`; without it an admin can only reach enrolment (`/auth/mfa/setup|enable`).
  Enabling revokes older sessions. Super Admin can reset a lost authenticator.
- **Audit trail**: new `audit` module owning `audit_log`; `middleware.AuditTrail` records every
  state-changing platform-admin request, applied or refused (actor, role, action, path, status,
  IP, user agent) — never request bodies. `GET /audit-log` is Super Admin only.
- **Accountable manual activation**: reason + proof reference required (note for "other"),
  stored with the acting admin in `platform.manual_activations`; reset endpoints no longer echo
  passwords.

## Consequences
- Rotating `JWT_SECRET` invalidates enrolled authenticators (admins re-enrol).
- The seed `superadmin` must enrol two-factor at first login.
- An audit-log UI and richer per-domain events (before/after values) are planned for Fase 6.
