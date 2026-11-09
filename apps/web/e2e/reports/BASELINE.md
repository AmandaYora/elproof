# Baseline performa dan jumlah panggilan API

Diukur dengan `npm run e2e:perf` (garis dasar UI lama ada di `reports/before-t0/`; berkas ini selalu pengukuran terakhir). Profil: HP 375×812, UA Android Chrome, CPU 4×,
1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas
build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi
(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,
pengumuman), bukan hanya data halaman.

| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Beranda pemasaran | 0 | 0.0 | 12 | 155.7 | 1904 | 0.424 | 25 | - |
| Ringkasan Platform | 3 | 1.4 | 21 | 168.4 | 2672 | 0.127 | 24 | 48 |
| Tagihan Platform | 3 | 0.6 | 27 | 170.7 | 2420 | 0.044 | 23 | - |
| Daftar Tenant | 3 | 2.3 | 53 | 242.7 | 3204 | 0.044 | 25 | - |
| Detail Tenant | 4 | 1.9 | 49 | 227.8 | 3320 | 0.125 | 22 | - |
| Beranda Portal | 4 | 2.7 | 24 | 172.3 | 2424 | 0 | 22 | 368 |
| Portal, Jadwal | 4 | 2.7 | 21 | 171.2 | 2312 | 0 | 25 | - |
| Portal, Kendala | 4 | 2.4 | 24 | 172.7 | 2348 | 0 | 23 | - |
| Portal, Pembayaran | 5 | 2.6 | 23 | 173.0 | 2316 | 0 | 26 | - |
| Portal, Vendor | 6 | 3.4 | 28 | 176.2 | 2404 | 0 | 25 | - |
| Beranda WO | 6 | 6.7 | 31 | 215.0 | 2788 | 0.442 | 24 | 56 |
| Daftar Client | 6 | 4.1 | 39 | 235.8 | 2780 | 0.098 | 21 | - |
| Daftar Project | 5 | 3.6 | 28 | 185.0 | 2384 | 0.098 | 23 | - |
| Detail Proyek, tab Kendala | 7 | 5.2 | 46 | 252.7 | 1548 | 0.24 | 26 | - |
| Detail Proyek, tab Keuangan | 9 | 5.7 | 60 | 270.2 | 1524 | 0.196 | 24 | - |
| Detail Proyek (Ringkasan) | 5 | 3.6 | 43 | 249.3 | 1548 | 0.24 | 24 | - |
| Detail Proyek, tab Vendor | 7 | 5.2 | 56 | 272.6 | 1532 | 0.24 | 27 | - |
| Daftar Vendor | 6 | 3.2 | 40 | 233.8 | 1540 | 0.098 | 22 | - |

**Total JS seluruh aplikasi:** 1700.4 KB mentah, **588.6 KB gzip** (177 chunk).

Chunk terbesar (gzip):

- `index-DDJ9idaY.js`: 146.1 KB gzip (439.6 KB mentah)
- `ProjectMilestoneTabPage-C1gnRT75.js`: 87.3 KB gzip (264.2 KB mentah)
- `ProjectDetailLayout-Dh5Hb2r6.js`: 21.4 KB gzip (62.3 KB mentah)
- `DialogTitle-Dy5At2mk.js`: 19.1 KB gzip (55.5 KB mentah)
- `usePositioner-CvT2Gytk.js`: 14.8 KB gzip (39.7 KB mentah)
- `types-B4jcjfVQ.js`: 12.2 KB gzip (52.9 KB mentah)
- `MfaSetupPage-RHaEuqQx.js`: 11.9 KB gzip (30.4 KB mentah)
- `index-BmPwTbZw.js`: 9.6 KB gzip (34.3 KB mentah)
- `ProjectVendorTabPage-DNdehpyx.js`: 9.5 KB gzip (35.9 KB mentah)
- `DashboardPage-BH-CUi0T.js`: 7.6 KB gzip (27.2 KB mentah)

## Panggilan API per layar

**Beranda pemasaran** (0):


**Ringkasan Platform** (3):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/platform/trends?… (795 B)
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

**Beranda Portal** (4):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/milestones (941 B)

**Portal, Jadwal** (4):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/milestones (941 B)

**Portal, Kendala** (4):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/issues (535 B)

**Portal, Pembayaran** (5):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (400 B)

**Portal, Vendor** (6):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/venue (44 B)
- GET /api/v1/projects/1/issues (535 B)
- GET /api/v1/projects/1/vendors (1008 B)

**Beranda WO** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/dashboard (5522 B)
- GET /api/v1/tenants/me/onboarding (237 B)

**Daftar Client** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/clients?… (546 B)
- GET /api/v1/projects?… (2539 B)

**Daftar Project** (5):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects?… (2594 B)

**Detail Proyek, tab Kendala** (7):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2574 B)
- GET /api/v1/projects/1/issues (547 B)
- GET /api/v1/projects/1/vendors (1102 B)

**Detail Proyek, tab Keuangan** (9):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2574 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (431 B)
- GET /api/v1/projects/1/evidence (913 B)
- GET /api/v1/projects/1/payment-terms (392 B)

**Detail Proyek (Ringkasan)** (5):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2574 B)

**Detail Proyek, tab Vendor** (7):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2574 B)
- GET /api/v1/projects/1/issues (547 B)
- GET /api/v1/projects/1/vendors (1102 B)

**Daftar Vendor** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/vendors?… (1169 B)
