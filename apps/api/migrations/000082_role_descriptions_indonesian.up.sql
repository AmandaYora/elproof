-- FQ-08: the built-in role descriptions seeded by migration 000058 (and by every
-- signup since, from access/domain/catalog.go Templates) used English words
-- ("project", "client"). Rewrite only rows that still hold the exact seeded text,
-- so a description a WO edited stays as they wrote it. BINARY: an exact match,
-- not the collation's case/accent-insensitive one.
UPDATE tenant_roles
SET description = 'Mengelola semua proyek, klien, vendor, dan keuangan. Tanpa pengaturan workspace.', updated_at = updated_at
WHERE template_key = 'admin'
  AND BINARY description = 'Mengelola semua project, client, vendor, dan keuangan. Tanpa pengaturan workspace.';

UPDATE tenant_roles
SET description = 'Menjalankan proyek yang ditugaskan kepadanya.', updated_at = updated_at
WHERE template_key = 'wedding_planner'
  AND BINARY description = 'Menjalankan project yang ditugaskan kepadanya.';

UPDATE tenant_roles
SET description = 'Membuat proyek baru dan mengelola klien dari proyek yang ia pegang.', updated_at = updated_at
WHERE template_key = 'sales'
  AND BINARY description = 'Membuat project baru dan mengelola client dari project yang ia pegang.';

UPDATE tenant_roles
SET description = 'Mencatat dan memantau pembayaran di semua proyek.', updated_at = updated_at
WHERE template_key = 'finance'
  AND BINARY description = 'Mencatat dan memantau pembayaran di semua project.';
