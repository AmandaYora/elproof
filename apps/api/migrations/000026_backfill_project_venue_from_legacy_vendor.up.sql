-- One-time data cleanup, follow-up to ADR-0016 / migration 000022: when
-- Venue was extracted, existing project_vendors rows engaging a
-- "Venue"-category vendor were deliberately left untouched -- auto-linking
-- them to the new venues table at that time carried real risk of matching
-- the wrong venue (see ADR-0016's "Migration of existing data"). Once live,
-- these leftover rows were confirmed as a real problem worth cleaning up now
-- wherever the match is unambiguous, leaving anything else alone for manual
-- review in the app (Project Detail's own Venue tab, "Pilih Venue").
--
-- "Unambiguous" here means, for one project: exactly one project_vendors row
-- whose vendor's category is 'Venue' (not two or more), AND exactly one row
-- in `venues` sharing that vendor's tenant_id + name (not zero, not
-- several). Anything else is left completely alone -- venue_id stays NULL,
-- the old project_vendors row (and its children) stay exactly as they are.

CREATE TEMPORARY TABLE venue_backfill_matches AS
SELECT
  pv.project_id AS project_id,
  pv.id AS project_vendor_id,
  ve.id AS venue_id
FROM project_vendors pv
JOIN vendors v ON v.id = pv.vendor_id
JOIN vendor_categories vc ON vc.id = v.category_id AND vc.name = 'Venue'
JOIN projects p ON p.id = pv.project_id AND p.venue_id IS NULL
JOIN venues ve ON ve.tenant_id = v.tenant_id AND ve.name = v.name
WHERE (
    SELECT COUNT(*) FROM project_vendors pv2
    JOIN vendors v2 ON v2.id = pv2.vendor_id
    JOIN vendor_categories vc2 ON vc2.id = v2.category_id AND vc2.name = 'Venue'
    WHERE pv2.project_id = pv.project_id
  ) = 1
  AND (
    SELECT COUNT(*) FROM venues ve2 WHERE ve2.tenant_id = v.tenant_id AND ve2.name = v.name
  ) = 1;

CREATE TEMPORARY TABLE venue_backfill_milestones AS
SELECT vm.id AS vendor_milestone_id, m.project_vendor_id AS project_vendor_id
FROM vendor_milestones vm
JOIN venue_backfill_matches m ON vm.project_vendor_id = m.project_vendor_id;

-- Attach the matched venue to its project.
UPDATE projects p
JOIN venue_backfill_matches m ON m.project_id = p.id
SET p.venue_id = m.venue_id;

-- Record it on each affected project's own activity log -- same visibility
-- the Owner already has for every other change on a project, no separate
-- tooling needed to see which projects this migration touched.
INSERT INTO activity_log (project_id, type, actor_staff_id, entity_type, entity_id, entity_label, description, created_at)
SELECT
  m.project_id, 'project_updated', p.pic_staff_id, 'project', m.project_id, '',
  'Venue ditautkan otomatis dari data vendor lama (migrasi pasca ADR-0016)', NOW()
FROM venue_backfill_matches m
JOIN projects p ON p.id = m.project_id;

-- Cascade-delete the now-redundant legacy vendor engagement and everything
-- hanging off it, same table order ADR-0013 already established for a full
-- project hard-delete (vendor_payments references evidence, so it goes
-- first) -- just scoped to these specific project_vendor rows instead of an
-- entire project.
DELETE pay FROM vendor_payments pay
JOIN venue_backfill_matches m ON pay.project_vendor_id = m.project_vendor_id;

DELETE iss FROM vendor_issues iss
JOIN venue_backfill_matches m ON iss.project_vendor_id = m.project_vendor_id;

DELETE vm FROM vendor_milestones vm
JOIN venue_backfill_matches m ON vm.project_vendor_id = m.project_vendor_id;

DELETE e FROM evidence e
JOIN venue_backfill_milestones bm ON e.related_kind = 'vendorMilestone' AND e.related_id = bm.vendor_milestone_id;

DELETE e FROM evidence e
JOIN venue_backfill_matches m ON e.related_kind = 'projectVendor' AND e.related_id = m.project_vendor_id;

DELETE pv FROM project_vendors pv
JOIN venue_backfill_matches m ON pv.id = m.project_vendor_id;

DROP TEMPORARY TABLE venue_backfill_milestones;
DROP TEMPORARY TABLE venue_backfill_matches;
