# Baseline performa dan jumlah panggilan API

Diukur dengan `npm run e2e:perf` (garis dasar UI lama ada di `reports/before-t0/`; berkas ini selalu pengukuran terakhir). Profil: HP 375×812, UA Android Chrome, CPU 4×,
1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas
build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi
(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,
pengumuman), bukan hanya data halaman.

| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Beranda pemasaran | 0 | 0.0 | 12 | 155.8 | 1924 | 0.424 | 26 | - |
| Ringkasan Platform | 3 | 1.4 | 21 | 168.5 | 2644 | 0.127 | 22 | 48 |
| Tagihan Platform | 2 | 0.5 | 33 | 214.3 | 2648 | 0.044 | 22 | - |
| Daftar Tenant | 3 | 2.3 | 58 | 253.8 | 3356 | 0.044 | 22 | - |
| Detail Tenant | 4 | 1.9 | 51 | 229.0 | 3328 | 0.125 | 22 | - |
| Beranda Portal | 4 | 2.7 | 24 | 172.4 | 2416 | 0 | 23 | 360 |
| Portal, Jadwal | 4 | 2.7 | 21 | 171.4 | 2312 | 0 | 23 | - |
| Portal, Kendala | 4 | 2.4 | 24 | 172.9 | 2352 | 0 | 22 | - |
| Portal, Pembayaran | 5 | 2.6 | 23 | 173.2 | 2320 | 0 | 24 | - |
| Portal, Vendor | 6 | 3.4 | 28 | 176.4 | 2408 | 0 | 22 | - |
| Beranda WO | 6 | 6.7 | 33 | 216.3 | 2836 | 0.442 | 25 | 56 |
| Daftar Client | 6 | 4.5 | 44 | 247.0 | 2952 | 0.054 | 24 | - |
| Daftar Project | 6 | 4.0 | 34 | 228.6 | 2704 | 0.098 | 23 | - |
| Detail Proyek, tab Kendala | 7 | 5.2 | 50 | 254.9 | 1548 | 0.24 | 28 | - |
| Detail Proyek, tab Keuangan | 9 | 5.7 | 64 | 272.5 | 1536 | 0.196 | 31 | - |
| Detail Proyek (Ringkasan) | 5 | 3.6 | 47 | 251.5 | 1540 | 0.24 | 27 | - |
| Detail Proyek, tab Vendor | 7 | 5.2 | 60 | 274.9 | 1532 | 0.24 | 27 | - |
| Daftar Vendor | 6 | 3.3 | 45 | 245.5 | 1544 | 0.098 | 28 | - |

**Total JS seluruh aplikasi:** 1726.1 KB mentah, **601.8 KB gzip** (182 chunk).

Chunk terbesar (gzip):

- `index-DE3VUrpR.js`: 146.2 KB gzip (440.1 KB mentah)
- `ProjectMilestoneTabPage-BpcdlTSi.js`: 87.3 KB gzip (264.3 KB mentah)
- `ProjectDetailLayout-m7F7vPzr.js`: 20.4 KB gzip (59.4 KB mentah)
- `useScrollLock-B48kFFs-.js`: 17.2 KB gzip (49.2 KB mentah)
- `usePositioner-B5ib_XBc.js`: 14.2 KB gzip (38.0 KB mentah)
- `types-B4jcjfVQ.js`: 12.2 KB gzip (52.9 KB mentah)
- `MfaSetupPage-cw0pE0Ys.js`: 11.9 KB gzip (30.4 KB mentah)
- `index-QjTtyh93.js`: 9.6 KB gzip (34.3 KB mentah)
- `ProjectVendorTabPage-L-zqwpra.js`: 9.6 KB gzip (36.0 KB mentah)
- `useSavedViews-Z-gZ6tvd.js`: 8.0 KB gzip (21.5 KB mentah)

## Panggilan API per layar

**Beranda pemasaran** (0):


**Ringkasan Platform** (3):

- GET /api/v1/platform-admins/1 (211 B)
- GET /api/v1/platform/metrics (422 B)
- GET /api/v1/platform/trends?… (795 B)

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
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/clients?… (756 B)
- GET /api/v1/projects?… (2743 B)

**Daftar Project** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
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
- GET /api/v1/vendors?… (1289 B)
