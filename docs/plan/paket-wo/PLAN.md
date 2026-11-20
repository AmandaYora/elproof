# PLAN: Paket WO, acara & rincian, portal, dan Penawaran

Dasar:
- [pemahaman.md](pemahaman.md): keputusan pemilik produk beserta kutipannya (#1–#27 di bawah).
- [DESAIN.md](DESAIN.md): alur penggunaan, struktur menu, tata letak, interaksi, keadaan, teks layar, dan PDF untuk semua layar di PLAN ini.
- Riset praktik WO di Indonesia, riset perangkat lunak sejenis, dan tinjauan terhadap kode yang ada (§10).

## 0. Ringkasan

ElProof sekarang hanya mengenal satu bentuk bisnis. Klien membayar WO dengan satu angka (`projects.contract_value`), lalu WO membayar vendor dan satu venue. PLAN ini mengganti dasar itu dengan tiga hal yang dipakai **bersama oleh Penawaran dan proyek**:

| Bagian | Menjawab | Contoh |
|---|---|---|
| **Acara** | Kapan, di mana, untuk berapa orang | Akad (Jumat, Masjid Al-Ikhlas, duduk, 1 sesi 120 tamu); Resepsi (Sabtu, Gedung Melati, berdiri, 2 sesi × 550 tamu) |
| **Kerja sama pemasok** | Siapa yang memasok, siapa yang membayar | Katering Rasa (dibayar WO, biaya rekanan Rp85 jt); Fotografer Lensa (dibayar klien langsung) |
| **Rincian** | Apa yang ditagih ke klien | Paket Resepsi 500 pax, tambahan tamu, item tambahan, bonus, potongan, PPN bila PKP |

Tiga bentuk bisnis WO lahir dari tanda **dibayar WO / dibayar klien langsung** pada kerja sama pemasok, tanpa jenis proyek terpisah. Klien selalu bertransaksi dengan WO. Harga ke klien ditentukan WO, sedangkan biaya rekanan bersifat opsional, internal, dan hanya terlihat oleh pemegang izin margin.

Tahapan (§8), masing-masing bisa dipasang dan langsung dipakai:

| Tahap | Isi |
|---|---|
| 1 | Direktori & katalog |
| 2 | Acara, vendor & venue di proyek |
| 3 | Rincian & nilai kontrak |
| 4 | Vendor dibayar klien |
| 5 | Penawaran + PDF |

Sebelum Tahap 1 ada **riset desain UI/UX** untuk semua layar baru: alur penggunaan per peran, struktur menu, tata letak, interaksi, dan keadaan. Hasilnya (`DESAIN.md`) disetujui dulu (§8). Di layar, kerja sama pemasok selalu disebut **Vendor & venue** (butirnya vendor atau venue); "pemasok" hanya istilah teknis di PLAN ini. Karena ElProof adalah SaaS banyak WO, setiap data di PLAN ini milik satu WO dan dipisahkan dengan aturan §2.3.

ElProof belum diluncurkan dan datanya masih dummy (#16). Karena itu skema yang ada sekarang **dibuang, bukan ditambal**.

## 1. Model bisnis (bahasa layar)

| Istilah | Arti |
|---|---|
| **Acara** | Satu bagian pernikahan: Akad, Resepsi, Lamaran, Pengajian, Siraman, Ngunduh mantu, dan seterusnya. Daftarnya bisa diubah WO (#21). Punya tanggal, satu venue, tata letak, dan sesi |
| **Sesi** | Rentang waktu dalam satu acara, beserta jumlah tamunya. Acara tanpa pembagian cukup punya satu sesi |
| **Tamu (pax)** | Orang yang diperkirakan hadir, **bukan** jumlah undangan dan bukan porsi katering. Porsi ditulis di deskripsi isi paket |
| **Tata letak** | Berdiri atau duduk. Menentukan angka kapasitas venue yang dipakai |
| **Batas kapasitas** | Tamu **per sesi** tidak boleh melebihi kapasitas venue acara itu untuk tata letaknya (#20). Sistem menolak dengan pesan yang menyebut angkanya |
| **Paket** | Produk WO di katalog. Punya harga, acara yang dicakup, tamu yang sudah termasuk (opsional), aturan tambahan tamu, dan isi paket. Tingkat lain (300 / 500 / 1.000 pax) dibuat sebagai paket terpisah |
| **Isi paket** | Layanan dalam paket, mis. "Katering buffet 450 porsi + 6 gubukan". Tanpa harga ke klien |
| **Tambahan tamu** | `ceil((tamu − termasuk) / kelipatan) × harga per kelipatan`. Kelipatan 1 berarti per pax. Tamu dihitung dari total tamu acara terbesar yang dicakup paket. Di bawah jumlah termasuk, harga tetap (#5) |
| **Item** | Baris berharga di luar paket: jasa WO, item tambahan, atau **bonus** (harga 0, berlabel Bonus). Dihitung `jumlah × harga satuan` (mis. 300 pcs × Rp12.000) |
| **Potongan** | Nominal pengurang. Harga paket tetap seperti katalog (#3) |
| **PPN** | Hanya bila workspace PKP (#23): 12% × DPP 11/12, dihitung dari subtotal dan dibulatkan ke bawah ke rupiah penuh |
| **Nilai kontrak** | Subtotal (paket + tambahan tamu + item − potongan) + PPN. Selalu dihitung dan tidak pernah kurang dari 0 |
| **Kerja sama pemasok** | Satu vendor atau venue dalam proyek atau penawaran: dibayar WO atau dibayar klien langsung, biaya rekanan, progres, dan pembayaran. Satu kerja sama bisa menjadi pemasok beberapa baris rincian, mis. katering untuk isi paket sekaligus gubukan tambahan |
| **Biaya rekanan** | Yang WO bayar ke pemasok yang dibayar WO. Opsional dan hanya untuk pemegang izin margin (#22) |
| **Penawaran** | Acara, pemasok, dan rincian untuk calon klien, ditambah termin dan syarat & ketentuan. Punya revisi dan status |
| **Kesepakatan awal** | Revisi penawaran yang diterima. Selalu bisa dibuka dari proyek sebagai pembanding rincian sekarang |

Glosarium (`knowledge/DESIGN.md` §7) dan `apps/web/scripts/check-ui-terms.mjs`:
- **Paket** = produk WO. **Paket langganan** = langganan ElProof. "Paket" tanpa keterangan dilarang untuk langganan.
- "Paket harga vendor" diganti **Jenis acara**.

## 2. Arsitektur modul

### 2.1 Modul baru `offers` (layar: Paket & Penawaran)

| Modul | Memiliki tabel | Memakai |
|---|---|---|
| `offers` (baru) | `event_kinds`, `packages`, `package_event_kinds`, `package_components`, `quotations`, `quotation_events`, `quotation_event_sessions`, `quotation_suppliers`, `quotation_lines`, `quotation_terms`, `quotation_revisions`, `quotation_activity`, `quotation_counters` | `vendors/contracts` (validasi, harga acuan, kapasitas); port `StaffNameResolver` dan `WorkspaceProfile` yang diisi adaptor di `main.go` |
| `projects` (dirombak) | `projects`, `project_events`, `project_event_sessions`, `project_suppliers`, `project_lines`, `supplier_payments`, `supplier_bill_terms`, ditambah tabel yang sudah ada (tahapan, kendala, bukti, termin & pembayaran klien, aktivitas) | `offers` (port `OffersPort`, setter), `vendors` (sudah ada, setter), port `WorkspaceProfile` (status PKP saat proyek dibuat) |
| `vendors` | tetap | Port `OfferReferences` (setter) untuk mengecek rujukan sebelum hapus permanen |
| `platform` | `tenant_business_details` + `document_code`, `is_pkp`, `default_quote_validity_days`, `default_terms_text` | Port `EventKindSeeder` (setter, sama dengan `SetVendors`) untuk mengisi jenis acara bawaan saat WO mendaftar |

**Aturan ketergantungan (ADR-0049):**

- `offers` tidak pernah mengimpor `projects/contracts` maupun `staff/contracts`. Kalau mengimpor, akan terbentuk siklus import Go: `staff/application` sudah mengimpor `projects/contracts`, dan `projects/contracts` mengimpor `projects/application` (`projects/contracts/contracts.go:12`, `staff/application/staff_service.go:10`).
- `offers` hanya memakai `vendors/contracts` secara langsung. Nama staf diambil lewat port.
- `projects → offers` dan `vendors → offers` memakai port lokal yang diisi setter di `cmd/server/main.go`, polanya sama dengan `SetVenueResolver`. Urutan pembangunan: `projects` → `vendors` → `offers`, lalu `projectsModule.SetOffers(...)` dan `vendorsModule.SetOfferReferences(...)`.
- `platform` belum punya `contracts/`. Profil usaha WO (nama, logo, kontak, rekening, kode usaha, PKP, syarat & ketentuan bawaan) dibaca `offers` dan `projects` lewat port `WorkspaceProfile(ctx, tenantID)`, diisi adaptor di `main.go` dari `platform`. Port ini dipasang di Tahap 3 (PKP saat proyek dibuat) dan dipakai lagi di Tahap 5 (kop PDF); port nama staf dipasang di Tahap 5 (PIC penawaran). Tahap 1 tidak memerlukan keduanya: pengaturan penawaran dibaca dan ditulis di dalam `platform`. Arah sebaliknya (`platform` mengisi jenis acara bawaan WO baru) memakai setter `SetEventKinds`. Dengan begitu tidak ada siklus `platform ↔ offers`.

**Logika murni di `offers/domain`, satu tempat, dipakai Penawaran dan proyek:**

- `PackageRule`: salinan aturan paket (acara yang dicakup, tamu termasuk, kelipatan, harga per kelipatan).
- `ExtraGuests(rule, events)`.
- `LineTotals(lines, isPKP)`: subtotal, PPN, total, dan tolak bila hasilnya kurang dari 0.
- `CheckCapacity(events, venues)`.

Fungsi-fungsi ini tidak membaca database. `projects` memanggilnya lewat `OffersPort` dengan data miliknya sendiri. Jadi tidak ada logika domain yang terduplikasi, dan tidak ada tabel yang dibaca lintas modul.

### 2.2 Web

- Penyunting acara, pemasok, dan rincian dibuat **satu kali** di modul web `rincian`. Modul ini mengekspor komponennya lewat `index.ts` sebagai satu-satunya pintu publik, lalu dipakai oleh `offers`, `projects`, dan `client-portal` (tampilan baca).
- ADR-0049 mencatat pengecualian atas `ARCHITECTURE_OVERVIEW.md:44`: modul boleh mengimpor pintu publik `rincian`, tidak boleh isinya.
- Komponen di `shared/` tetap bebas domain.
- Rupa, susunan, dan perilaku semua layar baru mengikuti brief desain fitur ini (`docs/plan/paket-wo/DESAIN.md`, §8 Riset desain), yang dibangun di atas `knowledge/DESIGN.md`.
- **Total sementara saat mengetik** diambil dari server: `POST …/lines/preview` (penawaran dan proyek) menghitung rincian yang belum disimpan dengan fungsi `offers/domain` yang sama, tanpa menulis apa pun. Web memanggilnya dengan jeda 500 ms. Tidak ada rumus harga kedua di web.

### 2.3 Banyak WO dalam satu ElProof

ElProof adalah SaaS: banyak WO mendaftar, dan setiap WO punya klien, katalog, penawaran, dan proyeknya sendiri. Semua bagian PLAN ini mengikuti ADR-0004:

- **Setiap tabel baru** di §2.1 punya `tenant_id NOT NULL`, termasuk tabel anak (sesi, baris, revisi, aktivitas, penghitung nomor). `tenant_id` tidak pernah disimpulkan lewat join ke tabel induk.
- `tenant_id` **selalu** diambil dari sesi login, tidak pernah dari isian. Setiap kueri menyaring `tenant_id`, termasuk kueri tabel anak dan cache `contract_*`.
- **Kunci unik selalu memuat `tenant_id`:** nomor penawaran, `quotation_id` di proyek, nama jenis acara, nama paket. Dua WO boleh punya paket bernama sama dan nomor urut yang sama.
- **ID milik WO lain diperlakukan seperti tidak ada (404)**, bukan 403, supaya keberadaannya tidak bocor. Ini berlaku juga untuk rujukan di dalam isian: vendor/venue, staf PIC, paket, jenis acara, dan penawaran yang dikonversi wajib milik WO yang sama. `vendors/contracts` dan port staf selalu menerima `tenantID`.
- **Data bawaan per WO:** jenis acara bawaan diisi saat WO mendaftar (`SeedDefaultEventKinds`, idempoten, polanya sama dengan `SeedDefaultCategories`) dan lewat migrasi untuk WO yang sudah ada. Paket, kode usaha, PKP, syarat & ketentuan, dan penghitung nomor juga milik masing-masing WO.
- **PDF** hanya memuat identitas WO pemiliknya. Logo dan lampiran dibaca dari jalur penyimpanan WO itu (ADR-0006).
- **Portal klien** tetap dibatasi ke proyek klien itu di WO-nya (`client_scope.go`). Kontak di penawaran tidak pernah memberi akses portal.
- **Langganan:** penawaran dan paket ikut mode baca-saja saat langganan WO habis (ADR-0022). Kuota masa coba berlaku saat konversi ke proyek (§5.4).
- **Platform Console** tidak punya endpoint untuk membaca paket, penawaran, atau rincian WO.
- **Tes isolasi** (`apps/api/tests/isolation`) ditambah di setiap tahap, lihat §8.

## 3. Model data

### 3.1 Bentuk yang sama di Penawaran (`quotation_*`) dan proyek (`project_*`)

**Acara** (`*_events`):

| Kolom | Arti |
|---|---|
| `event_kind_id`, `kind_name` | Jenis acara dari `event_kinds`. Nama disalin, supaya mengganti nama jenis tidak mengubah dokumen lama |
| `event_date`, `start_time`, `end_time` | Di penawaran boleh kosong (perkiraan). Di proyek tanggal wajib |
| `venue_supplier_id` | Menunjuk satu kerja sama pemasok berjenis venue (nullable). Satu venue boleh dipakai beberapa acara, mis. akad dan resepsi di gedung yang sama |
| `layout` | `Standing` · `Seated` |
| `sort_order` | Urutan tampil |

**Sesi** (`*_event_sessions`): `label` ("Sesi 1"), `start_time`, `end_time`, `guests`. Total tamu acara = Σ tamu sesi.

**Kerja sama pemasok** (`*_suppliers`):

| Kolom | Arti |
|---|---|
| `supplier_kind`, `supplier_id`, `supplier_name` | `vendor` / `venue` + ID primitif + nama yang disalin. Nama disalin karena direktori bisa dihapus permanen (ADR-0018) |
| `category_id` | Disalin saat dipilih |
| `paid_by` | `WO` · `Client` (#12). **Terkunci** begitu ada pembayaran atau bukti yang belum dihapus (422) |
| `show_name` | Hanya berlaku untuk vendor yang dibayar WO. Bawaan `false` (#14). Venue dan pemasok yang dibayar klien **selalu** tampil, karena klien datang ke venue dan membayar pemasoknya sendiri |
| `capacity_standing`, `capacity_seated` | Hanya venue. Disalin dari direktori, bisa diperbarui lewat aksi "Perbarui dari direktori" |
| `agreed_cost` | Biaya rekanan. Hanya untuk dibayar WO, opsional, dan hanya untuk pemegang izin margin |
| Proyek saja: `engagement_status`, `booking_date`, `due_date`, `pic_staff_id`, `scope`, `notes` | Data keterlibatan dari `project_vendors` |

Tahapan vendor, kendala, bukti, dan pembayaran menunjuk `project_supplier_id`.

**Baris rincian** (`*_lines`):

| Kolom | Arti |
|---|---|
| `kind` | `Package` · `Component` · `ExtraGuests` · `Item` · `Discount` |
| `parent_line_id` | `Component` dan `ExtraGuests` menunjuk `Package`-nya. Menghapus paket ikut menghapus anak-anaknya, tetapi **tidak** menyentuh kerja sama pemasok beserta pembayarannya |
| `package_rule` (JSON) | Hanya `Package`: salinan aturan paket saat ditambahkan. Mengedit katalog tidak mengubah proyek yang sedang berjalan |
| `description`, `category_id`, `event_id` | `event_id` opsional: baris ini untuk acara mana |
| `quantity`, `unit`, `unit_price` | Hanya `Item`. `amount = quantity × unit_price` |
| `amount` | Nominal **positif**. Arah (tambah/kurang) ditentukan `kind`. `Component` = 0. `ExtraGuests` selalu dihitung, tidak bisa diketik; selisih nego masuk ke Potongan |
| `is_bonus` | `Item` dengan harga 0 dan label Bonus |
| `supplier_ref` | Kerja sama pemasok yang memasok baris ini (nullable). Hanya boleh menunjuk pemasok yang dibayar WO |
| `sort_order` | Urutan tampil |

Validasi:
- Jenis acara yang dicakup paket harus ada di daftar acara.
- Venue sebuah acara harus pemasok berjenis venue. Memilih venue dari direktori untuk sebuah acara otomatis membuat kerja sama venue itu (dibayar WO) bila belum ada, dalam transaksi yang sama. Mengganti venue tidak menghapus kerja sama lama yang sudah punya pembayaran; kerja sama itu tetap ada dan ditandai tidak dipakai acara mana pun.
- Isi paket selalu mengikuti paketnya, yang berarti selalu dibayar WO.

### 3.2 Proyek

- **Dihapus:**
  - `projects.package_name`, `venue`, `venue_id`, `venue_rental_price`, `venue_charge`, dan `contract_value` yang diketik;
  - tabel `project_vendors`, `vendor_payments`, `venue_payments`;
  - jenis bukti `projectVendor`, `payment`, `venuePayment`, diganti `projectSupplier` dan `supplierPayment`.
- **Ditambah di `projects`:**
  - `quotation_id`, dengan `UNIQUE (tenant_id, quotation_id)`;
  - `is_pkp` (disalin dari workspace saat proyek dibuat);
  - `contract_subtotal`, `contract_tax`, `contract_total`: cache yang diperbarui dengan `UPDATE … SET … = (SELECT SUM …)` dalam transaksi yang sama dengan setiap perubahan baris, tamu, hapus lunak, pulihkan, duplikat, atau konversi;
  - `lines_version`: versi rincian sendiri untuk `If-Match`, supaya mengubah rincian tidak membuat form proyek di tab lain gagal dengan 412.
- **Lokasi yang tampil** diturunkan dari venue tiap acara. Contoh: "Akad: Masjid Al-Ikhlas · Resepsi: Gedung Melati". Yang memakainya: beranda klien, riwayat vendor "Lihat Project", utang, dan kalender.
- **`supplier_payments`** per `project_supplier_id` menyimpan **`payer` sendiri** (`WO` / `Client`). Semua kueri kas, utang, dan dasbor memfilter kolom ini, bukan join ke kerja sama. Ini mencegah arti pembayaran lama berubah diam-diam.
- **`supplier_bill_terms`** berisi termin tagihan untuk pemasok yang dibayar klien (#15): label, nominal, dan jatuh tempo.
- **`client_payment_terms` dan `client_payments` tetap.** Bila jadwal termin berbeda dari nilai kontrak, respons perubahan rincian membawa `termsMismatch`, lalu kartu Jadwal Termin menawarkan **Hitung ulang termin**. Jadwal **tidak pernah** ditulis ulang diam-diam (ADR-0042).
- **Lebih bayar:** nilai kontrak boleh turun di bawah uang yang sudah masuk. Keadaan itu diberi label "Lebih bayar" (ADR-0027) dan diuji.
- **Margin** = subtotal (tanpa PPN, karena PPN milik negara) − Σ `agreed_cost` pemasok yang dibayar WO. Ada pemasok dibayar WO tanpa biaya → "Margin belum lengkap (n pemasok)". Tanpa biaya, pembayaran ke pemasok itu tidak dibatasi aturan lebih bayar dan utangnya ditandai belum lengkap.
- **Duplikat proyek (ADR-0014)** menyalin acara, pemasok, dan baris tanpa uang, kendala, bukti, dan `quotation_id`. Kapasitas dicek ulang.
- **Riwayat perubahan:** setiap perubahan baris atau tamu tercatat di `activity_log` dengan nilai lama dan baru.

### 3.3 Direktori (`vendors`) dan workspace (`platform`)

- **Venue:**
  - `capacity_standing` dan `capacity_seated`; minimal salah satu wajib dan ≥ 1 (#18, #20);
  - `rental_price` diganti `price_akad`, `price_resepsi`, `price_akad_resepsi` (#19);
  - `charge` tetap sebagai biaya tambahan opsional;
  - impor/ekspor xlsx venue ikut diperbarui.
- **Vendor:** `price_akad`, `price_resepsi` (sudah ada sejak migrasi 000042), dan `price_akad_resepsi`. Semua harga ini diberi label **Harga rekanan acuan**. Gunanya hanya untuk mengisi `agreed_cost` sesuai jenis acara, dan tidak pernah tampil ke klien. Untuk acara lain (lamaran, pengajian), biaya rekanan diketik.
- **Hapus permanen vendor/venue** juga mengecek rujukan di paket dan penawaran (port `OfferReferences`). Paket dinonaktifkan, tidak dihapus permanen, selama dirujuk penawaran. Vendor/venue nonaktif tidak bisa dipilih untuk baris baru.
- **Workspace:** di Pengaturan › Profil usaha, bagian Penawaran:
  - `document_code`: kode usaha untuk nomor dokumen, 2–5 huruf atau angka kapital (§5.1). Diisi otomatis saat WO mendaftar, dan untuk WO yang sudah ada saat pengaturan penawarannya pertama kali dibaca, dari huruf awal nama usaha: Melati Organizer → `MO`, Bunga Wedding Organizer → `BWO`, Lavanya → `LAV`. WO bisa mengubahnya. Tidak harus unik antar WO, karena nomor dokumen tidak pernah tampil di luar WO-nya;
  - `is_pkp` (bawaan `false`);
  - `default_quote_validity_days`: masa berlaku bawaan penawaran, bawaan 14 hari;
  - `default_terms_text` (syarat & ketentuan bawaan). Hal yang tidak termasuk penawaran ditulis WO di sini, tanpa isian terpisah (#27).

## 4. Aturan bisnis kunci

### 4.1 Kapasitas (#18, #20)

`CheckCapacity` menolak bila ada sesi dengan tamu melebihi kapasitas venue acaranya untuk tata letak acara itu. Contoh pesan: "Gedung Melati muat 600 orang berdiri, Sesi 1 Resepsi 650".

Pengecekan berjalan saat:
- tamu atau sesi diubah;
- venue ditambah, diganti, atau kapasitasnya diperbarui;
- tata letak diubah;
- konversi penawaran atau duplikat proyek.

Ketentuan lain:
- Bila venue tidak punya angka untuk tata letak yang dipilih, sistem menolak dengan "Isi kapasitas duduk Gedung Melati di Venue".
- Selama acara belum punya venue, tamu boleh diisi dan acara itu bertanda "Belum dicek ke kapasitas venue".
- Paket yang punya venue bawaan dicek terhadap kapasitas **terbesar** venue itu saat disimpan (paket belum punya tata letak), lalu dicek lagi terhadap tata letak acara saat dipakai. Contoh pesan: "Gedung Melati muat paling banyak 600 orang berdiri, tamu termasuk 650".

### 4.2 Harga paket dan tambahan tamu

- `ExtraGuests` dihitung ulang otomatis setiap kali tamu acara yang dicakup berubah. Dasarnya salinan `package_rule` di baris, bukan katalog yang mungkin sudah diedit.
- Paket berbasis pax di proyek yang belum punya tamu: tambahan tamu = 0, dengan tanda.
- Kelipatan wajib > 0 bila harga per kelipatan diisi.

### 4.3 Siapa melihat apa (tim WO, #22)

| Data | Izin |
|---|---|
| Rincian, nilai kontrak, uang masuk klien | Seperti sekarang (`finance.record` untuk mencatat) |
| `agreed_cost`, margin, utang ke rekanan | `finance.margin.view` |
| Pembayaran ke pemasok yang dibayar WO (lihat dan catat) | `finance.margin.view`. Wedding Planner tidak lagi mencatatnya |
| Pembayaran klien ke pemasok yang dibayar klien | `finance.record` |
| Perkiraan biaya di isi paket katalog | `finance.margin.view` |

Respons API mengosongkan kolom-kolom itu bila izinnya tidak ada. Tes memastikan kolom itu tidak muncul.

### 4.4 Portal klien: satu kebijakan untuk semua rute

Fungsi-fungsi redaksi yang tersebar (`client_view.go`, `ClientCanSeeEvidence`, potongan per handler di `project_endpoints.go:384`, `home_endpoint.go:125`, `issue_endpoints.go:70`) diganti `projects/domain.PortalPolicy`, dengan satu metode per sumber daya: `Project`, `Home`, `Event`, `Supplier`, `Line`, `Issue`, `Evidence`, `SupplierPayment`, `ClientPayment`, `Term`.

| Data | Aturan |
|---|---|
| Acara, sesi, tamu, venue (nama, alamat, foto) | `portal.timeline.view` |
| Nama baris rincian dan isi paket | `portal.vendor.view` |
| Nominal baris, nilai kontrak, PPN, termin, uang masuk | `portal.finance.view` (Keluarga tidak) |
| Harga per isi paket | Tidak pernah |
| Nama vendor yang dibayar WO | Hanya bila `show_name`, **termasuk** di kendala, tahapan, dan nama bukti |
| `agreed_cost`, pembayaran WO ke rekanan beserta buktinya, catatan internal | Tidak pernah |
| Tagihan dan pembayaran pemasok yang dibayar klien | `portal.finance.view`. Kirim bukti dengan `portal.finance.submit` |
| Tahapan vendor dan buktinya | `portal.vendor.view`, berlabel nama layanan bila nama disembunyikan. Bukti tahapan pemasok yang dibayar WO bawaannya **tidak** tampil ke klien, kecuali WO menandai unggahan itu "tampilkan ke klien" (foto nota bisa membuka nama atau harga rekanan) |
| Kendala | `is_client_visible`, lalu disaring kebijakan nama |

**Tes rute:** membaca daftar rute di `cmd/server/client_scope.go`, memanggil setiap GET sebagai Pengantin dan Keluarga, lalu memastikan tidak ada `agreedCost`, pembayaran WO, atau nama vendor yang disembunyikan di JSON mana pun. Tes ini juga mencakup `/evidence/{id}/file|link` dan lampiran venue.

## 5. Penawaran (Tahap 5)

### 5.1 Isi
- **Nomor:** `KODE-PNW-YYMMDD-NNN`, contoh `MO-PNW-261119-001`:
  - `KODE` = kode usaha WO (§3.3), supaya nomor di PDF langsung menunjukkan WO-nya;
  - `PNW` = jenis dokumen (penawaran). Dokumen lain kelak memakai kode jenisnya sendiri;
  - `YYMMDD` = tanggal penawaran dibuat (WIB), mis. 261119 = 19 Nov 2026. Urutan tahun-bulan-tanggal membuat nomor terurut menurut waktu saat diurutkan sebagai teks;
  - `NNN` = urutan penawaran WO itu pada tanggal itu, 001–999.
- **Panjang** 17–20 karakter (kode 2–5 huruf), cukup pendek untuk diucapkan lewat telepon atau diketik di WhatsApp, dan cukup lengkap untuk dicari.
- **Penghitung** milik tiap WO per tanggal: `quotation_counters (tenant_id, doc_date, last_seq)` dengan `SELECT … FOR UPDATE`. `quotations` punya `UNIQUE (tenant_id, number)`. Dua WO yang membuat penawaran di hari yang sama sama-sama mendapat `001`. Penawaran ke-1.000 dalam sehari ditolak dengan pesan yang jelas.
- **Nomor dibekukan saat penawaran dibuat.** Mengubah kode usaha tidak mengubah nomor lama. Revisi tidak mengubah nomor; PDF menulis "Revisi 2" di samping nomor. "Salin jadi penawaran baru" mendapat nomor baru.
- **Kepala** juga memuat nama pengantin, kontak, PIC, `valid_until` (akhir hari WIB; bawaan = tanggal dibuat + `default_quote_validity_days`), catatan, dan `alternative_of_id` (§5.3).
- **Membuat penawaran** cukup dengan nama pasangan, WhatsApp opsional, dan PIC. Nomor terbentuk saat itu juga; isi lain dilengkapi di halaman penawaran.
- **Isi:** acara, sesi, pemasok, dan rincian (§3.1).
- **Termin:** label, persen atau nominal, dan jatuh tempo relatif terhadap tanggal diterima atau tanggal acara. Bawaan 30/40/30, sama dengan termin bawaan proyek sekarang.
- **Syarat & ketentuan:** teks bawaan workspace yang bisa diubah per penawaran, mis. DP hangus, pembatalan, penundaan, dan batas final jumlah tamu.

### 5.2 Revisi (#9)
- **Pratinjau PDF** berwatermark "DRAF" dan tidak membuat revisi.
- **Buat revisi untuk klien** (`POST /quotations/{id}/revisions`) membekukan seluruh isi dan identitas WO saat itu, termasuk kunci file logo yang tidak berubah, ke `snapshot` JSON. Isi yang identik tidak menambah revisi (`UNIQUE (quotation_id, content_hash)`), sehingga dua klik bersamaan tetap menghasilkan satu revisi.
- **Unduh PDF revisi** (`GET …/revisions/{n}/pdf`) selalu dibuat ulang dari snapshot, jadi Revisi 2 tetap sama persis bulan depan.

### 5.3 Status
Status yang disimpan: `Draft` · `Accepted` · `Rejected` · `Superseded`. **Kedaluwarsa** dihitung saat dibaca: `Draft` yang `valid_until`-nya lewat.

| Dari → ke | Syarat |
|---|---|
| Draf → Diterima | Pilih revisi. Salinan kerja harus sama dengan suatu revisi (kalau berbeda, buat revisi dulu). Yang kedaluwarsa diperpanjang di dialog yang sama, tercatat sebagai dua aktivitas (diperpanjang, diterima). Dicatat: tanggal, saluran (WhatsApp / tatap muka / email / lainnya), dan lampiran opsional (foto PDF bertanda tangan) |
| Draf → Ditolak | Alasan opsional |
| Ditolak → Draf | Buka lagi |
| Diterima → Draf | **Batalkan terima**, hanya selama belum dijadikan proyek |
| Draf / Diterima → Digantikan | Otomatis saat WO memakai "Salin jadi penawaran baru" dari penawaran itu, atau saat salah satu alternatifnya diterima. Penawaran yang digantikan tidak bisa dijadikan proyek |
| Diterima → (dijadikan proyek) | Penawaran menjadi baca-saja. Perubahan berikutnya terjadi di proyek, bukan lewat penawaran baru |

**Penawaran alternatif (#26).** "Buat alternatif" menyalin isi ke penawaran baru bernomor sendiri dengan `alternative_of_id` menunjuk penawaran asal (kelompoknya = penawaran asal beserta semua alternatifnya). Semua anggota kelompok tetap Draf dan bisa direvisi serta dibagikan sendiri-sendiri; setiap PDF hanya memuat satu pilihan. Saat satu anggota diterima, anggota lain yang masih Draf menjadi Digantikan dalam transaksi yang sama, dan dialog terima menyebutkannya lebih dulu. Batalkan terima tidak menghidupkan kembali yang sudah Digantikan.

Semua perubahan bersifat eksplisit (`setQuotationStatus(id, status)`) dan tercatat di `quotation_activity`: siapa membuat revisi, siapa menerima atau menolak, kapan.

### 5.4 Jadikan proyek (#8)

`POST /api/v1/projects/from-quotation/{id}` adalah milik `projects`.

- **Isian pelengkap:** tanggal pasti tiap acara, tanggal mulai persiapan, PIC, dan PIC Sales (bawaan: PIC penawaran). Opsional: **tanda jadi yang sudah diterima**, yang dicatat sebagai `client_payment` pertama.
- **Dalam satu transaksi `projects`:**
  - buat proyek dengan `quotation_id` (unik);
  - salin acara, sesi, pemasok, baris, dan termin (nominalnya dihitung dari nilai kontrak);
  - buat Timeline dari Template Timeline Default;
  - cek ulang kapasitas;
  - berlakukan kuota masa coba (`trial_quota.go`).
- **Setelah itu:** panggil `offers.MarkConverted`.
- **Bila proyek dengan `quotation_id` itu sudah ada**, endpoint mengembalikan proyek itu dan memanggil ulang `MarkConverted`. Endpoint ini idempoten per penawaran, jadi tab kedua tidak membuat proyek kedua. Status "dijadikan proyek" di `offers` diperlakukan sebagai cache yang bisa diperbaiki.
- **Proyek yang kemudian dihapus permanen:** penawaran tetap tercatat sudah dijadikan proyek, dengan keterangan "Proyek sudah dihapus".
- **Kontak di penawaran tidak otomatis menjadi akun portal.**
- **Proyek juga bisa dibuat langsung tanpa penawaran** (§9.1). Penyunting yang dipakai sama.

### 5.5 PDF (#11)
- Dibuat di server Go dengan pustaka PDF Go murni, tanpa Chrome, karena RAM VPS 2 GB. Kandidat: `go-pdf/fpdf` atau `maroto v2`. Syarat lolos: penawaran 40 baris di bawah 30 MB dan di bawah 1 detik.
- Huruf Plus Jakarta Sans (OFL) ditanam di biner.
- **Isinya:**
  - kop WO (nama, logo, alamat, WhatsApp, rekening bila diisi);
  - nomor dan revisi;
  - masa berlaku;
  - acara (tanggal, venue, sesi, tamu);
  - rincian seperti tampilan Pengantin di portal, tanpa biaya rekanan;
  - pemasok yang dibayar klien langsung, tanpa harga;
  - PPN bila PKP;
  - termin;
  - syarat & ketentuan;
  - blok tanda tangan "Hormat kami" dan "Disetujui oleh" (tanpa kotak meterai).
- **Susunan:** halaman 1 meringkas seluruh kesepakatan (kop, nomor, acara, ringkasan biaya, nilai kontrak, termin); halaman berikutnya rincian, vendor yang dibayar klien, syarat & ketentuan, dan tanda tangan (`DESAIN.md` §6.12).
- **Kaki halaman:** "Halaman n dari m", nomor dan revisi, dan tulisan kecil "Dibuat dengan ElProof".
- Renderer ditaruh di `offers/infrastructure/pdf` sebagai utilitas teknis tanpa logika domain.

## 6. Izin

Nama izin mengikuti katalog (`access/domain/catalog.go`):

| Izin | Untuk | Bawaan |
|---|---|---|
| `package.manage` (baru) | Menambah, mengubah, menonaktifkan, dan menghapus paket dan jenis acara; menu Katalog › Paket dan Jenis acara. Melihat daftar paket dan jenis acara terbuka untuk semua tim, karena Sales memilih paket saat menyusun penawaran | Owner, Admin |
| `quotation.view` (baru) | Lihat penawaran | Owner, Admin, Sales |
| `quotation.manage` (baru) | Buat, ubah, revisi, terima/tolak | Owner, Admin, Sales |
| `project.create` (sudah ada) | Juga "Jadikan proyek" | |
| `finance.margin.view` (diperluas, #22) | §4.3 | Owner, Admin |

Cakupan `assigned` untuk penawaran berarti "PIC penawaran". Tidak ada izin portal baru.

## 7. Layar

| Layar | Isi |
|---|---|
| **Penawaran** (menu utama, sebelum Proyek) | Daftar dengan tab status (Draf, Kedaluwarsa, Diterima, Ditolak, Digantikan), saringan "Segera kedaluwarsa (≤ 7 hari)", alternatif sebagai baris anak. Halaman penawaran satu halaman berseksi: info, acara & sesi, vendor & venue, rincian, termin, syarat & ketentuan, riwayat revisi; panel Ringkasan biaya menempel di kanan; satu aksi utama per status (Buat revisi untuk klien / Bagikan PDF / Perpanjang / Jadikan proyek / Buka lagi). Halaman penuh **Jadikan proyek** |
| **Katalog › Paket** | Daftar paket (sort/filter di server); form paket halaman penuh dengan kotak "Contoh hitung": acara yang dicakup, harga, tamu termasuk, aturan tambahan, venue bawaan, isi paket |
| **Katalog › Jenis acara** | Daftar jenis acara bawaan dan tambahan WO |
| **Katalog › Jadwal bawaan, Kategori vendor** | Pindahan dari grup Templat, isinya tetap |
| **Pengaturan › Profil usaha** | Ditambah bagian Penawaran: kode usaha (dengan contoh nomor), PKP, masa berlaku bawaan, syarat & ketentuan bawaan |
| **Topbar** | Tombol "Baru ▾" (Penawaran, Proyek; sesuai izin) |
| **Proyek › Acara & rincian** (tab baru) | Acara, sesi, tamu, venue, tanda kapasitas; rincian; panel nilai kontrak / biaya rekanan / margin (berizin); tautan **Kesepakatan awal** |
| **Proyek › Vendor & venue** (isi baru) | Kerja sama vendor dan venue: dibayar oleh, progres, pembayaran ke rekanan (berizin), tagihan vendor yang dibayar klien |
| **Proyek › Keuangan** | Buku Klien, Vendor & venue (berizin margin), Dibayar klien langsung |
| **Venue** | Dua kapasitas dan tiga harga |
| **Portal › Jadwal** | Bagian Acara (tanggal, venue, peta, sesi) di atas jadwal tahapan |
| **Portal › Layanan** (pengganti tab Vendor) | Paket & rincian (nominal hanya untuk izin keuangan), lalu progres vendor |
| **Portal › Pembayaran** | Ditambah bagian Tagihan vendor: vendor yang dibayar klien, tagihan, dan kirim bukti |

Form memakai `useZodForm`. Daftar memakai `shared/data-table` dengan pengurutan dan filter di server. Sheet dan dialog dimuat saat diklik, dan `npm run budget` tetap hijau. Susunan, wadah (halaman, drawer, dialog), keadaan, dan teks tiap layar ditetapkan di `DESAIN.md`, bukan ditentukan saat mengode. Detail proyek tetap satu baris tab (7 tab): Ringkasan · Acara & rincian · Jadwal · Vendor & venue · Keuangan · Kendala · Dokumen. Portal tetap 5 tab bawah: Beranda · Jadwal · Pembayaran · Layanan · Lainnya.

## 8. Tahapan

Setiap tahap dikerjakan di branch tugas. Tahap baru digabung setelah `go test ./...`, `npx tsc -b && npm run check:terms && npm run build && npm run budget`, dan `npm run e2e` hijau. Tahap 2 dan 3 boleh dipasang bersamaan, tetapi ditinjau terpisah.

Setiap tahap juga baru selesai bila:
- **tes isolasi** membuktikan WO B tidak bisa melihat, mengubah, menghapus, atau memakai data WO A yang dibuat tahap itu (daftar, detail, ubah, hapus, rujukan di isian, unduhan berkas/PDF), dan klien WO A tidak bisa menjangkau WO B;
- **alur dan layar tahap itu sesuai `DESAIN.md`**: alur tahap itu dijalankan dari awal sampai selesai di aplikasi oleh peran yang memakainya, dan tangkapan layar desktop (≥ 1280 px) dan HP (375 px) menunjukkan tata letak serta keadaan kosong, memuat, dan galat.

**Riset desain (sebelum Tahap 1, sekali untuk semua layar)**

Tidak ada kode layar yang ditulis sebelum brief desain disetujui pemilik produk.

Yang dirancang adalah **pengalaman pakai (UX) dan antarmuka (UI): alur penggunaan, susunan layar, dan perilakunya**, bukan warna. Warna, huruf, dan komponen sudah ditetapkan `knowledge/DESIGN.md` dan dipakai apa adanya.

**Yang dirancang:**

| Lapisan | Pertanyaan yang dijawab | Hasil di `DESAIN.md` |
|---|---|---|
| **Alur penggunaan** | Langkah apa yang dilalui tiap peran dari awal sampai tugasnya selesai, di mana ia bisa salah, dan bagaimana ia kembali ke jalur | Diagram alur per tugas, lengkap dengan titik keputusan, jalur galat, dan jalan pintas |
| **Struktur informasi** | Di menu mana tiap fitur berada, bagaimana berpindah antara penawaran, proyek, paket, dan venue | Peta menu WO Console dan Portal, jalur tautan antarlayar |
| **Tata letak layar** | Apa yang dilihat pertama, apa yang disembunyikan sampai dibutuhkan, di mana tombol utama | Wireframe desktop (≥ 1280 px) dan HP (375 px) per layar |
| **Wadah dan interaksi** | Halaman penuh, drawer, atau dialog; cara menambah, menyusun ulang, dan menghapus baris; kapan sistem menghitung ulang; cara menampilkan kapasitas terlampaui | Aturan interaksi per komponen (penyunting acara & sesi, vendor & venue, rincian) |
| **Keadaan** | Apa yang tampil saat kosong, memuat, galat, kapasitas terlampaui, kedaluwarsa, baca-saja, tanpa izin | Contoh tiap keadaan per layar |
| **Teks layar** | Nama tombol, label, pesan galat, penjelasan singkat, dalam istilah glosarium | Daftar teks per layar |
| **Dokumen untuk klien** | Susunan PDF penawaran yang mudah dibaca pengantin dan keluarganya | Tata letak halaman PDF |

**Alur yang wajib dirancang:**
1. Sales: wawancara klien → buat penawaran → susun acara & sesi → pilih paket, vendor & venue → atur harga, bonus, potongan → pratinjau → buat revisi → kirim PDF lewat WhatsApp → revisi ulang → tandai diterima atau ditolak.
2. Sales atau Owner: penawaran diterima → jadikan proyek (isian pelengkap, tanda jadi) → proyek siap dikerjakan.
3. Owner atau Admin: menyiapkan katalog (jenis acara, paket, isi paket) dan Profil usaha (kode usaha, PKP, masa berlaku, syarat & ketentuan) saat pertama kali memakai fitur ini.
4. Wedding Planner: tamu bertambah → kapasitas venue terlampaui → ganti venue atau bagi sesi → rincian dan termin menyesuaikan → Hitung ulang termin.
5. Wedding Planner: mencatat progres vendor & venue; Owner mencatat biaya dan pembayaran ke rekanan, lalu melihat margin.
6. Proyek dibuat langsung tanpa penawaran.
7. Pengantin di HP: melihat acara dan rincian, membayar vendor yang dibayar sendiri, lalu mengirim bukti; WO memeriksa bukti itu.
8. Keluarga di HP: melihat acara dan isi paket tanpa angka.

**Riset internet bersumber dan bertanggal:**
- cara produk sejenis menyusun alur dan layar ini: perangkat lunak WO dan event, pembuat penawaran;
- pola penyunting baris harga, umpan balik batas atau kapasitas, status dan revisi dokumen, serta alur "terima lalu jadikan proyek";
- contoh surat penawaran WO di Indonesia;
- aksesibilitas pola-pola itu.

**Dasar yang dipakai, bukan diganti:** `knowledge/DESIGN.md` (token navy + emas segel, kit shadcn, kepadatan console/portal, aturan wadah, keadaan, glosarium) dan komponen `shared/components/ui`.

**Uji alur sebelum kode:** setiap alur di atas dijalankan sebagai uji jalan di atas wireframe dari sudut pandang perannya. Langkah yang membingungkan, berlebih, atau buntu diperbaiki di `DESAIN.md`. Ukuran yang dipakai sama dengan `knowledge/DESIGN.md` §1: keberhasilan tugas, jumlah langkah, salah langkah, dan skor kemudahan per tugas.

**Selesai bila** pemilik produk menyetujui `DESAIN.md`. Perubahan alur atau rupa di tengah tahap memperbarui `DESAIN.md` lebih dulu.

**Tahap 1: Direktori & katalog** (tidak menyentuh proyek)
- ADR-0048 (acara/pemasok/rincian sebagai dasar) dan ADR-0049 (modul `offers`, port, pintu publik `rincian`).
- Venue: dua kapasitas, tiga harga, impor/ekspor. `price_akad_resepsi` tetap di vendor.
- `offers`:
  - `event_kinds` dengan isian awal Lamaran, Pengajian, Siraman, Akad, Resepsi, Ngunduh mantu;
  - paket, isi paket, acara yang dicakup;
  - fungsi murni §2.1;
  - port rujukan untuk `vendors` (`SetOfferReferences`) dan pengisian jenis acara bawaan untuk `platform` (`SetEventKinds`).
- Menu **Katalog** (Paket, Jenis acara, Jadwal bawaan, Kategori vendor) menggantikan grup Templat.
- Profil usaha bagian Penawaran: kode usaha (isian awal dari nama usaha untuk WO baru dan WO yang ada), PKP, masa berlaku bawaan, dan syarat & ketentuan bawaan. `SeedDefaultEventKinds` saat WO mendaftar. Glosarium dan `check:terms`.
- **Selesai bila:**
  - venue tanpa kapasitas tidak bisa disimpan;
  - paket dengan venue bawaan 600 berdiri menolak tamu termasuk 650;
  - tes tabel `ExtraGuests` (500 termasuk, kelipatan 50 @ Rp4 jt: 620 tamu → Rp12 jt, 450 → 0, kelipatan 1 → per pax);
  - tes `LineTotals` (PKP: subtotal Rp100 jt → PPN Rp11 jt; potongan melebihi subtotal ditolak);
  - vendor yang dirujuk paket tidak bisa dihapus permanen;
  - WO yang baru mendaftar langsung punya enam jenis acara bawaan dan kode usaha;
  - paket WO A tidak muncul, tidak bisa dibuka, dan venue WO B tidak bisa dipakai sebagai venue bawaan paket WO A.

**Tahap 2: Acara, vendor & venue di proyek**
- Migrasi:
  - `project_events`, `project_event_sessions`, `project_suppliers`, `supplier_payments` (+`payer`);
  - hapus `project_vendors`, `vendor_payments`, `venue_payments`, `projects.venue*`;
  - jenis bukti dan aktivitas baru;
  - seed dev dan fixture e2e ditulis ulang.
- Yang ikut disesuaikan:
  - kas, utang, dasbor, dan antrean verifikasi;
  - hapus lunak, pulihkan, purge, duplikat;
  - riwayat vendor "Lihat Project";
  - kalender;
  - `adminseed`;
  - tes isolasi.
- `PortalPolicy` untuk Event, Supplier, Issue, Evidence, SupplierPayment, ditambah tes rute portal.
- Layar: tab Proyek › Acara & rincian (bagian acara), Proyek › Vendor & venue, Proyek › Keuangan (tiga buku), Portal › Jadwal (bagian Acara), tab portal Vendor menjadi Layanan (progres vendor).
- Proyek baru tanpa penawaran: dua langkah (pasangan & tim, lalu minimal satu acara), tanpa langkah termin. Termin dibuat setelah rincian tersimpan (Tahap 3). Tombol topbar "Baru ▾".
- **Selesai bila:**
  - masjid 150 duduk (Akad, 1 sesi 120) dan gedung 600 berdiri (Resepsi, 2 sesi × 550) bisa disimpan;
  - sesi 650 ditolak dengan pesan berangka;
  - mengganti gedung dengan yang muat 500 ditolak;
  - `paid_by` tidak bisa diubah setelah ada pembayaran;
  - Wedding Planner tidak melihat maupun mencatat pembayaran ke rekanan;
  - nama vendor tersembunyi tidak muncul di kendala;
  - memilih venue untuk acara langsung memunculkannya di Vendor & venue;
  - e2e "buat proyek → acara → vendor & venue → bayar rekanan".

**Tahap 3: Rincian & nilai kontrak**
- Migrasi: `project_lines`; `projects.contract_*`, `lines_version`, `is_pkp`; hapus `package_name` dan `contract_value`.
- Penyunting rincian, salinan aturan paket, tambahan tamu otomatis, potongan, bonus, PPN, margin, `termsMismatch` + Hitung ulang termin, riwayat perubahan.
- `PortalPolicy` untuk Project, Home, Line, Term. Endpoint `…/lines/preview`.
- Layar: tab Proyek › Acara & rincian (bagian rincian dan panel Ringkasan biaya), Portal › Layanan (paket & rincian).
- **Selesai bila:**
  - nilai kontrak = subtotal + PPN dan tidak bisa negatif;
  - menaikkan tamu resepsi menambah baris tambahan tamu dalam transaksi yang sama;
  - mengubah harga paket di katalog tidak mengubah proyek;
  - Keluarga tidak melihat angka sama sekali;
  - Pengantin melihat isi paket tanpa harga per isi;
  - pratinjau total sama dengan total setelah disimpan, dan tidak menulis apa pun;
  - "Margin belum lengkap" muncul bila ada vendor dibayar WO tanpa biaya;
  - e2e "tambah paket → ubah tamu → lihat margin & termin".

**Tahap 4: Vendor dibayar klien** (#15)
- `supplier_bill_terms`, kiriman bukti dari portal (daftar izin tulis di `client_scope.go`), antrean verifikasi WO, butir perhatian dasbor, "tagihan berikutnya" di beranda portal. Pembayaran ini dikecualikan dari kas dan omzet WO.
- **Selesai bila:**
  - Pengantin mengirim bukti bayar fotografer, WO memverifikasi, dan kas WO tidak berubah;
  - Keluarga tidak melihat tagihan itu;
  - e2e dua peran.

**Tahap 5: Penawaran + PDF** (§5)
- ADR-0050. Migrasi `quotation_*` dan `projects.quotation_id`.
- Revisi lewat POST + hash, penghitung nomor, transisi lengkap, konversi idempoten, PDF.
- **Selesai bila:**
  - nomor berformat `KODE-PNW-YYMMDD-NNN`; dua WO yang membuat penawaran di hari yang sama sama-sama mendapat `001`; mengubah kode usaha tidak mengubah nomor lama;
  - WO B tidak bisa membuka, mengunduh PDF, menerima, atau mengonversi penawaran WO A (404);
  - pratinjau tidak membuat revisi;
  - dua klik "Buat revisi" bersamaan menghasilkan satu revisi;
  - Revisi 1 tetap persis setelah Revisi 2 dibuat;
  - penawaran kedaluwarsa tidak bisa diterima sebelum diperpanjang;
  - batalkan terima bisa dilakukan sebelum konversi;
  - salinan menandai yang asli Digantikan;
  - menerima satu alternatif menjadikan alternatif lain Digantikan; batalkan terima tidak menghidupkannya kembali;
  - penawaran baru mendapat masa berlaku dari Profil usaha (bawaan 14 hari);
  - dua permintaan konversi bersamaan menghasilkan satu proyek;
  - proyek hasil konversi sama dengan revisi yang diterima (acara, vendor & venue, baris, termin) dan menautkan Kesepakatan awal;
  - PDF memuat kop, termin, syarat & ketentuan, PPN (bila PKP), dan "Dibuat dengan ElProof", tanpa biaya rekanan, di bawah 30 MB;
  - e2e "penawaran → revisi → terima → jadikan proyek".

**Dokumen:** perbarui setiap tahap: `MODULE_MAP.md`, `DB_SCHEMA.md`, `API_CONTRACT.md`, `DOMAIN_GLOSSARY.md`. ADR-0013, 0014, 0016, 0018, 0019, 0020, 0034, dan 0036 diberi catatan "sebagian digantikan oleh ADR-0048".

**Pemasangan:** tahap 2 dan 3 **menghapus tabel**. Sebelum dipasang, backup `elproof_db` (`pre_deploy_<ts>.sql`, chmod 600). Pemasangan tetap menunggu kata "deploy" dari Anda per tahap, lalu `/opt/elcodelabs/check-all-apps.sh`.

## 9. Keputusan pelengkap

1. **Proyek bisa dibuat langsung tanpa penawaran**, dengan penyunting acara, vendor & venue, dan rincian yang sama. Aturan "hanya penawaran diterima yang bisa dijadikan proyek" (#8) berlaku untuk jalur dari Penawaran. Termin proyek seperti ini dibuat setelah rincian tersimpan.
2. **Paket dan jenis acara ada di menu Katalog**, bersama Jadwal bawaan dan Kategori vendor, karena dipakai setiap kali menyusun, bukan pengaturan yang jarang dibuka. Penawaran menjadi menu utama.
3. **Item diketik bebas.** Tidak ada katalog item tambahan.
4. **Isi paket boleh menyebut vendor** (opsional). Memilih vendor menyalin kategorinya bila kategori belum diisi, dan vendor yang disebut paket tidak bisa dihapus permanen. Perkiraan biaya per isi paket opsional.
5. **Jenis acara bawaan dikenali lewat kunci tetap**, bukan namanya, sehingga jenis bawaan yang sudah diganti nama tidak ditambahkan lagi saat pengisian awal diulang. Jenis acara yang dicakup paket tidak bisa dihapus, hanya dinonaktifkan; paket lama boleh tetap mencakup jenis yang sudah nonaktif, tetapi jenis nonaktif tidak bisa dipilih untuk paket baru.
6. **Paket boleh dihapus permanen sampai Penawaran ada (Tahap 5).** Sejak itu paket yang dirujuk penawaran hanya bisa dinonaktifkan. Kolom "Dipakai di" di Jenis acara menghitung paket; hitungan penawaran ditambahkan di Tahap 5.
7. **Contoh hitung paket dihitung server** (`POST /packages/preview`, fungsi `offers/domain` yang sama), dengan jeda 500 ms, seperti total sementara rincian (§2.2).
8. **Pengaturan penawaran disimpan utuh** (PUT tanpa `If-Match`), sama dengan data usaha di Profil usaha: satu pemilik mengubahnya jarang, dan tidak ada daftar yang bisa basi.
9. **Satu acara adalah acara utama**, dan tanggalnya adalah tanggal acara proyek (Hari H). Saat proyek dibuat, acara pertama menjadi utama; setelahnya diganti lewat aksi "Jadikan acara utama". Menghapus acara utama menjadikan acara paling awal yang tersisa sebagai utama. Mengubah tanggal acara utama menggeser tahapan H-n dalam transaksi yang sama.
10. **Proyek minimal punya satu acara.** Menghapus acara terakhir ditolak (422 `last_event`). Hapus acara bersifat permanen, tanpa pulihkan.
11. **Urutan acara mengikuti tanggal.** Pindah ke atas atau ke bawah hanya bisa antar acara bertanggal sama (422 `event_order_by_date`); selain itu, ubah tanggalnya.
12. **Venue hanya masuk lewat acara.** Tambah vendor & venue di tab Vendor & venue khusus vendor. Venue yang masih dipakai acara tidak bisa dihapus dari proyek (422 `venue_in_use`). Venue yang dilepas acara dan tidak punya pembayaran, bukti, tahapan, atau kendala langsung dihapus dari proyek.
13. **Kapasitas venue di proyek adalah salinan.** Cek kapasitas memakai salinan itu; angka direktori hanya dipakai untuk venue yang belum ada di proyek. "Perbarui kapasitas" menyalin ulang angka direktori ke proyek.
14. **Ganti ke Klien langsung mengosongkan biaya rekanan dan "tampilkan nama".** Biaya rekanan tidak boleh di bawah yang sudah dibayar WO.
15. **Tanpa izin margin, uang keluar ikut tersembunyi di Beranda dan Keuangan:** arus kas tanpa uang keluar, tanpa daftar utang dan jatuh tempo rekanan, dan antrean bukti hanya berisi pembayaran oleh klien.
16. **Di portal, nama vendor yang disembunyikan diganti nama kategorinya**, atau "Layanan" bila tidak berkategori, di semua tempat termasuk kendala.
17. **Duplikat proyek tidak menyalin vendor & venue yang batal**; sisanya kembali berstatus Planned beserta biaya rekanannya, tanpa pembayaran.

## 10. Dasar keputusan

| Temuan | Sumber | Keputusan |
|---|---|---|
| Resepsi sering dibagi sesi; kapasitas gedung berbeda untuk berdiri dan duduk | Riset WO (IDN Times, Cermati, Alissha), riset produk (Perfect Venue) | Batas kapasitas per sesi dan per tata letak (#20) |
| Lamaran, pengajian, dan acara lain umum; akad dan resepsi sering beda hari | Riset WO (Ambarrukmo, Bridestory), riset produk (Tripleseat multi-event) | Daftar acara (#21) |
| Satu vendor bisa memasok beberapa baris; tahapan, kendala, dan pembayaran melekat pada keterlibatan vendor | Kode (`migrations/000008`, `overpayment.go`) | Kerja sama pemasok terpisah dari baris rincian |
| Kas, utang, dan dasbor membaca tabel pembayaran langsung | Kode (`mysql_finance_repository.go:183`) | `paid_by` terkunci setelah ada uang; `payer` per pembayaran |
| Idempotensi hanya memutar ulang respons dengan kunci yang sama | Kode (`idempotency.go`) | `UNIQUE quotation_id` + endpoint idempoten per penawaran |
| `staff/application` mengimpor `projects/contracts`, yang mengimpor `projects/application` | Kode (`contracts.go:12`, `staff_service.go:10`) | `offers` memakai port `StaffNameResolver` |
| Katalog paket bisa diedit kapan saja | — | Salinan aturan paket di baris |
| Rute portal banyak dan redaksinya tersebar; kendala mengirim nama vendor | Kode (`client_scope.go:36-92`, `issue_endpoints.go:70`) | `PortalPolicy` per sumber daya + tes semua rute |
| Jadwal termin tidak boleh ditulis ulang diam-diam | ADR-0042 | Nominal baris positif, total ≥ 0, `termsMismatch` + Hitung ulang termin |
| Revisi lahir saat dikirim ke klien, bukan saat pratinjau | Riset produk (HoneyBook, HubSpot) | Pratinjau DRAF + "Buat revisi untuk klien" |
| Penawaran memuat termin dan syarat; DP mengunci tanggal | Riset WO (perjanjian D'Banquet, S&K Get Her Ring), riset produk (Dubsado, Odoo) | Termin + syarat & ketentuan di penawaran; tanda jadi saat konversi |
| Perubahan setelah sepakat perlu jejak | Riset produk (Planning Pod, Salesforce CPQ) | Kesepakatan awal + riwayat perubahan; salinan menandai Digantikan |
| Item dihitung jumlah × harga satuan; bonus umum di paket WO | Riset produk (HoneyBook, Planning Pod), riset WO | `quantity`, `unit`, `unit_price`, `is_bonus` |
| PPN wajib bagi WO PKP | DJP PMK 131/2024 | Pengaturan PKP (#23) |
| Klien datang ke venue; foto nota vendor bisa memuat harga rekanan | Kode, riset produk | Venue selalu tampil; bukti pemasok dibayar WO bawaannya tersembunyi |
| Wedding Planner punya `finance.record` tanpa izin margin | Kode (`catalog.go:126`) | Izin margin untuk pembayaran rekanan (#22) |
| "Paket" sudah dipakai untuk paket langganan | Kode (25+ kemunculan di `apps/web/src`) | Glosarium + `check:terms` |
| Perombakan menyentuh ±87 berkas Go, 51 berkas web, dan 16 berkas e2e | Kode | Pekerjaan dibagi lima tahap, masing-masing bisa dipasang sendiri |
| ElProof adalah SaaS banyak WO; pemisahan data bergantung pada disiplin `tenant_id` di setiap kueri, tanpa bantuan database | ADR-0004, `tests/isolation` | Aturan §2.3 untuk setiap tabel baru + tes isolasi di setiap tahap |
| Data bawaan WO baru sudah diisi saat pendaftaran (`SeedDefaultCategories`, `SeedDefaultRoles`) | Kode (`signup_service.go:211`, `tenant_service.go:124`) | `SeedDefaultEventKinds` dengan pola yang sama |
| `platform` belum punya `contracts/`, sedangkan `platform` sudah mengimpor `vendors/contracts` | Kode (`tenant_service.go:96`) | Port `WorkspaceProfile` + setter `SetEventKinds`, tanpa siklus |
| Aplikasi bisnis menyusun nomor dokumen dari awalan teks, tanggal, dan nomor urut; awalan membedakan jenis dokumen | Riset produk (MASERP, Odoo, Enerpize) | Nomor `KODE-PNW-YYMMDD-NNN`: kode usaha WO, jenis dokumen, tanggal, urut harian per WO |
| Produk sejenis menyusun penawaran sebagai satu dokumen berseksi dengan total yang selalu terlihat; total sementara tidak boleh punya rumus kedua | `DESAIN.md` §2 (D1, D2, D7) | Halaman penawaran berseksi, panel Ringkasan biaya, `…/lines/preview` |
| Klien sering diberi dua pilihan jumlah tamu atau paket | Keputusan #26, riset produk | Penawaran alternatif yang tertaut |
| `knowledge/DESIGN.md` sudah menetapkan token, kit, wadah, dan glosarium, tetapi belum mencakup alur dan layar penyunting harga, penawaran, kapasitas, dan PDF | `knowledge/DESIGN.md` | Riset desain UI/UX khusus fitur ini sebelum Tahap 1 (alur, struktur menu, tata letak, interaksi, keadaan, teks), diuji jalan sebelum kode, hasilnya `DESAIN.md` |

**Dipertimbangkan, tidak diambil:**
- **Peringatan kapasitas yang bisa dilewati:** keputusan #20 memakai aturan keras per sesi.
- **Tabel harga per tingkat tamu:** tingkat cukup dibuat sebagai paket terpisah.
- **E-sign, klien menerima sendiri lewat tautan, change order bertingkat, mesin pajak katering (PBJT):** terlalu berat untuk WO kecil.

**Sengaja tidak termasuk:**
- Timeline per acara.
- PDF "Perubahan rincian".
- Mesin aturan pembatalan/refund (jenis pembayaran Refund sudah ada).
- Faktur pajak dan PBJT katering (perlu konsultan pajak).
- Katalog item tambahan.
- White-label PDF.
- Invoice dan kwitansi.
- Pembatasan fitur per paket langganan.

**Sumber** (diakses 19 Nov 2026):
- Alissha, All-in Package: https://alexandra.bridestory.com/raw/upload/assets/alissha-all-in-package_may2024-R1Kjx0m8c.pdf
- D'Banquet Pantai Mutiara, Perjanjian Pesta: https://alexandra.bridestory.com/raw/upload/assets/perjanjian-pesta-H16BDXoE_.pdf
- Get Her Ring, Syarat & ketentuan: https://alexandra.bridestory.com/raw/upload/assets/syarat-and-ketentuan-Hke6PvHnI.pdf
- Ambarrukmo, Paket Wedding 2026: https://ambarrukmo.com/paket-wedding-hotel-jogja
- ANTARA, Cara hitung biaya katering resepsi (29-07-2024): https://www.antaranews.com/berita/4224291/cara-hitung-biaya-katering-resepsi-pernikahan
- IDN Times, Mengatur resepsi banyak tamu: https://www.idntimes.com/life/inspiration/mengatur-resepsi-banyak-tamu-c1c2-01-dv2xl-ndym9c
- Cermati, Gedung pernikahan di Jakarta: https://www.cermati.com/artikel/dibawah-rp30-juta-ini-rekomendasi-gedung-pernikahan-di-jakarta
- DJP, PMK 131/2024: https://www.pajak.go.id/id/artikel/pmk-1312024-tarif-ppn-sebelas-dua-belas
- DJP, Jasa boga tidak kena PPN: https://www.pajak.go.id/id/artikel/jasa-boga-jasa-tidak-kena-ppn
- DDTCNews, PPh 23 jasa event organizer (14-10-2024): https://news.ddtc.co.id/berita/nasional/1806174/jasa-event-organizer-kena-pph-pasal-23-begini-ketentuannya
- Dubsado, Payment plans: https://help.dubsado.com/en/articles/467089-payment-plans
- Dubsado, Connect a contract and invoice to a proposal: https://help.dubsado.com/en/articles/6943800-connect-a-contract-and-invoice-to-a-proposal
- HoneyBook, Add services to a file: https://help.honeybook.com/en/articles/5643712-add-services-to-a-file
- HoneyBook, When to notify clients of file changes: https://help.honeybook.com/en/articles/6252653-when-to-notify-clients-of-file-changes
- Planning Pod, Proposal management: https://planningpod.com/proposal-management.cfm
- Tripleseat, Proposals, contracts & BEO: https://tripleseat.com/features/proposals-contracts-beo/
- Perfect Venue, Venue capacity system: https://www.perfectvenue.com/post/venue-capacity-system
- Odoo, Create quotations: https://www.odoo.com/documentation/18.0/applications/sales/sales/sales_quotations/create_quotations.html
- HubSpot, Manage quotes: https://knowledge.hubspot.com/quotes/manage-quotes
- Salesforce Trailhead, Contract amendments (CPQ): https://trailhead.salesforce.com/content/learn/modules/contract-amendments-with-salesforce-cpq/change-an-existing-contract
- MASERP, Cara setting nomor otomatis: https://intercom.help/maserp/id/articles/5396310-cara-setting-nomor-otomatis
- Enerpize, Adding a prefix to the auto numbering: https://docs.enerpize.com/user_manual/adding-a-prefix-to-the-auto-numbering/
- Odoo forum, Quotation/sales order sequence number: https://www.odoo.com/id_ID/forum/help-1/quotationsales-order-sequence-number-270177
