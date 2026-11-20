# DESAIN: Paket WO, acara & rincian, portal, dan Penawaran

Status: **disetujui pemilik produk** (19 Nov 2026, dikutip: "Sepakat"). Dokumen ini menjadi dasar rupa, alur, dan perilaku semua layar Tahap 1–5 (PLAN.md §8). Perubahan alur atau rupa di tengah tahap memperbarui dokumen ini lebih dulu.

Dasar:
- [PLAN.md](PLAN.md): model data, aturan bisnis, izin, tahapan.
- [riset/produk-alur.md](riset/produk-alur.md): cara 15 produk sejenis menyusun alur dan layar (HoneyBook, Dubsado, Planning Pod, Tripleseat, Perfect Venue, Aisle Planner, Event Temple, 17hats, Odoo, HubSpot, PandaDoc, Gatera, surat penawaran WO Indonesia).
- [riset/pola-ui.md](riset/pola-ui.md): pola interaksi dan tata letak dari NN/g, GOV.UK, Carbon, Atlassian, Shopify, Salesforce CPQ, Stripe, W3C APG, dan WCAG 2.2.
- `knowledge/DESIGN.md`: token, kit shadcn, kepadatan, wadah, keadaan, dan glosarium. Dokumen ini **tidak** mengubah warna, huruf, atau komponen dasar.

Isi dokumen ini adalah pengalaman pakai: alur per peran, struktur menu, tata letak tiap layar, interaksi, keadaan, teks layar, dan PDF.

## 1. Kasus

| | |
|---|---|
| **Subjek** | Penawaran untuk calon klien, katalog paket WO, dan rincian kesepakatan (acara, vendor & venue, harga) yang dibawa ke proyek dan portal |
| **Pengguna** | Sales dan Owner (menyusun penawaran, laptop), Wedding Planner (menjalankan proyek, laptop dan HP di lapangan), Owner/Admin (katalog, margin), Pengantin dan Keluarga (portal di HP Android) |
| **Pertanyaan yang harus terjawab cepat** | Sales: "penawaran mana yang perlu ditindaklanjuti, dan berapa totalnya kalau tamunya segini?" WP: "apa yang disepakati dan apakah masih muat di venue?" Owner: "untungnya berapa?" Pengantin: "saya dapat apa, berapa, dan apa yang harus saya bayar sendiri?" |
| **Ukuran mudah dipakai** | Sama dengan `knowledge/DESIGN.md` §1: tugas berhasil, jumlah langkah, salah langkah, skor kemudahan per tugas (SEQ 1–7) |

## 2. Temuan → keputusan

Kode bukti mengacu ke berkas riset: **P** = produk-alur.md §2/§3, **U** = pola-ui.md bagian n.

| # | Temuan | Bukti | Keputusan desain |
|---|---|---|---|
| D1 | Penawaran disusun sebagai **satu dokumen berseksi** dengan urutan tetap, bukan wizard. Wizard hanya untuk konversi | P4, P14 | Halaman Penawaran = satu halaman: Info → Acara → Vendor & venue → Rincian → Termin → Syarat & ketentuan → Riwayat. "Jadikan proyek" = halaman pelengkap pendek |
| D2 | Total harus terlihat saat menyusun; produk menaruh ringkasan di akhir dan di bilah menempel | P6, U1.5 | Desktop ≥ 1280 px: **panel Ringkasan biaya menempel di kanan**. Lebih sempit: ringkasan di bawah judul dan total ikut di bilah simpan |
| D3 | Baris harga diedit langsung di tabel; isian jarang dipakai dibuka di panel samping non-modal. Tabel HTML biasa, bukan grid/treegrid | P5, U1.1 | Rincian: jumlah, satuan, harga, uraian diedit di baris. Vendor/venue, acara, catatan di **drawer "Detail baris"** |
| D4 | Paket = satu harga dengan isi tanpa harga yang bisa dilipat; bonus ditulis sebagai kelompok sendiri | P2, P3, U1.2 | Baris paket bisa dibuka/tutup; isi bertuliskan "Termasuk paket" (bukan Rp0). **Bonus** dan **Potongan** tampil sebagai kelompok sendiri di penyunting, PDF, dan portal |
| D5 | Jumlah tamu menggerakkan harga; angka terhitung harus menjelaskan asalnya | P7, U1.3 | "Tambahan tamu" dan PPN baca-saja dengan rumus tertulis di baris ("650 − 500 termasuk = 150 → 3 × 50 tamu × Rp4.000.000") dan toggletip "Cara menghitung" |
| D6 | Data uang tidak disimpan otomatis; simpan eksplisit dengan bilah menempel dan peringatan saat meninggalkan halaman | U1.6 | Rincian dan termin disimpan dengan **bilah simpan menempel**. Acara, vendor & venue, dan info disimpan per drawer/dialog |
| D7 | Total sementara saat mengetik tidak boleh punya rumus kedua di web | U1.5 | Angka sementara diambil dari server (`POST …/lines/preview`, jeda 500 ms, tanpa menyimpan). Rumus tetap satu, di `offers/domain` |
| D8 | Kapasitas: hitung sisa saat mengetik tanpa memblokir, tolak saat simpan dengan angka dan jalan keluar | U2.2, P9 | Sisa kapasitas di samping isian tamu ("50 lagi" / "50 di atas kapasitas"). Saat simpan: pesan berangka + **[Bagi jadi 2 sesi] [Ganti venue] [Ubah ke duduk (muat 800)]** |
| D9 | Revisi dibekukan saat dikirim; versi lama tetap bisa diunduh; tanda jelas bila salinan kerja sudah berubah | P10, U4.1–4.3 | Banner **"Ada perubahan yang belum dikirim ke klien (sejak Revisi 2)"**; riwayat revisi terbaru di atas, dengan Bandingkan dan Unduh PDF |
| D10 | Satu aksi utama di kepala dokumen yang berganti menurut status | U4.1 | Tabel aksi per status di §6.3 |
| D11 | Daftar dibuka dengan saringan yang menuntut tindakan; masa berlaku punya bawaan dan bisa diperpanjang | P11, P12 | Tab status + saringan "Segera kedaluwarsa (≤ 7 hari)". Masa berlaku bawaan diatur di Profil usaha (bawaan 14 hari) |
| D12 | Membuat penawaran dari menu utama langsung membuat wadahnya, tanpa wajib membuat klien dulu | P13 | "Penawaran baru" = dialog 3 isian (nama pasangan, WhatsApp, PIC) lalu langsung ke halaman penawaran |
| D13 | Konversi menyalin semua isi dengan sedikit isian dan halaman periksa sebelum simpan | P14, U4.5 | "Jadikan proyek" = halaman penuh: tanggal pasti per acara, tim, tanda jadi, ringkasan apa yang disalin, [Buat proyek] |
| D14 | Dokumen klien dan internal berasal dari data yang sama, berbeda tampilan | P15, U3 | Biaya rekanan dan margin **tidak dirender** tanpa izin (bukan nonaktif atau "Rp•••"). Keluarga: tanpa kolom nominal sama sekali |
| D15 | "Dibayar oleh" yang terkunci harus terbaca beserta alasannya | U3 | Baca-saja dengan ikon kunci + "Terkunci: sudah ada 2 pembayaran. Hapus pembayaran itu untuk mengubah" |
| D16 | Susun ulang harus bisa tanpa menyeret | U1.7 | Menu ⋯: Pindah ke atas/bawah/paling atas/paling bawah; pegangan seret hanya tambahan di desktop; HP hanya menu |
| D17 | Rincian di HP: total dulu, daftar dua kolom, isi paket dibuka "Lihat semua" tanpa gulir otomatis | U5 | Portal › Layanan memakai pola ini |
| D18 | Surat penawaran WO Indonesia: kop, paket + estimasi, isi per kategori tanpa harga, bonus, S&K, tanda tangan; orang tua sering hanya membaca halaman 1 | U6, P12 | PDF halaman 1 = ringkasan seluruh kesepakatan; halaman 2 dst. = rincian, vendor dibayar klien, S&K, tanda tangan |
| D19 | Data kerja yang dipakai saat menyusun (bukan pengaturan jarang) dikelompokkan sendiri | `revamp-wo-console/RISET.md` #6 | Menu **Katalog**: Paket, Jenis acara, Jadwal bawaan, Kategori vendor (pengganti grup "Templat"). Paket **tidak** di Pengaturan |
| D20 | Detail proyek sudah diringkas ke 6 tab; ≤ 7 tab yang muat satu baris | `revamp-wo-console/RISET.md` #4 | Satu tab baru saja: **Acara & rincian**. "Vendor & venue" tetap, isinya diperbarui |
| D21 | WO menyebut "vendor" dan "venue" (glosarium mempertahankannya) | `knowledge/DESIGN.md` §7 | Di layar tidak ada kata "pemasok". Kumpulan = **Vendor & venue**; butir = vendor atau venue. "Kerja sama pemasok" hanya istilah teknis di PLAN |
| D22 | Klien sering diberi dua pilihan (300 vs 500 tamu) | P §3, keputusan #26 | **Buat alternatif**: penawaran kedua tertaut, keduanya hidup; saat satu diterima, yang lain Digantikan |

**Elemen khas fitur ini:** angka yang bisa dipercaya karena **selalu menjelaskan dirinya**. Setiap angka terhitung (tambahan tamu, PPN, sisa kapasitas, margin) menampilkan asal-usulnya dalam satu baris bahasa biasa. Sisanya tenang, mengikuti `knowledge/DESIGN.md` §5.

## 3. Struktur menu dan navigasi

### 3.1 WO Console

```
Beranda
Penawaran            ← baru (izin quotation.view)
Proyek
Kalender
Keuangan
Klien
Direktori ▸ Vendor
            Venue
Katalog   ▸ Paket            ← baru (package.manage)
            Jenis acara      ← baru (package.manage)
            Jadwal bawaan    (pindahan dari Templat)
            Kategori vendor  (pindahan dari Templat)
Pengaturan ▸ Profil usaha   (+ kode usaha, PKP, masa berlaku & syarat penawaran)
             Tim
             Peran dan akses
             Langganan
```

- Penawaran diletakkan sebelum Proyek karena urutan kerja WO: penawaran → proyek.
- Tombol di topbar berubah dari "Proyek baru" menjadi **"Baru ▾"** berisi Penawaran dan Proyek, sesuai izin. Bila pengguna hanya punya satu izin, tombolnya langsung "Penawaran baru" atau "Proyek baru".
- Pencarian ⌘K ikut mencari penawaran (nomor dan nama pasangan).
- Beranda › Perlu tindakan menambah dua butir:
  - "Penawaran MO-PNW-261119-001 kedaluwarsa 2 hari lagi";
  - "Bukti bayar fotografer dari Citra menunggu diperiksa".

URL berbahasa Indonesia, mengikuti pola yang ada:

| Layar | URL |
|---|---|
| Penawaran | `/penawaran`, `/penawaran/:id`, `/penawaran/:id/jadikan-proyek` |
| Paket | `/paket`, `/paket/baru`, `/paket/:id` |
| Jenis acara | `/jenis-acara` |
| Tab proyek | `/proyek/:id/acara-rincian` |
| Tab portal | `/portal/layanan` |

### 3.2 Detail proyek (7 tab, satu baris)

```
Ringkasan · Acara & rincian · Jadwal · Vendor & venue · Keuangan · Kendala · Dokumen
```

- **Acara & rincian** (baru): acara dan sesi, lalu rincian harga, dengan komponen yang sama seperti di penawaran.
- **Vendor & venue**: kerja sama vendor/venue (dibayar oleh, progres, tahapan), termasuk tagihan vendor yang dibayar klien.
- **Keuangan**: tiga buku:
  - Klien;
  - Vendor & venue (dibayar WO, hanya untuk izin margin);
  - Dibayar klien langsung.
- Kepala proyek: fakta "Paket Premium" diganti nama paket dari rincian (atau "Tanpa paket"). Fakta venue diganti lokasi per acara ("Akad: Masjid Al-Ikhlas · Resepsi: Gedung Melati"), yang terpotong satu baris dengan tooltip.
- Proyek dari penawaran menampilkan tautan kecil di kepala: "Dari MO-PNW-261119-001 Revisi 2 · Lihat kesepakatan awal".

### 3.3 Portal klien (5 tab bawah, tetap)

```
Beranda · Jadwal · Pembayaran · Layanan · Lainnya
```

| Tab | Isi |
|---|---|
| **Beranda** | Hero menampilkan acara utama. Bila acara > 1, ada baris kecil "2 acara: Akad 13 Mar · Resepsi 14 Mar" |
| **Jadwal** | Bagian **Acara** di atas (kartu per acara: tanggal, venue + alamat + peta, sesi dan tamu), lalu jadwal tahapan seperti sekarang |
| **Pembayaran** (izin keuangan) | Kartu nilai kontrak + tautan "Lihat rincian biaya", termin, riwayat, lalu bagian **Tagihan vendor** untuk vendor yang Anda bayar langsung |
| **Layanan** (pengganti "Vendor") | Bagian **Paket & rincian** (isi paket, item, bonus, potongan; nominal hanya untuk izin keuangan), lalu **Progres vendor** seperti tab Vendor sekarang (nama disembunyikan sesuai aturan) |

Keluarga tidak punya tab Pembayaran. Di Layanan, Keluarga melihat isi paket dan item tanpa satu pun angka.

## 4. Alur penggunaan

Notasi: `[Tombol]`, `‹layar›`, `◇` titik keputusan, `✕` jalur galat.

### A1. Sales menyusun dan mengirim penawaran

```
Menu Penawaran / topbar [Baru ▾ › Penawaran]
  → dialog "Penawaran baru": Nama pasangan*, WhatsApp, PIC* (bawaan: saya)   [Buat penawaran]
  → ‹Penawaran MO-PNW-261119-001, Draf›  (nomor terbentuk, masa berlaku = hari ini + 14)
  → Acara: [Tambah acara] → drawer: jenis, tanggal (boleh kosong), venue dari Direktori, tata letak, sesi + tamu   [Simpan acara]
       ◇ venue dipilih → otomatis tercatat di Vendor & venue (dibayar WO bawaan), tidak perlu ditambah dua kali
       ✕ tamu > kapasitas → pesan berangka + [Bagi jadi 2 sesi] [Ganti venue] [Ubah tata letak]
  → Rincian: [Tambah ▾ › Paket dari katalog] → pilih paket
       → baris paket + isi paket + Tambahan tamu (dihitung dari tamu acara)
       ◇ paket punya venue bawaan dan acara belum punya venue → tawaran "Pakai Gedung Melati untuk Resepsi?" [Pakai venue]
       ✕ tamu termasuk paket > kapasitas venue → pesan berangka
     [Tambah ▾ › Item] souvenir 300 pcs × Rp12.000; [Tambah ▾ › Bonus]; [Tambah ▾ › Potongan]
     ‹panel Ringkasan biaya berubah sambil mengetik›   [Simpan rincian]
  → Vendor & venue: periksa yang otomatis masuk; [Tambah vendor] untuk yang dibayar klien langsung (fotografer)
  → Termin (bawaan 30/40/30) dan Syarat & ketentuan (bawaan dari Profil usaha) → ubah bila perlu
  → [Pratinjau PDF] (watermark DRAF) → ◇ sudah benar
  → [Buat revisi untuk klien] → Revisi 1 → [Bagikan PDF]
       HP: lembar bagikan perangkat berisi berkas PDF (WhatsApp ada di dalamnya)
       Desktop: PDF terunduh + WhatsApp Web terbuka ke nomor klien dengan pesan siap kirim
```

**Klien minta ubah:** Sales mengubah rincian → banner "Ada perubahan yang belum dikirim ke klien (sejak Revisi 1)" → [Buat revisi untuk klien] → Revisi 2 → [Bagikan PDF].

**Dua pilihan:** dari menu ⋯, [Buat alternatif] → penawaran baru `MO-PNW-261119-002` dengan isi tersalin dan tautan "Alternatif dari …001" → ubah jumlah tamu atau paket → revisi → bagikan.

### A2. Penawaran diterima → jadi proyek

```
‹Penawaran, Draf, sama dengan Revisi 2› [Tandai diterima]
  → dialog: ringkasan "Revisi 2 · Nilai kontrak Rp152.400.000 · 3 termin"; revisi (bawaan terakhir), tanggal, saluran
            (WhatsApp / tatap muka / email / lainnya), lampiran opsional (foto PDF bertanda tangan)
       ◇ ada alternatif hidup → "MO-PNW-261119-002 akan menjadi Digantikan"
       ◇ sudah kedaluwarsa → isian "Perpanjang sampai" muncul di dialog yang sama (dua aksi tercatat: diperpanjang, diterima)
       ✕ ada perubahan yang belum dikirim → dialog berisi penjelasan + [Buat revisi untuk klien] (bukan tombol Terima nonaktif)
     [Tandai diterima]
  → status Diterima; aksi utama berganti [Jadikan proyek]
  → ‹Jadikan proyek›: tanggal pasti per acara, tanggal mulai persiapan, PIC, PIC Sales, tanda jadi (opsional)
       → ringkasan: nilai kontrak, termin yang akan dibuat, 2 acara · 5 vendor & venue · 9 baris, kapasitas dicek ulang
       ✕ kapasitas tidak lolos (venue diubah di direktori) → pesan di ringkasan + tautan ke acara
       ✕ kuota masa coba habis → pesan langganan + [Lihat langganan]
     [Buat proyek] → ‹Proyek Citra & Bima, tab Ringkasan› dengan banner "Dibuat dari MO-PNW-261119-001 Revisi 2"
```

### A3. Owner menyiapkan fitur pertama kali

```
Penawaran (kosong) → keadaan kosong: "Susun penawaran dari paket Anda. Mulai dengan menyiapkan paket." [Siapkan paket] [Buat penawaran tanpa paket]
  → ‹Katalog › Paket› (kosong): [Buat paket] / [Isi dengan contoh] (mengisi formulir, belum menyimpan)
  → ‹Paket baru›: nama, harga, acara yang dicakup, tamu termasuk, kelipatan + harga per kelipatan, venue bawaan (opsional),
     isi paket (daftar), perkiraan biaya per isi (hanya izin margin)
     ‹kotak "Contoh hitung": untuk [620] tamu → Rp120.000.000 + 3 × Rp4.000.000 = Rp132.000.000›   [Simpan paket]
  → ‹Profil usaha › Penawaran›: kode usaha (MO, dengan contoh nomor), PKP, masa berlaku bawaan, syarat & ketentuan bawaan
  → ‹Venue›: setiap venue wajib punya kapasitas berdiri dan/atau duduk; tiga harga
```

Jenis acara bawaan (enam) sudah ada. Owner hanya membuka ‹Jenis acara› bila ingin menambah, misalnya "Mapacci".

### A4. Wedding Planner: tamu bertambah, venue tidak muat

```
‹Proyek › Acara & rincian› kartu Resepsi [Ubah acara] → drawer
  → Sesi 1 tamu 550 → 650; di samping isian: "50 di atas kapasitas (600 berdiri)"
  → [Simpan acara]
  ✕ ditolak: "Gedung Melati muat 600 orang berdiri, Sesi 1 Resepsi 650."
     ◇ [Bagi jadi 2 sesi] → sesi 1 dan 2 masing-masing 325, jam diisi WP → simpan
     ◇ [Ubah ke duduk]: hanya tampil bila kapasitas tata letak lain cukup
     ◇ [Ganti venue] → pilih venue lain yang muat (daftar menampilkan kapasitas)
  → tersimpan → pesan status: "Tambahan tamu berubah dari Rp4.000.000 menjadi Rp12.000.000. Nilai kontrak Rp160.400.000."
  → kartu Termin: banner "Jadwal termin (Rp152.400.000) tidak sama dengan nilai kontrak (Rp160.400.000)" [Hitung ulang termin]
```

### A5. Progres vendor, biaya rekanan, margin

```
WP: ‹Vendor & venue› baris Katering Rasa → drawer: status, PIC, tahapan, bukti (seperti sekarang)
Owner: drawer yang sama juga menampilkan "Biaya rekanan" dan "Pembayaran ke rekanan" [Catat pembayaran]
Owner: ‹Acara & rincian› panel Ringkasan biaya: Nilai kontrak · Biaya rekanan · Margin
       ◇ ada vendor dibayar WO tanpa biaya → "Margin belum lengkap (2 vendor)" → tautan ke daftar tersaring
```

### A6. Proyek tanpa penawaran

```
[Baru ▾ › Proyek] → ‹Proyek baru› (2 langkah, seperti sekarang):
  1. Pasangan, PIC, tanggal persiapan
  2. Acara (minimal satu: jenis + tanggal)
→ ‹Proyek› → tab Acara & rincian (keadaan kosong rincian: [Tambah paket] [Tambah item])
```

Termin dibuat setelah rincian tersimpan. Kartu termin kosong menampilkan "Susun rincian dulu agar termin bisa dihitung" [Buka rincian].

### A7. Pengantin di HP

```
Beranda (hero acara utama, "Yang perlu Anda lakukan": "Bayar fotografer Rp5.000.000 sebelum 1 Des")
  → Layanan: nilai kontrak, Paket Resepsi 500 tamu + "Lihat 12 isi paket", tambahan tamu dengan penjelasan awam, item, bonus, potongan
  → Pembayaran › Tagihan vendor: Fotografer Lensa · DP Rp5.000.000 · jatuh tempo 1 Des   [Kirim bukti bayar]
     → sheet: tagihan, nominal, tanggal, foto/berkas   [Kirim bukti]
     → "Menunggu diperiksa wedding organizer Anda"
WO: Beranda › Perlu tindakan → drawer bukti → [Tandai sesuai] / [Tolak dengan alasan]
```

### A8. Keluarga di HP

```
Beranda → Jadwal (acara, venue, sesi) → Layanan (isi paket, item, bonus; tanpa angka) → Lainnya (kendala yang dibagikan, dokumen)
```

Keluarga tidak melihat Pembayaran, nilai, maupun tagihan vendor.

## 5. Komponen bersama (modul web `rincian`)

### 5.1 Acara & sesi

**Tampilan baca** (penawaran dan proyek):

```
Acara                                                         [Tambah acara]
┌──────────────────────────────────────────────────────────────────────────┐
│ Akad · Jumat, 13 Mar 2027 · 08.00–10.00                          [Ubah] ⋯ │
│ Masjid Al-Ikhlas · Duduk (kapasitas 150)                                 │
│ 1 sesi · 120 tamu                                                         │
├──────────────────────────────────────────────────────────────────────────┤
│ Resepsi · Sabtu, 14 Mar 2027                                     [Ubah] ⋯ │
│ Gedung Melati · Berdiri (kapasitas 600)                                  │
│ Sesi 1 11.00–13.00  550 tamu · Sesi 2 18.00–21.00  550 tamu              │
├──────────────────────────────────────────────────────────────────────────┤
│ Lamaran · tanggal belum ditentukan                               [Ubah] ⋯ │
│ ⓘ Belum ada venue: tamu belum dicek ke kapasitas                          │
└──────────────────────────────────────────────────────────────────────────┘
```

Satu kartu berisi daftar dengan pemisah, bukan kartu per acara. Urutannya menurut tanggal, lalu acara tanpa tanggal di bawah. Menu ⋯ berisi Pindah ke atas/bawah, Duplikat acara, dan Hapus acara.

**Drawer "Ubah acara"** (lebar 560 px):

```
Ubah acara: Resepsi                                                   ✕
Jenis acara      [Resepsi ▾]
Tanggal          [14/03/2027]   Mulai [  ]  Selesai [  ]
Venue            [Gedung Melati ▾]   Berdiri 600 · Duduk 400
Tata letak       (•) Berdiri — 600   ( ) Duduk — 400
Sesi
  Nama       Mulai    Selesai   Tamu        Sisa kapasitas
  [Sesi 1]   [11.00]  [13.00]   [ 550 ]     50 lagi
  [Sesi 2]   [18.00]  [21.00]   [ 650 ]     ⚠ 50 di atas kapasitas
  [Tambah sesi]                              Total tamu 1.200
──────────────────────────────────────────────────────────────────────
                                         [Batal]  [Simpan acara]
```

Aturan perilaku:
- **Sisa kapasitas** dihitung di layar dari angka yang sudah tampil (kapasitas venue dikurangi tamu sesi). Ini perbandingan tampilan, bukan rumus harga, jadi tidak melanggar D7.
- Ketikan di atas batas tidak dipotong. Penolakan resmi datang dari server saat simpan.
- **Tolak saat simpan:** pesan di isian sesi dan ringkasan galat di atas drawer. Di bawah pesan ada tombol jalan keluar:
  - [Bagi jadi 2 sesi] membagi tamu sama rata ke sesi baru;
  - [Ubah ke duduk — muat 800] hanya tampil bila kapasitas tata letak lain cukup;
  - [Ganti venue] membuka pemilih venue yang menampilkan kapasitas dan menandai venue yang tidak muat.
- **Venue tanpa angka** untuk tata letak itu: "Isi kapasitas duduk Gedung Melati" dengan [Ubah venue], yang membuka formulir venue di drawer kedua. Isian acara tidak hilang.
- **Memilih venue** dari direktori otomatis menambahkannya ke Vendor & venue (dibayar WO, bisa diubah di sana). Mengganti venue tidak menghapus yang lama bila sudah punya pembayaran. Venue lama tetap di Vendor & venue dengan catatan "Tidak dipakai acara mana pun".
- Isian tamu: `inputmode="numeric"`, tanpa tombol naik-turun.

### 5.2 Vendor & venue

```
Vendor & venue                                         [Saring ▾]  [Tambah vendor]
Nama                      Dibayar oleh     Tampil ke klien   Biaya rekanan*   Progres     ⋯
Gedung Melati  Venue      WO               Selalu            Rp40.000.000     —           ⋯
Katering Rasa  Katering   WO               Tidak             Rp85.000.000     1/3 tahap   ⋯
Fotografer Lensa  Foto    Klien langsung   Selalu            —                0/2 tahap   ⋯
                                                    * hanya terlihat oleh pemegang izin margin
```

- Kolom Biaya rekanan **tidak ada sama sekali** tanpa izin margin.
- Di penawaran, kolom Progres tidak ada.
- Klik baris membuka **drawer detail**:
  - vendor atau venue dari direktori;
  - **Dibayar oleh** (radio WO / Klien langsung);
  - Tampilkan nama ke klien (sakelar, hanya untuk vendor dibayar WO; selain itu tertulis "Selalu tampil ke klien");
  - biaya rekanan (berizin), dengan tombol "Isi dari harga rekanan acuan (Resepsi: Rp80.000.000)";
  - catatan.
- Di proyek, drawer ditambah tahapan, bukti, dan pembayaran:
  - pembayaran ke rekanan untuk vendor dibayar WO (berizin);
  - tagihan dan bukti dari klien untuk vendor dibayar klien.
- **Terkunci:** setelah ada pembayaran atau bukti, "Dibayar oleh" tampil baca-saja: 🔒 "Klien langsung. Terkunci karena sudah ada 2 pembayaran. Hapus pembayaran itu untuk mengubah."

### 5.3 Rincian

**Desktop** (penawaran; di proyek sama, ditambah tautan "Kesepakatan awal"):

```
Rincian                                                                   [Tambah ▾]
⠿  Uraian                                Acara      Jumlah × Harga satuan        Nominal     ⋯
⠿  ▾ Paket Resepsi 500 tamu              Resepsi                            Rp120.000.000    ⋯
       Termasuk: Akad, Resepsi · 500 tamu
       Katering buffet 450 porsi + 6 gubukan                                Termasuk paket
       Dekorasi pelaminan & area foto                                       Termasuk paket
       … 10 isi lainnya  [Lihat semua]
       Tambahan tamu  ⓘ                                                      Rp12.000.000
       650 tamu − 500 termasuk = 150 → 3 × 50 tamu × Rp4.000.000
⠿  Souvenir kipas                         Resepsi   [300] [pcs] × [Rp12.000]   Rp3.600.000    ⋯
Bonus
⠿  Undangan digital                        —                               Bonus             ⋯
Potongan
⠿  Potongan harga paket                                                   −Rp5.000.000       ⋯
─────────────────────────────────────────────────────────────────────────────────────────────
                                                        Subtotal          Rp130.600.000
                                                        PPN 12% × 11/12   Rp14.366.000
                                                        Nilai kontrak     Rp144.966.000
```

Bila ada perubahan, muncul **bilah simpan** menempel di bawah:

```
Perubahan belum disimpan · Nilai kontrak Rp144.966.000 (+Rp3.600.000)   [Batalkan perubahan] [Simpan rincian]
```

- **[Tambah ▾]** berisi Paket dari katalog (combobox pencarian), Item, Bonus, dan Potongan. Fokus pindah ke isian pertama baris baru.
- **Paket** dari katalog adalah salinan:
  - tertulis kecil "Disalin dari katalog 19 Nov 2026";
  - bila katalog sudah berubah, muncul tautan **[Perbarui dari katalog]** yang membuka dialog berisi perbandingan harga dan isi lama → baru. Tidak pernah sinkron diam-diam.
- **Isi paket** tidak bisa diberi harga dan hanya berpindah di dalam paketnya. **Tambahan tamu** tidak bisa dipindah.
- **Toggletip ⓘ "Cara menghitung"** pada Tambahan tamu memuat:
  - acara penentunya ("Resepsi, 650 tamu: acara terbesar yang dicakup paket"), dengan tautan ke acara itu;
  - aturan paket ("500 termasuk, tambahan per 50 tamu Rp4.000.000");
  - "Mau harga lain? Tambahkan Potongan."
- **Detail baris** (ikon ⋯ › Detail, atau klik uraian dua kali) membuka drawer non-modal:
  - uraian panjang;
  - acara;
  - vendor/venue yang memasok (hanya yang dibayar WO);
  - kategori.
- **Galat:** "Potongan melebihi subtotal (Rp95.000.000). Kurangi potongan." Ringkasan galat tampil di atas tabel bila ada ≥ 3 galat atau ada galat di luar layar. Setiap butir ringkasan menaut ke isiannya.
- **Konflik (412):** isian Anda dipertahankan, dan banner menjelaskan "Rincian sudah diubah Rina pada 14.03. Perubahan Anda belum disimpan." dengan [Lihat perbedaan] [Muat versi terbaru]. Tidak ada tombol "Timpa".
- **Pengumuman pembaca layar** (`role="status"`, jeda 500 ms):
  - "Nilai kontrak sekarang Rp144.966.000";
  - "Souvenir kipas dipindah ke posisi 2 dari 5".

**HP (375 px):** setiap baris menjadi kartu (uraian, acara, nominal). Ketuk kartu membuka sheet layar penuh untuk mengedit. Susun ulang hanya lewat menu, dan bilah simpan tetap menempel.

### 5.4 Panel Ringkasan biaya (desktop ≥ 1280 px, menempel di kanan)

```
┌ Ringkasan biaya ──────────────┐
│ Paket            Rp120.000.000│
│ Tambahan tamu     Rp12.000.000│
│ Item               Rp3.600.000│
│ Potongan          −Rp5.000.000│
│ PPN               Rp14.366.000│
│ Nilai kontrak                 │
│ Rp144.966.000                 │  ← 24 px, tabular
│ ───────────────────────────── │
│ Biaya rekanan*   Rp125.000.000│  * hanya izin margin
│ Margin*            Rp5.600.000│
│ Margin belum lengkap (1 vendor)│
└───────────────────────────────┘
```

- Di penawaran, panel ini juga memuat ringkasan Revisi terakhir: "Revisi 2 · 18 Nov · Rp141.366.000 · Lihat riwayat".
- Di bawah 1280 px, panel pindah ke atas isi sebagai satu baris ringkas ("Nilai kontrak Rp144.966.000 · Lihat ringkasan"), dan total juga ada di bilah simpan.

## 6. Layar

### 6.1 Daftar Penawaran

```
Penawaran                                                          [Penawaran baru]
[Semua] [Draf 6] [Kedaluwarsa 1] [Diterima 2] [Ditolak] [Digantikan]
[Cari nomor atau pasangan]  [PIC ▾]  [☐ Segera kedaluwarsa (≤ 7 hari)]
Nomor               Pasangan         Acara utama      Nilai kontrak    Status        Berlaku s.d.    PIC
MO-PNW-261119-001   Citra & Bima     14 Mar 2027      Rp144.966.000    ◷ Draf · Rev 2  30 Nov (H-11)  Rina
  └ 1 alternatif
MO-PNW-261112-004   Dewi & Arga      2 Feb 2027       Rp98.000.000     ⊘ Kedaluwarsa   12 Nov         Sari
MO-PNW-261101-002   Sari & Doni      —                Rp120.000.000    ✓ Diterima      → Proyek        Rina
```

- Urutan bawaan: terakhir diubah.
- "Berlaku s.d." berwarna peringatan hanya bila ≤ 7 hari.
- Di HP, setiap baris menjadi kartu berisi nomor, pasangan, nilai, dan status + H-n.
- Klik baris membuka penawaran. Alternatif tampil sebagai baris anak yang bisa dibuka.

### 6.2 Halaman Penawaran

**Desktop ≥ 1280 px:**

```
Penawaran › MO-PNW-261119-001
Citra & Bima   ◷ Draf · Revisi 2 · berlaku s.d. 30 Nov 2026 (H-11)        [Pratinjau PDF] [Buat revisi untuk klien] ⋯
⚠ Ada perubahan yang belum dikirim ke klien (sejak Revisi 2).
┌──────────────────────────────────────────────────────────┬──────────────────────────────┐
│ Info  · Acara · Vendor & venue · Rincian · Termin · S&K · Riwayat   (lompat ke bagian, menempel) │
│                                                          │ ┌ Ringkasan biaya ──────────┐ │
│ Info                                          [Ubah]     │ │ … (5.4)                    │ │
│ Pasangan Citra & Bima · WA 0812… · PIC Rina · Catatan …  │ │ Revisi 2 · 18 Nov · Lihat  │ │
│                                                          │ └────────────────────────────┘ │
│ Acara                         (5.1)                      │                                │
│ Vendor & venue                (5.2)                      │                                │
│ Rincian                       (5.3)                      │                                │
│ Termin                                        [Ubah]     │                                │
│ DP 30% · saat diterima · Rp43.489.800 …                  │                                │
│ Syarat & ketentuan                            [Ubah]     │                                │
│ Riwayat                                                  │                                │
│ Revisi 2 · 18 Nov 14.10 · Rina · Rp141.366.000 · +1 item [Unduh PDF] [Bandingkan dengan Revisi 1] │
│ Revisi 1 · 15 Nov 09.30 · Rina · Rp136.000.000           [Unduh PDF]                       │
│ Aktivitas: dibuat 15 Nov · dibagikan via WhatsApp 15 Nov · …                                 │
└──────────────────────────────────────────────────────────┴──────────────────────────────┘
```

- Menu ⋯ berisi Buat alternatif, Salin jadi penawaran baru, Tolak, dan Hapus draf (hanya draf tanpa revisi).
- **HP:** bagian bertumpuk. Navigasi bagian menjadi daftar lompat di atas, dan aksi utama ada di bilah bawah.
- **Baca-saja** (Diterima, Ditolak, Digantikan, sudah jadi proyek): semua tombol ubah hilang. Banner menjelaskan keadaannya:
  - "Diterima 20 Nov lewat WhatsApp · Revisi 2";
  - "Sudah dijadikan proyek: Citra & Bima →";
  - "Digantikan oleh MO-PNW-261119-002 →".

### 6.3 Aksi per keadaan

| Keadaan | Aksi utama | Aksi lain |
|---|---|---|
| Draf, belum ada revisi atau ada perubahan sejak revisi terakhir | **Buat revisi untuk klien** | Pratinjau PDF · Buat alternatif · Tolak |
| Draf, sama dengan Revisi n | **Bagikan PDF** | Tandai diterima · Unduh PDF · Buat alternatif · Tolak |
| Kedaluwarsa | **Perpanjang masa berlaku** | Tandai diterima (dengan perpanjangan) · Salin |
| Diterima, belum jadi proyek | **Jadikan proyek** | Batalkan terima · Unduh PDF |
| Ditolak | **Buka lagi** | Salin jadi penawaran baru |
| Digantikan / sudah jadi proyek | — (baca-saja, tautan ke penggantinya atau ke proyek) | Unduh PDF |

Lencana status selalu memakai ikon + teks: ◷ Draf · ⊘ Kedaluwarsa · ✓ Diterima · ✕ Ditolak · ⇢ Digantikan.

### 6.4 Bandingkan revisi (drawer lebar)

```
Revisi 1 → Revisi 2
Rincian
  + Ditambah   Souvenir kipas 300 pcs × Rp12.000            Rp3.600.000
  ~ Diubah     Potongan harga paket   Rp3.000.000 → Rp5.000.000
  − Dihapus    Photobooth                                    Rp2.500.000
  8 baris tidak berubah  [Tampilkan]
Acara
  ~ Diubah     Resepsi Sesi 2 tamu   450 → 550
Nilai kontrak  Rp136.000.000 → Rp141.366.000
```

### 6.5 Dialog "Tandai diterima" dan halaman "Jadikan proyek"

```
Tandai penawaran diterima                                        ✕
Revisi 2 · Nilai kontrak Rp141.366.000 · 3 termin
Revisi yang diterima   [Revisi 2 (terbaru) ▾]
Tanggal diterima       [20/11/2026]
Lewat                  ( ) WhatsApp ( ) Tatap muka ( ) Email ( ) Lainnya
Lampiran (opsional)    [Unggah foto PDF bertanda tangan]
ⓘ MO-PNW-261119-002 (alternatif) akan menjadi Digantikan.
                                        [Batal]  [Tandai diterima]
```

```
Penawaran › MO-PNW-261119-001 › Jadikan proyek
1. Tanggal acara      Akad [13/03/2027]   Resepsi [14/03/2027]   Lamaran [wajib diisi]
2. Persiapan & tim    Mulai persiapan [ ]   PIC [Rina ▾]   PIC Sales [Rina ▾]
3. Tanda jadi         ( ) Belum ada  (•) Sudah diterima: [Rp10.000.000] [20/11/2026] [Transfer ▾]
4. Periksa sebelum membuat
   Nilai kontrak Rp141.366.000 · Termin: DP 30% Rp42.409.800 (tanda jadi Rp10.000.000 tercatat) · Termin 2 … · Pelunasan …
   Disalin: 3 acara · 5 vendor & venue · 9 baris rincian · syarat & ketentuan
   ✓ Kapasitas semua sesi muat
                                                    [Batal]  [Buat proyek]
```

### 6.6 Katalog › Paket

```
Paket                                                                    [Buat paket]
[Cari nama paket]  [Semua status ▾]                                       [Kepadatan]
Paket                          Acara           Harga          Tamu termasuk        Status
Paket Resepsi 500 tamu         Akad, Resepsi   Rp120.000.000  500                  Aktif
  5 isi paket · Gedung Melati                                 +Rp4 jt per 50 tamu
Paket Akad Intim               Akad            Rp25.000.000   100                  Aktif
```

Kolom "Dipakai" (jumlah penawaran) ditambahkan di Tahap 5, bersama Penawaran. Harga dan tamu termasuk bisa diurutkan; saringan status dan pencarian berjalan di server.

Formulir paket (halaman penuh, dua kolom pada ≥ 1280 px; kanan = pratinjau):

```
Paket baru
Nama*                [Paket Resepsi 500 tamu]
Harga*               [Rp120.000.000]
Acara yang dicakup*  [☑ Akad] [☑ Resepsi] [☐ Lamaran] …
Tamu
  Tamu termasuk      [500]      (kosongkan bila harga tidak bergantung tamu)
  Tambahan per       [50] tamu   seharga [Rp4.000.000]
Venue bawaan         [Gedung Melati ▾]  Berdiri 600 · Duduk 400   (opsional)
Isi paket
  1 Isi paket 1* [Katering buffet 450 porsi + 6 gubukan]                       ⋯
    Vendor [Katering Rasa ▾]  Kategori [Katering ▾]  Perkiraan biaya* [ ]
  2 Isi paket 2* [Dekorasi pelaminan & area foto]                              ⋯
    Vendor [Tanpa vendor ▾]   Kategori [Dekorasi ▾]  Perkiraan biaya* [ ]
  [Tambah isi]                 Perkiraan biaya Rp90.000.000 · selisih dengan harga Rp30.000.000
                                                    [Batal]  [Simpan paket]

┌ Pratinjau ──────────────────────────────┐
│ Contoh hitung untuk [620] tamu            │
│ Rp120.000.000 + 3 × Rp4.000.000           │
│ = Rp132.000.000                           │
│ Seperti ini klien melihatnya:             │
│ Paket Resepsi 500 tamu    Rp120.000.000   │
│ • Katering buffet 450 porsi + 6 gubukan   │
│ • Dekorasi pelaminan & area foto          │
└───────────────────────────────────────────┘
```

- \* Perkiraan biaya hanya tampil bagi pemegang izin margin ("Hanya terlihat oleh tim"), dan boleh dikosongkan. Vendor opsional; memilih vendor menyalin kategorinya bila kategori belum diisi.
- Isi paket disusun ulang lewat menu ⋯ (Pindah ke atas, Pindah ke bawah, Hapus isi), tanpa pegangan seret, sehingga sama di desktop, HP, dan papan ketik (D16). Paling banyak 50 isi.
- Contoh hitung dihitung server dengan jeda 500 ms. Paket tanpa tamu termasuk menampilkan "Harga tetap Rp…, berapa pun jumlah tamunya".
- Di bawah 1280 px pratinjau berada di bawah formulir, dan [Batal] [Simpan paket] menempel di bawah layar sepanjang formulir.
- Venue bawaan menampilkan kapasitasnya; tamu termasuk di atas kapasitas terbesar venue diberi peringatan sebelum disimpan dan ditolak server dengan pesan yang menyebut kedua angka.
- Paket yang dirujuk penawaran tidak bisa dihapus permanen. Menu ⋯ menawarkan "Nonaktifkan" dengan penjelasan. Sampai Penawaran ada (Tahap 5), Hapus permanen tetap tersedia dengan dialog yang menawarkan "Nonaktifkan saja".
- Keadaan kosong: "Paket adalah produk yang Anda tawarkan ke klien: harga, acara yang dicakup, jumlah tamu, dan isinya." [Buat paket] [Isi dengan contoh]. "Isi dengan contoh" hanya mengisi formulir dan tidak menyimpan apa pun.

### 6.7 Katalog › Jenis acara

Daftar sederhana: nama (jenis bawaan berlabel "Bawaan"), dipakai di berapa paket (penawaran menyusul di Tahap 5), aktif/nonaktif, dan [Tambah jenis acara] (dialog satu isian). Menu baris: Ganti nama, Nonaktifkan/Aktifkan, Hapus permanen. Enam jenis bawaan bisa diganti nama atau dinonaktifkan, tetapi tidak bisa dihapus bila sudah dipakai: dialog hapus untuk jenis yang dipakai paket menawarkan "Nonaktifkan" saja.

### 6.8 Pengaturan › Profil usaha › bagian "Penawaran"

```
Penawaran
Kode usaha            [MO]     Contoh nomor: MO-PNW-261119-001
                      2–5 huruf atau angka. Nomor yang sudah dibuat tidak berubah.
Status pajak          (•) Bukan PKP — tanpa PPN
                      ( ) PKP — PPN 12% × 11/12 ditambahkan otomatis
Masa berlaku bawaan   [14] hari
Syarat & ketentuan    [teks bernomor, contoh bawaan disediakan]
```

Bagian ini tampil sebagai kartu baca di Profil usaha; [Ubah] membuka drawer berisi isian di atas. Status pajak dipilih dengan dua kartu radio.

### 6.9 Direktori › Venue (perubahan formulir)

```
Kapasitas*   Berdiri [600] orang   Duduk [400] orang   (isi minimal satu)
Harga        Akad [ ]   Resepsi [ ]   Akad + Resepsi [ ]   Charge tambahan [ ]
```

Daftar venue menampilkan kolom kapasitas ("600 berdiri · 400 duduk").

### 6.10 Proyek › Acara & rincian

Isinya Acara (5.1) dan Rincian (5.3) dengan panel Ringkasan biaya (5.4). Ada tiga tambahan:
- tautan "Kesepakatan awal (Revisi 2)", yang membuka revisi beku itu dalam drawer baca-saja dengan [Bandingkan dengan rincian sekarang];
- banner `termsMismatch` di kartu termin (tab Keuangan › Klien) dan sebagai pesan status setelah simpan;
- riwayat perubahan rincian di Ringkasan › Riwayat ("Tamu Resepsi Sesi 1 550 → 650 · Rina").

### 6.11 Portal (HP 375 px)

**Layanan:**

```
┌──────────────────────────┐
│ Layanan                   │
│ Nilai kontrak             │
│ Rp144.966.000             │  ← hanya izin keuangan
│ termasuk PPN Rp14.366.000 │
│ ───────────────────────── │
│ Paket Resepsi 500 tamu    │
│              Rp120.000.000│
│ • Katering buffet 450 …   │
│ • Dekorasi pelaminan …    │
│ • MC & hiburan            │
│ Lihat 9 isi lainnya       │
│ Tambahan tamu Rp12.000.000│
│ 150 tamu di atas 500 yang │
│ termasuk paket            │
│ Souvenir kipas Rp3.600.000│
│ 300 pcs × Rp12.000        │
│ Bonus: Undangan digital   │
│ Potongan     −Rp5.000.000 │
│ ───────────────────────── │
│ Progres vendor            │
│ Katering ▸ 1 dari 3 tahap │  ← nama tersembunyi → jenis layanan
│ Gedung Melati ▸ …         │
├──────────────────────────┤
│ Beranda Jadwal Pembayaran Layanan Lainnya │
└──────────────────────────┘
```

Keluarga melihat layar yang sama tanpa kartu nilai kontrak dan tanpa kolom nominal.

**Pembayaran › Tagihan vendor:**

```
Tagihan vendor
Anda membayar vendor ini langsung, di luar nilai kontrak.
Fotografer Lensa
DP · jatuh tempo 1 Des 2026   Rp5.000.000   ◷ Belum dibayar
[Kirim bukti bayar]
Pelunasan · 7 Mar 2027        Rp10.000.000
```

**Jadwal › Acara:** kartu per acara berisi tanggal lengkap, venue + alamat + [Buka peta], dan sesi dengan jam dan jumlah tamu.

### 6.12 PDF penawaran (A4 tegak)

**Halaman 1: ringkasan**

```
┌────────────────────────────────────────────────────────────┐
│ [logo] Melati Organizer                  WA 0812… · alamat │
│ ────────────────────────────────────────────────────────── │
│ Penawaran                                                  │
│ MO-PNW-261119-001 · Revisi 2 · 18 November 2026            │
│ Berlaku sampai 30 November 2026                            │
│ Untuk Citra & Bima · PIC Rina Marlina                      │
│                                                            │
│ Acara                                                      │
│ Akad     Jumat, 13 Maret 2027   Masjid Al-Ikhlas  Duduk  120 tamu │
│ Resepsi  Sabtu, 14 Maret 2027   Gedung Melati     Berdiri 2 sesi × 550 │
│                                                            │
│ Ringkasan biaya                                            │
│ Paket Resepsi 500 tamu                     Rp120.000.000   │
│ Tambahan tamu (150 tamu)                    Rp12.000.000   │
│ Item tambahan                                Rp3.600.000   │
│ Potongan                                    −Rp5.000.000   │
│ PPN                                         Rp14.366.000   │
│ Nilai kontrak                              Rp144.966.000   │  ← terbesar di halaman
│                                                            │
│ Termin                                                     │
│ DP 30%       saat penawaran diterima          Rp43.489.800 │
│ Termin 2 40% 60 hari sebelum Resepsi          Rp57.986.400 │
│ Pelunasan 30% 14 hari sebelum Resepsi         Rp43.489.800 │
│ Halaman 1 dari 3 · MO-PNW-261119-001 Rev. 2 · Dibuat dengan ElProof │
└────────────────────────────────────────────────────────────┘
```

**Halaman 2 dst.:**
1. **Rincian:** paket dengan isi berpoin dikelompokkan per kategori (tanpa harga per isi), item "jumlah × harga satuan", bagian **Bonus**, potongan, PPN, dan nilai kontrak. Tajuk tabel diulang di tiap halaman, dan blok total tidak terpisah dari baris terakhir.
2. **Vendor yang dibayar langsung oleh klien:** nama dan layanan, tanpa harga. Kalimat penjelas: "dibayar langsung oleh klien ke vendor, tidak termasuk nilai kontrak".
3. **Syarat & ketentuan** bernomor, termasuk "tidak termasuk" yang ditulis WO di sini (keputusan #27).
4. **Rekening pembayaran** bila diisi di Profil usaha.
5. Penutup singkat dan **blok tanda tangan dua kolom**: "Hormat kami" (nama, jabatan) dan "Disetujui oleh" (nama pasangan, tanggal). Tidak ada kotak meterai: urusan bea meterai di luar cakupan fitur ini.

Tipografi: Plus Jakarta Sans tertanam. Badan 10,5 pt, judul bagian 13 pt, nilai kontrak 16 pt. Angka rata kanan; tanggal ditulis lengkap ("Sabtu, 14 Maret 2027"). Watermark diagonal "DRAF" hanya di pratinjau.

## 7. Keadaan per layar

| Layar | Kosong | Memuat | Galat | Khusus |
|---|---|---|---|---|
| Daftar Penawaran | "Belum ada penawaran. Susun penawaran untuk calon klien dan bagikan sebagai PDF." [Penawaran baru]. Kosong karena saringan: [Reset saringan] | Skeleton berbentuk baris | "Penawaran tidak bisa dimuat." [Coba lagi] | Mode baca-saja langganan: tombol tulis hilang, banner langganan |
| Halaman Penawaran | Tiap bagian punya kalimat + satu aksi (Acara: "Tambahkan acara agar tamu dan kapasitas venue bisa dicek."; Rincian: [Tambah paket] [Tambah item]) | Skeleton per bagian | Per bagian, bukan seluruh halaman | Kedaluwarsa, baca-saja, perubahan belum dikirim, alternatif |
| Drawer acara | — | "Menyimpan…" di tombol | Ringkasan galat + pesan berangka di sesi | Belum ada venue; venue tanpa kapasitas tata letak itu |
| Rincian | lihat atas | Total menampilkan "Menghitung…" selama pratinjau | Pratinjau gagal: total terakhir yang sah tetap tampil dengan "belum diperbarui" | Konflik 412; katalog berubah; potongan melebihi subtotal |
| Paket | lihat 6.6 | Skeleton | [Coba lagi] | Dirujuk penawaran (tidak bisa dihapus) |
| Jadikan proyek | — | "Membuat proyek…" | Kuota masa coba; kapasitas tidak lolos | Klik ganda aman (endpoint idempoten) |
| Portal Layanan | "Rincian belum disusun wedding organizer Anda." | Skeleton | [Coba lagi] | Keluarga tanpa angka |
| Portal Tagihan vendor | Bagian tidak tampil bila tidak ada vendor dibayar klien | — | — | Bukti menunggu diperiksa (belum dihitung lunas) |

## 8. Teks layar dan glosarium

Istilah baru, ditambahkan ke `knowledge/DESIGN.md` §7 dan `check:terms` saat Tahap 1:

| Konsep | Dipakai di layar | Bukan |
|---|---|---|
| quotation | **Penawaran** | Quotation, Proposal |
| revision | **Revisi** (Revisi 2) | Versi |
| package (produk WO) | **Paket**; langganan ElProof = **Paket langganan** | "Paket" untuk langganan |
| package component | **Isi paket** | Komponen |
| event kind / event | **Jenis acara** / **Acara** | Event |
| session | **Sesi** | — |
| guests | **Tamu** (jumlah orang hadir) | Pax di layar klien |
| layout | **Tata letak**: Berdiri / Duduk | Standing, Seated |
| supplier (kerja sama) | **Vendor & venue** (kumpulan); **vendor** / **venue** (butir) | Pemasok |
| paid_by | **Dibayar oleh**: WO / Klien langsung | — |
| agreed_cost | **Biaya rekanan** | Harga modal |
| discount | **Potongan** | Diskon |
| bonus line | **Bonus** | Gratis |
| contract total | **Nilai kontrak** | Total harga |
| accepted revision in project | **Kesepakatan awal** | Snapshot |
| alternative quotation | **Alternatif** | Opsi |
| superseded | **Digantikan** | Superseded |
| document code | **Kode usaha** | Prefix |

Teks tombol mengikuti `knowledge/DESIGN.md` §7, dan nama aksi tetap sama sepanjang alur:

| Tombol | Pesan sesudahnya |
|---|---|
| "Buat revisi untuk klien" | "Revisi 2 dibuat" |
| "Tandai diterima" | "Penawaran ditandai diterima" |
| "Jadikan proyek" | judul halaman "Jadikan proyek" → "Proyek dibuat" |
| "Bagikan PDF" | "PDF siap dibagikan" |

## 9. Aksesibilitas (gerbang, sesuai `knowledge/DESIGN.md` §9)

- Rincian berupa tabel HTML biasa. Baris paket memakai tombol `aria-expanded` yang namanya menyebut isinya: "Tampilkan 12 isi Paket Resepsi 500 tamu".
- Susun ulang selalu lewat menu, dengan pengumuman posisi. Fokus kembali ke pemicu.
- Isian uang memakai `inputmode="numeric"` dan label yang menyebut rupiah. Tidak ada stepper.
- Perubahan total dan hasil hitung ulang diumumkan lewat `role="status"` berupa kalimat utuh, tanpa memindahkan fokus.
- Kapasitas terlampaui ditunjukkan dengan ikon + teks, bukan warna saja. Galat memakai `aria-invalid` + `aria-describedby`.
- Status yang terkunci memakai baca-saja (bisa dibaca pembaca layar), bukan nonaktif. Data tanpa izin tidak dirender.
- Bilah simpan dan panel menempel tidak menutupi fokus (WCAG 2.4.11). Target di portal ≥ 44 px.
- Dialog "Tandai diterima" menampilkan ringkasan sebelum komitmen, dan bisa dibalik dengan "Batalkan terima" (WCAG 3.3.4).

## 10. Uji jalan alur di atas wireframe

Setiap alur §4 dijalankan langkah demi langkah dari sudut pandang perannya. Langkah = satu tindakan (klik, ketik satu isian, atau pilih).

| Alur | Langkah | Masalah yang ditemukan | Perbaikan di dokumen ini |
|---|---|---|---|
| A1 Sales menyusun & mengirim (2 acara, 1 paket, 1 item, 1 bonus, 1 potongan) | ±38 (±14 di antaranya mengetik isian) | (1) Venue harus ditambahkan dua kali: di acara dan di Vendor & venue. (2) Paket dengan venue bawaan tidak terhubung ke acara. (3) Mengirim PDF lewat WhatsApp di desktop butuh unduh manual lalu lampirkan | (1) Memilih venue di acara otomatis menambah Vendor & venue (5.1). (2) Tawaran "Pakai Gedung Melati untuk Resepsi?" saat paket ditambahkan (A1). (3) "Bagikan PDF": lembar bagikan di HP; di desktop unduh + WhatsApp Web terbuka dengan pesan siap (A1) |
| A1 revisi ulang | 4 (ubah → Buat revisi → Bagikan) | Sales bisa lupa bahwa yang dilihat klien adalah revisi, bukan salinan kerja | Banner "Ada perubahan yang belum dikirim ke klien" + aksi utama berganti menjadi "Buat revisi untuk klien" (6.3) |
| A2 Diterima → proyek | 6 (dialog) + 7 (jadikan proyek) | (1) Penawaran kedaluwarsa saat klien baru menjawab: harus perpanjang dulu, lalu terima (2 dialog). (2) Tidak jelas apa yang terjadi pada alternatif | (1) Isian "Perpanjang sampai" di dialog terima (A2). (2) Kalimat "…002 akan menjadi Digantikan" di dialog (6.5) |
| A3 Persiapan pertama | ±20 untuk satu paket berisi 5 isi | Pengguna tidak tahu apa arti "kelipatan" dan "tamu termasuk" | Kotak "Contoh hitung" dengan tamu yang bisa diubah, plus pratinjau tampilan klien (6.6); kata "kelipatan" diganti "Tambahan per … tamu seharga …" |
| A4 Tamu bertambah | 3 (ubah → simpan → pilih jalan keluar) + 2 | Setelah simpan, WP tidak sadar termin jadi tidak cocok | Pesan status berisi perubahan nilai + banner Hitung ulang termin (A4, 6.10) |
| A5 Margin | 2 (Owner membuka panel) | "Margin belum lengkap" tanpa jalan ke vendor yang kurang | Tautan ke daftar tersaring (5.4) |
| A6 Proyek tanpa penawaran | 2 langkah wizard + rincian | Termin tidak bisa dibuat sebelum ada rincian (nilai kontrak 0) | Wizard tidak lagi meminta termin; kartu termin mengarahkan ke rincian (A6) |
| A7 Pengantin bayar vendor sendiri | 5 (Beranda → Tagihan → Kirim bukti → isi → kirim) | Pengantin bisa mengira tagihan vendor bagian dari nilai kontrak | Kalimat "Anda membayar vendor ini langsung, di luar nilai kontrak" di atas bagian (6.11) |
| A8 Keluarga | 3 | Tab "Vendor" kosong angka terasa janggal | Tab berganti nama menjadi "Layanan" dan dibuka dengan isi paket (3.3) |

**Pengukuran dengan orang sungguhan** (keberhasilan, waktu, salah langkah, SEQ) dijalankan pada prototipe setiap tahap, memakai tugas A1–A8 dengan data contoh WO Contoh. Uji di atas adalah uji jalan oleh perancang, bukan pengganti uji pengguna.

## 11. Keputusan rancangan yang juga tercatat di PLAN.md

Keputusan berikut lahir dari rancangan ini dan menjadi aturan di PLAN.md:

1. **Menu Katalog** (Paket, Jenis acara, Jadwal bawaan, Kategori vendor) menggantikan grup Templat. Paket tidak di Pengaturan (D19).
2. Pengaturan usaha bernama **Profil usaha** (nama menu yang ada). Ditambah **masa berlaku bawaan** penawaran (bawaan 14 hari).
3. Tab proyek: **Acara & rincian** (baru) dan **Vendor & venue** (isi baru). Tidak ada tab "Pemasok". Portal: tab Vendor menjadi **Layanan**; acara di Jadwal; tagihan vendor di Pembayaran.
4. Di layar tidak ada kata **"pemasok"** (D21).
5. **Pratinjau hitungan** `POST …/lines/preview` tanpa menyimpan (D7).
6. **Buat alternatif** (keputusan #26) di §5.3 PLAN.
7. Memilih venue di acara otomatis menambah kerja sama venue (5.1).
8. Wizard proyek tanpa penawaran tidak lagi meminta termin (A6).
9. Tombol topbar menjadi "Baru ▾" (Penawaran, Proyek).

## 12. Belum terverifikasi

- Ukuran huruf PDF untuk orang tua (10,5 pt) belum bersumber; diuji dengan cetakan nyata di Tahap 5.
- Tampilan konflik versi dan bandingkan revisi hanya berdasar praktik produk, bukan riset pengguna.
- Bukti untuk Gatera dan produk Indonesia lain hanya dari halaman pemasaran mereka.
- Lembar bagikan berkas (Web Share dengan berkas) didukung Chrome Android; di peramban lain jatuh ke unduhan biasa.
- Hipotesis catatan "Rincian biaya dapat dilihat pengantin" untuk Keluarga **tidak** diambil, supaya tidak membuka bahwa ada angka.

## 13. Sumber

Semua diakses 19 Nov 2026. Daftar lengkap ada di [riset/produk-alur.md §5](riset/produk-alur.md) (49 sumber) dan [riset/pola-ui.md §9](riset/pola-ui.md) (±40 sumber). Yang paling menentukan:
- NN/g: Data tables, Form errors, Mobile tables, Accordions, Sticky elements, Change blindness.
- GOV.UK Design System: Error message, Error summary, Character count, Check answers.
- IBM Carbon: Read-only states, Disabled states, Empty states.
- Atlassian: Drag and drop accessibility.
- W3C: APG Grid, APG Treegrid, WCAG 2.5.7, 3.3.4, 4.1.3.
- Stripe Quotes, Salesforce CPQ bundles, Odoo quotations, HubSpot quotes.
- Planning Pod, Aisle Planner, Event Temple, Tripleseat, Perfect Venue, Dubsado, HoneyBook help centers.
- Gets Wedding Organizer, WO 92, Alissha (PDF penawaran WO Indonesia).
