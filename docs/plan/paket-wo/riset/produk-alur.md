# Riset produk: alur & susunan layar Paket, Penawaran, Acara

Analis · 2026-11-19 · Bahan untuk `DESAIN.md` (PLAN.md §8 Riset desain). Fokus: **alur dan tata letak**, bukan warna.
Melengkapi, tidak mengulang: `docs/plan/revamp-wo-console/RISET.md`, `docs/plan/revamp-client-portal/RISET.md`, dan
sumber di PLAN.md §10 (HoneyBook "add services" dan "notify", Dubsado payment plans/contract, Planning Pod proposal
management, Tripleseat proposals, Perfect Venue capacity, Odoo create quotations, HubSpot manage quotes). Artikel yang sama
dibaca ulang hanya bila ada isi baru yang relevan untuk alur.

**Tanda bukti:** [V] halaman dibaca sendiri (curl/WebFetch, 2026-11-19) · [S] sekunder (ringkasan mesin pencari, forum,
situs ulasan) · [?] tidak terverifikasi / tafsiran. Tidak ada situs yang dimasuki dengan login. Tanggal artikel ditulis
bila halaman menampilkannya.

---

## 1. Per produk

### 1.1 HoneyBook (layanan kreatif & event, AS)
- **Alur:** layanan disimpan di menu **Services** (pengganti "packages") → smart file (proposal = blok Services + blok
  Invoice + blok Contract dalam satu dokumen) → kirim via email/tautan → klien memilih layanan, menandatangani, membayar →
  proyek dianggap *booked* saat kontrak ditandatangani atau pembayaran pertama masuk. [V] `hb-services`; booked: [S] `hb-booked`
- **Builder:** halaman dokumen berblok (tambah blok/halaman). Blok layanan punya mode **Select one / Select multiple /
  View only** dan "wajib pilih". Isi layanan = *sub-items* di bawah layanan; harga, qty, unit, pajak di "Advanced options".
  **Total**: ringkasan pilihan klien + total berjalan di **bilah navigasi** dokumen sisi klien (bisa dimatikan). Pilihan
  klien langsung mengisi blok Invoice; termin wajib berbasis persen bila ada layanan dinamis. [V] `hb-services`
- **Paket berisi item:** tidak ada paket berharga dengan isi tanpa harga secara bawaan; komunitas memakai **layanan dasar
  wajib (select one) + add-on terpisah**, dan item yang sudah termasuk dibuat **Rp0**. Keluhan nyata: bila paket juga
  dijual per item, invoice **menagih ganda** item yang sudah termasuk paket. [V] `hb-community-addons`
- **Tamu/acara:** tidak ada logika tamu atau tata letak; harga tidak berubah mengikuti jumlah tamu/jenis acara
  ("everything in a Smart File is static"). [V] `hb-community-dynamic`
- **Revisi & status:** **tidak ada versi**. File yang sudah dikirim diedit di tempat; tabel aturan menentukan kapan klien
  harus diberi tahu dan menyetujui. Klien melihat **banner "Accept changes"**; bila kontrak memuat rekap layanan/pembayaran,
  klien **tanda tangan ulang**. Setelah ada pembayaran, blok layanan **tidak bisa diedit**. Setelah ada tanda tangan,
  kanvas terkunci; "Edit file" membuka kunci dan memaksa tanda tangan ulang. [V] `hb-notify`, `hb-edit-client`
- **Tampilan klien:** tautan/email, opsi kode verifikasi untuk aksi sensitif (tanda tangan, bayar), opsi **view only**;
  setelah dikirim klien bisa membuka lagi tetapi tidak mengubah. [V] `hb-client-access`
- **Keluhan:** proposal tidak bisa menyesuaikan diri dengan tamu/tata letak/jenis acara (permintaan fitur terbuka) [V];
  otomasi pipeline memindah tahap terlalu dini [S] `hb-booked`.

### 1.2 Dubsado (layanan kreatif, AS)
- **Alur:** **Invoicing ▸ Packages** (paket dulu) → **Forms ▸ Form templates ▸ Proposal** atau langsung dari tab **Forms**
  di dalam proyek → klien memilih paket, menandatangani kontrak, membayar → saat dikirim (submit) **invoice dibuat otomatis**
  dari pilihan klien (meskipun invoice tidak ditampilkan). [V] `dub-quote`, `dub-proposal`
- **Paket = kelompok baris berharga yang dipakai ulang.** Editor paket berupa **tabel baris inline** (nama, qty, harga,
  pajak, deskripsi berpoin, kategori pendapatan), urutkan dengan seret, **simpan otomatis** ("All changes saved"). Saat
  dipasang ke invoice/proposal, baris **disalin**: mengubah paket di katalog tidak mengubah dokumen klien. Add-on dibuat
  sebagai paket terpisah, karena satu paket = "ambil semua atau tidak sama sekali". [V] `dub-packages`
- **Builder:** kanvas seret-lepas; elemen Package (pra-pilih ya/tidak, qty min/maks); Container berkolom untuk
  membandingkan paket berdampingan; panel Settings (kontrak, invoice, boleh pilih banyak paket, diskon, payment plan). [V] `dub-proposal`
- **Tamu/acara/kapasitas:** tidak ada. **Revisi:** tidak dibahas di bantuan yang dibaca [?].
- **Keluhan:** kurva belajar berat saat menyiapkan alur [S] `rev-capterra`.

### 1.3 Planning Pod (planner & venue, AS)
- **Alur:** proposal dibuat dari **menu utama Proposals** atau dari **dasbor Event/Lead** → isian awal (klien, nomor otomatis,
  nama, tanggal, perlu tanda tangan) → **Proposal Dashboard** (satu halaman berseksi: pengantar/syarat, baris, kategori,
  biaya tambahan, lampiran, galeri, pesan klien) → Email (log aktivitas) / unduh PDF → tanda tangan → **Create Invoice**
  menyalin semua baris, kategori, biaya tambahan. [V] `pp-new`, `pp-overview`, `pp-email`, `pp-invoice`
- **Baris:** jumlah baris tak terbatas, **total di bawah**, **subtotal per kategori**; opsi sembunyikan total (untuk
  proposal berisi beberapa pilihan) dan sembunyikan harga per baris sehingga hanya subtotal kategori tampil. Baris ditambah
  lewat **modal** ("Add Line Item"), aksi per baris di menu "Actions". Kolom tampilan/PDF bisa disembunyikan. [V] `pp-line`, `pp-new`
- **Paket:** *line item template* untuk paket/menu/sewaan; diimpor lalu diedit di proposal. Paket F&B ditautkan dari event
  atau diimpor dari templat. [V] `pp-lit`, `pp-fb`
- **Tamu:** headcount **Estimated / Actual / Minimum** di dasbor event; baris proposal/invoice bisa dihitung "by Headcount"
  sehingga **harga ikut berubah** saat headcount diubah (juga dari BEO). [V] `pp-headcount`
- **Revisi:** proposal yang sudah ditandatangani **terkunci**; perubahan lewat **Change Order** bernama; tanda tangan
  "Original Version" dan PDF tiap versi bertanda tangan **disimpan**. [V] `pp-change`
- **Daftar:** "stat circles" di atas (total diajukan, ditandatangani, belum) + tabel bisa diurutkan. [V] `pp-overview`
- **Multi-acara per proposal:** tidak ditemukan bukti [?]. **Keluhan:** antarmuka terasa kuno, kurva belajar [S] `rev-capterra`.

### 1.4 Tripleseat (venue/restoran/hotel, AS)
- **Alur:** lead → **Convert** (tanggal, jam, perkiraan tamu ikut terbawa) → **Booking** sebagai wadah berisi **banyak
  event**, masing-masing dengan tamu, ruang, jam sendiri; ruang + jumlah tamu wajib sebelum simpan. [S] `ts-convert`, `ts-multi`
- **Events Grid** (hotel, 2023-03-14): buat "kerangka" booking dulu, lalu tambah/ubah semua event di **satu grid** dengan
  toolbar (tambah, ubah massal, salin, pasang set dokumen); tampilan **Timeline** menunjukkan semua event dalam rentang
  booking. [V] `ts-grid`
- **Dokumen:** proposal, BEO, kontrak, invoice **dibuat dari record event dan ikut berubah saat detail berubah** ("same
  current version"); tata letak dokumen dibedakan untuk **tamu** (harga, gratuity) dan **internal** (headcount, menu,
  susunan ruang, tanpa angka keuangan). Kalkulasi minimum F&B, pajak, biaya ada di dalam dokumen. [V] `ts-docs`
- **Kapasitas:** tidak ditemukan bukti peringatan kapasitas [?] (halaman bantuan Tripleseat tidak bisa diakses dari jaringan ini).

### 1.5 Perfect Venue (venue, AS)
- **Alur:** event dibuat → tab **Proposal** di dalam event → menu item dari **Settings ▸ Menu** (harga per jam/orang/item,
  pilihan & add-on) → Preview tampilan klien → klien menandatangani & membayar di portal. [V] `pv-menu`, `pv-sessions`
- **Sesi acara (Enterprise):** "Add Session" di tab proposal: nama (mis. "Lunch"), jam, **ruang** (ketersediaan dicek
  otomatis), catatan; setiap menu item dipilih masuk **sesi mana**; di BEO tiap sesi satu halaman. [V] `pv-sessions`
- **Beberapa proposal dalam satu event (Enterprise):** tiap opsi punya menu & total sendiri; **detail event (tanggal,
  jam, kontak) dipakai bersama** dan tidak bisa beda per opsi; klien membandingkan nama + total, memilih satu, menandatangani;
  opsi lain hilang dari tampilan klien setelah ditandatangani. [V] `pv-multi`
- **Potongan:** di **Edit di samping Totals** (nominal/persen) atau per item. [V] `pv-discount`
- **Kapasitas:** artikel pemasaran: ruang yang sama ±280 berdiri vs ±150–170 duduk; "selalu tanya format sebelum menyebut
  jumlah tamu" (2026). [V] `pv-capacity` (sudah dipakai PLAN #20; dicatat sebagai penguat)

### 1.6 Aisle Planner (wedding planner, AS)
- **Alur:** **Templates ▸ Product** (katalog: nama, deskripsi, gambar, qty min/maks, harga) → **Templates ▸ Quote** →
  kirim dari **Lead Record** atau menu **Quotes** (Add Quote / Add From Template), bisa dikaitkan ke lead → tombol ekspor
  emas → klien memilih opsi → quote **diterima** → **Create Invoice** berisi opsi yang dipilih. Proyek dibuat dari lead
  (isian terisi dari lead), lead jadi "Booked". [V] `ap-catalog` (2025-12-08); proyek: [S] `ap-lead`
- **Struktur quote (paling dekat dengan paket WO):** seksi **Included Items** (isi paket, klien tidak bisa mengubah) ·
  **Choose One** (pilih salah satu) · **A la carte** (add-on). Seksi bisa diulang. Saat menyusun, **total hanya memuat
  Included Items**; total berubah saat klien memilih. Item yang ditambah langsung di quote tidak masuk katalog. [V] `ap-catalog`
- **Revisi & masa berlaku:** quote yang **diterima menjadi final, tidak bisa diedit**; ubah lewat **salinan bernama baru**
  ("Revised Quote") atau ubah di invoice. Tanggal kedaluwarsa bisa diubah kapan saja, **juga setelah lewat**; klien yang
  membuka dokumen kedaluwarsa melihat pesan untuk menghubungi WO. [V] `ap-expiry`

### 1.7 Event Temple (hotel & venue)
- **Dua cara membuat proposal** (Masterclass, 2026-09-30): **Quick mode** dari halaman Proposals (pilih templat → kontak,
  nama booking, tanggal, tahap pipeline, **kedaluwarsa bawaan 14 hari** → booking dibuat otomatis) atau **dari Booking**
  (semua event, setup style, baris terisi dari booking). [V] `et-master`
- **Templat** per segmen (pernikahan vs rapat), 3 kelompok elemen: Konten (tanpa batas), **Pricing & Revenue (satu)**,
  **Totals & Summary (satu, di akhir)**; ringkasan biaya bisa diganti nama ("Estimated Total"), pos tertentu dikeluarkan dari
  total (mis. kamar yang dibayar tamu sendiri). Add-on dengan qty min/maks. [V] `et-master`
- **Status:** Draft · Sent · Accepted; pelacakan dibuka/berapa kali/terakhir dilihat; dasbor: **Kedaluwarsa minggu ini ·
  Menunggu jawaban · Baru diterima**; tombol Terima/Tolak bisa dimatikan untuk proposal harga awal; pengingat otomatis
  (mis. H+7 kirim, H-7 kedaluwarsa); booking *Lost* → kedaluwarsa disamakan supaya klien tidak bisa membuka. [V] `et-master`
- **Baris dikelompokkan per event** (seksi bisa dilipat, "No Event" di atas, edit inline, urutan seret = urutan di portal
  tamu, dokumen, dan proposal). [V] `et-lines`

### 1.8 17hats (usaha kecil, AS)
- Katalog **Products & Services**; baris quote diisi dengan mengetik nama item tersimpan; "Save this item for use later";
  menghapus katalog tidak mengubah quote lama; quote + kontrak + invoice bisa dikirim sebagai satu "combined document".
  Alur terima → invoice tidak ditemukan. [S] `17h-products`, `17h-combined`

### 1.9 Pembanding quote umum
- **Odoo Sales 18:** daftar Quotations (filter "My Quotations") → **New** → form: pelanggan, templat, kedaluwarsa (bawaan
  dari pengaturan) → tab **Order Lines** (tabel inline; **Add a section** = judul tebal, **Add a note** = teks miring,
  seret untuk menyusun; di bawah baris tombol kupon/diskon) · **Optional Products** · Other Info. Tombol atas: **Send by
  Email** (PDF terlampir) · **Confirm** (staf mengonfirmasi → status **Sales Order**) · **Preview** (tampilan portal pelanggan,
  banner "Back to edit mode") · Cancel. Opsi **Lock Confirmed Sales**. Klien bisa **Sign & Pay** di portal. [V] `odoo-create`, `odoo-template`, `odoo-sign`
- **HubSpot Quotes:** index dengan **tab: Semua · Segera kedaluwarsa · Menunggu diterima · Menunggu persetujuan**; aksi:
  **Recall & edit** (selama belum disetujui/diterima/dibayar), **Clone** (salin baris & syarat), **Mark signed** (tanda tangan
  manual), Convert to invoice, Void (tetap tersimpan, tak bisa diedit, tautan mati), Archive. Satu deal punya **primary
  quote** yang mengisi nilai deal. Quote yang sudah diterima tidak bisa di-void/hapus. [V] `hs-manage`
- **PandaDoc (quote builder):** **bundle** produk dari katalog: isi bundle tampil **di bawah nama bundle**, bisa dilipat
  (**Hide products / Show products**); harga bundle = **jumlah tetap** atau **subtotal isi**; penerima hanya bisa mengubah qty
  atau memilih **seluruh** bundle, bukan isinya; isi bisa dijadikan opsional (pra-pilih). [V] `pd-bundle`; opsi qty/opsional
  untuk penerima [S] `pd-optional`
- **Proposify:** baris opsional berkotak centang di sisi klien, total langsung berubah; perilaku di HP tidak didokumentasikan. [S] `prop-optional`

### 1.10 Indonesia
- **Gatera (aplikasi WO, halaman produk):** alur 6 tahap calon klien → **Penawaran** → MOU → acara → hari-H → pembayaran.
  Penawaran **disusun dari paket atau baris sendiri**; **baris dikelompokkan per paket**; tiap baris bisa memuat **beberapa
  pilihan vendor** dan tanda **"Estimasi"**; DP dan termin terisi dari templat; **diskon dibatasi per peran** (di atas batas
  perlu persetujuan sebelum bisa dikirim). Harga terkirim tersimpan apa adanya, **revisi = versi baru**. Klien membuka tautan
  tanpa akun dan bisa **menerima/menolak** serta memilih vendor bila ada pilihan. Penawaran diterima → **acara + anggaran**
  tanpa ketik ulang; acara bisa **ditahan sampai DP dikonfirmasi** keuangan. MOU dibuat saat menjadi acara, per versi. [V] `gatera`, `gatera-fitur`
- **Inframe Linea (aplikasi WO):** modul "Manajemen Paket & Price List": paket all-in & custom, **rincian include/exclude
  vendor**. Klaim pemasaran, tidak ada detail layar. [V] `inframe`
- **Contoh surat penawaran WO (PDF di Bridestory, dibuat 2022):** kop WO → judul "Penawaran paket pernikahan <venue>" →
  salam → **nama paket + "Estimasi harga" satu angka** → isi per kategori (fasilitas hotel & katering 200 pax dengan dua
  pilihan menu, dekorasi, dokumentasi, MUA & busana, hiburan, MC, WO) dengan **nama vendor alternatif per kategori**
  ("A / B / C") dan **tanpa harga per isi** → seksi **BONUS** → penutup → **Term condition**: termin 25/50/25, DP hangus bila
  batal, tambahan/ganti vendor dikenai biaya vendor. [V] `id-cordela`
- **Price list WO 2025 (Bridestory):** dua tingkat **300 pax / 500 pax**, harga ditentukan **venue** (daftar per wilayah),
  isi per kategori tanpa harga, **tambahan katering Rp65.000/pax**, bonus gubukan 100/150 porsi, add-on album; dekorasi & MC
  disebut untuk akad dan resepsi. [V] `id-mahakarya` (melengkapi Alissha di PLAN §10)

---

## 2. Pola yang berulang → implikasi untuk ElProof

| # | Pola (produk) | Implikasi untuk ElProof |
|---|---|---|
| 1 | **Paket katalog disalin ke dokumen**; mengubah katalog tidak mengubah dokumen yang sudah ada (Dubsado, Aisle Planner, Planning Pod, 17hats) | Sesuai "salinan aturan paket di baris" (PLAN §10). Tampilkan di penyunting bahwa baris adalah salinan + aksi eksplisit "Perbarui dari katalog" (seperti "Perbarui dari direktori" di venue), bukan sinkron diam-diam |
| 2 | **Bundle berharga tetap, isi tampil di bawah nama paket tanpa harga, bisa dilipat** (PandaDoc custom amount; Aisle Planner Included Items; price list & surat penawaran WO Indonesia; Inframe include/exclude) | Baris Paket = satu harga + daftar isi (teks) yang bisa dilipat; di PDF/portal isi tampil sebagai butir di bawah paket. Pertimbangkan isi "Tidak termasuk" (pola include/exclude) — **keputusan produk** |
| 3 | **Bonus ditulis sebagai seksi terpisah**, dan cara Rp0 di produk tanpa konsep bonus memicu tagihan ganda (surat Indonesia "BONUS"; keluhan HoneyBook) | `is_bonus` tampil sebagai kelompok "Bonus" sendiri di penyunting, PDF, dan portal; bonus tidak ikut subtotal |
| 4 | **Dokumen berseksi satu halaman** dengan urutan tetap, bukan wizard (Planning Pod Proposal Dashboard, Odoo tab, Event Temple elemen templat, surat Indonesia) | Penyunting Penawaran satu halaman: Kepala → Acara & sesi → Pemasok → Rincian → Termin → Syarat & ketentuan, dengan navigasi seksi; wizard hanya untuk "Jadikan proyek" (isian pelengkap) |
| 5 | **Baris diedit inline di tabel**, detail jarang dipakai di modal/panel (Dubsado, Odoo, Event Temple inline; Planning Pod modal) | Tabel rincian inline untuk jumlah × harga; deskripsi panjang, pemasok, dan bonus di panel samping/drawer per baris |
| 6 | **Total di akhir dokumen + subtotal per kelompok**; sisi klien punya total berjalan (Planning Pod, Event Temple "Totals & Summary" satu di akhir, HoneyBook bilah ringkasan) | Kartu ringkasan (Paket, Tambahan tamu, Item, Potongan, PPN, Nilai kontrak) **lengket** di sisi kanan penyunting desktop dan di bawah pada HP; subtotal per kelompok di tabel |
| 7 | **Jumlah tamu menggerakkan harga baris** (Planning Pod headcount Estimated/Actual/Minimum → baris "by Headcount") | Baris "Tambahan tamu" dibuat sistem, baca-saja, menampilkan rumusnya ("650 tamu − 500 termasuk = 150 → 3 × kelipatan 50 × Rp…") dan tautan ke acara penentunya (acara terbesar) |
| 8 | **Banyak acara dalam satu wadah**, tampil sebagai grid/daftar baris dengan tamu, ruang, jam per acara; baris harga bisa dikelompokkan per acara (Tripleseat Booking + Events Grid, Event Temple grouped by event, Perfect Venue sessions) | Seksi Acara = daftar baris (Akad, Resepsi, …) yang bisa ditambah/diurutkan; sesi bersarang di bawah acara. Rincian **dikelompokkan per paket** (pola Indonesia/Gatera), bukan per acara, karena paket WO mencakup beberapa acara |
| 9 | **Sesi = bagian acara dengan jam + ruang sendiri**, ketersediaan dicek saat diisi (Perfect Venue sessions) | Sesi: label, jam, tamu; kapasitas venue × tata letak ditampilkan **di samping isian tamu** ("maks. 600 berdiri") dan ditolak saat simpan dengan pesan berangka (PLAN #20) |
| 10 | **Revisi dibekukan saat dikirim / disetujui**, ubah lewat versi/salinan baru, PDF versi lama tetap bisa diunduh (Planning Pod change order, Aisle Planner, Gatera, HubSpot recall/clone) | Mendukung PLAN §5.2. Tampilkan riwayat revisi sebagai daftar (Revisi n · tanggal · total · oleh) dengan Unduh PDF; tandai jelas "Salinan kerja berbeda dari Revisi 2" sebelum Terima |
| 11 | **Daftar penawaran dibuka dengan saringan yang menuntut tindakan**: kedaluwarsa minggu ini, menunggu jawaban, baru diterima (Event Temple dasbor, HubSpot tab, Planning Pod stat circles) | Daftar Penawaran: tab status (Draf · Diterima · Ditolak · Kedaluwarsa · Digantikan) + saringan "Kedaluwarsa ≤ 7 hari"; Beranda/"Perlu ditindaklanjuti" memuat penawaran yang segera kedaluwarsa |
| 12 | **Masa berlaku dengan bawaan dan bisa diperpanjang setelah lewat** (Event Temple 14 hari, Odoo validity, Aisle Planner) | `valid_until` terisi bawaan dari pengaturan WO; aksi "Perpanjang" pada yang kedaluwarsa (sesuai §5.3) |
| 13 | **Buat dari menu utama atau dari konteks**, dan membuat dari menu utama otomatis membuat wadah (Event Temple quick mode, Planning Pod, Aisle Planner, Dubsado) | ElProof belum punya modul lead: Penawaran dibuat dari menu Penawaran dengan kepala berisi nama pengantin & kontak (quick mode); tidak perlu memaksa membuat klien dulu |
| 14 | **Konversi menyalin semua isi dan meminta sedikit isian** (Planning Pod Create Invoice, Aisle Planner, Tripleseat convert, Gatera penawaran → acara + anggaran, DP hold) | "Jadikan proyek" = dialog/halaman pendek (tanggal pasti, PIC, tanda jadi opsional) yang menampilkan ringkasan apa yang akan disalin; selesai → langsung ke proyek baru |
| 15 | **Dokumen klien ≠ dokumen internal** dari data yang sama (Tripleseat guest proposal vs BEO; Gatera "biaya vendor dan margin tidak pernah tampil") | Satu sumber, dua tampilan: PDF/portal tanpa biaya rekanan & tanpa harga isi paket; konsol menampilkan margin hanya untuk izin margin (PLAN §6) |

---

## 3. Pola yang saling bertentangan antarproduk

| Topik | Kubu A | Kubu B | Catatan untuk ElProof |
|---|---|---|---|
| **Revisi** | Edit di tempat, satu dokumen hidup: HoneyBook (banner "Accept changes"), Tripleseat (dokumen ikut berubah otomatis) | Bekukan & buat versi/salinan: Planning Pod, Aisle Planner, Gatera, HubSpot recall/clone | PLAN memilih B (pas untuk PDF yang dikirim lewat WhatsApp: klien memegang berkas statis). Ambil dari A: penanda jelas bila salinan kerja sudah berbeda dari revisi terakhir yang dikirim |
| **Siapa menerima** | Klien menerima/menandatangani sendiri lewat tautan: HoneyBook, Dubsado, Perfect Venue, Odoo, Event Temple, Gatera | Staf menandai: HubSpot "Mark signed", Odoo "Confirm" | PLAN memilih B (WO menandai, catat saluran + lampiran). Mayoritas produk memakai A; e-sign/terima sendiri sudah dicatat "dipertimbangkan, tidak diambil" (PLAN §10) |
| **Klien memilih isi** | Proposal interaktif: klien memilih paket/add-on/qty dan total berubah (HoneyBook, Dubsado, Aisle Planner, PandaDoc, Proposify, Perfect Venue) | Penawaran tetap yang sudah disusun dari wawancara: surat penawaran & price list WO Indonesia, Odoo, Planning Pod | ElProof = B (sesuai #8: disusun dari wawancara di luar aplikasi). Jangan meniru builder interaktif AS |
| **Beberapa opsi** | Banyak opsi dalam satu dokumen/event untuk dibandingkan klien (Perfect Venue multiple proposals, Planning Pod sembunyikan total, Dubsado paket berdampingan) | Satu dokumen = satu opsi; opsi lain = salinan (Aisle Planner "Revised Quote", HubSpot Clone + primary quote) | ElProof = B ("Salin jadi penawaran baru" → Digantikan). Bila WO sering menawarkan 300 vs 500 pax sekaligus, ini **keputusan produk** yang perlu dikonfirmasi |
| **Kelompok baris** | Per acara (Event Temple, Perfect Venue sessions) | Per paket/kategori (Gatera, Planning Pod kategori, surat Indonesia) | Pilih per paket (isi paket mencakup beberapa acara); acara cukup disebut di baris paket ("Mencakup: Akad, Resepsi") |
| **Tamu → harga** | Harga ikut headcount dan berubah otomatis saat headcount diubah (Planning Pod) | Harga statis, tamu hanya informasi (HoneyBook, Dubsado) | ElProof = A dengan rumus kelipatan (PLAN #5); karena revisi dibekukan, perubahan otomatis hanya terjadi di salinan kerja |
| **Kapasitas** | Ketersediaan/ruang dicek saat isi sesi (Perfect Venue) dan validasi yang memblokir (Jestor, sekunder) | Tidak ada pengecekan (HoneyBook, Dubsado, Planning Pod) | ElProof = blokir keras per sesi (PLAN #20); kebanyakan produk tidak punya, jadi teks galat harus menjelaskan angka dan jalan keluarnya (ganti tata letak, tambah sesi, ganti venue) |

---

## 4. Celah & batas bukti
- Halaman bantuan Tripleseat (Zendesk) dan sebagian Proposify tidak bisa diakses dari jaringan ini → klaim Tripleseat
  tentang convert/multi-event bertanda [S]; Events Grid dan halaman dokumen [V].
- Tidak ditemukan bukti publik: Planning Pod multi-acara per proposal, peringatan kapasitas di Tripleseat/Event Temple,
  alur terima → invoice di 17hats, tampilan HP proposal Proposify/PandaDoc.
- Bukti Indonesia untuk aplikasi (Gatera, Inframe Linea) berasal dari halaman pemasaran sendiri, bukan ulasan independen.
- Tidak ada tangkapan layar yang diunduh; susunan layar dibaca dari teks bantuan resmi.

---

## 5. Sumber (diakses 2026-11-19)

| Kode | Sumber |
|---|---|
| hb-services | HoneyBook Help, Add, manage, and use services in smart files — https://help.honeybook.com/en/articles/5643712-add-services-to-a-file |
| hb-notify | HoneyBook Help, When to notify clients of file changes — https://help.honeybook.com/en/articles/6252653 |
| hb-edit-client | HoneyBook Help, Edit a smart file as your client (status, kunci setelah tanda tangan) — https://help.honeybook.com/en/articles/9247056 |
| hb-client-access | HoneyBook Help, How clients access and submit smart files — https://help.honeybook.com/en/articles/9768365-how-clients-access-and-submit-smart-files |
| hb-community-dynamic | HoneyBook Community, Dynamic proposals — https://community.honeybook.com/talk-product-7/dynamic-proposals-5911 |
| hb-community-addons | HoneyBook Community, Default package with add-ons — https://community.honeybook.com/talk-product-7/can-i-make-a-default-package-with-an-option-for-add-ons-4223 |
| hb-booked | HoneyBook Community, Projects marked as booked — https://community.honeybook.com/business-talk-46/projects-automatically-marked-as-booked-when-a-consult-fee-is-paid-5802 ; Lead conversion — https://community.honeybook.com/talk-product-7/lead-conversion-to-client-4998 [S] |
| dub-quote | Dubsado Help, Build a proposal or quote (2026-09-15) — https://help.dubsado.com/en/articles/12859419-build-a-proposal-or-quote |
| dub-proposal | Dubsado Help, Build a proposal — https://help.dubsado.com/en/articles/14426389-build-a-proposal |
| dub-packages | Dubsado Help, How to create packages — https://help.dubsado.com/en/articles/15000577-how-to-create-packages |
| pp-overview | Planning Pod Help, Overview – How do proposals work — https://planningpod.com/help-center/overview-how-do-proposals-work |
| pp-new | Planning Pod Help, Adding a new proposal — https://planningpod.com/help-center/adding-a-new-proposal |
| pp-line | Planning Pod Help, Adding a proposal line item — https://planningpod.com/help-center/adding-a-proposal-line-item |
| pp-lit | Planning Pod Help, Line item templates — https://planningpod.com/help-center/creating/using-line-item-templates |
| pp-fb | Planning Pod Help, Adding a F&B package into a proposal — https://planningpod.com/help-center/adding-a-fb-package-into-a-proposal |
| pp-headcount | Planning Pod Help, Managing/using headcounts for events — https://planningpod.com/help-center/managing/using-headcounts-for-events |
| pp-change | Planning Pod Help, Creating a change order for a signed proposal — https://planningpod.com/help-center/creating-a-change-order-for-a-signed-proposal |
| pp-email | Planning Pod Help, Emailing proposals to clients — https://planningpod.com/help-center/emailing-proposals-to-clients |
| pp-invoice | Planning Pod Help, Creating an invoice from a proposal — https://planningpod.com/help-center/creating-an-invoice-from-a-proposal |
| ts-grid | Tripleseat, Feature release: The Events Grid (2023-03-14) — https://tripleseat.com/product-updates/feature-release-the-events-grid/ |
| ts-docs | Tripleseat, Proposals, contracts & BEO — https://tripleseat.com/features/proposals-contracts-beo/ |
| ts-convert | Tripleseat Support, How to create an event — https://support.tripleseat.com/hc/en-us/articles/1500010839401-How-to-create-an-event [S] |
| ts-multi | Tripleseat Support, Creating events in a booking — https://support.tripleseat.com/hc/en-us/articles/18604795748247-Creating-Events-in-a-Booking [S] |
| pv-multi | Perfect Venue Help, Multiple proposals — https://help.perfectvenue.com/knowledge/multiple-proposals-enterprise-feature |
| pv-sessions | Perfect Venue Help, Event sessions — https://help.perfectvenue.com/knowledge/event-sessions-enterprise-plan-only |
| pv-menu | Perfect Venue Help, All things menu — https://help.perfectvenue.com/knowledge/all-things-menu |
| pv-discount | Perfect Venue Help, Applying discounts to proposals — https://help.perfectvenue.com/knowledge/applying-discounts-to-proposals |
| pv-capacity | Perfect Venue, Venue capacity system — https://www.perfectvenue.com/post/venue-capacity-system |
| ap-catalog | Aisle Planner, Introducing product catalogs and quotes (2025-12-08) — https://aisleplanner.com/introducing-product-catalogs-and-quotes/ |
| ap-expiry | Aisle Planner Help, Expiration dates to quotes, proposals, contracts — https://help.aisleplanner.com/en/articles/3130688-adding-expiration-dates-to-quotes-proposals-and-contracts |
| ap-lead | Aisle Planner Help, Creating a project from a lead — https://help.aisleplanner.com/en/articles/1901865-creating-a-project-from-a-lead [S] |
| et-master | Event Temple Help, Proposals masterclass (2026-09-30) — https://support.eventtemple.com/en/articles/15217273-proposals-masterclass |
| et-lines | Event Temple Help, New line items experience — https://support.eventtemple.com/en/articles/12004028-new-line-items-experience |
| 17h-products | 17hats Help, Products & services page — https://help.17hats.com/en/articles/849333-money-matters-products-services-page [S] |
| 17h-combined | 17hats Help, Combined documents — https://help.17hats.com/articles/924358-combined-documents [S] |
| odoo-create | Odoo 18 docs, Create quotations — https://www.odoo.com/documentation/18.0/applications/sales/sales/sales_quotations/create_quotations.html |
| odoo-template | Odoo 18 docs, Quotation templates — https://www.odoo.com/documentation/18.0/applications/sales/sales/sales_quotations/quote_template.html ; Optional products — https://www.odoo.com/documentation/18.0/applications/sales/sales/sales_quotations/optional_products.html |
| odoo-sign | Odoo 18 docs, Online signatures — https://www.odoo.com/documentation/18.0/applications/sales/sales/sales_quotations/get_signature_to_validate.html |
| hs-manage | HubSpot KB, Manage quotes — https://knowledge.hubspot.com/quotes/manage-quotes |
| pd-bundle | PandaDoc Help, Quote builder: product bundles — https://support.pandadoc.com/en/articles/9714716-quote-builder-block-working-with-product-bundles |
| pd-optional | PandaDoc Help, Pricing table: optional items & editable quantity — https://support.pandadoc.com/en/articles/9714705-pricing-table-optional-items-and-editable-quantity [S] |
| prop-optional | Proposify Support, Interactive pricing tables — https://support.proposify.com/hc/en-us/articles/37892793603099-Interactive-Pricing-Tables [S] |
| gatera | Gatera, beranda — https://gatera.id/ |
| gatera-fitur | Gatera, fitur — https://gatera.id/fitur |
| inframe | Inframe Linea, beranda — https://www.inframelinea.com/ |
| id-cordela | Contoh surat penawaran paket pernikahan (PDF, dibuat 2022-03) — https://alexandra.bridestory.com/raw/upload/assets/penawaran-wedding-package-cordela-HpTOLfN_g.pdf |
| id-mahakarya | Price list wedding 2025 (PDF) — https://alexandra.bridestory.com/raw/upload/assets/pricelist-2025-mahakarya-ocSXd4Xxt.pdf |
| rev-capterra | Capterra/G2/Software Advice, ulasan Dubsado & Planning Pod — https://capterra.com/p/206389/Dubsado/reviews/ ; https://www.capterra.com/p/125947/Planning-Pod/reviews [S] |
| jestor | Jestor, Manage venue capacity and room layouts (validasi blokir) — https://blog.jestor.com/venue-capacity-room-layouts/ [S] |

Pengerjaan: seluruhnya oleh Analis (tanpa subagent).
