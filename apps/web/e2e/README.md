# e2e: harness Playwright, potret dasar, axe, dan garis dasar performa

Pengaman untuk revamp UI (`docs/plan/revamp-ui-shadcn/PLAN.md` Tahap 0). Potret dasar diambil dari **UI lama** sebelum
perubahan visual pertama; setiap tahap harus menunjukkan selisih yang disengaja saja.

## Menjalankan

```bash
npm run e2e                 # potret desktop + mobile dan axe (186 tes)
npm run e2e:perf            # pengukuran garis dasar performa (sendirian: ukuran terganggu bila berbarengan dengan tes lain)
npm run e2e:update          # perbarui potret (desktop + mobile) setelah perubahan visual yang disengaja
E2E_REUSE=1 npm run e2e     # pakai tumpukan yang sudah menyala (cepat); mati dengan `npm run e2e:down`
npm run e2e:stack           # hanya menyalakan tumpukan untuk dipakai manual (sesi login di e2e/.tmp/auth/*.json)
npm run e2e:typecheck       # tsc untuk folder e2e (tidak ikut `tsc -b` aplikasi)
AXE_STRICT=1 npm run e2e -- --project=desktop tests/axe.spec.ts   # gagal bila ada pelanggaran serius/kritis
```

Yang dijalankan `globalSetup` (semua dari TypeScript/Node, tanpa kode tes Go): menyalakan MySQL sementara bila belum ada,
membangun biner API (`go build`), membuat basis data sementara `elproof_e2e_pw` (dibuang di akhir), `migrate up` dan
`seed` (satu Super Admin + satu paket), menyalakan Google palsu (JWKS RS256 + `/token`), API, lalu **build produksi** web
(`vite build`, `VITE_API_BASE_URL` menunjuk API) yang disajikan `vite preview`. Setelah itu data benih dibuat lewat API.

## Variabel lingkungan

| Variabel | Bawaan | Arti |
|---|---|---|
| `E2E_MYSQL_ADMIN` | `root@127.0.0.1:3399` | `user[:pass]@host:port` untuk membuat basis data sementara. Bila bawaan dan belum menyala, `e2e/scripts/start-mysql.sh` menyalakan MySQL 8 (Homebrew) dengan data di `e2e/.tmp/mysql`. MySQL di :3306 dan basis data lain tidak pernah disentuh |
| `E2E_MYSQL_BIN` | `mysql` | klien CLI |
| `E2E_DB_NAME` | `elproof_e2e_pw` | nama basis data sementara |
| `E2E_API_PORT` / `E2E_WEB_PORT` / `E2E_GOOGLE_PORT` | 18091 / 14173 / 18099 | port tetap (origin sesi login ikut port) |
| `E2E_S3_ENDPOINT`, `E2E_S3_ACCESS_KEY`, `E2E_S3_SECRET_KEY` | kosong | tanpa ini API memakai `127.0.0.1:9` dan **tidak ada dokumen bukti** yang dibenihkan (tab Dokumen kosong). Dengan MinIO lokal, dokumen "Kontrak katering" ikut dibuat |
| `E2E_REUSE` | kosong | `1` = pakai tumpukan yang menyala, jangan matikan di akhir |
| `E2E_WORKERS` | 3 | jumlah worker potret |
| `E2E_API_WRAPPER` | kosong | awalan perintah untuk API, mis. `faketime -f "@2026-11-09 03:00:00"` untuk membekukan jam server (lihat Determinisme) |
| `AXE_STRICT` | kosong | `1` = axe gagal pada pelanggaran serius/kritis |

## Isi

- `lib/` tumpukan (`stack.ts`), benih (`seed.ts`), Google palsu, TOTP, daftar halaman emas (`pages.ts`), fixture (`fixtures.ts`).
- `tests/wo|portal|platform|marketing.spec.ts` potret; `axe.spec.ts` axe; `baseline.spec.ts` performa. Dua proyek viewport:
  **desktop 1280×800** dan **mobile 375×812** (`isMobile`, `hasTouch`, UA Android Chrome, DPR 1).
- `reporters/summary-reporter.ts` menulis `reports/axe-baseline.json`, `reports/baseline.json`, `reports/BASELINE.md`.
- `snapshots/<konsol>/<halaman>-<keadaan>-<viewport>.png` potret dasar (tanpa akhiran OS).

Keadaan per halaman: **normal**; **empty** (workspace "WO Kosong": Beranda, Proyek, Klien, Vendor, Venue); **error** (permintaan
API utama dibatalkan lewat `route.abort`, tampil kondisi galat halaman); **dialog** (WO: Tambah Vendor, Tambah Pembayaran;
Platform: Tambah Tenant). Portal Klien tidak punya dialog di baseline (satu-satunya, penampil bukti, butuh penyimpanan objek).

## Data benih (deterministik)

Workspace **WO Contoh** (preset navy): Owner "Rina Marlina", 2 staf lain, 3 vendor, 2 venue; proyek **Citra & Bima** (Persiapan, termin
terlambat, 2 tahapan selesai, 2 vendor, pembayaran klien dan vendor, 1 kendala, 2 klien: Bride dengan login `citra.contoh` /
`Kata-sandi-klien-123`), **Dewi & Arga** (Selesai), **Sari & Doni** (Draf). Workspace **WO Berbayar** (langganan aktif dan satu
transaksi lunas) dan **WO Kosong** (tanpa data). Platform: Super Admin `superadmin` dengan 2FA aktif (TOTP dihitung di Node),
satu pengumuman, satu permintaan data. Semua tanggal benih dihitung dari hari tetap **9 Nov 2026**, bukan jam nyata.

## Determinisme dan batasnya

Dibuat tetap: jam peramban dibekukan di **9 Nov 2026 10:00 WIB** (`page.clock.setFixedTime`, timer tetap jalan; zona
`Asia/Jakarta`, locale `id-ID`), animasi dan gerak dimatikan, font Google (Inter, Fraunces) disajikan dari cache lokal
(`e2e/.tmp/fonts`) setelah unduhan pertama, service worker diblokir, semua cap waktu basis data (created_at, paid_at, audit)
dinormalkan ke 10:00 WIB + id detik, dan tunggu jaringan idle + font siap + teks "Memuat" hilang.

**Batas yang jujur:**
1. **Jam server tidak bisa dibekukan** tanpa alat tambahan. Teks yang dihitung server terhadap jam nyata (sisa hari masa coba,
   "terlambat N hari") **ditutup magenta** di setiap potret (`volatile()` di `lib/fixtures.ts`). Data benih berpatokan pada
   9 Nov 2026; bila dijalankan jauh setelah tanggal itu, angka seperti "jatuh tempo ≤ 7 hari" dapat berubah. Di CI pakai
   `libfaketime` lewat `E2E_API_WRAPPER`.
2. **Potret berbeda per platform.** Baseline ini diambil di macOS (Chromium 156 dari Playwright 1.64). Rendering teks di Linux
   berbeda sedikit; CI harus memakai satu lingkungan saja dan membuat ulang potret di sana (`npm run e2e:update`) sebelum
   dipakai sebagai gerbang. Ambang `maxDiffPixelRatio: 0.01`.
3. Halaman penuh (`fullPage`) mengambil bilah yang `fixed` (tab bawah Portal, sidebar) pada posisi viewport, jadi tab bawah
   tampak di tengah gambar mobile yang panjang. Itu perilaku tangkapan, bukan bug aplikasi.
4. Teks bawaan peramban (mis. "Choose File") mengikuti bahasa UI peramban, bukan locale halaman.
5. Fixture axe memakai `playwright-core` 1.65 milik `@axe-core/playwright` berdampingan dengan 1.64: API sama, tipe diberi `as never`.

## Garis dasar performa

`npm run e2e:perf` mengukur tiap rute pada profil HP: CPU 4×, 1,6 Mbps / 150 ms RTT, cache dingin, build produksi.
Dicatat: jumlah respons `/api/` sampai jaringan idle dan bobot JSON, chunk JS yang dimuat dan ukuran gzip-nya, LCP, CLS, TBT-like
(jumlah tugas panjang di atas 50 ms) dan **INP-like** (durasi acara terbesar pada satu klik navigasi). **Tidak terukur:** INP
lapangan sungguhan (tidak bisa dari skrip), LCP/INP di perangkat Android fisik, dan performa di jaringan nyata (font diambil dari
cache lokal). Lihat `reports/BASELINE.md`.

## CI

Pakai gambar resmi dengan versi yang sama dengan `@playwright/test` (sekarang **1.64.0**):
`mcr.microsoft.com/playwright:v1.64.0-noble`, jalankan `npm ci` lalu `npm run e2e`. Butuh MySQL 8 (service container) dan Go.
`E2E_MYSQL_ADMIN` menunjuk ke servicenya; `start-mysql.sh` hanya untuk lokal macOS/Homebrew.
