# FLOW-IDEAL.md — Alur & UI/UX Ideal per Peran (Platform Owner · WO · Klien)

> Pendamping [PLAN.md](PLAN.md). Disusun 6 Nov 2026 dari pemetaan kode (as-is) dan riset
> eksternal (sumber di §6). Dokumen ini menjawab "seperti apa yang ideal", sedangkan PLAN.md
> mengatur "urutan pengerjaan". Belum ada kode yang diubah.

## 0. Prinsip lintas peran

| Prinsip | Aturan |
|---|---|
| **Satu design system, dua kepadatan** | **Console** (Platform & WO): padat, sans-serif 14 px, baris tabel 36–40 px, tabel untuk daftar, kartu hanya di Ringkasan. **Portal Klien**: lega, teks ≥ 16 px, berbasis kartu, boleh serif display untuk nama pasangan & hitung mundur, foto header. Palet netral hangat + satu aksen brand tenant yang dipakai hemat; warna status adalah set terpisah dan tidak ikut brand. Acuan nuansa: Linear (refresh Mar 2026) untuk console, HoneyBook client portal untuk portal. |
| **Bahasa** | Bahasa Indonesia penuh, sapaan **"Anda"**. Tidak ada enum Inggris yang tampil ke pengguna (lihat §4). Satu istilah per konsep: *Proyek*, *Klien*, *Dokumen* (bukan Evidence), *Kendala*, *Termin*, *Kata sandi*. |
| **Wadah interaksi** | **Modal**: hanya konfirmasi atau isian 1–3 field. **Drawer/panel samping**: edit sambil melihat konteks (catat pembayaran, tambah vendor ke proyek, update kendala). **Halaman penuh**: form kompleks/bertahap (buat proyek, Timeline Default, detail tenant). |
| **Feedback** | Toast ±5 detik untuk sukses, dengan "Urungkan" bila aksinya bisa dibalik. **Galat tidak hilang sendiri** dan tampil di dalam form/drawer, bukan di belakang overlay. Tombol simpan nonaktif + "Menyimpan…" saat proses. Skeleton bila memuat > 1 detik. Empty state dibedakan: "belum ada data" (dengan CTA) vs "tidak ada hasil filter". |
| **Aksi berbahaya** | Dialog dengan tombol bertuliskan aksinya ("Hapus vendor"), fokus awal di "Batal", dampak disebut (jumlah data terkait). Aksi tak bisa dibalik di level tenant (suspend, hapus, reset secret, ubah gateway) mewajibkan **mengetik nama/slug**. |
| **Keamanan data per peran** | Penyembunyian dilakukan di **API**, bukan hanya di UI. Setiap peran mendapat DTO-nya sendiri. |
| **Aksesibilitas** | WCAG 2.2 AA: kontras 4,5:1, target sentuh ≥ 24 px (44 px di portal), focus trap di modal/drawer, galat merah + `aria-invalid`, `<form>` + `autocomplete`. |

## 1. Platform Owner (tim internal ElProof)

### 1.1 Kondisi sekarang → masalah utama
- 7 menu datar. Role Super Admin dan Support **sama kuasanya**, tanpa 2FA.
- Tidak ada **halaman detail tenant**: semua aksi lewat tombol di baris tabel.
- Metrik dihitung di browser dari seluruh data (`all=true`). "Menunggu Pembayaran" dan "Tenant Perlu Diperhatikan" praktis selalu kosong karena status yang dicari tidak pernah ditulis.
- Suspend tidak berefek. Aktivasi manual tidak mencatat siapa, kenapa, dan buktinya. Tidak ada audit log.
- Fitur paket hanya teks; harga diubah di tempat (perpanjangan berikutnya ikut harga baru); tidak ada kuota.
- Tidak ada impersonasi, pengumuman, catatan internal, atau permintaan data (UU PDP).
- Slide login hanya bisa lewat CLI. Custom domain dipasang manual oleh ops.

### 1.2 Susunan menu ideal

```
Ringkasan
Tenant            → daftar + Detail Tenant 360°
Tagihan           → Invoice · Pembayaran (ledger) · Antrean penagihan
Katalog           → Paket (versi harga) · Fitur & Batas · Kupon (nanti)
Komunikasi        → Pengumuman · Banner pemeliharaan
Aplikasi (PGaaS)  → registry App, kunci write-only, log webhook
Audit Log
Pengaturan        → Admin & Peran (2FA) · Kebijakan langganan · Gateway · Slide login · Data & Privasi
```

### 1.3 Isi halaman kunci

**Ringkasan**
- Satu baris metrik: **MRR**, tenant berbayar, trial berjalan, konversi trial 30 hari, churn 30 hari, nilai tagihan lewat jatuh tempo.
- **Pergerakan MRR** bulan ini: Baru · Ekspansi · Kontraksi · Churn · Reaktivasi.
- Panel **"Perlu tindakan"**: trial habis ≤ 3 hari belum aktivasi, tagihan lewat jatuh tempo, tenant baca-saja, permintaan data PDP yang mendekati tenggat.
- Dihitung di backend (endpoint agregat), tidak lagi dari `all=true`.

Definisi metrik (mengikuti Stripe Billing analytics):
- **MRR**: nilai bulanan langganan `active` + `past_due`. Paket tahunan dibagi 12. Trial, aktivasi gratis (`granted`), dan pajak tidak dihitung.
- **Churn pelanggan 30 hari**: tenant yang masuk `read_only`/`canceled` ÷ (tenant berbayar 30 hari lalu + tenant baru).
- **Konversi trial**: trial yang menjadi berbayar ÷ trial yang berakhir dalam 30 hari.
- **ARPA**: MRR ÷ tenant berbayar.
- **Aktivasi**: tenant yang membuat proyek pertama **dan** mengundang klien pertama dalam trial.

**Tenant (daftar)**
- Pencarian & filter di server (status, paket, sumber daftar Google/manual), tampilan tersimpan sebagai tab ("Trial", "Lewat jatuh tempo", "Baca-saja", "Tidak aktif 14 hari"), paginasi server, ekspor CSV (tercatat di audit).
- Kolom: WO, Owner, Paket, Status langganan, Berakhir, Aktivitas terakhir, Health.

**Detail Tenant 360°** (halaman penuh, header + tab)
- Header: nama WO, slug, status (badge), paket, sisa hari, tombol aksi (menu "⋯").
- Tab **Profil**: data usaha, Owner, sumber daftar, tanggal daftar, persetujuan S&K/DPA (versi + waktu).
- Tab **Langganan**: status, paket + versi harga, periode, **riwayat transisi status** (siapa/sistem, alasan).
- Tab **Tagihan**: invoice & pembayaran tenant ini.
- Tab **Pemakaian**: proyek aktif, staf, klien, storage **vs batas paket**.
- Tab **Pengguna**: staf tenant (nama, peran, login terakhir), tanpa melihat kata sandi.
- Tab **Aktivitas**: login, peristiwa inti (proyek dibuat, klien diundang).
- Tab **Catatan internal**: hanya terlihat tim ElProof.
- Tab **Permintaan data**: ekspor/hapus (UU PDP) dengan tenggat & bukti selesai.
- Aksi: Aktivasi manual, Perpanjang trial, Ubah paket, Suspend/Pulihkan, **Lihat sebagai tenant** (impersonasi), Kirim ulang undangan Owner.

**Aktivasi manual** (drawer)
- Wajib: paket, periode, **alasan** (pilihan: deal sales, kompensasi, uji coba diperpanjang, lainnya), **bukti** (no. transaksi/foto transfer), catatan.
- Menghasilkan entri ledger berstatus `granted` atau `paid_manual` dan entri audit.

**Katalog → Paket**
- Harga di-**snapshot ke setiap invoice** (sudah ada di transaksi). Perubahan harga berlaku di perpanjangan berikutnya. Versi harga/grandfathering **tidak dibangun sekarang** karena belum ada pelanggan riil (K13); bisa ditambahkan tanpa mengubah alur.
- **Fitur & Batas** terstruktur: `max_active_projects`, `max_staff`, `max_clients_per_project`, `storage_mb`, plus flag fitur. Aplikasi WO membaca entitlement ini, bukan nama paket.
- Upgrade berlaku langsung (tagihan selisih prorata harian); downgrade berlaku di periode berikutnya.

**Komunikasi**: pengumuman in-app dengan target (semua / paket / status), jadwal tayang & kedaluwarsa; banner pemeliharaan terjadwal.

**Audit Log**: siapa, kapan, IP, objek, aksi, alasan, nilai sebelum → sesudah, hasil. Tidak menyimpan rahasia. Hanya-baca untuk semua peran, bisa difilter & diekspor.

**Pengaturan → Admin & Peran**

| Peran | Hak |
|---|---|
| Super Admin | Semua, termasuk admin, peran, gateway, kebijakan |
| Billing | Tagihan, aktivasi manual, paket, kupon |
| Support | Lihat tenant, catatan, impersonasi baca-saja, kirim ulang undangan |
| Read-only | Ringkasan & laporan |

- **2FA wajib** (TOTP) untuk semua admin platform. Step-up (masukkan ulang kode) untuk aksi bernilai tinggi.

**Pengaturan → Kebijakan langganan**: lama trial (14), lama tenggang (usulan 7 hari), lama baca-saja sebelum suspend (usulan 23 hari); dapat diubah tanpa deploy. Data tidak dihapus otomatis.

### 1.4 Siklus langganan tenant (state machine)

```
trialing (14 hari) ──bayar──────────────────────────────► active
trialing ──habis, belum bayar──► grace (tenggang) ──► read_only
active ──periode habis, invoice terbuka──► past_due (tenggang: akses penuh + banner)
past_due ──H+N1──► read_only (lihat & ekspor saja; portal klien tetap terbaca)
read_only ──H+N2──► suspended (staf hanya bisa ke halaman Langganan & Ekspor)
suspended : bertahan tanpa batas — **tidak ada penghapusan otomatis** (keputusan K10)
suspended ──permintaan hapus dari Owner (atau keputusan Super Admin)──► closed → data dihapus
grace/past_due/read_only/suspended ──bayar atau aktivasi manual──► active
active ──berhenti di akhir periode──► canceled ──► read_only/closed sesuai kebijakan
suspended_by_admin (pelanggaran) : terpisah dari siklus tagihan, hanya Super Admin
```

- Setiap transisi mencatat aktor (sistem/admin), alasan, dan bukti.
- **Akses** = f(status, entitlement), ditegakkan di backend (middleware), bukan di UI saja.
- Invoice perpanjangan terbit H-7. Pengingat H-7, H-3, H0, H+3 sebagai banner (email/WA menyusul).

### 1.5 Impersonasi aman ("Lihat sebagai tenant")
- Owner tenant memberi **izin akses dukungan** dari Pengaturan (berlaku 24 jam, bisa dicabut).
- Admin wajib mengisi alasan. Sesi maksimal 60 menit, **baca-saja** secara bawaan.
- Banner jelas di layar tenant selama sesi. Awal/akhir sesi tercatat di audit dan terlihat oleh Owner tenant.

## 2. Tenant / WO Console

### 2.1 Kondisi sekarang → masalah utama
- Tidak ada onboarding, profil usaha, ganti/lupa kata sandi, maupun undangan. Kata sandi staf dan klien diketik orang lain.
- Dashboard berorientasi modul (omzet, jumlah proyek), bukan tindakan.
- Detail proyek mendarat di tab Vendor; tidak ada Ringkasan. **Kendala tidak punya tab** (tersembunyi di kartu vendor), padahal portal klien punya.
- Pembayaran hanya riwayat; tidak ada **jadwal termin**, jatuh tempo, atau status terlambat. Lebih bayar tidak dicegah.
- Status proyek bebas diubah; tidak ada penutupan proyek. Venue tercatat ganda.
- Branding & domain hanya bisa diatur tim platform.
- Status Inggris mentah, banyak form di modal dengan galat tersembunyi, tanpa toast.

### 2.2 Susunan menu ideal

```
Beranda (Dashboard)
Proyek
Klien
Direktori        → Vendor · Venue
Keuangan         → Tagihan klien · Pembayaran vendor/venue · Arus kas   (Owner/Admin)
Pengaturan       → Profil Usaha · Tim · Langganan · Kategori Vendor · Timeline Default · Akses Dukungan
```
- Topbar: pencarian global (server-side, ⌘K), **pusat notifikasi**, menu akun.
- Banner global langganan (trial/tenggang/baca-saja) untuk semua peran, dengan CTA sesuai peran ("Bayar" untuk Owner, "Hubungi Owner" untuk lainnya).
- Checklist **"Mulai di sini"** tampil sampai selesai atau ditutup.

### 2.3 Beranda (Dashboard) — berbasis waktu & tindakan
1. Sapaan + tanggal hari ini.
2. **Satu angka utama**: total tagihan klien yang terlambat (Rp).
3. **Perlu tindakan**: termin terlambat, kendala terbuka, item timeline lewat tenggat, vendor belum terkonfirmasi H-30.
4. **Acara 14 hari ke depan** (H-x per proyek).
5. **Tugas jatuh tempo minggu ini** (timeline lintas proyek).
6. **Pembayaran ≤ 30 hari**: masuk dari klien & keluar ke vendor/venue.
7. Arus kas 8 minggu (batang masuk vs keluar).
8. Proyek per tahap.
- Wedding Planner/Sales melihat versi "Proyek saya" (tanpa angka omzet), bukan dilempar ke daftar proyek.

### 2.4 Detail Proyek (tata letak dua kolom)
- **Header**: nama pasangan, tanggal acara + H-x, venue (dari direktori, satu sumber), status (label Indonesia), PIC, tombol **"Lihat sebagai klien"**, aksi utama + menu "⋯" (Duplikat, Arsipkan, Batalkan, Hapus).
- **Panel kanan** (bertumpuk di HP): tahap proyek, sisa tagihan klien, kendala terbuka, tim, tautan & status undangan portal.
- **Tab**: **Ringkasan** · Timeline · Vendor & Venue · Keuangan · Kendala · Dokumen · Klien · Aktivitas.
  - **Ringkasan**: perlu tindakan, 5 item timeline berikutnya, termin berikutnya, vendor belum terkonfirmasi.
  - **Timeline**: daftar per fase (H-90, H-30, H-7, Hari H, H+) dengan tanggal relatif & absolut; Hari H sebagai jangkar (geser tanggal acara → tenggat ikut bergeser); Gantt tetap opsional.
  - **Keuangan**: segmen Klien / Vendor / Venue. Masing-masing: **jadwal termin** (nominal, jatuh tempo, status), bilah dibayar/total, "Sisa Rp…", dikelompokkan Terlambat → Jatuh tempo → Lunas. Catat pembayaran di drawer, menampilkan sisa & isi otomatis untuk Pelunasan; lebih bayar diblokir kecuali tipe Tambahan/dengan konfirmasi.
  - **Kendala**: semua kendala proyek (lintas vendor), dengan sakelar "Terlihat klien".
  - **Dokumen**: sakelar "Terlihat klien" per dokumen (bawaan tersembunyi).
- **Siklus proyek eksplisit**: Draf → Persiapan → Siap (H-7) → Selesai, atau Dibatalkan. Transisi lewat tombol ("Mulai persiapan", "Tandai siap", "Selesaikan proyek"), bukan dropdown. "Selesaikan" menampilkan **checklist penutupan** (semua termin klien lunas, pembayaran vendor tuntas, kendala ditutup).

### 2.5 Hak akses yang diatur tiap WO (RBAC per tenant — keputusan K12)

Kebijakan "siapa boleh apa" berbeda di tiap WO, jadi peran **tidak lagi dikodekan tetap**
(saat ini Owner/Admin/Staff/Sales tertanam di kode, ADR-0017). Model idealnya:

- **Katalog izin tetap** milik ElProof (kunci stabil, dikelompokkan), misalnya:
  - Proyek: `project.view`, `project.create`, `project.edit`, `project.change_status`, `project.archive`, `project.delete`, `project.duplicate`
  - Keuangan: `finance.client.view`, `finance.client.record`, `finance.vendor.view`, `finance.vendor.record`, `finance.margin.view`, `finance.payment.delete`
  - Klien: `client.manage`, `client.invite`, `client.reset_access`
  - Direktori: `vendor.view`, `vendor.manage`, `venue.view`, `venue.manage`, `directory.import_export`
  - Pengaturan: `team.manage`, `role.manage`, `subscription.manage`, `business_profile.manage`, `template.manage`, `support_access.grant`
  - Dasbor: `dashboard.view`, `dashboard.revenue.view`
- **Cakupan data** per izin proyek: *semua proyek* atau *hanya proyek yang saya pegang* (PIC). Ini menggantikan aturan khusus Staff/Sales yang sekarang tertanam di kode.
- **Peran = kumpulan izin + cakupan**, dikelola WO di **Pengaturan → Peran & Akses** (matriks centang per grup, dengan deskripsi awam tiap izin).
  - **Owner**: peran sistem, semua izin, tidak bisa diubah/dihapus (mencegah WO mengunci diri sendiri).
  - **Templat bawaan** yang bisa diedit dan diduplikasi: Admin, Wedding Planner (cakupan: proyek saya), Sales (cakupan: proyek saya, boleh membuat proyek), Keuangan (baru).
  - WO bisa membuat peran kustom (mis. "Koordinator Lapangan").
- **Akses klien juga peran**, dengan katalog izin portal: `portal.finance.view` (nilai kontrak, tagihan, riwayat bayar), `portal.vendor.view`, `portal.timeline.view`, `portal.documents.view`, `portal.issues.view`.
  - Templat bawaan: **Pengantin** (semua izin portal) dan **Keluarga** (tanpa keuangan), keduanya bisa diedit WO. Peran klien kustom diperbolehkan (mis. "Orang tua — penanggung biaya" dengan keuangan).
- **Ditegakkan di backend**: setiap endpoint memeriksa izin + cakupan dari peran yang disimpan di DB (di-cache per sesi), bukan dari string role di kode. UI hanya menyembunyikan apa yang memang ditolak API.
- **Batas paket** (entitlement) tetap berlaku di atas RBAC: izin tidak bisa membuka fitur yang tidak ada di paket.
- Perubahan peran tercatat di audit log tenant (siapa mengubah izin apa).

### 2.6 Alur kunci
1. **Daftar & onboarding**: lihat PLAN.md §3 (Google → workspace → trial → checklist).
2. **Profil Usaha (self-service)**: logo, warna brand (preset), alamat, No. WhatsApp, rekening tujuan pembayaran klien (ditampilkan di portal), permintaan custom domain (diproses tim).
3. **Buat proyek** (halaman penuh, 2 langkah): (1) pasangan, tanggal, paket, nilai kontrak, PIC; (2) venue dari direktori + **jadwal termin klien** (template DP/Termin/Pelunasan dengan tanggal relatif ke acara). Selesai → langsung ke Ringkasan proyek dengan checklist "Undang klien · Tambah vendor".
4. **Undang klien**: tab Klien → "Tambah Klien" (nama, peran, No. WhatsApp; username otomatis) → link atur kata sandi → **"Kirim via WhatsApp"**. Status undangan terlihat (Terkirim / Sudah aktif / Kedaluwarsa → kirim ulang).
5. **Undang staf**: Pengaturan → Tim → email Google + peran → link 7 hari via WhatsApp/salin.
6. **Langganan**: kartu paket dengan harga per bulan, batas, dan rekomendasi; QRIS dengan hitung mundur & masa aktif baru; riwayat + kuitansi PDF.

## 3. Klien (Portal, mobile-first)

### 3.1 Kondisi sekarang → masalah utama
- **Kebocoran data di API**: akun klien dibatasi hanya ke metode GET, sehingga bisa membaca pembayaran ke vendor/venue, nilai kontrak vendor, log aktivitas internal, dan **semua dokumen** termasuk yang tidak dibagikan (`projects/presentation/handler.go`). `GET /projects/me` juga membawa Tanggal Booking dan harga venue.
- Mendarat di tab Vendor. Tidak ada beranda, "yang perlu Anda lakukan", tagihan berikutnya, cara bayar, atau kontak WhatsApp PIC.
- Status Inggris internal bocor ("Negotiation", "Critical", "In Resolution").
- Galat apa pun melempar ke halaman login.

### 3.2 Susunan ideal

**Bottom tab bar (HP)**: Beranda · Jadwal · Pembayaran · Vendor · Lainnya (Dokumen, Kendala, Bagikan, Akun). Di desktop menjadi navigasi atas.

**Beranda** (berurutan):
1. Header foto + nama pasangan (serif) + hitung mundur "H-45".
2. **Yang perlu Anda lakukan**: tagihan jatuh tempo, keputusan yang diminta WO. Penanda merah **hanya** di sini dan hanya bila klien harus bertindak.
3. **Tagihan berikutnya**: nominal, jatuh tempo, **rekening/cara bayar** dari Profil Usaha, tombol "Konfirmasi via WhatsApp".
4. **Progres persiapan**: tahapan vertikal ala pelacak pesanan (Sedang dipersiapkan → Persiapan akhir → Hari bahagia Anda → Selesai).
5. **Kabar terbaru**: item timeline/vendor/dokumen yang baru dibagikan.
6. **Kontak PIC**: foto, nama, tombol WhatsApp (`wa.me`) & telepon.
7. Dokumen penting (rundown, buku acara).

**Kendala** untuk klien selalu memuat tiga hal: apa yang terjadi, apa yang sedang dilakukan WO, kapan kabar berikutnya. Warna kuning/netral, bukan merah. Dampak "Critical" tidak pernah tampil mentah.

**Lainnya**: petunjuk "Tambahkan ke Layar Utama" (PWA, termasuk panduan iOS), ganti kata sandi sendiri, "Bagikan ke keluarga".

**Galat**: halaman "Gagal memuat — Coba lagi / Hubungi WO", bukan logout.

### 3.3 Data per peran klien (ditegakkan di API)

Apa yang dilihat tiap klien ditentukan oleh **peran klien yang diatur WO** (§2.5). Tabel di
bawah adalah templat bawaan; batas paling bawah berlaku mutlak untuk semua peran klien.

| Data | Pengantin (bawaan) | Keluarga (bawaan) |
|---|---|---|
| Ringkasan, jadwal, vendor (nama, kategori, status label klien) | ✓ | ✓ |
| Nilai kontrak, tagihan, riwayat pembayaran klien (`portal.finance.view`) | ✓ | ✗ (bisa diubah WO) |
| Dokumen & kendala yang ditandai "Terlihat klien" | ✓ | ✓ |
| **Tidak pernah** untuk peran klien mana pun: harga/pembayaran vendor & venue, margin, tanggal booking, log aktivitas, dokumen internal | ✗ | ✗ |

## 4. Pemetaan label status (internal → staf → klien)

| Enum (DB) | Label staf | Label klien |
|---|---|---|
| **Proyek** `Draft` | Draf | *(tidak tampil)* |
| `Preparation` | Persiapan | Sedang dipersiapkan |
| `Ready` | Siap | Persiapan akhir |
| `Completed` | Selesai | Selesai, terima kasih |
| `Cancelled` | Dibatalkan | Dibatalkan |
| **Vendor (engagement)** `Planned` | Direncanakan | Sedang dipilih WO |
| `Negotiation` | Negosiasi | Sedang dipilih WO |
| `Booked` | Dipesan | Sudah dipesan |
| `DP Paid` | DP dibayar | Sudah dipesan |
| `In Progress` | Dikerjakan | Sedang disiapkan |
| `Fully Paid` | Lunas | Sudah dipesan |
| `Ready` | Siap | Siap |
| `Completed` | Selesai | Selesai |
| `Cancelled` | Dibatalkan | *(tidak tampil, atau "Diganti WO")* |
| **Timeline** `Not Started` | Belum mulai | Akan datang |
| `In Progress` | Berjalan | Sedang berjalan |
| `Completed` | Selesai | Selesai |
| `Blocked` | Terhambat | Sedang berjalan |
| `Cancelled` | Dibatalkan | *(tidak tampil)* |
| *(turunan)* lewat tenggat | Lewat tenggat | Sedang berjalan |
| **Kendala** `Open` | Baru | Sedang kami periksa |
| `In Review` | Ditinjau | Sedang kami periksa |
| `In Resolution` | Ditangani | Sedang ditangani WO |
| `Resolved` | Teratasi | Sudah beres |
| `Closed` | Ditutup | Sudah beres |
| **Dampak** `Low`/`Medium`/`High`/`Critical` | Rendah/Sedang/Tinggi/Kritis | *(tidak tampil)* |
| **Termin** *(baru, turunan)* belum jatuh tempo / ≤ 7 hari / sebagian / terlambat / lunas | Belum jatuh tempo / Segera jatuh tempo / Dibayar sebagian / Terlambat / Lunas | Menunggu pembayaran / Jatuh tempo [tgl] / Dibayar sebagian, sisa Rp… / Lewat jatuh tempo — hubungi PIC / Lunas |
| **Langganan tenant** `trialing`/`active`/`past_due`/`read_only`/`suspended` | Masa coba / Aktif / Menunggu pembayaran / Mode baca-saja / Ditangguhkan | — |

Label disimpan di satu modul (`shared/lib/labels.ts` atau sejenis), dipakai semua badge; enum DB tetap Inggris agar tidak memicu migrasi besar.

## 5. Kesenjangan as-is → ideal (ringkas, untuk PLAN.md)

| # | Kesenjangan | Peran | Prioritas |
|---|---|---|---|
| G1 | Kebocoran data klien di API | Klien | **P0** |
| G2 | Langganan/suspend/nonaktif tidak ditegakkan | Semua | **P0** |
| G3 | Role admin platform tanpa pembeda, tanpa 2FA & audit | Platform | **P0** |
| G4 | Tidak ada registrasi, undangan, ganti/lupa sandi | WO, Klien | P1 |
| G5 | Tidak ada detail tenant 360°, metrik SaaS di backend, state machine langganan | Platform | P1 |
| G6 | Paket tanpa versi harga & entitlement | Platform, WO | P1 |
| G7 | Dashboard & Detail Proyek tidak berbasis tindakan; Kendala tanpa tab; tanpa jadwal termin | WO | P1 |
| G8 | Portal tanpa beranda/kontak/tagihan berikutnya; label Inggris | Klien | P1 |
| G9 | Profil Usaha self-service, pusat notifikasi, pengumuman | WO, Platform | P2 |
| G10 | Impersonasi aman, catatan internal, permintaan data PDP | Platform | P2 |
| G11 | Design system dua kepadatan, pola modal/drawer/halaman, toast | Semua | P1 (fondasi UI) |
| G12 | Peran staf & klien tertanam di kode; WO tidak bisa mengatur "siapa boleh apa" | WO, Klien | P1 |

## 6. Sumber (diakses 5–6 Nov 2026)

- Stripe Docs: subscriptions overview, billing analytics, manage prices, prorations, team roles
- ChartMogul (MRR movements), Chargebee (price change, offline dunning)
- WorkOS Impersonation; Zendesk account assumption; OWASP Logging Cheat Sheet; OWASP ASVS V4.3
- Shopify Polaris (index filters, resource details layout, toast); Primer (confirmation dialog, saving); Atlassian (flag); Carbon (modal)
- HoneyBook help (navigate a project, client portal), HoneyBook blog (new client portal 2025/2026), Planning Pod client portal, Dubsado client portals (Jul 2026), Aisle Planner (project management, budget manager), monday.com event planning, Zola app
- NN/g: status tracker, data visualization dashboards, modal vs nonmodal, skeleton screens, empty states
- GOV.UK Design System (task list, form structure); web.dev PWA installation; MDN Web Share
- Linear design refresh (12 Mar 2026); Tokopedia status pemesanan
- UU 27/2022 PDP, PP 33/2026 (sekunder: siplawfirm.id, bisnis.com)
