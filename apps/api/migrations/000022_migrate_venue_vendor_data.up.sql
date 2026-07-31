-- One-time data migration (ADR-0016): copy every existing "Venue"-category
-- vendor row into the new venues directory, then retire the source rows.
-- New venue-only fields (phone_venue, city, rental_price, charge, capacity,
-- facilities, social_media, attachment_path) are left NULL -- there is no
-- historical data for them, the WO fills them in afterward. `phone` (the old
-- vendor's single phone column) maps to the new `phone_pic` -- the closest
-- semantic match, since it was the vendor's actual contact number.
--
-- The NOT EXISTS guard makes this migration safe to re-run (ADR-0011
-- documents a real incident where a migration needed a manual re-run after
-- a dirty state) -- without it, a second run would duplicate every venue,
-- since the UPDATE below doesn't change vc.name and so doesn't stop the
-- JOIN from matching the same source rows again.
INSERT INTO venues (tenant_id, name, pic_name, phone_pic, email, address, notes, is_active, created_at, updated_at)
SELECT v.tenant_id, v.name, v.pic_name, v.phone, v.email, v.address, v.notes, v.is_active, v.created_at, v.updated_at
FROM vendors v
JOIN vendor_categories vc ON vc.id = v.category_id
WHERE vc.name = 'Venue'
  AND NOT EXISTS (
    SELECT 1 FROM venues existing
    WHERE existing.tenant_id = v.tenant_id AND existing.name = v.name
  );

-- Deactivated, never deleted -- consistent with this module's existing
-- no-hard-delete convention (SetActive, same as project milestones).
-- Existing project_vendors history rows engaging one of these vendors are
-- left untouched (see ADR-0016 -- not auto-linked to the new venue_id).
UPDATE vendors v
JOIN vendor_categories vc ON vc.id = v.category_id
SET v.is_active = FALSE
WHERE vc.name = 'Venue';

UPDATE vendor_categories SET is_active = FALSE WHERE name = 'Venue';
