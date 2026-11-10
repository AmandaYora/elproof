# Baseline performa dan jumlah panggilan API

Diukur dengan `npm run e2e:perf` (garis dasar UI lama ada di `reports/before-t0/`; berkas ini selalu pengukuran terakhir). Profil: HP 375×812, UA Android Chrome, CPU 4×,
1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas
build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi
(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,
pengumuman), bukan hanya data halaman.

| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Beranda pemasaran | 0 | 0.0 | 12 | 156.4 | 1964 | 0.424 | 29 | - |
| Ringkasan Platform | 3 | 1.4 | 22 | 169.5 | 2732 | 0.083 | 27 | 16 |
| Tagihan Platform | 2 | 0.5 | 37 | 216.9 | 2780 | 0 | 24 | - |
| Daftar Tenant | 3 | 2.3 | 64 | 256.8 | 3544 | 0 | 23 | - |
| Detail Tenant | 4 | 1.9 | 54 | 231.4 | 3520 | 0.081 | 26 | - |
| Beranda Portal | 4 | 3.0 | 24 | 173.3 | 2452 | 0 | 24 | 384 |
| Portal, Jadwal | 4 | 3.0 | 22 | 172.4 | 2336 | 0 | 24 | - |
| Portal, Kendala | 4 | 2.4 | 25 | 174.2 | 2404 | 0 | 22 | - |
| Portal, Pembayaran | 5 | 2.6 | 24 | 174.4 | 2428 | 0 | 24 | - |
| Portal, Vendor | 6 | 3.5 | 29 | 177.6 | 2484 | 0 | 22 | - |
| Beranda WO | 6 | 6.8 | 36 | 219.1 | 3060 | 0.442 | 71 | 24 |
| Daftar Client | 6 | 4.6 | 49 | 250.1 | 3104 | 0.098 | 25 | - |
| Daftar Project | 6 | 4.1 | 38 | 231.6 | 2844 | 0.098 | 26 | - |
| Detail Proyek, tab Kendala | 7 | 5.4 | 55 | 258.3 | 1556 | 0.113 | 28 | - |
| Detail Proyek, tab Keuangan | 9 | 5.9 | 70 | 276.5 | 1572 | 0.113 | 27 | - |
| Detail Proyek (Ringkasan) | 5 | 3.7 | 52 | 254.9 | 1572 | 0.113 | 30 | - |
| Detail Proyek, tab Vendor | 7 | 5.4 | 65 | 278.7 | 1556 | 0.113 | 26 | - |
| Daftar Vendor | 6 | 3.3 | 50 | 248.3 | 1556 | 0.098 | 28 | - |

**Total JS seluruh aplikasi:** 1799.2 KB mentah, **629.5 KB gzip** (195 chunk).

Chunk terbesar (gzip):

- `index-D3w6M99y.js`: 146.8 KB gzip (442.9 KB mentah)
- `ProjectMilestoneTabPage-CBEoWq2c.js`: 87.8 KB gzip (265.4 KB mentah)
- `ProjectDetailLayout-BRfXyvc2.js`: 20.5 KB gzip (59.9 KB mentah)
- `CommandPaletteDialog-D7JjtleH.js`: 19.5 KB gzip (57.5 KB mentah)
- `usePositioner-ucm-NbkO.js`: 14.2 KB gzip (38.0 KB mentah)
- `store-BfbrsJsD.js`: 12.6 KB gzip (35.5 KB mentah)
- `types-B4jcjfVQ.js`: 12.2 KB gzip (52.9 KB mentah)
- `MfaSetupPage-M4p2c-fC.js`: 11.9 KB gzip (30.4 KB mentah)
- `index-B2TFLBt3.js`: 9.6 KB gzip (34.3 KB mentah)
- `ProjectVendorTabPage-pS8g0Y8-.js`: 9.6 KB gzip (36.0 KB mentah)

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
- GET /api/v1/projects/1/milestones (1193 B)

**Portal, Jadwal** (4):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/milestones (1193 B)

**Portal, Kendala** (4):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/issues (577 B)

**Portal, Pembayaran** (5):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (442 B)

**Portal, Vendor** (6):

- GET /api/v1/access/me (239 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/projects/me/home (1525 B)
- GET /api/v1/projects/1/venue (44 B)
- GET /api/v1/projects/1/issues (577 B)
- GET /api/v1/projects/1/vendors (1134 B)

**Beranda WO** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/dashboard (5606 B)
- GET /api/v1/tenants/me/onboarding (237 B)

**Daftar Client** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/clients?… (756 B)
- GET /api/v1/projects?… (2869 B)

**Daftar Project** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects?… (2874 B)

**Detail Proyek, tab Kendala** (7):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2616 B)
- GET /api/v1/projects/1/issues (589 B)
- GET /api/v1/projects/1/vendors (1228 B)

**Detail Proyek, tab Keuangan** (9):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2616 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (473 B)
- GET /api/v1/projects/1/evidence (1039 B)
- GET /api/v1/projects/1/payment-terms (392 B)

**Detail Proyek (Ringkasan)** (5):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2616 B)

**Detail Proyek, tab Vendor** (7):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1/overview (2616 B)
- GET /api/v1/projects/1/issues (589 B)
- GET /api/v1/projects/1/vendors (1228 B)

**Daftar Vendor** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/vendors?… (1289 B)
