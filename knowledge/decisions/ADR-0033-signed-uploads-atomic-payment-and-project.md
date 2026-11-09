# ADR-0033: Signed uploads, a payment recorded with its proof in one transaction, Idempotency-Key, a project created whole

## Status
Accepted — implemented 2026-11-09 (docs/plan/revamp-ui-shadcn/PLAN.md Tahap 4, endpoints E3, E7; E10 on the same routes).

## Context
Recording a payment was two to four calls: `POST` the payment, then one base64 `POST /evidence` per file, then reload.
A failure in the middle left a payment with no proof (the web needed a special "Pembayaran tersimpan, tetapi … gagal
diunggah" error class), a double tap or a dropped connection recorded the money twice, and every file travelled through
the API as base64 (a third larger than the file). Creating a project was five sequential calls (project, reload, read the
venue's prices, patch the whole project to attach the venue, put the schedule), each able to fail alone and leave a
half-made project.

## Decision

### Signed uploads (`POST /projects/{id}/uploads`)
- The browser asks for an upload (`{fileName, mimeType, size}`) and gets `{uploadToken, method:"PUT", url, headers,
  expiresAt, direct}`. It PUTs the file to `url`, then names `uploadToken` in the payment request. Same permission as the
  evidence upload (`project.edit` or `finance.record`); a `client` principal never reaches it (the portal is GET-only).
- **`direct`.** When object storage is reachable from browsers (`S3_PRESIGN`, `CanPresign()`), `url` is a presigned PUT of
  the bucket (`storage.Client.PresignPut`, minio-go `PresignedPutObject`) and the bytes skip the API. Otherwise `url` is the
  API's own `PUT /projects/{id}/uploads/{token}`, which stores the bytes with `storage.Save` after the same re-compression
  as every other upload (ADR-0010). One code path in the browser either way.
- **The token is stateless.** `<base64url(json)>.<base64url(HMAC-SHA256)>`, carrying workspace, project, object key,
  declared type, file name, size cap, expiry (15 minutes) and a random nonce. The signing key is derived from the JWT secret
  with a purpose label (`HMAC(secret, "elproof/projects/upload-token/v1")`): the application keeps **one** secret, and a
  login token can never be taken for an upload token or the other way round. Verification checks the signature first (constant
  time), then expiry, then that workspace and project are the caller's; a wrong one is a 422 on the field that carried it.
  Nothing is stored at grant time, so there is nothing to clean up for a grant nobody uses.
- **What may be uploaded** is narrow because the grant hands out a URL that writes to the bucket: JPEG, PNG, WebP, HEIC/HEIF,
  PDF, at most 15 MiB, a file name with no path characters. (The base64 route enforces no type list; it stays as it is.)
- Object keys are `elproof/upload/{tenant}/{project}/uploads/{uuid}-{name}` (`storage.BuildKey`), so a token's object is
  always under its own workspace and project.

### One transaction for payment + proof
- `POST …/client-payments`, `…/payments`, `…/venue-payments` accept `evidence: [{uploadToken, type, name?}]` (≤ 4; client:
  "Transfer Proof"; vendor/venue: "Invoice" and/or "Transfer Proof"). **Before** anything is written, every item is verified
  (token, type, not already used, the object is in storage — `Stat`, not read — within its cap and of the declared type); all
  problems of all items come back together as 422 `evidence[i]` / `evidence[i].uploadToken` / `evidence[i].type`.
- Then **one MySQL transaction**: lock the project row (`SELECT … FOR UPDATE`, so two money writes of one project run one
  after the other and each sees the other's payment in the overpayment check — which also closes a race the old flow had), apply
  the unchanged overpayment rule, insert the payment, insert one `evidence` row per file (same defaults as the two-call flow:
  name `Bukti Transfer - <ref>` / `Invoice - <ref>`, `documentDate` = payment date, uploaded by the actor, related kind/id = the
  payment), write the same activity entries, compute the ledger from those very rows, commit. Any error rolls the whole thing back.
- **The transaction is module-local.** Repositories read their executor from the context (`infrastructure.conn`), a
  `TxManager.InTx` puts the transaction there, and the projects module's repositories — payments, evidence, activity, projects,
  milestones, payment terms, vendor engagements — all use it. Everything touched belongs to `projects`, so no cross-module
  transaction is needed (and none is allowed): the only other module consulted during project creation (the venue, through
  `vendors/contracts`) is read before the transaction starts.
- **A token proves one payment.** `evidence.storage_path` is now UNIQUE (migration 67). The early "already used?" check is only
  a courtesy; the index is the guard. A request that loses a race inserts its payment, fails on the evidence row, and its
  transaction rolls back — covered by a concurrent integration test (8 requests, one token, exactly one payment and one proof).
  `PUT` through the API refuses a token an evidence row already owns (409), so a recorded proof cannot be overwritten.
- **Response**: the payment as the list returns it (`evidence: [{id,type,fileName,uploadedAt}]` is now on vendor and venue
  payments as well) plus `ledgerSummary`: `{contractValue, paid, remaining}` for the engagement (vendor) or the venue's rental +
  charge (venue); and for a client additionally `terms` — the server's own allocation, the same objects `GET /payment-terms`
  returns. `paid` is `AppliedToContract` (DP + Termin + Pelunasan − Refund, never negative), `remaining` is never negative
  (`domain.ComputeLedger`). Note this differs from the vendor list's `paidAmount`, which also counts `Tambahan`.

### Orphaned objects
A file uploaded but never claimed (the person walked away, the payment was refused) stays in the bucket. The stance:
- an object that is present but unusable (empty, above its cap, not the declared type) is deleted when a payment names it;
- after a refused payment (overpayment, validation) the object is **kept**, because the corrected retry should reuse the same
  token (valid 15 minutes) instead of making the person upload again;
- everything else is left. Cleanup, when wanted, is a job that lists objects under the `…/uploads/` category older than a day and
  deletes those no `evidence.storage_path` references. A plain bucket lifecycle rule cannot do it (claimed and unclaimed objects share
  the prefix and must not expire), so none is configured and no sweeper is built: an orphan costs bytes, never correctness, and no
  read path lists them.
- Direct (presigned) uploads skip the server-side image re-compression of ADR-0010 (the bytes never reach the API); the web still
  compresses before upload. Re-compressing at commit would put object reads inside the transaction, so it is not done.
- A presigned PUT URL stays usable until it expires (15 minutes) even after a payment claimed its object; only the person who was
  authorised to upload that proof holds it.

### `Idempotency-Key`
- Header on the three payment POSTs and on `POST /projects`. Table `idempotency_keys` (migration 66), owned by `projects`:
  `tenant_id, staff_id, route, idempotency_key, request_hash, status_code, response_body, created_at`, UNIQUE on
  `(tenant_id, idempotency_key, route)`; `route` is method + path, so project and call are part of the scope.
- Decision table (`application.DecideIdempotency`, unit-tested): no record → run; same person + same body + finished → replay the
  stored status/body with `Idempotent-Replay: true`; different body or person → 422 `idempotency_key_reuse`; same request still running
  → 409 `idempotency_in_progress`; same request "running" for more than 5 minutes (it died) → the key is re-claimed. The body hash is
  taken over canonical JSON (key order, whitespace and big integers are safe).
- **Only 2xx answers are stored.** A refused request (422, 403, …) releases its key, so fixing the form and resending with the same key is a new
  attempt rather than a "reuse" error. A handler that panics also releases it. Retention is 24 hours, deleted opportunistically on each
  reservation (no scheduler). Malformed keys (not 8–128 of `[A-Za-z0-9_-]`) are 422 on `Idempotency-Key`. No header, no change in behaviour.
- Known limit: the response is stored after the business transaction commits, in a second step. If that write itself fails the money
  is recorded but no replay exists; a retry within 5 minutes gets 409, a later one would run again. (Storing it inside the business
  transaction would remove this; it was left out to keep the wrapper independent of the four handlers.)

### Atomic project creation (E7)
- `POST /projects` also takes `venueId`, `venueRentalPrice`, `venueCharge`, `terms[]`. `ProjectService.CreateAtomic` validates
  everything first (dates, PIC, venue, venue prices, every term row through the unchanged `buildTerms`, count, sum ≤ contract) and
  returns **one** 422 with every key (`eventDate`, `prepStartDate`, `picStaffId`, `venueId`, `venueRentalPrice`, `venueCharge`, `terms`,
  `terms[i].label|amount|dueDate`); nothing is created on any problem. Then one transaction creates the project (the INSERT now carries the
  venue columns), its default milestones and the schedule. Trial quota and permissions are unchanged; the quota is checked after validation.
- The venue is resolved through `vendors/contracts` (new `GetVenueForAttach`: name + current prices, kept apart from the public-safe
  `VenueSummary` so prices never reach clients); a venue of another workspace is "not found". Prices left out default to the venue's
  current ones — the project's own snapshot (ADR-0016).
- The response is the project exactly as before (`venue` stays the text) plus `terms` and `venueSummary` (its own key: `venue` is taken).
- `POST /projects/{id}/vendors` derives `categoryId` (the vendor's) and `eventDate` (the project's) when omitted; if sent they must equal
  them (422 on that field). PATCH is unchanged, so a vendor's own date can still drift from the project's after an edit.
- Every date that is not `YYYY-MM-DD` on create/update/duplicate is now reported under its own key (before: always `eventDate`) — E10.

## Consequences
- The web records a payment or creates a project with one business call, with a retry that cannot double-book, and per-field errors from the server.
- Old clients keep working: the evidence route, the old bodies and the old responses are unchanged (responses only gained fields).
- The projects module owns a transaction mechanism; any future multi-step action inside the module can use `Transactor.InTx` — cross-module
  orchestration still goes through module clients, outside transactions.
- A sweeper for unclaimed objects under the upload category can be added later without touching the API.
- A deployment without presign (local dev) serves uploads through the API and is limited by the API's 15 s read timeout for large files;
  presigned storage is not.
