# DESIGN-PUBLIC.md — Situs publik ElProof (12 halaman sebelum masuk)

> Status: dipakai sejak 10 Nov 2026 (ADR-0038). Ringkasan dari brief desain bersumber tugas redesain halaman publik (riset
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
- **Logo:** tanda bulat "EP" + nama ElProof (`BrandMark`), sama di header, footer, dan layar masuk.

## 2. Temuan → keputusan (ringkas)

| Temuan | Keputusan |
|---|---|
| Landing B2B yang dinilai profesional menaruh UI produk nyata dan judul spesifik di atas lipatan [linear][attio][stripe] | Hero = judul tentang hasil untuk WO + pratinjau Portal Klien (HTML/CSS, bukan gambar/video) |
| Demo interaktif pendek (≤ 6 langkah, tanpa formulir gerbang) lebih efektif dari video; di HP perlu bentuk khusus [navattic] | Pratinjau WO Console bertab yang bisa diklik + cerita H-180 → hari H; dimuat lazy |
| Pesaing lokal tanpa harga publik atau dengan klaim yang tidak cocok hitungannya [gatera][yourdae][aksana] | Harga dari data paket aktif, "Paling hemat"/"Hemat X%" dihitung, tanpa superlatif dan tanpa testimoni |
| Scrolljacking membingungkan; animasi saat menggulir harus bisa dimatikan [nng-scrolljack][wcag-233] | Reveal CSS saja, hanya Beranda/Tentang, mati bila "kurangi gerakan"; tanpa parallax |
| Masuk/daftar: tombol Google sama menonjol, formulir di atas pada HP, `autocomplete` benar [google-brand][webdev-signup] | Tata letak terbelah; di HP formulir lebih dulu, panel foto disembunyikan |

## 3. Token (`src/theme/theme.css` + permukaan di `marketing.css`)

Sejak ADR-0039 (10 Nov 2026) palet ini dipakai juga oleh semua konsol dan nilainya tinggal di `theme.css`; `marketing.css` hanya
mengganti permukaan publik (halaman putih, pita blush, garis blush). Tabel kontras konsol: `DESIGN.md` §4.1.

| Peran | Nilai | Kontras (dihitung) |
|---|---|---|
| `--background` / `--card` | #FFFFFF | — |
| `--foreground` | #2A1A27 | 16,9:1 di putih |
| `--muted` (pita blush) | #F7EEEC | — |
| `--muted-foreground` | #6B5566 | 6,74:1 di putih · 5,90:1 di blush |
| `--subtle-foreground` (placeholder) | #7E6A7A | 4,96:1 di putih saja |
| `--input` (batas kontrol) | #8E7A8B | 3,96:1 di putih · 3,47:1 di blush |
| `--primary` plum (tombol, pita gelap, fokus) | #3E1F3B | 14,34:1 dengan teks putih |
| `--accent` | #EFE3EC | plum di atasnya ≥ 11:1 |
| `--seal` emas (segel, garis; non-teks) | #B88A3E | 3,11:1 di putih |
| `--seal-text` / `--seal-soft` | #7A5A1E / #F8F0E1 | 5,60:1 |
| `--highlight-text` (mis. "H-180") | #9B3D4A | 6,65:1 di putih |
| `--on-primary-muted` (teks redup di plum) | #E9D9C4 | 10,37:1 |

Status (Lunas, Belum dibayar) tetap memakai token status konsol + ikon + teks. Huruf: Plus Jakarta Sans; hero
`font-extrabold`, tracking −0,035em, line-height 1,04; h2 extrabold −0,03em; badan 16–18 px, baris ≤ 70 karakter; uang
"Rp1.500.000" tabular. Sudut: kontrol & tombol pil, kartu 16 px, bingkai HP 32–40 px. Target sentuh ≥ 44 px (CTA 48 px).

## 4. Susunan halaman

- **Beranda:** hero (judul + Portal Klien dengan segel) → pita fakta → cara kerja H-180 → hari H (`#cara-kerja`) →
  pratinjau WO Console yang bisa diklik → dua layar, satu catatan (pita plum) → harga (`#harga`) → pertanyaan umum ringkas
  → ajakan penutup.
- **Tentang Kami:** satu kolom prosa, tiga prinsip, ajakan.
- **FAQ:** akordeon berkelompok, setiap pertanyaan punya tautan `#id`.
- **Kontak:** WhatsApp dengan pesan terisi, email, jam layanan, alamat; tanpa formulir pajangan.
- **Halaman hukum:** tata letak baca, daftar isi lengket (desktop) / lipat (HP), teks hukum tidak diubah.
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
