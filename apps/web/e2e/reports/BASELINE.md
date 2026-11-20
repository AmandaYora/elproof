# Baseline performa dan jumlah panggilan API

Diukur dengan `npm run e2e:perf` (garis dasar UI lama ada di `reports/before-t0/`; berkas ini selalu pengukuran terakhir). Profil: HP 375×812, UA Android Chrome, CPU 4×,
1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas
build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi
(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,
pengumuman), bukan hanya data halaman.

| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Beranda pemasaran | 1 | 0.4 | 18 | 184.9 | 2284 | 0.32 | 37 | - |
| Ringkasan Platform | 3 | 1.3 | 27 | 176.5 | 3276 | 0.082 | 85 | 64 |
| Tagihan Platform | 2 | 0.6 | 48 | 225.4 | 3284 | 0 | 30 | - |
| Daftar Tenant | 3 | 2.3 | 75 | 265.0 | 4072 | 0 | 35 | - |
| Detail Tenant | 4 | 1.9 | 61 | 238.8 | 3876 | 0.081 | 44 | - |
| Beranda Portal | 3 | 2.1 | 32 | 186.1 | 2840 | 0 | 25 | 0 |
| Portal, Jadwal | 5 | 4.5 | 54 | 269.9 | 3588 | 0 | 26 | - |
| Portal, Kendala | 4 | 2.6 | 53 | 267.6 | 3800 | 0 | 38 | - |
| Portal, Layanan | 6 | 6.2 | 58 | 271.9 | 3664 | 0.085 | 30 | - |
| Portal, Pembayaran | 6 | 3.1 | 58 | 274.9 | 3724 | 0.156 | 26 | - |
| Beranda WO | 6 | 7.2 | 54 | 278.6 | 4048 | 0.5 | 63 | 32 |
| Daftar Client | 6 | 5.0 | 70 | 285.2 | 3880 | 0.038 | 25 | - |
| Daftar Project | 6 | 4.5 | 61 | 275.7 | 1912 | 0.038 | 33 | - |
| Detail Proyek, tab Acara & rincian | 9 | 10.4 | 71 | 293.4 | 1892 | 0.286 | 29 | - |
| Detail Proyek, tab Kendala | 7 | 6.8 | 67 | 288.4 | 4332 | 0.164 | 25 | - |
| Detail Proyek, tab Keuangan | 8 | 5.4 | 78 | 298.2 | 1888 | 0.158 | 24 | - |
| Detail Proyek (Ringkasan) | 6 | 5.5 | 71 | 294.5 | 1892 | 0.158 | 28 | - |
| Detail Proyek, tab Vendor & venue | 6 | 6.2 | 74 | 294.5 | 1912 | 0.158 | 37 | - |
| Daftar Vendor | 6 | 3.3 | 64 | 268.7 | 3752 | 0.038 | 25 | - |

**Total JS seluruh aplikasi:** 2363.7 KB mentah, **845.1 KB gzip** (293 chunk).

Chunk terbesar (gzip):

- `index-gqPtvkZ3.js`: 151.6 KB gzip (460.7 KB mentah)
- `ProjectMilestoneGanttView-BI2DnLcl.js`: 83.6 KB gzip (252.3 KB mentah)
- `CommandPaletteDialog-BtjDnLud.js`: 19.8 KB gzip (58.2 KB mentah)
- `dropdown-menu-VFGYkNce.js`: 14.3 KB gzip (40.8 KB mentah)
- `usePositioner-D_OeMBp2.js`: 14.2 KB gzip (38.0 KB mentah)
- `store-bL_hFTvb.js`: 12.6 KB gzip (35.5 KB mentah)
- `types-B149ZHCL.js`: 12.2 KB gzip (52.9 KB mentah)
- `QuotationPage-gFNz-fW5.js`: 12.1 KB gzip (39.5 KB mentah)
- `MfaSetupPage-BwPNe9lz.js`: 11.9 KB gzip (30.4 KB mentah)
- `direct-upload-B3ZZP-nx.js`: 10.8 KB gzip (33.4 KB mentah)

## Panggilan API per layar

**Beranda pemasaran** (1):

- GET /api/v1/public/plans (361 B)

**Ringkasan Platform** (3):

- POST /api/v1/auth/refresh (490 B)
- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/platform/metrics (422 B)
- GET /api/v1/platform/trends?… (721 B)

**Tagihan Platform** (2):

- POST /api/v1/auth/refresh (490 B)
- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/subscription-transactions?… (409 B)

**Daftar Tenant** (3):

- POST /api/v1/auth/refresh (490 B)
- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/plans?… (404 B)
- GET /api/v1/tenants?… (1705 B)

**Detail Tenant** (4):

- POST /api/v1/auth/refresh (490 B)
- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/plans?… (404 B)
- GET /api/v1/subscription-transactions?… (122 B)
- GET /api/v1/console/tenants/1/overview (1251 B)

**Beranda Portal** (3):

- POST /api/v1/auth/refresh (477 B)
- GET /api/v1/access/me (287 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/projects/me/home (1730 B)

**Portal, Jadwal** (5):

- POST /api/v1/auth/refresh (477 B)
- GET /api/v1/access/me (287 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/projects/me/home (1730 B)
- GET /api/v1/projects/1/milestones (1580 B)
- GET /api/v1/projects/1/events (899 B)

**Portal, Kendala** (4):

- POST /api/v1/auth/refresh (477 B)
- GET /api/v1/access/me (287 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/projects/me/home (1730 B)
- GET /api/v1/projects/1/issues (581 B)

**Portal, Layanan** (6):

- POST /api/v1/auth/refresh (477 B)
- GET /api/v1/access/me (287 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/projects/me/home (1730 B)
- GET /api/v1/projects/1/issues (581 B)
- GET /api/v1/projects/1/lines (1736 B)
- GET /api/v1/projects/1/suppliers (1901 B)

**Portal, Pembayaran** (6):

- POST /api/v1/auth/refresh (477 B)
- GET /api/v1/access/me (287 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/projects/me/home (1730 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (459 B)
- GET /api/v1/projects/1/supplier-bills (181 B)

**Beranda WO** (6):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/dashboard (5938 B)
- GET /api/v1/tenants/me/onboarding (237 B)

**Daftar Client** (6):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/clients?… (756 B)
- GET /api/v1/projects?… (3187 B)

**Daftar Project** (6):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects?… (3192 B)

**Detail Proyek, tab Acara & rincian** (9):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (3061 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/projects/1/events (1041 B)
- GET /api/v1/projects/1/suppliers (2175 B)
- GET /api/v1/projects/1/lines (2289 B)

**Detail Proyek, tab Kendala** (7):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (3061 B)
- GET /api/v1/projects/1/issues (593 B)
- GET /api/v1/projects/1/suppliers (2175 B)

**Detail Proyek, tab Keuangan** (8):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (3061 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (490 B)
- GET /api/v1/projects/1/payment-terms (392 B)

**Detail Proyek (Ringkasan)** (6):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (3061 B)
- GET /api/v1/projects/1/activity?… (1423 B)

**Detail Proyek, tab Vendor & venue** (6):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (3061 B)
- GET /api/v1/projects/1/suppliers (2175 B)

**Daftar Vendor** (6):

- POST /api/v1/auth/refresh (464 B)
- GET /api/v1/tenants/me/branding (84 B)
- GET /api/v1/access/me (571 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/vendors?… (1289 B)
