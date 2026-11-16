# ADR-0047: The Client Portal gains two write paths (a transfer proof, a finished task) and a cover photo

## Status
Accepted — 2026-11-16. Product decisions approved by the owner: `docs/plan/revamp-client-portal/RISET.md` findings #4, #10, #11.
Contract: `docs/API_CONTRACT.md` › "Client Portal writes and cover". Schema: migrations `000077`–`000079`, `docs/DB_SCHEMA.md`.

## Context
Since Fase 6 the Client Portal was strictly read-only: `cmd/server/client_scope.go` allowed only GET, and `resolveProjectAccess` refused
every other method for a `client` principal. The research found three things the couple expects from a portal like this: a personal
hero (a photo instead of a plain navy block, #4), sending a transfer proof themselves instead of WhatsApp-ing it to the WO (#10), and
tasks of their own ("kirim daftar tamu", #11). Each needs data the WO owns, so the rules have to hold at the API, not in the portal UI.

## Decision
* **Cover photo (staff write, both read).** `projects.cover_photo_path` + `cover_updated_at`. `PUT/DELETE /projects/{id}/cover` need
  `project.edit` (the usual project scoping); `GET` streams it to staff of the project and to the project's own client without a portal
  permission. JPEG/PNG/WebP ≤ 6 MB, bytes checked (`shared/filetype`), then **always** re-encoded by `compress.Photo` (≤ 2000 px, no
  EXIF, 40 MP decode cap against decompression bombs; WebP becomes JPEG because Go has no WebP encoder). `coverUrl` carries
  `?v=<unix of the last change>`, so a short private cache is safe. The cover does not bump the project's version: it is not part of
  the PATCH surface, and an open project form must not turn stale because someone changed the photo.
* **Exactly four client writes.** The allowlist stays deny-by-default and names four routes on the client's own project:
  `POST /projects/{id}/uploads`, `PUT /projects/{id}/uploads/{token}`, `POST /projects/{id}/client-payments/submissions` (all
  `portal.finance.submit`) and `POST /projects/{id}/milestones/{mid}/complete` (`portal.tasks.complete`). The same list lives in
  `projects/presentation.clientWritePermission`; each handler also checks the principal type, so staff cannot use the client routes and
  clients cannot use any staff write. Two new client permissions, on for the built-in *Pengantin* role and off for *Keluarga* (data
  migrations add them to existing workspaces; the catalog test reads 000058 + 000078 + 000079).
* **A proof becomes an unverified client payment.** The submission reuses the ADR-0033 atomic path (uploads verified first; then one
  transaction: project lock, overpayment rule, payment + evidence rows + activity). The row carries `submitted_by_client_id`
  (`recorded_by_staff_id` stays NULL) and every payload says `source: "client" | "staff"`. The couple does not pick a payment type: the
  server chooses DP / Pelunasan / Termin from the ledger, and staff may change it before verifying. Staff verify and delete with the
  existing endpoints and permissions. Like a staff-recorded unverified payment, it counts on the ledger at once (the ledger has never
  depended on verification); the "Verifikasi bukti" queue and the "Dikirim pasangan" badge are where the WO checks it.
* **Tasks are flagged Timeline steps.** `project_milestones.for_client` (staff PATCH `forClient`) + `completed_by_client_id`. The couple
  may only complete a flagged, non-cancelled step; completing is idempotent; a staff status change clears the client mark. The portal
  home lists them as `clientTasks`.
* **Idempotency keys per actor.** The stored key's owner is the client on the client route (never shared between the bride and the
  groom), the staff member everywhere else.
* **Portal WhatsApp.** `contact.whatsapp` is the PIC's phone when usable, else the business number — the same rule the web already
  applied (`pickContactPhone`), now on the server too.
* **Billing state does not block the couple.** As before (K4), a WO's read-only or suspended subscription never restricts a client; what
  the couple sends simply waits for the WO.

## Consequences
* The portal is no longer read-only; any new client write must be added in both allowlists, behind its own `portal.*` permission, with a
  test in `client_scope_test.go` and `client_portal_test.go`.
* A WO that customised its *Pengantin* role gets the two new permissions on that role too (it kept its template key); it can remove them
  on *Peran & Akses*.
* Orphaned objects: a client may request uploads it never submits (same stance as staff, ADR-0033); a replaced or removed cover is
  deleted best effort.
* Tests: unit tests per rule (`domain/client_writes_test.go`, `application/client_writes_test.go`, `presentation/client_portal_test.go`,
  `shared/compress/photo_test.go`, `cmd/server/client_scope_test.go`), the isolation scenario `scenarios_portal_writes_test.go`, and the
  e2e `portal-writes.spec.ts` (WO side in the console, couple side through the API until the portal screens land).
