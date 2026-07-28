-- Archive is orthogonal to `status` (a Completed or Cancelled project can
-- independently be archived/unarchived) -- see knowledge/decisions/
-- ADR-0013-project-archive-and-hard-delete.md.
ALTER TABLE projects ADD COLUMN is_archived BOOLEAN NOT NULL DEFAULT FALSE;
