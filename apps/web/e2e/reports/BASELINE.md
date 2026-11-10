# Baseline performa dan jumlah panggilan API

Diukur dengan `npm run e2e:perf` (garis dasar UI lama ada di `reports/before-t0/`; berkas ini selalu pengukuran terakhir). Profil: HP 375×812, UA Android Chrome, CPU 4×,
1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas
build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi
(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,
pengumuman), bukan hanya data halaman.

| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Beranda pemasaran | 0 | 0.0 | 12 | 156.1 | 1940 | 0.424 | 31 | - |
| Ringkasan Platform | 3 | 1.4 | 21 | 168.7 | 2696 | 0.083 | 27 | 16 |
| Tagihan Platform | 2 | 0.5 | 36 | 216.4 | 2756 | 0 | 33 | - |
| Daftar Tenant | 3 | 2.3 | 62 | 256.1 | 3472 | 0 | 25 | - |
| Detail Tenant | 4 | 1.9 | 53 | 230.9 | 3416 | 0.081 | 27 | - |
| Beranda Portal | 4 | 2.7 | 24 | 173.0 | 2448 | 0 | 35 | 360 |
| Portal, Jadwal | 4 | 2.7 | 21 | 171.7 | 2324 | 0 | 27 | - |
| Portal, Kendala | 4 | 2.4 | 24 | 173.5 | 2380 | 0 | 30 | - |
| Portal, Pembayaran | 5 | 2.6 | 23 | 173.7 | 2340 | 0 | 27 | - |
| Portal, Vendor | 6 | 3.4 | 28 | 177.0 | 2420 | 0 | 27 | - |
| Beranda WO | 6 | 6.7 | 34 | 217.9 | 2916 | 0.442 | 25 | 16 |
| Daftar Client | 6 | 4.5 | 48 | 249.5 | 3004 | 0.098 | 23 | - |
| Daftar Project | 6 | 4.0 | 37 | 231.0 | 2788 | 0.054 | 34 | - |
| Detail Proyek, tab Kendala | 7 | 5.2 | 53 | 257.4 | 1548 | 0.113 | 31 | - |
| Detail Proyek, tab Keuangan | 9 | 5.7 | 68 | 275.2 | 1536 | 0.113 | 24 | - |
| Detail Proyek (Ringkasan) | 5 | 3.6 | 50 | 254.0 | 1528 | 0.069 | 28 | - |
| Detail Proyek, tab Vendor | 7 | 5.2 | 63 | 277.4 | 1552 | 0.113 | 29 | - |
| Daftar Vendor | 6 | 3.3 | 49 | 247.9 | 1584 | 0.098 | 37 | - |

**Total JS seluruh aplikasi:** 1793.8 KB mentah, **626.5 KB gzip** (191 chunk).

Chunk terbesar (gzip):

- `index-Du76Lh60.js`: 146.6 KB gzip (442.3 KB mentah)
- `ProjectMilestoneTabPage-BQjFF5ia.js`: 87.4 KB gzip (264.4 KB mentah)
- `ProjectDetailLayout-CkC3XfNI.js`: 20.5 KB gzip (59.7 KB mentah)
- `CommandPaletteDialog-Bs2MSKA8.js`: 19.5 KB gzip (57.5 KB mentah)
- `usePositioner-BuR1lLHY.js`: 14.2 KB gzip (38.0 KB mentah)
- `store-B3BPt1LE.js`: 12.6 KB gzip (35.5 KB mentah)
- `types-B4jcjfVQ.js`: 12.2 KB gzip (52.9 KB mentah)
- `MfaSetupPage-pAwvEeEP.js`: 11.9 KB gzip (30.4 KB mentah)
- `index-gzBnxZX6.js`: 9.6 KB gzip (34.3 KB mentah)
- `ProjectVendorTabPage-CThorqMk.js`: 9.6 KB gzip (36.0 KB mentah)

## Panggilan API per layar

**Beranda pemasaran** (0):


**Ringkasan Platform** (3):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/platform/trends?… (795 B)
- GET /api/v1/platform/metrics (422 B)

**Tagihan Platform** (2):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/subscription-transactions?… (352 B)

**Daftar Tenant** (3):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/plans?… (392 B)
- GET /api/v1/tenants?… (1783 B)

**Detail Tenant** (4):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/plans?… (392 B)
- GET /api/v1/subscription-transactions?… (122 B)
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
- GET /api/v1/clients?… (756 B)
- GET /api/v1/projects?… (2743 B)

**Daftar Project** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects?… (2748 B)

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
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2574 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (431 B)
- GET /api/v1/projects/1/evidence (913 B)
- GET /api/v1/projects/1/payment-terms (392 B)

**Detail Proyek (Ringkasan)** (5):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/tenants/me/branding (110 B)
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
- GET /api/v1/vendors?… (1289 B)
