# ADR-0026: Per-workspace roles and permissions (RBAC per tenant)

## Status
Accepted — implemented 2026-11-06 (PLAN.md Fase 3a, decision K12). Supersedes ADR-0017.

## Context
ADR-0017 hard-coded three staff roles (`Owner`/`Admin`/`Staff`) as an enum and branched on the
role string in handlers and pages. Clients were all equal: the only way to hide money from a
family member was Fase 1's whitelist of client-readable routes. Wedding organizers work
differently (some have a sales person, a finance person, a field coordinator), so decision K12
lets each workspace edit its roles while ElProof keeps a fixed catalog of what can be granted.

## Decision
- **New module `access`** owns `tenant_roles` (tenant, kind `staff`/`client`, name, description,
  `project_scope` `all`/`assigned`, `is_system`, `template_key`) and `tenant_role_permissions`.
- **Fixed permission catalog in code** (`access/domain/catalog.go`), not a table: 21 staff
  permissions (dashboard, project, finance, client, vendor, template, team, role, subscription,
  business profile) and 5 portal permissions (`portal.finance.view`, `portal.vendor.view`,
  `portal.timeline.view`, `portal.documents.view`, `portal.issues.view`). Each has a plain
  Indonesian label and description shown in the editor. A unit test checks that migration 000058
  seeds exactly the Go templates.
- **Templates seeded per workspace** (migration 000058 for existing tenants, `SeedDefaults` at
  signup and at registration by ElProof's team): Owner (system, locked, always every staff
  permission), Admin, Wedding Planner (assigned), Sales (assigned), Keuangan; portal roles
  Pengantin and Keluarga (no finance). Built-in roles can be edited but not deleted; custom roles
  can be deleted only when nobody uses them. The name "Owner" is reserved.
- **Project scope**: `all`, or `assigned` = only projects where the person is PIC or PIC Sales.
  A role with `assigned` scope never sees tenant-wide lists (dashboard, all clients).
- **Assignment**: `staff_members.role_id` (the `role` column keeps the role's name for display),
  `clients.access_role_id` (default by relation: bride/groom → Pengantin, family → Keluarga),
  `credentials.role_id`. Legacy `Staff` became Wedding Planner.
- **Enforcement on the server**: the JWT carries `rid`; `middleware.LoadGrants` resolves the role's
  permissions per request (30-second cache, invalidated on edit) and puts `Grants` in the context.
  Handlers check `Grants.Can(key)` / `AllProjects`. Errors are `403 forbidden_permission`
  (the browser re-fetches its grants), `403 role_missing` (role deleted → sign out) and `401
  session_revoked` for tokens without `rid`. Changing someone's role makes the access
  tokens they hold stop working at once (`401 session_revoked`); the web client then refreshes and
  carries on with the new role, without a new sign-in, and nothing of the old role is kept
  (revised 2026-11-10, T-20261110-12: this sentence used to say the change "ends their sessions").
  Deactivating or deleting someone does end every session. Editing a role's permissions applies at
  once without a new sign-in.
- **No escalation**: someone who is not the Owner may only hand out, edit, create or delete a staff
  role that their own role covers (`Grants.Covers`: same permissions or fewer, and `all` scope only
  if they have it), may not change the role they hold, and may not edit, re-invite, deactivate or
  delete a member whose role goes beyond theirs. So `team.manage` / `role.manage` never become a way
  to gain access.
- **Client portal** reads are gated by `portal.*` instead of the Fase 1 whitelist; without
  `portal.finance.view` the project's contract value is zeroed, payment routes are refused and
  payment receipts are hidden from the documents list and file download. The portal's tabs and
  summary card follow the same permissions.
- **Vendor directory**: the Vendor & Venue menu (paginated list, export, project history) needs
  `vendor.view`; looking up single vendors/venues and the picker list is also open to people who
  work on projects (`project.edit`, `project.create`, `finance.record`, `dashboard.view`), because
  a project's vendor and venue forms need prices.
- **Audit**: `audit_log` gains `tenant_id`; role creation, edits and deletion are recorded.
- **Web**: `GET /access/me` feeds `usePermissionStore` (`useCan`, `useAllProjects`,
  `RequirePermission`); sidebar entries and action buttons follow permissions; Pengaturan →
  Peran & Akses edits roles in a drawer; staff and client forms pick a role.

## Consequences
- Adding a permission means a code change (catalog + checks + migration for existing roles);
  workspaces cannot invent permissions, only combine them.
- Handler checks are spread across modules; `projects/presentation/permission_test.go` pins the
  write-route mapping, but new routes must remember their permission.
- `finance.margin.view` only controls whether the margin summary is shown: the numbers it is
  computed from (contract value, vendor and venue costs) reach everyone who works on the project.
- A role edit can lock people out of features immediately; the Owner role cannot be edited, so a
  workspace can never lock itself out completely.
- Down migration 000058 maps role names back to the old enum; anything custom becomes `Staff`.
