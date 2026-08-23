# PLAN — Timeline Default: dukung H+ (Setelah Acara), bukan cuma H-

> ## ⚠️ Hasil Verifikasi Senior System Analyst (sebelum implementasi)
>
> Kedua rancangan di file ini (Timeline H+/H- di bawah, dan "Lihat Bukti" di
> bagian akhir file) sudah ditelusuri ulang sampai ke titik konsumsi
> terjauh masing-masing tipe/data. Hasilnya:
>
> - **Irisan antar 2 rancangan baru ini: nihil.** Timeline H+/H- (modul
>   `milestone-templates`) dan Lihat Bukti (modul `projects`/pembayaran)
>   tidak berbagi file, state, maupun endpoint apapun.
> - **Irisan dengan fitur Edit/Delete Payment yang sudah live**: positif,
>   tidak konflik — "Lihat Bukti" murni menambah kemampuan (lihat file
>   sebelum memutuskan Delete), tidak mengubah `evidence`/`updatePayment`/
>   `deletePayment` yang sudah ada.
> - 🐛 **Ditemukan 1 bug nyata di rancangan Timeline H+/H-** (bukan cuma
>   risiko teoritis) — sudah diperbaiki di §3.2 di bawah, lihat kotak
>   "KOREKSI" di sana. Ringkasnya: rancangan awal menyuruh mengganti field
>   `daysBeforeEvent` jadi `daysOffset`+`daysDirection` di tipe
>   `MilestoneTemplateFormValues` yang **sama** dipakai `useMilestoneTemplateStore.ts`
>   untuk membangun body request. Kalau tidak diberi nama tipe baru yang
>   terpisah, store akan diam-diam mengirim `{name, daysOffset, daysDirection}`
>   ke backend yang mengharapkan `{name, daysBeforeEvent}` — Go akan
>   mem-decode `daysBeforeEvent` yang tidak ada sebagai `0`, artinya
>   **setiap template yang disimpan akan diam-diam jadi "Hari-H" (0)**
>   berapa pun angka & toggle yang dipilih user, tanpa error apapun yang
>   kelihatan (compile TypeScript pun tidak akan menangkap ini kalau nama
>   tipenya dipakai ulang untuk 2 bentuk berbeda). Sudah diperbaiki dengan
>   memperkenalkan 1 tipe baru (`MilestoneTemplateSubmitValues`) yang
>   terpisah dari tipe form internal, dan mengoreksi klaim "tidak ada
>   perubahan di `useMilestoneTemplateStore.ts`" jadi "ada, cuma perubahan
>   nama tipe, bukan logic".
> - **Rancangan "Lihat Bukti" (nested modal)**: diverifikasi langsung ke
>   implementasi `Modal.tsx` — tidak ada `transform`/`filter`/`opacity` di
>   ancestor manapun dari children-nya, jadi modal ber-`position: fixed`
>   yang di-render sebagai sibling di dalamnya (bukan lewat prop `children`)
>   akan tetap benar secara z-index/stacking-context, **tidak ada bug
>   clipping atau salah tumpuk** — sudah dikonfirmasi aman, bukan asumsi.
>   Satu syarat implementasi yang dipertegas: modal harus di-mount secara
>   kondisional (`{state && <Komponen />}`, pola yang sudah dipakai di file
>   yang sama untuk modal Edit/Delete), bukan selalu di-render dengan cuma
>   toggle prop `open` — kalau tidak, state "evidence yang sedang dilihat"
>   bisa nyangkut dari payment sebelumnya. Detail lengkap di bagian "Lihat
>   Bukti" di akhir file.

## 0. Tujuan

Item **Timeline Default** (`Pengaturan → Timeline Default`, disalin otomatis ke
tab Timeline setiap project baru) hari ini hanya bisa dijadwalkan **H-**
(sejumlah hari **sebelum** hari-H). Tapi ada item yang pada praktiknya SELALU
terjadi **setelah** hari-H — contoh dari user: "Pengembalian Kotak Seserahan &
Hiasannya". Tujuan plan ini: memungkinkan staff memilih **Sebelum Acara**
atau **Setelah Acara** saat membuat/mengubah item Timeline Default, tanpa
memaksa mereka mengetik angka negatif di bawah label yang bertuliskan "H-".

## 1. Kondisi saat ini (sudah dikonfirmasi lewat audit kode)

- Kolom DB `project_milestone_templates.days_before_event` bertipe **`INT`
  (signed)**, bukan `UNSIGNED` — `apps/api/migrations/000024_create_project_milestone_templates.up.sql:11`.
  Jadi secara skema **sudah bisa** menyimpan nilai negatif — **tidak perlu
  migration baru sama sekali**.
- Rumus penerjemah offset → tanggal absolut,
  `apps/api/internal/modules/projects/application/default_milestones.go:25`:
  ```go
  target := eventDate.AddDate(0, 0, -tmpl.DaysBeforeEvent)
  ```
  Kalau `DaysBeforeEvent` negatif, hasilnya otomatis `eventDate + N hari` —
  **H+ sudah didukung secara matematis oleh rumus yang ada**, tanpa perlu
  diubah sama sekali.
- **Satu-satunya penghalang** adalah validasi eksplisit yang menolak nilai
  negatif, di 2 tempat:
  - Backend, `apps/api/internal/modules/projects/application/milestone_template_service.go:49-51`
    (`Create`) dan `:64-66` (`Update`): `if input.DaysBeforeEvent < 0 { return apperror.Validation(...) }`.
  - Frontend, `apps/web/src/modules/milestone-templates/schemas/milestone-template.schema.ts:5`:
    `daysBeforeEvent: z.number().int(...).min(0, "Tidak boleh negatif")`.
- Clamp `target.Before(prepStartDate)` di `default_milestones.go:26-28`
  (pengaman untuk project mendadak/short-notice, supaya tidak ada milestone
  yang target-nya jatuh sebelum tanggal booking) **tidak akan pernah
  tersentuh oleh item H+**: tanggal H+ selalu `eventDate + N hari` yang pasti
  setelah `eventDate`, dan `eventDate` sendiri selalu ≥ `prepStartDate` —
  tidak ada konflik apapun dengan mekanisme clamp yang sudah ada.
- UI form: `apps/web/src/modules/milestone-templates/components/MilestoneTemplateFormModal.tsx:80-87`
  — satu field, label "Hari Sebelum Acara (H-)", `<input type="number" min={0}>`.
- Tampilan list: `apps/web/src/modules/milestone-templates/pages/MilestoneTemplateListPage.tsx:160,178`
  — `daysBeforeEvent === 0 ? "Hari-H" : \`H-${daysBeforeEvent}\`` (card & tabel).
- **Scope-nya murni di Timeline Default Template.** Tab Timeline milik satu
  project yang sedang berjalan (`ProjectMilestonesSection.tsx`) memakai date
  picker tanggal absolut langsung — tidak memakai konsep hari-offset sama
  sekali, jadi **tidak ada irisan** ke sana. Begitu template disalin jadi
  `project_milestones` (`domain.ProjectMilestone.TargetDate`, tanggal
  absolut), offset-nya tidak pernah dibaca ulang — jadi tidak ada data lama
  yang perlu di-migrasi/disentuh.
- 6 item bawaan tenant baru (`defaultMilestoneTemplateSeed`,
  `milestone_template_service.go:120-130`) semuanya H- secara alami (Survei
  Venue s/d Hari-H) — tidak ada yang perlu diubah jadi H+.

## 2. Keputusan desain

1. **Bukan sekadar mengizinkan angka negatif diketik di field yang sama.**
   Field-nya secara visual tetap berlabel "H-" — kalau cuma dibuka untuk
   negatif, staff harus mengetik misalnya "-3" untuk maksud "3 hari setelah
   acara" (double-negative, rawan salah paham/salah ketik). Sebagai
   gantinya: tambah **toggle dua-opsi "Sebelum Acara" / "Setelah Acara"** di
   sebelah input angka. Angka yang diketik user **selalu positif** (jumlah
   hari); tanda (+/-) ditentukan oleh toggle, dikonversi ke
   `daysBeforeEvent` bertanda **hanya di level komponen form** — payload API
   yang dikirim ke backend tetap satu angka bertanda seperti sekarang,
   `store`/backend tidak perlu tahu soal toggle sama sekali.
2. **Backend**: cukup **hapus** 2 blok validasi `< 0` di
   `milestone_template_service.go` (`Create` & `Update`). Tidak diganti
   validasi baru apapun, karena rumus `AddDate` sudah otomatis benar untuk
   kedua arah. Tidak ada perubahan skema, tidak ada migration.
3. **Tampilan list**: formatter jadi 3-arah — `0` → `"Hari-H"`, positif →
   `"H-{d}"`, negatif → `"H+{Math.abs(d)}"` — di kedua tempat render
   (`MilestoneTemplateListPage.tsx`, card & tabel).
4. **Tidak menyentuh clamp `prepStartDate`** di `default_milestones.go` —
   sudah otomatis benar untuk H+ (lihat §1), sengaja dibiarkan supaya blast
   radius seminimal mungkin.
5. **Tidak mengubah 6 item seed bawaan** — H+ murni opsi baru yang staff
   pilih sendiri saat menambah item custom seperti "Pengembalian Kotak
   Seserahan", bukan migrasi data existing.
6. **Hari-H (`0`) diperlakukan sebagai kasus khusus**, sama untuk kedua
   arah toggle — `daysOffset=0` selalu berarti "Hari-H" apapun posisi
   togglenya, sama seperti hari ini `H-0` sudah ditampilkan sebagai
   "Hari-H", bukan "H-0".

## 3. Perubahan konkret

### 3.1 Backend

- `apps/api/internal/modules/projects/application/milestone_template_service.go`:
  hapus blok berikut di `Create` (baris 49-51) **dan** `Update` (baris 64-66):
  ```go
  if input.DaysBeforeEvent < 0 {
      return nil, apperror.Validation("Hari sebelum acara tidak boleh negatif", map[string][]string{"daysBeforeEvent": {"Tidak boleh negatif"}})
  }
  ```
- Tidak ada perubahan di `default_milestones.go`, `domain/milestone_template.go`,
  `mysql_milestone_template_repository.go`, `milestone_template_handler.go`,
  atau migration manapun — semuanya sudah benar/kompatibel apa adanya.

### 3.2 Frontend

> **🐛 KOREKSI (temuan verifikasi)**: draf awal bagian ini menyuruh
> mengganti isi `MilestoneTemplateFormValues` (tipe hasil `z.infer` dari
> `milestoneTemplateSchema`) dari `{name, daysBeforeEvent}` jadi
> `{name, daysOffset, daysDirection}`. Masalahnya, **nama tipe yang sama
> itu juga dipakai sebagai tipe parameter** `createTemplate`/`updateTemplate`
> di `useMilestoneTemplateStore.ts:21-22`, yang isinya cuma
> `httpClient.post(url, values)` — mengirim `values` mentah-mentah sebagai
> body. Kalau bentuk tipe-nya diganti tapi nama tipenya dipakai ulang tanpa
> disadari, store akan mengirim `{name, daysOffset, daysDirection}` ke
> endpoint yang mengharapkan `{name, daysBeforeEvent}` — field yang hilang
> di-decode Go sebagai `0`, jadi **setiap simpan/ubah template akan diam-diam
> tersimpan sebagai Hari-H (0)**, tanpa error apapun yang terlihat. Perbaikan:
> pisahkan jadi 2 tipe — satu untuk state internal form (dengan toggle), satu
> untuk apa yang benar-benar dikirim ke `onSubmit`/store/API (tetap
> `{name, daysBeforeEvent}` seperti sekarang). Versi di bawah ini sudah
> dikoreksi.

- `apps/web/src/modules/milestone-templates/schemas/milestone-template.schema.ts`:
  - `milestoneTemplateSchema`/`MilestoneTemplateFormValues` (dipakai HANYA
    sebagai state internal `MilestoneTemplateFormModal.tsx`, tidak lagi
    dikirim langsung ke `onSubmit`) berubah jadi:
    `{ name: string; daysOffset: number (min 0); daysDirection: "before" | "after" }`.
  - Tambah tipe baru `MilestoneTemplateSubmitValues = { name: string; daysBeforeEvent: number }`
    — inilah yang mengalir ke `onSubmit` → `handleSubmit` di list page →
    `createTemplate`/`updateTemplate` di store, **sama persis seperti bentuk
    yang sudah ada hari ini**, tidak berubah.
  - Tambah 2 fungsi konversi murni (tanpa I/O):
    ```ts
    export function toDaysBeforeEvent(daysOffset: number, direction: "before" | "after"): number {
      return direction === "after" ? -daysOffset : daysOffset;
    }
    export function fromDaysBeforeEvent(daysBeforeEvent: number): { daysOffset: number; daysDirection: "before" | "after" } {
      return { daysOffset: Math.abs(daysBeforeEvent), daysDirection: daysBeforeEvent < 0 ? "after" : "before" };
    }
    ```
- `apps/web/src/modules/milestone-templates/components/MilestoneTemplateFormModal.tsx`:
  - State internal (`useState`) pakai `MilestoneTemplateFormValues` yang baru (dengan toggle).
  - `toFormValues` pakai `fromDaysBeforeEvent(template.daysBeforeEvent)` untuk prefill Edit.
  - Tambah `Select` 2-opsi di sebelah input angka: "Sebelum Acara" (`before`) / "Setelah Acara" (`after`).
  - **Ubah tipe prop `onSubmit`** dari `(values: MilestoneTemplateFormValues) => void` jadi
    `(values: MilestoneTemplateSubmitValues) => void` — ini perubahan public API komponen, jadi
    pemanggilnya (`MilestoneTemplateListPage.tsx`) ikut berubah.
  - `handleSubmit`: setelah validasi `daysOffset`+`daysDirection` lolos, konversi ke
    `{ name, daysBeforeEvent: toDaysBeforeEvent(daysOffset, daysDirection) }` **sebelum** memanggil `onSubmit`.
- `apps/web/src/modules/milestone-templates/pages/MilestoneTemplateListPage.tsx`:
  - **Ganti tipe parameter** `handleSubmit(values: MilestoneTemplateFormValues)` jadi
    `handleSubmit(values: MilestoneTemplateSubmitValues)` — menerima bentuk yang sudah dikonversi,
    diteruskan apa adanya ke `createTemplate`/`updateTemplate` (tidak ada logic baru di sini).
  - Ganti formatter di baris 160 & 178 jadi 3-arah (lihat §2.3):
    ```ts
    const daysLabel = t.daysBeforeEvent === 0 ? "Hari-H" : t.daysBeforeEvent > 0 ? `H-${t.daysBeforeEvent}` : `H+${Math.abs(t.daysBeforeEvent)}`;
    ```
- `apps/web/src/modules/milestone-templates/stores/useMilestoneTemplateStore.ts`:
  - **Ada perubahan** (koreksi dari klaim draf awal "tidak ada perubahan"):
    ganti import & tipe parameter `createTemplate`/`updateTemplate` dari
    `MilestoneTemplateFormValues` jadi `MilestoneTemplateSubmitValues`.
    Murni perubahan nama tipe — badan fungsi (`httpClient.post/patch(url, values)`)
    tidak berubah sama sekali, karena bentuk datanya (`{name, daysBeforeEvent}`)
    memang tidak berubah, cuma nama tipenya yang sekarang benar/tidak ambigu.

## 4. Urutan implementasi

1. Backend: hapus 2 blok validasi di `milestone_template_service.go`.
2. Frontend: 2 fungsi konversi + field baru di schema.
3. Frontend: toggle di `MilestoneTemplateFormModal.tsx`.
4. Frontend: formatter 3-arah di `MilestoneTemplateListPage.tsx`.
5. Build/vet/typecheck (Go + `tsc --noEmit` + `npm run build`).
6. Uji end-to-end terhadap data lokal nyata (lihat §5).
7. Commit/push/deploy — hanya setelah verifikasi §5 lolos dan diminta.

## 5. Checklist verifikasi

- [x] **Kritis (menutup bug yang ditemukan saat verifikasi)**: diverifikasi
      lewat curl langsung ke `POST /milestone-templates` dengan
      `{"name":"...","daysBeforeEvent":-3}` — tersimpan & dikembalikan apa
      adanya sebagai `-3`. Konversi frontend (`daysOffset`+`daysDirection` →
      `daysBeforeEvent`) sudah benar secara tipe (`tsc --noEmit` bersih,
      `MilestoneTemplateSubmitValues` terpisah dari `MilestoneTemplateFormValues`).
- [x] Tambah template baru via curl dengan `daysBeforeEvent: -3` → tersimpan
      sebagai `-3` di DB, dikonfirmasi lewat `GET` ulang.
- [x] Formatter 3-arah (`formatDaysLabel`) sudah benar secara logic: `0` →
      "Hari-H", positif → "H-{d}", negatif → "H+{|d|}" — diverifikasi lewat
      pembacaan kode & tsc, tidak lewat klik browser (tidak ada tool browser
      di environment ini).
- [x] Update template existing (id sama) dari `daysBeforeEvent=-3` →
      `-5` → `2` lewat `PATCH` — semua tersimpan benar, tidak ada blokade
      validasi tersisa untuk arah manapun.
- [x] **Buat project baru → milestone hasil seed dari template H+3 punya
      `targetDate = eventDate + 3 hari`** — diverifikasi end-to-end: project
      dibuat dengan `eventDate=2027-01-01`, milestone "Pengembalian Kotak
      Seserahan" (template H+3) ter-seed dengan `targetDate=2027-01-04`
      (persis `eventDate + 3`), bukan `2026-12-29` (`eventDate - 3`).
- [x] 6 item H- bawaan lama (90/60/14/7/1/0) tetap berperilaku identik —
      dikonfirmasi di seeding project yang sama: `targetDate` untuk
      "Survei Venue & Vendor" (H-90) = `2026-10-03`, cocok dengan
      `eventDate - 90 hari`, tidak berubah dari sebelumnya.
- [x] Project dengan `prepStartDate` sangat dekat ke `eventDate` (short-notice)
      → item H+ tetap terjadwal `eventDate + N`, **tidak** ikut ter-clamp ke
      `prepStartDate` — dikonfirmasi lewat pembacaan kode
      (`default_milestones.go`'s clamp tidak disentuh sama sekali oleh
      implementasi ini, dan secara konstruksi tanggal H+ selalu > `eventDate`
      ≥ `prepStartDate`, jadi `target.Before(prepStartDate)` tidak pernah true).
- [x] `go build ./... && go vet ./...`, `npx tsc --noEmit`, `npm run build`
      semua bersih (dijalankan ulang setelah implementasi selesai).

---

## Lampiran — perubahan role Delete pembayaran (sudah diimplementasikan)

Di luar scope Timeline di atas: atas permintaan eksplisit user, gate hard
delete pembayaran (vendor/client/venue — fitur dari plan sebelumnya, sudah
live di kode) diperluas dari **Owner-only** jadi **Owner-atau-Admin**,
mengikuti bar yang sama dengan aksi destruktif `vendors`/`venues`
(`requireManagerRole`, walau namanya menyesatkan, sebenarnya mengecek persis
`Owner`/`Admin`). Sudah diimplementasikan & diverifikasi lewat curl (Admin
sekarang bisa DELETE, Staff biasa masih 403):
- Backend: `apps/api/internal/modules/projects/presentation/util.go` —
  helper baru `isOwnerOrAdmin(role string) bool`, dipakai di
  `deletePayment`/`deleteClientPayment`/`deleteVenuePayment` (masing-masing
  di `payment_endpoints.go`/`client_payment_endpoints.go`/`venue_payment_endpoints.go`),
  menggantikan `claims.role != "Owner"`.
- Frontend: variabel `isOwner` di `ProjectPaymentsSection.tsx`/
  `ClientPaymentsSection.tsx`/`VenuePaymentsSection.tsx` diganti jadi
  `canDelete` dengan cek `role !== "Staff"` (idiom yang sama dengan
  `ProjectHeaderCard.tsx`'s `canDuplicate`/`canSeeMargin`).
- Edit tetap tanpa role-gate sama sekali (sudah dari awal begitu, jadi Admin
  sudah otomatis bisa Edit sejak awal — tidak ada yang perlu diubah di sisi
  Edit).

---

# PLAN — Tombol "Lihat Bukti" di Pembayaran (Vendor/Client/Venue)

## 0. Tujuan

Setiap baris pembayaran (vendor/client/venue) sudah menampilkan badge
**Lengkap/Belum Lengkap** (atau **Ada/Belum Ada** untuk client) yang
menandakan evidence (Invoice/Bukti Transfer) sudah/belum terlampir — tapi
**tidak ada cara untuk benar-benar melihat file-nya** dari tab Pembayaran
itu sendiri. Satu-satunya jalan hari ini adalah pindah ke tab Dokumen &
Evidence lalu mencari baris yang `Konteks`-nya cocok secara manual. Tujuan
plan ini: tambah **1 tombol "Lihat Bukti" per baris pembayaran** yang
langsung menampilkan bukti terlampirnya lewat modal, tanpa pindah tab.

## 1. Kondisi saat ini (sudah dikonfirmasi lewat audit kode)

- Evidence terhubung ke payment secara polimorfik: `evidence.relatedKind`
  = `"payment"`/`"clientPayment"`/`"venuePayment"`, `evidence.relatedId` =
  ID baris payment — sudah ada di state Zustand (`useProjectStore.evidence`),
  sudah di-fetch oleh ketiga section pembayaran (`fetchEvidence` dipanggil
  di `useEffect` masing-masing — ditambahkan di plan Edit/Delete
  sebelumnya, untuk keperluan dialog konfirmasi Delete).
- Vendor & venue payment bisa punya **0–2 evidence** (Invoice + Bukti
  Transfer, dua slot independen — lihat `createPayment`/`createVenuePayment`
  di `useProjectStore.ts`). Client payment cuma punya **0–1** (Bukti
  Transfer saja, tidak ada slot Invoice — lihat `client-payment.schema.ts`).
- Komponen viewer file **sudah ada dan generik**:
  `apps/web/src/shared/components/ui/EvidenceViewerModal.tsx` — menerima
  `{ open, onClose, projectId, evidence: {id, name, type, documentDate,
  fileName}, contextLabel? }`, sudah menangani fetch file terautentikasi
  (blob via `httpClient`), preview PDF/gambar, dan tombol Unduh. **Tidak
  perlu diubah sama sekali** — tinggal dipanggil dengan evidence yang tepat.
- Pola "baris evidence + tombol Lihat" yang identik **sudah ada** di
  `ProjectEvidenceSection.tsx` (`IconActionButton icon={Eye} label="Lihat
  Evidence"` → `setViewingEvidence(item)` → render `EvidenceViewerModal`).
  Plan ini murni mengulang pola yang sudah terbukti, bukan pola baru.
- **Perbedaan** dari pola di atas: di tab Evidence, setiap baris = 1
  evidence, jadi 1 tombol = 1 file langsung. Di tab Pembayaran, setiap
  baris = 1 payment yang **bisa punya sampai 2 evidence** — jadi "1 tombol"
  yang diminta user perlu menengahi kasus 0/1/2 file dalam satu modal yang
  sama, bukan langsung 1:1 ke `EvidenceViewerModal`.

## 2. Keputusan desain

1. **Komponen baru, generik & dapat dipakai ulang**:
   `apps/web/src/shared/components/ui/EvidenceListModal.tsx` — menerima
   `{ open, onClose, projectId, title, items: Evidence[] }`. Merender:
   - Kosong (`items.length === 0`) → `EmptyState` ("Belum ada bukti yang
     terlampir pada pembayaran ini.").
   - Berisi (1 atau lebih) → daftar baris (ikon jenis file, nama, badge
     `type`, tanggal), masing-masing punya tombol "Lihat" sendiri yang
     men-set `viewingEvidence` lokal lalu merender `EvidenceViewerModal`
     bertumpuk di atasnya (nested modal) — **menggunakan ulang
     `EvidenceViewerModal` apa adanya**, tidak ada logic viewer baru yang
     ditulis.
   - **✅ Sudah diverifikasi aman (bukan asumsi)**: `<Modal>` dan
     `<EvidenceViewerModal>` harus di-render sebagai **sibling** di dalam
     satu `<>...</>` yang dikembalikan `EvidenceListModal` — **bukan**
     dengan menaruh `<EvidenceViewerModal>` sebagai `children` di dalam
     `<Modal>...</Modal>`. Sudah dicek langsung ke `Modal.tsx`: container
     children-nya (`<div className="max-h-[65vh] overflow-y-auto ...">`)
     tidak punya `transform`/`filter`/`opacity` apapun, jadi tidak
     membentuk containing-block baru — elemen `position: fixed` di dalam
     maupun di luar itu akan tetap terhitung dalam stacking context global
     yang sama. Keduanya sama-sama `z-50`; karena urutan render
     `EvidenceViewerModal` ada SETELAH `Modal` di JSX, dia akan tetap
     tergambar di atas selama disusun sebagai sibling seperti di atas —
     tidak ada bug salah tumpuk/ketutup.
   - **Syarat implementasi lain yang wajib dipenuhi**: pemanggil
     (`ProjectPaymentsSection.tsx` dkk.) harus me-mount `EvidenceListModal`
     **secara kondisional** — `{viewingEvidencePayment && (<EvidenceListModal ... />)}`
     — bukan selalu me-render elemen-nya dan cuma men-toggle prop `open`.
     Kalau dirender selalu (instance komponen tidak pernah unmount), state
     internal `viewingEvidence` (evidence yang sedang dibuka di viewer)
     bisa **nyangkut dari payment sebelumnya**: tutup viewer tanpa
     menutup daftar dulu, tutup daftar, buka "Lihat Bukti" untuk payment
     LAIN — kalau instance-nya sama & belum pernah unmount, viewer bisa
     langsung nongol lagi menampilkan file dari payment yang salah. Pola
     mounting kondisional ini **sudah** jadi konvensi di file yang sama
     (`{editingPayment && <EditPaymentModal ... />}`,
     `{deletingPayment && <Modal ...>}`) — tinggal diikuti persis, bukan
     pola baru.
   - Domain-agnostic sesuai `.claude/rules/frontend-react.md` ("Shared UI
     components must be domain-agnostic") — namanya sengaja `EvidenceListModal`
     bukan `PaymentEvidenceModal`, supaya nanti bisa dipakai ulang untuk
     entitas lain yang punya evidence tapi belum ada tombol lihatnya (mis.
     Vendor Milestone, Kendala) — di luar scope permintaan sekarang, cuma
     dipakai untuk Pembayaran dulu.
2. **Kenapa tidak langsung buka `EvidenceViewerModal` untuk kasus 1
   evidence (skip daftar)?** Dipertimbangkan, tapi dilepas demi
   kesederhanaan: satu komponen (`EvidenceListModal`) dipakai identik di
   ketiga section tanpa percabangan "kalau cuma 1 file, lewati daftar" di
   3 tempat berbeda. Trade-off-nya: kasus paling umum (Lengkap, 2 evidence
   atau client dengan 1 evidence) butuh 1 klik ekstra (pilih dari daftar,
   lalu klik "Lihat") dibanding langsung ke viewer. Kalau UX ini dirasa
   kurang ringkas setelah dicoba, bisa disesuaikan belakangan — bukan
   keputusan yang mahal untuk diubah.
3. **Posisi tombol**: "Lihat Bukti" (ikon `Eye`, tone `info`, sama seperti
   `ProjectEvidenceSection.tsx`) ditaruh **paling kiri** di kolom Aksi,
   sebelum Edit/Delete — melihat adalah aksi paling aman & paling sering
   dipakai, dan tersedia untuk **semua role** (tidak digate sama sekali,
   selaras dengan tab Dokumen & Evidence yang juga tidak digate role).
   Tombol tetap muncul walau evidence-nya 0 (bukan disembunyikan) — klik
   tetap membuka modal, cuma menampilkan empty state, supaya user tidak
   bingung kenapa tombolnya hilang padahal badge bilang "Belum Lengkap"
   (bukan "tidak ada tombol").
4. **Tidak ada perubahan backend sama sekali** — evidence yang dibutuhkan
   sudah ada di state frontend (`evidence` array, sudah di-fetch), tidak
   ada endpoint baru, tidak ada perubahan skema.

## 3. Perubahan konkret

### Frontend

- **Baru**: `apps/web/src/shared/components/ui/EvidenceListModal.tsx` —
  komponen modal generik seperti dijelaskan di §2.1, meniru struktur JSX
  yang sama dengan baris evidence di `ProjectEvidenceSection.tsx` (ikon
  jenis file, nama, badge, tanggal, tombol `Eye`).
- `apps/web/src/modules/projects/components/detail/ProjectPaymentsSection.tsx`:
  - Tambah state `viewingEvidencePayment: VendorPayment | null`.
  - Tambah `IconActionButton icon={Eye} label="Lihat Bukti" tone="info"` di
    awal kolom Aksi (CardList & Table), `onClick={() =>
    setViewingEvidencePayment(payment)}`.
  - Render (kondisional, lihat §2.1):
    ```tsx
    {viewingEvidencePayment && (
      <EvidenceListModal
        open
        projectId={projectId}
        title={`Bukti — ${viewingEvidencePayment.type}`}
        items={evidence.filter(e => e.relatedKind === "payment" && e.relatedId === viewingEvidencePayment.id)}
        onClose={() => setViewingEvidencePayment(null)}
      />
    )}
    ```
- `ClientPaymentsSection.tsx` — pola identik, filter
  `e.relatedKind === "clientPayment"`.
- `VenuePaymentsSection.tsx` — pola identik, filter
  `e.relatedKind === "venuePayment"`.
- Import `Eye` dari `lucide-react` di ketiga file (kalau belum — cek dulu,
  karena `EvidenceViewerModal`/pola lain mungkin sudah mengimpornya).

## 4. Urutan implementasi

1. `EvidenceListModal.tsx` (komponen baru, sekali, dipakai 3x).
2. Wiring di `ProjectPaymentsSection.tsx`.
3. Wiring di `ClientPaymentsSection.tsx`.
4. Wiring di `VenuePaymentsSection.tsx`.
5. `npx tsc --noEmit` + `npm run build`.
6. Uji manual/terverifikasi: payment dengan 2 evidence, 1 evidence, 0
   evidence — pastikan modal menampilkan yang benar di ketiganya.
7. Commit/push/deploy — hanya setelah diminta.

## 5. Checklist verifikasi

> **Catatan verifikasi**: tidak ada tool browser di environment implementasi
> ini, jadi item-item berikut diverifikasi lewat pembacaan kode + `tsc`/
> `build` (bukan klik manual). Logic-nya sendiri sederhana & murni
> client-side (filter array yang sudah ada + 2 komponen modal yang sudah
> terbukti dipakai di tempat lain), jadi risiko residualnya rendah — tapi
> tetap disarankan sekali klik manual di browser sebelum dianggap 100% final.

- [x] Klik "Lihat Bukti" pada payment dengan 2 evidence (Invoice + Bukti
      Transfer) → modal menampilkan 2 baris, masing-masing bisa dibuka lewat
      `EvidenceViewerModal` dan menampilkan file yang benar (bukan tertukar) —
      dikonfirmasi lewat pembacaan kode: `items.map` me-render satu baris per
      elemen `Evidence`, `onClick` masing-masing meng-capture `item` yang
      benar lewat closure (bukan index bersama), tidak ada shared state antar
      baris yang bisa tertukar.
- [x] **Regresi state (menutup risiko yang ditemukan saat verifikasi)**:
      dikonfirmasi lewat pembacaan kode — `EvidenceListModal` di-mount
      kondisional (`{viewingEvidencePayment && <EvidenceListModal .../>}`) di
      ketiga section, sesuai syarat di §2.1. Instance benar-benar unmount
      setiap `viewingEvidencePayment` jadi `null`, jadi `viewingEvidence`
      internal-nya ikut reset otomatis — tidak mungkin nyangkut ke payment
      lain.
- [x] Modal `EvidenceViewerModal` (nested) tergambar **di atas**
      `EvidenceListModal` — dikonfirmasi keduanya di-render sebagai sibling
      di dalam satu `<>...</>` (bukan lewat prop `children`), sesuai
      implementasi `Modal.tsx` yang sudah diverifikasi di §2.1.
- [x] Klik pada client payment dengan 1 evidence → modal menampilkan 1 baris
      — sama seperti kasus 2 evidence, cuma `items.length === 1`.
- [x] Klik pada payment dengan 0 evidence → modal menampilkan empty state,
      tidak error/blank — `EvidenceListModal` punya cabang eksplisit
      `items.length === 0 ? <EmptyState .../> : ...`, tidak ada
      null/undefined access yang bisa crash.
- [x] Tombol "Lihat Bukti" tetap muncul untuk semua role (tidak digate),
      beda dengan tombol Delete yang Owner-atau-Admin saja — dikonfirmasi:
      tombol Eye di ketiga section tidak dibungkus kondisi `canDelete`
      apapun, beda dengan tombol Trash2 di sebelahnya.
- [x] Tidak ada request/endpoint baru yang terpanggil — murni memanfaatkan
      `evidence` yang sudah di-fetch di state — dikonfirmasi: `EvidenceListModal`
      tidak melakukan fetch apapun sendiri, cuma menerima `items` sebagai prop.
- [x] `npx tsc --noEmit` & `npm run build` bersih — sudah dijalankan ulang
      setelah seluruh implementasi (Timeline + Lihat Bukti) selesai, 0 error.
