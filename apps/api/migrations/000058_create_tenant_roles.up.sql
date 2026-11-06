-- Per-tenant roles and permissions (PLAN.md Fase 3a, decision K12; ADR-0026).
-- Each WO decides who may do what: a role is a set of permissions from a
-- fixed catalog (owned by the code, apps/api/internal/modules/access/domain)
-- plus a project scope (every project, or only projects the person is PIC /
-- PIC Sales of). Staff and client access are both roles. Owned by `access`;
-- staff_members.role_id, clients.access_role_id and credentials.role_id
-- refer to tenant_roles.id by value (no cross-module foreign keys).
CREATE TABLE tenant_roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  principal_kind ENUM('staff', 'client') NOT NULL,
  name VARCHAR(60) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  template_key VARCHAR(30) NULL,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  project_scope ENUM('all', 'assigned') NOT NULL DEFAULT 'all',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_tenant_roles_name (tenant_id, principal_kind, name),
  UNIQUE KEY uq_tenant_roles_template (tenant_id, template_key),
  KEY idx_tenant_roles_tenant (tenant_id)
);

CREATE TABLE tenant_role_permissions (
  role_id BIGINT UNSIGNED NOT NULL,
  permission VARCHAR(64) NOT NULL,
  PRIMARY KEY (role_id, permission),
  CONSTRAINT fk_tenant_role_permissions_role FOREIGN KEY (role_id) REFERENCES tenant_roles (id) ON DELETE CASCADE
);

-- Built-in templates for every existing tenant (new tenants get them from
-- the access module when their workspace is created). Keep in step with
-- access/domain/templates.go.

INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope)
  SELECT id, 'staff', 'Owner', 'Pemilik workspace. Semua akses; tidak bisa diubah atau dihapus.', 'owner', TRUE, 'all' FROM tenants;
INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope)
  SELECT id, 'staff', 'Admin', 'Mengelola semua project, client, vendor, dan keuangan. Tanpa pengaturan workspace.', 'admin', FALSE, 'all' FROM tenants;
INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope)
  SELECT id, 'staff', 'Wedding Planner', 'Menjalankan project yang ditugaskan kepadanya.', 'wedding_planner', FALSE, 'assigned' FROM tenants;
INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope)
  SELECT id, 'staff', 'Sales', 'Membuat project baru dan mengelola client dari project yang ia pegang.', 'sales', FALSE, 'assigned' FROM tenants;
INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope)
  SELECT id, 'staff', 'Keuangan', 'Mencatat dan memantau pembayaran di semua project.', 'finance', FALSE, 'all' FROM tenants;
INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope)
  SELECT id, 'client', 'Pengantin', 'Melihat seluruh portal, termasuk tagihan dan riwayat pembayaran.', 'bride', FALSE, 'all' FROM tenants;
INSERT INTO tenant_roles (tenant_id, principal_kind, name, description, template_key, is_system, project_scope)
  SELECT id, 'client', 'Keluarga', 'Melihat portal tanpa informasi keuangan.', 'family', FALSE, 'all' FROM tenants;

INSERT INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'dashboard.view' AS permission UNION ALL SELECT 'dashboard.revenue.view' AS permission UNION ALL SELECT 'project.create' AS permission UNION ALL SELECT 'project.edit' AS permission UNION ALL SELECT 'project.assign_pic' AS permission UNION ALL SELECT 'project.duplicate' AS permission UNION ALL SELECT 'project.archive' AS permission UNION ALL SELECT 'project.delete' AS permission UNION ALL SELECT 'finance.record' AS permission UNION ALL SELECT 'finance.margin.view' AS permission UNION ALL SELECT 'finance.payment.delete' AS permission UNION ALL SELECT 'client.manage' AS permission UNION ALL SELECT 'vendor.view' AS permission UNION ALL SELECT 'vendor.manage' AS permission UNION ALL SELECT 'vendor.delete' AS permission UNION ALL SELECT 'vendor.category.manage' AS permission UNION ALL SELECT 'template.manage' AS permission UNION ALL SELECT 'team.manage' AS permission UNION ALL SELECT 'role.manage' AS permission UNION ALL SELECT 'subscription.manage' AS permission UNION ALL SELECT 'business_profile.manage' AS permission) p WHERE r.template_key = 'owner';
INSERT INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'dashboard.view' AS permission UNION ALL SELECT 'dashboard.revenue.view' AS permission UNION ALL SELECT 'project.create' AS permission UNION ALL SELECT 'project.edit' AS permission UNION ALL SELECT 'project.assign_pic' AS permission UNION ALL SELECT 'project.duplicate' AS permission UNION ALL SELECT 'project.archive' AS permission UNION ALL SELECT 'finance.record' AS permission UNION ALL SELECT 'finance.margin.view' AS permission UNION ALL SELECT 'finance.payment.delete' AS permission UNION ALL SELECT 'client.manage' AS permission UNION ALL SELECT 'vendor.view' AS permission UNION ALL SELECT 'vendor.manage' AS permission) p WHERE r.template_key = 'admin';
INSERT INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'project.edit' AS permission UNION ALL SELECT 'finance.record' AS permission) p WHERE r.template_key = 'wedding_planner';
INSERT INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'project.create' AS permission UNION ALL SELECT 'project.edit' AS permission UNION ALL SELECT 'project.assign_pic' AS permission UNION ALL SELECT 'project.archive' AS permission UNION ALL SELECT 'finance.record' AS permission UNION ALL SELECT 'finance.margin.view' AS permission UNION ALL SELECT 'client.manage' AS permission) p WHERE r.template_key = 'sales';
INSERT INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'dashboard.view' AS permission UNION ALL SELECT 'dashboard.revenue.view' AS permission UNION ALL SELECT 'finance.record' AS permission UNION ALL SELECT 'finance.margin.view' AS permission UNION ALL SELECT 'finance.payment.delete' AS permission UNION ALL SELECT 'vendor.view' AS permission) p WHERE r.template_key = 'finance';
INSERT INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'portal.finance.view' AS permission UNION ALL SELECT 'portal.vendor.view' AS permission UNION ALL SELECT 'portal.timeline.view' AS permission UNION ALL SELECT 'portal.documents.view' AS permission UNION ALL SELECT 'portal.issues.view' AS permission) p WHERE r.template_key = 'bride';
INSERT INTO tenant_role_permissions (role_id, permission)
  SELECT r.id, p.permission FROM tenant_roles r JOIN (SELECT 'portal.vendor.view' AS permission UNION ALL SELECT 'portal.timeline.view' AS permission UNION ALL SELECT 'portal.documents.view' AS permission UNION ALL SELECT 'portal.issues.view' AS permission) p WHERE r.template_key = 'family';

-- Staff: the fixed role enum becomes the role's name; authority moves to role_id.
ALTER TABLE staff_members
  MODIFY COLUMN role VARCHAR(60) NOT NULL,
  ADD COLUMN role_id BIGINT UNSIGNED NULL AFTER role,
  ADD KEY idx_staff_members_role (role_id);
UPDATE staff_members s
JOIN tenant_roles r ON r.tenant_id = s.tenant_id AND r.template_key = CASE s.role
  WHEN 'Owner' THEN 'owner' WHEN 'Admin' THEN 'admin' WHEN 'Staff' THEN 'wedding_planner' WHEN 'Sales' THEN 'sales' END
SET s.role_id = r.id, s.role = r.name;

-- Clients: the portal access role (Pengantin / Keluarga by default).
ALTER TABLE clients
  ADD COLUMN access_role_id BIGINT UNSIGNED NULL AFTER role,
  ADD KEY idx_clients_access_role (access_role_id);
UPDATE clients c
JOIN tenant_roles r ON r.tenant_id = c.tenant_id
  AND r.template_key = CASE c.role WHEN 'Family Representative' THEN 'family' ELSE 'bride' END
SET c.access_role_id = r.id;

-- Logins carry the role id into the session token.
-- role holds the role's name, which may be up to 60 characters like tenant_roles.name.
ALTER TABLE credentials MODIFY role VARCHAR(60) NOT NULL, ADD COLUMN role_id BIGINT UNSIGNED NULL AFTER role;
UPDATE credentials c
JOIN staff_members s ON c.principal_type = 'staff' AND c.principal_id = CAST(s.id AS CHAR) COLLATE utf8mb4_0900_ai_ci AND c.tenant_id = s.tenant_id
SET c.role_id = s.role_id, c.role = s.role;
UPDATE credentials c
JOIN clients cl ON c.principal_type = 'client' AND c.principal_id = CAST(cl.id AS CHAR) COLLATE utf8mb4_0900_ai_ci AND c.tenant_id = cl.tenant_id
SET c.role_id = cl.access_role_id;
