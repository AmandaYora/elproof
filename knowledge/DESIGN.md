# DESIGN.md — Brief desain ElProof (revamp ke shadcn/ui)

> Status: **brief revisi 2, 9 Nov 2026** (revisi 1: 8 Nov 2026). Disusun dari riset internet (sumber di §12, diakses
> 8–9 Nov 2026), inventaris kode, dan audit endpoint terhadap alur UI. Belum ada kode yang diubah. Rencana kerjanya di
> `docs/plan/revamp-ui-shadcn/PLAN.md`. Alur dan susunan menu per peran mengikuti
> `docs/plan/registrasi-wo-self-service/FLOW-IDEAL.md`; dokumen ini mengatur **rupa, komponen, bahasa, dan perilaku UI**.
>
> **Yang berubah di revisi 2:** (1) tujuan diperluas dari "ganti lapisan komponen" menjadi "mudah dipahami, mudah
> dipakai, rupa modern"; **endpoint boleh diubah** bila bentuk API membuat alur UI buruk (keputusan pemilik produk,
> 9 Nov 2026, daftar di PLAN.md §4); (2) arah rupa baru (§5); (3) token teks status dan netral hangat (§4), karena warna
> status lama gagal kontras; (4) pedoman bahasa dan glosarium (§7); (5) bantuan dan onboarding (§8); (6) sitasi memakai
> kunci bernama, karena nomor di revisi 1 bertabrakan (mis. [5], [13], [22], [27] menunjuk dua sumber berbeda).
>
> Kejujuran bukti: **[V]** saya baca sendiri halaman sumbernya; **[R]** dilaporkan agen riset dari halaman yang ia
> baca (ringkasan alat fetch); **[S]** sumber sekunder atau opini; **[?]** tidak berhasil diverifikasi. Keputusan yang
> bertumpu pada [S] atau [?] ditandai sebagai hipotesis. Angka kontras di dokumen ini dihitung dengan rumus luminansi
> relatif WCAG, bukan perkiraan.

## 1. Kasus

- **Subjek:** ElProof, aplikasi wedding organizer (WO) Indonesia: banyak proyek, vendor, termin pembayaran,
  jadwal, kendala, dokumen bukti, dan peran staf. Janji merek: **transparansi dan bukti** ke klien.
- **Audiens dan perangkat:** (a) staf WO di laptop/tablet, data padat; (b) pengantin dan keluarga di HP Android
  kelas menengah, sinyal tidak selalu baik, berkomunikasi lewat WhatsApp; (c) tim ElProof di laptop (Platform Console).
- **Tugas utama layar:** WO: "apa yang perlu saya tangani sekarang?" Klien: "berapa yang harus saya bayar,
  kapan, dan sudah sampai mana persiapannya?" Tim ElProof: "bisnis sehat, siapa perlu ditindaklanjuti?"
- **Ukuran "mudah dipakai"** (bukan perasaan): tingkat keberhasilan tugas, waktu tugas, jumlah salah langkah, dan skor
  kemudahan per tugas (SEQ 1–7), diukur **sebelum** revamp dan diulang dengan tugas yang sama (PLAN.md §7).
- **Sistem yang sudah ada dan tetap dipakai:** token `--color-*` dan `--brand-navy-*` yang ditimpa saat runtime oleh
  21 preset warna tenant (`src/theme/brandPresets.ts`), token kepadatan `data-density` (console compact, portal
  comfortable), label status terpusat (`shared/lib/labels.ts`, FLOW-IDEAL §4), `ProofSeal` sebagai tanda merek,
  `@svar-ui/react-gantt` untuk Gantt, `lucide-react` untuk ikon.

## 2. Temuan → keputusan

| # | Temuan (sumber) | Keputusan desain | Bukti |
|---|---|---|---|
| T1 | Produk WO sejenis membuka beranda dengan antrean "perlu tindakan": Gatera "Perlu Tindakan dan pemantauan semua acara" [gatera], Planning Pod "Attention Required" [planningpod-client], Aisle Planner tugas dan tenggat [aisle-portal]. Dashboard melayani satu tujuan, dibaca cepat, hindari pie/gauge [nng-dash] | Beranda WO dan Portal berbasis tindakan. Grafik hanya batang/garis. Kartu KPI menaut ke daftar terfilter | [V] gatera; [R] lainnya |
| T2 | Klien hanya melihat item yang ditugaskan; biaya vendor dan margin WO tidak tampil [aisle-portal] | Pemisahan data klien di API (FLOW-IDEAL §3.3); catatan "yang Anda lihat" di Portal | [R] |
| T3 | Navigasi tersembunyi dipakai jauh lebih jarang di HP; tab bar cocok untuk 4–5 pilihan [nng-hamburger][nng-mobilenav] | Portal: tab bawah 5 item berlabel; tidak ada fitur yang hanya bisa dicapai lewat hamburger | [R] |
| T4 | WCAG 2.2 AA: target ≥ 24×24 px (2.5.8), fokus tidak tertutup elemen sticky (2.4.11), kontras komponen UI 3:1 (1.4.11); Android menyarankan 48dp [wcag-258][wcag-1411][wcag-2411][android-touch] | Target Portal ≥ 44 px berjarak 8 px; console kontrol 36 px, ikon aksi baris ≥ 24 px; indikator fokus 3:1; header sticky tidak menutup fokus | [R] |
| T5 | Status tidak boleh hanya warna; warna + ikon + teks; batasi ±5–6 status terlihat [carbon-status] | Lencana status selalu berteks dan berikon; uji tampilan skala abu-abu | [R] |
| T6 | Tabel operasional: angka rata kanan, tabular figures; baris Carbon 24/32/40 (default)/48/64 px; toolbar ≤ 5 aksi, bilah aksi massal menggantikan toolbar saat ada baris terpilih, paginasi di bawah; header sortable = `th` + `button` + `aria-sort` [carbon-table][primer-table][apg-sort] | Baris 40 px (default) / 32 px (padat), rupiah rata kanan `tabular-nums`, header sortable APG, selalu ada urutan default, bilah aksi massal | [R] |
| T7 | Empat keadaan daftar; jangan tulis "tidak ada data" saat masih memuat; kosong menjelaskan fitur + satu aksi [nng-empty][stripe-patterns] | Urutan render: memuat → galat → kosong → isi (§6) | [R] |
| T8 | Validasi diperdebatkan: GOV.UK validasi saat kirim + ringkasan galat [govuk-validation]; NN/g dan Baymard inline setelah field selesai [nng-form-errors][baymard-inline]; jangan nonaktifkan tombol kirim [S] | **Hibrida:** validasi saat kirim, lalu per field saat blur; ringkasan galat bila ≥ 3; tombol kirim tidak dinonaktifkan karena validitas; input tidak dihapus; galat menyebut cara memperbaiki | [R], hipotesis |
| T9 | Modal untuk konfirmasi dan formulir pendek; halaman penuh untuk alur banyak langkah [nng-modal]. Konfirmasi hanya untuk aksi serius/tak terpulihkan, spesifik, tombol bertuliskan aksinya, bukan Ya/Tidak; ketik kata untuk risiko tinggi yang jarang [nng-confirm][apg-alertdialog] | Wadah per FLOW-IDEAL §0 (modal 1–3 field, drawer untuk edit sambil melihat konteks, halaman penuh untuk bertahap). AlertDialog hanya untuk aksi tak terpulihkan | [R] |
| T10 | Banner untuk status yang menuntut tindakan, toast untuk konfirmasi pasif, galat tidak lewat toast [nng-notif] | Banner permanen untuk langganan/baca-saja; toast hanya sukses/info (+ "Urungkan"); galat di dalam form/halaman | [R] |
| T11 | Mode "lihat sebagai": token terpisah, baca-saja, kedaluwarsa pendek, bingkai permanen, log sesi [pigment] | Sudah dibangun (Fase 6); revamp hanya memindahkan ke banner baru | [R]/[S] |
| T12 | Aksi tak tersedia: disabled + alasan bila izin bisa didapat, sembunyi bila tak pernah relevan [S] | Default; teks menyebut peran yang diperlukan | [S], hipotesis |
| T13 | Gaya kaca/translusen menurunkan keterbacaan [nng-glass]. Mode gelap: preferensi terbagi; ikuti OS; bisa merusak QR [nng-dark] | Tidak ada kaca pada teks atau jumlah uang. Mode gelap ditunda, token tetap semantik | [R] |
| T14 | Android ±79% mobile Indonesia [statcounter]; kelas entry/mid dominan [selular]; INP bagus ≤ 200 ms, LCP ≤ 2,5 dtk, CLS ≤ 0,1 pada p75 HP [webdev-vitals][webdev-inp] | Anggaran performa sebagai gerbang CI; rute lazy; komponen berat lazy; **satu panggilan per layar** di Portal (T29) | [R] |
| T15 | Bukti transfer dicek manual di WO Indonesia [telkomu]; akses dokumen tanpa akun [gatera]; WhatsApp click-to-chat `wa.me/62…?text=` tanpa `+`/0 [wa] | Catatan pembayaran = jejak bukti (tanggal, metode, foto, siapa memverifikasi); "Bagikan via WhatsApp"; nomor dinormalkan ke 62… | [V] wa, gatera; [R] |
| T16 | Pelacak status: keadaan terbaru paling atas dan menonjol, daftar vertikal di HP, bahasa awam, riwayat bertanggal, status sama di semua kanal, tautan notifikasi tanpa login ulang [nng-status-tracker][nng-progress]; tanpa progres palsu | Jadwal Portal = rel vertikal, terbaru di atas, tanggal + penanggung jawab; bukti terbuka di tempat | [R] |
| T17 | Palet perintah: bukti adopsi lemah [S]. Pintasan papan ketik hanya untuk aksi inti yang sering, opsional, ditampilkan di menu/tooltip [nng-accelerators] | ⌘K sebagai percepatan pencarian global console (sudah ada `GlobalSearch`), tidak pernah satu-satunya jalan | [R]/[S] |
| T18 | PUEBI (Permendikbud 50/2015): "Rp5.000.000,00", tanpa spasi dan tanpa titik setelah Rp [puebi]. `Intl.NumberFormat('id-ID')` menghasilkan "Rp 1.500.000" dengan spasi tak-putus | Satu pemformat uang: **"Rp1.500.000"** (buang spasi dari `Intl`), tanpa ",00" untuk rupiah bulat. D5 diputuskan oleh sumber primer | [R] primer |
| T19 | Rupa modern yang tetap terbaca: Linear (2026) meredupkan sidebar agar konten memimpin, pemisah lebih sedikit dan lembut, pindah dari abu-abu kebiruan ke **abu-abu hangat netral** [linear-refresh]; Vercel Geist: dasar monokrom, warna hanya untuk keadaan/aksi/data, tanpa bayangan dekoratif/gradien, tabular figures, baris prosa 60–68 karakter [geist]; Atlassian: kartu datar memakai garis, bayangan hanya untuk yang melayang (menu, dialog, kartu diseret) [atl-elevation] | Arah rupa §5: netral hangat, merek tenant hemat, garis untuk permukaan, bayangan hanya untuk overlay, sidebar netral | [R] |
| T20 | Skala huruf: Atlassian badan 14/20, kecil 12/16, besar 16/24, judul 16–32 [atl-type]; Primer rem + line-height tanpa satuan di grid 4 px, bobot terbatas [primer-type]; spasi Stripe 2/4/8/16/24/32/48 [stripe-style] | Console badan 14 px (naik dari 13,5), grid 4 px, bobot 400/500/600 | [R] |
| T21 | Gerak Carbon: 70/110/150/240/400/700 ms; easing "productive" `cubic-bezier(0.2,0,0.38,0.9)` untuk UI harian, "expressive" untuk momen jarang [carbon-motion] | 110–150 ms untuk hover/menu, 240 ms untuk dialog/drawer, productive; expressive hanya untuk `ProofStamp` | [R] |
| T22 | Waktu tanggap: 0,1 dtk terasa instan, 1 dtk alur tetap, 10 dtk butuh progres + batal [nng-response]. Skeleton untuk halaman penuh 2–10 dtk, tanpa indikator < 1 dtk, hindari skeleton yang hanya bingkai [nng-skeleton] | Indikator muncul setelah 400 ms; skeleton berbentuk isi; unggah bukti memakai bilah progres + batal | [R] |
| T23 | Lebih baik **urungkan** daripada konfirmasi untuk aksi rutin [nng-confirm]; React 19 `useOptimistic` menampilkan hasil seketika dan kembali otomatis bila gagal [react-optimistic] | Aksi rutin (arsipkan, ubah status, sembunyikan dari klien, hapus vendor dari proyek) optimistis + toast "Urungkan"; butuh soft delete/restore di API (PLAN.md E13) | [R] |
| T24 | Edit di tempat untuk data yang sering berubah, Enter simpan, Esc batal [atl-inline-edit]; aksi massal dengan pilih-semua + bilah kontekstual + urungkan [nng-bulk]; tampilan tersimpan di URL [polaris-filters]; progressive disclosure maks. 2 tingkat [nng-progressive] | Console: edit di tempat untuk status, PIC, tanggal; aksi massal di Vendor/Venue/Jadwal; filter + urutan di URL (bisa dibagikan, tombol kembali bekerja) | [R] |
| T25 | Setup guide Polaris: checklist yang bisa ditutup, ada progres, idealnya ≤ 5 langkah [polaris-setup]; tampilan kosong yang menjelaskan lebih efektif daripada tur [nng-empty] | `OnboardingChecklist` (sudah ada) ≤ 5 langkah, bisa ditutup; tidak ada tur popup; bantuan kontekstual (§8) | [R] |
| T26 | Pembayaran termin di produk sejenis: Aisle Planner menghitung sisa, pengingat otomatis, "jatuh tempo ≤ 30 hari" di dashboard [aisle-invoices]; Dubsado termin terkunci setelah lunas, terlambat ditandai [dubsado]; HoneyBook 4 pengingat (H-7, hari H, H+2) [honeybook-reminders] | Termin lunas terkunci (ubah lewat koreksi tercatat); kartu "jatuh tempo ≤ 30 hari"; pengingat otomatis masuk backlog fitur (bukan revamp) | [R] |
| T27 | UX luring: bahasa awam, jangan abu-abu (terbaca nonaktif), teks + ikon + toast, umumkan saat tersambung lagi [webdev-offline] | Banner "Koneksi terputus, perubahan belum terkirim" dan "Tersambung lagi" di Portal dan console | [R] |
| T28 | Tulisan UX: tombol diawali kata kerja, 1–4 kata, menyebut hasilnya; galat menyebut apa yang terjadi dan cara memperbaiki, tanpa "tidak valid" [grafana-ux] | Pedoman bahasa §7 | [R] |
| T29 | API untuk UX: endpoint "tampilan" per layar cocok bila klien berbeda kebutuhan (Portal HP) [ms-bff]; respons update = sumber daya terbaru + etag [aip-134]; galat field-level dengan penunjuk field [rfc9457]; idempotensi pembayaran [stripe-idem]; unggah lewat presigned URL [s3-presigned]; offset cocok untuk tabel "hal. X dari Y", kursor untuk feed/log [zalando] | Perubahan endpoint di PLAN.md §4 (E1–E14): satu panggilan per layar, mutasi mengembalikan ringkasan terbaru, galat per field dengan indeks baris, `Idempotency-Key`, `If-Match` | [R] primer |

## 3. Elemen khas (satu, bersumber dari subjek)

**Jejak bukti.** Nama produknya "proof", dan merek sudah punya stempel `ProofSeal`. Elemen khas revamp: **setiap
angka dan status yang penting punya bukti yang bisa dibuka di tempat.** Pembayaran "Lunas" menampilkan stempel kecil
berisi tanggal diverifikasi, siapa yang memverifikasi, dan satu ketukan membuka foto bukti transfer; tahapan "Selesai"
menampilkan foto/dokumennya. Ini yang membedakan ElProof dari portal WO lain yang hanya menampilkan status (T15, T16).
Semua yang lain dibuat tenang (§5).

Aturan `ProofStamp` (turunan `ProofSeal`):
- Hanya muncul di atas sesuatu yang **benar-benar terverifikasi**. Tidak pernah hiasan. Butuh data yang **belum ada di
  API** (siapa dan kapan memverifikasi, daftar bukti pada respons): PLAN.md E4.
- Kecil dan statis di aplikasi (24–32 px); animasi "cap ditekan" (easing expressive, ±400 ms) hanya satu kali saat
  pembayaran baru terverifikasi, dan tidak jalan bila `prefers-reduced-motion`.
- Bukan hanya warna: ikon + teks ("Terverifikasi 7 Nov oleh Rina"). Teks yang sama tampil di Portal, console, dan
  pesan WhatsApp (T16: status sama di semua kanal).

## 4. Token

### 4.1 Netral (revisi 2: dari slate dingin ke netral hangat)

FLOW-IDEAL §0 dan arah Linear 2026 meminta netral hangat, tetapi `theme.css` memakai slate (abu kebiruan). Usulan
memakai keluarga stone Tailwind; **bukan krem** (latar tetap hampir putih) supaya warna merek tenant tetap jernih.
Keputusan D13.

| Peran | Sekarang | Usulan | Kontras (dihitung) |
|---|---|---|---|
| `--background` | #f8fafc | #fafaf9 | — |
| `--card`, `--popover` | #ffffff | #ffffff | — |
| `--muted` (latar) | #f1f5f9 | #f5f5f4 | — |
| `--foreground` | #0f172a | #1c1917 | 16,7:1 di atas #fafaf9 |
| `--muted-foreground` | #64748b (4,34:1 di atas muted, **gagal**) | #57534e | 7,6:1 di atas putih; 7,0:1 di atas muted |
| teks pembantu/placeholder | — | #78716c | 4,80:1 di atas putih (**hanya** di atas putih; 4,40:1 di atas muted) |
| `--border` (dekoratif) | #d6e0ea | #e7e5e4 | dekoratif, tak wajib 3:1 |
| `--input` (batas kontrol) | #d6e0ea (1,34:1, **gagal 1.4.11**) | #8a847e | 3,70:1 di atas putih |

Bila D13 ditolak, perbaikan minimum di slate: `--muted-foreground` ±#5b6b82 (4,95:1 di atas #f1f5f9) dan `--input`
±#7b8ba1 (3,47:1).

### 4.2 Status (revisi 2: temuan baru, gagal kontras)

Teks status sekarang memakai warna yang sama dengan titik/ikon. Dihitung: di atas latar lencana, success 3,00:1,
warning 2,86:1, danger 3,95:1, info 4,24:1; di atas putih, success 3,30:1 dan warning 3,19:1. Semua di bawah 4,5:1.
Dipakai ±173 kali di 86 berkas (`text-success|warning|danger|info`) termasuk `Badge` (48 berkas). Pisahkan peran:

| Token | Nilai | Dipakai untuk | Kontras |
|---|---|---|---|
| `--color-success` / `-warning` / `-danger` / `-info` | #16a34a / #d97706 / #dc2626 / #2563eb (tetap) | ikon, titik, garis, bilah progres (non-teks, butuh 3:1) | 3,30 / 3,19 / 4,83 / 5,17 di atas putih |
| `--color-success-text` / `-warning-text` / `-danger-text` / `-info-text` (baru) | #15803d / #b45309 / #b91c1c / #1d4ed8 | semua teks berwarna status | 5,02 / 5,02 / 6,47 / 6,70 di atas putih |
| `--color-*-soft` (latar lencana, diganti) | #f0fdf4 / #fffbeb / #fef2f2 / #eff6ff | latar lencana dan banner | teks `-text` di atasnya 4,79 / 4,84 / 5,91 / 6,16 |

Tabrakan merek vs status: preset tenant **gold** memakai #b45309/#d97706 (sama dengan warning) dan **emerald** dekat
dengan success. Karena itu status **selalu** ikon + teks (T5), dan warna merek tidak pernah dipakai untuk status.

### 4.3 Pemetaan semantik shadcn

| Token shadcn | Sumber | Catatan |
|---|---|---|
| `--primary` / `--primary-foreground` | `--brand-navy-900` (preset tenant) / putih | tiap preset ≥ 4,5:1 terhadap putih |
| `--primary` hover | `--brand-navy-800` | |
| `--ring` | `--brand-navy-900` | fokus ≥ 3:1 |
| `--accent` | `--color-primary-soft` (preset `soft`) | latar sorot lembut, item navigasi aktif |
| `--sidebar` | **`--background`** (revisi 2; sebelumnya `--brand-navy-950`) | sidebar netral, merek hanya di logo, indikator item aktif, tombol utama (§5, D13) |
| `--destructive` | `--color-danger` (teks: `--color-danger-text`) | tindakan merusak, bukan status klien |
| `--radius` | 10 px | kontrol 8 px, kartu 12 px, lencana penuh (pil) — radius mengikuti hierarki, bukan satu nilai |
| `--chart-1..5` | tidak dipakai | grafik buatan sendiri (D11) |

### 4.4 Tipografi, kepadatan, gerak, lapisan

- **Huruf UI:** Inter (sekarang) dengan `tabular-nums` untuk angka. Opsi D14: **Plus Jakarta Sans** (Tokotype,
  dirancang untuk identitas kota Jakarta, OFL) memberi karakter Indonesia yang khas tanpa mengorbankan keterbacaan;
  **syarat**: punya fitur `tnum` dan ukuran berkas variabel wajar; dicek di prototipe Tahap 0 [?: `tnum` belum
  terverifikasi]. Bila tidak lolos, tetap Inter.
- **Fraunces** hanya untuk judul besar Beranda Portal (nama pasangan, hitung mundur; D8); situs publik memakai satu keluarga, Plus Jakarta Sans (ADR-0038). Tidak pernah
  di tabel/formulir.
- **Skala:** 12 / 14 / 16 / 20 / 24 / 30 px; line-height 16 / 20 / 24 / 28 / 32 / 38; bobot 400/500/600 (700 hanya
  angka besar). Badan console 14 px, Portal 16 px. Panjang baris prosa ≤ 70 karakter. Label tidak huruf kapital semua.
- **Kepadatan** per `data-density`: console `compact` (kontrol 36 px, baris 40/32 px, celah 16 px), Portal
  `comfortable` (kontrol 44 px, celah 24 px). Grid 4 px.
- **Gerak:** 110–150 ms hover/menu, 240 ms dialog/drawer, easing productive; satu momen expressive (`ProofStamp`).
  Semua dibungkus `motion-reduce:`. Tidak ada animasi masuk per bagian (pengecualian terlingkup untuk Beranda dan Tentang
  Kami situs publik: ADR-0038, `DESIGN-PUBLIC.md`).
- **Lapisan (z-index):** konten 0, sticky 10, sidebar 30, dropdown/popover 40, dialog 50, toast 60, banner sesi
  dukungan 70. Root `isolation: isolate` (Base UI); variabel tenant di `:root`/`body` agar konten portal mewarisinya.
- **Elevasi:** permukaan datar = garis `--border`, tanpa bayangan. Bayangan hanya untuk popover/menu (kecil) dan
  dialog/drawer (sedang).

## 5. Rupa (look & feel)

Arah: **tenang, padat, dan bisa dipercaya**; satu-satunya hal yang mencolok adalah bukti (§3). Prinsip:

1. **Konten memimpin, kerangka mundur.** Sidebar netral (latar sama dengan halaman, ikon 16 px abu, item aktif
   memakai latar `--accent` + teks `--primary`), bukan blok warna merek penuh. Ini perubahan yang terlihat oleh tenant
   (D13): warna merek tetap hadir di logo, tombol utama, cincin fokus, dan item aktif.
2. **Warna hanya bila bermakna.** Abu hangat untuk semua permukaan; warna merek untuk aksi utama; warna status untuk
   keadaan (selalu dengan ikon + teks); tanpa gradien dekoratif, kaca, atau bayangan di kartu datar.
3. **Struktur dari jarak dan garis tipis**, bukan kotak di dalam kotak. Halaman detail memakai bagian berjudul dengan
   pemisah, bukan tumpukan kartu seragam. Kartu hanya untuk hal yang berdiri sendiri (tagihan berikutnya, KPI).
4. **Angka adalah isi utama** untuk WO dan klien: rupiah rata kanan, `tabular-nums`, format PUEBI, nominal penting
   memakai ukuran 20–30 px dengan label kecil di atasnya, bukan sebaliknya.
5. **Portal berbeda suasana dengan console, satu sistem:** Portal lebih lega, judul Fraunces untuk nama pasangan dan
   hitung mundur, foto header dari WO bila ada; console padat dan netral.
6. **Hindari ciri template:** label huruf kapital di atas tiap judul, kartu seragam dengan bayangan sama, angka besar +
   gradien sebagai pahlawan, ikon di setiap judul, panah "→" di setiap tautan.

### 5.1 WO Console, desktop (≥ 1024 px)

```
┌──────────────┬─────────────────────────────────────────────────────────────┐
│ [logo WO]    │ Beranda                         [⌘K Cari]  [🔔]  [Akun]     │
│              │ ─ Banner langganan / pengumuman (bila ada) ───────────────  │
│ ▣ Beranda    │ Selamat pagi, Rina · Senin, 9 Nov                           │
│ ▢ Proyek     │                                                             │
│ ▢ Klien      │ Tagihan klien terlambat        Perlu tindakan (7)           │
│ ▢ Direktori  │ Rp20.000.500                   • Termin DP Citra & Bima…    │
│ ▢ Keuangan   │ 3 proyek · lihat daftar        • Kendala katering terbuka…  │
│              │                                                             │
│ Pengaturan   │ Acara 14 hari ke depan                                      │
│              │ Proyek          Tanggal     H-     Sisa tagihan      PIC    │
│ (sidebar     │ Citra & Bima    20 Nov      H-11   Rp5.000.000      Rina   │
│  netral)     │ ...baris 40 px, rupiah rata kanan, aksi baris ⋯              │
└──────────────┴─────────────────────────────────────────────────────────────┘
```

Daftar (Proyek, Klien, Vendor): toolbar [cari] [filter ▾] [urut ▾] [tampilan tersimpan] ... [Unduh CSV]; filter dan
urutan di URL; pilih baris → bilah aksi massal menggantikan toolbar; paginasi bawah "1–10 dari 148".

### 5.2 Portal Klien, HP (375 px)

```
┌──────────────────────────┐
│ Nama WO                   │
│ Citra & Bima        (serif)│
│ H-266 menuju hari bahagia │
│ ┌──────────────────────┐ │
│ │ Tagihan berikutnya   │ │  ← kartu terbesar, satu-satunya yang menonjol
│ │ Rp5.000.000          │ │
│ │ jatuh tempo 20 Nov   │ │
│ │ Transfer ke BCA …    │ │
│ │ [Konfirmasi via WA]  │ │
│ └──────────────────────┘ │
│ Yang perlu Anda lakukan   │
│ Kabar terbaru (terbaru di atas)
│  ✓ DP lunas · Terverifikasi 7 Nov oleh Rina  ← ProofStamp, ketuk = foto bukti
├──────────────────────────┤
│ Beranda Jadwal Bayar Vendor Lainnya │  ← tab bawah ≥ 44 px
└──────────────────────────┘
```

Tabel di HP menjadi daftar kartu dengan kolom pertama terkunci. Platform Console memakai tata letak console dengan
sidebar bergrup (Pelanggan, Produk, Sistem; sudah ada).

## 6. Keadaan dan umpan balik

- **Daftar/halaman:** memuat (skeleton berbentuk isi, muncul setelah 400 ms), galat (apa yang terjadi + Coba lagi,
  bukan logout), kosong belum ada data (penjelasan + satu CTA, §8), kosong karena filter (teks + Reset filter), data
  basi (cap waktu bila relevan), luring (banner T27).
- **Formulir:** galat per field berikon + teks, ringkasan galat berfokus bila ≥ 3, `aria-invalid` +
  `aria-describedby`, galat dari server dipetakan ke field (butuh E10). Tombol simpan menampilkan "Menyimpan…" saat
  proses (bukan dinonaktifkan karena validitas).
- **Aksi rutin:** optimistis + toast "Urungkan" ±6 detik (T23). **Aksi tak terpulihkan** (hapus pembayaran
  terverifikasi, suspend tenant): AlertDialog spesifik, fokus ke "Batal", ketik nama untuk tingkat tenant.
- **Aksi tak tersedia:** disabled + alasan, atau tersembunyi (T12). Mode baca-saja langganan: banner permanen + tombol
  tulis disembunyikan.

## 7. Bahasa dan microcopy (baru di revisi 2)

FLOW-IDEAL §0 menetapkan Bahasa Indonesia penuh dengan "Anda", tetapi UI sekarang campur: menu WO "Project", "Client",
"Venue", "Timeline Default", "Pengguna"; hitungan kasar teks TSX: "Project" ±53 vs "Proyek" ±4, "Client" ±63 vs "Klien"
±8. Pencarian global menulis sublabel "Client".

**Glosarium UI** (satu istilah per konsep; enum DB tetap Inggris; label di `shared/lib/labels.ts`):

| Konsep | Dipakai | Bukan |
|---|---|---|
| project | **Proyek** | Project |
| client (orang/akun) | **Klien** | Client |
| staff tenant | **Tim** (menu), anggota tim | Pengguna, Staff |
| payment installment | **Termin** | Term, cicilan |
| invoice ke klien | **Tagihan** | Invoice |
| evidence / bukti | **Bukti** (untuk pembayaran/tahapan); **Dokumen** (berkas umum) | Evidence |
| issue | **Kendala** | Issue |
| timeline / milestone | **Jadwal** (daftar), **tahapan** (butir) — D15 | Timeline, Milestone |
| vendor, venue | **Vendor**, **Venue** (lazim di industri WO, dipertahankan) | — |
| tenant (hanya Platform Console internal) | **Tenant**; untuk WO sendiri: "usaha Anda" | — |

**Aturan tulisan:**
- Tombol: kata kerja + objek, 1–4 kata, sentence case: "Simpan pembayaran", "Undang klien", "Kirim via WhatsApp".
  Tidak ada "Submit", "OK", "Ya/Tidak". Nama aksi sama sepanjang alur: tombol "Arsipkan proyek" → toast "Proyek
  diarsipkan · Urungkan".
- Galat: apa yang terjadi + cara memperbaiki, tanpa "tidak valid", tanpa menyalahkan, tanpa minta maaf:
  "Nominal melebihi sisa tagihan (Rp2.500.000). Ubah nominal atau pilih jenis Tambahan." Bukan "Input tidak valid".
- Sapaan "Anda" (kapital) di Portal dan pesan ke klien; console boleh label ringkas tanpa sapaan.
- Uang "Rp1.500.000" (T18); tanggal "9 Nov 2026" (ringkas) / "Senin, 9 November 2026" (lengkap); jarak acara "H-30",
  "Hari H", "H+3".
- Status untuk klien selalu bahasa awam dan tidak menyalahkan (FLOW-IDEAL §4, D12).

## 8. Bantuan dan onboarding (baru di revisi 2)

- **Checklist "Mulai di sini"** (`OnboardingChecklist`, sudah ada): ≤ 5 langkah (Lengkapi profil usaha, Buat proyek
  pertama, Atur termin, Undang klien, Undang tim), progres terlihat, bisa ditutup dan dibuka lagi dari menu akun (T25).
- **Tampilan kosong yang mengajar:** satu kalimat untuk apa halaman ini, gambaran isi bila terisi (baris contoh
  pudar), satu CTA. Contoh Termin: "Atur jadwal termin agar klien tahu kapan harus membayar. [Pakai templat DP–Termin–
  Pelunasan]".
- **Bantuan kontekstual, bukan tur:** teks pembantu di bawah field untuk istilah yang tidak jelas (termin, nilai
  kontrak, "terlihat klien"), ikon (?) dengan popover untuk konsep, bukan untuk setiap field. Tanpa tur popup.
- **Nilai bawaan yang cerdas:** termin dari templat relatif ke tanggal acara, PIC = pembuat, tanggal pembayaran =
  hari ini, nominal pelunasan = sisa.
- **Portal:** petunjuk "Tambahkan ke Layar Utama" sekali, kontak PIC selalu satu ketukan.

## 9. Aksesibilitas (gerbang, bukan saran)

- WCAG 2.2 AA: kontras teks 4,5:1, komponen UI dan fokus 3:1, target ≥ 24 px (Portal ≥ 44 px), fokus tidak tertutup
  header sticky, otentikasi mengizinkan pengelola kata sandi [wcag-258][wcag-1411][wcag-2411].
- Pola APG untuk dialog, alertdialog, combobox, tabel sortable [apg-dialog][apg-alertdialog][apg-combobox][apg-sort].
  Base UI menyediakan perilakunya; tanggung jawab kita: label, judul dialog, nama aksesibel ikon.
- `prefers-reduced-motion` untuk semua animasi [mdn-reduced-motion].
- Otomatis: `@axe-core/playwright` per rute setelah membuka dialog/menu; cakupan alat otomatis terbatas [deque], jadi
  uji papan ketik dan pembaca layar manual tetap wajib [playwright-a11y].
- Perangkat nyata: iOS Safari (dialog, input, scroll lock) dan Android Chrome kelas menengah.

## 10. Yang sengaja tidak diadopsi

Glass/translusen, mode gelap pertama, bento seragam, asisten AI, palet perintah sebagai navigasi utama, tur popup,
Gantt di HP, grafik pie/donat/gauge, animasi masuk per bagian, label huruf kapital semua, kartu seragam dengan bayangan
sama, sidebar blok warna merek penuh, gradien dekoratif.

## 11. Yang belum terverifikasi

Fitur `tnum` di Plus Jakarta Sans (D14); ukuran gzip tiap komponen shadcn; perilaku portal Base UI terhadap variabel
tenant (prototipe Tahap 0); bukti adopsi palet perintah; nada pesan tunggakan (D12); pedoman tulisan resmi dari
perusahaan Indonesia (GoTo Asphalt, Tokopedia Unify tidak menerbitkan pedoman publik yang bisa dibuka); aturan resmi
"Anda" vs "kamu" di antarmuka (tidak ada).

## 12. Sumber (diakses 8–9 Nov 2026)

**Dibaca langsung [V]:** [shadcn-changelog] https://ui.shadcn.com/docs/changelog · [shadcn-baseui]
https://ui.shadcn.com/docs/changelog/2026-07-base-ui-default · [baseui-releases]
https://base-ui.com/react/overview/releases · [radix-releases] https://www.radix-ui.com/primitives/docs/overview/releases ·
[shadcn-toast] https://ui.shadcn.com/docs/changelog/2026-07-toast · [shadcn-field]
https://ui.shadcn.com/docs/components/radix/field · [gatera] https://gatera.id/ · [wa]
https://faq.whatsapp.com/5913398998672934

**Dilaporkan agen riset [R], primer:**
- Produk: [planningpod-client] https://planningpod.com/help-center/for-clients-what-is-the-client-portal-and-how-do-i-use-it ·
  [aisle-portal] https://aisleplanner.com/products/client-portal/ · [aisle-invoices]
  https://help.aisleplanner.com/en/articles/2084588-tracking-vendor-invoices-in-aisle-planner-s-budget · [dubsado]
  https://help.dubsado.com/en/articles/467089-payment-plans · [honeybook-reminders]
  https://honeybook.com/product/payment-reminders · [inframe] https://www.inframelinea.com/ · [telkomu]
  https://repositori.telkomuniversity.ac.id/pustaka/files/174785/bab1/aplikasi-berbasis-web-untuk-pengelolaan-piutang-pada-pembayaran-customer-ke-wedding-organizer-studi-kasus-indah-wedding-organizer-kota-padang-.pdf
- NN/g: [nng-dash] https://www.nngroup.com/articles/dashboards-preattentive/ · [nng-hamburger]
  https://www.nngroup.com/articles/hamburger-menus/ · [nng-mobilenav] https://www.nngroup.com/articles/mobile-navigation-patterns/ ·
  [nng-empty] https://www.nngroup.com/articles/empty-state-interface-design/ · [nng-form-errors]
  https://www.nngroup.com/articles/errors-forms-design-guidelines/ · [nng-modal] https://www.nngroup.com/articles/modal-nonmodal-dialog/ ·
  [nng-confirm] https://www.nngroup.com/articles/confirmation-dialog/ · [nng-notif]
  https://www.nngroup.com/articles/indicators-validations-notifications/ · [nng-glass] https://www.nngroup.com/articles/liquid-glass/ ·
  [nng-dark] https://www.nngroup.com/articles/dark-mode-users-issues/ · [nng-progress]
  https://www.nngroup.com/articles/progress-indicators/ · [nng-status-tracker]
  https://www.nngroup.com/articles/status-tracker-progress-update/ · [nng-skeleton] https://www.nngroup.com/articles/skeleton-screens/ ·
  [nng-response] https://www.nngroup.com/articles/response-times-3-important-limits/ · [nng-bulk]
  https://www.nngroup.com/videos/bulk-actions-design-guidelines/ · [nng-accelerators] https://www.nngroup.com/articles/ui-accelerators/ ·
  [nng-progressive] https://www.nngroup.com/articles/progressive-disclosure/ · [nng-redesign]
  https://www.nngroup.com/videos/redesign-incremental-vs-overhaul/
- Sistem desain: [linear-refresh] https://linear.app/now/behind-the-latest-design-refresh · [geist] https://vercel.com/design.md ·
  [atl-elevation] https://atlassian.design/foundations/elevation · [atl-type] https://atlassian.design/foundations/typography ·
  [atl-inline-edit] https://atlassian.design/components/inline-edit/usage · [primer-type] https://primer.style/foundations/typography ·
  [primer-table] https://primer.style/product/components/data-table/ · [stripe-style] https://docs.stripe.com/stripe-apps/style ·
  [stripe-patterns] https://docs.stripe.com/stripe-apps/patterns · [carbon-motion]
  https://carbondesignsystem.com/elements/motion/overview/ · [carbon-table]
  https://carbondesignsystem.com/components/data-table/usage/ · [carbon-status]
  https://carbondesignsystem.com/patterns/status-indicator-pattern/ · [polaris-filters]
  https://shopify.dev/docs/apps/build/app-home/migrate-from-polaris-react/index-filters · [polaris-setup]
  https://shopify.dev/docs/api/app-home/patterns/compositions/setup-guide · [shadcn-theming] https://ui.shadcn.com/docs/theming ·
  [govuk-validation] https://design-system.service.gov.uk/patterns/validation/ · [grafana-ux]
  https://grafana.com/docs/writers-toolkit/write/style-guide/ux-writing/
- Standar dan platform: [wcag-258] https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html · [wcag-1411]
  https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html · [wcag-2411]
  https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum.html · [android-touch]
  https://support.google.com/accessibility/android/answer/7101858 · [apg-dialog] https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/ ·
  [apg-alertdialog] https://www.w3.org/WAI/ARIA/apg/patterns/alertdialog/ · [apg-combobox]
  https://www.w3.org/WAI/ARIA/apg/patterns/combobox/ · [apg-sort] https://www.w3.org/WAI/ARIA/apg/patterns/table/examples/sortable-table/ ·
  [mdn-reduced-motion] https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion · [webdev-vitals]
  https://web.dev/articles/vitals · [webdev-inp] https://web.dev/articles/inp · [webdev-offline]
  https://web.dev/articles/offline-ux-design-guidelines · [react-optimistic] https://react.dev/reference/react/useOptimistic ·
  [playwright-a11y] https://playwright.dev/docs/accessibility-testing · [deque]
  https://www.deque.com/blog/automated-testing-study-identifies-57-percent-of-digital-accessibility-issues ·
  [statcounter] https://gs.statcounter.com/os-market-share/mobile/indonesia · [puebi]
  https://id.wikisource.org/wiki/Halaman:Peraturan_Menteri_Pendidikan_dan_Kebudayaan_Nomor_50_Tahun_2015_tentang_Pedoman_Umum_Ejaan_Bahasa_Indonesia.pdf/22
- API: [ms-bff] https://learn.microsoft.com/en-us/azure/architecture/patterns/backends-for-frontends · [aip-134]
  https://google.aip.dev/134 · [rfc9457] https://www.rfc-editor.org/rfc/rfc9457.html · [zalando]
  https://opensource.zalando.com/restful-api-guidelines/ · [stripe-idem] https://docs.stripe.com/api/idempotent_requests ·
  [s3-presigned] https://docs.aws.amazon.com/AmazonS3/latest/userguide/PresignedUrlUploadObject.html
- Lain: [pigment] https://engineering.pigment.com/2026/04/08/safe-user-impersonation/ · [baymard-inline]
  https://baymard.com/blog/inline-form-validation · [tldraw-baseui] https://github.com/tldraw/tldraw/issues/7584

**Sekunder [S]:** [selular] https://selular.id/2025/08/pasar-smartphone-indonesia-kuartal-ii-2025-hp-murah-dan-menengah-jadi-andalan/ ·
perbandingan shadcn/Radix/Base UI (dev.to, openreplay), saasframe cmd+K, aturan rupiah di media (Detik), blog tabel/UX,
penulisan "Anda" (komunitasbelajar.id).

**Tidak berhasil dibuka [?]:** pedoman tulisan GoTo/Tokopedia/Traveloka/Bukalapak, panduan pelacakan pesanan Baymard
(berbayar), pola struk GoPay/DANA/Flip, angka desain Linear dan gerak Material 3, sumber bentuk timeline (rel vs Gantt),
nada pesan tunggakan, fitur `tnum` Plus Jakarta Sans.
