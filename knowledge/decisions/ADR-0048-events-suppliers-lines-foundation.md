# ADR-0048: Events, supplier engagements and line items as the foundation of packages, quotations and projects

## Status
Accepted — 2026-11-19. Product decisions approved by the owner: `docs/plan/paket-wo/PLAN.md` (findings #1–#27) and its design brief
`docs/plan/paket-wo/DESAIN.md`. Lands in five stages (PLAN §8); **Tahap 1 (directory & catalog) and Tahap 2 (events, vendors & venues in a project) are implemented**, Tahap 3–5 follow.
Schema: migrations `000084`–`000086`, `docs/DB_SCHEMA.md`. Contract: `docs/API_CONTRACT.md` › "Catalog (event kinds, packages)" and
"Quote settings". Partly supersedes ADR-0013, 0014, 0016, 0018, 0019, 0020, 0034 and 0036 as each stage lands (each carries a note).

## Context
ElProof knew one business shape: the client pays the WO one typed number (`projects.contract_value`), and the WO pays vendors and at
most one venue. Real WOs sell **packages** ("Paket Resepsi 500 tamu") that cover several **events** (akad, resepsi, …), price extra
guests by steps, and mix three models: the WO pays every supplier (all-in), the client pays suppliers directly (organizer only), or a
mix. Capacity matters per event and per seating layout, and the client must never see what the WO pays its suppliers. ElProof is
pre-launch with dummy data, so the old shape is replaced, not patched.

## Decision
* **Three shared building blocks**, used the same way by quotations (`quotation_*`) and projects (`project_*`):
  **events** (kind, date, venue, layout, sessions with guests), **supplier engagements** (a vendor or venue, `paid_by` WO or Client,
  optional internal `agreed_cost`) and **line items** (package, its components, extra guests, items, bonus, discount). The three
  business models come from `paid_by`; there is no project type.
* **Pure rules live once, in `offers/domain`** and never read the database: `ValidateRule`, `ExtraGuests` /
  `ExtraGuestsFor` (`ceil((guests − included) / step) × stepPrice`, nothing at or below the included guests), `LineTotals` (PPN 12 % ×
  DPP 11/12 for a PKP workspace, **rounded down**; a total below zero is refused), `CheckCapacity` (guests per session against the
  venue's capacity for the event's layout, with a message naming both numbers). Projects call them through a port with their own data.
* **The server computes, the web shows.** Running totals while typing come from a preview endpoint (`POST /packages/preview` in Tahap 1,
  `…/lines/preview` later) that runs the same functions and writes nothing; the web has no second price formula.
* **Venue: two capacities and three prices.** `capacity_standing` / `capacity_seated`, at least one required and ≥ 1;
  `rental_price` became `price_akad`, `price_resepsi`, `price_akad_resepsi`; `charge` stays. Import/export follow.
* **Catalog per workspace.** `event_kinds` (six built-ins seeded at sign-up and by migration for existing WOs, matched by a stable
  `default_key` so a renamed built-in is not added again; deactivate instead of delete while a package covers it), `packages` with
  covered kinds, optional included guests and extra-guest rule, an optional default venue, and ordered components (free text, optional
  vendor and category, optional internal `estimated_cost`).
* **Quote settings on the workspace.** `document_code` (2–5 capital letters or digits, derived from the business name; for a workspace
  that predates the column it is filled on first read), `is_pkp`, default validity days (14) and default terms text.
* **Supplier cost is internal.** `estimated_cost` (catalog) and `agreed_cost` (engagements) are read and written only with
  `finance.margin.view`; an edit by someone without it keeps the stored cost.
* **Deleting directory rows that offers refer to.** A vendor, venue or vendor category referenced by a package cannot be deleted
  permanently (409 with `vendor_in_offers` / `venue_in_offers` / `category_in_offers`); deactivate it instead.

## Decisions taken while building Tahap 1
* A package's default venue is checked against the venue's **largest** capacity (a package has no layout yet); the event's layout is
  checked again when the package is used.
* Packages may be deleted permanently until quotations exist (Tahap 5); from then on a package a quotation refers to is deactivated.
* Reads of the catalog are open to every staff member (a salesperson picks from it); writes need `package.manage`, and so do the
  Katalog menu items for packages and event kinds.
* "Dipakai di" on event kinds counts packages; the count of quotations waits for Tahap 5.

## Consequences
* Tahap 2–5 remove `projects.contract_value`, `package_name`, the venue columns, `project_vendors`, `vendor_payments` and
  `venue_payments`; each stage backs up `elproof_db` before it is deployed.
* Every new table carries `tenant_id NOT NULL`, every unique key includes it, and another workspace's id answers 404 — including ids
  inside a request body (ADR-0004; tests in `apps/api/tests/isolation/scenarios_paket_tahap1_test.go`).
* UI words: **Paket** is a WO product, **Paket langganan** is the ElProof subscription; `check:terms` enforces both.
