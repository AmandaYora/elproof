# Pemahaman bersama — Paket WO & portal klien (draf penyelarasan)

Status: **PLAN.md disepakati pengguna** (19 Nov 2026, dikutip: "Sepakat"). PLAN.md adalah sumber akhir: bila tabel di bawah berbeda dengan PLAN.md (mis. urutan tahap di #17), yang berlaku PLAN.md §8 (Tahap 1–5). Pengguna minta PLAN selalu final saat dibuat, tanpa versi, dan tidak merujuk PLAN lain. Kode, commit, dan pemasangan menunggu permintaan pengguna. PLAN.md kemudian diperbarui dengan keputusan #24 (SaaS & nomor dokumen) dan #25 (riset desain sebelum Tahap 1). `DESAIN.md` disetujui pengguna (19 Nov 2026, dikutip: "Sepakat"), dan PLAN.md diselaraskan dengan keputusan rancangannya (menu Katalog, Profil usaha, label "Vendor & venue", tab proyek dan portal, pratinjau hitungan, penawaran alternatif #26).

## Permintaan
ElProof perlu mendukung tiga model bisnis WO tanpa tambal sulam:
- organizer saja (klien bayar vendor langsung);
- paket all-in (semua termasuk, WO bayar vendor, bisa tim internal);
- campuran.

Termasuk katalog paket, jenis acara (Akad / Resepsi / Akad + Resepsi), dan portal klien yang menyesuaikan.

## Fakta yang sudah ditemukan
- Model uang sekarang: klien bayar WO (`projects.contract_value` + `client_payment_terms`), WO bayar vendor (`vendor_payments`) dan venue (`venue_payments`). Margin = kontrak − Σ vendor − venue.
- `package_name` di proyek hanya teks bebas. Tidak ada katalog paket. Tidak ada jumlah tamu (pax) di proyek.
- Akad / Akad + Resepsi / Resepsi saja hanya ada sebagai harga vendor (`price_akad`, `price_akad_resepsi`, `price_resepsi`) dan `pricing_tier` per keterlibatan vendor.
- Venue terhubung 1:1 dengan harga dari direktori. Tidak ada harga per proyek.
- Portal (`client_view.go`) sudah menyembunyikan nilai kontrak vendor, DP, pembayaran vendor, catatan, dan harga venue.
- Bukti pembayaran vendor tidak pernah tampil ke klien (`ClientCanSeeEvidence`).
- Tab Vendor di portal tetap menampilkan semua vendor beserta tahapan dan bukti tahapannya.
- Kendala punya `is_client_visible` dengan nilai bawaan TRUE.
- Peran portal bawaan:
  - Pengantin: semua izin `portal.*`.
  - Keluarga: vendor, jadwal, dokumen, kendala. Tanpa keuangan.
- Fitur paket langganan (`plan_features`) hanya label, tidak membatasi fitur.
- Workspace sudah punya logo (`tenants.logo_storage_path`) dan data usaha termasuk rekening (`tenant_business_details`): bahan kop PDF sudah ada.
- PLAN ditaruh di `docs/plan/<slug>/PLAN.md`.

## Keputusan (diisi sambil jalan)
| # | Cabang | Jawaban | Sumber |
|---|---|---|---|
| 1 | Lingkup | Tahap 1–3 dalam satu PLAN, dikerjakan bertahap, **ditambah fitur Penawaran** (lihat #8). Sesi acara terpisah tetap kelanjutan | pengguna (diperbarui oleh #8) |
| 2 | Pemicu | Ada WO nyata yang butuh (paket all-in/campuran) | pengguna |
| 3 | Nego harga | Harga paket tetap seperti katalog; selisih dicatat sebagai baris "Potongan". Nilai kontrak selalu = jumlah rincian | pengguna |
| 4 | Pax | Paket per pax: paket punya jumlah tamu yang sudah termasuk, proyek menyimpan jumlah tamu per acara (lihat #18), tambahan pax = opsi tambahan berharga satuan | pengguna |
| 5 | Hitung pax | Di atas kapasitas: tambahan per kelipatan yang ditentukan WO (dibulatkan ke atas). Di bawah kapasitas: harga tetap | pengguna |
| 6 | Tim internal | Tidak perlu konsep tim internal: meski paket all-in, vendor tetap perusahaan terpisah yang sudah bekerja sama dengan WO, dan kerja sama itu di luar aplikasi. Atribut "dikerjakan oleh" dihapus dari rancangan | pengguna (dikutip: "itu baiknya tidak perlu karena umumnya meskipun All In Package, tetap antara WO dan vendor itu perusahaan terpisah yang memang menjalin kerja sama sebelumnya, dan kerjasama tersebut di luar aplikasi") |
| 7 | Biaya vendor rekanan | Opsional. Klien bertransaksi dengan WO, bukan dengan vendor lewat WO. Harga ke klien per item ditentukan WO (boleh disesuaikan dari harga rekanan); nilai kerja sama & pembayaran ke vendor rekanan boleh diisi WO untuk memantau margin, tidak pernah tampil ke klien. Bila kosong, margin ditandai belum lengkap | pengguna (dikutip: "bisa jadi opsional … Client bertransaksi dengan WO, dan urusan WO bertransaksi dengan vendor itu hal berbeda tidak perlu diketahui client") |
| 8 | Penawaran | Menu baru "Penawaran". WO menyusun penawaran dari hasil wawancara dengan klien (di luar aplikasi), bisa diunduh sebagai PDF untuk klien, bisa diubah berkali-kali sebelum disetujui. WO menandai penawaran "diterima"; hanya penawaran yang diterima yang bisa dijadikan proyek, dan proyek dibuat dari data penawaran itu | pengguna (dikutip: "baru yang bisa dijadikan proyek itu adalah penawaran penawaran yang sudah diterima saja") |
| 9 | Revisi penawaran | Setiap unduhan PDF menyimpan isi saat itu sebagai Revisi 1, 2, 3 yang bisa dibuka lagi. Proyek dibuat dari revisi yang diterima | pengguna |
| 10 | Status penawaran | Draf, Diterima, Ditolak, Kedaluwarsa (otomatis setelah masa berlaku yang diisi WO lewat) | pengguna |
| 11 | PDF | Identitas WO (nama, logo, kontak, rekening dari pengaturan workspace) + tulisan kecil "dibuat dengan ElProof" di kaki halaman | pengguna |
| 12 | Dibayar oleh | Per item: "dibayar WO" (masuk nilai kontrak, harga dari WO) atau "dibayar klien langsung" (tidak masuk kontrak; WO tetap memantau progres vendor). Organizer saja / all-in / campuran dibedakan oleh tanda ini, bukan oleh jenis proyek | pengguna |
| 13 | Rincian di portal | Klien melihat nama & harga paket, daftar isi paket (tanpa harga per isi), opsi tambahan, dan potongan — sama dengan PDF penawaran | pengguna |
| 14 | Nama vendor | Item dibayar WO: nama vendor tersembunyi secara bawaan (klien melihat jenis layanan), WO bisa menampilkannya per item | pengguna |
| 15 | Tagihan vendor milik klien | Opsional: WO boleh mencatat nilai & jadwal bayar vendor yang dibayar klien langsung sebagai pengingat. Klien melihatnya di portal dan bisa mengunggah bukti bayar; WO memeriksa. Keluarga tidak melihat (mengikuti izin keuangan) | pengguna |
| 16 | Data lama | Tidak ada data lama yang dijaga: ElProof belum diluncurkan, datanya dummy milik developer. Skema, modul, dan alur boleh dirombak sampai ke dasar arsitektur; tanpa lapisan kompatibilitas | pengguna (dikutip: "jangan ragu untuk penyesuaian, dan hindari tambal sulam, pastikan ideal bahkan jika menyentuh dasar architecture nya") |
| 17 | Tenggat | Bertahap tanpa tenggat; tiap tahap bisa dipakai setelah dipasang. Urutan: (1) katalog paket + rincian proyek, (2) portal, (3) Penawaran + PDF | pengguna |
| 18 | Kapasitas venue | Jumlah tamu dicatat per acara (tamu akad, tamu resepsi). Tiap venue di rincian ditandai untuk Akad / Resepsi / Akad + Resepsi. Ditolak sistem bila tamu suatu acara melebihi kapasitas venue acara itu; satu venue untuk keduanya dibatasi oleh angka terbesar. Venue belum dipilih: jumlah tamu boleh diisi dengan tanda "belum dicek ke kapasitas venue". Paket per pax tetap: dihitung dari tamu acara yang dicakup paket (Akad + Resepsi = angka terbesar), kelebihan tamu hanya sampai kapasitas venue; paket all-in yang memuat venue tidak boleh punya jumlah tamu termasuk melebihi kapasitas venue itu. Kapasitas wajib diisi di manajemen venue | pengguna (dikutip: "maksimal pax itu harusnya tidak boleh lebih dari kapasitas venue"; pola disetujui: "Ya, pakai pola ini") |
| 19 | Harga venue | Venue punya harga per jenis acara: harga akad, harga resepsi, harga akad + resepsi (seperti vendor), menggantikan "harga sewa". "Charge" tetap sebagai biaya tambahan opsional (usulan, tidak dikoreksi) | pengguna (dikutip: "dan ada harga akad, harga resepsi, dan harga akad + resepsi") |
| 20 | Kapasitas per sesi | Tetap ditolak sistem, tetapi yang dibandingkan adalah tamu per sesi dengan kapasitas venue sesuai tata letak (berdiri/duduk). Venue menyimpan dua angka kapasitas. Memperbarui #18 (dari "total per acara") | pengguna (pilihan "Keras, per sesi") setelah riset menunjukkan resepsi bersesi dan kapasitas berdiri/duduk |
| 21 | Daftar acara | Penawaran dan proyek punya daftar acara (jenis yang bisa ditambah WO: Akad, Resepsi, Lamaran, …), masing-masing dengan tanggal, venue, tata letak, dan sesi beserta tamunya. Paket menyebut acara yang dicakup. Timeline tetap satu per proyek. Menggantikan kolom tamu akad/resepsi | pengguna (pilihan "Daftar acara sekarang") |
| 22 | Biaya rekanan | Biaya rekanan dan pembayaran ke rekanan yang dibayar WO hanya untuk peran dengan izin lihat margin (bawaan Owner, Admin). Wedding Planner tetap mencatat uang masuk klien dan memantau progres vendor | pengguna (pilihan "Hanya yang boleh lihat margin") |
| 23 | Pajak | Pengaturan workspace "PKP / bukan PKP", bawaan bukan PKP tanpa baris pajak. PKP: baris PPN otomatis (12% × DPP 11/12) tercetak di PDF. Faktur pajak di luar aplikasi | pengguna (pilihan "Pengaturan PKP sederhana") |
| 24 | SaaS & nomor dokumen | ElProof melayani banyak WO; setiap data milik satu WO (PLAN §2.3, tes isolasi di setiap tahap). Nomor penawaran memuat kode usaha WO, jenis dokumen, tanggal, dan urutan harian: `KODE-PNW-YYMMDD-NNN` (mis. `MO-PNW-261119-001`, 17–20 karakter). Kode usaha diisi otomatis dari nama usaha dan bisa diubah WO | pengguna (dikutip: "jangan melupakan bahwa Elproof ini adalah Saas … pastikan format code nya itu sesuai dengan tenant nya"; "ada bulan, tanggal nya juga pada code, namun hati hati jangan terlalu panjang atau pendek, buatkan ideal saja") |
| 25 | Riset desain | Sebelum kode layar ditulis: satu riset desain UI/UX dan satu brief (`DESAIN.md`) untuk semua layar baru, disetujui pengguna dulu. Cakupannya alur penggunaan per peran, struktur menu, tata letak desktop & HP, interaksi, keadaan, teks layar, dan PDF, diuji jalan sebelum kode; warna dan komponen memakai `knowledge/DESIGN.md` apa adanya | pengguna (dikutip: "pastikan Juga desain UI UX nya itu modern, ideal, dan mudah difahami, itu perlu riset desain dahulu"; pilihan "Sekali, semua layar"; "pastikan desain yang dimaksud itu UI UX beserta flow penggunaan nya bukan sekedar warna saja") |
| 26 | Penawaran alternatif | WO bisa memberi pilihan lewat "Buat alternatif": penawaran kedua yang tertaut dan tetap hidup berdampingan; saat salah satu diterima, alternatif lainnya otomatis Digantikan. Setiap PDF tetap satu pilihan | pengguna (pilihan "Penawaran alternatif") setelah riset produk menunjukkan WO kadang menawarkan dua pilihan sekaligus |
| 27 | Tidak termasuk | Tidak ada isian khusus "Tidak termasuk" di paket; WO menuliskannya di syarat & ketentuan bila perlu | pengguna (pilihan "Cukup di syarat & ketentuan") |

## Asumsi
- Jenis acara (Akad / Resepsi / Akad + Resepsi) adalah atribut paket dan proyek; harga vendor per jenis acara tetap ada di direktori vendor.
- Venue diperlakukan sebagai salah satu item rincian (dibayar WO atau klien langsung), bukan hubungan khusus 1:1, dan ditandai untuk acara mana.
- Pemeriksaan mutu tetap seperti biasa: `npx tsc -b`, `check:terms`, build, budget, e2e, dan tes Go.

Kemampuan: skill yang ada cukup (rudis-business-systems-analyst untuk rancangan bila perlu; tidak ada skill baru).

## Pertanyaan keahlian (untuk penyusun PLAN)
- Penempatan modul (`packages` baru vs di `projects`) dan pemutusan siklus dependensi vendors ↔ projects ↔ packages.
- Bentuk satu fungsi aturan tampil portal yang menggantikan fungsi-fungsi redaksi yang tersebar.
- Bentuk data Penawaran dan revisinya (salinan beku per revisi vs. tabel revisi), serta cara "jadikan proyek" menyalin rincian tanpa impor lintas modul.
- Pembuatan PDF di server Go (pustaka, huruf, logo) dalam batas RAM VPS 2 GB.
- Nasib `project_vendors.pricing_tier`, `contract_value` proyek, dan relasi venue 1:1 setelah rincian menjadi sumber nilai kontrak.
