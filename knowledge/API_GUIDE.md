# API Guide

> API conventions specific to this project (on top of the `/api/v1` standard).

Base rules live in `.claude/rules/api-standard.md` (versioning, envelope, pagination meta). This
file adds ElProof-specific conventions.

## Auth header

Every endpoint except `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`, and the
`/api/v1/public/*` endpoints requires:
```
Authorization: Bearer <access_token>
```
The `shared` auth middleware verifies the token and injects `principal_type`, `principal_id`,
`tenant_id` (nullable), `role` into the request context. Tenant-scoped modules read `tenant_id`
from context — never from a request parameter or body.

`/api/v1/public/*` is registered outside `authed` by each owning module's `RegisterPublicRoutes`
(`platform`: login slides, signup; `billing`: `GET /public/plans`, the homepage price list) and
returns only what an anonymous visitor may see.

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

## Editing: partial PATCH, `If-Match`, explicit state (ADR-0036)

* `PATCH` on a project, milestone, project vendor, vendor milestone, kendala, the three payments and a document changes **only the keys present**; `null` clears
  only nullable fields; `{}` is a 422 on `errors.body`.
* Every item and list item of those resources carries `updatedAt` (RFC 3339, microseconds, UTC) and item responses set `ETag`. A PATCH may send
  `If-Match: "<updatedAt>"`: the write is a compare-and-set, and a stale version is a **409** `{ errors: { _conflict: ["stale"] }, data: { current } }` with nothing written.
  Without the header the write is last-write-wins (Tahap 9 decides on 428). Any write to a row bumps its version, so reload an item after something else changed it.
* State is set, never flipped: `PATCH /projects/{id}` `{isArchived}`, `PATCH …/evidence/{id}` `{isClientVisible}`, `PATCH /vendors|venues|vendor-categories|clients|staff/{id}` `{isActive}`
  alone. Saying the state it already has is a 200 and writes nothing. The old `POST …/toggle-*` routes are gone (platform-console toggles remain).

## Full endpoint list

See [`docs/API_CONTRACT.md`](../docs/API_CONTRACT.md) — kept in sync with the implemented backend
as each module lands.

## External-facing integration guide (Fase 10)

The `/auth/app/token` + `/external/payments/*` routes (`payment` module, external mode) are meant
for **other SaaS products'** engineering teams, not ElProof's own frontend — they get a dedicated,
audience-appropriate guide instead of just an API_CONTRACT.md entry:
[`docs/PAYMENT_INTEGRATION_GUIDE.md`](../docs/PAYMENT_INTEGRATION_GUIDE.md), plus a ready-to-import
Postman collection at `docs/postman/ElProof-Payment-Gateway.postman_collection.json`.
