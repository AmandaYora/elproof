-- New role "Sales" (PLAN.md revisi-timeline-vendor-role-sales) -- a staff
-- member who creates projects and hands them over to a Wedding Planner via
-- the new pic_sales_staff_id PIC slot (see 000037).
ALTER TABLE staff_members MODIFY COLUMN role ENUM('Owner', 'Admin', 'Staff', 'Sales') NOT NULL;
