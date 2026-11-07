# DESIGN.md — Brief desain ElProof (revamp ke shadcn/ui)

> Status: **brief untuk revamp, disusun 8 Nov 2026** dari riset internet (sumber di §9, diakses 8 Nov 2026) dan
> inventaris kode. Belum ada kode yang diubah. Rencana kerjanya ada di
> `docs/plan/revamp-ui-shadcn/PLAN.md`. Alur dan susunan menu per peran tetap mengikuti
> `docs/plan/registrasi-wo-self-service/FLOW-IDEAL.md`; dokumen ini mengatur **rupa, komponen, dan perilaku UI**.
>
> Kejujuran bukti: tanda **[V]** berarti saya baca sendiri halaman sumbernya; **[R]** berarti dilaporkan agen riset dari
> halaman yang ia baca (ringkasan alat fetch, bukan HTML mentah); **[S]** sumber sekunder atau opini; **[?]** tidak
> berhasil diverifikasi. Keputusan yang bertumpu pada [S] atau [?] ditandai sebagai hipotesis.

## 1. Kasus

- **Subjek:** ElProof, aplikasi wedding organizer (WO) Indonesia: banyak proyek, vendor, termin pembayaran,
  timeline, kendala, dokumen bukti, dan peran staf. Janji merek: **transparansi dan bukti** ke klien.
- **Audiens dan perangkat:** (a) staf WO di laptop/tablet, data padat; (b) pengantin dan keluarga di HP Android
  kelas menengah, sinyal tidak selalu baik, berkomunikasi lewat WhatsApp; (c) tim ElProof di laptop (Platform Console).
- **Tugas utama layar:** WO: "apa yang perlu saya tangani sekarang?" Klien: "berapa yang harus saya bayar,
  kapan, dan sudah sampai mana persiapannya?" Tim ElProof: "bisnis sehat, siapa perlu ditindaklanjuti?"
- **Sistem yang sudah ada dan wajib dipakai (bukan diganti):** token `--color-*` dan `--brand-navy-*` yang ditimpa
  saat runtime oleh 21 preset warna tenant (kontras sudah diverifikasi per preset di `src/theme/brandPresets.ts`),
  token kepadatan `data-density` (console compact, portal comfortable), Inter untuk UI dan Fraunces hanya untuk
  pemasaran, label status terpusat (`shared/lib/labels.ts`, FLOW-IDEAL §4), `ProofSeal` (stempel bukti) sebagai tanda
  merek, `@svar-ui/react-gantt` untuk Gantt, `lucide-react` untuk ikon.

## 2. Temuan → keputusan

| # | Temuan (sumber) | Keputusan desain | Kekuatan bukti |
|---|---|---|---|
| T1 | Produk WO sejenis membuka beranda dengan antrean "perlu tindakan" untuk WO maupun klien: Gatera "Perlu Tindakan" [1], Planning Pod portal beranda "alerts them to items that require attention" [5], Aisle Planner dashboard tugas dan tenggat [4]. NN/g: dashboard melayani satu tujuan, satu layar, dibaca cepat, hindari pie/gauge/warna sebagai besaran [8][23] | Beranda WO dan Beranda Portal tetap berbasis tindakan (sudah dibangun di Fase 4–5). Grafik hanya batang/garis. Kartu KPI menaut ke daftar terfilter | [R] sumber produk; [R] NN/g |
| T2 | Klien hanya melihat item yang ditugaskan, biaya vendor dan margin WO tidak tampil [1][4] | Pertahankan pemisahan data klien di API (FLOW-IDEAL §3.3); tambah catatan "yang Anda lihat" di Portal | [R] |
| T3 | Studi NN/g 179 peserta: navigasi tersembunyi dipakai 57% di mobile versus 86% untuk kombinasi; tab bar cocok untuk 4–5 pilihan [20][21] | Portal: tab bawah 4–5 item berlabel (sudah: Beranda, Jadwal, Pembayaran, Vendor, Lainnya); tidak ada fitur yang hanya bisa dicapai lewat hamburger | [R] |
| T4 | WCAG 2.2 AA: target minimal 24×24 px (2.5.8); Android menyarankan 48dp; fokus tidak boleh tertutup elemen sticky (2.4.11); kontras komponen UI 3:1 (1.4.11) [17][18][25][26] | Target sentuh Portal ≥ 44 px dengan jarak 8 px; console ≥ 32 px tinggi kontrol, ikon aksi baris ≥ 24 px; indikator fokus 3:1; header tabel sticky tidak boleh menutup fokus | [R] W3C primer |
| T5 | Status tidak boleh hanya warna; pasangkan warna + bentuk/ikon + teks; batasi sekitar 5–6 status terlihat [10][23] | Lencana status selalu berteks, berikon bila berguna; uji tampilan skala abu-abu | [R] |
| T6 | Tabel operasional: angka rata kanan, tabular figures; dua kepadatan (Carbon 40 px default, Primer condensed/normal); header sortable = `th` + `button` + `aria-sort`; satu kolom urut default [1][2][4][27] | Tabel console: baris 40 px (default) dan 32 px (padat), angka/rupiah rata kanan dengan `tabular-nums`, header sortable sesuai APG, selalu ada urutan default | [R] |
| T7 | Empat keadaan daftar: memuat (skeleton, jangan tulis "tidak ada data" saat memuat), galat, kosong-belum-ada-data (dengan CTA), kosong-karena-filter (teks biasa + reset) [4][7][8] | Satu pola untuk semua daftar, urutan render: memuat → galat → kosong → isi | [R] |
| T8 | Validasi formulir diperdebatkan: GOV.UK validasi saat kirim + ringkasan galat berfokus [9]; NN/g dan Baymard: inline setelah field selesai, bukan saat mengetik [10][11]; Adam Silver: jangan menonaktifkan tombol kirim [11] | **Hibrida:** validasi saat kirim, sesudah percobaan pertama validasi ulang per field saat blur; ringkasan galat bila ≥ 3 galat; tombol kirim tidak pernah dinonaktifkan karena validitas; input tidak pernah dihapus; galat menyebut cara memperbaiki | [R]; sumber saling bertentangan, ini hipotesis |
| T9 | Modal untuk peringatan kritis dan formulir pendek; halaman penuh untuk alur banyak langkah dan keputusan kompleks [13] | Buat/ubah dengan ≤ 6 field: Dialog/Sheet; alur panjang (buat proyek 2 langkah): halaman penuh. Tindakan merusak: AlertDialog, fokus ke tombol paling tidak merusak, ketik nama untuk yang tak terpulihkan [21][22][28] | [R] |
| T10 | Banner untuk status yang menuntut tindakan, toast hanya untuk konfirmasi pasif, galat tidak lewat toast [17] | Status langganan dan mode baca-saja: banner permanen; toast hanya sukses/info; galat di dalam dialog atau di halaman | [R] |
| T11 | Mode "lihat sebagai": JWT terpisah, tanda baca-saja, kedaluwarsa pendek, bingkai permanen, keluar sekali klik, log mulai sesi; tolak mutasi di server [19][20] | Sudah dibangun (Fase 6); revamp hanya memindahkannya ke komponen Alert/banner baru tanpa mengubah perilaku | [R]/[S] |
| T12 | Aksi yang tak tersedia: disabled + alasan bila pengguna bisa mendapatkan izin, sembunyi bila tak pernah relevan; tak ada sumber primer NN/g [16] | Pola ini dipakai sebagai default; tooltip/teks menyebut peran yang diperlukan | [S], hipotesis |
| T13 | Gaya "Liquid Glass"/translusen: NN/g mencatat keterbacaan turun di atas latar sibuk, tab bar sesak [27]. Mode gelap: sepertiga pengguna gelap, sepertiga terang, sepertiga campuran; ikuti OS, bukan prioritas; bisa merusak QR [28]. Bento grid hanya bila ukuran blok bermakna [29] | **Tidak ada kaca/translusen** pada permukaan teks atau jumlah uang. **Mode gelap ditunda** (token tetap semantik agar mungkin nanti). Kartu tidak seragam: tagihan berikutnya lebih besar dari yang lain | [R] NN/g; [S] bento |
| T14 | Android 78,68% dari mobile Indonesia (Okt 2026), penjualan terbanyak kelas entry/mid [13][14]; median unduh seluler ±30 Mbps tetapi bervariasi [15]; anggaran web.dev: JS jalur kritis < 170 KB dan TTI < 5 dtk di 3G lambat pada Moto G4 (usang) [22]; INP bagus ≤ 200 ms p75 [35] | Anggaran performa sebagai gerbang CI (mulai dari angka ini lalu ukur data lapangan sendiri); route-level lazy; komponen berat (Calendar, Chart, Command) lazy | [R]; angka 170 KB usang |
| T15 | Rekening pembayaran dan bukti transfer dicek manual di WO Indonesia; Gatera/Inframe: akses tanpa akun, pengingat via email, WhatsApp berbayar tinggi [2][3] | Catatan pembayaran = jejak bukti (tanggal, metode, foto bukti, siapa memverifikasi); tombol "Bagikan via WhatsApp"; nomor dinormalkan ke 62… sebelum tautan `wa.me` [24] | [R]; belum ada produk lokal dengan portal HP + bukti: **peluang**, bukan fakta |
| T16 | Baymard: pelacakan pesanan butuh tanggal, tahap, dan gagal di pergantian kanal; NN/g: diam menimbulkan cemas, indikator kemajuan harus mencerminkan keadaan nyata [30][31] | Timeline Portal = rel vertikal dengan tanggal dan penanggung jawab; tidak ada progres palsu; bukti terbuka di tempat tanpa lompat login | [R] |
| T17 | Perintah palet (cmd+K): bukti adopsi lemah, GitHub sempat menghapusnya lalu memulihkannya [14][15] | Opsional, tahap akhir, hanya percepatan untuk pengguna mahir di console, tidak pernah satu-satunya jalan | [R]/[S] |
| T18 | Aturan penulisan rupiah: sumber PUEBI sekunder sepakat "Rp" tanpa titik dan umumnya **tanpa spasi** ("Rp100.000"); satu sumber sebaliknya; teks resmi Badan Bahasa tidak dibuka [25][26] | Satu pemformat uang di seluruh aplikasi. **Keputusan produk D5**: saat ini aplikasi menulis "Rp 1.000.000" (spasi); rekomendasi pindah ke "Rp1.000.000" setelah teks resmi dicek | [S], konflik |

## 3. Elemen khas (satu, bersumber dari subjek)

**Jejak bukti.** Nama produknya "proof", dan merek sudah punya stempel `ProofSeal` (segel lilin dari token
`--color-warning`, hanya dipakai di pemasaran). Elemen khas revamp: **setiap angka dan status yang penting punya
bukti yang bisa dibuka di tempat.** Pembayaran "Lunas" menampilkan stempel kecil berisi tanggal diverifikasi, siapa
yang memverifikasi, dan satu ketukan membuka foto bukti transfer; tonggak "Selesai" menampilkan foto/dokumennya.
Ini yang membedakan ElProof dari portal WO lain yang hanya menampilkan status (T15, T16) dan memberi klien alasan
percaya. Semua yang lain dibuat tenang: permukaan putih/slate buram, satu warna merek tenant, satu keluarga huruf.

Aturan pemakaian `ProofStamp` (turunan `ProofSeal`):
- Hanya muncul di atas sesuatu yang **benar-benar terverifikasi** (pembayaran dengan bukti, tonggak dengan
  dokumen). Tidak pernah hiasan.
- Kecil dan statis di dalam aplikasi (24–32 px); animasi "cap ditekan" hanya satu kali di momen bermakna (pembayaran
  baru terverifikasi), dan tidak jalan bila `prefers-reduced-motion`.
- Bukan hanya warna: ada teks ("Terverifikasi 7 Nov") dan ikon.

## 4. Token

Pemetaan dari token sekarang ke token semantik shadcn (nilai tetap dari `brandPresets`, tidak ada pembuat skala
otomatis karena warna tenant hanya 21 preset yang sudah diverifikasi kontrasnya).

| Token shadcn | Sumber sekarang | Catatan |
|---|---|---|
| `--background` | `--color-background` #f8fafc | |
| `--foreground` | `--color-text-primary` #0f172a | |
| `--card`, `--popover` | `--color-surface` #fff | buram, tanpa kaca |
| `--muted` / `--muted-foreground` | `--color-surface-muted` / `--color-text-secondary` | **Terhitung:** #64748b di atas #fff = 4,76:1 dan di atas #f8fafc = 4,55:1 (lolos), tetapi di atas `--color-surface-muted` #f1f5f9 = **4,34:1 (gagal AA untuk teks biasa)**. Gelapkan `muted-foreground` ke ±#5b6b82 atau jangan taruh teks sekunder di atas latar muted |
| `--primary` / `--primary-foreground` | `--brand-navy-900` (preset tenant) / putih | tiap preset sudah ≥ 4,5:1 terhadap putih |
| `--primary` hover | `--brand-navy-800` | |
| `--sidebar` | `--brand-navy-950` | |
| `--accent` | `--color-primary-soft` (preset `soft`) | latar sorot lembut |
| `--ring` | `--brand-navy-900` | indikator fokus ≥ 3:1 terhadap latar |
| `--border` / `--input` | `--color-border` #d6e0ea / `--color-border-light` #e2e8f0 | **Terhitung:** #d6e0ea di atas putih = **1,34:1**, jadi batas input gagal 1.4.11 (3:1). Pisahkan: `--border` tetap lembut untuk kartu/pemisah dekoratif, `--input` ±#7b8ba1 (3,47:1) untuk batas kontrol |
| `--destructive` | `--color-danger` | untuk tindakan merusak, bukan status klien |
| status sukses/peringatan/info | `--color-success/warning/info` (+ `-soft`) | tetap token proyek, ditambahkan di luar set shadcn |
| `--radius` | 10 px (md) | sama dengan sekarang |
| `--chart-1..5` | dari palet netral + warna merek | hanya bila Recharts dipakai (sekarang grafik buatan sendiri) |

Tipografi: **Inter** untuk semua UI, angka dengan `tabular-nums`. **Fraunces** tetap untuk pemasaran dan boleh
dipakai untuk judul besar Beranda Portal (satu-satunya tempat di app; sumber riset menyebut aksen serif cocok untuk
suasana pernikahan tetapi bukan untuk back office [S]); tidak pernah di tabel/formulir. Skala: 12 / 13,5 / 15 / 18 / 24 /
30 px, bobot 400/500/600/700, `line-height` 1,5 untuk bacaan. Panjang baris ≤ 75 karakter.

Kepadatan (rancangan sendiri, tidak ada sumber shadcn): `--control-h`, `--row-y`, `--gap`, `--text` per `data-density`.
Console `compact` (kontrol 36 px, baris tabel 40/32 px), portal `comfortable` (kontrol 44 px, target ≥ 44 px).

Gerak: durasi 150–200 ms untuk membuka/menutup; semua animasi dibungkus `motion-reduce:`; tidak ada gerak
dekoratif yang berjalan sendiri kecuali satu cap `ProofStamp` di momen bermakna.

Z-index (satu skala): konten 0, sticky 10, sidebar 30, dropdown/popover 40, dialog 50, toast 60, banner sesi
dukungan 70. Root aplikasi diberi `isolation: isolate` (Base UI) dan variabel tenant diletakkan di `:root`/`body`
agar konten yang dirender di portal ikut mewarisinya.

## 5. Tata letak

### WO Console, desktop (≥ 1024 px)

```
┌─────────┬──────────────────────────────────────────────────────────┐
│ Sidebar │ Judul halaman                       [aksi utama]         │
│ (logo   │ ── Banner langganan / pengumuman (bila ada) ───────────  │
│  WO)    │ ┌ Perlu tindakan ┐ ┌ Tagihan terlambat ┐ ┌ Acara 14 hr ┐ │
│ Beranda │ │ antrean         │ │ Rp 20.000.500     │ │ daftar      │ │
│ Project │ └─────────────────┘ └───────────────────┘ └─────────────┘ │
│ Client  │ Tabel: [cari] [filter ▾] [urut]            [Unduh CSV]    │
│ Vendor  │ Nama        Status      Tgl acara     Sisa tagihan (kanan)│
│ Venue   │ ...baris 40 px, angka rata kanan, aksi baris ⋯            │
│ Setelan │ 1–10 dari 148   ‹ 1 2 3 … ›                               │
└─────────┴──────────────────────────────────────────────────────────┘
```

### Portal Klien, HP (375 px)

```
┌──────────────────────────┐
│ Nama WO                   │
│ Selamat malam, Citra      │
│ H-266 menuju hari bahagia │
│ ┌──────────────────────┐ │
│ │ Tagihan berikutnya   │ │  ← kartu terbesar
│ │ Rp 5.000.000         │ │
│ │ jatuh tempo 20 Nov   │ │
│ │ [Bayar / Konfirmasi] │ │  ← WhatsApp ke PIC
│ └──────────────────────┘ │
│ Yang perlu Anda lakukan   │
│ Kabar terbaru ● ● ●       │
│  ✓ Terverifikasi 7 Nov    │  ← ProofStamp
├──────────────────────────┤
│ Beranda Jadwal Bayar Vendor Lainnya │  ← tab bawah ≥ 44 px
└──────────────────────────┘
```

Tabel di HP menjadi daftar kartu dengan kolom pertama terkunci (T2/T6, [11]). Platform Console memakai tata letak
console yang sama dengan sidebar bergrup (Pelanggan, Produk, Sistem; sudah ada).

## 6. Keadaan

Setiap daftar dan halaman memakai satu set: **memuat** (skeleton, bentuk sama dengan isi), **galat** (apa yang
terjadi + Coba lagi), **kosong belum ada data** (satu CTA), **kosong karena filter** (teks + Reset filter),
**data basi** (cap waktu bila relevan). Formulir: galat per field berikon + teks (tidak hanya merah), ringkasan galat
berfokus bila ≥ 3, `aria-invalid` + `aria-describedby`. Aksi tak tersedia: disabled + alasan atau tersembunyi (T12).
Mode baca-saja langganan: banner permanen + tombol tulis disembunyikan (perilaku sekarang dipertahankan).

## 7. Aksesibilitas (gerbang, bukan saran)

- WCAG 2.2 AA: kontras teks 4,5:1, komponen UI dan fokus 3:1, target ≥ 24 px (Portal ≥ 44 px), fokus tidak tertutup
  header sticky, otentikasi mengizinkan pengelola kata sandi, tidak ada isian berulang yang tak perlu [5][25][26].
- Pola APG untuk dialog (fokus masuk, perangkap fokus, Esc, kembali ke pemicu), alertdialog (fokus ke tombol paling
  tidak merusak), combobox, tabel sortable (`th` + `button` + `aria-sort`) [22][27][28][29]. Pustaka dasar (Radix atau
  Base UI) menyediakan ini; yang tetap tanggung jawab kita: label, judul dialog, nama aksesibel ikon.
- `prefers-reduced-motion`: semua animasi punya varian `motion-reduce` [30].
- Otomatis: `@axe-core/playwright` pada tiap rute setelah membuka dialog/menu; sumber menyebut cakupan alat otomatis
  terbatas (angka 57% dari studi vendor Deque) sehingga uji keyboard dan pembaca layar manual tetap wajib [31][45].
- Verifikasi di perangkat nyata: iOS Safari (dialog, input, scroll lock) dan Android Chrome kelas menengah.

## 8. Yang sengaja tidak diadopsi

Glass/translusen, mode gelap pertama, bento seragam, asisten AI, palet perintah sebagai navigasi utama, Gantt di
HP, grafik pie/donat/gauge, animasi masuk di tiap bagian, label huruf kapital semua, kartu seragam dengan bayangan sama.

## 9. Sumber (diakses 8 Nov 2026)

Nomor mengikuti laporan agen riset; tanda verifikasi per §awal. Sumber yang tidak terbuka ditulis jujur.

**Saya baca langsung [V]:** shadcn/ui changelog https://ui.shadcn.com/docs/changelog · "Base UI as the Default"
https://ui.shadcn.com/docs/changelog/2026-07-base-ui-default · Base UI releases https://base-ui.com/react/overview/releases ·
Radix Primitives releases https://www.radix-ui.com/primitives/docs/overview/releases · shadcn Toast
https://ui.shadcn.com/docs/changelog/2026-07-toast · shadcn Field https://ui.shadcn.com/docs/components/radix/field ·
pencarian aturan rupiah (hasil media, tidak resmi).

**Dilaporkan agen riset [R]** (setiap tautan dibuka agen lewat alat fetch yang meringkas):
[1] Gatera https://gatera.id/ · [2] Inframe Linea https://www.inframelinea.com/ · [3] Telkom University, piutang WO
https://repositori.telkomuniversity.ac.id/pustaka/files/174785/bab1/aplikasi-berbasis-web-untuk-pengelolaan-piutang-pada-pembayaran-customer-ke-wedding-organizer-studi-kasus-indah-wedding-organizer-kota-padang-.pdf ·
[4] Aisle Planner https://aisleplanner.com/products/client-portal/ · [5] Planning Pod https://planningpod.com/client-portals ·
[8][23] NN/g dashboards https://www.nngroup.com/articles/dashboards-preattentive/ · [10] Carbon status
https://carbondesignsystem.com/patterns/status-indicator-pattern/ · [11] NN/g mobile tables
https://www.nngroup.com/articles/mobile-tables/ · [13] StatCounter Indonesia https://gs.statcounter.com/os-market-share/mobile/indonesia ·
[14] Selular.id/Canalys https://selular.id/2025/08/pasar-smartphone-indonesia-kuartal-ii-2025-hp-murah-dan-menengah-jadi-andalan/ ·
[15] Ookla via TelecomLead https://telecomlead.com/?p=122145 · [17][18][25][26] W3C WCAG 2.2 Understanding
https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html, https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html,
Android https://support.google.com/accessibility/android/answer/7101858 · [20][21] NN/g hamburger
https://www.nngroup.com/articles/hamburger-menus/ dan navigasi mobile https://www.nngroup.com/articles/mobile-navigation-patterns/ ·
[22] web.dev budgets https://web.dev/articles/performance-budgets-101 · [24] WhatsApp click-to-chat
https://faq.whatsapp.com/5913398998672934 · [27] NN/g Liquid Glass https://www.nngroup.com/articles/liquid-glass/ ·
[28] NN/g dark mode https://www.nngroup.com/articles/dark-mode-users-issues/ · [30][31] Baymard order tracking
https://baymard.com/ecommerce-design-examples/63-order-tracking-page, NN/g progress indicators
https://www.nngroup.com/articles/progress-indicators/ · Primer DataTable https://primer.style/product/components/data-table/ ·
Carbon data table https://carbondesignsystem.com/components/data-table/usage/ · Polaris table
https://shopify.dev/docs/api/app-home/polaris-web-components/layout-and-structure/table · GOV.UK validation
https://design-system.service.gov.uk/patterns/validation/ · NN/g form errors https://www.nngroup.com/articles/errors-forms-design-guidelines/ ·
Baymard inline validation https://baymard.com/blog/inline-form-validation · NN/g modal vs nonmodal
https://www.nngroup.com/articles/modal-nonmodal-dialog/ · NN/g notifications
https://www.nngroup.com/articles/indicators-validations-notifications/ · Stripe Apps patterns
https://docs.stripe.com/stripe-apps/patterns · NN/g empty states https://www.nngroup.com/articles/empty-state-interface-design/ ·
APG: https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/, /alertdialog/, /combobox/,
/table/examples/sortable-table/ · MDN reduced motion
https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion · Deque
https://www.deque.com/blog/automated-testing-study-identifies-57-percent-of-digital-accessibility-issues ·
Playwright https://playwright.dev/docs/test-snapshots, https://playwright.dev/docs/accessibility-testing · web.dev INP
https://web.dev/articles/inp · Pigment impersonation https://engineering.pigment.com/2026/04/08/safe-user-impersonation/ ·
shadcn docs: https://ui.shadcn.com/docs/installation/vite, /docs/components-json, /docs/theming, /docs/tailwind-v4,
/docs/changelog/2026-03-cli-v4, /docs/changelog/2026-02-radix-ui · Radix accessibility
https://www.radix-ui.com/primitives/docs/overview/accessibility · tldraw Radix→Base UI
https://github.com/tldraw/tldraw/issues/7584 · MDN relative colors
https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_colors/Relative_colors · Tailwind theme
https://tailwindcss.com/docs/theme.

**Sekunder atau opini [S]:** beberapa blog tabel/UX, `dev.to` perbandingan shadcn/Radix/Base UI, line25 tren 2026,
saasframe cmd+K, Detik/Media Indonesia/Reku/IDXChannel (aturan rupiah), aggregator ukuran bundel.

**Tidak berhasil dibuka atau tidak ditemukan [?]:** teks resmi Badan Bahasa/PUEBI, dokumen Linear/Vercel/Material 3/
Atlassian/Chargebee/Paddle/Intercom/Zendesk/Auth0, halaman GOV.UK task list, Opensignal angka, sumber untuk bentuk
timeline (rel vs Gantt), nada pesan tunggakan, format +62, bukti adopsi palet perintah, ukuran gzip tiap komponen
shadcn, perilaku portal Radix terhadap variabel tenant (kesimpulan dari pewarisan CSS, **harus diuji di prototipe**).
