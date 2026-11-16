-- RT-07: a proof file the couple sent from the Client Portal names who sent it.
-- uploaded_by_client_id is a primitive reference into `clients` (no foreign key);
-- NULL for a file staff uploaded (uploaded_by_staff_id stays the staff id, 0 for
-- a client's file).
ALTER TABLE evidence ADD COLUMN uploaded_by_client_id BIGINT UNSIGNED NULL AFTER uploaded_by_staff_id;

-- Files already sent from the portal (migration 000078): the proof of a client
-- payment the couple submitted that no staff member uploaded.
UPDATE evidence e
  JOIN client_payments cp ON cp.id = e.related_id AND cp.project_id = e.project_id
SET e.uploaded_by_client_id = cp.submitted_by_client_id,
    e.updated_at = e.updated_at
WHERE e.related_kind = 'clientPayment'
  AND e.uploaded_by_staff_id = 0
  AND cp.submitted_by_client_id IS NOT NULL;
