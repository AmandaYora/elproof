-- Role Sales / PIC ganda (PLAN.md revisi-timeline-vendor-role-sales): a
-- second, independent PIC slot alongside the existing pic_staff_id (PIC
-- Wedding Planner). Sentinel 0 = "belum ditugaskan", same convention as
-- pic_staff_id itself and Project.VenueID's tri-state (AUTO_INCREMENT never
-- starts at 0) -- no nullable column needed.
ALTER TABLE projects ADD COLUMN pic_sales_staff_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER pic_staff_id;
