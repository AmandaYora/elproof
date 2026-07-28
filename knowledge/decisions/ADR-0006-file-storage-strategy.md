# ADR-0006: Evidence/file storage strategy

## Status
Accepted (superseded local-disk plan — see Revision below)

## Context
`shared/lib/evidence-file.ts` in the frontend always resolves to one of two static demo files
regardless of the `Evidence` record — real file upload/storage has never been built.

## Decision
- The `projects` module owns the `evidence` table (domain data: name, type, uploader, related
  entity, storage key). It does **not** own the file bytes themselves.
- A generic storage utility lives in `apps/api/internal/shared/storage` behind a small interface
  (`Save(ctx, key, data) (path string, err error)` / `Open(ctx, path) (io.ReadCloser, err error)`),
  so `projects` module code and the API contract never depend on which provider backs it.
- Download access always goes through an authenticated endpoint (same tenant/role scoping as the
  rest of `projects`) — never a public/world-readable object URL, even though the bucket itself is
  reachable directly by anyone with the object key.

## Revision (2026-07-16): object storage provider = IDCloudHost (S3-compatible)

The original decision defaulted to local disk per the anti-overengineering rule ("no cloud
infrastructure by default"). The user has since **explicitly** requested a specific provider —
this is an explicit choice, not a default, so the anti-overengineering rule does not apply here.

- **Provider:** IDCloudHost Object Storage, S3-compatible API, endpoint `is3.cloudhost.id`, bucket
  `elcodelabs` (bucket name **and endpoint are per-environment config**, not hardcoded — see
  `S3_ENDPOINT`/`S3_BUCKET`/`S3_ACCESS_KEY`/`S3_SECRET_KEY`/`S3_USE_SSL` in `.env`/`.env.example`).
- **Client library:** `github.com/minio/minio-go/v7` — purpose-built for S3-compatible non-AWS
  backends (IDCloudHost, MinIO, Ceph RGW, etc.), lighter dependency footprint than `aws-sdk-go-v2`
  for this single use case.
- **Object key convention (path):** `elproof/upload/{tenantId}/{projectId}/{category}/{filename}`
  — `category` is the evidence's `relatedKind` (`vendorMilestone`|`payment`|`projectVendor`|`issue`)
  lower-cased, `filename` is `{uuid}-{sanitizedOriginalName}` to avoid collisions while keeping the
  original name recognizable. Full key is stored on the `evidence.storage_path` column (already
  designed for this in `docs/DB_SCHEMA.md`) — never reconstructed from other columns at read time.
- **Credentials** live only in `.env` (gitignored) / the process environment — never hardcoded in
  source, never logged, never echoed back in any API response. `.env.example` only ever holds
  placeholder values.
- Secrets rotation is a single config change (`S3_ACCESS_KEY`/`S3_SECRET_KEY` in `.env`), same
  "one place, one restart" property the identity/JWT secret already has.

See ADR-0010 for the compression and transfer-encoding decisions that go with this (both sides
compress before/after the base64 HTTP transfer; storage always holds the already-compressed bytes).

## Consequences
- Object storage is now a hard external dependency for evidence upload/download — local dev needs
  real network access to `is3.cloudhost.id` (no local-disk fallback mode is maintained, to avoid two
  code paths silently drifting apart).
- `docs/DEPLOYMENT.md` must document the four `S3_*` env vars as required configuration for Fase 4
  onward (Fase 8 scope), same as `DATABASE_URL`/`JWT_SECRET` today.
- Because the object key already encodes `tenantId`/`projectId`, listing/auditing a tenant's or a
  project's files is possible via prefix listing on the bucket without needing a separate index —
  though the `evidence` table remains the source of truth for what a file *means* (name, type,
  uploader), matching the "module owns its own domain data" rule; the bucket only ever answers
  "what bytes does this key point to."

## Revision: second consumer — tenant logos (`platform` module, ADR-0012)

This ADR was originally scoped entirely to `projects`/evidence. Tenant branding (ADR-0012) reuses
the exact same `internal/shared/storage` utility for logo upload/download — confirming the utility
really is generic (not `projects`-specific) rather than something that needed duplicating:

- `platform/application` declares its own local `ObjectStorage` interface (identical shape to
  `projects/application`'s) and receives `*storage.Client` directly as a `platform.NewModule`
  constructor argument — a module can't import another module's `application` package to reuse its
  interface type, so this small duplication is intentional, not an oversight (same reasoning as any
  other contracts-only module boundary).
- **Different key shape, no `projectId`:** a logo isn't scoped to a project, so `BuildKey` is called
  as `BuildKey(tenantId, "0", "branding", filename)` — `"0"`/`"branding"` standing in for the
  project-scoped fields `BuildKey`'s signature still expects. `BuildKey` itself is unchanged.
- **Smaller size cap:** 2 MB decoded (vs. evidence's 15 MB) — a logo is a small header/sidebar
  image, not a document.
- Same byte-proxy download shape (`storage.Open` → `io.Copy`), same base64-JSON upload shape
  (ADR-0010) — no new transfer-encoding decision needed.
