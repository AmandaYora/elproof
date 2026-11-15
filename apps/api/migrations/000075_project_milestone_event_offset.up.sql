-- EP-06: a Timeline step seeded from the Jadwal bawaan remembers its offset from
-- the event date (H-14 = 14, H+3 = -3), so its target follows the project's event
-- and booking dates when those change. NULL = the date is fixed: the step was
-- added by hand, or someone set its target by hand.
ALTER TABLE project_milestones ADD COLUMN days_before_event INT NULL AFTER completed_date;

-- Link the steps that still look exactly as they were seeded: same name as one of
-- the workspace's Jadwal bawaan rows, not finished, and the date the seeding rule
-- gave them (event_date minus the offset, or the booking date it was pulled to).
-- Steps someone dated by hand keep their fixed date. MIN() picks one offset when a
-- workspace has two template rows with the same name.
UPDATE project_milestones m
JOIN projects p ON p.id = m.project_id
JOIN (
  SELECT tenant_id, name, MIN(days_before_event) AS days_before_event
  FROM project_milestone_templates
  GROUP BY tenant_id, name
) t ON t.tenant_id = p.tenant_id AND t.name = m.name
SET m.days_before_event = t.days_before_event
WHERE m.status NOT IN ('Completed', 'Cancelled')
  AND (m.target_date = DATE_SUB(p.event_date, INTERVAL t.days_before_event DAY)
       OR m.target_date = p.prep_start_date);

-- Re-date the linked steps with the corrected rule (domain.MilestoneTarget): the
-- event date minus the offset, pulled to the booking date only when the booking is
-- on or before the event. This repairs steps that were all stuck on the booking date
-- (a project whose event date was entered or changed after the steps were seeded).
UPDATE project_milestones m
JOIN projects p ON p.id = m.project_id
SET m.target_date = CASE
    WHEN p.prep_start_date <= p.event_date
         AND DATE_SUB(p.event_date, INTERVAL m.days_before_event DAY) < p.prep_start_date
      THEN p.prep_start_date
    ELSE DATE_SUB(p.event_date, INTERVAL m.days_before_event DAY)
  END
WHERE m.days_before_event IS NOT NULL;
