# Brief desain — Redesain situs publik ElProof ("Bukti yang Terlihat")

Tech Lead · 2026-11-16 · Proyek: `apps/web` (modul `homepage`, `auth`) · Mengikuti skill `rudis-riset-desain`
**Status: dibangun 16 Nov 2026 (ADR-0045).** Keputusan pemilik: palet navy + emas untuk **seluruh aplikasi**. Default yang dipakai
untuk keputusan yang belum dijawab: tanpa foto pernikahan (hak pakai belum pasti), tanpa bagian kutipan (belum ada WO yang bersedia),
tanpa huruf serif, pita bukti hanya berisi fakta yang sudah tertulis di situs (14 hari gratis, harga terbuka, Portal dari WhatsApp,
Elcodelabs Bandung). Penyimpangan dari brief setelah kritik visual: pita terang memakai kabut dingin #EEF1F7, bukan krem #F6F3EE
(krem + serif + aksen hangat adalah tampilan templat yang paling umum); hero memakai judul lebar di atas dan dua layar selebar halaman di
bawahnya (judul di kolom sempit pecah 4 baris dan konsolnya terlalu kecil).

## 0. Diagnosis situs sekarang (production, 16 Nov 2026, 1440 px dan iPhone 13)

| Gejala | Dampak ke kesan "meyakinkan" |
|---|---|
| UI berwarna **plum** (#3E1F3B), sementara logo baru **navy + emas** | Merek terasa tidak satu; logo dan halaman seperti milik dua produk berbeda |
| Hero = judul besar + **satu mockup HP kecil** (±300 px) di ruang putih yang luas | Produk tidak "terlihat". Situs terbaik menaruh UI produk besar dan detail di atas lipatan |
| Tidak ada satu pun bukti di bawah hero (logo, angka, kutipan, wajah, alamat perusahaan) | Pengunjung tidak tahu siapa di balik produk dan siapa yang sudah memakainya |
| Bagian "Cara kerja" = garis waktu teks dengan jarak kosong ratusan px; HP lengket kecil | Terasa kosong dan lambat, bukan premium |
| Harga = satu baris tipis, fitur berbahasa Inggris ("Full Features…") | Terlihat belum selesai |
| Tidak ada foto manusia/pernikahan sama sekali | Untuk produk wedding, terasa dingin dan generik |
| WhatsApp hanya di halaman Kontak | Audiens Indonesia terbiasa bertanya lewat WhatsApp sebelum mendaftar |

## 1. Subjek, audiens, tugas utama

- **Subjek:** ElProof, SaaS untuk wedding organizer Indonesia. WO mencatat tahapan, vendor, dan pembayaran; kliennya
  (pasangan) melihat progres dan buktinya di Portal Klien.
- **Audiens & perangkat:** pemilik/tim WO, kebanyakan membuka dari tautan WhatsApp/Instagram di HP Android kelas
  menengah; sebagian membandingkan di laptop sebelum membayar.
- **Tugas utama halaman (5 detik):** "Ini untuk WO seperti saya? Seperti apa aplikasinya? Siapa yang menjamin? Berapa
  harganya? Bagaimana mulai / tanya dulu?"
- **Design system yang tetap dipakai:** token di `src/theme/theme.css` + `marketing.css`, kit shadcn/Base UI,
  `shared/components/ui`, Plus Jakarta Sans (sudah di-bundle), `AppLogoMark`, prarender + `usePageMeta` (ADR-0043),
  reveal CSS (ADR-0038). Yang berubah adalah nilai token dan komposisi halaman, bukan arsitekturnya.

## 2. Temuan riset → keputusan

| Temuan (sumber) | Keputusan |
|---|---|
| Linear, Attio, Mercury, Notion, Ramp menaruh **UI produk besar dan berisi data realistis** tepat di bawah/di samping judul; UI itu memenuhi ±60–70 % lebar layar [linear][attio][mercury][notion][ramp] | Hero baru: judul + **komposisi dua layar**: WO Console (bingkai laptop, lebar) bertumpuk dengan Portal Klien (bingkai HP), keduanya HTML/CSS hidup, data contoh realistis |
| Tidak ada studi terkontrol screenshot vs ilustrasi; rekomendasi praktisi condong ke **UI produk asli**; ilustrasi hanya untuk konsep yang tak bisa dipotret [screenshot-vs-illus] | Tidak memakai ilustrasi abstrak/3D. Semua visual = UI produk atau foto pernikahan nyata |
| Situs vertikal event/wedding (Aisle Planner, HoneyBook, Dubsado, Planning Pod) memakai **foto manusia & momen pernikahan**, pemilihan "jenis bisnis", dan pita angka/kutipan berwarna hangat (Aisle Planner: hitam + emas) [aisleplanner][honeybook][dubsado][planningpod] | Pakai foto pernikahan (sudah ada di Tampilan Login platform) di dua tempat saja: pita "dua layar, satu catatan" dan panel masuk/daftar. Emas logo jadi warna aksen pita bukti |
| Kepercayaan dinilai dari: kualitas desain, keterbukaan (kontak, harga, biaya), isi lengkap & terkini, dan keterhubungan dengan web luar [nng-trust]; B2B harus **menampilkan harga** [nng-price] | Pita bukti di bawah hero berisi fakta yang bisa dicek (lihat §2a); alamat & badan hukum di footer dan Tentang; harga tetap terbuka dan diperjelas |
| Tanpa logo pelanggan, praktisi menyarankan bukti spesifik (sedikit tapi nyata), bukti proses, bukti pendiri, label tahap yang jujur; **jumlah pelanggan kecil bisa merugikan** [trust-early][logo-not-proof] | **Tidak** menampilkan jumlah tenant. Pakai: 1–3 kutipan WO nyata **dengan izin**, "Cara kami menjaga data", siapa pembuatnya (Elcodelabs, Bandung) |
| Situs SaaS Indonesia (Mekari, Majoo) menaruh **WhatsApp** di navigasi/tombol mengambang, rating pihak ketiga, dan foto orang lokal [mekari][majoo] | Tombol "Tanya lewat WhatsApp" di header (desktop) dan bilah bawah lengket (HP), pesan terisi; tanpa widget chat pihak ketiga |
| Bagian fitur di Stripe/Ramp/Notion = **grid bento** kartu tidak seragam, masing-masing berisi potongan UI kecil, bukan ikon [stripe][ramp][notion] | Ganti garis waktu teks dengan **bento "dari H-180 ke hari H"**: 5 kartu ukuran berbeda, tiap kartu memuat potongan UI asli (termin, bukti, vendor, jadwal, portal) |
| Desain sederhana dengan tawaran jelas mengalahkan yang "berkilau"; CTA berulang; cepat di HP [perspective][vezadigital] | Tetap tenang: satu warna gelap, satu aksen, banyak ruang tetapi tidak kosong; CTA di hero, setelah bento, setelah harga, penutup; anggaran bundel tetap |
| Scroll-jacking membingungkan; gerak harus bisa dimatikan [nng-scrolljack][wcag-233] | Tetap reveal CSS ringan + satu momen segel; tanpa parallax/pin; `prefers-reduced-motion` mematikan semua |

Elemen khas: **segel bukti emas** dari logo (lingkaran emas bercentang), dipakai *hanya* pada hal yang sudah
terverifikasi: di hero ia "mencap" termin yang lunas di Portal Klien, lalu muncul sebagai penanda di setiap kartu bento.
Bersumber dari subjek (bukti pembayaran/tahapan yang diverifikasi) dan dari logo baru, bukan hiasan.

### 2a. Pita bukti di bawah hero (hanya fakta yang benar)

Usulan isi, masing-masing harus dikonfirmasi pemilik produk sebelum tayang:
"Coba gratis 14 hari, tanpa kartu kredit" · "Harga terbuka sejak hari pertama" · "Data Anda bisa diekspor kapan saja"
(cek fitur ekspor) · "Dibuat oleh Elcodelabs — PT Prayora Karya Pratama, Bandung" · kutipan WO nyata bila ada izin.
Dilarang: angka pelanggan, logo yang bukan pelanggan, rating yang tidak ada, "#1 di Indonesia".

## 3. Token (usulan; nilai kontras dihitung)

Arah warna: **navy logo sebagai warna utama, emas logo sebagai aksen segel**, latar putih dan krem-sangat-tipis
(bukan krem + terakota). Plum dihapus dari situs publik.

| Peran | Nilai | Kontras |
|---|---|---|
| `--primary` navy (tombol, pita gelap, fokus) | #0B1B43 | 16,8:1 dengan putih |
| `--primary-hover` | #16245A | 14,6:1 dengan putih |
| `--foreground` teks | #0B1B43 | 16,8:1 di putih · 15,2:1 di #F6F3EE |
| `--muted-foreground` | #5A6585 | 5,8:1 di putih · 5,2:1 di #F6F3EE |
| `--muted` (pita terang) | #EEF1F7 (semula #F6F3EE, lihat Status) | — |
| `--seal` emas (segel, garis, ikon; **non-teks** di latar terang) | #E0B074 (warna logo) | 2,6:1 di putih (hanya dekorasi) · 8,5:1 di navy |
| `--seal-text` (teks emas di latar terang) | #8A6424 | 5,3:1 di putih |
| teks emas di pita navy | #E8D3AE | 11,5:1 di navy |
| teks redup di pita navy | #A7B0C8 | 7,7:1 di navy |

- Status (Lunas, Jatuh tempo, Perlu ditangani) tetap token status + ikon + teks.
- **Huruf:** tetap Plus Jakarta Sans. Hero 56–64 px desktop / 36–40 px HP, extrabold, tracking −0,03em. Opsi (keputusan
  pemilik, §7): judul display serif (mis. Instrument Serif / Fraunces) untuk kesan "wedding premium" seperti Aisle
  Planner & Dubsado — risikonya terasa templat "serif + krem"; rekomendasi: **tidak**, keanggunan dari navy-emas + foto.
- **Spasi & sudut:** jarak antarbagian 96–128 px desktop / 64–80 px HP (sekarang ada celah >300 px yang harus hilang);
  kartu 20 px, bingkai laptop 14 px, HP 36 px; tombol pil 48 px.
- **Gerak:** satu momen segel (≤ 700 ms) di hero; reveal opacity + 12 px; hover kartu bento naik 2 px. Semua mati
  dengan `prefers-reduced-motion`.

## 4. Tata letak

```
Desktop (1440):
┌──────────────────────────────────────────────────────────────────────────┐
│ [logo] ElProof   Cara kerja  Harga  Tentang  FAQ     Masuk [WA] [Coba gratis]│
├──────────────────────────────────────────────────────────────────────────┤
│  Klien percaya karena          ┌──────────── WO Console (laptop) ───────┐ │
│  bisa melihat buktinya.        │ Beranda │ Raka & Dinda  H-32          │ │
│  sub (≤2 baris)                │ Proyek  │ 18/24 tahapan  Rp35jt/45jt  │ │
│  [Coba gratis 14 hari]         │ Jadwal  │ Yang perlu dikerjakan …     │ │
│  [Tanya lewat WhatsApp]        └──────────────────┬─────────────────────┘ │
│  Tanpa kartu kredit · Google                ┌─────┴──── Portal Klien (HP)┐│
│                                             │ Termin 2  ✔(segel emas)   ││
├──────────────────────────────────────────────────────────────────────────┤
│ pita bukti: 14 hari gratis · harga terbuka · ekspor data · Elcodelabs Bdg │
├──────────────────────────────────────────────────────────────────────────┤
│ "Dari H-180 ke hari H" — bento                                           │
│ ┌──── Termin & bukti (2x) ────┐ ┌ Vendor & venue ┐ ┌ Jadwal tahapan ┐     │
│ │ potongan UI + segel          │ │ potongan UI    │ │ potongan UI    │     │
│ └──────────────────────────────┘ └────────────────┘ └────────────────┘     │
│ ┌ Kendala ┐ ┌──────── Portal Klien dari tautan WhatsApp (2x) ────────┐    │
├──────────────────────────────────────────────────────────────────────────┤
│ pita NAVY: foto pernikahan | "Dua layar, satu catatan" + 2 daftar + CTA    │
├──────────────────────────────────────────────────────────────────────────┤
│ Pratinjau WO Console bertab (yang sekarang, diperbesar)                   │
├──────────────────────────────────────────────────────────────────────────┤
│ Harga: kartu paket (nama, harga, per bulan, fitur ✔, CTA) + "yang termasuk" │
│ Kutipan WO (bila ada izin) · Cara kami menjaga data (3 butir)              │
├──────────────────────────────────────────────────────────────────────────┤
│ FAQ ringkas │ Penutup navy: "Mulai dengan proyek berikutnya" [Coba][WA]   │
│ Footer: alamat, PT, kontak, legal                                         │
└──────────────────────────────────────────────────────────────────────────┘

HP (375):
┌───────────────────────┐
│ [logo] ElProof    [≡] │
│ Klien percaya karena  │
│ bisa melihat buktinya.│
│ sub                    │
│ [Coba gratis 14 hari] │
│ Portal Klien (HP) +    │
│ potongan Console di    │
│ belakangnya (miring)   │
│ pita bukti (gulir ↔ 2  │
│ baris, bukan marquee)  │
│ bento → 1 kolom kartu  │
│ pita navy + foto       │
│ harga (kartu)          │
│ FAQ · penutup · footer │
├───────────────────────┤
│ [Tanya WA] [Coba gratis]│ ← bilah bawah lengket, muncul setelah hero lewat
└───────────────────────┘
```

Halaman lain: Tentang (cerita pembuat + foto tim/lokasi bila ada), FAQ, Kontak, hukum ikut token baru tanpa ubah
struktur; Masuk/Daftar memakai panel navy dengan foto pernikahan dan segel emas.

## 5. State

- **Harga memuat:** kerangka kartu tanpa angka. **Galat/kosong:** "Tanya harga lewat WhatsApp" + tombol.
- **Foto pernikahan gagal dimuat:** latar navy + segel emas (sama seperti panel masuk sekarang).
- **JavaScript belum jalan / crawler:** semua isi ada di HTML prarender; mockup UI = HTML statis.
- **Kurangi gerakan:** segel langsung tampil, reveal mati.

## 6. Aksesibilitas

Kontras di §3 (emas tidak pernah jadi teks di latar terang kecuali #8A6424); fokus 2 px navy (di pita navy: emas
#E8D3AE); status selalu ikon + teks; foto ber-`alt` deskriptif, mockup UI ditandai `role="img"` + label ringkas agar
pembaca layar tidak membaca data contoh satu per satu; target sentuh ≥ 44 px; bilah bawah HP tidak menutupi isi
(padding bawah halaman); tab pratinjau tetap pola APG.

## 7. Keputusan yang perlu pemilik produk ambil sebelum dibangun

1. **Palet:** navy + emas hanya di situs publik dulu, atau seluruh aplikasi (konsol WO, Portal Klien, Platform)?
   ADR-0039 menetapkan satu palet untuk semuanya. Rekomendasi: seluruh aplikasi, agar logo dan produk satu rupa.
2. **Bukti sosial nyata:** ada WO yang bersedia dikutip (nama, usaha, foto) — mis. pelanggan yang sudah memakai?
   Tanpa itu, bagian kutipan dihilangkan, bukan dikarang.
3. **Foto:** boleh memakai foto Tampilan Login yang sekarang (perlu kepastian hak pakai), atau ada foto lain?
4. **Huruf display serif:** ya/tidak (rekomendasi: tidak).
5. **Fakta pita bukti** di §2a: mana yang benar dan boleh ditulis.

## 8. Rencana verifikasi setelah dibangun

Playwright: 1440×900 dan iPhone 13 (emulasi perangkat), terang; `prefers-reduced-motion`; konsol bebas galat dan
tanpa pelanggaran CSP; prarender tetap tanpa skrip inline; `npm run budget` (entry ≤ 165 KB) dan `budget:routes`
marketing-beranda; snapshot e2e marketing diperbarui setelah ditinjau; satu putaran kritik → perbaiki → screenshot ulang.

## Sumber (diakses 16 Nov 2026)

Situs yang diamati langsung (tangkapan layar hero + dua bagian berikutnya, 1440 px; lembar kontak di `riset/ref-*.webp`,
kondisi ElProof sekarang di `riset/elproof-sekarang-*.webp`):
- [linear] https://linear.app/ · [attio] https://attio.com/ · [stripe] https://stripe.com/ · [ramp] https://ramp.com/
- [mercury] https://mercury.com/ · [notion] https://www.notion.com/ · Cal.com https://cal.com/ · Superhuman https://superhuman.com/
- [honeybook] https://www.honeybook.com/ · [aisleplanner] https://www.aisleplanner.com/ · [dubsado] https://www.dubsado.com/ ·
  [planningpod] https://www.planningpod.com/
- [mekari] https://www.mekari.com/ · [majoo] https://majoo.id/

Riset dan panduan:
- [nng-trust] NN/g, "Trustworthiness in Web Design: 4 Credibility Factors" (2016) — https://www.nngroup.com/articles/trustworthy-design/
- [nng-price] NN/g, "B2B Websites Should Show Prices" (2013) — https://www.nngroup.com/articles/show-price/
- [nng-scrolljack] NN/g, "Scrolljacking 101" — https://www.nngroup.com/articles/scrolljacking-101/
- [wcag-233] W3C, Animation from Interactions — https://www.w3.org/WAI/WCAG22/Understanding/animation-from-interactions.html
- [screenshot-vs-illus] Ringkasan praktisi (tanpa studi terkontrol): https://saashero.net/content/b2b-saas-landing-page-examples/ ·
  https://getillustrations.com/blog/saas-illustration-styles-that-convert/
- [trust-early] Saas Hero, trust signals — https://www.saashero.net/uncategorized/landing-page-design-trust-signals/
- [logo-not-proof] https://www.pravinkumar.co/blog/why-customer-logos-are-not-proof-b2b-software-2026 ·
  https://www.pravinkumar.co/blog/should-a-saas-show-customer-count-on-its-homepage-2026
- [perspective] https://www.perspective.co/article/saas-landing-page-examples ·
  [vezadigital] https://vezadigital.com/post/best-saas-homepage-design-examples
- HoneyBook untuk wedding planner — https://www.honeybook.com/all-in-one-software-for-event-and-wedding-planners

Catatan: sebagian besar artikel "landing page terbaik" adalah blog vendor dan angkanya tidak dapat diverifikasi; dipakai
sebagai arah, bukan bukti. Bukti utama brief ini adalah pengamatan langsung atas situs di atas dan dua artikel NN/g.
