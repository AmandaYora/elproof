-- Puts back the seeded English wording, only where the row still holds the exact
-- text 000082 wrote.
UPDATE tenant_roles
SET description = 'Mengelola semua project, client, vendor, dan keuangan. Tanpa pengaturan workspace.', updated_at = updated_at
WHERE template_key = 'admin'
  AND BINARY description = 'Mengelola semua proyek, klien, vendor, dan keuangan. Tanpa pengaturan workspace.';

UPDATE tenant_roles
SET description = 'Menjalankan project yang ditugaskan kepadanya.', updated_at = updated_at
WHERE template_key = 'wedding_planner'
  AND BINARY description = 'Menjalankan proyek yang ditugaskan kepadanya.';

UPDATE tenant_roles
SET description = 'Membuat project baru dan mengelola client dari project yang ia pegang.', updated_at = updated_at
WHERE template_key = 'sales'
  AND BINARY description = 'Membuat proyek baru dan mengelola klien dari proyek yang ia pegang.';

UPDATE tenant_roles
SET description = 'Mencatat dan memantau pembayaran di semua project.', updated_at = updated_at
WHERE template_key = 'finance'
  AND BINARY description = 'Mencatat dan memantau pembayaran di semua proyek.';
