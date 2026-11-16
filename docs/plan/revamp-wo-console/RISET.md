# Riset WO Console — menu, penyajian fitur, dan penggunaan

Tech Lead · 2026-11-16 · Mengikuti skill `rudis-riset-desain` (langkah 1–3: kasus, riset, temuan → keputusan). Bagian 7 = riset UI visual. Belum ada kode
yang diubah. Bahan: 16 potret e2e WO Console terbaru (desktop 1280 px dan HP 375 px, data contoh, palet navy), konfigurasi
menu `apps/web/src/shared/layouts/Sidebar.tsx`, rute `app/routes/route-paths.ts`, dan riset luar (sumber di akhir).

## 1. Kasus

- **Pengguna:** pemilik WO (Owner), admin keuangan, wedding planner, dan sales, sebagian besar di laptop untuk kerja harian dan HP
  saat di lapangan atau di acara.
- **Pertanyaan harian yang harus terjawab cepat:** "Apa yang harus dikerjakan hari ini? Uang siapa yang belum masuk dan yang
  harus dibayar? Acara apa saja minggu/bulan ini? Proyek X sampai mana?"
- **Dua hal yang dikejar pemilik** (keputusan 16 Nov 2026): kerja lebih ringan (efektif, efisien) serta kepercayaan klien dan
  penjualan.

## 2. Kondisi sekarang

**Menu sidebar:** Beranda · Proyek · Klien · Direktori ▸ (Vendor, Venue) · Pengaturan ▸ (Profil usaha, Tim, Peran dan akses,
Langganan, Kategori vendor, Jadwal bawaan). Atas: kotak cari (Ctrl+K) dan tanggal. Bawah: nama pengguna + ikon keluar.
**Detail proyek:** kepala proyek + 9 tab (Ringkasan, Jadwal, Vendor, Venue, Keuangan, Kendala, Dokumen, Klien, Aktivitas).

**Yang sudah baik (pertahankan):**
- Sidebar kiri selalu terlihat, ringkas, bisa diciutkan jadi rail; item menyesuaikan izin peran (sesuai pedoman NN/g [nng-vnav]).
- Palet perintah Ctrl+K, beranda yang menyorot tagihan terlambat dan "Mulai di sini", tabel dengan cari/saring/urut di server,
  status selalu ikon + teks, lolos axe ketat.

## 3. Temuan → rekomendasi

Prioritas: **P1** berdampak langsung ke dua tujuan pemilik · **P2** kerapian yang terasa setiap hari · **P3** konsistensi.

| # | Temuan (bukti) | Rekomendasi | Prio |
|---|---|---|---|
| 1 | **Tidak ada pusat uang lintas proyek.** Tagihan klien, pembayaran vendor, dan arus kas hanya ada sebagai kartu di Beranda atau di tab Keuangan per proyek. Riset aplikasi WO/event: pembayaran & penagihan adalah fitur yang paling sering disebut paling penting (termin bertahap 6–18 bulan) [planningpod-blog][honeybook-help] | Menu **Keuangan**: tagihan klien (semua proyek, saring jatuh tempo/terlambat), pembayaran ke vendor & venue, arus kas 8 minggu, verifikasi bukti yang menunggu. Datanya sudah ada (`CashFlowCard`, termin, pembayaran) | P1 |
| 2 | **Tidak ada kalender lintas proyek.** Untuk melihat acara dan tenggat bulan ini, pengguna harus menyaring tabel Proyek per tanggal. Kalender adalah fitur inti di HoneyBook, Aisle Planner (didesain ulang Agu 2025), dan Planning Pod [aisle-cal][planningpod-g2] | Menu **Kalender**: tampilan bulan/minggu berisi hari acara, tahapan, dan jatuh tempo termin dari semua proyek; saring per PIC | P1 |
| 3 | **Halaman detail proyek berat.** Kepala proyek setinggi ±420 px (judul, 6 fakta, catatan, blok progres) diulang di setiap tab, sehingga isi tab mulai di bawah lipatan; di HP isi baru terlihat setelah ±900 px. Pola CRM modern: kepala ringkas berisi identitas + aksi, detail di tab, konteks di panel samping [attio-record][hubspot-record] | Kepala ringkas satu baris (nama, status, tanggal acara, H-, nilai kontrak, aksi); fakta lengkap dan progres pindah ke tab Ringkasan | P1 |
| 4 | **9 tab di detail proyek**, di HP bergulir menyamping. Pedoman tab umumnya ≤ 5–7 dan muat satu baris [nng-tabs][tabs-limit] | Ringkas jadi 6: Ringkasan · Jadwal · Vendor & venue · Keuangan · Kendala · Dokumen. Klien (akses portal) pindah ke menu ⋯ / panel di Ringkasan; Aktivitas jadi panel "Riwayat" di Ringkasan | P1 |
| 5 | **Dua banner bertumpuk di atas setiap halaman** (pemeliharaan + masa coba): ±150 px di desktop, ±200 px di HP, sebelum judul halaman. Banner yang bertumpuk membuat pesan penting ikut diabaikan; tutup harus diingat [banner-saasui][banner-eleken] | Satu slot banner dengan prioritas (pemeliharaan > tagihan > masa coba); masa coba pindah jadi chip kecil di kaki sidebar ("Masa coba: 9 hari lagi · Pilih paket"); tutup tersimpan | P2 |
| 6 | **"Pengaturan" mencampur dua jenis hal:** pengaturan workspace yang jarang dibuka (Profil usaha, Tim, Peran, Langganan) dengan data kerja yang dipakai saat membuat proyek (Kategori vendor, Jadwal bawaan). Pedoman settings SaaS: kelompokkan menurut cakupan dan frekuensi [settings-saasframe][settings-saasui] | Grup baru **Templat** (Jadwal bawaan, Kategori vendor, nanti templat termin) di bawah Direktori; Pengaturan hanya workspace & akun | P2 |
| 7 | **Aksi tabel 4 ikon per baris** (ubah, nonaktifkan, lihat, hapus) di Vendor/Venue; di HP jadi 4 tombol per kartu. Praktik umum: ≥ 3 aksi → menu ⋯, satu aksi tersering tetap terlihat, klik baris membuka detail [table-actions][carbon-table] | Klik baris = buka detail; "Ubah" tetap ikon; sisanya di menu ⋯ (nonaktifkan, hapus) | P2 |
| 8 | **Daftar di HP:** saringan (status, PIC, tanggal, arsip, Tampilan, Normal) memenuhi layar pertama; setiap kartu menampilkan semua kolom sebagai label–nilai (8 baris per proyek) | Satu tombol "Saring (n)" membuka sheet; kartu HP hanya 3–4 data kunci (nama, tanggal acara + H-, status, sisa tagihan) | P2 |
| 9 | **Tidak ada menu pengguna.** Kaki sidebar hanya nama + ikon keluar; tidak ada tempat untuk profil pribadi, ganti kata sandi, bantuan. Pedoman: pisahkan pengaturan pribadi dan workspace [settings-saasui] | Menu pengguna (klik nama): Profil saya, Bantuan & WhatsApp ElProof, Keluar | P2 |
| 10 | **Tidak ada tombol "Buat" global.** Membuat proyek/pembayaran/kendala harus masuk halaman yang tepat dulu | Tombol "+ Baru" di topbar (Proyek, Catat pembayaran, Kendala, Klien) | P3 |
| 11 | **Copy tidak konsisten:** "Peran & Akses", "Profil Usaha", "Tambah Venue" (Title Case) vs "Tambah vendor"; Langganan masih "project", "client"; **bug** "Termin Termin 2 terlambat" (`overview.go:73`, `attention.ts:28` menambah kata "Termin" di depan label yang sudah berbunyi "Termin 2") | Sentence case di semua judul/tombol; bersihkan istilah Inggris; perbaiki label termin ganda | P3 (bug: P1) |
| 12 | **URL campur bahasa:** `/projects`, `/clients`, `/vendors` vs `/pengguna` (menu "Tim"), `/timeline-default` (menu "Jadwal bawaan"), `/peran-akses` | Seragamkan saat menu disusun ulang, dengan redirect dari alamat lama | P3 |

## 4. Usulan susunan menu

```
Sekarang                         Usulan
────────────────────────         ──────────────────────────────
Beranda                          Beranda
Proyek                           Proyek
Klien                            Kalender            ← baru
Direktori ▸ Vendor               Keuangan            ← baru
            Venue                Klien
Pengaturan ▸ Profil usaha        Direktori ▸ Vendor
             Tim                             Venue
             Peran dan akses     Templat   ▸ Jadwal bawaan
             Langganan                       Kategori vendor
             Kategori vendor     Pengaturan ▸ Profil usaha
             Jadwal bawaan                    Tim
                                              Peran dan akses
                                              Langganan
[nama]  [keluar]                 [chip masa coba]
                                 [nama ▾] → Profil saya · Bantuan · Keluar
```

## 5. Penilaian

Fondasinya **sudah profesional** untuk SaaS (sidebar yang benar, izin per peran, Ctrl+K, beranda yang menyorot tindakan,
aksesibilitas). Yang membuatnya **belum ideal**:
1. Dua pertanyaan terpenting pemilik WO, "uang saya di mana" dan "acara apa saja bulan ini", belum punya rumah sendiri
   (temuan 1–2).
2. Halaman yang paling sering dibuka, detail proyek, terasa berat (temuan 3–4).
3. Kerapian sehari-hari: banner, aksi tabel, daftar di HP, dan konsistensi copy (temuan 5–12).

## 6. Usulan tahapan

1. **Tahap A (cepat, risiko rendah):** bug "Termin Termin", banner satu slot + chip masa coba, sentence case, aksi tabel ⋯,
   menu pengguna, grup Templat.
2. **Tahap B:** detail proyek (kepala ringkas, 6 tab) dan daftar HP (sheet saring, kartu ringkas).
3. **Tahap C (fitur baru):** halaman Keuangan dan Kalender lintas proyek. Perlu endpoint agregat di API (modul projects) dan
   riset desain per layar sebelum dibangun.

## 7. UI visual (riset kedua, 16 Nov 2026)

Bagian 3–6 menilai struktur dan kegunaan. Bagian ini menilai **rupa**: tipografi, kepadatan, komponen, tabel, grafik, dan
keadaan, dibandingkan dengan tampilan *dalam* aplikasi SaaS modern (bukan situs pemasarannya).

### 7.1 Pembanding yang diamati

| Referensi | Yang diambil |
|---|---|
| shadcn/ui `dashboard-01` dan `sidebar-07` [shadcn-blocks] (kit yang sama dengan ElProof) | Baris 4 kartu KPI dengan angka besar + perubahan + satu kalimat konteks; satu grafik tren; tabel bertab di bawahnya. Sidebar: pemilih workspace berlogo di atas, label seksi kecil, menu pengguna (avatar, nama, email, ▾) di bawah; header berisi tombol sidebar + breadcrumb |
| shadcn/ui `tasks` [shadcn-tasks] | Tabel satu baris per item, judul dipotong dengan elipsis, lencana jenis di depan judul, status = ikon + teks, saringan "chip" (Status, Prioritas) dan tombol "Tampilan" untuk kolom, menu ⋯ per baris |
| Linear, halaman proyek [linear] | Kepala record: judul + **satu baris properti berupa chip** (status, prioritas, PIC, tanggal, label), lalu isi. Tab ringkas di atas (Overview, Activity, …) |
| Twenty CRM [twenty] | Baris rapat dengan ikon/avatar per entitas, kolom yang rapi tidak membungkus, "+ New Record" di kanan atas, seksi Favorit/Workspace di sidebar |
| HoneyBook, Beranda [honeybook-help] | Bilah "business overview" di atas (prospek baru, pesan belum dibaca, tugas, nilai booking setahun), lalu pintasan "Buat" dan ringkasan pembayaran |
| Vercel admin template [vercel-admin] | Tab status di atas tabel (Semua, Aktif, Draf, Arsip) menggantikan dropdown status; thumbnail per baris; ⋯ |

Lembar kontak: `riset/ref-inapp-*.webp` (pembanding) dan `riset/wo-sekarang-*.webp` (WO Console sekarang).

Pola yang sama di semuanya: **hierarki dibangun dengan ukuran/berat huruf dan ruang, bukan dengan kotak**; warna netral +
satu aksen, warna lain hanya untuk status; tabel rapat satu baris; angka penting dibuat besar; data kunci record tampil
sebagai baris properti, bukan kisi label–nilai.

### 7.2 Penilaian layar WO Console

| Kriteria | Temuan (potret 9 Nov 2026, data contoh) | Rekomendasi |
|---|---|---|
| **Tipografi & hierarki** | Judul halaman 20 px, judul kartu 15 px, isi 13–14 px, label 11,5 px: jarak antar-level kecil, sehingga judul kartu, subjudul, dan isi terasa sama berat. Hampir setiap kartu punya judul **dan** kalimat penjelas abu-abu, jadi halaman penuh teks sekunder | Skala minor third dengan dasar 14 px (pola Atlassian [atl-type]): 12 · 13 · 14 · 16 · 20 · 24, angka KPI 28–32. Judul halaman 24/600, judul seksi 16/600. Kalimat penjelas hanya bila menambah informasi, bukan di setiap kartu |
| **Label huruf kapital** | 14 tempat memakai label `uppercase tracking-wide` (KPI Keuangan "NILAI KONTRAK", kepala tabel pembayaran "JENIS / NOMINAL", "TAHAPAN VENDOR"), sementara tabel lain sentence case. DESIGN.md sudah melarangnya | Satu gaya label: 12–13 px sentence case, warna `muted-foreground` |
| **Kepadatan & lebar** | Tabel di tab Keuangan terjepit di kolom 2/3 (kolom "Aksi" terpotong, "2 Okt 2026" dan "TRF-0001" membungkus 2 baris). Daftar Proyek: nama pasangan membungkus 3 baris. Sebaliknya Profil usaha dan Langganan menyisakan setengah layar kosong | Tabel selalu memakai lebar penuh area isi (panel samping turun ke bawah tabel di < 1440 px); teks dipotong elipsis + tooltip, angka/tanggal/ID tidak pernah dibungkus, angka rata kanan tabular [eui-table][primer-table]. Halaman pengaturan memakai tata letak 2 kolom (navigasi seksi kiri, isian kanan) |
| **Komponen tidak seragam** | Dua gaya tabel (DataTable vs tabel polos pembayaran), lencana status dengan bentuk berbeda (pil "Terlambat" merah, lencana verifikasi emas 3 baris), tombol "Tambah" ada yang di kepala kartu ada yang di bawah judul | Satu `DataTable` untuk semua daftar (termasuk pembayaran), satu `Badge` dengan varian status, lencana bukti satu baris ("✓ Diverifikasi") dengan detail di tooltip; aksi utama selalu kanan atas kartu/halaman |
| **Kepala record** | Detail proyek: 6 fakta dalam kisi label–nilai + blok catatan + blok progres (±420 px), datar tanpa penekanan | Pola Linear: judul + chip status + **baris properti** (tanggal acara · H-125 · paket · PIC); 3 angka kunci (nilai kontrak, sisa tagihan, progres) sebagai KPI kecil di tab Ringkasan |
| **Beranda / dashboard** | 3.385 px tinggi, ±10 kartu bertumpuk dengan bobot sama; tidak ada baris KPI; dua grafik "Omzet" dan "Tren proyek baru" berisi hampir semua nol sehingga tampak rusak; "Arus kas 8 minggu" berupa 16 bilah tipis | Tiga lapis [dash-setproduct]: **(1) 4 KPI** (tagihan terlambat, uang masuk 30 hari, acara 30 hari ke depan, proyek aktif) dengan pembanding; **(2) 1–2 grafik** (arus kas masuk/keluar per minggu sebagai satu grafik batang ganda); **(3) daftar "Perlu ditindaklanjuti"**. Grafik tanpa data cukup → keadaan kosong dengan penjelasan, bukan 12 batang nol [gitlab-empty] |
| **Warna** | Netral navy + status sudah benar; emas hanya di bukti (baik). Banner pemeliharaan abu tua terlalu berat dibanding isinya | Pertahankan. Banner memakai gaya info ringan (lihat temuan 5) |
| **Sidebar & header** | Nama workspace teks polos tanpa logo; menu pengguna tidak ada; topbar hanya cari + tanggal | Pola `sidebar-07`: logo WO + nama di atas (jadi pemilih workspace bila nanti ada >1), label seksi kecil, menu pengguna ber-avatar di bawah; topbar: breadcrumb, cari, "+ Baru" |
| **HP** | Lihat temuan 8; kartu proyek 8 baris label–nilai | Kartu 3–4 data kunci; aksi di ⋯; saring di sheet |
| **Keadaan** | Keadaan kosong per halaman sudah ada dan baik ("Mulai di sini"); keadaan memuat memakai kerangka; grafik nol belum ditangani | Lengkapi untuk grafik dan KPI (lihat atas) |

### 7.3 Aturan visual yang diusulkan (masuk `DESIGN.md` setelah disetujui)

- **Huruf:** 12 (meta) · 13 (label, sel tabel padat) · 14 (isi) · 16 (judul seksi) · 20 (judul kartu besar/modal) · 24 (judul
  halaman) · 28–32 (angka KPI, tabular). Berat: 400 isi, 500 label aktif, 600 judul, 700 hanya angka KPI. Tanpa uppercase.
- **Ruang:** kisi 4 px; isi halaman 24 px (≥ 1280: 32 px); jarak antarseksi 24 px; dalam kartu 16–20 px; baris tabel 44 px
  (kompak 36 px).
- **Kotak seperlunya:** kartu hanya untuk satu kesatuan yang bisa dipindah/ditindak (KPI, satu daftar); seksi lain dipisah
  dengan judul + ruang atau garis tipis.
- **Tabel:** kolom pertama = nama + 1 baris meta, dipotong; angka rata kanan; tanggal format pendek ("14 Mar 2027"); status =
  ikon + teks; aksi: ikon "Ubah" + ⋯; tab status di atas tabel bila status adalah saringan utama.
- **KPI:** angka 28 px, label 13 px, satu baris konteks ("dari 3 proyek", "+Rp15 jt dari bulan lalu" bila datanya ada).
- **Grafik:** satu warna data (navy), pembanding abu, status hanya untuk ambang (terlambat = danger); tanpa data cukup →
  pesan kosong + langkah berikutnya.

### 7.4 Wireframe

```
Beranda (desktop)
┌ sidebar ┬────────────────────────────────────────────────────────────────┐
│[logo] WO│ Beranda                                   [Cari ⌘K] [+ Baru]   │
│Beranda  │ Selamat pagi, Rina · Senin 9 Nov                                │
│Proyek   │ ┌Tagihan terlambat┐┌Masuk 30 hari ─┐┌Acara 30 hari┐┌Proyek aktif┐│
│Kalender │ │ Rp40 jt  1 termin││ Rp45 jt        ││ 2 acara      ││ 3          ││
│Keuangan │ └──────────────────┘└────────────────┘└──────────────┘└────────────┘│
│Klien    │ ┌ Arus kas 8 minggu (batang masuk/keluar) ─────┐┌ Acara berikutnya ┐│
│Direktori│ │ ▇▁ ▁▁ ▁▁ ▁▇ …                                ││ 14 Mar Citra&Bima ││
│Templat  │ └──────────────────────────────────────────────┘└───────────────────┘│
│Pengatur.│ Perlu ditindaklanjuti   [Semua 3] [Tagihan 1] [Kendala 1] [Bukti 1] │
│         │ ● Termin 2 terlambat 22 hari · Citra & Bima · Rp40 jt      [Buka]   │
│[chip coba]│ ● Kendala: menu vegetarian belum dikonfirmasi · Citra & Bima        │
│[RM ▾]   │ Aktivitas terbaru (5) …                                             │
└─────────┴────────────────────────────────────────────────────────────────┘

Detail proyek (desktop)
│ Proyek / Citra & Bima                                                     │
│ Citra & Bima  ⓘ Persiapan            [Tandai siap] [Ubah] [⋯]             │
│ 📅 14 Mar 2027 · H-125   📦 Paket Premium   👤 Rina   📍 Gedung Graha Sabuga │
│ Ringkasan  Jadwal  Vendor & venue  Keuangan  Kendala  Dokumen             │
│ ┌Nilai kontrak┐┌Sisa tagihan ┐┌Progres 16% ┐   ┌ Panel: Klien & akses  ┐ │
│ │Rp150 jt     ││Rp105 jt  ⚠  ││▇▁▁▁▁ 2/7   │   │ Riwayat (5 terbaru)   │ │
│ Perlu tindakan …  Jadwal berikutnya …  Termin berikutnya …               │

Daftar proyek (desktop)
│ Proyek                                      [Tampilan] [+ Tambah proyek] │
│ [Semua 3] [Draf 1] [Persiapan 1] [Selesai 1]     [Cari…] [PIC ▾] [Tanggal]│
│ Proyek ↕            Tanggal acara ↕  Status      Nilai kontrak  Sisa  ⋯  │
│ Citra & Bima        14 Mar 2027      ⓘ Persiapan  Rp150.000.000 Rp105 jt  │
│ Citra Lestari & Bima Pratama (1 baris, dipotong)                          │
```

### 7.5 Kesimpulan visual

Rupa WO Console **rapi dan konsisten secara warna**, tetapi belum terasa modern karena tiga hal: hierarki huruf yang datar
dan terlalu banyak teks sekunder, kotak-kotak dengan bobot sama (terutama Beranda dan detail proyek), serta tabel yang
membungkus dan berganti gaya. Ketiganya bisa diperbaiki dengan aturan 7.3 tanpa mengganti kit atau palet, dan sebaiknya
dikerjakan **bersama** Tahap A–B (bagian 6) supaya setiap layar hanya disentuh sekali.

### 7.6 Keuangan (brief layar, riset ketiga, 16 Nov 2026)

**Kasus.** *Subjek:* uang semua proyek: termin klien (masuk), kewajiban ke vendor dan venue (keluar), arus kas ke depan, bukti
transfer yang belum diperiksa. *Audiens:* pemilik WO dan admin keuangan (izin `finance.record`), di laptop untuk menagih dan
membayar, sesekali di HP. *Tugas utama (≤ 5 detik):* "Siapa yang terlambat bayar dan berapa? Apa yang jatuh tempo bulan ini?
Berapa yang masih harus saya bayar ke vendor?"

| Pembanding | Yang diamati |
|---|---|
| HoneyBook, Finance › Overview [hb-finance] dan widget pembayaran Beranda [hb-home] | Pembayaran dipisah **lunas / terlambat / akan datang**; tab per status; angka "semua terlambat" berdiri sendiri; dari ringkasan ada jalan ke daftar pembayaran lengkap |
| Dubsado, saring tagihan per status [dubsado] | Satu daftar tagihan dengan saringan status (belum dibayar, sebagian, lunas, terlambat) alih-alih halaman terpisah |
| Wedsites (aplikasi wedding planner), pembayaran vendor [wedsites] | Setiap pembayaran vendor punya nominal, jatuh tempo, status; saringan Semua · Hari ini · Terlambat · Akan datang · Lunas |
| OfficeRnD, dasbor piutang [officernd-ar]; masukan pengguna Xero [xero-ideas] | Ringkasan dibuka dengan **total terlambat**; "menunggu pembayaran" (belum jatuh tempo) dipisah dari "terlambat"; pengguna meminta **jumlah hari terlambat per baris** dan angka yang bisa diklik sampai ke baris tagihan |
| Atlassian, grouped bar chart [atl-grouped] | Batang berkelompok cocok untuk membandingkan dua seri per periode (masuk vs keluar), garis dasar nol, sedikit seri; bukan untuk membandingkan total |

**Temuan → keputusan.**

| Temuan | Keputusan |
|---|---|
| Semua pembanding memisahkan terlambat / akan datang / lunas | Tab status di atas tabel tagihan: **Terlambat · Jatuh tempo 30 hari · Semua · Lunas** (pola Vercel/Wedsites, temuan 7.1), angka per tab |
| Ringkasan dibuka dengan total terlambat; angka bisa diklik | 5 KPI (tagihan terlambat, jatuh tempo 30 hari, uang masuk 30 hari, belum dibayar ke vendor & venue, bukti menunggu verifikasi); setiap KPI membuka tab/saringan yang menjelaskan angkanya |
| Hari terlambat per baris diminta pengguna | Kolom jatuh tempo menulis "terlambat 12 hari" (teks + warna bahaya, tidak warna saja) |
| Keluar (vendor & venue) dicatat terpisah dari masuk | Tab **Pembayaran keluar**: vendor dan venue dalam satu tabel dengan lencana jenis; status Lunas / Belum lunas / Terlambat |
| Grafik batang berkelompok untuk masuk vs keluar | Tab **Arus kas**: satu grafik batang berkelompok 8 minggu (data arus kas Beranda yang sama, dihitung server) + tabel angka yang sama; tanpa data → pesan kosong, bukan 8 batang nol |
| Bukti transfer klien yang belum diperiksa tidak punya tempat | Tab **Verifikasi bukti (n)**: pembayaran klien yang sudah berbukti tetapi belum diverifikasi, klik baris → tab Keuangan proyek |

**Elemen khas:** tidak ada hiasan baru; yang menonjol hanya **angka uang** (KPI 28 px tabular) dan **status termin** (ikon +
teks). Mencatat pembayaran tetap di proyek; halaman ini membaca saja, aksi baris "Catat pembayaran" menautkan ke tab Keuangan
proyek.

**Token & grafik.** Token yang ada (`StatCard`, `Badge`, `DataTable`, status `--color-*`). Grafik: dua seri, *Masuk* biru navy
`--chart-in` #33519c dan *Keluar* abu navy `--chart-out` #7d89ab (aturan 7.3: satu warna data + pembanding abu). Diuji dengan
`validate_palette.js` skill dataviz: pita terang, CVD ΔE 18,5, penglihatan normal ΔE 19,4, kontras ≥ 3:1 pada putih: lolos;
"chroma floor" gagal hanya untuk abu, disengaja (abu = pembanding). Identitas tidak bergantung warna: legenda selalu ada, tabel
angka di bawahnya, tooltip per batang. Batang ≤ 24 px, ujung atas membulat 4 px, celah 2 px antar batang. Mode gelap ditunda
proyek (DESIGN.md T13), jadi hanya tema terang yang diverifikasi.

```
Keuangan (desktop ≥ 1280)
│ Keuangan                                                                 │
│ Tagihan klien, pembayaran ke vendor dan venue, arus kas semua proyek.    │
│ ┌Terlambat───┐┌Jatuh tempo 30h┐┌Masuk 30 hari┐┌Belum dibayar┐┌Bukti menunggu┐│
│ │Rp40 jt     ││Rp25 jt        ││Rp45 jt      ││Rp12 jt      ││2 pembayaran  ││
│ │1 termin·1 p││2 termin       ││3 pembayaran ││vendor & venue││verifikasi    ││
│ Tagihan klien  Pembayaran keluar  Arus kas  Verifikasi bukti (2)          │
│ [Terlambat 1] [Jatuh tempo 30 hari 2] [Semua] [Lunas]   [Cari] [PIC ▾] [tgl]│
│ Proyek / pasangan      Termin  Jatuh tempo ↕         Nominal  Dibayar  Sisa ↕ Status  ⋯│
│ Citra & Bima           Termin 2  2 Nov 2026 · terl. 7h  …                 ⓘ Terlambat │

HP (375)
│ Keuangan            │
│ [KPI 2 kolom, 5]    │
│ tab bergulir ⇢      │
│ ┌ Citra & Bima ────┐│
│ │ Termin 2         ││
│ │ Jatuh tempo  2 Nov · terlambat 7 hari │
│ │ Sisa      Rp40.000.000 │
│ │ Status    ⓘ Terlambat  │
```

**Keadaan:** memuat (kerangka setelah 400 ms), galat dengan "Coba lagi" per bagian (KPI dan tabel gagal sendiri-sendiri),
kosong yang mengajari (belum ada termin → buat jadwal pembayaran di proyek), kosong karena saringan ("Reset filter"), kosong
yang menenangkan (tab Terlambat tanpa baris: "Tidak ada tagihan terlambat").
**Aksesibilitas:** tab = tautan/`aria-current`, status ikon + teks, grafik punya tabel setara dan `role="img"` + ringkasan,
fokus terlihat, tanpa animasi (aman untuk `prefers-reduced-motion`).
### 7.7 Kalender (riset layar, 16 Nov 2026)

**Kasus.** *Subjek:* tanggal yang menentukan kerja WO dari semua proyek — hari acara, tahapan jadwal yang belum selesai, dan
jatuh tempo termin klien. *Audiens:* pemilik/admin di laptop (merencanakan bulan, melihat uang yang akan masuk) dan planner
di HP (cek cepat minggu ini di lapangan). *Tugas utama:* "Acara apa saja bulan ini, apa yang jatuh tempo minggu ini, dan
mana yang sudah terlambat?" terjawab dalam beberapa detik, lalu satu klik ke proyeknya.

**Temuan → keputusan**

| Temuan | Keputusan |
|---|---|
| Aisle Planner (desain ulang Agu 2025) menyatukan acara, tugas, dan jatuh tempo pembayaran dalam satu kalender akun; pembayaran baca-saja dengan pintasan ke record asalnya; butir hanya tampil bila punya tanggal dan pengguna berhak melihatnya [aisle-cal][aisle-help] | Satu kalender berisi tiga jenis butir; kalender **tidak mengubah apa pun**, setiap butir membuka tab proyek yang tepat (acara → Ringkasan, tahapan → Jadwal, termin → Keuangan). Cakupan per peran dari API, sama dengan daftar Proyek |
| HoneyBook menampilkan tanggal proyek dan pembayaran di kalender yang sama, pembayaran bisa disembunyikan; di HP jenis dibedakan dengan titik warna [honeybook-cal] | Saringan **Jenis** (Hari acara · Tahapan · Termin) dan **PIC**. Jenis dibedakan dengan **ikon + teks + bentuk**, bukan warna saja (DESIGN T5) |
| Kisi bulan di HP: kolom terlalu sempit, judul butir hilang jadi titik; daftar kronologis lebih mudah dibaca di layar kecil; pustaka kalender memakai titik henti ±700 px untuk menukar tampilan [teamup-views][schedulex-views] | **Bulan** di desktop, **Agenda** (daftar per hari) bawaan di < 768 px; pengguna bisa berpindah, pilihan tersimpan di URL (`?tampilan=`) |
| Google Calendar dan kalender desktop umumnya menampilkan beberapa butir per hari lalu "+N lainnya" | Maks. 3 butir per sel; sisanya "+N lainnya" membuka panel hari (Drawer, dimuat saat diklik) berisi semua butir hari itu |
| APG: kisi tanggal = `role="grid"`, satu titik Tab, panah memindah hari, PageUp/PageDown ganti bulan, judul bulan sebagai nama kisi [apg-datepicker]. Roselli: tabel biasa cukup, naikkan ke `grid` hanya bila ada navigasi panah antar-sel atau sel berisi tautan [roselli-cal] | Kisi bulan = `<table role="grid">` dengan **roving tabindex** di sel hari: panah, Home/End (awal/akhir minggu), PageUp/PageDown (bulan sebelumnya/berikutnya), Enter/Spasi membuka panel hari berisi tautan butir. Chip di dalam sel adalah pintasan tetikus (`tabindex=-1`) supaya satu bulan tidak menjadi ±90 titik Tab; tampilan **Agenda** adalah alternatif linear yang setara (judul per hari + daftar tautan). Nama hari lengkap untuk pembaca layar, singkatan untuk mata |

**Elemen khas.** Hari acara adalah jangkar bulan: satu-satunya chip **terisi navy** dengan ikon cincin (permata) dan nama
pasangan; tahapan = chip netral bergaris; termin = latar status lembut + ikon + teks status ("Terlambat", "Lunas"). Emas tidak
dipakai (hanya untuk bukti, §4.2 DESIGN.md). Hari ini = angka dalam lingkaran navy; tanggal di luar bulan memudar.

**Token & rupa (memakai 7.3).** Judul 24/600 (PageHeader); nama bulan 16/600; angka tanggal 13 `tabular-nums`; chip 12/500,
tinggi 22 px, radius 6, satu baris dipotong elipsis + `title`. Sel bulan minimal 112 px tinggi; garis `--border` antar sel,
tanpa kartu di dalam kartu. Termin memakai token status `-soft` + `-text` (kontras ≥ 4,5:1, DESIGN §4.2). Tanpa tema gelap
(DESIGN §10, tidak diadopsi). Gerak hanya transisi hover 120 ms, hilang di `prefers-reduced-motion` (aturan global theme.css).

**Wireframe**

```
Desktop (≥ 1024)
Kalender                                   [‹] November 2026 [›] [Hari ini]   [Bulan|Agenda]
Jenis: [◆ Hari acara ✓] [☐ Tahapan ✓] [Rp Termin ✓]   PIC [Semua PIC ▾]        12 agenda bulan ini
┌ Sen ─┬ Sel ─┬ Rab ─┬ Kam ─┬ Jum ─┬ Sab ─────────┬ Min ─┐
│ 26   │ 27   │ …    │      │      │ 1            │ 2    │
│      │      │      │      │      │[◆ Citra&Bima]│      │  ← navy terisi
│ 9    │ 10   │      │      │      │              │ (16) │  ← hari ini
│      │[!Termin 2·Terlambat] │    │              │      │  ← status lembut + ikon
│      │[☐ Fitting baju]     │     │ +2 lainnya   │      │
└──────┴──────┴──────┴──────┴──────┴──────────────┴──────┘

HP (375, Agenda bawaan)
Kalender
[‹] November 2026 [›]  [Hari ini]
[Bulan|Agenda]   [Saring (1)]
Sab, 21 Nov · H-5
  ◆ Hari H · Citra & Bima        Persiapan
  ! Termin 2 · Rp40 jt           Terlambat
Sen, 23 Nov
  ☐ Fitting baju · Ayu & Dimas   Berjalan
```

**Keadaan.** Memuat: kerangka kisi/daftar (tanpa kedip data lama). Galat: `LoadError` + Coba lagi. Bulan kosong: kisi tetap
tampil (orientasi tanggal) dengan pesan di atasnya "Tidak ada agenda di November 2026" + tombol ke bulan berikutnya yang berisi
bila filter tidak aktif; bila filter aktif: "Reset filter".

**Aksesibilitas.** Kisi: `aria-labelledby` ke judul bulan (`aria-live="polite"` agar ganti bulan diumumkan), `aria-current="date"`
di hari ini, sel di luar bulan tetap terfokus (tanggal tetap bisa dicapai). Saringan jenis = tombol `aria-pressed`. Target
≥ 24 px (≥ 44 px di layar sentuh). Status termin selalu ikon + teks.

**Hasil putaran kritik (setelah dibangun, potret 1440 px dan iPhone 13).** (1) Nama aksesibel chip dimulai dengan teks yang
terlihat lalu menambah termin/nominal sebagai teks tersembunyi (axe `label-content-name-mismatch`, WCAG 2.5.3); di HP teks chip
hanya disembunyikan secara visual. (2) Cincin fokus sel memakai `:focus`, bukan `:focus-visible`: fokus yang dipindah skrip
setelah tombol panah tidak selalu dianggap *focus-visible* oleh peramban. (3) Ringkasan bulan hanya menyebut bagian yang
bukan nol. (4) Baris hari acara tidak mengulang status dua kali. (5) Sel bulan di HP 64 px tinggi (chip jadi ikon saja).
Tanpa tema gelap: ElProof tidak punya tema gelap (DESIGN §10), jadi tidak diverifikasi.

## Sumber (diakses 16 Nov 2026)

- [nng-vnav] NN/g, Left-Side Vertical Navigation on Desktop — https://www.nngroup.com/articles/vertical-nav/
- [nng-tabs] NN/g, Tabs, Used Right — https://www.nngroup.com/articles/tabs-used-right/
- [tabs-limit] KDE HIG Tabs (≤ 7, satu baris) — https://develop.kde.org/hig/components/navigation/tab/ ; CXL tabbed navigation — https://cxl.com/blog/tabbed-navigation/
- [attio-record] Attio, Record page redesign (2026) — https://attio.com/changelog/2026/record-page-redesign
- [hubspot-record] HubSpot, Understand the default record layout — https://knowledge.hubspot.com/records/understand-the-default-record-layout
- [settings-saasframe] SaaSFrame, SaaS settings page design — https://www.saasframe.io/blog/saas-settings-page-design-examples
- [settings-saasui] SaaSUI, Settings page UX patterns — https://www.saasui.design/blog/saas-settings-page-ux-patterns
- [table-actions] SaaSUI, Context menu & overflow actions — https://www.saasui.design/blog/saas-context-menu-overflow-actions-ux-patterns
- [carbon-table] Carbon Design System, Data table usage — https://v10.carbondesignsystem.com/components/data-table/usage/
- [banner-saasui] SaaSUI, Banner & announcement UX — https://www.saasui.design/blog/saas-banner-announcement-ux-patterns
- [banner-eleken] Eleken, Notification UX — https://www.eleken.co/blog-posts/notification-ux
- [honeybook-help] HoneyBook Help Center (dashboard, project navigation) — https://help.honeybook.com/en/articles/5339081-your-honeybook-dashboard
- [aisle-cal] Aisle Planner, New & improved account calendar — https://aisleplanner.com/its-here-new-improved-account-calendar/
- [planningpod-g2] Planning Pod features (G2) — https://g2.com/products/planning-pod/features
- [shadcn-blocks] shadcn/ui blocks dashboard-01 & sidebar-07 — https://ui.shadcn.com/view/new-york-v4/dashboard-01 · https://ui.shadcn.com/view/new-york-v4/sidebar-07
- [shadcn-tasks] shadcn/ui Tasks example — https://ui.shadcn.com/examples/tasks
- [linear] Linear, Plan (tampilan halaman proyek) — https://linear.app/plan
- [twenty] Twenty CRM — https://twenty.com/
- [vercel-admin] Vercel Admin Dashboard template — https://vercel.com/templates/next.js/admin-dashboard
- [atl-type] Atlassian Design System, product typefaces and scale — https://atlassian.design/foundations/typography/product-typefaces-and-scale
- [eui-table] Elastic UI, table layout guidelines — https://eui.elastic.co/docs/components/tables/layout-guidelines/
- [primer-table] GitHub Primer, data table guidelines — https://primer.style/product/components/data-table/guidelines
- [dash-setproduct] Setproduct, dashboard UI design — https://setproduct.com/blog/dashboard-ui-design
- [gitlab-empty] GitLab, chart-level empty state — https://gitlab.com/gitlab-org/gitlab-ce/issues/60947
- [planningpod-blog] Planning Pod, How to choose wedding planner software — https://planningpod.com/blog/how-to-choose-the-best-online-wedding-planner-software-for-your-business
- [hb-finance] HoneyBook Help, Use the Finance overview page — https://help.honeybook.com/en/articles/9510015-use-the-finance-overview-page
- [hb-home] HoneyBook Help, Track upcoming payments from your home screen — https://help.honeybook.com/en/articles/8178991-new-and-improved-july-2023
- [dubsado] Dubsado updates, Invoice filtering by payment status — https://updates.dubsado.com/3815-invoice-filtering-by-payment-status
- [wedsites] Wedsites planner docs, Payments — https://docs.wedsites.com/planner/payments
- [officernd-ar] OfficeRnD Help, AR aging dashboard — https://help.officernd.com/en/articles/248663-dashboard-ar-aging
- [xero-ideas] Xero Product Ideas, Aged receivables: show days overdue — https://productideas.xero.com/forums/939198-for-small-businesses/suggestions/45721732-aged-receivables-report-show-days-overdue
- [atl-grouped] Atlassian, A complete guide to grouped bar charts — https://www.atlassian.com/data/charts/grouped-bar-chart-complete-guide

Catatan: sebagian sumber adalah blog vendor/design system (dipakai sebagai praktik umum, bukan standar). Bukti utama adalah
pengamatan langsung atas potret WO Console dan kode menu/rute.
- [aisle-help] Aisle Planner Help, AP Connect integrated account calendar — https://help.aisleplanner.com/en/articles/11786770-ap-connect-integrated-account-calendar
- [honeybook-cal] HoneyBook Help Center, Your HoneyBook calendar — https://help.honeybook.com/en/articles/3444494-your-honeybook-calendar
- [teamup-views] Teamup, How to use calendar views on Teamup mobile apps — https://calendar.teamup.com/kb/how-to-use-calendar-views-on-teamup-mobile-apps/
- [schedulex-views] Schedule-X, Calendar views (small-screen views, 700 px breakpoint) — https://schedule-x.dev/docs/calendar/views
- [apg-datepicker] W3C WAI-ARIA APG, Date Picker Dialog example (grid, keyboard) — https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/examples/datepicker-dialog/
- [roselli-cal] Adrian Roselli, Brief Note on Calendar Tables (2022) — https://adrianroselli.com/2022/08/brief-note-on-calendar-tables.html
