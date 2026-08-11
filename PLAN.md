# PLAN — Dokumen Umum Project (tanpa keterkaitan spesifik) + tab Dokumen di Client Portal

## 0. Goal

Dua bagian, satu fitur:

1. **WO Console**: tab "Dokumen" (evidence) saat ini mewajibkan setiap file dikaitkan ke salah satu
   dari 6 konteks spesifik (Timeline Vendor, Pembayaran, Kerja Sama Vendor, Kendala, Pembayaran
   Client, Pembayaran Venue) — tidak bisa dipakai untuk dokumen operasional project yang tidak
   terkait entitas manapun (rundown, buku acara, teks juru bicara, banquet order, rekap order,
   rekap dekor, dll). Tambahkan konteks ke-7: **"Umum"** — tanpa `relatedId` wajib.
2. **Client Portal**: tab baru **"Dokumen"** yang menampilkan dokumen "Umum" — **hanya yang secara
   eksplisit ditandai staff sebagai boleh dilihat client** (toggle per dokumen, default
   disembunyikan — dikonfirmasi user, lihat §1).

## 1. Current state, confirmed precisely

### `evidence` — polymorphic, wajib berelasi, tidak bisa diedit setelah upload
`evidence.related_kind` (`ENUM(...) NOT NULL`) dan `evidence.related_id` (`BIGINT UNSIGNED NOT
NULL`) — keduanya wajib di level database
(`apps/api/migrations/000008_create_project_tables.up.sql:121-122`), bukan cuma validasi UI.
`related_kind` sudah dua kali diperluas dengan pola `ALTER TABLE evidence MODIFY COLUMN
related_kind ENUM(...)` (migration `000027` untuk `clientPayment`, `000030` untuk `venuePayment`)
— preseden langsung untuk menambah nilai baru.

`evidence_endpoints.go` cuma punya 3 handler: `listEvidence`, `uploadEvidence`, `downloadEvidence`
— **tidak ada edit/delete/toggle apapun untuk evidence hari ini**. `uploadEvidence` hanya digerbang
`requireStaff` (peran manapun, termasuk Wedding Planner, boleh upload — tidak ada
`requireManagerRole`).

### Evidence saat ini TIDAK punya kontrol visibilitas client sama sekali — temuan kritis
Diverifikasi langsung: `listEvidence`/`downloadEvidence`
(`apps/api/internal/modules/projects/presentation/evidence_endpoints.go:13,56`) dipanggil **tanpa
parameter `claims` sama sekali** — `EvidenceService.List`/`Download`
(`application/evidence_service.go`) tidak pernah membedakan `PrincipalType`. Satu-satunya gerbang
untuk client adalah `resolveProjectAccess` di level project (client cuma boleh baca project
miliknya sendiri, GET-only) — begitu lolos itu, **evidence apapun yang menempel ke apapun di dalam
project itu otomatis terlihat & bisa diunduh client, tanpa kecuali**. Ini beda dari Venue, yang
punya gerbang MIME-type eksplisit untuk client (ADR-0016).

Konsekuensinya: untuk konteks Umum yang baru, **tidak bisa** mengikuti pola "otomatis terlihat"
yang sudah ada pada 6 konteks lain — sudah dikonfirmasi ke user (lihat §2) bahwa dokumen operasional
macam rundown vs rekap dekor punya sensitivitas berbeda, jadi konteks Umum butuh kontrol
visibilitasnya sendiri, terpisah dari (dan tidak mengubah) perilaku 6 konteks lama.

### Client Portal belum punya tab "Dokumen" berdiri sendiri
Client Portal hari ini hanya melihat evidence *dalam konteks* — tombol "Lihat Evidence" di dalam
kartu Timeline Vendor / Kendala (`VendorTabPage.tsx`, `KendalaTabPage.tsx`, `IssueCard.tsx`), lewat
`EvidenceViewerModal.tsx` yang sudah dipakai bersama WO Console — tidak ada daftar dokumen mandiri.

## 2. Design decisions

### Konteks baru: `related_kind = "general"`, `related_id = 0` (sentinel, bukan kolom nullable)
Menambah nilai enum ketujuh, pola sama persis dengan `000027`/`000030`. `related_id` **tidak**
diubah jadi nullable — pakai `0` sebagai sentinel "tidak ada target", mengikuti preseden yang sudah
ada di codebase ini (`venueId: 0` = "lepas venue", `AUTO_INCREMENT` tidak pernah mulai dari 0).
Nama `"general"` dipilih ketimbang `"project"` supaya tidak mirip secara string dengan
`"projectVendor"` yang sudah ada (risiko salah baca sekilas).

### Kontrol visibilitas client: kolom baru `is_client_visible`, hanya berlaku untuk konteks Umum
`evidence.is_client_visible BOOLEAN NOT NULL DEFAULT FALSE` — kolom ini ditambahkan ke seluruh
tabel secara struktural, tapi **hanya benar-benar berarti/ditegakkan untuk `related_kind =
'general'`**. Untuk 6 konteks lama, kolom ini tidak pernah dibaca/ditegakkan — visibilitasnya tetap
persis seperti hari ini (otomatis terlihat begitu terpasang ke sesuatu di dalam project client),
**tidak ada perubahan perilaku maupun regresi untuk konteks yang sudah ada**.

Konfirmasi eksplisit dari user: toggle per dokumen saat upload, **default OFF** (aman-secara-default
— dokumen internal seperti rekap dekor tidak akan ter-expose kecuali staff sengaja mencentang).

### Bisa diubah setelah upload — satu endpoint toggle baru, bukan edit umum
Evidence hari ini tidak bisa diedit sama sekali setelah diunggah. Supaya staff tidak perlu
re-upload seluruh file cuma untuk memperbaiki kesalahan centang visibilitas, tambahkan **satu**
endpoint bertarget sempit — `POST /projects/{id}/evidence/{evidenceId}/toggle-client-visible` —
mengikuti pola penamaan `toggle-active`/`toggle-suspension` yang sudah dipakai konsisten di seluruh
codebase ini (Vendor/Venue/Staff/Tenant/Plan/PlatformAdmin), bukan pola PATCH-dengan-body yang baru.
Ditolak (422) kalau dipanggil pada evidence yang `related_kind`-nya bukan `general` — tidak ada
alasan untuk menyalakan/mematikan visibilitas pada evidence yang memang selalu terlihat client.

### Endpoint baca baru untuk Client Portal — bukan menyaring endpoint lama
`GET /projects/{id}/documents` (baru, terpisah dari `GET /projects/{id}/evidence` yang sudah ada) —
selalu memfilter `related_kind = 'general' AND is_client_visible = true`, **tanpa syarat, tidak
tergantung siapa pemanggilnya**. Dipilih ketimbang menambahkan percabangan per-`PrincipalType` di
dalam `listEvidence` yang sudah ada, supaya endpoint ini gampang diaudit ("endpoint ini SELALU cuma
mengembalikan dokumen umum yang aman-untuk-client, titik") — lebih aman daripada logika bersyarat
yang bisa salah ketik/lupa saat diedit nanti. WO Console sendiri tetap pakai
`GET /projects/{id}/evidence` yang sudah ada (semua 7 konteks tetap kelihatan, ditambah badge
visibilitas — lihat §3), tidak berubah.

### Field "Jenis Evidence" tidak disentuh
Pilihan yang sudah ada (`Document`, `Other`, dst) sudah cukup menaungi semua contoh dokumen yang
diminta — yang hilang murni soal konteks/visibilitas, bukan soal taksonomi jenis file.

## 3. Perubahan konkret

### Backend
- **Migration baru**: `ALTER TABLE evidence MODIFY COLUMN related_kind ENUM('vendorMilestone',
  'payment','projectVendor','issue','clientPayment','venuePayment','general') NOT NULL;` +
  `ALTER TABLE evidence ADD COLUMN is_client_visible BOOLEAN NOT NULL DEFAULT FALSE;`
- `domain/evidence.go`: `RelatedGeneral EvidenceRelatedKind = "general"`; `Evidence.IsClientVisible
  bool`.
- `application/evidence_service.go`:
  - `Upload` menerima `IsClientVisible bool` di input; dinormalisasi paksa ke `false` di service
    layer kalau `RelatedKind != RelatedGeneral` (defensif — tidak percaya body request begitu saja
    untuk konteks yang memang tidak seharusnya bisa diatur visibilitasnya).
  - `ListClientDocuments(ctx, projectID) ([]Evidence, error)` — query baru:
    `WHERE project_id = ? AND related_kind = 'general' AND is_client_visible = true`.
  - `ToggleClientVisible(ctx, projectID, evidenceID int64) (*Evidence, error)` — baca row, tolak
    (422) kalau `RelatedKind != RelatedGeneral`, flip `IsClientVisible`, simpan.
- `presentation/evidence_endpoints.go`:
  - `evidenceUploadBody` tambah `IsClientVisible bool`.
  - `GET /projects/{id}/documents` (baru) — handler tipis, panggil `ListClientDocuments`, response
    shape sama dengan `evidenceResponse` yang sudah ada (reuse `toEvidenceResponse`).
  - `POST /projects/{id}/evidence/{evidenceId}/toggle-client-visible` (baru) — `requireStaff` sama
    seperti upload, tidak ada gerbang role tambahan (konsisten dengan siapa saja yang boleh upload).
- Routing (`handler.go`'s dispatcher): daftarkan 2 endpoint baru di atas.

### Frontend — WO Console
- `evidence.schema.ts`: `EVIDENCE_RELATED_KIND_OPTIONS` tambah `"general"`; `relatedId` jadi
  opsional khusus saat `relatedKind === "general"` (via `.superRefine`, bukan `.min(1)` flat).
- `ProjectEvidenceSection.tsx`:
  - `RELATED_KIND_LABEL` tambah `general: "Umum (Tidak Terkait Spesifik)"`.
  - Field "Terkait" **disembunyikan sepenuhnya** dari form saat konteks = Umum (bukan tampil kosong)
    — `relatedOptionsFor("general")` dicek di render, bukan cuma dikembalikan array kosong.
  - Checkbox baru **"Tampilkan ke Client"** — hanya muncul saat konteks = Umum, default tidak
    dicentang.
  - Kolom "Konteks" di tabel/list dapat badge kecil untuk baris Umum: "Terlihat Client" (hijau) /
    "Internal Saja" (abu-abu) — beserta tombol toggle sebaris (ikon mata) yang memanggil
    `toggle-client-visible` tanpa perlu buka modal.
- `useProjectStore.ts`: tambah `toggleEvidenceClientVisible(projectId, evidenceId)`, `uploadEvidence`
  diperluas menerima `isClientVisible`.

### Frontend — Client Portal (bagian baru)
- `apps/web/src/modules/client-portal/pages/tabs/DokumenTabPage.tsx` (baru) — fetch
  `GET /projects/{id}/documents`, render daftar sederhana (Nama, Jenis, Tanggal Dokumen, tombol
  "Lihat"), reuse `EvidenceViewerModal.tsx` (sudah dipakai lintas WO Console & Client Portal, tidak
  perlu diubah). Empty state: "Belum ada dokumen yang dibagikan untuk project ini."
- `ClientPortalLayout.tsx`: tab baru **"Dokumen"** ditambahkan di akhir urutan tab
  (Vendor, Timeline, Pembayaran, Kendala, **Dokumen**) — ditambahkan, bukan menggantikan, supaya
  urutan 4 tab yang sudah ada tidak berubah sama sekali.
- `route-paths.ts`: `ClientPortalTab` tambah `"dokumen"`.
- `client-portal.routes.tsx`: route baru `{ path: "dokumen", element: <DokumenTabPage /> }`.
- `apps/web/src/shared/services/api-endpoints.ts`: endpoint baru `projects.documents(id)`.

## 4. Order of implementation

1. Migration (enum + kolom baru) → `go build`/`go vet`.
2. Backend: `domain/evidence.go` → `evidence_service.go` (Upload diperluas + 2 method baru) →
   `evidence_endpoints.go` (2 route baru) → `go build`/`go vet` setiap langkah.
3. WO Console frontend: schema → form (checkbox + sembunyikan field Terkait) → badge + tombol
   toggle di list → `tsc --noEmit`.
4. Client Portal frontend: `DokumenTabPage.tsx` → wiring layout/routes/api-endpoints →
   `tsc --noEmit`.
5. Verifikasi menyeluruh (lihat §5).

## 5. Verification

- Upload evidence konteks Umum tanpa isi field Terkait (karena memang disembunyikan) — berhasil,
  `related_id` tersimpan `0`.
- Upload konteks Umum dengan "Tampilkan ke Client" **tidak** dicentang → **tidak** muncul di
  `GET /projects/{id}/documents` maupun tab Dokumen Client Portal; **tetap** muncul di tab Dokumen
  WO Console dengan badge "Internal Saja".
- Upload konteks Umum dengan checkbox dicentang → langsung muncul di kedua sisi.
- Toggle visibilitas dari "Internal Saja" ke "Terlihat Client" pada dokumen yang sudah ada (tanpa
  re-upload) → langsung muncul di tab Dokumen Client Portal begitu di-refresh.
- `POST .../toggle-client-visible` pada evidence berkonteks selain `general` → ditolak 422.
- Regresi: 6 konteks lama (Timeline Vendor, Pembayaran, Kerja Sama Vendor, Kendala, Pembayaran
  Client, Pembayaran Venue) — perilaku upload, list, dan visibilitas client-nya **sama persis**
  seperti sebelum perubahan ini (client masih otomatis lihat evidence yang menempel di sana, tidak
  terpengaruh kolom `is_client_visible` sama sekali).
- Client Portal: 4 tab lama tetap di urutan yang sama, tab "Dokumen" baru muncul di akhir tanpa
  mengubah default-landing tab (`vendor`).
