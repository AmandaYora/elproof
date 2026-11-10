# ADR-0035: Server-side sorting and filtering on every list, derived list columns, tenant names on the transaction list, bulk activate/deactivate

## Status
Accepted — implemented 2026-11-10 (docs/plan/revamp-ui-shadcn/PLAN.md Tahap 6, backend half: endpoints E11, E14 and the bulk part of E13).

## Context
`pagination.FromRequest` read only `page` and `limit`, so every list had one fixed order and the web sorted whatever page it had. The Klien page
loaded all clients and all projects and grouped, filtered and paged them in the browser; the project list could not show what is still owed or when
the next installment falls due; the directories had no active/inactive filter and no category name; the Platform Console's transaction table needed a
second call (`/tenants/names`) to label its rows; and activating or deactivating many vendors or venues meant one `toggle-active` call per row (a toggle,
so a retry undoes itself).

## Decision

### One sorting helper (`internal/shared/pagination/sort.go`)
`?sort=field:asc|desc`, read by `pagination.SortFromRequest(r, whitelist, default)` on every paginated list.
* **A whitelist per endpoint** maps the public field name to a **fixed SQL fragment written in the repository** (`pagination.Col("v.name")`,
  `NullableCol("c.name")`). Nothing the request says reaches the SQL: a field that is not a key — injection attempts, a comma list, a backquoted
  name, `id` — is the same 422 `errors.sort: ["Urutkan menurut salah satu dari: …"]`; a bad direction is a 422 too; two `sort` parameters are refused
  (single column only). The direction defaults to `asc` and is case-insensitive, the field name is exact.
* **The primary key is always the last sort key**, in the same direction (`ORDER BY name ASC, id ASC`), so equal values page stably. A column that may
  be empty (`NullableCol`) puts its empty values **last in both directions** (`ORDER BY (col IS NULL), col DIR, id DIR`).
* **The effective sort is echoed in `meta.sort`** (`"remaining:desc"`), an additive key next to `page`, `limit`, `total`, `total_pages`; lists without
  sorting leave it out.
* **The default is the order the list already had**, named by a field of the whitelist: `GET /projects` `eventDate:desc` (was `event_date DESC, id DESC`),
  the platform lists `createdAt:desc`, and the lists that were `ORDER BY id` now read `createdAt:asc` (`created_at, id`: identical order, every row is created after
  the previous one and ties fall back to the id).
* `RankedCol(column, idsOrderedByName)` renders `FIELD(column, 7, 3, 12)` for a column whose natural order is not the wanted one because the display value lives in
  another module (a plan's or a role's *name*, stored here as an id): the ids are numbers read from the database, never request text. `Sort.With` swaps it in for the
  whitelisted placeholder.
* `pagination.SearchFromRequest` gives every list `q` as the common name of the free-text search; `search` keeps working (`q` wins when both are sent).

Whitelists and defaults per endpoint are in `docs/API_CONTRACT.md` (Tahap 6).

### Derived columns of `GET /projects` (E11)
`remaining` and `nextDueDate` follow the rules the overview and the portal already use (`domain.ComputeLedger`, `AllocateTerms`, `NextOpenTerm`): applied payments
= live DP + Termin + Pelunasan − Refund (Tambahan out, soft-deleted payments out), floored at 0; installments are allocated earliest due date first, then id;
`nextDueDate` is the due date of the first one not fully paid (not "the first one not yet overdue"). The page's values are computed in Go from **two batched queries**
(`domain.ComputeListMoney`, unit-tested, no N+1). Sorting by them needs the same rule in SQL (`projectRemainingSQL`, `projectNextDueSQL`: correlated sub-selects, the
running total of the schedule compared with the applied amount); because that is a second implementation of one rule, the integration suite builds projects chosen
so that a wrong SQL rule (a deleted payment counted, a Tambahan counted, an off-by-one in "unpaid") reverses the order, and checks the sorted order against the displayed values.
Scope is untouched: `picFilter` (PIC or PIC Sales of the caller, for a role limited to its own projects) is ANDed with every new filter, so `?picStaffId=` of someone else
gives a planner nothing. New filters: `picStaffId`, `eventFrom`, `eventTo` (inclusive dates; reversed range → 422).

### Data that another module owns: batch, then order after reading — never a join
Three lists order or filter by values the listing module does not store.
* **Clients** (`GET /clients`): `projectName` and `projectEventDate` come from `projects.contracts.ProjectLabels` (new; one query for the page's distinct project ids, filtered by
  tenant) and the sign-in state from `identity.contracts.LoginStatuses` (new; one query per 500 principals, filtered by tenant and principal type). The filters that live in the
  clients table (`q`, `role`, `projectId`) and the orders `name`, `role`, `createdAt` stay SQL. When the order is `projectName`, `projectEventDate` or `loginStatus`, or the
  `loginStatus` filter is set, the service reads the tenant's matching clients (after the SQL filters), resolves the foreign values in batch, filters, orders (case-insensitive
  strings, ties by id, unresolved last) and slices the page itself (`domain.SortRows`). A workspace has hundreds of clients, not millions; the price of staying inside
  the module boundary is O(clients of one tenant) memory for those requests, and the unit tests pin the ordering rules to what the database does for the SQL columns.
* **Staff** (`roleName`) and **tenants** (`planName`): the names are few (roles of a workspace, plans of the platform), so the owning module is asked for them
  (`access` through the existing `RoleResolver` port, `billing.contracts.ListPlans`), the ids are ranked by name and the SQL orders by `RankedCol`. Real SQL paging, no memory path.
* **Transactions** (`tenantName`): `billing` owns the ledger, `platform` the names. `billing` declares a local `TenantDirectory` port (`TenantNames`, `SearchTenantIDs`),
  `platform.Module` satisfies it and `cmd/server` wires it (`billingModule.SetTenantDirectory(platformModule)`), the same bridge idiom as `projects`' `ClientDirectory`. Names of a
  page come in one batch; `?search=` matches the payment reference or the tenants whose name matches (at most 1000, resolved first and passed as `tenant_id IN (…)`); ordering by
  `tenantName` reads the matching transactions, their names (batches of 500), orders and slices in memory — the one request shape whose cost grows with the whole ledger of the
  filter, used by the Platform Console only. `from`/`to` are Indonesian calendar days on `created_at`, inclusive. A workspace Owner is still forced to their own tenant by the
  query, whatever they search. `GET /tenants/names` stays.

### `GET /clients` keeps its older shapes
`?all=true` and `?projectId=` alone still answer the plain arrays the pickers and the project tab read. The paginated list is chosen by any of `page`, `limit`, `sort`, `q`, `role`,
`loginStatus`; then `projectId` is a filter with the project tab's access rule (a role limited to its own projects must be PIC or PIC Sales of it). Without `projectId` the list is the tenant-wide
one (`client.manage` with every-project scope, as before). `search` is not one of the switches, so an older call `?projectId=1&search=x` keeps its shape.

### Console lists
* **Tenants:** `sort` (`name`, `status`, `createdAt`, `trialEndsAt`, `planName`); `status`, `planId`, `source`, `search` already existed. `status` orders by the stored column the hourly sweeper keeps current
  (the same column the `status` filter uses); `createdAt` is `joined_at`.
* **Audit log:** `from`, `to`, `q` existed; added `action` (exact label), `search` (alias of `q`) and `sort` (`createdAt`, `action`, `actor`). Migration `000070` adds `KEY (action, id)` so the
  exact filter and `ORDER BY action` do not scan the append-only table; `actor` orders by `CONCAT(actor_type, '/', actor_role, '/', LPAD(actor_id, 20, '0'))` (no index; Super Admin only).

### Bulk activate/deactivate (E13 P3)
`POST /vendors/bulk` and `POST /venues/bulk` with `{"ids":[…], "action":"activate"|"deactivate"}` and the permission of the resource's single toggle (`vendor.manage`). 1–100 ids as sent
(deduplicated, order kept), positive integers; validation is a 422 with per-field keys `ids` and `action`, all problems at once. One transaction: the matching rows of **the caller's tenant** are
read `FOR UPDATE`, `domain.PlanBulk` decides (not found / already in that state / to update), one `UPDATE … WHERE tenant_id = ? AND id IN (…)` applies. Answer `{updated, unchanged[], notFound[]}`:
another workspace's id and a nonexistent id are the same `notFound`, and the other workspace's row is never read into the answer or touched. Unlike the toggle it is idempotent (a retry is a no-op).
The single toggle writes no activity record (the `vendors` module has no activity log), so neither does the bulk action.

### What was not done
* No `?sort=` on the `?all=true` pickers or on `/vendors/summary` (small, bounded lists the web orders itself).
* No index for the other sort columns: they order inside one workspace's rows. The `amount` order of the Platform Console's transaction list scans the table.
* Tenant `status` sorts by the stored column, not by the lifecycle computed at read time.

## Consequences
* The web can drop `all=true` for filtering and grouping (Stage 6 "done when"): `/clients` is paginated server-side, `/projects` filters by PIC and event period, directories by `isActive`.
* Adding a sortable column is one line in the repository's whitelist and one row in the API contract; a new list gets the same behaviour by calling `SortFromRequest` and `Sort.OrderBy`.
* The two implementations of the project money rule (Go for display, SQL for ordering) must change together; the integration scenario "b: sorting by the derived columns" fails when they do not.
