-- Tenant custom domains are discontinued (PLAN.md decision K15): every tenant
-- and client uses the platform's own address. ADR-0015 is deprecated.
ALTER TABLE tenants DROP COLUMN custom_domain;
