# PLAN.md — Revamp UI ke shadcn/ui

> Status: **RENCANA, belum dikerjakan. Disusun 8 Nov 2026. D1 diputuskan: Base UI (8 Nov 2026).** Tahap 0 siap dimulai;
> keputusan D2–D12 (§8) masih usulan dan dikonfirmasi saat tahap terkait.
> Dasar: riset internet dan inventaris kode, dirangkum di `knowledge/DESIGN.md` (brief desain, temuan → keputusan,
> daftar sumber dengan tingkat verifikasi). Alur dan menu per peran tidak berubah: `FLOW-IDEAL.md`.

## 0. Ringkasan

Aplikasi sudah punya sistem desain buatan sendiri yang konsisten (token, dua kepadatan, 21 preset warna tenant,
~35 komponen), tetapi perilaku aksesibilitas, overlay, dan combobox ditulis tangan dan hanya teruji sejauh yang pernah
dicoba. Revamp ini memindahkan **lapisan komponen** ke shadcn/ui di atas primitif headless (Radix atau Base UI) tanpa
mengubah alur, API, atau data, dan **tanpa menulis ulang semua halaman sekaligus**: komponen diganti dari dalam, nama
dan props lama tetap dipakai sampai pemanggil dimigrasikan.

Temuan riset yang paling menentukan rencana ini:

1. **shadcn/ui sudah memakai Base UI sebagai default sejak Juli 2026**; Radix "tidak dihentikan" dan tetap
   mendapat pembaruan, dan ada `init -b radix` serta skill migrasi per komponen. Panduan resmi: proyek baru pakai
   Base UI, aplikasi yang sudah jalan di Radix tidak perlu pindah. Aplikasi kita **belum punya pustaka komponen sama
   sekali**, jadi bagi shadcn ini proyek baru. Pilihan Radix vs Base UI harus diputuskan eksplisit (D1).
2. Pemeliharaan Radix **melambat tetapi hidup**: rilis 6 Jun, 30 Jun, 6 Jul, 20 Jul 2026 dan tidak ada sesudahnya di
   halaman rilis; Base UI stabil 1.0 sejak 11 Des 2025 dan rilis bulanan (v1.9.0, 9 Okt 2026), dengan Combobox dan Toast
   bawaan.
3. **Dua masalah kontras di token sekarang** (dihitung): batas input `#d6e0ea` di atas putih hanya 1,34:1 (WCAG 1.4.11
   minta 3:1) dan teks sekunder di atas `--color-surface-muted` 4,34:1 (AA minta 4,5:1). Ini diperbaiki bersama revamp.
4. Warna tenant hanya 21 preset yang sudah diverifikasi kontrasnya, sehingga **tidak perlu pembuat skala warna
   otomatis**; cukup memetakan preset ke token semantik shadcn. Pola penimpaan CSS variable saat runtime sudah terbukti di
   aplikasi ini (`@theme inline` di `theme.css`).
5. Elemen khas desain adalah **jejak bukti** (stempel `ProofStamp` turunan `ProofSeal` di atas pembayaran/tonggak yang
   benar-benar terverifikasi); selebihnya sengaja tenang.

## 1. Kondisi sekarang (terukur 8 Nov 2026)

| Hal | Angka |
|---|---|
| Berkas `.tsx` / baris TS+TSX | 194 / ±33.500 |
| Halaman / modal+drawer / form dengan `safeParse` | 60 / 31 / 33 |
| Komponen bersama (`shared/components`) | ±35 |
| Pemakaian: Button 78 berkas, Input 56, Card 53, Badge 48, Modal 41, EmptyState 36 | |
| `<Select` (alias Combobox, anak `<option>`) / `<Field` / `<Textarea` | 69 (33 berkas) / 249 / 32 |
| `type="date"` / `datetime-local` | 25 / 4 |
| Panggilan `toast.*` | 36 |
| `<button>` mentah di luar komponen bersama / `<input>` mentah | 42 / 26 |
| Aksesibilitas manual | `useDialogBehavior` (3 pemakai), 153 atribut `aria-*` |
| Pustaka UI pihak ketiga | hanya `lucide-react` dan `@svar-ui/react-gantt` |
| Tes UI otomatis / Storybook | tidak ada / tidak ada |
| Mode gelap | belum (3 `dark:`) |

## 2. Tujuan dan bukan tujuan

**Tujuan**
- Komponen dasar berstandar shadcn/ui + primitif headless yang dirawat pihak lain (fokus, ARIA, papan ketik, tumbukan).
- Memenuhi WCAG 2.2 AA pada tiga console, dengan gerbang otomatis (axe) dan manual (papan ketik, iOS Safari).
- Satu bahasa visual: token semantik, dua kepadatan, jejak bukti, keadaan daftar yang seragam.
- Tidak ada regresi fungsi, tidak ada perubahan API/DB, tidak ada big-bang.

**Bukan tujuan** (sengaja, dengan alasan di `DESIGN.md` §8): mode gelap, glass/translusen, asisten AI, palet perintah
sebagai navigasi utama, Recharts (grafik buatan sendiri tetap), Storybook, penulisan ulang alur atau menu,
pembuat warna bebas untuk tenant.

## 3. Arsitektur target

- `shadcn init` di Vite, `components.json` dengan alias `@/*`; komponen salinan shadcn tinggal di
  **`src/shared/components/ui/`** (lokasi yang sama dengan sekarang, sehingga impor lama tetap berlaku) dan tetap
  **bebas domain** (aturan `.claude/rules/frontend-react.md`). Versi CLI **dikunci** (`shadcn` 4.x; perubahannya cepat).
- **Token:** `theme.css` tetap sumber kebenaran; blok `@theme inline` memetakan token shadcn (`--primary`, `--ring`,
  `--sidebar`, ...) ke token yang ada (`DESIGN.md` §4). `applyBrandColorPreset` tetap menimpa `--brand-navy-*`; token
  semantik hanya merujuknya, jadi 21 preset otomatis berlaku. Variabel tenant tetap di `:root`/`body` agar konten yang
  dirender di portal ikut mewarisinya.
- **Strangler:** nama ekspor lama (`Button`, `Modal`, `ConfirmDialog`, `Drawer`, `Select`, `Table`, `toast`, ...)
  dipertahankan sebagai pembungkus tipis di atas komponen shadcn, sehingga ratusan titik panggil tidak berubah di
  tahap awal. API komposisi baru (`Dialog*`, `Field*`, `Sheet*`) dipakai untuk layar baru dan dimigrasikan per modul.
- **Formulir:** komponen `Field` shadcn (presentasional, menerima `issues` Zod langsung lewat Standard Schema) di atas pola
  `safeParse` yang sudah ada. react-hook-form hanya ditambahkan pada formulir yang membutuhkannya (banyak field, larik,
  `dirty` lintas field). Tidak ada migrasi massal ke react-hook-form.
- **Toast:** pembungkus `toast.success/error/info` dipertahankan; mesin di baliknya Sonner (D4).
- **Tanggal:** `type="date"` bawaan dipertahankan (pemilih native terbaik di HP); `Calendar` shadcn (react-day-picker,
  paling berat dan paling sulit dilokalkan id-ID) hanya bila ada kebutuhan rentang.
- **Gantt, MilestoneRail, ProgressMeter, QrCode, EvidenceViewer:** tidak ada padanan shadcn; dipertahankan dan hanya
  di-restyle ke token baru.
- **Pengaman:** pengujian visual Playwright (§6), `@axe-core/playwright`, anggaran bundel dan INP.

### Pemetaan komponen

| Sekarang (pemakai) | Target | Catatan |
|---|---|---|
| Button (78) | `Button` (+ varian `neutral` Login, `icon`) | `neutral` tidak terikat warna tenant, dipertahankan |
| Input/Textarea (56/32) | `Input`, `Textarea`; `Field`, `FieldLabel`, `FieldError`, `FieldDescription` | batas input 3:1 (D tokens) |
| Card (53), StatCard (1) | `Card` | kartu tidak seragam: tagihan berikutnya lebih besar |
| Badge (48) + `TenantStatusBadge` dll. | `Badge` + `StatusChip` (ikon + teks + warna) | label tetap dari `shared/lib/labels.ts` |
| Modal (41), AccessLinkDialog, Evidence*Modal, ImportBulkModal | `Dialog` | modal penuh-layar untuk viewer bukti |
| ConfirmDialog (6) | `AlertDialog` | fokus ke "Batal"; ketik nama dipertahankan |
| Drawer (3) | `Sheet` | juga untuk menu mobile |
| Combobox/`Select` (5 / 69 titik) | `Select` + `Combobox` | `Select` lama menerima anak `<option>`; pembungkus tetap menerimanya |
| ActionMenu (1), IconActionButton (19) | `DropdownMenu`; `Button size=icon` + `Tooltip` | ≥ 24 px; nama aksesibel wajib |
| TabNav (6 titik) | `Tabs` bergaya, dibungkus `NavLink` untuk tab berbasis rute | tab dalam-halaman memakai `Tabs` biasa |
| Table (19), CardList (18) | `Table` + kartu mobile; TanStack Table hanya bila perlu | angka rata kanan, header sticky |
| Pagination (18) | `Pagination` | tampil total dan ukuran halaman |
| EmptyState (36), LoadError (20), Skeleton (7) | tetap komponen sendiri atas `Alert`/`Skeleton` | satu pola empat keadaan |
| Toaster (3), `toast` (36) | Sonner di balik `toast.*` | D4 |
| CurrencyInput (10), SearchInput (10) | `Input` + logika yang sudah ada | satu pemformat rupiah (D5) |
| Avatar (6), ProgressMeter (3) | `Avatar`; tetap sendiri | |
| Sidebar WO/Platform | `Sidebar` shadcn | Portal Klien **tidak** memakainya |

## 4. Tahap

Setiap tahap: cabang `revamp/ui-t<N>`, satu komit per komponen/berkas, boleh berhenti di akhir tahap mana pun dengan
aplikasi yang utuh. **Gerbang tahap** (wajib lolos sebelum lanjut): `tsc` dan build bersih; selisih visual Playwright
disetujui (bukan diabaikan); axe tanpa pelanggaran serius/kritis pada rute yang tersentuh; uji papan ketik halaman yang
berubah; INP p75 ≤ 200 ms dan anggaran bundel tidak terlampaui; tidak ada galat konsol.

### Tahap 0 — Fondasi dan pengaman (sebelum menyentuh satu komponen)
- `shadcn init` (Vite) dengan basis **Base UI** (D1), kunci versi CLI; pastikan `init` tidak menimpa
  `theme.css` (jalankan pada cabang, tinjau diff). Cek `add sonner --dry-run` untuk D4.
- Lapisan token (`DESIGN.md` §4): pemetaan semantik, **perbaikan dua kontras** (`--input` ±#7b8ba1; `muted-foreground`
  atau larangan teks sekunder di atas muted), skala z-index, `isolation: isolate` di root, token kepadatan
  `--control-h/--row-y/--gap/--text` per `data-density`.
- Satu pemformat uang (D5) menggantikan format tersebar.
- **Harness Playwright** (`apps/web/e2e/`): menjalankan tumpukan lokal (MySQL sementara + API + data benih dibuat lewat
  API, memakai pola `apps/api/tests/isolation`), potret halaman emas di 1280 px dan 375 px (WO Console, Portal, Platform
  Console, halaman pemasaran, Login) dengan keadaan: normal, kosong, galat, dialog terbuka; plus `@axe-core/playwright`.
  Potret dasar diambil **sebelum** perubahan visual pertama dan di-commit; CI memakai satu OS/peramban (gambar Docker
  Playwright) karena potret berbeda per platform.
- Ukur garis dasar: ukuran bundel per rute (`rollup-plugin-visualizer`), INP, Lighthouse.
- Prototipe tiga risiko sebelum berkomitmen: (a) Dialog + Select di dalam Dialog + scroll lock di iOS Safari nyata,
  (b) variabel tenant di konten yang dirender di portal (uji 3 preset terang dan gelap-ish: gold, emerald, navy),
  (c) pembungkus `Select` yang menerima `<option>`.
- **Selesai bila:** harness hijau pada kode lama, token dipetakan tanpa selisih visual selain dua perbaikan kontras,
  tiga prototipe lulus atau keputusan D1 diubah berdasarkan hasilnya.

### Tahap 1 — Primitif dasar
Button, Input/Textarea/Field (+ `FieldError`), Badge dan `StatusChip`, Card, Skeleton, Avatar, Separator, Tooltip, Progress.
Pembungkus mempertahankan props lama. **Selesai bila:** 78+56+53+48 titik panggil tidak berubah di sumber dan potret
hanya berubah pada perbaikan yang disengaja; 42 `<button>` mentah dan 26 `<input>` mentah diganti.

### Tahap 2 — Lapisan overlay
Dialog (Modal), AlertDialog (ConfirmDialog), Sheet (Drawer), DropdownMenu (ActionMenu), Popover, Tabs/TabNav, Sonner
(`toast`). `useDialogBehavior` dipensiunkan. Perilaku yang sudah dibangun dipertahankan dan **diuji**: modal menunggu
hasil simpan, galat di dalam modal, Escape hanya menutup dialog teratas, fokus kembali ke pemicu.
**Selesai bila:** 41 pemakai Modal dan 6 ConfirmDialog tanpa perubahan sumber; uji manual iOS Safari dan Android Chrome
(input di dalam dialog, papan ketik, scroll lock) lulus; tidak ada `role="dialog"` buatan sendiri tersisa.

### Tahap 3 — Formulir
`Field` di 33 formulir (urut dari yang paling banyak dipakai): aturan validasi hibrida (`DESIGN.md` T8), ringkasan galat
berfokus bila ≥ 3, `aria-invalid` + `aria-describedby`, tanda wajib tidak hanya warna, tombol kirim tidak dinonaktifkan karena
validitas, input tidak dihapus. `Select`/`Combobox` baru menggantikan Combobox lama (69 titik lewat pembungkus).
Satu formulir yang berat (Buat Proyek) dicoba dengan react-hook-form sebagai ukuran manfaatnya.
**Selesai bila:** semua formulir memenuhi aturan di atas (daftar periksa per formulir di komit), pengujian papan ketik
penuh pada tiga formulir representatif (Buat Proyek, Peran & Akses, Daftar/Login).

### Tahap 4 — Tabel dan daftar
`Table` shadcn dengan angka rata kanan + `tabular-nums`, dua kepadatan (40/32 px), header sticky yang tidak menutup
fokus, header sortable (`th` + `button` + `aria-sort`) dan urutan default, empat keadaan seragam, `Pagination` dengan total,
daftar kartu di HP dengan kolom pertama terkunci. Mulai dari Platform Console (Tenant, Tagihan, Audit Log) lalu
WO Console (Project, Vendor, Klien, Venue, Staf). TanStack Table hanya bila sortir/visibilitas kolom dibutuhkan.
**Selesai bila:** 19 pemakai Table dan 18 CardList memakai pola baru; tabel invoice lolos potret + axe; INP daftar
panjang tidak memburuk.

### Tahap 5 — Layout dan navigasi
`Sidebar` shadcn untuk WO dan Platform (Sheet di HP, `Cmd/Ctrl+B`), `Breadcrumb` di halaman detail (Tenant 360°,
Detail Proyek), banner (`Alert`) untuk langganan, pengumuman, mode "lihat sebagai". Portal Klien: tab bawah ≥ 44 px
buatan sendiri, target sentuh ≥ 44 px, tanpa fitur yang hanya terjangkau lewat hamburger. Fraunces hanya untuk judul
Beranda Portal bila D8 disetujui.
**Selesai bila:** perilaku peran/izin (menu tersembunyi, tombol tertulis-terkunci) identik dengan sebelum revamp;
uji target sentuh di HP nyata; banner sesi dukungan tetap di atas segalanya (z-index 70).

### Tahap 6 — Layar khusus dan elemen khas
`ProofStamp` di pembayaran terverifikasi dan tonggak berbukti (data sudah ada: tanggal, pemverifikasi, berkas bukti);
EvidenceViewer dan timeline rel Portal di-restyle; Gantt (`@svar-ui`) disesuaikan temanya; grafik TrendBarChart
di-restyle; opsional palet perintah (`Command`) di console bila D9 disetujui; keadaan data basi.
**Selesai bila:** cap tidak pernah muncul tanpa bukti (tes), animasi hormati `prefers-reduced-motion`.

### Tahap 7 — Pengerasan dan penutupan
Axe pada seluruh rute dan semua keadaan; uji papan ketik dan pembaca layar manual pada alur utama (masuk, buat proyek,
catat pembayaran, lihat bukti, bayar langganan); iOS Safari dan Android kelas menengah nyata; anggaran performa sebagai
gerbang CI; hapus komponen lama dan `useDialogBehavior`; dokumentasi (ADR-0032, `FRONTEND_GUIDE.md`, aturan
`.claude/rules/frontend-react.md`); inventaris ulang angka §1. **Selesai bila:** tidak ada komponen buatan sendiri yang
menggantikan fungsi shadcn, semua gerbang hijau di CI.

## 5. Estimasi

| Tahap | Perkiraan (1 pengembang, hari kerja) |
|---|---|
| 0 Fondasi, harness, prototipe | 4–6 |
| 1 Primitif | 4–5 |
| 2 Overlay | 5–7 |
| 3 Formulir (33) | 8–10 |
| 4 Tabel dan daftar | 6–8 |
| 5 Layout dan navigasi | 5–6 |
| 6 Layar khusus | 5–7 |
| 7 Pengerasan | 5–6 |
| **Jumlah** | **42–55 hari kerja (±9–11 minggu)** |

Dasar angka: ±35 komponen, 31 modal, 33 formulir, 60 halaman, dengan pembungkus yang menjaga titik panggil tetap. Sinyal
pembanding dari proyek lain yang **tidak setara** (MUI→shadcn): epik Coder 5,5–8 minggu; tldraw memperkirakan 14–20
minggu-pengembang untuk pindah Radix→Base UI di basis kode besar. Harness uji (Tahap 0) menyumbang waktu tetapi
mencegah regresi senyap. Kerja berbantuan AI dapat memangkas sebagian besar penulisan ulang mekanis, tidak memangkas
tinjauan visual dan uji perangkat nyata.

## 6. Pengujian dan pengukuran

- **Visual:** Playwright `toHaveScreenshot` pada halaman emas × {1280, 375} × keadaan; potret dasar di-commit; selisih
  ditinjau manusia pada setiap tahap (boleh berubah hanya bila disengaja dan tercatat).
- **Aksesibilitas:** `@axe-core/playwright` (tag wcag2a/2aa/21a/21aa/22aa) setelah membuka dialog/menu; alat otomatis
  hanya menangkap sebagian masalah, jadi uji papan ketik dan pembaca layar manual tetap ada.
- **Performa:** INP p75 ≤ 200 ms [R], anggaran JS per rute (garis dasar diukur di Tahap 0; angka 170 KB dari web.dev
  usang dan hanya titik awal), komponen berat (Calendar, Command, Table dengan TanStack) di-lazy per rute.
- **Perangkat nyata:** iOS Safari dan Android Chrome kelas menengah (dialog, input, scroll lock, tab bawah).
- **Keberhasilan:** nol pelanggaran axe serius/kritis; WCAG 2.2 AA pada rute emas; tidak ada regresi fungsi (tes
  isolasi dan tes Go tetap hijau); ukuran bundel dan INP tidak lebih buruk dari garis dasar; jumlah komponen buatan
  sendiri yang menyamai shadcn = 0; waktu menyelesaikan tugas dan tingkat galat diukur dengan 5–8 pengguna WO dan pasangan
  (hipotesis nada tunggakan, D12, diuji di sini).

## 7. Risiko

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Pemeliharaan Radix melambat (bila D1 = Radix) | bug iOS/ARIA menumpuk, migrasi ulang mahal nanti | pilih Base UI (rekomendasi), atau terima dan simpan skill migrasi per komponen shadcn sebagai jalan keluar |
| Base UI lebih muda, jawaban komunitas lebih sedikit (bila D1 = Base UI) | waktu debug | prototipe Tahap 0; `isolation: isolate`; backdrop `position: absolute` di iOS 26 |
| Kode shadcn disalin ke repo | pembaruan manual | kunci versi CLI; `add --diff` per komponen; jangan `--overwrite` tanpa komit |
| Penimpaan tema tenant tidak sampai ke konten portal | dialog/menu tak berwarna merek | variabel di `:root`; diuji di prototipe (b) |
| Regresi senyap pada 41 pemanggil Modal / 69 Select | fungsi rusak tanpa terlihat | pembungkus berprops lama + harness visual + uji perilaku dialog |
| `init` menimpa `theme.css` atau gaya global | tampilan berubah luas | jalankan di cabang, tinjau diff, simpan token kita sebagai sumber |
| Bundel membengkak | LCP/INP buruk di HP Android | anggaran per rute di CI, lazy untuk komponen berat |
| Kontras batas input (1,34:1) dan teks sekunder (4,34:1) | kegagalan WCAG yang sudah ada | diperbaiki di Tahap 0 |
| Perubahan format rupiah terlihat pengguna | kebingungan, tes yang memuat string | keputusan D5, satu pemformat, ubah sekaligus |
| Satu pengembang, 9–11 minggu | konteks hilang di tengah | tahap bisa berhenti kapan saja dengan aplikasi utuh; daftar periksa di komit |

## 8. Keputusan yang dibutuhkan

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| **D1** | **Primitif dasar: Radix atau Base UI** (keduanya "shadcn/ui") | **Base UI** (default shadcn sejak Jul 2026, stabil 1.0 Des 2025, rilis bulanan, Combobox dan Toast bawaan; Radix melambat dengan 4 rilis Jun–Jul 2026 dan tak ada sesudah 20 Jul). **Radix** layak bila Anda menimbang jumlah panduan dan jawaban komunitas lebih penting; jalan keluarnya ada (`init -b radix`, skill migrasi per komponen). Pilih sekali; mencampur menggandakan permukaan debug | ✅ **Diputuskan 8 Nov 2026: Base UI** (pemilik produk; sebelumnya menyebut Radix, memilih Base UI setelah melihat bukti). `shadcn init` memakai basis Base UI; Radix tetap jalan keluar lewat `init -b radix` bila prototipe Tahap 0 gagal |
| D2 | Strategi: pembungkus berprops lama, migrasi pemanggil bertahap | Ya | Usulan |
| D3 | Formulir: `Field` + Zod `safeParse`; react-hook-form hanya bila perlu | Ya | Usulan |
| D4 | Toast: Sonner di balik `toast.*` (36 titik tidak berubah) atau Toast bawaan Base UI | Sonner; cek `add sonner` di Tahap 0 (status rekomendasi shadcn untuk Sonner belum terverifikasi) | Usulan |
| D5 | Format rupiah: "Rp 1.000.000" (sekarang) atau "Rp1.000.000" (sumber PUEBI sekunder, belum diverifikasi ke teks resmi) | "Rp1.000.000" setelah teks resmi Badan Bahasa dicek; satu pemformat | Terbuka |
| D6 | Mode gelap | Tunda; token tetap semantik | Usulan |
| D7 | Tanggal: pemilih native; `Calendar` shadcn hanya bila perlu rentang | Ya | Usulan |
| D8 | Fraunces untuk judul Beranda Portal | Ya, hanya di situ (sumber lemah, hipotesis) | Terbuka |
| D9 | Palet perintah di console | Opsional di Tahap 6 | Terbuka |
| D10 | Storybook | Tidak; potret Playwright sebagai basis visual | Usulan |
| D11 | Grafik: tetap buatan sendiri (bar SVG), tanpa Recharts | Ya | Usulan |
| D12 | Nada tunggakan untuk klien: nominal, tanggal asli dan aksi ("Bayar"/"Hubungi PIC"), tanpa kata menyalahkan | Sudah berlaku sebagian di Fase 5 (merah hanya di kartu "Yang perlu Anda lakukan", lencana klien tidak pernah merah); diuji di Tahap 7 dengan 5–8 pasangan | Hipotesis (tak ada sumber riset, hanya penilaian) |

## 9. Hal yang belum bisa diverifikasi

Dari riset: teks resmi PUEBI untuk rupiah, status rekomendasi Sonner vs Toast di shadcn, ukuran gzip tiap komponen
shadcn, perilaku portal terhadap variabel tenant (diuji di prototipe), bukti adopsi palet perintah, bentuk timeline
terbaik (rel vs Gantt, hanya penalaran), nada pesan tunggakan. Sumber lengkap dengan tingkat verifikasi ada di
`knowledge/DESIGN.md` §9.
