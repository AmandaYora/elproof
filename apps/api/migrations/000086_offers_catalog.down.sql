DELETE FROM tenant_role_permissions WHERE permission = 'package.manage';
DROP TABLE IF EXISTS package_components;
DROP TABLE IF EXISTS package_event_kinds;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS event_kinds;
