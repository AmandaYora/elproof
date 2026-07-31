-- Cross-module primitive reference (see ADR-0016) -- resolved through the
-- vendors module's contract, never a SQL foreign key. projects.venue (free
-- text) is untouched: it stays the fallback for projects that never get a
-- structured venue_id.
ALTER TABLE projects ADD COLUMN venue_id BIGINT UNSIGNED NULL;
