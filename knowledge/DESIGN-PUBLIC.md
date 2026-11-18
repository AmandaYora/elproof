# DESIGN-PUBLIC.md — Situs publik ElProof (12 halaman sebelum masuk)

> Status: arah "Bukti yang Terlihat" sejak 16 Nov 2026 (ADR-0045; riset dan brief: `docs/plan/redesign-situs-publik/BRIEF.md`).
> Bagian di bawah yang masih menyebut plum/blush adalah arah lama "Janji Terbukti" (ADR-0038); nilai warna yang berlaku ada di §3.
> Dipakai sejak 10 Nov 2026 (ADR-0038). Ringkasan dari brief desain bersumber tugas redesain halaman publik (riset
> 10 Nov 2026; sumber di §8). Aplikasi setelah login mengikuti `DESIGN.md`; dokumen ini hanya untuk Beranda, Tentang Kami,
> FAQ, Kontak, Syarat & Ketentuan, Kebijakan Privasi, Pemrosesan Data, Kebijakan Refund, Masuk, Daftar, Siapkan workspace,
> dan Undangan.

## 1. Kasus dan arah

- **Audiens:** pemilik dan tim WO Indonesia yang menilai ElProof sebelum mendaftar, banyak dari tautan WhatsApp/Instagram di
  HP Android kelas menengah. Pertanyaan yang harus terjawab dalam beberapa detik: "Ini untuk WO seperti saya? Apa bedanya
  dengan WhatsApp + spreadsheet? Berapa harganya? Bagaimana mulai?"
- **Arah terpilih (pengguna):** **"Janji Terbukti"** — editorial yang tenang dan hangat: plum tua, emas segel hanya sebagai
  hiasan, latar putih dengan pita blush. **Satu keluarga huruf** (Plus Jakarta Sans), judul tebal dan rapat. Keanggunan
  datang dari warna, ruang kosong, segel, dan komposisi, bukan dari huruf serif.
- **Elemen khas:** segel bukti (`ProofSeal`) yang dicapkan satu kali pada pembayaran yang sudah diverifikasi di pratinjau
  Portal Klien. Segel hanya muncul pada hal yang benar-benar terverifikasi.
- **Logo:** logo navy-emas (`AppLogoMark`) + nama ElProof (`BrandMark`), sama di header, footer, dan layar masuk.
- **Arah sekarang (16 Nov 2026, ADR-0045):** navy + emas segel dari logo; hero = judul lebar + `HeroScreens` (WO Console + Portal Klien,
  segel yang sama pada "Termin 2" di kedua layar); pita bukti berisi fakta yang bisa dicek; bento `Journey` "Dari H-180 sampai hari H";
  kartu harga; WhatsApp di header, di samping setiap "Coba gratis", dan bilah aksi HP yang muncul setelah hero lewat.
- **Copy (16 Nov 2026, keputusan pemilik):** ditulis dari sudut pandang pemilik WO, di dua pilar yang mereka pedulikan:
  **kerja tim lebih ringan** (efektif, efisien) dan **klien lebih percaya, penjualan ikut naik**. Hero "Kerja tim lebih ringan,
  klien makin percaya."; pita di bawah hero berisi manfaat (bukan siapa pembuatnya); bento = pilar 1; pita navy = pilar 2.
  Setiap manfaat disertai mekanismenya (fitur yang benar-benar ada), tanpa angka yang tidak bisa dibuktikan. Siapa pembuatnya
  (Elcodelabs) cukup di footer dan Tentang Kami.

## 2. Temuan → keputusan (ringkas)

| Temuan | Keputusan |
|---|---|
| Landing B2B yang dinilai profesional menaruh UI produk nyata dan judul spesifik di atas lipatan [linear][attio][stripe] | Hero = judul tentang hasil untuk WO + pratinjau Portal Klien (HTML/CSS, bukan gambar/video) |
| Demo interaktif pendek (≤ 6 langkah, tanpa formulir gerbang) lebih efektif dari video; di HP perlu bentuk khusus [navattic] | Pratinjau WO Console bertab yang bisa diklik + cerita H-180 → hari H; dimuat lazy |
| Pesaing lokal tanpa harga publik atau dengan klaim yang tidak cocok hitungannya [gatera][yourdae][aksana] | Harga dari data paket aktif, "Paling hemat"/"Hemat X%" dihitung, tanpa superlatif dan tanpa testimoni |
| Scrolljacking membingungkan; animasi saat menggulir harus bisa dimatikan [nng-scrolljack][wcag-233] | Reveal CSS saja, hanya Beranda/Tentang, mati bila "kurangi gerakan"; tanpa parallax |
| Masuk/daftar: tombol Google sama menonjol, formulir di atas pada HP, `autocomplete` benar [google-brand][webdev-signup] | Tata letak terbelah; di HP formulir lebih dulu, panel foto disembunyikan |

## 3. Token (`src/theme/theme.css` + permukaan di `marketing.css`)

Sejak ADR-0045 (16 Nov 2026) palet navy + emas dipakai semua permukaan; `marketing.css` hanya mengganti permukaan publik (halaman
putih, pita kabut, garis). Tabel kontras lengkap: `DESIGN.md` §4.1.

| Peran | Nilai | Kontras (dihitung) |
|---|---|---|
| `--background` / `--card` | #FFFFFF | — |
| `--foreground` | #0B1B43 | 16,77:1 di putih · 14,82:1 di kabut |
| `--muted` (pita kabut) | #EEF1F7 | — |
| `--muted-foreground` | #4F5A78 | 6,85:1 di putih · 6,05:1 di kabut |
| `--input` (batas kontrol) | #7A84A0 | 3,73:1 di putih · 3,29:1 di kabut |
| `--primary` navy (tombol, pita gelap, fokus) | #0B1B43 | 16,77:1 dengan teks putih |
| `--seal` emas (segel, garis, ikon di latar terang; non-teks) | #B4853D | 3,30:1 di putih |
| `--seal-on-primary` emas logo (di pita navy, boleh teks) | #E0B074 | 8,49:1 di navy |
| `--seal-text` / `--seal-soft` | #7A5A1E / #F8F0E1 | 5,60:1 |
| `--highlight-text` (mis. "H-180") | #8A5A12 | 5,91:1 di putih |
| `--on-primary-muted` (teks redup di navy) | #C3CBE0 | 10,33:1 |

Status (Lunas, Belum dibayar) tetap memakai token status konsol + ikon + teks. Huruf: Plus Jakarta Sans; skala judul
satu sumber di `homepage/components/typography.ts` (`TYPE`), semua **bold 700** (bukan extrabold) dengan tracking tipis:
hero 52/44/34 px (desktop/tablet/HP, line-height 1,12, dua baris di desktop), h1 halaman lain 40/30 px, h2 bagian 34/26 px,
subjudul 21/19 px. Judul hanya mengantar; yang "keras" adalah layar produk (18 Nov 2026: hero 72 px extrabold dinilai
pemilik terlalu besar dan tidak profesional). Badan 16–18 px, baris ≤ 70 karakter; uang "Rp1.500.000" tabular. Sudut: kontrol & tombol pil, kartu 16 px, bingkai HP 32–40 px. Target sentuh ≥ 44 px (CTA 48 px).

## 4. Susunan halaman

- **Beranda:** hero (judul + Portal Klien dengan segel) → pita fakta → cara kerja H-180 → hari H (`#cara-kerja`) →
  pratinjau WO Console yang bisa diklik → dua layar, satu catatan (pita plum) → harga (`#harga`) → pertanyaan umum ringkas
  → ajakan penutup. **Satu contoh data** untuk semua ilustrasi (hero, kartu "Cara kerja", pratinjau): `homepage/data/sample.ts`
  (Raka & Dinda, Sabtu 12 Des 2026, kontrak Rp45.000.000 = uang muka 30% + Termin 2 40% + pelunasan 30%, diverifikasi Rina,
  dipotret pada H-32). Angka H-, jumlah, dan tanggal diturunkan dari situ; jangan menulis angka contoh langsung di komponen
  (QA FQ-11). Grid satu kolom di HP memakai `grid-cols-1` (`minmax(0,1fr)`), bukan trek `auto` yang melebar ke baris
  terpanjang (QA RT-05; dijaga `structure.spec.ts` "no sideways scroll at 360 and 375 px").
- **Tentang kami:** satu kolom prosa, tiga prinsip, ajakan.
- **FAQ:** akordeon berkelompok, setiap pertanyaan punya tautan `#id`. Tidak menyebut nama paket atau harga (halaman
  diprarender; paket yang dijual dibaca langsung oleh bagian Harga, QA FQ-10).
- **Kontak:** WhatsApp dengan pesan terisi, email, jam layanan, alamat; tanpa formulir pajangan.
- **Halaman hukum:** tata letak baca, daftar isi lengket (desktop) / lipat (HP). Di bawah judul selalu ada penyelenggara
  (`COMPANY` + `CONTACT` di `homepage/data/contact.ts`: PT Prayora Karya Pratama (Elcodelabs), alamat, kontak pelindungan
  data; QA FQ-12). Istilah mengikuti glosarium aplikasi (proyek, klien, jadwal/tahapan, progres, baca-saja, Portal Klien,
  anggota tim, workspace; "Owner" tetap karena nama peran terkunci) dan didefinisikan sekali di Syarat dan ketentuan §1
  (QA FQ-09); `check:terms` juga memindai paragraf `body: [...]`.
- **Nama halaman:** sentence case dan satu sumber: judul di `PUBLIC_PAGES` dipakai tab, h1 halaman hukum, tautan menu dan
  footer, sitemap, dan Open Graph ("Tentang kami", "Syarat dan ketentuan", "Kebijakan privasi", "Perjanjian pemrosesan
  data", "Kebijakan refund"; QA FQ-14).
- **Masuk, Daftar, Siapkan workspace, Undangan:** `AuthSplitLayout` — panel plum dengan foto dari Platform Console (segel bila
  tidak ada foto), formulir di kanan; h1 di kolom formulir.

## 5. Gerak dan aksesibilitas

- `.reveal`: CSS `animation-timeline: view()` dalam `@supports` + `prefers-reduced-motion: no-preference`; opacity + ≤ 12 px
  selama paruh pertama elemen masuk layar, terikat posisi gulir (bisa diputar balik saat elemen keluar lewat bawah); hanya
  Beranda dan Tentang; tidak pada harga, FAQ, formulir, atau hukum. Tidak ada JavaScript yang menyembunyikan isi.
- Gerak yang menjawab tindakan (pudar singkat saat langkah "Cara kerja" atau tab pratinjau dipilih) boleh, di balik `motion-safe:`.
- `.seal-press`: satu momen ekspresif (≤ 700 ms) di hero; statis bila "kurangi gerakan".
- Tab pratinjau pola APG (panah, Home/End); akordeon dengan `aria-expanded`; "Lewati ke isi"; fokus 2 px warna plum.

## 6. Harga

Sumber `GET /api/v1/public/plans` (ADR-0038). Baris per paket, urut durasi; "RpX per bulan" dihitung; "Paling hemat" untuk
harga per bulan termurah (tidak ada bila seri); "Hemat X%" terhadap paket terpendek. Memuat = kerangka tanpa angka; galat
atau kosong = ajakan bertanya lewat WhatsApp.

Judul dan pengantar bagian harga mengikuti data (`pricingIntro`, EP-14), tidak pernah menjanjikan pilihan yang tidak ada:

| Paket aktif | Judul | Pengantar |
|---|---|---|
| belum terbaca / galat / kosong | Harga terbuka | Mulai dengan coba gratis 14 hari. Tanpa kartu kredit. |
| satu paket | Harga terbuka, satu paket | … lanjutkan dengan paket {nama}, ditagih per bulan / per N bulan. |
| beberapa paket, satu durasi | Harga terbuka, pilih paketnya | … pilih paket yang sesuai kebutuhan tim Anda; semuanya ditagih … |
| beberapa durasi | Harga terbuka, pilih lamanya | … pilih paket sesuai lama yang Anda butuhkan. |

Satu paket saja → tombol "Pilih paket" berwarna utama. Fitur paket ditulis admin di Platform Console (data, bukan teks
UI); formulir paket ElProof mengingatkan agar ditulis dalam bahasa Indonesia.

**Daftar dengan paket (`/daftar?paket=<id>`):** kotak "Paket pilihan Anda" (nama, harga, periode, tautan "Ganti paket" ke
`/#harga`) dan catatan bahwa pendaftaran dimulai dengan coba gratis 14 hari tanpa tagihan; paket dibayar nanti di menu
Langganan dan masa langganannya dimulai setelah masa coba habis (`SubscriptionPage`: sisa masa coba dipertahankan).
Id yang tidak dikenal, paket nonaktif, atau API gagal → Daftar biasa tanpa kotak. Kotak muncul setelah data paket
terbaca (render pertama sama dengan HTML prarender).

## 6a. Mesin pencari (ADR-0043)

Sepuluh halaman (`PUBLIC_PAGES`: Beranda, Tentang Kami, FAQ, Kontak, empat halaman hukum, Daftar, Masuk) ditulis ke HTML
saat build dengan judul, deskripsi, canonical, Open Graph/Twitter, dan isinya sendiri, lalu dihidrasi di browser.
`sitemap.xml` memuat halaman yang boleh diindeks (tanpa Masuk); alamat yang tidak dikenal router dijawab 404 oleh server.
Render pertama halaman publik tidak boleh bergantung pada browser (ukuran layar, storage, query string): baca di efek
atau lewat `useMediaQuery`.

## 7. Anti-pola

Gradien/kaca dekoratif, satu frasa diwarnai beda di judul, label huruf kapital di atas judul, kartu ikon seragam, angka
atau logo pelanggan yang tidak benar, superlatif, hex di komponen, isi yang disembunyikan menunggu JavaScript.

## 8. Sumber (diakses 10 Nov 2026)

[linear] https://linear.app/ · [attio] https://attio.com/ · [stripe] https://stripe.com/ · HoneyBook https://www.honeybook.com/ ·
Aisle Planner https://www.aisleplanner.com/ · [gatera] https://gatera.id/ · [yourdae] https://yourdae.com/ · [aksana]
https://aksana.co.id/ · [navattic] https://navattic.com/report/state-of-the-interactive-product-demo-2026 · [nng-scrolljack]
https://www.nngroup.com/articles/scrolljacking-101/ · NN/g harga https://www.nngroup.com/articles/show-price/ · [wcag-233]
https://www.w3.org/WAI/WCAG22/Understanding/animation-from-interactions.html · [google-brand]
https://developers.google.com/identity/branding-guidelines · [webdev-signup] https://web.dev/articles/sign-up-form-best-practices ·
Scroll-driven animations https://developer.chrome.com/docs/css-ui/scroll-driven-animations · DataReportal Indonesia 2026
https://datareportal.com/reports/digital-2026-indonesia
