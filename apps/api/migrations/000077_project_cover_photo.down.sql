-- The stored objects are left in object storage (orphans; nothing refers to them).
ALTER TABLE projects DROP COLUMN cover_updated_at, DROP COLUMN cover_photo_path;
