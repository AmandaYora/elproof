# API Guide

> API conventions specific to this project (on top of the `/api/v1` standard).

Base rules live in `.claude/rules/api-standard.md` (versioning, envelope, pagination meta). This
file adds ElProof-specific conventions.

## Auth header

Every endpoint except `POST /api/v1/auth/login` and `POST /api/v1/auth/refresh` requires:
```
Authorization: Bearer <access_token>
```
The `shared` auth middleware verifies the token and injects `principal_type`, `principal_id`,
`tenant_id` (nullable), `role` into the request context. Tenant-scoped modules read `tenant_id`
from context — never from a request parameter or body.

## Path conventions

- Resource paths are plural, tenant scope is implicit via the token (not in the URL): e.g.
  `GET /api/v1/projects`, not `GET /api/v1/tenants/{id}/projects`.
- Nested resources under a project use the project ID as a path segment matching the frontend's own
  tab structure: `GET /api/v1/projects/{id}/vendors`, `/milestones`, `/payments`, `/issues`,
  `/evidence`, `/activity`.
- Platform Console endpoints are prefixed by their module, not by "platform": `GET
  /api/v1/tenants`, `GET /api/v1/plans`, `GET /api/v1/subscription-transactions`, `GET
  /api/v1/platform-admins` — the "Platform Console" is a frontend grouping, not an API namespace.

## Errors

Validation errors map to `{success:false, message, errors: {field: [messages]}}` (object keyed by
field, matching what Zod-shaped forms on the frontend can render directly under each input).
Domain errors (not found, forbidden, conflict) map to standard HTTP status with a flat `message`.

## Full endpoint list

See [`docs/API_CONTRACT.md`](../docs/API_CONTRACT.md) — kept in sync per phase as `PLAN.md` phases
land, not written all at once.

## External-facing integration guide (Fase 10)

The `/auth/app/token` + `/external/payments/*` routes (`payment` module, external mode) are meant
for **other SaaS products'** engineering teams, not ElProof's own frontend — they get a dedicated,
audience-appropriate guide instead of just an API_CONTRACT.md entry:
[`docs/PAYMENT_INTEGRATION_GUIDE.md`](../docs/PAYMENT_INTEGRATION_GUIDE.md), plus a ready-to-import
Postman collection at `docs/postman/ElProof-Payment-Gateway.postman_collection.json`.
