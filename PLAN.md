# PLAN — Venue Payments + Pembayaran tab restructuring into sub-tabs

## 0. Goal

Add a third payment ledger, "Pembayaran ke Venue" — venue costs today are only a static per-project
snapshot (`venueRentalPrice`/`venueCharge`), with no transaction history at all, the same gap vendor
payments had before being fixed. Stacking a third section vertically on the "Pembayaran" tab would be
unwieldy, so that tab is restructured into three sub-tabs — "Uang Masuk dari Client" / "Pembayaran ke
Vendor" / "Pembayaran ke Venue" — each its own URL, reusing the app's existing tab-navigation pattern
rather than inventing a new one.

## 1. Decisions confirmed

1. **Venue payments follow Vendor's evidence model, not Client's.** Two optional slots (Invoice,
   Bukti Transfer) — a `Refund` needs only Bukti Transfer, every other type needs both — because
   money going out to a venue has the same two-sided paper trail as money going out to a vendor
   (what they invoiced, proof the WO paid it). This is also a return to an earlier reality: before
   ADR-0016, Venue *was* a vendor category, so this isn't inventing a new pattern, it's restoring the
   one Venue already had.
2. **`venue_payments` mirrors `client_payments`' shape, not `vendor_payments`' — no intermediate
   engagement table.** A project has at most one venue (`project.venue_id`), so a venue payment ties
   directly to `project_id`, the same way a client payment does (no `project_vendor_id`-equivalent
   column needed).
3. **Sub-tabs are real nested routes** (`/pembayaran/client`, `/pembayaran/vendor`,
   `/pembayaran/venue`), reusing the existing `TabNav` component as-is — confirmed by reading it that
   it's hard-wired to react-router-dom (`NavLink`-based active state, not a controlled/local-state
   prop), and confirmed there is no existing precedent anywhere in this codebase for a
   non-URL-driven sub-tab. Matching the app's own established "every view is a URL" convention was
   judged more valuable than a smaller, one-off local-state tab widget.
4. **Dashboard's "incomplete payments" widget merges vendor and venue candidates**, labeled by
   source, rather than staying vendor-only or becoming two separate widgets.
5. **`ComputeProgress`'s `incompleteEvidenceCount`** (feeds the project's On Track/Attention/At Risk
   condition) **also folds in venue payments**, for the same consistency reason — the two ledgers are
   now identically modeled, so one counting toward project health and the other not would be an
   arbitrary inconsistency.
6. **Margin/Keuntungan and Client Portal are untouched.** Margin already derives venue cost from the
   static snapshot, never from a payment ledger (matches how it never used vendor's `paidAmount`
   either) — no change needed. Client Portal shows neither vendor nor venue payment data today
   (business-confidential WO cost management) — venue payments don't change that boundary.

## 2. Current state (verified precisely)

- `ProjectPaymentsTabPage.tsx` (17 lines) stacks `<ClientPaymentsSection>` then
  `<ProjectPaymentsSection>` vertically inside one `<div>` — no sub-navigation, no venue reference at
  all today.
- `TabNav.tsx`: `items: {to, label, end?}[]`, renders `NavLink`s — active state is computed
  internally by react-router-dom matching the URL, not a prop this component exposes. Used in exactly
  two places today (`ProjectDetailLayout.tsx`'s top-level project tabs, `ClientPortalLayout.tsx`'s own
  top tabs) — both are single-level, page-top, URL-routed tab bars. No sub-tab precedent exists.
- `protected.routes.tsx`'s `/projects/:projectId` children: `pembayaran` is a single flat route
  entry (`{ path: "pembayaran", element: <ProjectPaymentsTabPage /> }`) with no children of its own.
- `domain/payment.go`'s `PaymentEvidenceStatus(evidences []Evidence)` and
  `IsPaymentEvidenceComplete(p VendorPayment, hasInvoice, hasProof map[int64]bool)` — both currently
  hardcoded to `RelatedPayment` (the vendor kind) and to the `VendorPayment` struct type respectively.
  Three call sites depend on this exact signature today: `payment_endpoints.go`'s `listPayments`,
  `project_service.go`'s `ComputeProgress`, `dashboard_service.go`'s `Get`.
- `mysql_dashboard_repository.go`'s `ListPaymentCandidates` queries only `vendor_payments` (JOIN
  `projects` + `project_vendors`) into `domain.DashboardPaymentRow{Payment VendorPayment, ProjectName
  string, VendorID int64}` — `DashboardStats.IncompletePayments []DashboardPaymentRow` is the one
  field the dashboard's widget reads today.
- `evidence.related_kind` (5 values today): `vendorMilestone`, `payment`, `projectVendor`, `issue`,
  `clientPayment`. A new venue payment cannot reuse `payment` — `venue_payments.id` and
  `vendor_payments.id` are independent `AUTO_INCREMENT` sequences that can collide in value, so
  sharing a `relatedKind` would make `relatedId` ambiguous between the two tables.
- `api-endpoints.ts`'s `projects` group already has `payments`/`clientPayments`/`evidence` as
  `(id) => `/api/v1/projects/${id}/<segment>`` functions — a `venuePayments` entry fits the same
  convention directly.

## 3. Design

### 3.1 Database

New migration `apps/api/migrations/000030_create_venue_payments_table.up.sql`:

```sql
CREATE TABLE venue_payments (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  project_id BIGINT UNSIGNED NOT NULL,
  type ENUM('DP','Termin','Pelunasan','Tambahan','Refund') NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  payment_date DATE NOT NULL,
  method VARCHAR(100) NOT NULL,
  reference_number VARCHAR(100) NOT NULL,
  notes TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_venue_payments_project FOREIGN KEY (project_id) REFERENCES projects (id),
  INDEX idx_venue_payments_project (project_id)
);

ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone', 'payment', 'projectVendor', 'issue', 'clientPayment', 'venuePayment') NOT NULL;
```

No `invoice_evidence_id`/`proof_evidence_id` columns on `venue_payments` itself — same lesson already
learned from `vendor_payments`' now-removed dead columns: evidence-completeness is computed via the
polymorphic `evidence` table from day one here, never a direct stored FK.

`.down.sql`: narrow `related_kind` back to 5 values, then `DROP TABLE venue_payments`.

### 3.2 Backend — domain

`domain/payment.go`: add
```go
type VenuePayment struct {
	ID              int64
	ProjectID       int64
	Type            PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}
```
(identical shape to `ClientPayment` — no per-vendor column, same reasoning as §1.2).

**Generalize the two shared helpers** so both `vendor_payments` and `venue_payments` reuse one
definition instead of a third copy-pasted variant:
```go
func PaymentEvidenceStatus(evidences []Evidence, kind EvidenceRelatedKind) (hasInvoice, hasProof map[int64]bool) {
	...
	if e.RelatedKind != kind { continue }
	...
}

func IsPaymentEvidenceComplete(paymentType PaymentType, paymentID int64, hasInvoice, hasProof map[int64]bool) bool {
	if paymentType == PaymentRefund { return hasProof[paymentID] }
	return hasInvoice[paymentID] && hasProof[paymentID]
}
```
Note `IsPaymentEvidenceComplete` now takes the two primitives (`Type`, `ID`) rather than a
`VendorPayment` struct — `VenuePayment` is a different Go type, and passing the primitives avoids
introducing an interface just to satisfy two field reads. **All three existing call sites must be
updated** to pass `domain.RelatedPayment` explicitly and `p.Type, p.ID` instead of `p`:
`payment_endpoints.go:listPayments`, `project_service.go:ComputeProgress`,
`dashboard_service.go:Get`.

`domain/evidence.go`: add `RelatedVenuePayment EvidenceRelatedKind = "venuePayment"`.

`domain/dashboard.go`: add, additively (existing `DashboardPaymentRow`/`IncompletePayments` stay
exactly as they are — no change to the vendor path)
```go
type DashboardVenuePaymentRow struct {
	Payment     VenuePayment
	ProjectName string
}
```
and a new `DashboardStats.IncompleteVenuePayments []DashboardVenuePaymentRow` field alongside the
existing `IncompletePayments`. Kept as two separate slices (not one merged/discriminated array) so
the already-working vendor path is untouched — the frontend widget merges the two into one labeled
list at render time (§3.9).

### 3.3 Backend — repository

New `mysql_venue_payment_repository.go` — `ListByProject`, `FindByID`, `Create` — same shape as
`mysql_client_payment_repository.go` (the closest existing sibling), reading/writing
`venue_payments`.

`mysql_dashboard_repository.go`: new `ListVenuePaymentCandidates(ctx, tenantID)
([]domain.DashboardVenuePaymentRow, error)` — mirrors `ListPaymentCandidates` exactly, minus the
`project_vendors` JOIN and `VendorID` column (venue payments have no vendor):
```sql
SELECT vp.id, vp.project_id, vp.type, vp.amount, vp.payment_date, vp.method, vp.reference_number, vp.notes, p.name
FROM venue_payments vp
JOIN projects p ON p.id = vp.project_id
WHERE p.tenant_id = ?
ORDER BY vp.payment_date DESC, vp.id DESC
```

### 3.4 Backend — application

New `venue_payment_service.go` — `VenuePaymentRepository` interface, `VenuePaymentService` (`List`,
`Create`), `VenuePaymentInput` — copies `client_payment_service.go`'s exact shape. `Create` logs
activity reusing `domain.ActivityPaymentRecorded` with `entityType: "venue_payment"`, description
"Pembayaran venue dicatat" — same reuse pattern `client_payment_service.go` already established (no
new `ActivityType` value needed).

`project_service.go`'s `ComputeProgress`: after the existing vendor-payments block, add the same
shape for venue payments —
```go
venuePayments, err := s.venuePayments.ListByProject(ctx, projectID)
if err != nil { return nil, err }
vHasInvoice, vHasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedVenuePayment)
for _, p := range venuePayments {
	if !domain.IsPaymentEvidenceComplete(p.Type, p.ID, vHasInvoice, vHasProof) {
		incompleteCount++
	}
}
```
(the `evidences` slice is already fetched once for the project; reused for both kinds, just filtered
differently per call — no second `s.evidence.List` call needed). `ProjectService` gains a
`venuePayments VenuePaymentRepository` constructor dependency, wired the same way `payments
PaymentRepository` already is.

`dashboard_service.go`'s `Get`: after the existing vendor-payments block, fetch
`s.repo.ListVenuePaymentCandidates(ctx, tenantID)`, cross-reference the same way (per-distinct-project
evidence fetch, `domain.RelatedVenuePayment` this time), and append incomplete rows to the new
`stats.IncompleteVenuePayments`. `DashboardService` needs no new constructor dependency — it already
holds `evidence *EvidenceService` and `repo DashboardRepository`.

### 3.5 Backend — presentation

New `venue_payment_endpoints.go`: `listVenuePayments` (mirrors the vendor version's
evidence-cross-reference-then-map pattern, using `domain.RelatedVenuePayment`), `createVenuePayment`
— routed in `handler.go`'s existing big `Item` switch as `rest[0] == "venue-payments"`, alongside the
existing `payments`/`client-payments` cases.

`dto.go`: new `venuePaymentResponse` (identical shape to `clientPaymentResponse`, plus the
`evidenceComplete bool` computed field like `paymentResponse`), `toVenuePaymentResponse`.

`dashboard_endpoint.go`: extend the response DTO with `incompleteVenuePayments`, alongside the
existing `incompletePayments` field — read the current file during implementation to match its exact
existing field-naming/casing convention.

### 3.6 Frontend — data layer

- `apps/web/src/modules/projects/schemas/venue-payment.schema.ts` (new) — identical shape to
  `client-payment.schema.ts` plus the two file fields `payment.schema.ts` has (`invoiceFile?: File`,
  `proofFile?: File`), importing the shared `PAYMENT_METHOD_OPTIONS` from `payment.schema.ts`.
- `types.ts`: new `VenuePayment` interface (same shape as `ClientPayment` plus `evidenceComplete:
  boolean`).
- `api-endpoints.ts`: `venuePayments: (id: string) => \`/api/v1/projects/${id}/venue-payments\`` next
  to `payments`/`clientPayments`.
- `useProjectStore.ts`: `venuePayments: VenuePayment[]` state, `fetchVenuePayments`,
  `createVenuePayment` — the latter a three-step orchestration identical to the just-fixed
  `createPayment` (POST payment, then independently-caught Invoice/Bukti Transfer uploads with
  `relatedKind: "venuePayment"`), throwing a new `VenuePaymentEvidenceError` on partial failure —
  same shape as `VendorPaymentEvidenceError`.

### 3.7 Frontend — `VenuePaymentsSection.tsx` (new component)

Structurally `ProjectPaymentsSection.tsx` minus the vendor picker (like `ClientPaymentsSection.tsx`
has no picker) — one venue per project, nothing to choose. Three summary stats: **Nilai Sewa Venue**
(`project.venueRentalPrice + project.venueCharge`), **Total Sudah Dibayar** (Σ venue_payments, Refund
subtracted), **Sisa Pembayaran** (difference) — same 3-stat layout as vendor's, relabeled. "Tambah
Pembayaran" modal has the same fields as vendor's `AddPaymentModal` (Jenis, Nominal, Tanggal, Metode
dropdown, No. Referensi optional, Invoice + Bukti Transfer, Catatan) minus the Vendor picker.

**No-venue-attached state**: if `project.venueId` is null, render an `EmptyState` ("Belum ada venue
terpasang untuk project ini — pasang venue terlebih dahulu di tab Venue") instead of the summary
cards/table — there is nothing to record a payment against.

### 3.8 Frontend — tab restructuring

`ProjectPaymentsTabPage.tsx` becomes a small layout (mirrors `ProjectDetailLayout.tsx`'s own
`TabNav` + `Outlet` shape, one level deeper): renders a `TabNav` with three items —
```tsx
const PEMBAYARAN_TABS = [
  { to: "client", label: "Dari Client" },
  { to: "vendor", label: "Ke Vendor" },
  { to: "venue", label: "Ke Venue" },
];
```
— above an `<Outlet context={{projectId}} />` (re-passing the same `ProjectDetailContext` shape
through, since `useOutletContext` reads from the nearest ancestor regardless of nesting depth).

`protected.routes.tsx`: `pembayaran` gains children, mirroring the existing top-level shape exactly:
```tsx
{
  path: "pembayaran",
  element: <ProjectPaymentsTabPage />,
  children: [
    { index: true, element: <Navigate to="client" replace /> },
    { path: "client", element: <PembayaranClientTabPage /> },
    { path: "vendor", element: <PembayaranVendorTabPage /> },
    { path: "venue", element: <PembayaranVenueTabPage /> },
  ],
},
```
Three new thin tab-page files under `pages/tabs/` (`PembayaranClientTabPage.tsx`,
`PembayaranVendorTabPage.tsx`, `PembayaranVenueTabPage.tsx`), each the same one-line shape as every
other existing `XxxTabPage.tsx` — pull `projectId` via `useOutletContext`, render exactly one section
component. Default/index sub-tab is `client`, keeping today's "money in, then money out" top-to-bottom
reading order as the left-to-right tab order too.

### 3.9 Frontend — Dashboard widget

Wherever `DashboardPage` currently renders `incompletePayments` (to be located precisely during
implementation — not yet read in this planning pass), extend it to also map over
`incompleteVenuePayments`, merging both into one list sorted by date, each row labeled by source
("Vendor: {vendorName}" vs "Venue") rather than kept as two separate visual widgets — satisfying the
confirmed "gabungkan keduanya... dengan label yang membedakan sumbernya" decision without needing a
backend-side merge (§3.2 keeps the two arrays separate on purpose).

## 4. Order of implementation

1. Migration `000030` (table + evidence enum widen). Apply locally, confirm.
2. Backend domain generalization (§3.2) — update all 3 existing call sites for the new
   `PaymentEvidenceStatus`/`IsPaymentEvidenceComplete` signatures before anything else, so the
   codebase compiles at every intermediate step.
3. Backend repository + application + presentation for venue payments (§3.3-3.5). `go build`/`go vet`.
4. Frontend data layer + `VenuePaymentsSection.tsx` (§3.6-3.7).
5. Frontend tab restructuring (§3.8) — confirm the existing `ClientPaymentsSection`/
   `ProjectPaymentsSection` still render correctly once moved one route level deeper (Outlet context
   propagation).
6. Frontend Dashboard widget merge (§3.9).
7. `tsc --noEmit`, `npm run build`.
8. Manual/smoke-test pass (§5).
9. Sync `docs/DB_SCHEMA.md` (new `venue_payments` table entry, widened `evidence.related_kind`),
   `docs/API_CONTRACT.md` (new `venue-payments` endpoint, note the nested `pembayaran/*` routes),
   `knowledge/MODULE_MAP.md` (projects module's sub-entity table count).

## 5. Verification

- Project with no venue attached: "Pembayaran ke Venue" sub-tab shows the empty state, no crash, no
  "Tambah Pembayaran" button.
- Attach a venue, record a venue payment with both Invoice and Bukti Transfer — confirm
  `evidenceComplete: true`, and that this does NOT flip any *vendor* payment's own completeness (the
  two ledgers' evidence must stay fully independent despite sharing the generalized helper).
  Symmetric check the other way (a vendor payment's evidence doesn't leak into venue completeness).
- Confirm `incompleteEvidenceCount` on the project's progress increases when an incomplete venue
  payment exists, exactly as it already does for vendor payments.
- Confirm the Dashboard's incomplete-payments list shows both a vendor and a venue row, each legible
  as to which is which.
- Navigate directly to `/projects/{id}/pembayaran/venue` via a fresh page load (not a client-side tab
  click) — confirm it renders correctly (tests that `Outlet` context propagation and the nested-route
  lazy-loading both work, not just the happy path of clicking through).
- Confirm `/projects/{id}/pembayaran` (no sub-segment) redirects to `.../client`, preserving today's
  default view.
- Confirm Margin/Keuntungan is completely unchanged before/after this feature ships (still reads only
  the static snapshot, never the new ledger).
- `go build ./...`, `go vet ./...`, `npx tsc --noEmit -p apps/web/tsconfig.json`,
  `npm run build -w apps/web`.
