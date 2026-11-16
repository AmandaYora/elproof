-- Client Portal finding #4 (docs/plan/revamp-client-portal/RISET.md, ADR-0047): an
-- optional cover photo per project, uploaded by the WO and shown behind the hero
-- of the Client Portal. The object lives in object storage; the row keeps its key
-- and when it last changed (the `?v=` of the cover URL, so a browser cache of the
-- old photo is never shown after a replace). NULL = no cover.
ALTER TABLE projects
  ADD COLUMN cover_photo_path VARCHAR(500) NULL,
  ADD COLUMN cover_updated_at TIMESTAMP NULL;
