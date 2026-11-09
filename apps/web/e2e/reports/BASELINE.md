# Baseline performa dan jumlah panggilan API

Diukur dengan `npm run e2e:perf` pada UI lama (sebelum revamp). Profil: HP 375×812, UA Android Chrome, CPU 4×,
1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas
build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi
(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,
pengumuman), bukan hanya data halaman.

| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Beranda pemasaran | 0 | 0.0 | 11 | 138.5 | 1728 | 0.421 | 11 | - |
| Ringkasan Platform | 3 | 1.3 | 18 | 140.8 | 2272 | 0 | 11 | 56 |
| Tagihan Platform | 3 | 0.6 | 24 | 142.9 | 2220 | 0 | 11 | - |
| Daftar Tenant | 3 | 2.3 | 44 | 169.9 | 2720 | 0 | 11 | - |
| Detail Tenant | 4 | 1.9 | 41 | 174.1 | 2896 | 0 | 11 | - |
| Beranda Portal | 10 | 4.6 | 25 | 147.9 | 2292 | 0 | 11 | 208 |
| Portal, Jadwal | 10 | 4.6 | 14 | 140.7 | 1960 | 0 | 12 | - |
| Portal, Kendala | 13 | 5.2 | 21 | 146.2 | 2140 | 0.001 | 10 | - |
| Portal, Pembayaran | 11 | 4.0 | 18 | 144.8 | 2104 | 0 | 12 | - |
| Portal, Vendor | 15 | 6.2 | 23 | 149.3 | 2140 | 0 | 12 | - |
| Beranda WO | 7 | 7.0 | 27 | 163.1 | 2516 | 0.393 | 40 | 64 |
| Daftar Client | 6 | 4.1 | 32 | 163.7 | 2368 | 0.054 | 11 | - |
| Daftar Project | 5 | 3.6 | 26 | 157.2 | 2204 | 0.098 | 11 | - |
| Detail Proyek, tab Kendala | 16 | 7.9 | 36 | 167.3 | 2556 | 0.098 | 11 | - |
| Detail Proyek, tab Keuangan | 19 | 7.4 | 50 | 179.8 | 2572 | 0.098 | 10 | - |
| Detail Proyek (Ringkasan) | 16 | 7.5 | 36 | 168.2 | 2572 | 0.098 | 10 | - |
| Detail Proyek, tab Vendor | 24 | 14.2 | 47 | 183.2 | 2568 | 0.054 | 12 | - |
| Daftar Vendor | 6 | 3.2 | 31 | 161.5 | 1408 | 0.098 | 12 | - |

**Total JS seluruh aplikasi:** 1436.4 KB mentah, **489.5 KB gzip** (161 chunk).

Chunk terbesar (gzip):

- `index-v9K98F2_.js`: 129.2 KB gzip (398.3 KB mentah)
- `ProjectMilestoneTabPage-2Uub4FxP.js`: 87.0 KB gzip (262.9 KB mentah)
- `types-B4jcjfVQ.js`: 12.2 KB gzip (52.9 KB mentah)
- `MfaSetupPage-TTmiN_kA.js`: 11.7 KB gzip (30.1 KB mentah)
- `ProjectVendorTabPage-BlhEwQo_.js`: 9.7 KB gzip (37.3 KB mentah)
- `ProjectDetailLayout-DVICwT4Y.js`: 7.5 KB gzip (22.8 KB mentah)
- `DashboardPage-DVpIEXj1.js`: 6.9 KB gzip (23.9 KB mentah)
- `TenantDetailPage-dKjw6AsM.js`: 6.9 KB gzip (21.5 KB mentah)
- `PembayaranClientTabPage-C9Hd8EmK.js`: 5.3 KB gzip (16.7 KB mentah)
- `VendorListPage-DdQOETQO.js`: 5.1 KB gzip (16.0 KB mentah)

## Panggilan API per layar

**Beranda pemasaran** (0):


**Ringkasan Platform** (3):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/platform/trends?… (721 B)
- GET /api/v1/platform/metrics (422 B)

**Tagihan Platform** (3):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/subscription-transactions?… (301 B)
- GET /api/v1/tenants/names?… (59 B)

**Daftar Tenant** (3):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/plans?… (392 B)
- GET /api/v1/tenants?… (1759 B)

**Detail Tenant** (4):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/plans?… (392 B)
- GET /api/v1/subscription-transactions?… (98 B)
- GET /api/v1/console/tenants/1/overview (1201 B)

**Beranda Portal** (10):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/contact (105 B)
- GET /api/v1/projects/me (890 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/vendors (805 B)
- GET /api/v1/projects/1/client-payments (192 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/milestones (759 B)

**Portal, Jadwal** (10):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/contact (105 B)
- GET /api/v1/projects/me (890 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/client-payments (192 B)
- GET /api/v1/projects/1/vendors (805 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/milestones (759 B)

**Portal, Kendala** (13):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/contact (105 B)
- GET /api/v1/projects/me (890 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/client-payments (192 B)
- GET /api/v1/projects/1/vendors (805 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/vendors/summary (159 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/evidence (42 B)
- GET /api/v1/projects/1/vendors (805 B)

**Portal, Pembayaran** (11):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/contact (105 B)
- GET /api/v1/projects/me (890 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/client-payments (192 B)
- GET /api/v1/projects/1/vendors (805 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/evidence (42 B)
- GET /api/v1/projects/1/client-payments (192 B)

**Portal, Vendor** (15):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/contact (105 B)
- GET /api/v1/projects/me (890 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/client-payments (192 B)
- GET /api/v1/projects/1/vendors (805 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/vendors/summary (159 B)
- GET /api/v1/projects/1/evidence (42 B)
- GET /api/v1/projects/1/venue (44 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/projects/1/vendors (805 B)

**Beranda WO** (7):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/dashboard (4731 B)
- GET /api/v1/tenants/me/onboarding (237 B)

**Daftar Client** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/clients?… (546 B)
- GET /api/v1/projects?… (2539 B)

**Daftar Project** (5):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects?… (2594 B)

**Detail Proyek, tab Kendala** (16):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1 (900 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/client-payments (204 B)
- GET /api/v1/projects/1/vendors (875 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/vendors (875 B)

**Detail Proyek, tab Keuangan** (19):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1 (900 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/vendors (875 B)
- GET /api/v1/projects/1/client-payments (204 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/evidence (42 B)
- GET /api/v1/projects/1/client-payments (204 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/payment-terms (392 B)

**Detail Proyek (Ringkasan)** (16):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1 (900 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/vendors (875 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/client-payments (204 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/projects/1/payment-terms (392 B)

**Detail Proyek, tab Vendor** (24):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1 (900 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/client-payments (204 B)
- GET /api/v1/projects/1/milestones (759 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/vendors (875 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/projects/1/issues (421 B)
- GET /api/v1/projects/1/payments (212 B)
- GET /api/v1/projects/1/vendors (875 B)
- GET /api/v1/projects/1/evidence (42 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/projects/1/activity (2684 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/staff/summary (210 B)

**Daftar Vendor** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/vendors?… (1169 B)
