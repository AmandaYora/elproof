-- Back to the fixed role enum. Custom roles map to the closest built-in:
-- Owner stays Owner, a role named Admin/Sales keeps its name, anything else
-- (Wedding Planner, Keuangan, custom) becomes Staff (Wedding Planner).
UPDATE credentials c
JOIN staff_members s ON c.principal_type = 'staff' AND c.principal_id = CAST(s.id AS CHAR) COLLATE utf8mb4_0900_ai_ci AND c.tenant_id = s.tenant_id
SET c.role = CASE WHEN s.role IN ('Owner', 'Admin', 'Sales') THEN s.role ELSE 'Staff' END;
UPDATE staff_members SET role = CASE WHEN role IN ('Owner', 'Admin', 'Sales') THEN role ELSE 'Staff' END;
ALTER TABLE credentials DROP COLUMN role_id, MODIFY role VARCHAR(50) NOT NULL;
ALTER TABLE clients DROP KEY idx_clients_access_role, DROP COLUMN access_role_id;
ALTER TABLE staff_members
  DROP KEY idx_staff_members_role,
  DROP COLUMN role_id,
  MODIFY COLUMN role ENUM('Owner', 'Admin', 'Staff', 'Sales') NOT NULL;
DROP TABLE tenant_role_permissions;
DROP TABLE tenant_roles;
