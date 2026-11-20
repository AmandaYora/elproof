# ADR-0049: The `offers` module, its ports, and the web `rincian` public entry

## Status
Accepted — 2026-11-19. Plan: `docs/plan/paket-wo/PLAN.md` §2. Builds on ADR-0048. Implemented for Tahap 1 (catalog); the parts marked
*later* land with the stage named.

## Context
Packages and quotations need the vendor directory (validation, reference prices, capacity), staff names, and the WO's business profile;
projects and vendors in turn need offers (pricing rules, "is this vendor referenced?"), and platform must seed event kinds when a WO signs
up. A direct import in either direction would create Go import cycles: `staff/application` already imports `projects/contracts`, and
`projects/contracts` imports `projects/application`. `platform` has no `contracts/` package.

## Decision
* **New module `offers`** (screens: Paket & Penawaran) owns `event_kinds`, `packages`, `package_event_kinds`, `package_components`, and
  *later* the `quotation_*` tables. Its pure rules (`offers/domain`, ADR-0048) are the single place prices and capacity are computed.
* **Dependency rules.**
  * `offers` imports only `vendors/contracts`, never `projects/contracts` or `staff/contracts`.
  * Everything that needs offers gets `offers/contracts` through a **setter** in `cmd/server/main.go`, built in the order
    `vendors` → `offers`: `vendorsModule.SetOfferReferences(...)` (vendor, venue and category services refuse a permanent delete of a
    referenced row) and `platformModule.SetEventKinds(...)` (sign-up calls `SeedDefaultEventKinds`, idempotent). *Later:*
    `projectsModule.SetOffers(...)` for the rules and conversion.
  * The ports each consumer declares are local interfaces (`vendors/application.OfferReferences`,
    `platform/application.EventKindSeeder`), so no module imports another's internals and no cycle can form.
  * *Later:* `StaffNameResolver` (Tahap 5, quotation PIC names) and `WorkspaceProfile` (Tahap 3 for the PKP flag on projects,
    Tahap 5 for the PDF letterhead) are ports filled by adapters in `main.go`. Tahap 1 needs neither: the quote settings are read and
    written inside `platform`.
* **Web.** Catalog screens are their own modules (`modules/packages`, `modules/event-kinds`); quote settings live in
  `modules/business-profile`. *Later:* the event, supplier and line editors are built once in `modules/rincian`, whose `index.ts` is its
  only public entry; `offers`, `projects` and `client-portal` may import that entry and nothing behind it. This is a recorded exception
  to `ARCHITECTURE_OVERVIEW.md` ("modules never import each other"); shared components stay domain-agnostic.

## Consequences
* Wiring order in `main.go` matters and is commented there.
* The vendors module stays usable without offers (a nil `OfferReferences` skips the check), which keeps its unit tests independent.
* Platform Console gets no endpoint that reads a WO's packages or quotations.
