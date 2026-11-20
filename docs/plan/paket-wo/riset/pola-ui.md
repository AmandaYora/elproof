# Riset pola UI/UX: penyunting rincian, acara & sesi, pemasok, penawaran, portal, PDF

> Bahan untuk `docs/plan/paket-wo/DESAIN.md` (PLAN.md §8 Riset desain). Diakses **19 Nov 2026**. Dokumen ini **menambah**
> `knowledge/DESIGN.md` dan tidak menggantinya: token, kepadatan, wadah (T9), keadaan (§6), validasi hibrida (T8), urungkan (T23),
> edit di tempat (T24), status ikon + teks (T5), format uang PUEBI (T18), dan glosarium (§7) tetap berlaku dan tidak diulang di sini.
>
> **Tanda bukti:** **[V]** saya baca sendiri teks halamannya (diunduh lalu dibaca); **[V~]** saya baca lewat ringkasan alat fetch
> (bukan teks mentah); **[S]** sekunder, opini, data vendor, atau ringkasan mesin pencari; **[?]** tidak terverifikasi.
> Kunci sumber `[nama]` merujuk ke §9.

## 0. Ringkasan rekomendasi

| # | Layar | Pola yang dianjurkan | Bukti terkuat |
|---|---|---|---|
| 1 | Penyunting rincian | Tabel HTML biasa (bukan `grid`/`treegrid`) dengan input di baris untuk field pendek, drawer non-modal untuk detail baris; paket = baris induk dengan tombol buka/tutup isi paket; baris terhitung bersifat **baca-saja** (bukan nonaktif) dengan rumus tertulis; **simpan eksplisit** dengan bilah simpan menempel + total; susun ulang lewat menu ⋯ (+ pegangan seret sebagai tambahan) | [nng-datatables] [apg-grid] [roselli-expando] [carbon-readonly] [gitlab-saving] [wcag-257] [atl-dnd-a11y] |
| 2 | Acara & sesi | Kartu per acara + tabel sesi kecil; **hitung sisa kapasitas langsung** di samping isian tamu (seperti penghitung karakter), tidak memblokir ketikan, menolak saat simpan dengan pesan berangka + tiga jalan keluar (kurangi, bagi sesi, ganti venue) | [govuk-charcount] [govuk-errmsg] [nng-form-errors] [wcag-413] |
| 3 | Pemasok | Tabel + drawer; "Dibayar oleh" yang terkunci tampil **baca-saja** dengan alasan dan cara membukanya; kolom biaya rekanan **tidak dirender** tanpa izin | [carbon-readonly] [carbon-disabled] [sf-cpq-bundle] |
| 4 | Penawaran, revisi & status | Salinan kerja dapat diedit, revisi beku; riwayat revisi terbaru di atas + "Bandingkan dengan revisi sebelumnya" (tambah/hapus/ubah, bukan hanya warna); Terima = dialog pendek berisi ringkasan; Jadikan proyek = halaman penuh dengan ringkasan periksa-sebelum-simpan | [stripe-quotes] [conf-history] [wcag-334] [govuk-check] |
| 5 | Rincian di portal HP | Kartu total di atas, daftar dua kolom (uraian kiri, nominal kanan), isi paket sebagai daftar berpoin yang bisa dibuka; Keluarga: kolom nominal tidak dirender | [baymard-payux] [nng-mobiletables] [nng-accordion] [carbon-disabled] |
| 6 | PDF penawaran | Halaman 1 = ringkasan satu halaman (kop, nomor + revisi, berlaku s.d., acara, ringkasan biaya, total, termin); halaman berikut = rincian, pemasok dibayar klien, S&K, blok "Hormat kami / Disetujui" | Contoh WO Indonesia [wo-gets] [wo-92] [wo-alissha]; struktur surat [accurate-surat]; [proposify-len] (data vendor) |

## 1. Penyunting rincian harga (Penawaran dan Proyek)

### 1.1 Wadah: tabel dengan input di baris + drawer untuk detail

**Dianjurkan.** Satu tabel berkolom: `⠿` · Uraian · Acara · Jumlah × Harga satuan · Nominal · `⋯`. Field yang pendek dan sering
diubah (uraian, jumlah, satuan, harga satuan) diedit langsung di baris. Field yang jarang atau panjang (pemasok, kategori, acara,
catatan) dibuka di **drawer non-modal** "Detail baris" dari ⋯ atau dari chip di baris, supaya baris lain tetap terlihat.

- NN/g: edit di tempat hanya cocok bila tabel sempit, dan baris yang sedang diedit harus tampak berbeda; modal tidak dianjurkan
  untuk edit karena menutupi baris lain, padahal pengguna sering melihat baris lain sebagai acuan nilai; panel samping non-modal
  memberi tampilan penuh satu record sambil tetap melihat tabel [nng-datatables] [V].
- Produk sejenis memakai baris yang langsung diedit: Odoo menambah produk, **bagian (section)**, dan **catatan** langsung di tab
  Order Lines lalu menyeretnya ke posisi [odoo-quote] [V]; Shopify draft order menambah item kustom dan potongan per baris atau per
  pesanan [shopify-draft] [S].
- **Semantik: tabel HTML biasa, bukan `role="grid"`.** APG: grid adalah widget komposit dengan satu tab stop dan navigasi panah
  yang dikelola sendiri; panah tidak bisa dipakai untuk menggerakkan kursor di dalam isian kecuali ada mode edit (F2/Enter, Esc)
  [apg-grid] [V]. Penyunting ini berisi banyak isian teks dan combobox, jadi tabel biasa dengan setiap isian sebagai tab stop lebih
  sederhana dan tidak bentrok dengan pembaca layar. Pakai `grid` hanya bila kelak ada kebutuhan navigasi sel ala spreadsheet.

**Alternatif.** (a) Semua baris dibuka di drawer (tabel baca-saja): lebih rapi untuk tabel lebar, tetapi menambah satu klik per
perubahan harga, padahal harga dan jumlah adalah yang paling sering diubah saat nego. (b) Formulir bertumpuk per baris (kartu):
cocok untuk HP (lihat 1.9), bukan untuk konsol.

**Tidak cocok bila** baris punya lebih dari ±6 field yang wajib diisi bersamaan; pindahkan ke drawer.

### 1.2 Baris paket dengan isi paket bersarang

**Dianjurkan.** Baris **Paket** adalah baris induk. Sel pertamanya memuat tombol buka/tutup (`<button aria-expanded>`), nama
paket, dan jumlah isi ("12 isi paket"). Baris **isi paket** ditampilkan menjorok di bawahnya, dengan sel nominal bertuliskan
"Termasuk paket" (bukan "Rp0", supaya tidak terbaca gratis). Baris **Tambahan tamu** selalu tepat di bawah isi paket. Paket baru
ditambahkan dalam keadaan terbuka; WO bisa menutupnya.

- Pola "tabel dengan baris yang bisa dibuka" memakai tombol disclosure biasa di dalam tabel, dengan nama aksesibel yang menyebut
  isinya, dan `aria-expanded` sebagai kait gaya [roselli-expando] [V].
- **Hindari `treegrid`.** Halaman APG treegrid (diakses 19 Nov 2026) menjelaskan polanya tanpa peringatan, tetapi mencatat bahwa
  dalam mode aplikasi pembaca layar hanya mengumumkan elemen yang bisa difokus, dan panah tidak bisa mengoperasikan combobox di sel
  [apg-treegrid] [V~]. Roselli menilai dukungan treegrid buruk dan jarang ditemui di lapangan [roselli-expando] [S]. Lihat §8 (bertentangan).
- Salesforce CPQ menyembunyikan atau menampilkan baris komponen bundel secara terpisah untuk editor dan dokumen (Editor Only /
  Document Only / Always / Never), dan **mengunci jumlah** baris komponen yang diturunkan dari bundel [sf-cpq-bundle] [V~]; pilihan
  "indentasi hierarki produk" menjorokkan baris opsi di bawah produk induknya [sf-cpq-search] [S].
- WO Indonesia menulis isi paket sebagai daftar berkelompok per kategori dengan nama vendor dalam kurung, tanpa harga per isi, dan
  satu angka harga paket [wo-gets] [V]. Ini cocok dengan PLAN §1 (isi paket tanpa harga).

**Isi paket di penyunting:** hanya uraian yang bisa diedit (dan pemasok di drawer). Tidak ada kolom harga, jumlah, atau ⋯
"Hapus paket" di baris anak; menghapus isi paket satu per satu tetap boleh lewat ⋯ baris anak.

### 1.3 Baris terhitung (Tambahan tamu, PPN): baca-saja dan menjelaskan asal angkanya

**Dianjurkan.**
- Tampil **baca-saja**, bukan nonaktif: tanpa batas isian, teks tetap berkontras penuh, tanpa ikon interaktif. Carbon: nonaktif
  tidak dibaca pembaca layar dan tidak lolos kontras, jadi untuk informasi yang harus dibaca pakai baca-saja [carbon-readonly] [V].
- **Rumus tertulis di baris**, di bawah uraian dalam teks sekunder, bukan disembunyikan di tooltip:
  `620 tamu Resepsi − 500 termasuk = 120 → 3 × 50 tamu × Rp4.000.000`. Untuk PPN: `12% × DPP 11/12 × subtotal Rp100.000.000`.
- Satu **toggletip** (klik, bukan hover) "Cara menghitung" berisi aturan paket yang tersalin dan tautan **"Ubah tamu di Acara"**
  serta **"Tambah potongan"** (selisih nego masuk ke Potongan, PLAN §3.1). Carbon: tooltip untuk info singkat tanpa interaksi saat
  hover/fokus; toggletip dibuka dengan klik/Enter dan boleh berisi tombol; jangan pakai untuk info kritis [carbon-toggletip] [V].
- Bila tamu belum diisi: baris tetap ada dengan nominal Rp0 dan tanda "Tamu belum diisi" (PLAN §4.2), bukan disembunyikan.
- PPN: baris hanya muncul bila workspace PKP; non-PKP tidak menampilkan "PPN Rp0".

**Tidak cocok:** menampilkan baris terhitung sebagai isian yang tampak bisa diketik lalu menolak ketikan; itu "modal tersembunyi"
yang membingungkan.

### 1.4 Isian uang dan angka (format id-ID)

**Dianjurkan.**
- `<input type="text" inputmode="numeric">` untuk harga satuan, jumlah, dan tamu; **jangan `type="number"`**: GOV.UK beralih karena
  masalah aksesibilitas, angka besar dibulatkan atau menjadi notasi eksponensial saat ditambah/dikurangi, dan roda tetikus bisa
  mengubah nilai tanpa sengaja [govuk-numbers] [V] [govuk-textinput] [V].
- Awalan **"Rp"** tampil visual di dalam isian, **dan** labelnya menyebut satuannya ("Harga satuan (rupiah)"), karena awalan tidak
  dibacakan pembaca layar. Bila pengguna mengetik "Rp" sendiri, terima tanpa galat [govuk-textinput] [V].
- Terima "1500000", "1.500.000", "1 500 000", "Rp1.500.000"; normalkan ke angka; tampilkan ulang dengan pemisah ribuan saat blur
  (format T18). GOV.UK: batas panjang yang ketat bisa menghalangi orang menulis koma/spasi di angka besar; validasi setelah normalisasi
  [govuk-textinput] [V]. Rupiah bulat: tolak desimal dengan pesan, jangan diam-diam membulatkan.
- **Tanpa stepper** (+/−) untuk harga, jumlah item, dan tamu. NN/g: stepper hanya untuk nilai yang sedikit menyimpang dari bawaan;
  tidak untuk penyesuaian besar (1 → 50) atau besaran kontinu seperti harga [nng-steppers] [V]. Tamu 120 → 650 dan "300 pcs" adalah
  penyesuaian besar.
- Isian rata kanan, `tabular-nums`, lebar sesuai panjang wajar (GOV.UK: lebar isian mengisyaratkan panjang jawaban [govuk-textinput] [V]).

### 1.5 Total dan perubahan total

**Dianjurkan.**
- **Kaki tabel** memuat Subtotal, Potongan, PPN (bila PKP), dan **Nilai kontrak** (tebal, 20 px). Di Proyek, kartu biaya rekanan /
  margin (berizin) di sampingnya (PLAN §7).
- **Bilah simpan menempel di bawah** (lihat 1.6) juga memuat total ringkas ("Nilai kontrak Rp152.400.000"), supaya total terlihat
  saat menggulir tabel panjang. NN/g: elemen menempel harus kecil, buram (tidak transparan), dan minim animasi [nng-sticky] [V];
  fokus tidak boleh tertutup elemen menempel (DESIGN T4, WCAG 2.4.11).
- **Tandai perubahan.** Setelah perhitungan ulang, tampilkan selisih di samping total selama beberapa detik ("+Rp12.000.000") dan
  sorot baris yang ikut berubah (mis. Tambahan tamu). NN/g: perubahan di luar titik perhatian sering tidak terlihat tanpa isyarat
  yang kuat [nng-changeblind] [V].
- **Umumkan ke pembaca layar** lewat region `role="status"` (polite) yang memuat seluruh kalimat, mis. "Nilai kontrak sekarang
  Rp152.400.000", bukan hanya angkanya, dengan `aria-atomic`; jangan memindahkan fokus. WCAG 4.1.3: pesan status harus bisa
  diumumkan tanpa menerima fokus, dan bagian yang diumumkan sebaiknya seluruh teks, bukan hanya angka yang berubah [wcag-413] [V].
  Tunda ±500 ms setelah ketikan terakhir (GitLab: debounce 500 ms untuk validasi [gitlab-saving] [V]).

**Keputusan teknis untuk Tech Lead/BSA:** angka sementara di layar perlu dihitung sebelum disimpan. PLAN §2.1 menaruh `LineTotals` dan
`ExtraGuests` di `offers/domain` (Go) sebagai satu-satunya tempat logika. Pilihan: (a) endpoint pratinjau tanpa simpan
(`POST …/lines/preview`, debounce 500 ms) supaya rumus tetap satu tempat; atau (b) rumus disalin di web (cepat, tetapi dua
implementasi bisa berbeda). Riset ini menganjurkan (a); bukan keputusan desain murni.

### 1.6 Simpan eksplisit, perubahan belum tersimpan, dan konflik versi

**Dianjurkan: simpan eksplisit untuk seluruh rincian**, bukan simpan otomatis per baris.
- GitLab: simpan otomatis **tidak** untuk data berdampak keuangan, keamanan, atau privasi, dan tidak untuk seluruh formulir yang
  baru bermakna bila lengkap; bawaan adalah simpan manual [gitlab-saving] [V]. Rincian harga berdampak ke nilai kontrak dan termin
  (`termsMismatch`, ADR-0042) dan disimpan dengan `If-Match: lines_version` (PLAN §3.2).
- **Bilah simpan menempel** muncul begitu ada perubahan: "Perubahan belum disimpan · [Batalkan perubahan] [Simpan rincian]".
  Shopify: bilah muncul saat isian berbeda dari nilai tersimpan, selalu sediakan jalan membuang perubahan, tampilkan status memuat
  saat menyimpan, dan konfirmasi pembuangan bila kehilangannya besar [shopify-savebar] [V~]. SAP Fiori elements menaruh Simpan dan
  Buang draf di bilah kaki halaman objek, rata kanan, setelah pengguna meninjau isi [sap-footer] [S].
- **Peringatan saat meninggalkan halaman** dengan perubahan belum tersimpan: dialog "Perubahan rincian belum disimpan" dengan
  [Simpan perubahan] (utama) dan [Buang dan tinggalkan halaman] [gitlab-saving] [V].
- Setelah simpan: konfirmasi paling tidak mengganggu. Bila hasilnya sudah terlihat (total berubah), cukup status singkat di tombol
  ("Tersimpan"); toast tanpa aksi hanya bila hasil tidak terlihat [gitlab-saving] [V].
- **Konflik versi (412).** Pertahankan isian pengguna; tampilkan banner di atas tabel: "Rincian ini sudah diubah orang lain pada
  14.03. Perubahan Anda belum disimpan." dengan [Lihat perbedaan] dan [Muat versi terbaru]. "Lihat perbedaan" memakai tampilan
  banding yang sama dengan revisi (§4.3). Bukti untuk pola ini **lemah**: hanya praktik vendor dan forum (pesan "diubah pengguna
  lain, tidak bisa disimpan" yang polos dikritik di pelacak isu Drupal; Convert memakai pilihan Muat ulang / Timpa) [conflict-search] [S].
  Jangan sediakan "Timpa" untuk data uang.

**Alternatif.** Draf lokal di peramban (pulih setelah tab tertutup) sebagai tambahan, bukan pengganti simpan; SAP Fiori memakai
draf yang tersimpan otomatis di server dengan "Buang draf" [sap-footer] [S]. **Tidak cocok** untuk ElProof saat ini karena PLAN tidak
punya tabel draf rincian.

### 1.7 Susun ulang baris yang aksesibel

**Dianjurkan.**
- Menu ⋯ setiap baris memuat **Pindah ke atas · Pindah ke bawah · Pindah ke paling atas · Pindah ke paling bawah**. Ini cara
  utama, sesuai `RowActions` (DESIGN §4.5).
- Pegangan seret `⠿` selalu terlihat di kiri sebagai cara tambahan untuk tetikus (Atlassian: pegangan selalu terlihat bila seret
  adalah aksi utama; target ≥ 24 × 24 px; satu tombol aksi per entitas) [atl-dnd] [V].
- WCAG 2.5.7 (AA): setiap fungsi seret harus bisa dilakukan dengan satu penunjuk tanpa menyeret, mis. tombol naik/turun atau menu
  [wcag-257] [V~].
- Atlassian **tidak menganjurkan** tombol panah sebagai alternatif utama (sulit tanpa melihat layar, pengguna JAWS harus berganti
  mode, banyak tombol untuk perpindahan jauh) dan menganjurkan menu berisi hasil perpindahan; nama aksesibel menyebut barisnya
  ("Pindahkan Gubukan ke paling atas"); fokus kembali ke pemicu setelah pindah; umumkan nama baris serta posisi lama dan baru lewat
  live region [atl-dnd-a11y] [V]. Chrome: tombol/menu adalah bawaan terbaik untuk ditemukan dan kompatibel dengan pembaca layar;
  ubah urutan DOM yang sebenarnya, bukan `order` CSS; umumkan hanya bila posisi benar-benar berubah [chrome-dnd] [V].
- Contoh pengumuman: "Gubukan dipindah ke posisi 3 dari 8."
- **Aturan urutan khusus rincian:** paket berpindah bersama isi paket dan Tambahan tamu-nya; isi paket hanya berpindah di dalam
  paketnya; Tambahan tamu tidak bisa dipindah (selalu di bawah isi paket); Subtotal/PPN/Total bukan baris dan tidak ikut diurutkan.
- Di HP, **tidak ada seret**, hanya menu (NN/g: Gmail menghapus seret di HP dan memakai menu karena lebih sedikit galat) [nng-dnd] [V].

### 1.8 Menambah baris, bonus, potongan, galat per baris

- **Satu tombol "Tambah ▾"** di bawah tabel dengan pilihan: Paket dari katalog (combobox pencarian), Item, Bonus, Potongan. Fokus
  pindah ke isian pertama baris baru. Pola Odoo: Add a product / Add a section / Add a note di bawah baris [odoo-quote] [V].
- **Bonus**: item berlabel "Bonus" dengan harga satuan baca-saja "Rp0" (bukan isian yang bisa dikosongkan). Contoh WO Indonesia
  memisahkan bagian **BONUS** di penawaran [wo-gets] [V].
- **Potongan**: nominal diketik positif, ditampilkan "−Rp5.000.000"; galat bila melebihi subtotal, dengan angka:
  "Potongan melebihi subtotal (Rp95.000.000). Kurangi potongan." (gaya GOV.UK: deskripsi untuk nilai di luar batas, instruksi untuk
  isian kosong [govuk-errmsg] [V]).
- **Galat saat simpan:** pesan di baris yang salah + ringkasan galat di atas tabel yang menaut ke isian itu dan menerima fokus
  [govuk-errsum] [V]. Galat dari server memakai indeks baris (DESIGN E10). Lihat §8 soal ambang ringkasan galat.

### 1.9 Penyunting di HP (375 px)

Konsol dipakai di laptop, tetapi PLAN §8 meminta tangkapan 375 px. Dianjurkan: tiap baris menjadi kartu (uraian, acara, nominal),
ketuk kartu membuka sheet layar penuh untuk mengedit, susun ulang hanya lewat menu, bilah simpan menempel tetap ada. NN/g: di layar
sempit hanya ±2 kolom teks yang terbaca; kunci tajuk bila tetap tabel [nng-mobiletables] [V].

## 2. Penyunting acara & sesi (aturan keras kapasitas)

### 2.1 Tata letak

**Dianjurkan.** Satu bagian per acara (urut tanggal; acara tanpa tanggal di bawah, urutan manual lewat menu ⋯ seperti 1.7):

```
Resepsi · Sabtu, 14 Mar 2027 · Gedung Melati          [⋯]
Tata letak  (•) Berdiri  ( ) Duduk      Kapasitas berdiri 600
Sesi     Mulai   Selesai   Tamu        Sisa kapasitas
Sesi 1   11.00   13.00     [ 550 ]     50 lagi
Sesi 2   18.00   21.00     [ 650 ]     ⚠ 50 di atas kapasitas
[+ Tambah sesi]                         Total tamu acara 1.200
```

- Tata letak = radio berlabel (dua pilihan, keduanya terlihat), angka kapasitas yang dipakai tertulis di sebelahnya.
- Isian tamu: `inputmode="numeric"`, tanpa stepper (§1.4).
- Acara tanpa venue: lencana "Belum dicek ke kapasitas venue" (ikon + teks), tamu tetap boleh diisi (PLAN §4.1).

### 2.2 Umpan balik kapasitas: hitung langsung, tolak saat simpan

**Dianjurkan.**
- **Sisa kapasitas dihitung langsung** di samping isian tamu, seperti penghitung karakter: "50 lagi" → "50 di atas kapasitas".
  GOV.UK Character count menampilkan sisa saat mengetik, **tidak menghalangi** pengguna mengetik melebihi batas (supaya bisa
  menempel jawaban lengkap lalu menyunting), dan pembaca layar mendengar hitungan setelah berhenti mengetik [govuk-charcount] [V].
  Polanya sama: biarkan WO mengetik 650, tunjukkan selisihnya, jangan memotong angka.
- **Saat simpan, sistem menolak** (aturan keras #20) dengan pesan berangka di isian sesi dan di ringkasan galat:
  "Gedung Melati muat 600 orang berdiri, Sesi 2 Resepsi 650." Gaya pesan: apa yang terjadi + cara memperbaiki [govuk-errmsg] [V];
  NN/g: pesan eksplisit, sopan, tepat, memberi saran yang membangun, di dekat isian [nng-form-errors] [V].
- **Jalan keluar di tempat** (alur PLAN §8 no. 4), di bawah pesan: [Bagi jadi 2 sesi] · [Ganti venue] · [Ubah tata letak] (hanya bila
  kapasitas tata letak lain cukup, dengan angkanya: "Muat 800 bila duduk"). Tombol tidak dinonaktifkan karena validitas (DESIGN T8).
- **Venue tanpa angka untuk tata letak itu:** "Isi kapasitas duduk Gedung Melati di Venue" dengan tautan ke formulir venue
  (drawer, supaya isian acara tidak hilang).
- **Efek ke bagian lain diumumkan setelah simpan**, bukan diam-diam: "Tambahan tamu di Rincian berubah dari Rp8.000.000 menjadi
  Rp12.000.000" dan, bila termin tidak lagi cocok, banner Jadwal Termin "Hitung ulang termin" (ADR-0042). Isyarat perubahan perlu
  kuat agar terlihat [nng-changeblind] [V]; pengumuman lewat `role="status"` [wcag-413] [V].

**Alternatif.** Meter/bilah kapasitas per sesi: lebih visual, tetapi warna saja tidak cukup (DESIGN T5) dan memakan ruang di
konsol padat; pakai teks "x lagi / x di atas kapasitas", bilah opsional.

**Tidak cocok:** peringatan yang bisa dilewati (ditolak di PLAN §10), atau memblokir ketikan di atas batas (pengguna tidak tahu
mengapa angkanya tidak masuk).

## 3. Daftar pemasok (vendor/venue)

**Dianjurkan.** `DataTable` dengan kolom: Pemasok (nama + kategori), Jenis (Vendor/Venue), Dibayar oleh, Tampil ke klien,
Biaya rekanan (berizin), Progres, Pembayaran; aksi baris `RowActions`. Detail dan edit di drawer.

- **"Dibayar oleh" (WO / Klien langsung)** = radio di drawer. Setelah ada pembayaran atau bukti: tampil **baca-saja** dengan ikon
  kunci + teks "Terkunci: sudah ada 2 pembayaran", dan cara membukanya "Hapus pembayaran terkait untuk mengubah". Carbon: pakai
  baca-saja bila isinya masih perlu dibaca; nonaktif tidak dibaca pembaca layar [carbon-readonly] [V]; bila sesuatu nonaktif karena
  prasyarat, beri notifikasi yang menjelaskan cara mengaktifkannya kembali [carbon-disabled] [V]. Ini merinci DESIGN T12.
- **"Tampilkan nama ke klien"** = sakelar hanya untuk vendor dibayar WO. Untuk venue dan pemasok dibayar klien, jangan tampilkan
  sakelar nonaktif; tulis teks baca-saja "Selalu tampil ke klien" (PLAN §3.1). Pembanding: Salesforce CPQ membedakan visibilitas
  baris untuk editor dan dokumen keluaran [sf-cpq-bundle] [V~]. Opsional: tautan "Lihat sebagai klien" ke pratinjau portal.
- **Biaya rekanan tanpa izin `finance.margin.view`**: kolom dan isian **tidak dirender sama sekali** (bukan "Rp•••", bukan nonaktif).
  Carbon: varian "tersembunyi" dipakai bila pengguna tidak berizin melihat atau bertindak; satu-satunya cara memunculkannya adalah
  mengubah izin [carbon-disabled] [V]. Cocok dengan PLAN §4.3 (API mengosongkan kolom).
- **"Margin belum lengkap (3 pemasok)"** menaut ke tabel yang tersaring ke pemasok dibayar WO tanpa biaya rekanan.
- **Kosong:** "Belum ada pemasok di proyek ini." + [Tambah pemasok] (pilih dari Direktori). Kosong karena filter: teks + Reset filter
  (DESIGN §6).

## 4. Penawaran: revisi, status, terima/tolak, jadikan proyek

### 4.1 Kepala dan aksi per status

**Dianjurkan.** Kepala halaman: nomor + "Revisi 2" + lencana status (ikon + teks) + "Berlaku s.d. 30 Nov 2026 (H-11)". Satu aksi
utama yang berganti menurut keadaan:

| Keadaan | Aksi utama | Aksi lain |
|---|---|---|
| Draf, salinan kerja berbeda dari revisi terakhir (atau belum ada revisi) | **Buat revisi untuk klien** | Pratinjau PDF, Tolak, Salin jadi penawaran baru |
| Draf, salinan kerja sama dengan Revisi n | **Kirim via WhatsApp** / Unduh PDF Revisi n | Tandai diterima, Tolak |
| Kedaluwarsa | **Perpanjang masa berlaku** | Salin |
| Diterima, belum jadi proyek | **Jadikan proyek** | Batalkan terima |
| Ditolak | Buka lagi | Salin |
| Digantikan / sudah jadi proyek | (baca-saja) tautan ke penawaran pengganti / proyek | Unduh PDF |

- Penanda **"Salinan kerja berbeda dari Revisi 2"** di atas isi, karena klien hanya pernah melihat revisi. Stripe membedakan tegas:
  Draft bisa diedit; setelah difinalkan (open) hanya tanggal kedaluwarsa yang bisa diubah; yang kedaluwarsa atau dibatalkan tidak bisa
  diterima; nomor mendapat akhiran revisi (QT-…-0001-**2**) [stripe-quotes] [V]. Ini mendukung PLAN §5.1–5.3 (nomor tetap + "Revisi n").
- **Kedaluwarsa** tampil sebagai satu lencana "Kedaluwarsa" (bukan "Draf" + "Kedaluwarsa"), dengan subteks "berlaku s.d. 12 Nov".
  Zoho menjadikan Expired status tersendiri di daftar [zoho-est] [S]; Stripe membatalkan otomatis saat kedaluwarsa [stripe-quotes] [V].
  Tab saringan daftar: Semua, Draf, Kedaluwarsa, Diterima, Ditolak, Digantikan (DESIGN §4.5: tab status bila status saringan utama).
- **Pratinjau PDF** membuka penampil dengan watermark diagonal "DRAF" dan tidak membuat revisi (PLAN §5.2). Label tombol jelas
  berbeda dari "Buat revisi untuk klien".

### 4.2 Riwayat revisi

**Dianjurkan.** Bagian "Riwayat revisi" (terbaru di atas): Revisi n · tanggal · oleh · nilai kontrak · ringkasan perubahan
("+2 item, potongan Rp5.000.000") · [Unduh PDF] · [Bandingkan dengan Revisi n−1]. Di bawahnya log aktivitas (dibuat, diterima,
ditolak, saluran, lampiran). Pelacak status: terbaru di atas, bertanggal (DESIGN T16). Confluence menampilkan daftar versi dengan
penulis dan tanggal, dan memulihkan versi lama sebagai versi baru (riwayat tidak hilang) [conf-history] [V~].

### 4.3 Bandingkan revisi

**Dianjurkan.** Tabel banding per baris rincian dan per acara/sesi: baris ditambah ("Ditambah", ikon +), dihapus (dicoret,
"Dihapus", ikon −), diubah ("Rp4.000.000 → Rp4.500.000"); baris yang sama diciutkan ("8 baris tidak berubah"); total lama → baru di
kaki. Confluence: hijau = ditambah, merah = dihapus, biru = format berubah, bagian yang tidak berubah diciutkan, tombol ke pasangan
versi sebelumnya/berikutnya [conf-history] [V~]. Warna tidak boleh satu-satunya pembeda (DESIGN T5), jadi selalu ikon + teks.
Bukti hanya dari dokumentasi produk, bukan riset pengguna (§8).

### 4.4 Terima / tolak

- **Tandai diterima** = dialog pendek (DESIGN T9: modal 1–3 field): pilih revisi (bawaan revisi terakhir), tanggal, saluran
  (WhatsApp / tatap muka / email / lainnya), lampiran opsional; di atas isian tampil ringkasan "Revisi 2 · Nilai kontrak
  Rp152.400.000 · 3 termin". WCAG 3.3.4 (AA): untuk tindakan yang menimbulkan komitmen hukum atau keuangan, sediakan cara meninjau,
  memastikan, dan mengoreksi sebelum final, atau buat dapat dibalik [wcag-334] [V]. "Batalkan terima" (sebelum jadi proyek) adalah
  jalan balik itu.
- Bila salinan kerja berbeda dari revisi mana pun: dialog menjelaskan "Buat revisi dulu agar yang diterima sama dengan yang
  dilihat klien" dengan tombol [Buat revisi untuk klien], bukan tombol Terima yang nonaktif tanpa alasan.
- **Tolak** = dialog dengan alasan opsional; bisa dibuka lagi, jadi tidak perlu konfirmasi berat (DESIGN T23).

### 4.5 Jadikan proyek (form pelengkap)

**Dianjurkan: halaman penuh**, bukan drawer: isiannya berulang per acara (tanggal pasti tiap acara) ditambah tanggal mulai
persiapan, PIC, PIC Sales, dan tanda jadi opsional (DESIGN T9: halaman penuh untuk alur bertahap). Susunan:
1. Tanggal pasti per acara (terisi dari penawaran bila ada).
2. Tim: PIC, PIC Sales (bawaan PIC penawaran).
3. Tanda jadi yang sudah diterima (opsional): nominal, tanggal, metode.
4. **Ringkasan sebelum simpan**: nilai kontrak, termin yang akan dibuat (nominal dihitung), jumlah acara/pemasok/baris yang
   disalin, dan pemeriksaan kapasitas ulang. GOV.UK: satu halaman periksa jawaban tepat sebelum konfirmasi untuk transaksi kecil-sedang
   [govuk-check] [V]; WCAG 3.3.4 [wcag-334] [V].
5. Tombol [Buat proyek] → "Membuat proyek…"; endpoint idempoten (PLAN §5.4), jadi klik ganda aman. Setelah selesai: pindah ke
   proyek dengan banner "Dibuat dari MO-PNW-261119-001 Revisi 2 · Lihat kesepakatan awal".

## 5. Rincian di portal klien (HP Android kelas menengah)

**Dianjurkan.**

```
┌ Rincian ───────────────────────┐
│ Nilai kontrak                   │
│ Rp152.400.000                   │  ← angka besar, label kecil di atas (DESIGN §5.4)
│ termasuk PPN Rp15.100.000       │
├─────────────────────────────────┤
│ Paket Resepsi 500 tamu  Rp120.000.000 │
│ ▸ Lihat 12 isi paket            │  ← daftar berpoin, tanpa harga per isi
│ Tambahan tamu           Rp12.000.000 │
│ 120 tamu di atas 500 yang termasuk paket │
│ Souvenir                 Rp3.600.000 │
│ 300 pcs × Rp12.000              │
│ Undangan digital   Bonus        │
│ Potongan               −Rp5.000.000 │
│ PPN                    Rp15.100.000 │
│ Nilai kontrak         Rp152.400.000 │
└─────────────────────────────────┘
```

- **Total lebih dulu, rincian di bawahnya.** Baymard: total lengkap (termasuk pajak dan biaya) harus terlihat sebelum meminta tindakan
  bayar; ringkasan yang diciutkan tetapi bisa dibuka lebih baik daripada disembunyikan; menampilkan total berbeda dari menampilkan
  ringkasan, keduanya perlu [baymard-payux] [V~].
- **Daftar dua kolom, bukan tabel** (uraian kiri, nominal kanan, `tabular-nums`, nominal tidak dibungkus). NN/g: untuk isi bertele-tele
  hanya ±2 kolom yang terbaca di layar HP [nng-mobiletables] [V].
- **Isi paket** sebagai daftar yang bisa dibuka ("Lihat 12 isi paket"). NN/g: akordeon menghemat ruang di HP, tetapi jangan
  menggulir halaman otomatis saat dibuka karena terasa seperti pindah halaman [nng-accordion] [V]. GOV.UK: jangan sembunyikan
  informasi yang dibutuhkan sebagian besar pengguna [govuk-details] [V], jadi 3–5 isi pertama tampil, sisanya di balik "Lihat semua".
- **Baris terhitung dijelaskan dengan bahasa awam** ("120 tamu di atas 500 yang termasuk paket"), tanpa istilah "kelipatan"/"DPP".
- **Tanpa kolom menempel** di HP; total di kartu atas sudah cukup. NN/g: elemen menempel di HP memakan rasio konten [nng-sticky] [V].
- **Keluarga:** halaman yang sama **tanpa kolom nominal sama sekali** dan tanpa kartu total (bukan "Rp•••"); varian tersembunyi Carbon
  [carbon-disabled] [V]. Catatan pendek "Rincian biaya dapat dilihat pengantin" mencegah kebingungan; ini **hipotesis**, uji dengan
  keluarga sungguhan karena catatan itu juga membuka bahwa ada angka.
- Target sentuh ≥ 44 px, badan 16 px, sesuai DESIGN §4.4 dan T4.

## 6. PDF penawaran

### 6.1 Apa yang lazim di penawaran WO Indonesia (contoh nyata)

| Sumber | Yang terlihat |
|---|---|
| Gets Wedding Organizer, paket hotel Bandung (PDF 4 hlm., 2022) [wo-gets] [V] | Kop (alamat, kontak, email) **di setiap halaman**; judul "PENAWARAN PAKET PERNIKAHAN"; sapaan ke calon pengantin; satu "ESTIMASI HARGA" paket; isi dikelompokkan per kategori (katering, dekorasi, dokumentasi, MUA, hiburan, MC, WO) dengan nama vendor dalam kurung, **tanpa harga per isi**; bagian **BONUS**; penutup "Demikian disampaikan…"; nama + nomor kontak; halaman **TERM CONDITION**: bayar 3 tahap 25/50/25, DP hangus bila batal sepihak, tambahan mengikuti harga vendor |
| Wedding Organizer 92 Karawang (PDF 11 hlm., 2017) [wo-92] [V] | Proposal kerja sama: daftar fasilitas, alur kerja, termin 50% di awal, 30% H-2, 20% H-1, rekening |
| Alissha, All-in Package (PDF 22 hlm., 2024) [wo-alissha] [V] | Tabel harga per tingkat tamu (100–500 pax) per venue, masa berlaku harga, harga mengikat setelah DP |
| Ringkasan mesin pencari atas contoh WO lain [wo-search] [S] | Harga paket per 500 tamu + tanda jadi; tambahan tamu dihitung per orang di atas kuota paket |

Struktur surat penawaran resmi di Indonesia: kop surat (logo, nama, alamat, kontak), kepala surat (tanggal, nomor, lampiran, perihal,
tujuan), pembuka, isi (harga, kualitas, syarat bayar), penutup [accurate-surat] [V~]; artikel lain menambahkan "Hormat kami" + nama,
jabatan, tanda tangan, stempel, dan masa berlaku [surat-search] [S].

### 6.2 Tata letak yang dianjurkan (A4 tegak)

**Halaman 1: ringkasan satu halaman** (agar orang tua yang hanya membaca halaman pertama melihat seluruh kesepakatan):
1. Kop WO: logo, nama, alamat, WhatsApp (rekening di halaman termin).
2. Judul "Penawaran" · nomor `MO-PNW-261119-001` · **Revisi 2** · tanggal · **berlaku s.d.** (tebal).
3. "Untuk: Citra & Bima" + kontak + PIC WO.
4. **Acara**: tabel kecil (acara, hari/tanggal, venue, tata letak, sesi × tamu).
5. **Ringkasan biaya**: Paket · Tambahan tamu · Item tambahan · Potongan · PPN (bila PKP) · **Nilai kontrak** (angka terbesar di
   halaman).
6. **Termin**: label, persen, nominal, jatuh tempo (relatif → tanggal bila sudah ada).

**Halaman 2 dst.:**
7. **Rincian**: paket dengan isi paket berpoin per kategori (tanpa harga per isi, sesuai kebiasaan [wo-gets]), item dengan
   "jumlah × harga satuan", bonus berlabel "Bonus", potongan, PPN, total. Tajuk tabel diulang di halaman baru; satu baris tidak
   terpotong; blok total tidak terpisah dari baris terakhir.
8. **Pemasok yang dibayar langsung oleh klien**, tanpa harga, dengan kalimat "dibayar langsung oleh klien ke pemasok, tidak termasuk
   nilai kontrak".
9. **Syarat & ketentuan** bernomor (DP, pembatalan, penundaan, batas final jumlah tamu, masa berlaku).
10. Penutup singkat ("Demikian penawaran ini kami sampaikan…") dan **blok tanda tangan dua kolom**: "Hormat kami" (nama, jabatan WO)
    dan "Disetujui oleh" (nama pengantin, tanggal, tanda tangan). PLAN §5.3 menyimpan foto PDF bertanda tangan sebagai lampiran saat
    Diterima, jadi ruang tanda tangan diperlukan.
11. Kaki setiap halaman: "Halaman n dari m · MO-PNW-261119-001 Rev. 2 · Dibuat dengan ElProof".

- Proposify (data vendor dari 2,6 juta proposal): proposal pendek cenderung lebih sering ditutup (±50% untuk 5 halaman, ±35% untuk
  30 halaman), dan tiga bagian yang dianggap wajib oleh responden survei mereka adalah harga, syarat & ketentuan, dan tanda tangan
  [proposify-len] [V] (data vendor, tanpa uji independen: [S] untuk kesimpulannya).
- Tipografi: Plus Jakarta Sans tertanam (PLAN §5.5), badan ±10–11 pt agar terbaca orang tua saat dicetak [?], angka rata kanan,
  format "Rp1.500.000" dan "Sabtu, 14 Maret 2027" (DESIGN §7). Watermark "DRAF" hanya di pratinjau.

**Alternatif.** Format surat penuh (Nomor/Lampiran/Perihal/Kepada Yth./Dengan hormat) di halaman 1 lalu rincian sebagai lampiran:
paling "resmi", tetapi ringkasan biaya terdorong ke halaman 2. Bisa ditawarkan sebagai pilihan kelak, bukan bawaan.

## 7. Keadaan kosong untuk katalog dan penyunting

- **Pengaturan › Paket, belum ada paket:** satu kalimat ("Paket adalah produk yang Anda tawarkan ke klien, dengan isi dan aturan tamu.")
  + [Buat paket]. Carbon: elemen kosong rata kiri, aksi utama di bawah teks, tautan sekunder ke dokumentasi; untuk pemakaian pertama
  boleh dilengkapi dokumentasi singkat, onboarding, atau **konten contoh** [carbon-empty] [V]. Opsi: [Pakai contoh paket] yang membuat
  satu paket contoh yang bisa diubah; ini menulis data ke katalog WO, jadi perlu keputusan pemilik produk.
- **Penyunting rincian kosong:** "Belum ada rincian. Mulai dari paket di katalog atau tambah item." + [Tambah paket] [Tambah item].
  Bila katalog kosong: pengguna dengan `package.manage` melihat tautan "Buat paket di Pengaturan › Paket"; yang lain hanya [Tambah item].
- **Acara kosong:** "Tambahkan acara (Akad, Resepsi…) agar tamu dan kapasitas venue bisa dicek." + [Tambah acara].
- **Penawaran, daftar kosong:** satu kalimat + [Buat penawaran]; kosong karena filter: Reset filter (DESIGN §6).

## 8. Bukti lemah, bertentangan, atau belum terverifikasi

1. **Treegrid untuk paket bersarang.** APG (19 Nov 2026) menyajikan treegrid tanpa peringatan [apg-treegrid] [V~]; Roselli (2019)
   menyebut dukungannya buruk dan ada peringatan di halaman APG saat itu [roselli-expando] [S]. Dipilih tabel + tombol disclosure;
   uji dengan NVDA dan TalkBack.
2. **Ambang ringkasan galat.** GOV.UK: selalu tampilkan ringkasan galat, bahkan untuk satu galat [govuk-errsum] [V]; DESIGN §6:
   ringkasan bila ≥ 3. Di penyunting rincian yang panjang, galat bisa berada di luar layar, jadi usulan: ringkasan juga muncul bila
   ada galat di luar viewport. Perlu keputusan pemilik desain.
3. **Simpan otomatis.** GitLab menolaknya untuk data keuangan [gitlab-saving] [V]; SAP Fiori menyimpan draf otomatis di server
   [sap-footer] [S]. Dipilih simpan eksplisit; draf lokal opsional.
4. **Susun ulang dengan papan ketik.** Atlassian tidak menganjurkan panah sebagai alternatif utama [atl-dnd-a11y] [V]; Primer dan
   Salesforce menyediakan mode seret papan ketik [primer-dnd] [V] [sf-dnd-search] [S]; Chrome menerima keduanya dengan catatan bentrok
   dengan mode baca pembaca layar [chrome-dnd] [V]. Dipilih menu.
5. **Konflik versi:** tidak ada pedoman sistem desain atau riset yang kuat; hanya praktik vendor/forum [conflict-search] [S].
6. **Bandingkan revisi:** hanya dokumentasi produk (Confluence) [conf-history] [V~]; tidak ada riset pengguna.
7. **Total menempel di penyunting:** tidak ada riset primer khusus; disusun dari konvensi bilah simpan Shopify/SAP dan pedoman header
   menempel NN/g [shopify-savebar] [sap-footer] [nng-sticky].
8. **Panjang proposal dan tingkat tutup:** data Proposify sendiri, tidak diuji independen [proposify-len].
9. **Ukuran huruf PDF untuk orang tua (10–11 pt):** tidak menemukan sumber primer [?].
10. **Catatan untuk Keluarga** ("Rincian biaya dapat dilihat pengantin"): hipotesis, perlu uji.
11. **Bea meterai.** Penawaran yang ditandatangani kedua pihak bisa diperlakukan sebagai surat perjanjian, yang menurut UU 10/2020
    terutang bea meterai Rp10.000 saat ditandatangani [meterai-search] [S] `[perlu konfirmasi]` ke konsultan pajak. Dampaknya ke
    desain: perlu tidaknya ruang meterai di blok "Disetujui oleh". Riset ini tidak menambahkannya.
12. **HoneyBook:** halaman bantuan paket (harga paket di judul, harga item dihapus) hanya terbaca lewat ringkasan mesin pencari; tautan
    langsung 404 [hb-search] [?].

## 9. Sumber (semua diakses 19 Nov 2026)

**Standar & pola aksesibilitas**
- [wcag-257] WCAG 2.2 Understanding 2.5.7 Dragging Movements: https://www.w3.org/WAI/WCAG22/Understanding/dragging-movements.html [V~]
- [wcag-413] WCAG 2.2 Understanding 4.1.3 Status Messages: https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html [V]
- [wcag-334] WCAG 2.2 Understanding 3.3.4 Error Prevention (Legal, Financial, Data): https://www.w3.org/WAI/WCAG22/Understanding/error-prevention-legal-financial-data.html [V]
- [apg-grid] WAI-ARIA APG Grid Pattern: https://www.w3.org/WAI/ARIA/apg/patterns/grid/ [V]; contoh data grid: https://www.w3.org/WAI/ARIA/apg/patterns/grid/examples/data-grids/ [V]
- [apg-treegrid] WAI-ARIA APG Treegrid Pattern: https://www.w3.org/WAI/ARIA/apg/patterns/treegrid/ [V~]
- [apg-spin] WAI-ARIA APG Spinbutton Pattern: https://www.w3.org/WAI/ARIA/apg/patterns/spinbutton/ [V] (dibaca; tidak dipakai karena stepper tidak dianjurkan, §1.4)
- [roselli-expando] Adrian Roselli, "Table with Expando Rows" (Sep 2019) + komentar: https://adrianroselli.com/2019/09/table-with-expando-rows.html [V] (pendapat ahli: [S])
- [chrome-dnd] Chrome for Developers, Rearrange items with drag and drop: https://developer.chrome.com/docs/modern-web-guidance/guides/ui-behaviors/drag-and-drop [V]
- [primer-dnd] GitHub Primer, Drag and drop (accessibility): https://primer.style/accessibility/patterns/drag-and-drop [V]

**Sistem desain**
- [atl-dnd] Atlassian, Pragmatic drag and drop, design guidelines: https://atlassian.design/components/pragmatic-drag-and-drop/design-guidelines [V]
- [atl-dnd-a11y] Atlassian, Pragmatic drag and drop, accessibility guidelines: https://atlassian.design/components/pragmatic-drag-and-drop/accessibility-guidelines [V]
- [carbon-readonly] IBM Carbon, Read-only states pattern: https://www.carbondesignsystem.com/patterns/read-only-states-pattern/ [V]
- [carbon-disabled] IBM Carbon, Disabled states pattern: https://www.carbondesignsystem.com/patterns/disabled-states/ [V]
- [carbon-toggletip] IBM Carbon, Toggletip usage: https://www.carbondesignsystem.com/components/toggletip/usage/ [V]
- [carbon-empty] IBM Carbon, Empty states pattern: https://www.carbondesignsystem.com/patterns/empty-states-pattern/ [V]
- [carbon-notif] IBM Carbon, Notification pattern: https://www.carbondesignsystem.com/patterns/notification-pattern/ [V] (inline vs toast; mendukung DESIGN T10)
- [gitlab-saving] GitLab Pajamas, Saving and feedback (diperbarui 19 Agu 2026): https://design.gitlab.com/patterns/saving-and-feedback [V]
- [govuk-textinput] GOV.UK Design System, Text input (angka, awalan/akhiran, lebar): https://design-system.service.gov.uk/components/text-input/ [V]
- [govuk-numbers] GOV.UK Technology blog, "Why the GOV.UK Design System team changed the input type for numbers" (24 Feb 2020): https://technology.blog.gov.uk/2020/02/24/why-the-gov-uk-design-system-team-changed-the-input-type-for-numbers/ [V]
- [govuk-errmsg] GOV.UK Design System, Error message: https://design-system.service.gov.uk/components/error-message/ [V]
- [govuk-errsum] GOV.UK Design System, Error summary: https://design-system.service.gov.uk/components/error-summary/ [V]
- [govuk-charcount] GOV.UK Design System, Character count: https://design-system.service.gov.uk/components/character-count/ [V]
- [govuk-details] GOV.UK Design System, Details: https://design-system.service.gov.uk/components/details/ [V]
- [govuk-check] GOV.UK Design System, Check answers pattern: https://design-system.service.gov.uk/patterns/check-answers/ [V]
- [shopify-savebar] Shopify App Home, Save bar: https://shopify.dev/docs/api/app-home/latest/app-bridge-web-components/save-bar [V~]; pedoman lama Polaris (usang): https://polaris-react.shopify.com/components/deprecated/contextual-save-bar [S]
- [sap-footer] SAP Fiori, Object Page – Footer Bar & Draft Handling (ringkasan mesin pencari; halaman memblokir unduhan): https://www.sap.com/design-system/fiori-design-web/v1-142/discover/frameworks/sap-fiori-elements/object-page/object-page-footer-bar-sap-fiori-elements · https://www.sap.com/design-system/fiori-design-web/v1-108/foundations/best-practices/global-patterns/object-handling/draft-handling [S]

**Riset UX**
- [nng-datatables] NN/g, "Data Tables: Four Major User Tasks" (3 Apr 2022): https://www.nngroup.com/articles/data-tables/ [V]
- [nng-mobiletables] NN/g, "Mobile Tables" (17 Sep 2017): https://www.nngroup.com/articles/mobile-tables/ [V]
- [nng-sticky] NN/g, "Sticky Headers" (4 Apr 2021): https://www.nngroup.com/articles/sticky-headers/ [V]
- [nng-dnd] NN/g, "Drag–and–Drop: How to Design for Ease of Use" (23 Feb 2020): https://www.nngroup.com/articles/drag-drop/ [V]
- [nng-steppers] NN/g, "Design Guidelines for Input Steppers" (11 Nov 2018): https://www.nngroup.com/articles/input-steppers/ [V]
- [nng-changeblind] NN/g, "Change Blindness in UX" (23 Sep 2018): https://www.nngroup.com/articles/change-blindness-definition/ [V]
- [nng-accordion] NN/g, "Accordions on Mobile" (31 Mei 2015): https://www.nngroup.com/articles/mobile-accordions/ [V]
- [nng-form-errors] NN/g, "10 Design Guidelines for Reporting Errors in Forms" (3 Feb 2019, ditinjau 12 Des 2024): https://www.nngroup.com/articles/errors-forms-design-guidelines/ [V]
- [baymard-payux] Baymard, Payment UX: https://baymard.com/learn/payment-ux [V~]

**Produk sejenis**
- [stripe-quotes] Stripe Docs, Quotes overview (status, nomor + revisi): https://docs.stripe.com/quotes/overview [V]
- [odoo-quote] Odoo 18, Create quotations (section, note, optional products): https://www.odoo.com/documentation/18.0/applications/sales/sales/sales_quotations/create_quotations.html [V]
- [sf-cpq-bundle] Trailhead, How bundled quote lines behave (Salesforce CPQ): https://trailhead.salesforce.com/content/learn/modules/cpq-product-configuration/how-bundled-quote-lines-behave [V~]
- [sf-cpq-search] Ringkasan pencarian "Visualize Product Hierarchy" (Trailhead Control the configuration experience; Packt): https://trailhead.salesforce.com/content/learn/modules/cpq-product-configuration/control-the-configuration-experience [S]
- [sf-dnd-search] Salesforce UX, "4 major patterns for accessible drag and drop" (ringkasan pencarian): https://medium.com/salesforce-ux/4-major-patterns-for-accessible-drag-and-drop-1d43f64ebf09 [S]
- [shopify-draft] Shopify Help, Creating draft orders / discounts (ringkasan pencarian): https://help.shopify.com/en/manual/orders/create-orders/create-draft [S]
- [zoho-est] Zoho Invoice, Estimate statuses: https://www.zoho.com/invoice/kb/invoices/estimates-invoices-status.html [S]
- [conf-history] Atlassian Confluence, Page History and Page Comparison Views: https://confluence.atlassian.com/doc/page-history-and-page-comparison-views-139379.html [V~]
- [hb-search] HoneyBook Help, Creating a package template (404 saat dibuka): https://help.honeybook.com/en/articles/2209063-creating-a-package-template [?]
- [proposify-len] Proposify, proposal length vs close rate (Mar 2021): https://www.proposify.com/blog/proposal-length-close-rate [V] (data vendor)
- [conflict-search] Praktik konflik edit: https://www.drupal.org/project/drupal/issues/106953 · https://support.convert.com/hc/en-us/articles/edit-locking-version-conflict-prevention-in-the-visual-editor · https://docs.mendix.com/refguide/optimistic-locking/ [S]

**Indonesia**
- [wo-gets] Gets Wedding Organizer, Penawaran Paket Pernikahan Grand Cordela Hotel (PDF, dibuat 9 Mar 2022): https://alexandra.bridestory.com/raw/upload/assets/penawaran-wedding-package-cordela-HpTOLfN_g.pdf [V]
- [wo-92] Wedding Organizer 92 Karawang, Proposal penawaran kerja sama (PDF, 27 Okt 2017): https://alexandra.bridestory.com/raw/upload/assets/proposal-penawaran-kerjasama-wo-92-BJ0UgApfu.pdf [V]
- [wo-alissha] Alissha, All-in Package (PDF, 8 Mei 2024): https://alexandra.bridestory.com/raw/upload/assets/alissha-all-in-package_may2024-R1Kjx0m8c.pdf [V]
- [wo-search] Ringkasan pencarian contoh penawaran WO (Scribd, PDFCoffee; tidak dibuka): https://www.scribd.com/document/625107297/Surat-Penawaran-Aatreya-WO-Kuningan [S]
- [accurate-surat] Accurate, Contoh surat penawaran (6 Agu 2024, diperbarui 11 Agu 2025): https://accurate.id/bisnis-ukm/contoh-surat-penawaran/ [V~]
- [surat-search] Ringkasan pencarian struktur surat penawaran (kantorku.id, majoo.id, cake.me): https://kantorku.id/blog/contoh-surat-niaga/ [S]
- [meterai-search] UU No. 10 Tahun 2020 tentang Bea Meterai (ringkasan pencarian; teks: JDIH Setkab): https://jdih.setkab.go.id/PUUdoc/176264/UU_Nomor_10_Tahun_2020.pdf · https://pajak.go.id/id/bea-meterai [S]
