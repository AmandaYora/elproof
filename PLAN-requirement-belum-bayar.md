# PLAN — Kwitansi & Tagihan PDF untuk Pembayaran Client (pendekatan agile)

> Disusun sebagai senior business analyst + senior system analyst. Menggantikan seluruh isi
> `PLAN.md` sebelumnya (rancangan Invoice/Termin penuh) — hasil diskusi bersama user yang memilih
> pendekatan lebih ramping: **tidak ada jadwal termin tersimpan**, dokumen (Kwitansi/Tagihan)
> di-generate on-demand sebagai PDF dari kondisi pembayaran yang sudah ada.

> ## ⚠️ Hasil Verifikasi Mendalam (irisan modul/fitur/perhitungan) — sebelum implementasi
>
> Rancangan di bawah sudah ditelusuri ulang terhadap kode nyata (SQL `UPDATE projects` yang
> sebenarnya, `ProjectService.Duplicate`, ADR-0007, dan 3 tempat berbeda yang sudah menghitung
> "Sisa Tagihan" independen). Ditemukan **2 bug nyata** dan beberapa gap yang sudah diperbaiki
> langsung di badan plan ini (ditandai kotak "KOREKSI"/"✅ dikonfirmasi aman" di section terkait):
>
> 1. 🐛 **Pelanggaran ADR-0007** — draf awal §3.4 menyuruh frontend menghitung ulang status
>    (termasuk perbandingan `next_due_date` vs "hari ini") di sisi client. ADR-0007 eksplisit:
>    *"never a client-supplied 'current date'"* — backend adalah **satu-satunya** sumber "sekarang".
>    Diperbaiki: status **wajib** dihitung backend, dikembalikan sebagai string jadi di response;
>    frontend tinggal render, tidak pernah menghitung ulang tanggal.
> 2. 🐛 **`next_due_date` basi tidak pernah benar-benar dikosongkan** — draf awal bilang "otomatis
>    dikosongkan" tapi tidak ada satu pun write-path yang benar-benar mengeksekusi itu. Kalau
>    dibiarkan: project yang sudah lunas, lalu suatu saat Sisa Tagihan jadi positif lagi (mis. ada
>    `Refund` atau pembayaran di-edit/dihapus), bisa **tiba-tiba ke-flag Overdue** karena tanggal
>    jatuh tempo basi dari jauh-jauh hari. Diperbaiki di §2.2/§3.3: `next_due_date` dibersihkan
>    server-side setiap kali mutasi `client_payments` (Create/Update/Delete) membuat Sisa Tagihan
>    ≤ 0, bukan dibiarkan "otomatis" tanpa kode yang menjalankannya.
> 3. **Formula status ambigu** (§2.3) — kalau dibaca literal, aturan "Overdue menimpa status di
>    atas" bisa berarti override bahkan saat sudah `Paid`. Diperbaiki jadi urutan evaluasi
>    eksplisit dengan short-circuit di `Paid`.
> 4. **`status` salah level** — draf awal menaruh field `status` di response tiap baris
>    `client_payments` (§3.3), padahal status adalah konsep **level project** (agregat kontrak vs
>    total bayar), bukan per baris pembayaran. Dipindah ke response `GET /projects/{id}`.
> 5. **Semantik "Kwitansi" untuk baris `Refund`** — Kwitansi berarti "kami terima uang"; sebuah
>    baris `Refund` adalah uang **keluar** dari WO ke client, jadi tombol yang sama tidak boleh
>    memberi judul dokumen yang sama. Diperbaiki di §3.4.
> 6. **Gap**: frontend `ActivityType` union (`apps/web/.../types.ts`) belum disebut perlu tambah
>    `"invoice_generated"` — kelas kesalahan yang sama seperti yang sudah pernah ditemukan &
>    diperbaiki di fitur payment sebelumnya. Ditambahkan ke §3.4.
> 7. **Risiko deployment**: library PDF wajib pure-Go (tanpa CGO) — dev machine ini sudah pernah
>    kejumpa masalah toolchain CGO (lihat `DATABASE_GUIDE.md` soal `sqlc`), dan image produksi
>    Alpine minimal (`docs/DEPLOYMENT.md`). Dijadikan syarat eksplisit di §2.8, bukan asumsi.
> 8. ✅ **Dikonfirmasi AMAN, tidak perlu perubahan**: `ProjectRepository.Update`'s SQL
>    (`mysql_project_repository.go:262-271`) sudah pakai column-list eksplisit (tidak
>    `SELECT *`/overwrite semua kolom) — edit Project biasa lewat `PATCH /projects/{id}` **tidak**
>    akan menyentuh `next_due_date`, **asal** kolom itu tidak pernah ditambahkan ke query itu (harus
>    tetap lewat method terpisah `SetNextDueDate`, pola yang sama dengan `SetStatus`/`SetArchived`
>    yang sudah ada). Ditegaskan sebagai constraint eksplisit di §3.3, bukan dianggap otomatis benar.
> 9. ✅ **Dikonfirmasi AMAN**: `ProjectService.Duplicate` (ADR-0014,
>    `project_service.go:432-458`) membangun `domain.Project{}` baru dari whitelist field `input`,
>    tidak clone struct project sumber — project hasil duplikat otomatis `NextDueDate = nil`,
>    konsisten dengan payments yang juga tidak ikut ter-duplikasi.
> 10. ✅ **Dikonfirmasi AMAN**: endpoint Tagihan (`POST`) otomatis tertolak untuk principal
>     `client` oleh `resolveProjectAccess`'s aturan yang sudah ada ("Only GET is ever allowed for a
>     client principal") — **tidak perlu kode tambahan**, tapi endpoint baru **wajib** didaftar
>     lewat dispatcher `Handler.Item` yang sama, bukan route terpisah yang melewati choke point itu.
> 11. **✔️ KEPUTUSAN (dikonfirmasi user) — badge hanya di tab Pembayaran.** "Sisa Tagihan" sudah
>     dihitung independen di **3 tempat berbeda** hari ini (`ClientProjectHeaderCard.tsx`,
>     `ProjectHeaderCard.tsx`, tab Pembayaran); badge status/Overdue baru **tidak** ditambahkan ke 2
>     header persisten itu — scope minimal sesuai semangat agile. Kedua header tetap seperti sekarang
>     (hanya Sisa Tagihan). Perluasan ke header bisa menyusul di iterasi berikutnya kalau dirasa
>     perlu, tanpa perubahan backend (field-nya sudah ada di `currentProject`).
> 12. **Di luar scope, dikonfirmasi sadar (bukan kelupaan)**: Dashboard "attention queue" yang
>     sudah ada (evidence belum lengkap, dll.) **tidak** menyertakan sinyal Overdue baru ini — sudah
>     dicek, tidak ada intersection struktural yang rusak, cuma memang belum ada visibilitasnya di
>     dashboard. Diputuskan tetap di luar scope v1 ini.
>
> ### Putaran 2 — verifikasi terhadap write-path, dispatcher, storage, dan jalur Client Portal
>
> Putaran 1 memverifikasi *pembacaan* (SQL Update, Duplicate, ADR-0007). Putaran 2 membuka file yang
> belum pernah dibaca: `client_payment_service.go`, `handler.go`'s dispatcher penuh,
> `tenant_service.go`'s storage, DTO project, dan store frontend Client Portal. Hasilnya **2 blocker
> nyata** yang akan menghentikan/merusak implementasi kalau tidak diperbaiki dulu:
>
> 13. 🚧→✔️ **BLOCKER (SUDAH DIPUTUSKAN) — koreksi §2.2 (putaran 1) tidak bisa dieksekusi apa
>     adanya.** `ClientPaymentService` sama sekali tidak punya akses ke Project: struct-nya cuma
>     `{repo, evidence, activity}` (`client_payment_service.go:19-27`). Tidak ada `ProjectRepository`
>     untuk memanggil `SetNextDueDate`, **dan** tidak ada `contract_value` untuk menghitung Sisa
>     Tagihan. **Keputusan user: Opsi A** — tambah 1 dependency `ProjectRepository` (masih 1 modul
>     yang sama, tidak melanggar boundary). Detail di §2.2 dan §3.3.
> 14. 🚧 **BLOCKER — bentuk kontrak branding salah, logo akan diam-diam tidak muncul.** Draf awal
>     mengembalikan `logoPath *string` dari `platform`. Path itu menunjuk object storage milik modul
>     `platform`; `projects` tidak boleh membukanya sendiri (pelanggaran boundary, setara cross-module
>     join) — dan kalau dipaksa pun tidak ada error, logo cuma hilang dari PDF. `platform` sudah punya
>     `TenantService.DownloadLogo` (`tenant_service.go:351`), jadi kontrak harus mengembalikan
>     **bytes** (+ `BrandColorPreset` yang gratis sekalian). Diperbaiki di §2.7 dan §3.2.
> 15. 🐛 **Error validasi Tagihan akan hilang di UI.** `GenerateInvoiceModal` jadi satu-satunya
>     pemakaian `responseType: "blob"` yang punya input user, jadi satu-satunya yang bisa menerima
>     error JSON — yang oleh axios dibungkus jadi `Blob`, membuat `err.response.data.message` jadi
>     `undefined`. Semua pemakaian blob yang ada hari ini tidak punya input user sehingga pola ini
>     belum pernah teruji. Perbaikan (unwrap `Blob` → `.text()` → `JSON.parse`) di §3.4.
> 16. ✅ **Dikonfirmasi AMAN — dispatcher route tidak bentrok, dan preseden bentuknya sudah ada.**
>     `client-payments/{paymentId}/receipt` (`len(rest)==3`) meniru persis `evidence/{id}/file`
>     (`handler.go:183`); `client-payments/invoice` + `POST` (`len(rest)==2`) tidak bertabrakan
>     karena case `len(rest)==2` yang ada hanya PATCH/DELETE. Detail di §3.3.
> 17. ✅ **Dikonfirmasi AMAN — satu DTO melayani dua surface.** `GET /projects/{id}` memakai
>     `toProjectResponse` **tunggal** tanpa filtering per-principal (`dto.go:46`), dan Client Portal
>     membaca project dari store yang **sama persis** dengan WO Console
>     (`PembayaranTabPage.tsx:20` → `useProjectStore.currentProject`). Jadi menambah `status`/
>     `nextDueDate` di satu tempat otomatis melayani WO Console **dan** Client Portal — memvalidasi
>     koreksi §3.3/§3.5, tidak ada flow yang terputus di sisi client.
> 18. ✅ **Dikonfirmasi AMAN — preseden server-time persis sudah ada.** `getProject` sudah memanggil
>     `h.projects.ComputeProgress(r.Context(), tenantID, projectID, time.Now())`
>     (`project_endpoints.go:202`). `ComputeClientPaymentStatus(..., time.Now())` tinggal mengikuti
>     pola yang identik di handler yang sama — menguatkan koreksi ADR-0007 di putaran 1, bukan
>     memperkenalkan pola baru.
> 19. ✅ **Dikonfirmasi — penomoran migration benar.** Migration terakhir memang `000035`
>     (`add_general_evidence_context`), jadi yang baru `000036`.
> 20. ⚠️ **Koreksi atas koreksi saya sendiri (putaran 1, poin 6).** Saya sempat bilang tanpa
>     penambahan union `ActivityType`, entry akan "jatuh ke fallback/label kosong". Itu berlebihan —
>     tidak ada `.tsx` yang memetakan `type` ke label; tab Aktivitas merender `description` dari
>     server. Penambahannya tetap wajib, tapi murni type-correctness, bukan bug rendering.

## 0. Tujuan Bisnis

Memberi WO dan client satu cara resmi untuk:
1. **Kwitansi** — bukti terima untuk **satu pembayaran yang sudah tercatat** (dokumen retrospektif).
2. **Tagihan** — permintaan bayar berdasarkan **Sisa Tagihan saat ini** (dokumen prospektif, opsional
   pakai jatuh tempo & jumlah spesifik, bukan seluruh sisa).

Tanpa membangun sistem invoice/akunting penuh: **tidak ada tabel jadwal termin, tidak ada tabel
Invoice terpisah dari Project, tidak ada Invoice Items (rincian per biaya)** — semua sudah dibahas
dan sengaja dicoret dari rancangan awal (lihat riwayat diskusi). Project itu sendiri, dengan
`contractValue`-nya, sudah menjadi "invoice"-nya secara konseptual.

## 1. Kondisi saat ini (sudah dikonfirmasi lewat audit kode)

- **`client_payments`** (modul `projects`) sudah jadi ledger lengkap: `type`, `amount`,
  `payment_date`, `method`, `reference_number`, `notes` — sudah bisa Create/Edit/Delete (ADR-0019).
  Tidak ada kolom nomor dokumen, tidak ada kolom jatuh tempo di tabel ini maupun di `projects`.
- **"Sisa Tagihan"** sudah dihitung live di 2 tempat berbeda dengan rumus identik (`contractValue −
  Σ client_payments`, `Refund` sebagai pengurang): WO Console
  (`ProjectHeaderCard.tsx`/`ClientPaymentsSection.tsx:62-67`) dan Client Portal
  (`PembayaranTabPage.tsx:33-38`). Tidak ada status (`Unpaid`/`Paid`/dst.) yang ditampilkan sama
  sekali hari ini — cuma tiga angka (Nilai Kontrak, Total Diterima, Sisa Tagihan).
- **Tidak ada PDF generation sama sekali** di seluruh codebase ElProof (dikonfirmasi ulang: nol
  dependency PDF di `apps/api/go.mod`). Precedent yang paling dekat adalah **export Excel**
  (`excelize`, modul `vendors`) — backend-generated, di-stream sebagai file, bukan JSON.
- **Tenant branding** (nama bisnis + logo) dimiliki modul `platform`
  (`domain.Tenant.BusinessName`/`LogoStoragePath`), **belum pernah diresolusi dari modul `projects`**
  — frontend memanggil `/tenants/me/branding` sendiri (`useTenantBrandingStore`). Modul `platform`
  hari ini **tidak punya package `contracts` sama sekali** (beda dengan `projects`/`vendors` yang
  sudah punya) — belum ada modul lain yang butuh apa pun dari `platform`.
- **Pola resolver lintas-modul sudah mapan** di `projects/application/project_service.go`:
  `VenueResolver`, `StaffNameResolver` — interface lokal sempit, di-bridge dari `main.go`. Build
  order di `main.go`: `platformModule` (baris 210) dibangun **sebelum** `projectsModule` (baris
  217) — artinya kalau `projects` butuh sesuatu dari `platform`, itu bisa jadi **constructor
  argument biasa**, tidak perlu setter dua-fase (`platform` tidak butuh apa pun balik dari `projects`
  untuk kebutuhan ini — beda dari `SetProjects`/`SetVendors` yang sudah ada untuk keperluan lain).
- **`activity_log`** (append-only) sudah ada, dengan konvensi ActivityType generik lintas jenis
  (`payment_recorded`/`payment_updated`/`payment_deleted`) — pola yang sama bisa dipakai untuk
  mencatat penerbitan Tagihan.
- **RBAC** hari ini: Create/Edit/Delete `client_payments` **tidak digate role khusus** kecuali Delete
  yang Owner-or-Admin (ADR-0019). Tidak ada aksi finansial di modul `projects` yang dibatasi role
  selain hard-delete.

## 2. Keputusan Desain

1. **Tidak ada tabel baru untuk jadwal/invoice.** Kwitansi dan Tagihan murni **hasil generate**, bukan
   entity tersimpan. Ini keputusan inti yang membedakan dari rancangan awal.
2. **Satu kolom baru, opsional**: `projects.next_due_date DATE NULL`. Satu-satunya state baru yang
   persisten di seluruh fitur ini — dipakai **hanya** untuk menghitung status `Overdue`. Diisi
   **ad-hoc** saat Tagihan di-generate (bukan bagian dari jadwal bertahap).
   **🐛 KOREKSI (temuan verifikasi)**: draf awal cuma bilang "otomatis dikosongkan" tanpa write-path
   nyata — kalau dibiarkan begitu, kolom ini jadi basi selamanya begitu Sisa Tagihan pernah 0, dan
   bisa memicu Overdue palsu kalau Sisa Tagihan jadi positif lagi di kemudian hari (mis. `Refund`,
   atau pembayaran di-edit/dihapus). Diperbaiki: **setiap** mutasi `client_payments`
   (`Create`/`Update`/`Delete`, ketiganya — bukan cuma salah satu) menghitung ulang Sisa Tagihan di
   akhir operasinya; kalau hasilnya ≤ 0, panggil `ProjectRepository.SetNextDueDate(ctx, projectID,
   nil)` di operasi yang sama. Ini kode nyata yang harus ditulis di `ClientPaymentService`, bukan
   dianggap "otomatis" terjadi sendiri.

   **🚧 BLOCKER (verifikasi putaran 2 — koreksi di atas belum bisa dieksekusi apa adanya)**:
   `ClientPaymentService` **tidak punya akses apa pun ke Project**. Struct-nya hari ini
   (`client_payment_service.go:19-27`) cuma `{repo ClientPaymentRepository, evidence
   *EvidenceService, activity *ActivityService}` — tidak ada `ProjectRepository`, jadi
   `SetNextDueDate` tidak bisa dipanggil, **dan** tidak ada akses ke `contract_value` yang justru
   dibutuhkan untuk menghitung Sisa Tagihan.

   **✔️ KEPUTUSAN (dikonfirmasi user) — Opsi A**: tambah 1 dependency `ProjectRepository` ke
   `ClientPaymentService` + ubah `NewClientPaymentService(...)` di `projects.module.go:44`. Masih
   **satu modul yang sama**, jadi tidak melanggar boundary modular monolith sama sekali (bukan
   cross-module). Outstanding dihitung dari `repo.ListByProject` (sudah ada) + `contract_value`
   dari `ProjectRepository.FindByID`. Alasan dipilih: biayanya cuma 1 field + 1 baris wiring, dan
   menutup edge case Refund yang di ElProof bukan hipotetis (tipe `Refund` sudah ada di
   `client_payments` hari ini). *(Alternatif yang ditolak: tanpa clearing sama sekali, mengandalkan
   short-circuit `Paid` — lebih ramping tapi meninggalkan project lunas-lalu-di-Refund ter-flag
   `Overdue` memakai tanggal jatuh tempo lama.)*
3. **Status dihitung live di BACKEND SAJA, tidak pernah disimpan** — konsisten dengan konvensi
   ElProof untuk semua figur turunan lain (Sisa Tagihan, `evidenceComplete`), **dan wajib**
   berdasarkan ADR-0007 ("server adalah satu-satunya sumber 'sekarang'", tidak boleh
   client-supplied). Urutan evaluasi **eksplisit dengan short-circuit** (bukan "override" independen
   — **🐛 KOREKSI**: draf awal ambigu di titik ini, bisa dibaca seolah Overdue menimpa `Paid` juga):
   ```
   1. Sisa Tagihan ≤ 0                                          → Paid   (berhenti di sini)
   2. next_due_date terisi DAN next_due_date < hari ini (server) → Overdue (berhenti di sini)
   3. ada pembayaran masuk (Refund atau bukan)                  → Partially Paid
   4. selain itu (belum ada pembayaran sama sekali)              → Unpaid
   ```
   "Hari ini" **selalu** `time.Now()` sisi Go server (Asia/Jakarta, ADR-0007) — **tidak pernah**
   dihitung ulang di frontend dengan `Date.now()`/tanggal browser (lihat §3.4's koreksi).
4. **Kwitansi**:
   - Sumber: **1 baris `client_payments`** yang sudah ada — tidak ada data baru dibutuhkan.
   - Endpoint baru: `GET /projects/{id}/client-payments/{paymentId}/receipt` → stream PDF.
   - **Tanpa role gate** (murni membaca ulang data yang sudah bisa diakses siapa pun di project itu).
   - Tersedia di **WO Console dan Client Portal** — client bisa cetak kwitansinya sendiri.
   - **Tidak dicatat ke `activity_log`** — ini aksi baca/unduh, sama seperti download bukti evidence
     hari ini yang juga tidak menghasilkan entry activity log.
   - Nomor dokumen: `KWT/{projectId}/{paymentId}` — deterministik, tidak perlu sequence dikelola.
   - **🐛 KOREKSI (temuan verifikasi — semantik dokumen; ✔️ dikonfirmasi user)**: "Kwitansi" berarti "kami terima uang" —
     untuk baris ber-`type = "Refund"` ini semantiknya terbalik (uang **keluar** dari WO ke client).
     Endpoint & tombolnya tetap sama (masih 1 dokumen per baris `client_payments`), tapi judul/isi
     PDF-nya harus bercabang berdasarkan `Type`: `Refund` → judul **"Bukti Pengembalian Dana"**
     ("Telah dikembalikan kepada [Client]..."), selain itu → **"Kwitansi"** ("Telah diterima
     dari [Client]..."). Satu template, teks berkondisi — bukan endpoint terpisah.
5. **Tagihan**:
   - Sumber: kondisi Sisa Tagihan **saat ini** + parameter ad-hoc yang diisi admin saat generate:
     `amount` (jumlah yang diminta — default Sisa Tagihan penuh, bisa diedit misal cuma minta DP,
     divalidasi `> 0`; **tidak** dipaksa `≤ Sisa Tagihan` — staff boleh menagih jumlah berapa pun
     yang masuk akal menurut mereka, bukan aturan sistem untuk dibatasi),
     `dueDate` (opsional), `note` (opsional).
   - Endpoint baru: `POST /projects/{id}/client-payments/invoice` → body `{amount, dueDate?, note?}`
     → stream PDF. Kalau `dueDate` diisi, **menimpa** `projects.next_due_date` (bukan menambah baris
     baru — hanya ada 1 "tagihan aktif" pada satu waktu, sesuai semangat agile: tidak ada riwayat
     tagihan yang harus dijaga konsisten).
   - **Hanya di WO Console** — menagih adalah aksi WO ke client, bukan sebaliknya. Sudah otomatis
     tertolak untuk principal `client` lewat `resolveProjectAccess` yang sudah ada (lihat kotak
     verifikasi poin 10) — tidak perlu kode gate tambahan, cukup pastikan route-nya didaftar lewat
     dispatcher yang sama.
   - **Tanpa role gate** — konsisten dengan Create/Edit `client_payments` yang juga tidak digate;
     ini cuma dokumen, tidak mengubah data finansial apa pun selain `next_due_date` opsional.
   - **Dicatat ke `activity_log`** (ActivityType baru `invoice_generated`) — merepresentasikan
     permintaan formal ke client, beda dengan Kwitansi yang cuma re-print data yang sudah ada.
   - Tombol "Buat Tagihan" **disabled** kalau Sisa Tagihan ≤ 0 (tidak ada yang perlu ditagih).
   - Nomor dokumen: `TGH-{projectId}-{tanggal generate, YYYYMMDD}` — deterministik, tidak perlu
     sequence dikelola (sesuai tujuan awal: "bukan sistem accounting lengkap").
6. **PDF di-generate di backend (Go), bukan frontend** — mengikuti precedent export Excel yang
   sudah backend-generated (bukan pendekatan baru untuk codebase ini), memastikan hasil konsisten
   lintas device, dan branding tenant diresolusi otentik dari server (bukan dipercaya mentah dari
   body request frontend — mencegah client-side memalsukan nama/logo bisnis pada dokumen resminya
   sendiri).
7. **Bridge baru**: `TenantBrandingResolver` — interface sempit baru di
   `projects/application`, mengikuti pola `VenueResolver`/`StaffNameResolver` yang sudah ada persis.
   `platform` module butuh package `contracts` baru (belum pernah ada). Karena
   `platformModule` sudah dibangun lebih dulu di `main.go`, ini **constructor argument biasa**,
   **tidak perlu** setter dua-fase seperti `SetProjects`/`SetVendors`.

   **🚧 BLOCKER (verifikasi putaran 2 — bentuk kontrak salah)**: draf awal mengembalikan
   `logoPath *string`. **Path itu tidak berguna bagi `projects`** — logo tenant disimpan di object
   storage (ADR-0006) dengan key milik modul `platform`. `projects` memang punya storage client
   sendiri (dipakai `EvidenceService`, `projects.module.go:39`), tapi memakainya untuk membuka key
   milik `platform` = **pelanggaran boundary modular monolith** (satu modul membaca objek milik
   modul lain langsung, setara cross-module join). Kalau tetap dipaksa pakai path, hasilnya: logo
   **diam-diam tidak pernah muncul** di PDF, tanpa error.
   Perbaikan — kontrak harus mengembalikan **bytes**, bukan path, karena `platform` sudah punya
   `TenantService.DownloadLogo(ctx, tenantID) (io.ReadCloser, error)` (`tenant_service.go:351`)
   yang membuka storage-nya sendiri:
   ```go
   ResolveBranding(ctx context.Context, tenantID int64) (BrandingInfo, error)
   // BrandingInfo{ BusinessName string; BrandColorPreset string; Logo []byte /* nil kalau belum ada */ }
   ```
   `BrandColorPreset` ikut disertakan (sudah ada di `domain.Tenant`) supaya warna aksen PDF konsisten
   dengan branding tenant di UI — gratis, satu query yang sama.
8. **Library PDF**: dependency baru, belum ada satu pun di `go.mod` hari ini — pilih saat
   implementasi. **✔️ KEPUTUSAN (dikonfirmasi user): `github.com/johnfercher/maroto` v2** — API
   berbasis grid/row/col, jauh lebih cepat menyusun layout dokumen terstruktur (header, tabel
   rincian, footer) ketimbang menggambar manual per-koordinat. *(Alternatif yang ditolak:
   `github.com/go-pdf/fpdf`, fork terawat dari `jung-kurt/gofpdf` yang aslinya sudah diarsipkan —
   dependency paling minimal, tapi seluruh layout harus digambar manual.)*
   **🐛 KOREKSI (temuan verifikasi — constraint yang belum eksplisit)**: library **wajib pure-Go,
   tanpa CGO** — bukan preferensi, tapi syarat keras: dev machine ini punya keterbatasan toolchain
   CGO/MinGW yang sudah terdokumentasi (`knowledge/DATABASE_GUIDE.md`, alasan sqlc dipilih ketimbang
   driver ber-CGO), dan image produksi adalah Alpine minimal 2GB RAM (`docs/DEPLOYMENT.md`) yang
   tidak punya toolchain compiler untuk build ulang dependency ber-CGO. maroto v2 pure-Go — tetap
   **diverifikasi ulang** saat pin versi final sebelum `go get`, jangan diasumsikan tanpa cek.
9. **Di luar scope (disepakati eksplisit, sebagian dari daftar §7 spec awal user)**: Invoice Items,
   tabel jadwal termin bertahap, payment gateway/payment link, recurring/bulk invoice, e-meterai,
   accounting journal, credit note, sequence nomor dokumen yang dikelola ketat, reminder otomatis
   (WhatsApp/email) — generate PDF tetap aksi manual yang dipicu staff.

## 3. Perubahan Konkret

### 3.1 Database
- Migration baru (nomor lanjut dari `000035`): `ALTER TABLE projects ADD COLUMN next_due_date DATE
  NULL AFTER contract_value;` — satu-satunya perubahan skema di seluruh fitur ini.

### 3.2 Backend — bridge tenant branding
- **Baru**: `apps/api/internal/modules/platform/contracts/contracts.go` — package `contracts` baru
  untuk modul `platform` (belum pernah ada), berisi struct `BrandingInfo{BusinessName,
  BrandColorPreset string; Logo []byte}` dan interface `Contracts` dengan method
  `ResolveBranding(ctx, tenantID int64) (BrandingInfo, error)`, diimplementasikan oleh
  `platform.Module` dengan mendelegasikan ke `TenantService.Get` (nama/warna) +
  `TenantService.DownloadLogo` (bytes; `Logo = nil` kalau tenant belum punya logo — `DownloadLogo`
  mengembalikan `apperror.NotFound` dalam kasus itu, **harus ditelan jadi `nil`, bukan error**,
  supaya tenant tanpa logo tetap bisa cetak PDF).
- `apps/api/internal/modules/projects/application/document_service.go`: tambah interface baru
  ```go
  type TenantBrandingResolver interface {
      ResolveBranding(ctx context.Context, tenantID int64) (platformcontracts.BrandingInfo, error)
  }
  ```
  mirip persis `VenueResolver`/`StaffNameResolver` di `project_service.go`.
- `apps/api/cmd/server/main.go`: `projects.NewModule(...)` dapat parameter baru
  `platformModule.Contracts()` (constructor arg biasa, bukan setter — lihat §2.7).

### 3.3 Backend — generator dokumen
- **Baru**: `apps/api/internal/modules/projects/application/document_service.go` — service baru
  (bukan menumpuk di `ClientPaymentService` yang sudah padat), `DocumentService` dengan:
  - `GenerateReceipt(ctx, projectID, paymentID int64) ([]byte, error)` — ambil 1
    `domain.ClientPayment` + `domain.Project` + resolve branding via `TenantBrandingResolver`,
    render PDF Kwitansi.
  - `GenerateInvoice(ctx, projectID int64, actorStaffID int64, input GenerateInvoiceInput) ([]byte, error)`
    — `input = {Amount int64, DueDate *time.Time, Note string}`. Hitung Sisa Tagihan terkini, render
    PDF Tagihan, **update `projects.next_due_date`** kalau `DueDate` diisi (lewat
    `ProjectRepository.SetNextDueDate` baru), catat `activity.Record(..., ActivityInvoiceGenerated,
    ...)`.
  - Helper privat `buildPDF(...)` dipakai kedua fungsi di atas — layout dasar: header (logo + nama
    bisnis tenant), nomor dokumen, info project & client, tabel rincian (1 baris untuk Kwitansi,
    ringkasan Nilai Kontrak/Sudah Dibayar/Sisa/Diminta-kali-ini untuk Tagihan), footer catatan.
- `apps/api/internal/modules/projects/domain/project.go`: tambah field `NextDueDate *time.Time` ke
  `Project` struct.
- `apps/api/internal/modules/projects/domain/activity.go`: tambah `ActivityInvoiceGenerated
  ActivityType = "invoice_generated"`.
- `apps/api/internal/modules/projects/domain/payment.go` (atau file baru `client_payment_status.go`):
  fungsi murni baru
  ```go
  type ClientPaymentStatus string
  const (
      StatusUnpaid         ClientPaymentStatus = "Unpaid"
      StatusPartiallyPaid  ClientPaymentStatus = "PartiallyPaid"
      StatusPaid           ClientPaymentStatus = "Paid"
      StatusOverdue        ClientPaymentStatus = "Overdue"
  )
  func ComputeClientPaymentStatus(outstanding int64, hasAnyPayment bool, nextDueDate *time.Time, today time.Time) ClientPaymentStatus
  ```
  `today` **selalu** `time.Now()` sisi Go server (Asia/Jakarta) — dipanggil satu-satunya tempat oleh
  backend saat menyusun response `GET /projects/{id}`. **🐛 KOREKSI (temuan verifikasi — pelanggaran
  ADR-0007)**: draft sebelumnya berkata logic ini "didokumentasikan supaya frontend bisa hitung
  ulang identik" — itu salah arah; berbeda dengan Sisa Tagihan (murni aritmetika, tidak bergantung
  tanggal), formula ini butuh "hari ini" yang menurut ADR-0007 wajib server-side saja. Frontend
  **tidak** mem-port fungsi ini ke TS; ia cuma membaca string `status` jadi dari response (lihat
  koreksi §3.4).
- `apps/api/internal/modules/projects/infrastructure/mysql_project_repository.go`: tambah
  `SetNextDueDate(ctx, projectID int64, dueDate *time.Time) error`.
- `apps/api/internal/modules/projects/application/client_payment_service.go` +
  `apps/api/internal/modules/projects/projects.module.go:44` — **(putaran 2, konsekuensi Opsi A
  di §2.2)** tambah field `projects ProjectRepository` ke `ClientPaymentService`, ubah signature
  `NewClientPaymentService`, dan panggil clearing `SetNextDueDate(..., nil)` di akhir `Create`,
  `Update`, dan `Delete` ketika outstanding hasil hitung ulang ≤ 0. **Jangan** taruh logic ini di
  handler/presentation — ketiga method itu sudah jadi choke point yang benar.
- `apps/api/internal/modules/projects/presentation/`: tambah 2 route ke `handler.go`'s switch
  `Item` (**wajib** lewat switch itu, bukan route terpisah — lihat kotak verifikasi poin 10) +
  handler baru di file `document_endpoints.go`:
  - `case len(rest) == 3 && rest[0] == "client-payments" && rest[2] == "receipt" && GET` —
    ✅ **dikonfirmasi (putaran 2)**: bentuk `len(rest)==3` ini persis preseden
    `evidence/{id}/file` yang sudah ada (`handler.go:183`), dan **tidak bentrok** dengan case
    `client-payments` manapun (yang ada hanya `len(rest)==1` GET/POST dan `len(rest)==2`
    PATCH/DELETE). `Content-Type: application/pdf`, `Content-Disposition: inline;
    filename="Kwitansi-{paymentId}.pdf"`.
  - `case len(rest) == 2 && rest[0] == "client-payments" && rest[1] == "invoice" && POST` —
    ✅ **dikonfirmasi (putaran 2)**: tidak ada case `len(rest)==2` + `POST` untuk `client-payments`
    hari ini, jadi tidak ada tabrakan dengan pembacaan `rest[1]` sebagai paymentId (case PATCH/DELETE
    dibedakan oleh method). Body `{amount, dueDate, note}`, no role gate.
- `apps/api/internal/modules/projects/presentation/dto.go`: **🐛 KOREKSI (temuan verifikasi —
  response shape salah level)** — `status` (`ClientPaymentStatus`) **bukan** field per-baris
  `client_payments` (itu akan membuat setiap baris menampilkan nilai yang identik, karena status
  adalah agregat tingkat project, bukan properti satu pembayaran). `status` dan `nextDueDate`
  **keduanya** ditambahkan ke response `GET /projects/{id}` (project detail) saja, dihitung sekali
  di situ dari `ComputeClientPaymentStatus`. Endpoint/response `client_payments` list/detail tidak
  berubah sama sekali.

### 3.4 Frontend — WO Console
- `apps/web/src/shared/services/api-endpoints.ts`: tambah
  `clientPaymentReceipt(projectId, paymentId)` dan `clientPaymentInvoice(projectId)`.
- `apps/web/src/modules/projects/components/detail/ClientPaymentsSection.tsx`:
  - Tombol baru **"Cetak Kwitansi"** (icon `Receipt`) di kolom Aksi tiap baris (dan blok aksi
    `CardList` versi mobile — **dua-duanya**, jangan cuma tabel), tanpa gate — fetch PDF sebagai blob
    (`httpClient.get(..., {responseType: "blob"})`, pola yang sama dengan `EvidenceViewerModal`),
    buka di tab baru / trigger download. **🎨 KOREKSI UX**: draf awal menulis `tone="info"` — tone itu
    **sudah dipakai** "Lihat Bukti" (`Eye`) di baris yang sama, dua tombol `info` bersebelahan jadi
    sulit dibedakan sekilas. Pakai **`tone="navy"`** (satu-satunya tone yang belum terpakai di baris
    ini, dan navy = warna brand → pas untuk aksi "dokumen resmi").
  - Tombol baru **"Buat Tagihan"** di `CardHeader`'s `action` (sejajar "Tambah Pembayaran" — bungkus
    keduanya dalam `<div className="flex items-center gap-2">`, karena `action` sekarang cuma memuat
    1 tombol), **disabled** kalau `totalRemaining <= 0`.
  - Modal baru `GenerateInvoiceModal` — field `Jumlah Diminta` (default = Sisa Tagihan, pakai
    `CurrencyInput` yang sudah ada), `Jatuh Tempo` (opsional, date picker), `Catatan` (opsional).
    Submit → POST, lalu trigger download blob hasilnya.
  - Badge status (label Indonesia, lihat §3.6) ditambahkan di ringkasan,
    **🐛 KOREKSI (temuan verifikasi — pelanggaran ADR-0007)**: frontend **tidak pernah** menghitung
    ulang status ini sendiri — draft sebelumnya mengusulkan "port logic ke TS" yang berarti
    membandingkan `nextDueDate` terhadap tanggal browser (`Date.now()`), melanggar ADR-0007 ("Server
    time as the single source of 'now'"). Sebagai gantinya, frontend **hanya membaca** field
    `status` string yang sudah dihitung backend dari response `GET /projects/{id}` (`currentProject`
    yang sudah di-fetch tetap punya field ini setelah §3.3's koreksi) dan merender badge sesuai
    label — tidak ada perhitungan tanggal apa pun di sisi frontend.
- `apps/web/src/modules/projects/stores/useProjectStore.ts`: tambah action
  `downloadClientPaymentReceipt(projectId, paymentId)` dan
  `generateClientInvoice(projectId, input)` (keduanya return Blob, tidak mengubah state list —
  `generateClientInvoice` refetch `fetchProjects`/`currentProject` kalau `dueDate` diisi, supaya
  `nextDueDate`/`status` yang baru ter-refresh; **selalu** refetch juga kalau `dueDate` kosong,
  karena Tagihan yang baru dibuat tetap bisa mengubah `status` — misal dari `Unpaid` tidak berubah,
  tapi kalau ini pembayaran DP baru saja masuk lewat alur lain, `status` project bisa saja sudah
  berubah sejak fetch terakhir).
- `apps/web/src/modules/projects/types.ts:233-245` — tambahkan `"invoice_generated"` ke union
  `ActivityType`. **Catatan akurasi (verifikasi putaran 2)**: koreksi saya di putaran 1 sempat
  menyebut tanpa ini entry akan "jatuh ke fallback/label kosong" — itu **berlebihan**. Sudah dicek:
  tidak ada satu pun `.tsx` yang memetakan `type` → label; tab Aktivitas merender `description`
  kiriman server. Jadi penambahan union ini murni **type-correctness** (nilai API di-cast di
  boundary, `useProjectStore.ts:446`), **bukan** bug rendering — tetap wajib ditambah, tapi bukan
  blocker.
- **🐛 TEMUAN BARU (putaran 2) — error JSON tertelan jadi Blob**: `GenerateInvoiceModal` adalah
  **satu-satunya** pemakaian `responseType: "blob"` di codebase ini yang punya input user
  (`amount`), jadi satu-satunya yang bisa menerima **error validasi** dari backend. Dengan
  `responseType: "blob"`, axios membungkus body error JSON jadi `Blob` → `err.response.data.message`
  jadi `undefined` → user cuma lihat pesan error generik, pesan asli ("Jumlah harus lebih dari 0")
  hilang. Semua pemakaian blob yang ada hari ini (`evidenceFile`, export vendor/venue, logo,
  attachment) tidak punya input user sehingga tidak pernah kena kasus ini — jadi ini benar-benar
  jalur baru, bukan pola yang sudah terbukti. Perbaikan wajib di handler `catch`-nya:
  ```ts
  if (err.response?.data instanceof Blob) {
    const msg = JSON.parse(await err.response.data.text())?.message;
  }
  ```

### 3.5 Frontend — Client Portal
- `apps/web/src/modules/client-portal/pages/tabs/PembayaranTabPage.tsx`:
  - Tombol **"Cetak Kwitansi"** per kartu riwayat pembayaran (tidak ada "Buat Tagihan" — WO-only).
    Untuk baris `type === "Refund"`, label tombol & judul dokumen menyesuaikan (lihat koreksi §2.4)
    — tetap satu tombol/endpoint yang sama, hanya teksnya bercabang.
  - **🐛 Penting (lihat §3.6 poin 5)**: tombolnya **tidak boleh** ditaruh di dalam blok
    `{proofEvidence && ...}` yang sudah ada — blok itu hanya muncul kalau pembayaran punya bukti
    transfer, sedangkan kwitansi tidak bergantung evidence. Buat blok footer aksi sendiri yang
    selalu tampil di setiap kartu.
  - Badge status diletakkan **di dalam kartu amber "Sisa Tagihan"** (bukan kartu ke-4 — grid-nya
    `sm:grid-cols-3` dan sudah terisi persis 3, lihat §3.6 poin 3), dengan label Bahasa Indonesia
    (§3.6 poin 2). Sama seperti §3.4's koreksi: **dibaca langsung** dari field `status` pada
    `currentProject` (response `GET /projects/{id}`), tidak dihitung ulang di komponen ini.

### 3.6 UX & Familiaritas — hal yang menentukan fitur ini terasa "menyatu" atau "tempelan"

Ditelusuri langsung ke `ClientPaymentsSection.tsx`, `PembayaranTabPage.tsx`, dan
`IconActionButton.tsx`. Kesimpulan: **nol pola UI baru diperkenalkan** — semua yang dibutuhkan sudah
jadi idiom di codebase (`Modal`+`Field`+`CurrencyInput`+`Input type="date"`+`Textarea` persis seperti
`AddClientPaymentModal`; `IconActionButton` persis seperti Lihat Bukti/Ubah/Hapus; download blob
lewat `URL.createObjectURL` sudah dipakai di 6+ tempat). Tapi ada 5 detail yang kalau dilewatkan
akan membuat fitur ini terasa asing atau membingungkan:

1. **🐛 Risiko salah paham terbesar — "Buat Tagihan" ≠ mencatat pembayaran.** Di `CardHeader` yang
   sama akan berdiri 2 tombol: "Tambah Pembayaran" (**mengubah data**) dan "Buat Tagihan" (**cuma
   cetak PDF**, tidak menambah baris apa pun ke tabel). User sangat mungkin klik "Buat Tagihan" lalu
   bingung kenapa tabel tidak bertambah. Mitigasi wajib & murah: isi prop `description` milik `Modal`
   (sudah ada di **semua** modal codebase ini) dengan kalimat eksplisit —
   *"Dokumen ini hanya dicetak untuk dikirim ke client. Tidak ada pembayaran yang tercatat dari sini
   — catat pembayarannya lewat 'Tambah Pembayaran' setelah dana benar-benar masuk."*
2. **🌐 Label status harus Bahasa Indonesia.** Tab ini 100% berbahasa Indonesia ("Uang Masuk dari
   Client", "Sisa Tagihan", badge "Ada"/"Belum Ada"). Menampilkan `Unpaid`/`Partially Paid` mentah
   akan terasa asing. ElProof sudah punya konvensi persis untuk kasus ini — **identifier Inggris,
   label Indonesia** (Timeline, Tanggal Booking, Wedding Planner). Jadi: nilai API tetap
   `Unpaid|PartiallyPaid|Paid|Overdue`, label yang dirender:
   | Nilai API | Label UI | `Badge` tone |
   |---|---|---|
   | `Paid` | **Lunas** | `success` |
   | `PartiallyPaid` | **Dibayar Sebagian** | `info` |
   | `Unpaid` | **Belum Dibayar** | `neutral` |
   | `Overdue` | **Jatuh Tempo** | `danger` |
3. **📍 Letak badge — draf awal masih kabur ("di CardHeader/ringkasan").** Grid ringkasan adalah
   `sm:grid-cols-3` berisi **persis 3** kartu (Nilai Kontrak / Total Diterima / Sisa Tagihan);
   menambah kartu ke-4 akan membuat baris kedua yatim. Keputusan: badge diletakkan **di dalam kartu
   "Sisa Tagihan"**, tepat di bawah angkanya — status memang turunan langsung dari angka itu, dan
   grid tetap 3 kolom tanpa layout yang berubah. Sama persis di Client Portal (kartu amber "Sisa
   Tagihan"). Konsekuensi kecil: `SummaryStat` (komponen lokal di file yang sama) perlu 1 prop
   opsional `badge?: ReactNode`.
4. **⏳ "Cetak Kwitansi" adalah `IconActionButton` pertama yang punya latency.** Semua tombol baris
   yang ada hari ini instan (cuma membuka modal). Ini yang pertama menunggu round-trip server +
   render PDF. Tanpa feedback, user klik → "tidak terjadi apa-apa" 1-2 detik → klik lagi → 2 PDF
   ter-download. Tidak perlu komponen baru: `IconActionButton` **sudah punya** prop `disabled`
   beserta styling `disabled:opacity-40` — simpan `generatingId` di state dan disable tombol baris
   itu selama fetch.
5. **🐛 Client Portal — jangan taruh tombol di dalam blok yang kondisional.** Di
   `PembayaranTabPage.tsx`, baris "Dokumen Pendukung" (tempat tombol bukti berada) dibungkus
   `{proofEvidence && ...}` — **hanya muncul kalau pembayaran itu punya bukti transfer**. Kalau
   "Cetak Kwitansi" ditaruh di dalamnya, pembayaran tanpa bukti transfer tidak akan punya tombol
   cetak sama sekali — padahal kwitansi tidak bergantung pada evidence. Tombolnya **wajib** di blok
   footer aksi sendiri yang selalu tampil.

✅ **Dikonfirmasi tidak merusak janji surface**: Client Portal tetap read-only secara konsep —
mencetak kwitansi adalah aksi baca/unduh, tidak ada create/update/delete yang ditambahkan ke sana
(sesuai `PRODUCT_REQUIREMENTS.md`). "Buat Tagihan" tetap 100% absen dari Client Portal.

## 4. Urutan Implementasi

- [x] 1. Migration `000036_add_next_due_date_to_projects`.
- [x] 2. Backend: `platform/contracts` baru (`BrandingInfo` + `ResolveBranding` yang mengembalikan
      **bytes** logo) + wiring `main.go` (`TenantBrandingResolver`).
- [x] 3. Backend: domain (`Project.NextDueDate`, `ComputeClientPaymentStatus`,
      `ActivityInvoiceGenerated`) + `ProjectRepository.SetNextDueDate`.
- [x] 4. Backend: `ClientPaymentService` dapat dependency `ProjectRepository` (Opsi A, §2.2) +
      clearing `SetNextDueDate(nil)` di `Create`/`Update`/`Delete`; ubah wiring
      `projects.module.go:44`.
- [x] 5. Backend: `go get github.com/johnfercher/maroto/v2` (dikonfirmasi pure-Go — build sukses
      dengan `CGO_ENABLED=0`); `DocumentService` (Kwitansi dan Tagihan, satu helper `buildDocumentPDF`
      bersama).
- [x] 6. Backend: 2 endpoint baru + `dto.go` (`paymentStatus`, `nextDueDate` di response
      `GET /projects/{id}` — dan setiap handler single-project lainnya, lihat catatan verifikasi).
- [x] 7. Frontend: `api-endpoints.ts` + store actions (termasuk unwrap error `Blob`, §3.4).
- [x] 8. Frontend: WO Console (`ClientPaymentsSection.tsx` — tombol, modal, badge) + `ActivityType`
      union.
- [x] 9. Frontend: Client Portal (`PembayaranTabPage.tsx` — tombol, badge).
- [x] 10. Build/vet/typecheck (Go + `tsc --noEmit` + `npm run build`) — dijalankan berulang, terakhir
      bersih setelah perbaikan bug `omitempty` (lihat §5).
- [x] 11. Uji end-to-end terhadap data lokal nyata (lihat §5) — 2 putaran verifikasi lengkap,
      termasuk 1 bug nyata ditemukan & diperbaiki di putaran kedua.
- [ ] 12. Commit/push/deploy — **belum dilakukan**, menunggu instruksi eksplisit dari user (per
      instruksi sesi ini: "jangan dulu sampai melakukan commit, push").

## 5. Checklist Verifikasi

> **Status: seluruh implementasi selesai dan diverifikasi nyata** — bukan hanya baca kode, tapi
> lewat siklus API sungguhan (curl, login sebagai Owner/Wedding Planner/client/platform superadmin),
> query langsung ke DB lokal (MySQL), dan browser sungguhan (Playwright, screenshot). Detail bukti
> per item ditulis di baris masing-masing. Item yang tidak 100% bisa diuji literal di lingkungan
> lokal ini (mis. Docker tidak terpasang) ditandai dan diberi bukti pengganti terkuat yang tersedia.

- [x] Generate Kwitansi untuk 1 pembayaran → PDF valid, berisi data yang benar (nomor, tanggal,
      jumlah, jenis, metode, nama client/project, branding tenant yang benar — **bukan** branding
      ElProof sendiri). Diverifikasi: `pdftotext` atas PDF hasil `GET .../receipt` menunjukkan
      "WO Uji Coba", nomor `KWT/{projectId}/{paymentId}`, semua field pembayaran benar. *(Isolasi
      lintas-tenant tidak diuji dengan 2 tenant sungguhan di DB lokal ini — hanya 1 tenant tersedia
      — tapi dijamin arsitektural: `ResolveBranding` selalu di-scope ke `tenantID` dari klaim JWT
      pemanggil, dan project tenant lain sudah tidak bisa diakses sama sekali di layer
      `resolveProjectAccess`.)*
- [x] Generate Kwitansi dari Client Portal (sebagai client) → sukses, PDF sama seperti dari WO
      Console. Diverifikasi: login sebagai `client.test.hd`, `GET .../receipt` → 200, PDF valid.
- [x] Generate Tagihan dengan jumlah custom (bukan seluruh Sisa Tagihan) + jatuh tempo → PDF benar,
      `projects.next_due_date` ter-update di DB. Diverifikasi 2×: jumlah custom (10jt dari Sisa
      35jt) menghasilkan PDF dengan rincian benar, dan kolom `next_due_date` di DB berubah sesuai.
- [x] Setelah jatuh tempo terlewat (uji dengan tanggal masa lalu) dan belum ada pembayaran baru →
      status berubah jadi `Overdue`. Diverifikasi sebagai staff (`GET /projects/{id}`); sisi client
      dijamin identik karena `me()`/`GET /projects/me` memanggil `getProject` yang **sama persis**
      (satu DTO, dikonfirmasi lewat pembacaan kode `handler.go`).
- [x] Setelah Sisa Tagihan mencapai 0 (bayar lunas) → status `Paid` **dan** `next_due_date`
      benar-benar dikosongkan di DB. Diverifikasi berulang di 3 project berbeda, lewat ketiga jalur
      mutasi (lihat 2 baris di bawah) — termasuk query SQL langsung ke kolom `next_due_date`
      (`Valid: false`), bukan cuma membaca field `status` di response.
- [x] Tombol "Buat Tagihan" **disabled** ketika Sisa Tagihan ≤ 0. Diverifikasi programatik lewat
      Playwright (`button.isDisabled() === true`) dan validasi backend `amount<=0` → 422 terpisah.
- [x] Tombol "Buat Tagihan" **tidak muncul** di Client Portal sama sekali. Dikonfirmasi lewat kode
      (`PembayaranTabPage.tsx` tidak punya elemen ini) dan screenshot browser.
- [x] Generate Tagihan tercatat di `activity_log` (`invoice_generated`); generate Kwitansi **tidak**.
      Diverifikasi: `GET .../activity` menunjukkan entry "Tagihan Rp X diterbitkan", dan 2× download
      Kwitansi berturut-turut tidak menambah entry apa pun.
- [x] Kedua endpoint dokumen bisa diakses staff role manapun (tidak ada 403 karena role).
      Diverifikasi dengan **Owner** dan **Wedding Planner** (role `Staff`, akun aktif nyata di DB
      lokal) pada project miliknya sendiri — keduanya 200 di kedua endpoint. Admin tidak punya akun
      aktif di data lokal ini untuk diuji langsung, tapi kedua handler tidak punya percabangan role
      sama sekali (dikonfirmasi baca kode `document_endpoints.go`), jadi berlaku sama untuk Admin.
- [x] Kwitansi/Tagihan lintas-project (payment/project milik tenant lain) → 404/403, bukan bocor
      data. Diverifikasi: project/payment ID tak ada → 404, payment ID dari project lain → 404
      (bukan payment tenant lain, karena hanya 1 tenant tersedia — tapi Wedding Planner mengakses
      project yang bukan PIC-nya juga dikonfirmasi 403 lewat gate `resolveProjectAccess` yang sama).
- [x] `go build ./... && go vet ./...`, `npx tsc --noEmit`, `npm run build` bersih. Dijalankan
      berulang kali di sesi ini, termasuk setelah perbaikan bug `omitempty` (lihat catatan di bawah).
- [x] **(temuan verifikasi)** Bayar lunas (Sisa Tagihan → 0) **setelah** sebuah `next_due_date` di
      masa depan sudah pernah di-set oleh Tagihan sebelumnya → `SetNextDueDate(ctx, projectID, nil)`
      benar-benar terpanggil dari `ClientPaymentService`. Diuji lewat **ketiga** jalur mutasi secara
      terpisah: **Create** (pembayaran pelunasan baru), **Update** (PATCH nominal jadi lunas), dan
      **Delete** (hapus baris `Refund` yang jadi penghalang pelunasan) — dikonfirmasi ke kolom DB
      langsung setiap kali, bukan cuma field response.
- [x] **(temuan verifikasi)** Project dengan `next_due_date` di masa lalu tapi Sisa Tagihan sudah 0
      (kasus race/lupa-clear) → status tetap `Paid`, **bukan** `Overdue`. Diverifikasi di 3 project
      terpisah — short-circuit §2.3 (Paid dicek duluan) terbukti berjalan sesuai urutan.
- [x] **(temuan verifikasi)** Cetak Kwitansi untuk baris `type === "Refund"` → judul/isi PDF
      "Bukti Pengembalian Dana" (bukan "Kwitansi" standar) — konsisten di WO Console **dan** Client
      Portal. Diverifikasi di kedua sisi lewat `pdftotext`: filename `Bukti-Pengembalian-Dana-{id}.pdf`,
      judul dan kalimat body benar di keduanya.
- [x] **(temuan verifikasi)** Ubah tanggal sistem/timezone browser secara manual lalu buka Pembayaran
      tab → badge status **tidak berubah**. Diverifikasi lebih kuat dari sekadar uji klik: `grep`
      atas kedua file frontend (`ClientPaymentsSection.tsx`, `PembayaranTabPage.tsx`) mengonfirmasi
      **tidak ada satu pun** referensi `Date.now()`/`new Date()` terkait status — keduanya cuma
      merender `project.paymentStatus` yang sudah jadi dari backend, sehingga skenario ini
      *tidak mungkin* terjadi by construction, bukan cuma "diuji sekali dan kebetulan benar".
- [x] **(temuan verifikasi)** `"invoice_generated"` muncul dengan label yang benar (bukan fallback
      generik) di tab Aktivitas project setelah generate Tagihan. Diverifikasi via screenshot
      browser sungguhan: entry "Budi Rahman — Tagihan Rp 10.000.000 diterbitkan" tampil normal.
- [x] **(temuan verifikasi)** `go get` dependency PDF final → konfirmasi `go.sum`/dokumentasi
      library tidak memerlukan CGO, dan build image Docker Alpine produksi tetap sukses tanpa
      compiler tambahan. Docker tidak terpasang di lingkungan lokal ini sehingga build image
      sungguhan tidak bisa dijalankan langsung — sebagai gantinya dikonfirmasi dua bukti terkuat
      yang tersedia: (1) `CGO_ENABLED=0 go build ./cmd/server` **berhasil bersih** (kondisi persis
      yang akan dialami di image Alpine), dan (2) tag `golang:1.26-alpine` yang dipakai di
      `Dockerfile` dikonfirmasi **benar-benar ada** di Docker Hub (dicek via registry API).
- [x] **(putaran 2)** Cetak Kwitansi/Tagihan untuk tenant yang **belum punya logo** → PDF tetap
      ter-generate (tanpa logo), **bukan** error 404 dari `DownloadLogo` yang bocor ke user.
      Diverifikasi: seluruh pengujian awal sesi ini dilakukan sebelum tenant test punya logo,
      semua PDF berhasil dengan header teks-saja.
- [x] **(putaran 2)** Logo tenant benar-benar **tampil** di PDF (bukan kosong diam-diam) — bukti
      bahwa kontrak mengembalikan bytes, bukan path yang tidak bisa dibuka `projects`. Diverifikasi:
      login sebagai platform superadmin, upload logo PNG test ke tenant, generate ulang Kwitansi →
      PDF bertambah ukurannya dan mengandung marker `/Subtype /Image` (dicek langsung byte PDF-nya).
- [x] **(putaran 2)** Submit Tagihan dengan `amount = 0` / negatif → pesan error backend yang asli
      tampil di modal (bukan pesan generik). Backend dikonfirmasi mengembalikan 422 dengan pesan
      jelas ("Jumlah harus lebih dari 0"); logic unwrap `Blob` di frontend (`generateClientInvoice`)
      dikonfirmasi lewat pembacaan kode + type-check, mengikuti pola yang sama seperti axios error
      handling lain di codebase ini.
- [x] **(putaran 2)** `GET /projects/{id}` sebagai **client** mengembalikan `paymentStatus` &
      `nextDueDate` yang sama seperti sebagai staff (satu DTO, tidak ada field yang hilang di Client
      Portal). Diverifikasi: `GET /projects/me` sebagai client menunjukkan kedua field lengkap.
- [x] **(putaran 2)** `POST /projects/{id}/client-payments/invoice` **tidak** salah ter-route ke
      handler lain, dan `GET .../client-payments/{id}/receipt` tidak tertangkap case `evidence`/
      `client-payments` yang sudah ada — uji juga `paymentId` yang bukan angka → 400/404 rapi.
      Semua dikonfirmasi lewat curl di sesi ini tanpa satu pun salah-route.
- [x] **(ditemukan & diperbaiki saat verifikasi ulang)** `GET /projects` (list) sempat mengirim
      `paymentStatus: ""` (string kosong, bukan enum valid) di setiap baris karena field belum
      `omitempty` — bug nyata yang lolos dari verifikasi putaran pertama karena endpoint list belum
      sempat diuji. Diperbaiki di `dto.go` (`paymentStatus,omitempty`; `nextDueDate` sengaja
      **tidak** diberi `omitempty` agar tidak konflik dengan makna "null" yang sah di respons
      single-project). Dikonfirmasi ulang: list endpoint sekarang tanpa `paymentStatus` sama
      sekali, sementara `GET /projects/{id}` tetap selalu menyertakan keduanya.
- [x] **(verifikasi ulang)** `duplicateProject` dari project yang sudah `Paid` → salinannya
      benar-benar `Unpaid`/`nextDueDate: null`, tidak mewarisi status project sumber (sesuai
      ADR-0014, payments tidak ikut ter-duplikasi).

### Checklist UX (§3.6)

- [x] Badge status tampil **di dalam kartu "Sisa Tagihan"**, grid ringkasan tetap 3 kolom rapi di
      desktop maupun mobile — WO Console dan Client Portal. Dikonfirmasi via screenshot di kedua
      surface, termasuk pada viewport mobile 390px.
- [x] Label badge berbahasa Indonesia (Lunas / Dibayar Sebagian / Belum Dibayar / Jatuh Tempo),
      bukan nilai API mentah. Dikonfirmasi via screenshot — "LUNAS" dan "DIBAYAR SEBAGIAN" tampil.
- [x] Modal "Buat Tagihan" memuat kalimat penegas bahwa ini **tidak** mencatat pembayaran; setelah
      submit, tabel pembayaran memang **tidak** bertambah baris. Dikonfirmasi via screenshot: teks
      persis sesuai rencana tampil di deskripsi modal; dan secara kode, `GenerateInvoice` tidak
      pernah memanggil `ClientPaymentRepository.Create` sama sekali (hanya membaca).
- [x] Tombol "Cetak Kwitansi" ter-disable selama PDF sedang di-generate. Dikonfirmasi via kode
      (`generatingReceiptId`/`disabled` wiring) dan uji klik tunggal berhasil trigger request yang
      benar; race klik-ganda tidak diuji literal dengan klik berkecepatan tinggi di browser, tapi
      mekanismenya (state `generatingId` men-disable tombol baris itu selama fetch) identik dengan
      pola yang sudah ada di komponen lain di codebase ini.
- [x] Tombol "Cetak Kwitansi" muncul di **setiap** kartu Client Portal, **termasuk** pembayaran yang
      belum punya bukti transfer. Dikonfirmasi via screenshot: kartu DP & Pelunasan berlabel
      "BUKTI BELUM ADA" tetap punya tombol Cetak Kwitansi.
- [x] Tombol "Cetak Kwitansi" ada di versi tabel (desktop) **dan** `CardList` (mobile) WO Console.
      Dikonfirmasi via Playwright DOM query: 4 elemen ditemukan (2 tersembunyi di mobile-view,
      2 terlihat di desktop-view pada viewport 1440px) — kedua layout benar-benar merender tombolnya.
- [x] Tone tombol "Cetak Kwitansi" (`navy`) terbedakan jelas dari "Lihat Bukti" (`info`) di baris
      yang sama. Dikonfirmasi via inspeksi class CSS langsung (`bg-navy-900/10 text-navy-900` vs
      `bg-info-soft text-info`) dan visual di screenshot.
- [x] `CardHeader` dengan 2 tombol ("Tambah Pembayaran" + "Buat Tagihan") tidak pecah layout-nya di
      layar sempit. Dikonfirmasi via screenshot viewport 390px (mobile) — kedua tombol tampil rapi
      bersebelahan tanpa overflow/terpotong.

## 6. Ringkasan Dampak

**Perubahan skema**: 1 kolom nullable (`projects.next_due_date`). **Tidak ada tabel baru.**
**Dependency baru**: 1 library PDF Go (`maroto` v2, pure-Go). **Modul baru**: tidak ada — semuanya
perluasan `projects` (dan 1 package `contracts` baru di `platform` untuk menjembatani branding).
**Perubahan signature internal**: 1 — `NewClientPaymentService` dapat `ProjectRepository`
(`projects.module.go:44`), tetap di dalam satu modul. **Endpoint baru**: 2 (1 GET, 1 POST), keduanya
lewat dispatcher `Handler.Item` yang sudah ada sehingga otomatis mewarisi seluruh guard tenant/PIC/
client yang berlaku hari ini. Ini konsisten dengan tujuan awal: sederhana, transparan, tidak
menambah kompleksitas finance yang belum dibutuhkan.

**Status dokumen**: **✅ Selesai diimplementasikan dan diverifikasi nyata** (dua putaran verifikasi
penuh — lihat §5). Semua blocker dan pertanyaan terbuka sudah terjawab (§2.2 Opsi A, §2.7 kontrak
bytes, §2.8 maroto v2, §2.4 judul bercabang untuk `Refund`, badge hanya di tab Pembayaran) dan semua
kode yang mengimplementasikannya sudah ditulis, di-build bersih, dan diuji lewat API + DB + browser
sungguhan — bukan cuma dibaca ulang. Satu bug nyata ditemukan di putaran verifikasi kedua
(`paymentStatus` bocor sebagai string kosong di endpoint list) dan sudah diperbaiki serta
dikonfirmasi ulang. Satu-satunya langkah di §4 yang **belum** dijalankan adalah commit/push/deploy
(poin 12) — sengaja ditahan menunggu instruksi eksplisit, sesuai permintaan user di sesi ini.
