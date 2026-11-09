# Baseline performa dan jumlah panggilan API

Diukur dengan `npm run e2e:perf` pada UI lama (sebelum revamp). Profil: HP 375×812, UA Android Chrome, CPU 4×,
1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas
build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi
(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,
pengumuman), bukan hanya data halaman.

| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Beranda pemasaran | 0 | 0.0 | 12 | 155.4 | 1908 | 0.424 | 21 | - |
| Ringkasan Platform | 3 | 1.3 | 21 | 168.2 | 2656 | 0.127 | 18 | 48 |
| Tagihan Platform | 3 | 0.6 | 27 | 170.5 | 2432 | 0.044 | 19 | - |
| Daftar Tenant | 3 | 2.3 | 52 | 240.6 | 3176 | 0.044 | 20 | - |
| Detail Tenant | 4 | 1.9 | 48 | 225.1 | 3252 | 0.125 | 20 | - |
| Beranda Portal | 4 | 2.7 | 24 | 172.0 | 2420 | 0 | 19 | 376 |
| Portal, Jadwal | 4 | 2.7 | 23 | 171.2 | 2300 | 0 | 19 | - |
| Portal, Kendala | 4 | 2.3 | 26 | 172.8 | 2364 | 0 | 20 | - |
| Portal, Pembayaran | 5 | 2.6 | 25 | 173.0 | 2396 | 0 | 19 | - |
| Portal, Vendor | 6 | 3.3 | 30 | 176.2 | 2516 | 0 | 20 | - |
| Beranda WO | 8 | 7.2 | 33 | 213.3 | 2912 | 0.442 | 53 | 64 |
| Daftar Client | 6 | 4.1 | 40 | 233.8 | 2844 | 0.054 | 19 | - |
| Daftar Project | 5 | 3.6 | 29 | 184.9 | 2400 | 0.098 | 21 | - |
| Detail Proyek, tab Kendala | 16 | 8.8 | 43 | 246.4 | 1528 | 0.24 | 19 | - |
| Detail Proyek, tab Keuangan | 19 | 9.1 | 60 | 267.3 | 1532 | 0.24 | 18 | - |
| Detail Proyek (Ringkasan) | 16 | 8.2 | 43 | 247.2 | 1516 | 0.24 | 18 | - |
| Detail Proyek, tab Vendor | 24 | 16.8 | 54 | 267.3 | 1520 | 0.24 | 20 | - |
| Daftar Vendor | 6 | 3.2 | 39 | 231.5 | 1528 | 0.098 | 20 | - |

**Total JS seluruh aplikasi:** 1679.5 KB mentah, **579.0 KB gzip** (174 chunk).

Chunk terbesar (gzip):

- `index-CBiG6aO1.js`: 145.8 KB gzip (438.8 KB mentah)
- `ProjectMilestoneTabPage-DrhuViUO.js`: 87.1 KB gzip (263.2 KB mentah)
- `ProjectDetailLayout-0Omj4ioL.js`: 21.6 KB gzip (63.2 KB mentah)
- `DialogTitle-Cmr3vm_i.js`: 19.1 KB gzip (55.5 KB mentah)
- `usePositioner-Bk1DOpNR.js`: 14.8 KB gzip (39.7 KB mentah)
- `types-B4jcjfVQ.js`: 12.2 KB gzip (52.9 KB mentah)
- `MfaSetupPage-D6EQFOV2.js`: 11.8 KB gzip (30.2 KB mentah)
- `ProjectVendorTabPage-SCvh1n6p.js`: 9.8 KB gzip (37.5 KB mentah)
- `index-B6D7KlcD.js`: 9.6 KB gzip (34.3 KB mentah)
- `DashboardPage-Bd9eHBYa.js`: 7.0 KB gzip (24.2 KB mentah)

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
- GET /api/v1/projects/1/issues (499 B)

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
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/vendors (946 B)

**Beranda WO** (8):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/dashboard (4690 B)
- GET /api/v1/staff/summary (210 B)
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
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/client-payments (431 B)
- GET /api/v1/projects/1/milestones (941 B)
- GET /api/v1/projects/1/vendors (1016 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/vendors (1016 B)

**Detail Proyek, tab Keuangan** (19):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1 (900 B)
- GET /api/v1/projects/1/milestones (941 B)
- GET /api/v1/projects/1/client-payments (431 B)
- GET /api/v1/projects/1/vendors (1016 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/evidence (817 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/client-payments (431 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (499 B)

**Detail Proyek (Ringkasan)** (16):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1 (900 B)
- GET /api/v1/projects/1/milestones (941 B)
- GET /api/v1/projects/1/vendors (1016 B)
- GET /api/v1/projects/1/client-payments (431 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (499 B)

**Detail Proyek, tab Vendor** (24):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/projects/1 (900 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/client-payments (431 B)
- GET /api/v1/projects/1/milestones (941 B)
- GET /api/v1/projects/1/vendors (1016 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/payment-terms (392 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/staff/summary (210 B)
- GET /api/v1/projects/1/payments (212 B)
- GET /api/v1/projects/1/issues (499 B)
- GET /api/v1/projects/1/vendors (1016 B)
- GET /api/v1/projects/1/evidence (817 B)
- GET /api/v1/vendors?… (1114 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/projects/1/activity (3609 B)
- GET /api/v1/staff/summary (210 B)

**Daftar Vendor** (6):

- GET /api/v1/access/me (518 B)
- GET /api/v1/tenants/me/branding (110 B)
- GET /api/v1/tenants/me/subscription (200 B)
- GET /api/v1/announcements/active (306 B)
- GET /api/v1/vendor-categories?… (972 B)
- GET /api/v1/vendors?… (1169 B)
