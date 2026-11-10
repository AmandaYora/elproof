# PLAN.md — Revamp UI/UX ke shadcn/ui

> Status (10 Nov 2026): **Tahap 0–8 dikomit; Tahap 9 dikerjakan seluruhnya yang bisa dikerjakan mesin, belum dikomit; uji manusia belum dilakukan** (lihat catatan Tahap 9) (lihat "Catatan
> pelaksanaan" di akhir bagian Tahap). Rencana revisi 2 selebihnya belum dikerjakan. Revisi 1 (8 Nov 2026) hanya mengganti lapisan komponen.
> Revisi 2 mengikuti arahan pemilik produk (9 Nov 2026): **tujuannya fitur mudah dipahami, mudah dipakai, ideal, dan
> rupa modern; endpoint boleh diubah bila bentuk API membuat alur UI tidak ideal.**
> Keputusan: D1 = Base UI (8 Nov), D16 = perubahan API diizinkan (9 Nov). Keputusan lain (§9) masih usulan dan
> dikonfirmasi saat tahap terkait.
> Dasar: riset internet, inventaris kode, dan audit endpoint terhadap alur UI, dirangkum di `knowledge/DESIGN.md`
> (brief desain revisi 2, temuan → keputusan, sumber bertanda verifikasi). Susunan menu dan alur per peran:
> `docs/plan/registrasi-wo-self-service/FLOW-IDEAL.md`.

## 0. Ringkasan

Aplikasi sudah punya sistem desain buatan sendiri yang konsisten (token, dua kepadatan, 21 preset warna tenant,
±35 komponen), tetapi tiga hal membuatnya belum "mudah dan modern":

1. **Komponen:** perilaku aksesibilitas, overlay, dan combobox ditulis tangan dan hanya teruji sejauh yang pernah
   dicoba. Diganti ke shadcn/ui di atas Base UI, dari dalam, tanpa menulis ulang semua halaman sekaligus.
2. **Bentuk API memaksa UI bekerja keras:** satu layar memanggil 7–9 endpoint dan menghitung sendiri sisa tagihan,
   termin berikutnya, margin, dan antrean "perlu tindakan"; mencatat pembayaran dan membuat proyek butuh 2–5 panggilan
   berurutan yang bisa gagal setengah jalan; respons berisi ID tanpa nama. Akibatnya layar lambat di HP, data basi
   setelah menyimpan, dan pesan "tersimpan, tetapi … gagal". Revisi 2 menambah **14 perubahan endpoint** (§4), sebagian
   besar tambahan (tidak memutus klien lama).
3. **Rupa dan bahasa:** warna status gagal kontras, palet slate dingin dan sidebar blok warna merek terasa berat, istilah
   campur Inggris–Indonesia, belum ada pedoman tulisan dan bantuan kontekstual (DESIGN.md §4–§8).

Urutan kerja diubah agar pengguna merasakan perbaikan lebih awal: setelah fondasi dan komponen dasar, **Portal Klien
dikerjakan utuh sebagai irisan vertikal** (API + UI + `ProofStamp`), lalu diuji ke pengguna, baru meluas ke console.
Kemudahan diukur, bukan diasumsikan: uji tugas dengan pengguna nyata **sebelum** revamp dan diulang di tiga titik.

## 1. Kondisi sekarang (terukur 8–9 Nov 2026)

| Hal | Angka |
|---|---|
| Berkas `.tsx` / baris TS+TSX | 194 / ±33.500 |
| Halaman / modal+drawer / form dengan `safeParse` | 60 / 31 / 33 |
| Komponen bersama (`shared/components`) | ±35 |
| Pemakaian: Button 78 berkas, Input 56, Card 53, Badge 48, Modal 41, EmptyState 36 | |
| `<Select` / `<Field` / `<Textarea` | 69 (33 berkas) / 249 / 32 |
| `type="date"` / `datetime-local` | 25 / 4 |
| `<button>` mentah di luar komponen bersama / `<input>` mentah | 42 / 26 |
| Teks berwarna status (`text-success/warning/danger/info`) | ±173 kemunculan, 86 berkas; **semua gagal 4,5:1 di atas latar lencana** (DESIGN.md §4.2) |
| Istilah campur | "Project" ±53 vs "Proyek" ±4, "Client" ±63 vs "Klien" ±8 (hitungan kasar teks TSX) |
| **Panggilan API per layar** (audit kode) | Detail Proyek ±9 (sebagian ganda); tab Vendor proyek 8; Beranda Portal ±9; tab Vendor Portal 7; Kendala Portal 4 |
| **Aksi multi-panggilan tidak atomik** | Buat proyek 5 berurutan; catat pembayaran + bukti 2–3 (+ muat ulang) |
| Pustaka UI pihak ketiga / tes UI / Storybook / mode gelap | `lucide-react`, `@svar-ui/react-gantt` / tidak ada / tidak ada / belum |

Temuan sampingan yang perlu diperbaiki terlepas dari revamp:
- **Bug:** "Aktivitas Terbaru" di Beranda WO membaca nama staf dari store yang tidak dimuat halaman itu, sehingga
  pada muatan pertama semua pelaku tampil "Sistem" (`modules/dashboard/components/RecentActivity.tsx:9,17`). Diperbaiki
  tuntas oleh E12, tambalan cepat boleh lebih dulu.
- **Data:** akun klien diizinkan memanggil `GET /vendors/summary` (`cmd/server/client_scope.go`), yang mengembalikan
  **nama seluruh vendor tenant**, bukan hanya vendor proyeknya. Ditutup oleh E6.
- **Dokumen basi:** `API_CONTRACT.md` menyebut pencarian global difilter di browser (sebenarnya di server), daftar
  `/projects` tanpa `progress` (sebenarnya ada), dan PATCH kendala menerima `{status}` (sebenarnya butuh badan penuh).

## 2. Tujuan dan bukan tujuan

**Tujuan**
- **Mudah dipakai, terukur:** tingkat keberhasilan tugas inti ≥ 90% per tugas (bukan rata-rata), waktu tugas dan
  salah langkah turun dari garis dasar Tahap 0, skor SEQ ≥ 6 dari 7 untuk tugas Portal.
- **Rupa modern yang tenang** (DESIGN.md §5): netral hangat, merek tenant dipakai hemat, satu elemen khas (`ProofStamp`).
- Komponen shadcn/ui + Base UI; WCAG 2.2 AA pada tiga console, dengan gerbang otomatis (axe) dan manual.
- **API melayani layar:** satu panggilan untuk memuat layar utama, mutasi mengembalikan keadaan terbaru, aksi
  bisnis atomik, galat per field.
- Satu bahasa: Bahasa Indonesia penuh sesuai glosarium (DESIGN.md §7).
- Tanpa big-bang: setiap tahap berakhir dengan aplikasi utuh.

**Bukan tujuan** (DESIGN.md §10): mode gelap, kaca/translusen, asisten AI, palet perintah sebagai navigasi utama, tur
popup, Recharts, Storybook, mengubah susunan menu/alur per peran di luar FLOW-IDEAL, pembuat warna bebas untuk tenant,
fitur baru di luar kebutuhan layar (mis. pengingat otomatis termin dicatat sebagai backlog, bukan revamp).

## 3. Arsitektur target

**Frontend**
- `shadcn init` di Vite dengan basis Base UI; `components.json` beralias `@/*`; komponen di
  `src/shared/components/ui/` tetap bebas domain. Versi CLI dikunci (`shadcn` 4.x).
- **Token:** `theme.css` tetap sumber kebenaran; `@theme inline` memetakan token shadcn (`--primary`, `--ring`,
  `--sidebar`, …) ke token kita (DESIGN.md §4). `applyBrandColorPreset` tetap menimpa `--brand-navy-*`; 21 preset
  otomatis berlaku. Token teks status baru (`--color-*-text`).
- **Strangler:** nama ekspor lama (`Button`, `Modal`, `ConfirmDialog`, `Drawer`, `Select`, `Table`, `toast`, …)
  dipertahankan sebagai pembungkus tipis; API komposisi baru (`Dialog*`, `Field*`, `Sheet*`) untuk layar baru.
- **Data per layar:** tiap layar utama memuat satu endpoint tampilan (E1, E5) lewat store modulnya; mutasi memperbarui
  store dari respons (E8), bukan memuat ulang semua daftar. Aksi rutin optimistis dengan `useOptimistic` + "Urungkan".
- **Keadaan URL:** filter, urutan, halaman, dan tampilan tersimpan di query string (bisa dibagikan, tombol kembali
  bekerja).
- **Formulir:** `Field` shadcn di atas `safeParse`; galat server dipetakan ke field dari `errors` (E10).
  react-hook-form hanya untuk formulir yang membutuhkannya (Buat Proyek dicoba sebagai ukuran).
- **Toast:** pembungkus `toast.*` dipertahankan, mesinnya Sonner (D4). **Tanggal:** pemilih native.
- **Dipertahankan dan di-restyle:** Gantt, MilestoneRail, ProgressMeter, QrCode, EvidenceViewer.

**Backend** (aturan `.claude/rules/backend-modular-monolith.md`, `api-standard.md`, `database.md` tetap berlaku)
- Endpoint tampilan dimiliki modul pemilik layar (hampir semua `projects`); data modul lain (nama staf, nama vendor,
  kontak tenant) diambil lewat **kontrak** modul itu (pola `StaffNameResolver`, `GetVenueSummary`), tanpa join lintas
  modul.
- Format tetap `{ success, message, data }` / `{ success:false, message, errors }`; galat validasi 422 dengan kunci
  field berjalur (`terms[2].amount`) dan `code` opsional, mengikuti semangat RFC 9457 tanpa mengganti amplop.
- **Kompatibilitas:** perubahan tambahan dulu (field baru, endpoint baru, parameter opsional). Perubahan yang memutus
  (badan PATCH parsial, toggle eksplisit) memakai jalur baru, jalur lama hidup sampai semua pemanggil web pindah lalu
  dihapus di tahap yang sama; tidak ada klien eksternal untuk endpoint ini selain web (App eksternal hanya memakai modul
  `payment`, tidak tersentuh).
- Setiap perubahan endpoint: tes Go (unit + `tests/isolation` untuk data klien/tenant), `API_CONTRACT.md` diperbarui di
  komit yang sama, koleksi Postman diperbarui.

### Pemetaan komponen

| Sekarang (pemakai) | Target | Catatan |
|---|---|---|
| Button (78) | `Button` (+ `neutral`, `icon`) | |
| Input/Textarea (56/32) | `Input`, `Textarea`, `Field*` | batas input 3:1 |
| Card (53), StatCard (1) | `Card` | kartu hanya untuk hal yang berdiri sendiri |
| Badge (48) + `TenantStatusBadge` dll. | `Badge` + `StatusChip` (ikon + teks + token `-text`) | label dari `shared/lib/labels.ts` |
| Modal (41), Evidence*Modal, ImportBulkModal, AccessLinkDialog | `Dialog` | viewer bukti layar penuh |
| ConfirmDialog (6) | `AlertDialog` | hanya aksi tak terpulihkan; aksi rutin pindah ke "Urungkan" |
| Drawer (3) | `Sheet` | juga menu HP; catat pembayaran di drawer |
| Combobox/`Select` (5 / 69 titik) | `Select` + `Combobox` | pembungkus menerima `<option>` |
| ActionMenu (1), IconActionButton (19) | `DropdownMenu`; `Button size=icon` + `Tooltip` | ≥ 24 px, nama aksesibel |
| TabNav (6) | `Tabs` (+ `NavLink` untuk tab berbasis rute) | |
| Table (19), CardList (18), Pagination (18) | `Table` + kartu HP + `Pagination` + bilah aksi massal | TanStack Table hanya bila perlu |
| EmptyState (36), LoadError (20), Skeleton (7) | tetap sendiri di atas `Alert`/`Skeleton` | empty state mengajar (DESIGN.md §8) |
| Toaster (3), `toast` (36) | Sonner di balik `toast.*` | + aksi "Urungkan" |
| CurrencyInput (10), SearchInput (10) | `Input` + logika lama | satu pemformat "Rp1.500.000" |
| `OnboardingChecklist`, `CompleteProfileModal` | `Card` + `Progress`; `Dialog` | ≤ 5 langkah, bisa ditutup |
| Sidebar WO/Platform, `GlobalSearch` | `Sidebar` shadcn (netral), `Command` untuk ⌘K | Portal Klien tidak memakai sidebar |

## 4. Perubahan endpoint (E1–E14)

Dari audit kode 9 Nov 2026 (bukti `file:baris` di catatan audit; ringkasan di sini). Jalur relatif ke `/api/v1`.
**Tambahan** = tidak memutus pemanggil lama. Semua milik modul `projects` kecuali disebut lain.

| # | Layar | Masalah sekarang | Perubahan | Sifat | Prio | Tahap |
|---|---|---|---|---|---|---|
| **E1** | Detail Proyek (WO) | ±9 panggilan (sebagian ganda: `ProjectHeaderCard`, `ProjectSidePanel`, `ProjectSummaryTabPage`); margin, sisa, termin berikutnya, "perlu tindakan" dihitung di browser (`projects/lib/payments.ts` meniru aturan server) | `GET /projects/{id}/overview` → `{project, progress, finance:{contractValue, paid, remaining, overdueTotal, nextTerm, vendorCost, venueCost, margin?}, actions[], nextMilestones[5], unconfirmedVendors[], team:{picName, picSalesName}, clientInvites[]}`; `margin` hanya dengan `finance.margin.view` | tambahan | P1 | 5 |
| **E2** | Tab Vendor proyek (WO) | 8 panggilan paralel termasuk seluruh vendor tenant, semua kategori, semua bukti, seluruh log; digabung di browser | `GET /projects/{id}/vendors` menyertakan `vendorName, categoryName, picName, evidenceCount` dan `evidence[]` per tahapan; `GET /projects/{id}/activity?entityType=&entityId=&page=` dengan `actorName`. Butuh kontrak baru `vendors.contracts.VendorNames(ids)` | tambahan | P1 | 5 |
| **E3** | Semua buku pembayaran | Catat pembayaran = POST lalu unggah bukti base64 per berkas, lalu muat ulang; gagal sebagian butuh kelas galat khusus ("Pembayaran tersimpan, tapi … gagal diunggah"); respons selalu `evidenceComplete:false` | `POST /projects/{id}/uploads` → URL unggah bertanda tangan + `uploadToken` (fallback base64 bila penyimpanan lokal); `POST …/{client-|venue-|}payments` menerima `evidenceTokens[]` dalam satu transaksi dan header `Idempotency-Key`; respons = pembayaran lengkap + `ledgerSummary {paid, remaining, terms[]}` | tambahan | P1 | 4 |
| **E4** | `ProofStamp` (WO + Portal) | Respons pembayaran tanpa pencatat, pemverifikasi, dan referensi bukti; tidak ada kolom `verified_*`/`recorded_by`; UI memindai seluruh daftar bukti; bukti tahapan tidak pernah tampil ke klien (`client_view.go` `clientCanSeeEvidence` jatuh ke `false`) | Migrasi: `recorded_by_staff_id`, `verified_at`, `verified_by_staff_id` pada tabel pembayaran (milik `projects`); respons + `recordedBy{name}`, `verifiedAt`, `verifiedBy{name}`, `evidence[{id,type,fileName,uploadedAt}]` pada pembayaran dan tahapan; `POST /projects/{id}/client-payments/{pid}/verify`; bukti tahapan terlihat klien dengan `portal.timeline.view` bila ditandai terlihat | tambahan + migrasi | P1 | 3 |
| **E5** | Beranda Portal | ±9 panggilan (`ClientPortalLayout.tsx`); sisa, tagihan berikutnya, jumlah kendala, kabar terbaru dihitung di browser | `GET /projects/me/home` → `{project, countdown, nextTerm, remaining, actions[], openIssueCount, recentUpdates[], pic:{name, phone}, payTo}`, disaring izin portal; rekening/WA dari `platform` lewat kontrak | tambahan | P1 | 3 |
| **E6** | Vendor + Kendala Portal | 7 dan 4 panggilan; menarik `/vendors/summary` dan `/vendor-categories?all=true` seluruh tenant hanya untuk nama (klien melihat **semua** nama vendor tenant) | Nama dan bukti per item disertakan di respons yang sudah disaring untuk klien; hapus `vendors/summary` dan `vendor-categories` dari allowlist klien (`cmd/server/client_scope.go`) + tes isolasi | tambahan, lalu **menutup akses** | P1 | 3 |
| **E7** | Buat Proyek | 5 panggilan berurutan tidak atomik (`ProjectCreatePage.tsx`): POST, muat seluruh proyek, GET, PATCH badan penuh untuk venue, PUT termin; gagal = "Project dibuat, tetapi … belum tersimpan" | `POST /projects` menerima `venueId, venueRentalPrice, venueCharge, terms[]` dalam satu transaksi; `POST /projects/{id}/vendors` menurunkan `categoryId` dan `eventDate` di server | tambahan | P1 | 4 |
| **E8** | Setelah mutasi | `progress` hanya dihitung di `GET /projects/{id}`; mutasi tahapan/kendala/vendor hanya mengembalikan butir, sehingga persen dan kondisi header salah sampai muat ulang | Respons mutasi = `{item, projectProgress, ledgerSummary?}` | tambahan | P1 | 5 |
| **E9** | Edit di tempat, optimistis | PATCH proyek/tahapan/kendala butuh badan penuh; toggle (`toggle-active/archive/client-visible`) membalik keadaan, jadi klik ganda/ulang membatalkan dirinya; tanpa versi, dua staf saling menimpa diam-diam | PATCH parsial (JSON merge) di jalur yang sama dengan validasi "field yang dikirim saja"; `PATCH …/{id} {isActive|isArchived|isClientVisible: bool}`; `If-Match: <updatedAt>` → 409 bila basi. Jalur toggle lama dihapus setelah pemanggil pindah | **memutus** (bertransisi) | P2 | 8 |
| **E10** | Semua formulir | `PUT payment-terms` berhenti di baris galat pertama tanpa indeks; `code` hilang saat galat validasi juga punya field (`writeAppError`); tanggal salah apa pun dilaporkan sebagai `eventDate`; username/email ganda di `identity` = 409 tanpa field (modul `clients` sudah 422 + field); 13 dari 83 `apperror.Validation` tanpa field | Selalu 422 `{errors:{"terms[2].amount":[…]}, code?}`, kumpulkan semua galat sebelum kembali; `writeAppError` meneruskan `code`; seragamkan `identity` | tambahan (bentuk sama, isi lebih lengkap) | P2 | 0 (format) + 4 (per formulir) |
| **E11** | Daftar dan tabel | `pagination.FromRequest` hanya `page/limit`, tanpa urutan; halaman Klien memuat semua proyek + semua klien lalu menyaring, mengelompokkan, dan memaginasi di browser; respons klien tanpa nama proyek | `?sort=field:asc|desc` di semua daftar berpaginasi (daftar putih kolom per endpoint); `/clients?role=&loginStatus=&projectId=` + `projectName, projectEventDate`; `/projects?picStaffId=&eventFrom=&eventTo=` + `remaining, nextDueDate`; `/vendors`, `/venues`, `/staff` `?isActive=` + `categoryName` | tambahan | P2 | 6 |
| **E12** | Nama di respons | Aktivitas, bukti, kendala, baris dashboard hanya berisi ID; nama dicari dari daftar penuh di browser (termasuk bug "Sistem") | Tambah `actorName, uploadedByName, vendorName, relatedLabel` lewat kontrak staff/vendors | tambahan | P2 | 5 |
| **E13** | Urungkan, Kendala, Beranda | Pembayaran, klien, vendor dihapus permanen (tidak bisa "Urungkan"); kendala wajib punya vendor dan tanpa "Terlihat klien"; dashboard tanpa arus kas 8 minggu, pembayaran vendor ≤ 30 hari, vendor belum terkonfirmasi H-30 | `DELETE` = soft delete + `POST …/{id}/restore` (jendela ±10 dtk di UI, data dibersihkan terjadwal); kendala `projectVendorId` opsional + `isClientVisible`; tambah tiga bagian ke `/dashboard`. P3: `shiftMilestones` saat tanggal acara diubah; `POST /vendors/bulk {ids, action}`; `/finance/receivables` dan `/finance/payables` untuk menu Keuangan | tambahan + migrasi | P2/P3 | 5 (urungkan, kendala, dashboard), 6 (massal), backlog (finance) |
| **E14** | Transaksi Platform | Dua panggilan berurutan (halaman lalu `/tenants/names`) | `tenantName` di transaksi lewat kontrak platform→billing; `?search=&from=&to=` | tambahan | P3 | 6 |

**Tidak diubah** (sudah tepat): `/dashboard` dan `/dashboard/mine` (diperluas, bukan diganti), `/platform/metrics`,
`/platform/trends`, `/tenants/names`, `/console/tenants/{id}/overview`, `/projects/{id}/transition`,
`/closing-checklist`, `GET /payment-terms` (status dihitung server), `/projects/{id}/documents`, `/evidence/{id}/link`
(presigned), allowlist dan redaksi klien (`client_view.go`), daftar berpaginasi dengan pencarian di SQL, impor
vendor/venue dengan galat per baris, alur auth/undangan/langganan. Bagikan via WhatsApp dibangun di browser dari token
tautan akses (`shared/lib/access-link.ts`), tidak butuh endpoint.

## 5. Tahap

Setiap tahap: cabang `revamp/ui-t<N>`, satu komit per komponen/endpoint/berkas, boleh berhenti di akhir tahap mana pun
dengan aplikasi utuh. **Gerbang tahap:** `tsc`, build, dan tes Go bersih; selisih visual Playwright disetujui; axe tanpa
pelanggaran serius/kritis pada rute yang tersentuh; uji papan ketik halaman yang berubah; INP p75 ≤ 200 ms dan anggaran
bundel tidak terlampaui; tidak ada galat konsol; `API_CONTRACT.md` sesuai kode untuk endpoint yang berubah.

### Tahap 0 — Fondasi, pengukuran, dan pengaman
- **Uji kegunaan garis dasar** (sebelum perubahan apa pun, §7): 5 staf WO + 5 klien, tugas tetap, direkam dan dicatat.
- `shadcn init` (Base UI), kunci versi CLI, pastikan `theme.css` tidak tertimpa; cek `add sonner --dry-run` (D4).
- **Token** (DESIGN.md §4): pemetaan semantik, `--input` 3:1, `muted-foreground`, **token teks status baru** dan latar
  lencana baru, netral hangat bila D13 disetujui, skala z-index, `isolation: isolate`, token kepadatan.
- Satu pemformat uang "Rp1.500.000" dan tanggal (DESIGN.md §7); glosarium ke `shared/lib/labels.ts` (D15).
- **E10 format:** `writeAppError` meneruskan `code`, galat validasi berjalur; tambalan cepat bug "Sistem".
- Perbarui tiga bagian `API_CONTRACT.md` yang basi (§1).
- **Harness Playwright** (`apps/web/e2e/`): tumpukan lokal (MySQL sementara + API + data benih lewat API, pola
  `apps/api/tests/isolation`), potret halaman emas 1280/375 px × {normal, kosong, galat, dialog terbuka}, axe; potret
  dasar di-commit sebelum perubahan visual pertama; satu gambar Docker Playwright di CI.
- Garis dasar performa: bundel per rute, INP, LCP di Android kelas menengah, **jumlah panggilan API per layar**.
- Prototipe risiko: (a) Dialog + Select di dalam Dialog + scroll lock di iOS Safari nyata; (b) variabel tenant di konten
  portal (preset gold, emerald, navy); (c) pembungkus `Select` yang menerima `<option>`; (d) Plus Jakarta Sans `tnum`
  dan ukuran (D14).
- **Selesai bila:** harness hijau pada kode lama, garis dasar uji kegunaan tercatat, token dipetakan tanpa selisih
  visual selain perbaikan kontras yang disengaja, empat prototipe lulus atau keputusan diubah.

### Tahap 1 — Primitif dasar
Button, Input/Textarea/Field, Badge + `StatusChip`, Card, Skeleton, Avatar, Separator, Tooltip, Progress. Pembungkus
mempertahankan props lama. **Selesai bila:** titik panggil tidak berubah di sumber, potret hanya berubah pada perbaikan
yang disengaja, 42 `<button>` dan 26 `<input>` mentah diganti, seluruh teks status memakai token `-text`.

### Tahap 2 — Lapisan overlay
Dialog, AlertDialog, Sheet, DropdownMenu, Popover, Tabs, Sonner (+ aksi "Urungkan" di API toast). `useDialogBehavior`
dipensiunkan. Diuji: modal menunggu hasil simpan, galat di dalam modal, Escape hanya menutup dialog teratas, fokus
kembali ke pemicu. **Selesai bila:** 41 pemakai Modal dan 6 ConfirmDialog tanpa perubahan sumber; uji iOS Safari dan
Android Chrome lulus; tidak ada `role="dialog"` buatan sendiri.

**Catatan pelaksanaan (9 Nov 2026).**
- *Tahap 0:* harness, token, formatter, glosarium, format galat dan tambalan "Sistem" selesai. Garis dasar uji kegunaan
  menunggu manusia (`UJI-KEGUNAAN.md`). Prototipe (b) variabel tenant di portal, (c) pembungkus `Select` yang menerima
  `<option>`, (d) keterbacaan font kecil, dan `shadcn add sonner --dry-run` belum dijalankan; (a) iOS Safari nyata butuh
  perangkat. Axe: serius 68 → 10, kontras warna 0; sisanya struktural (Tahap 5/7).
- *Tahap 1:* penggantian 42 `<button>` dan 26 `<input>` mentah **ditunda**: input teks diganti saat formulirnya disentuh
  (Tahap 3/4), tombol struktural (sidebar, akordeon, tab) di Tahap 5–7. Huruf kapital pada pil ("BUKTI BELUM ADA") di
  sapuan bahasa Tahap 7.
- *Tahap 2:* `Modal`, `Drawer`, `ConfirmDialog` (kini AlertDialog), `ActionMenu` (DropdownMenu), `EvidenceViewerModal`,
  dan `Toaster` (Sonner) memakai kit tanpa perubahan di titik panggil; `useDialogBehavior` dihapus; `toast.success(msg,
  { undo })` menambah aksi "Urungkan". `TabNav` tetap `<nav>` karena itu navigasi rute, bukan tab dalam halaman.
  Tes perilaku: `e2e/tests/overlays.spec.ts` (fokus masuk/kembali, Tab terkunci, Escape, galat di dalam dialog,
  drawer menolak tutup saat menyimpan, menu → dialog). **Belum diuji:** Escape pada dialog bersarang, iOS Safari dan
  Android Chrome nyata. Ditemukan: konfirmasi batalkan/hapus proyek memakai banner inline, bukan dialog (diperbaiki di
  Tahap 5).

### Tahap 3 — Irisan vertikal: Portal Klien + `ProofStamp`
API: **E4** (migrasi verifikasi + bukti di respons + endpoint verifikasi), **E5** (`/projects/me/home`), **E6** (nama
dan bukti di respons klien, tutup `vendors/summary` dan `vendor-categories` untuk klien). UI: Beranda (kartu tagihan
berikutnya, yang perlu Anda lakukan, kabar terbaru dengan `ProofStamp`), Pembayaran, Jadwal (rel vertikal, terbaru di
atas), Vendor, Kendala, tab bawah ≥ 44 px, judul Fraunces (D8), banner luring, petunjuk Layar Utama. WO: tombol
"Verifikasi pembayaran" sederhana di buku pembayaran klien agar data E4 terisi.
**Selesai bila:** Beranda Portal memuat dengan ≤ 2 panggilan API; LCP ≤ 2,5 dtk di Android kelas menengah; tes isolasi
membuktikan klien tidak bisa membaca nama vendor di luar proyeknya; `ProofStamp` tidak pernah muncul tanpa bukti (tes);
**uji kegunaan putaran 2** dengan 5 klien pada tugas Portal.

*Tahap 3 (9 Nov 2026).* API: E4 (migrasi 000065, verifikasi pembayaran, `evidence[]` dan nama pada respons, bukti tahapan
terlihat klien bila ditandai), E5 (`GET /projects/me/home`), E6 (nama vendor/kategori/tahapan di respons; `vendors/summary`
dan `vendor-categories` ditutup untuk klien) selesai dengan tes unit dan 256 skenario isolasi hijau; ADR-0032. UI: Beranda
dari satu panggilan, `ProofStamp` (sekali per peramban untuk animasinya), Jadwal dua rel (berikutnya, selesai terbaru di atas),
Pembayaran/Vendor/Kendala dari satu panggilan masing-masing, banner luring, petunjuk Layar Utama, tombol "Verifikasi" di buku
pembayaran WO. Terukur (`reports/BASELINE.md` vs `before-t0/`): panggilan API saat Beranda Portal dibuka 10 → 3 (access/me,
branding, home), JS gzip 148 → 172 KB, LCP 2292 → 2420 ms (target ≤ 2500 terpenuhi, margin tipis). Sonner dan penampil bukti
kini dimuat saat dibutuhkan; tanpa itu JS gzip Beranda 213 KB dan LCP 2672 ms. **Belum:** target "≤ 2 panggilan" tercapai untuk data
Beranda (1 panggilan) tetapi 3 bila kerangka dihitung; uji kegunaan putaran 2 (manusia); LCP di Android nyata; iOS Safari.
**Perlu diketahui:** potret kini mengandaikan MinIO lokal (lihat `e2e/README.md`).

### Tahap 4 — Formulir dan aksi atomik
API: **E3** (unggah bertanda tangan, pembayaran + bukti satu transaksi, `Idempotency-Key`, `ledgerSummary`), **E7**
(buat proyek atomik), **E10** per formulir (`payment-terms` semua baris, `identity` 422 + field). UI: `Field` di 33
formulir (urut dari yang paling sering dipakai) dengan aturan validasi hibrida, ringkasan galat bila ≥ 3, galat server
per field, tombol kirim tidak dinonaktifkan karena validitas, nilai bawaan cerdas (DESIGN.md §8). Catat pembayaran
pindah ke drawer dengan sisa dan isi otomatis pelunasan. Buat Proyek dicoba dengan react-hook-form.
**Selesai bila:** tidak ada lagi pesan "tersimpan, tetapi … gagal" di kode; buat proyek dan catat pembayaran masing-masing
satu panggilan bisnis; uji papan ketik penuh pada Buat Proyek, Catat Pembayaran, Peran & Akses, Daftar/Masuk.

*Tahap 4 (9 Nov 2026).* API (ADR-0033): `POST /projects/{id}/uploads` (token bertanda tangan; PUT langsung ke penyimpanan atau lewat API),
pembayaran klien/vendor/venue menerima `evidence:[{uploadToken,type}]` dan menyimpan pembayaran + bukti dalam **satu transaksi**,
`Idempotency-Key` pada tiga POST pembayaran dan `POST /projects`, `ledgerSummary` di respons, `POST /projects` menerima venue + termin
(semua-atau-tidak-sama-sekali, semua galat sekaligus dengan kunci `terms[i].amount`), vendor proyek menurunkan kategori/tanggal di server.
UI: kit formulir (`shared/forms`: `useZodForm`, `FormErrorSummary`, `Field name`): validasi saat kirim lalu per field saat blur,
ringkasan galat dari 3 galat, 422 dipetakan ke field/baris, tombol kirim hanya nonaktif saat memproses, galat yang tidak punya field
tidak pernah diam. 17 formulir WO + 19 formulir auth/Platform dipindah (laporan agen, diverifikasi tsc/build/e2e), Catat Pembayaran
menjadi drawer (`RecordPaymentDrawer`: sisa tagihan, Pelunasan mengisi sisa, bukti ikut satu permintaan, aman diulang), Buat Proyek
satu panggilan atomik. Tes: `forms.spec.ts`, `payment-flow.spec.ts` (project "mutating", berjalan setelah semua potret selesai),
204 tes e2e hijau, tes Go dan 268 permintaan isolasi hijau.
**Belum / berbeda dari rencana:** (1) react-hook-form **tidak dicoba**: `useZodForm` memberi perilaku yang sama tanpa dependensi baru;
keputusan perlu dikonfirmasi. (2) Modal *ubah* pembayaran (tiga seksi), `EvidenceUploadModal`, `PaymentTermsEditor` (baris), `PortalRoleSelect`
belum memakai kit baru. (3) Daftar/Masuk: Masuk teruji dengan papan ketik, Daftar (alur Google) tidak otomatis. (4) Unggah langsung
**melewati kompresi ulang server** (ADR-0010; kompresi sisi browser tetap jalan) dan butuh **CORS** pada bucket produksi: belum diverifikasi
di S3/R2 nyata. (5) Objek unggahan yatim tidak dibersihkan terjadwal. (6) Pesan 'tersimpan, tetapi … gagal' sudah hilang dari kode.

### Tahap 5 — Detail Proyek dan Beranda WO
API: **E1** (`/projects/{id}/overview`), **E2** (nama + bukti di vendor proyek, aktivitas terpaginasi), **E8** (mutasi
mengembalikan progres/ringkasan), **E12** (nama di respons), **E13** bagian urungkan (soft delete + restore untuk
pembayaran, klien, vendor proyek), kendala tanpa vendor + "Terlihat klien", tiga bagian baru `/dashboard`. UI: header
proyek + panel kanan + tab Ringkasan dari satu endpoint; aksi rutin optimistis + "Urungkan"; edit di tempat untuk
status kendala, PIC, tanggal; `projects/lib/payments.ts` tidak lagi meniru aturan server; Beranda WO berbasis tindakan
(FLOW-IDEAL §2.3); `OnboardingChecklist` ≤ 5 langkah.
**Selesai bila:** Detail Proyek memuat dengan ≤ 3 panggilan; tab Vendor ≤ 2; header tidak basi setelah mutasi apa pun
(tes); tidak ada nama "Sistem" palsu.

*Tahap 5 (10 Nov 2026).* API (ADR-0034): `GET /projects/{id}/overview` (E1: proyek, keuangan dengan margin hanya untuk `finance.margin.view`, aksi,
tahapan berikutnya, tim, klien), vendor proyek dengan `picName`/`evidenceCount`, aktivitas dipaginasi dengan `actorName` (E2, E12), semua mutasi
membawa `projectProgress` (+ `ledgerSummary`) (E8), hapus lunak + `restore` + pembersihan terjadwal untuk pembayaran klien/vendor/venue,
vendor proyek, dan klien (E13), kendala tanpa vendor + `isClientVisible`, tiga bagian baru `/dashboard` (arus kas 8 minggu, tagihan vendor, vendor
belum pasti). UI: header, panel kanan, dan Ringkasan dari satu panggilan; penulisan apa pun di bawah `/projects/{id}` otomatis menerapkan progres
baru dan memuat ulang overview (header tidak basi); hapus pembayaran/klien tanpa "Yakin?" + toast "Urungkan" (pembayaran terverifikasi dan klien
tetap minta konfirmasi); Batalkan/Hapus proyek kini dialog konfirmasi nyata (hapus permanen minta nama diketik); tab Kendala WO (tambah, ubah,
status langsung, kendala tentang acara, terlihat klien); tab Vendor memuat dua panggilan dan sisanya saat dibuka; Beranda WO dengan arus kas
dan tagihan vendor dan tanpa memuat seluruh vendor.
Terukur (`reports/BASELINE.md` vs `before-t0/`, termasuk 4 panggilan kerangka): Detail Proyek Ringkasan 16 → 5 panggilan, tab Vendor 24 → 7, tab
Keuangan 19 → 9, tab Kendala 16 → 7. Tes: `overview.spec.ts` (anggaran panggilan), `tahap5.spec.ts` (header mengikuti perubahan, hapus + urungkan, project
`mutating`), 20+ tes lain hijau; tes Go dan 291 permintaan isolasi hijau.
**Belum / berbeda:** (1) edit di tempat hanya untuk status kendala (PIC dan tanggal menunggu PATCH parsial, Tahap 8). (2) `projects/lib/payments.ts` masih dipakai
modal *ubah* pembayaran dan petunjuk venue/vendor; sisa tagihan klien kini dari server. (3) `OnboardingChecklist` sudah 5 langkah (paket langganan, bukan
'atur termin'); tidak diubah. (4) Tidak ada UI hapus vendor proyek maupun hapus tahapan proyek (rutenya tidak ada). (5) Overview memakai 16 SELECT, di atas target ~10.
(6) Klien yang dihapus/dipulihkan belum masuk log aktivitas. (7) Perilaku API berubah: `GET /activity` tanpa parameter hanya 20 terbaru, DELETE tidak lagi `data:null`.

### Tahap 6 — Tabel dan daftar
API: **E11** (urutan dan filter server, kolom turunan), **E14**, aksi massal vendor/venue (E13 P3). UI: `Table` dengan
rupiah rata kanan `tabular-nums`, dua kepadatan (40/32 px), header sticky yang tidak menutup fokus, header sortable APG,
urutan default, empat keadaan seragam, `Pagination` dengan total, filter/urutan/halaman di URL, tampilan tersimpan,
bilah aksi massal, kartu HP. Mulai dari Platform Console (Tenant, Tagihan, Audit Log) lalu WO (Proyek, Klien, Vendor,
Venue, Tim). **Selesai bila:** tidak ada daftar yang memuat `all=true` untuk menyaring di browser (kecuali pemilih
kecil yang dibatasi); **uji kegunaan putaran 3** dengan 5 staf WO.

*Tahap 6 (10 Nov 2026).* API (ADR-0035): `?sort=field:asc|desc` dengan daftar putih per endpoint dan pemecah seri id (422 bila salah; `meta.sort`), filter dan kolom turunan
(proyek: `remaining`, `nextDueDate`, `picStaffId`, `eventFrom/To`; klien: daftar datar berhalaman dengan `projectName`, `projectEventDate`, `loginStatus`; vendor/venue/tim:
`isActive`, `categoryName`/`roleName`), E14 (`tenantName`, `search`, `from`, `to` di transaksi; sort tenant dan audit log, indeks `action`), `POST /vendors/bulk` dan
`/venues/bulk`. UI: kit `shared/data-table` (`DataTable`: header urut dengan `aria-sort`, rupiah rata kanan tabular, dua kepadatan 40/32 px, header lengket, empat keadaan
seragam, total di kaki, seleksi + bilah massal, kartu di HP; `useListState`: halaman, urutan, pencarian, dan filter di URL; tampilan tersimpan di peramban) dan 8 daftar
dipindah: Proyek (kartu menjadi tabel), Klien (datar, tanpa memuat semua proyek dan klien), Vendor, Venue, Tim, Tenant, Tagihan (tanpa panggilan `/tenants/names`), Audit Log.
Tes: `lists.spec.ts` (urutan lewat papan ketik, URL, "Reset filter", rupiah, tidak ada `all=true`), `tahap6.spec.ts` (aksi massal, project `mutating`); 40 potret diperbarui; axe tidak berubah.
**Belum:** (1) **uji kegunaan putaran 3** dengan 5 staf WO (manusia). (2) Pemilih kecil (proyek/staf, `limit=100`) masih ada di filter Klien dan Proyek. (3) Klien yang
diurutkan menurut proyek atau status masuk diurut di memori server (O(klien satu tenant), tercatat di ADR); urut transaksi menurut `tenantName` juga. (4) Aksi massal tidak
tercatat di log aktivitas (toggle tunggal pun tidak). (5) `ProjectCard` tidak terpakai lagi. (6) Daftar Tenant/Klien masih LCP ±3 dtk di profil lab (tidak diukur ulang
terpisah sebelum/sesudah). (7) Filter `Select` kini punya nama aksesibel lewat `aria-label` baru pada `Combobox`.

### Tahap 7 — Layout, navigasi, dan bahasa
`Sidebar` shadcn netral untuk WO dan Platform (D13; Sheet di HP, `Cmd/Ctrl+B`), menu sesuai FLOW-IDEAL §2.2 dengan
istilah glosarium (Proyek, Klien, Direktori, Keuangan bila E13 finance dikerjakan, Tim), `Breadcrumb` di halaman
detail, ⌘K memakai `Command` di atas pencarian global server yang sudah ada, banner langganan/pengumuman/"lihat sebagai".
Sapu bahasa: semua teks UI mengikuti DESIGN.md §7 (tombol kata kerja + objek, galat apa + cara memperbaiki).
**Selesai bila:** perilaku peran/izin identik; target sentuh diuji di HP nyata; banner sesi dukungan tetap teratas
(z-index 70); tidak ada istilah di luar glosarium (pemeriksaan otomatis teks).

*Tahap 7 (10 Nov 2026).* Hanya web; API tidak berubah. **Menu:** satu `AppSidebar` netral (`shared/layouts/nav/`) dipakai WO dan Platform: rel ikon di komputer
(`Ctrl/Cmd+B`, pilihan diingat), `Sheet` di HP yang menutup setelah pindah halaman, kelompok **Direktori** (Vendor, Venue) dan **Pengaturan** yang terbuka sendiri bila
halaman di dalamnya aktif (`aria-expanded`/`aria-controls`), izin per butir tetap lewat `visibleSections(grants)`. **Breadcrumb** (`Proyek › nama`, `Tenant › nama`) di dua halaman detail.
**⌘K** memakai `Command` di atas pencarian server yang ada (proyek, vendor, klien, masing-masing satu panggilan; satu jenis yang gagal atau tak boleh dibaca tidak menyembunyikan
yang lain), Platform hanya mencari halaman. Banner sesi dukungan `z-70`. **Bahasa:** `apps/web/scripts/check-ui-terms.mjs` (AST, hanya teks yang dibaca orang: JSX, label,
placeholder, toast, pesan galat; tanpa huruf besar-kecil; `// ui-terms-ok` untuk pengecualian) menjadi tolok ukur sapuan ~60 berkas dan dijalankan sebagai tes
(`terms.spec.ts`) dan `npm run check:terms`; tab proyek "Timeline" → "Jadwal", "Evidence" → "Dokumen/bukti", "Staf" → "anggota tim", jenis dokumen kini berlabel
Indonesia lewat `evidenceTypeLabel` (nilai enum ke API tetap). **Aksesibilitas:** axe 68 halaman **0 pelanggaran** (sebelumnya 10 serius): baris vendor dipisah menjadi tombol
buka-tutup dan tombol aksi bernama yang bersaudara (dulu `span role=button` di dalam `button`, tanpa nama dan tanpa papan ketik), daftar definisi Profil usaha, tombol mata kata
sandi 32 px. **Performa:** menu, palet, dan tooltip rel sempat menaikkan berkas entri 146 → 204 KB gzip karena `AppLayout` dimuat eager; sheet HP, dialog palet (cmdk), dan
tooltip rel kini dimuat saat dipakai, entri kembali 146,6 KB dan LCP setara Tahap 6. Tes: `nav.spec.ts` (menu, `Ctrl+B`, sheet HP, breadcrumb, ⌘K, baris vendor), `terms.spec.ts`;
potret desktop/mobile diperbarui untuk seluruh halaman (menu dan topbar baru); 226 tes lulus.
**Belum / berbeda:** (1) **target sentuh belum diuji di HP nyata** (iOS dan Android): hanya emulasi 375 px dan axe `target-size`; dan banner sesi dukungan `z-70` belum diuji di
e2e karena butuh sesi "lihat sebagai". (2) Butir **Keuangan** tidak ada di menu: E13 (keuangan lintas proyek) belum dikerjakan. (3) Halaman hukum (`TermsPage`, `PrivacyPage`,
`RefundPolicyPage`, `DataProcessingPage`) dan prosa panjang `FaqPage` tidak disapu: pemeriksa tidak membaca template literal, dan istilah terdefinisi (Tenant, Workspace, Owner,
Staff, Client) bagian dari teks hukum; keputusannya milik pemilik produk. `AuditLogPage` masih berjudul "Audit Log". (4) **Teks riwayat aktivitas disimpan API dalam bahasa campur**
("Evidence diunggah", "Pembayaran client dicatat"); baris lama tetap begitu walau teks di Go diganti, jadi perlu keputusan: ganti teks baru saja atau migrasi data. Beberapa judul
modal masih Title Case ("Tambah Bukti Booked"). (5) Hanya satu judul Title Case yang saya perbaiki ("Perlu ditindaklanjuti"); pemeriksa belum menilai huruf besar-kecil, jadi masih ada yang tersisa. (6) Palet ⌘K di
Platform tidak mencari tenant (menunggu endpoint pencarian ringan di console). (7) Ctrl+K pertama di sesi memuat ~19 KB gzip; sebelum itu terpasang, palet baru tampil setelah
unduhan selesai (tidak ada penanda memuat).

### Tahap 8 — Layar khusus dan edit aman
API: **E9** (PATCH parsial, toggle eksplisit, `If-Match` → 409, hapus jalur toggle lama). UI: EvidenceViewer dan
timeline di-restyle, Gantt (`@svar-ui`) disesuaikan tema, TrendBarChart di-restyle, keadaan data basi, dialog "Data
diubah orang lain, muat ulang?" untuk 409. **Selesai bila:** dua sesi yang mengedit butir yang sama tidak saling
menimpa diam-diam (tes); animasi hormati `prefers-reduced-motion`.

*Tahap 8 (10 Nov 2026).* API (ADR-0036): migrasi `000071` (`updated_at` menjadi `TIMESTAMP(6)` di sembilan tabel; delapan di antaranya belum punya kolom itu); **PATCH parsial** (hanya
kunci yang dikirim yang berubah, `{}` 422, `null` hanya di bidang nullable, aturan dinilai pada hasil gabungan, hanya kolom yang berubah ditulis) untuk proyek, tahapan, vendor proyek,
tahapan vendor, kendala, dan tiga jenis pembayaran; **`If-Match`** sebagai compare-and-set satu `UPDATE … WHERE updated_at = ?`, 409 `{_conflict:["stale"], data:{current}}` tanpa menulis
apa pun, `updatedAt` dan `ETag` di semua butir dan daftar; **keadaan eksplisit** (`isArchived`, `isClientVisible` lewat `PATCH …/evidence/{id}` baru, `isActive` di lima direktori),
mengulang keadaan yang sama = 200 tanpa tulis; **tujuh jalur toggle lama dihapus** (platform tidak disentuh). Satu temuan saya sendiri: respons `transition`/`cancel`/`verify`/batal vendor
membawa `updatedAt` sebelum tulisannya, sehingga suntingan berikutnya oleh orang yang sama ditolak 409; kini dibaca ulang sesudah menulis (dan dikunci tes). Web: `updatedAt` di sembilan tipe;
form menyimpan versi **saat dibuka** (bukan versi di store saat simpan, yang bisa sudah tergantikan segar-ulang) dan mengirimnya sebagai `If-Match`; edit inline (status tahapan, status
kendala, venue) hanya mengirim yang berubah; toggle diganti `setXActive(id, bool)`; **dialog global "Data diubah orang lain"** dibuka interceptor pada 409, dimuat saat dipakai, "Muat ulang" memuat
ulang proyek sehingga form lama ikut terbuang. `TrendBarChart` bersama menggantikan tiga grafik kembar (tabel data untuk pembaca layar, label ≥ 11 px), Gantt memakai token tema aplikasi dan label
status Indonesia, penampil bukti punya keadaan muat/galat dengan "Coba lagi" dan zoom gambar. Pemeriksa istilah kini menjangkau `setError(...)` dan sejenisnya. Tes: `tahap8.spec.ts` (reduced
motion, dua sesi pada satu proyek dan satu tahapan, versi API, arsip eksplisit, urutan suntingan satu orang, versi setelah ubah status), Go: unit baru dan `TestTahap8`/`TestTahap8Migration`
(361 butir lulus di putaran agen; putaran saya setelah perbaikan `afterWrite`: 6 tes integrasi lulus dengan MinIO sementara); **234 tes e2e lulus**, axe 68 halaman **0 pelanggaran**, entri JS 146,8 KB gzip.
**Belum / berbeda:** (1) **`If-Match` masih opsional** (tanpa header = tulis terakhir menang); Tahap 9 memutuskan `428`. Tes tidak bisa membuktikan bahwa *setiap* pemanggil web mengirimnya: semua
simpan form dan edit inline proyek sudah, tetapi tidak ada pemeriksaan otomatis. (2) `docs/API_CONTRACT.md` dan koleksi Postman belum diperbarui (Tahap 9). (3) Respons **restore** dan beberapa
penulisan non-PATCH lain masih membawa butir yang dimuat sebelum tulisan; aman hari ini karena web memuat ulang daftarnya, tetapi bukan jaminan kontrak. (4) "Keadaan data basi" saya artikan sebagai
dialog 409 + muat ulang; belum ada penyegaran otomatis saat tab kembali aktif. Edit direktori (vendor, venue, klien, tim, kategori) tidak memakai `If-Match`; hanya `isActive` yang eksplisit.
(5) Setiap penulisan ke baris menaikkan versinya, jadi orang yang membuka form lalu aksi lain (batal vendor, ubah urutan tahapan) di baris yang sama akan melihat 409 saat menyimpan: itu disengaja,
tetapi pesannya tidak menyebut aksi apa yang mengubah. (6) Gantt hanya penanda tahapan (bukan rentang), tidak bisa dioperasikan keyboard, dan butuh gulir horizontal di kartu sempit (di bawah ±900 px
pustaka menyembunyikan grafik waktunya); tampilan daftar "Jadwal" tidak diubah selain token. Penampil bukti: PDF tidak berubah dan tidak ada panah berikut/sebelumnya. (7) `lists.spec.ts` (urutan
vendor) rapuh terhadap tes lain yang menambah vendor di workspace yang sama; kini menilai data respons, bukan baris pertama di layar. (8) **Uji kegunaan putaran 4** dan uji di HP nyata masih menunggu
manusia.

### Tahap 9 — Pengerasan dan penutupan
Axe di seluruh rute dan keadaan; uji papan ketik dan pembaca layar manual pada alur utama (masuk, buat proyek, catat dan
verifikasi pembayaran, lihat bukti, bayar langganan); iOS Safari dan Android kelas menengah nyata; anggaran performa
sebagai gerbang CI; hapus komponen lama, `useDialogBehavior`, dan jalur API yang digantikan; dokumentasi (ADR-0032 untuk
revamp UI, ADR-0033 untuk endpoint tampilan dan kebijakan kompatibilitas, `FRONTEND_GUIDE.md`, `API_GUIDE.md`,
`.claude/rules/frontend-react.md`); inventaris ulang angka §1; **uji kegunaan akhir** dengan tugas yang sama dengan
Tahap 0. **Selesai bila:** semua gerbang hijau di CI dan target §2 tercapai atau selisihnya dijelaskan.

*Tahap 9 (10 Nov 2026).* **Gerbang:** `.github/workflows/ci.yml` (**belum pernah berjalan di GitHub**: divalidasi sebagai YAML, dan perintah tiap langkah dijalankan lokal dalam bentuk yang sama: gofmt, build, vet, tes, suite isolasi, tipe, `check:terms`, `budget`, axe ketat, `behavior`, `mutating --no-deps`, `e2e:perf` + `budget:routes`;
hanya pemasangan biner MinIO, peramban Playwright dan service MySQL yang tidak terbukti, dan versi MinIO yang dipin belum saya unduh) dengan tiga job: `api` (gofmt, build, vet, tes, suite isolasi dengan MySQL dan MinIO asli), `web` (tipe, `check:terms`, build, anggaran bundel `perf-budget.json`) dan `e2e` (axe **ketat**,
`behavior`, `mutating --no-deps`, `e2e:perf` + anggaran per layar). Potret piksel sengaja tidak masuk CI (render macOS). **Dihapus:** tujuh jalur toggle (Tahap 8), `GET /vendors/summary`, `GET /projects/{id}/vendors/{pv}/milestones`,
sembilan berkas web tanpa pengimpor (`ProjectCard`, `UpcomingEvents`, `condition.ts`, barrel `ui/index.ts` dan `theme/index.ts`, empat komponen kit tak terpakai), empat aksi store mati dan konstanta endpointnya
(`useDialogBehavior` sudah hilang sebelumnya). **`If-Match` kini wajib (428)** pada delapan PATCH edit-field (ADR-0036); PATCH keadaan (`isArchived`, `isClientVisible`, `isActive`) tetap tanpa header;
404 mendahului 428 agar keberadaan baris tidak bocor. **Dokumentasi:** `API_CONTRACT.md` disinkronkan dengan 211 rute yang benar-benar terdaftar (31 belum pernah terdokumentasi, 9 dihapus, ±60 baris salah),
`API-INVENTORY.md` (rute × pemakai web × kandidat hapus), ADR-0037 (revamp UI dan gerbangnya), `FRONTEND_GUIDE.md`, `API_GUIDE.md`, `.claude/rules/frontend-react.md`, README e2e. **Temuan yang diperbaiki di tahap ini:**
(a) **judul tab sama di seluruh halaman** tiap konsol ("WO Contoh — WO Console"; WCAG 2.4.2): kini per rute lewat `handle.title` + `useRouteTitle`; (b) halaman Masuk tanpa `main` dan `h1` hanya di layar lebar, tab Dokumen portal tanpa `h1`;
(c) slogan dan judul bawaan masih Inggris ("Every Step to Your Perfect Day.", "Client Transparency Portal"); (d) ±30 teks Inggris (Export, Import, Download, Reset, Password) lolos pemeriksa istilah, yang kini mencakup kata kerja itu;
(e) ekspor vendor/venue mengabaikan `q` dan `isActive` sehingga daftar terfilter dan berkasnya berbeda; (f) Gantt dimuat statis (tab Jadwal 87 KB → 4,7 KB, kini dimuat saat tampilan Gantt dipilih); (g) seed e2e dan tes dua-sesi mengedit tanpa versi.
Tes baru: `structure.spec.ts` (satu h1, satu main, judul tab unik per halaman, 4 konsol), `keyboard.spec.ts` (Masuk dengan papan ketik), ekspor membawa filter, 428 pada delapan sumber daya. **241 tes e2e lulus**; axe 68 halaman 0 pelanggaran;
Go: build, vet, tes unit, dan 387 subtes integrasi lulus dengan MySQL dan MinIO sementara.

**Inventaris ulang §1** (metode sama dengan Tahap 0, 10 Nov 2026): berkas `.tsx` 194 → 230 dan baris TS+TSX ±33.500 → 36.300 (kit, `shared/forms`, `data-table`, tes terpisah di `e2e/`); halaman 60 → 60; form dengan `safeParse` langsung 33 → 6
(sisanya lewat `useZodForm`: 39); komponen bersama ±35 → 56 (30 `ui` + 19 `kit` + `feedback`); `<Select` 69 → 72, `<Field` 249 → 235; `type="date"` 25 → 27; `<button>` mentah di luar komponen bersama 42 → 33 dan `<input>` mentah 26 → 17 (sisanya kontrol khusus: pemilih berkas,
kotak centang, akordeon, menu; **target penggantian penuh tidak tercapai**, axe bersih); teks berwarna status **tanpa** token `-text`: ±173 → **0** (145 pemakaian token `-text`); istilah campur "Project"/"Client" di teks JSX ±116 → 0 (pemeriksa otomatis);
**panggilan API per layar** (termasuk 3–4 panggilan kerangka: branding, izin, langganan, pengumuman): Beranda Portal 10 → 4, Jadwal 10 → 4, Kendala 13 → 4, Pembayaran 11 → 5, Vendor 15 → 6; Detail Proyek Ringkasan 16 → 5, Keuangan 19 → 9, Kendala 16 → 7, tab Vendor 24 → 7;
pustaka pihak ketiga baru: `@base-ui/react`, `sonner`, `cmdk`, `class-variance-authority`; tes UI: Playwright + axe (sebelumnya tidak ada); Storybook tidak ada (D10); mode gelap belum (D6).

**Bukti target §2:**
| Target | Hasil |
|---|---|
| Keberhasilan tugas ≥ 90% per tugas, waktu dan salah langkah turun, SEQ ≥ 6 | **Tidak terukur.** `uji-kegunaan-lembar.csv` hanya berisi baris judul: tidak ada hasil putaran 1–3 yang tercatat, dan putaran 4 belum dilakukan. Perbandingan "sebelum/sesudah" tidak bisa diklaim. |
| Rupa modern yang tenang | Dilaksanakan menurut DESIGN.md; penilaian rupa itu subjektif dan keputusan D8, D13, D14, D15 menunggu persetujuan pemilik (ADR-0037). |
| shadcn/ui + Base UI; WCAG 2.2 AA dengan gerbang otomatis dan manual | Otomatis: axe ketat 0 pelanggaran di 68 halaman × 2 viewport, gerbang CI, tes struktur dan papan ketik. **Manual: belum** (`UJI-MANUAL.md`: pembaca layar, iOS dan Android nyata, target sentuh, banner sesi dukungan z-70). |
| API melayani layar: satu panggilan, mutasi mengembalikan keadaan terbaru, aksi atomik, galat per field | Tercapai untuk **data layar**: Beranda Portal 1 panggilan (`/projects/me/home`), Detail Proyek 1 (`/overview`), tab Vendor 2; **total** termasuk kerangka 4, 5 dan 7, di atas target "≤ 2/≤ 3/≤ 2" bila kerangka dihitung (dijelaskan di catatan Tahap 3). Mutasi mengembalikan `projectProgress`/butir terbaru, buat proyek dan catat pembayaran atomik, 422 per field. |
| Satu bahasa Indonesia sesuai glosarium | Tercapai dan dijaga mesin (`check:terms`, tes e2e, CI). Teks yang disimpan API di basis data (riwayat aktivitas lama) dan halaman hukum/FAQ **belum** (lihat catatan Tahap 7). |
| Tanpa big-bang | Tiap tahap berakhir dengan aplikasi utuh dan tes hijau. |
| Bundel dan INP tidak lebih buruk dari garis dasar (§7) | **Tidak terpenuhi.** JS gzip seluruh aplikasi 489 → 630 KB; JS first-load naik 19–97 KB per layar (entri 129 → 147 KB; rute WO memuat 9 chunk kecil tambahan: Base UI, form, data-table). LCP lab (HP, CPU 4×, 1,6 Mbps) naik di Beranda WO 2,5 → 3,1 dtk, Klien 2,4 → 3,1, Tenant 2,7 → 3,6; turun di tab Detail Proyek 2,6 → 1,6 dtk; Beranda Portal 2,29 → 2,49 dtk (target 2,5: **margin 12 ms**, di lab, bukan di Android nyata). INP tidak diukur dari skrip. Harga komponen aksesibel dari Base UI; dibatasi dengan anggaran (`perf-budget.json`) supaya tidak naik lagi. |

**Belum dikerjakan, dan tidak bisa oleh mesin** (naskah dan lembar hasil di `UJI-MANUAL.md`): uji papan ketik tujuh alur (K1–K7), pembaca layar pada empat kombinasi, iPhone dan Android kelas menengah nyata (target sentuh, papan ketik layar, banner z-70, jaringan buruk), dan uji kegunaan putaran 4
(serta pencatatan putaran 1–3 jika memang pernah dilakukan). Kandidat hapus yang dibiarkan ('unsure' di `API-INVENTORY.md`): `GET /clients?all=true`, `GET /projects/me`, `GET /projects/{id}`, `GET /tenants/names`, `GET /tenants/{id}`, `GET /access/roles/{id}`, `GET /auth/me`,
DELETE/restore vendor proyek dan DELETE dokumen (belum ada UI). Temuan yang tidak diperbaiki: uang di dashboard hanya sebagian di balik `dashboard.revenue.view` (`revenue`, `revenueTrend`, `overdueTermsTotal` terlihat oleh semua pemegang `dashboard.view`); PATCH proyek dengan badan non-objek
dijawab 400 sebelum 428 (lainnya 404/428 dulu). **Tahap 9 belum boleh disebut selesai** sampai bagian manusia di atas terisi.

## 6. Estimasi

| Tahap | Perkiraan (1 pengembang, hari kerja) |
|---|---|
| 0 Fondasi, garis dasar uji, harness, prototipe, E10 format | 6–8 |
| 1 Primitif | 4–5 |
| 2 Overlay | 5–7 |
| 3 Portal Klien + `ProofStamp` (E4, E5, E6) | 7–9 |
| 4 Formulir + aksi atomik (E3, E7, E10) | 10–12 |
| 5 Detail Proyek + Beranda WO (E1, E2, E8, E12, E13) | 8–10 |
| 6 Tabel dan daftar (E11, E14, massal) | 7–9 |
| 7 Layout, navigasi, bahasa | 4–5 |
| 8 Layar khusus + edit aman (E9) | 5–7 |
| 9 Pengerasan + uji akhir | 5–6 |
| **Jumlah** | **61–78 hari kerja (±12–16 minggu)** |

Naik ±19–23 hari dari revisi 1 (42–55) karena 14 perubahan endpoint dengan tes dan migrasi, empat putaran uji
kegunaan, dan sapu bahasa. Pembanding yang tidak setara: epik MUI→shadcn Coder 5,5–8 minggu; tldraw 14–20
minggu-pengembang untuk Radix→Base UI. Bantuan AI memangkas penulisan mekanis, tidak memangkas tinjauan visual, uji
pengguna, dan uji perangkat nyata. Bila waktu harus dipotong, urutan pemangkasan: E13 P3 dan E14 → Tahap 8 → E11
sebagian; **Tahap 3 dan E3/E4 tidak dipangkas** karena itu inti janji merek.

## 7. Pengujian dan pengukuran

- **Uji kegunaan** (inti revisi 2). Putaran: Tahap 0 (garis dasar), akhir Tahap 3 (Portal), akhir Tahap 6 (console),
  Tahap 9 (akhir). Peserta: 5 staf WO (Owner/Admin/Wedding Planner) dan 5 klien (pengantin/keluarga, HP Android mereka
  sendiri) per putaran; 5 orang cukup untuk menemukan sebagian besar masalah kualitatif, angka dibaca per tugas, bukan
  rata-rata. Tugas tetap:
  - WO: buat proyek dengan termin; catat pembayaran klien beserta bukti; temukan proyek yang tagihannya terlambat;
    undang klien via WhatsApp; ubah status kendala dan tampilkan ke klien.
  - Klien: berapa yang harus dibayar dan kapan; ke rekening mana; apakah DP sudah diterima WO dan buktinya; apa
    yang terjadi minggu depan; hubungi PIC.
  - Ukuran: berhasil/gagal, waktu, salah langkah, SEQ 1–7, kutipan kebingungan. Target di §2.
- **Visual:** Playwright `toHaveScreenshot` halaman emas × {1280, 375} × keadaan; selisih ditinjau manusia.
- **Aksesibilitas:** `@axe-core/playwright` (wcag2a/2aa/21a/21aa/22aa) setelah membuka dialog/menu, plus uji papan
  ketik dan pembaca layar manual.
- **API:** tes Go per endpoint baru/berubah; tes isolasi untuk data klien dan tenant (E4, E5, E6); tes idempotensi (E3:
  POST ulang dengan kunci sama tidak menggandakan pembayaran); tes konkurensi (E9); `API_CONTRACT.md` dan Postman
  diperbarui di komit yang sama.
- **Performa:** INP p75 ≤ 200 ms, LCP ≤ 2,5 dtk di Android kelas menengah, anggaran JS per rute dari garis dasar,
  komponen berat lazy, **jumlah panggilan API per layar** (target: Beranda Portal ≤ 2, Detail Proyek ≤ 3, tab Vendor ≤ 2).
- **Keberhasilan:** target §2; nol pelanggaran axe serius/kritis; tidak ada regresi fungsi (tes Go dan isolasi hijau);
  bundel dan INP tidak lebih buruk dari garis dasar.

## 8. Risiko

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Lingkup naik (API + UI + bahasa) | 12–16 minggu untuk satu pengembang, konteks hilang | tahap bisa berhenti dengan aplikasi utuh; urutan pemangkasan di §6; daftar periksa per komit |
| Perubahan API memutus layar yang belum dimigrasi | fungsi rusak | tambahan dulu; jalur lama hidup sampai pemanggil pindah; hanya E9 yang memutus dan bertransisi; harness Playwright + tes Go |
| Migrasi E4 / soft delete E13 | laporan dan kueri lama ikut menghitung data terhapus | kolom `deleted_at` + filter di repositori pemilik; tes laporan keuangan; pembersihan terjadwal |
| Endpoint tampilan melanggar batas modul | kopling lintas modul | data modul lain hanya lewat kontrak (nama, kontak); tidak ada join lintas modul (aturan `database.md`) |
| Pembayaran ganda saat jaringan buruk | data keuangan salah | `Idempotency-Key` (E3) + tes |
| Base UI lebih muda | waktu debug | prototipe Tahap 0; `isolation: isolate`; jalan keluar `init -b radix` |
| Penimpaan tema tenant tidak sampai ke konten portal | dialog/menu tak berwarna merek | variabel di `:root`; prototipe (b) |
| Perubahan rupa terlihat tenant (netral hangat, sidebar netral, istilah) | keluhan "aplikasinya berubah" | D13/D15 dikonfirmasi pemilik; pengumuman lewat banner pengumuman yang sudah ada; uji kegunaan sebelum/sesudah |
| Kode shadcn disalin ke repo | pembaruan manual | kunci versi CLI; `add --diff` per komponen |
| Bundel membengkak | LCP/INP buruk di HP Android | anggaran per rute di CI, lazy untuk komponen berat |
| Format rupiah berubah | tes yang memuat string | satu pemformat, ubah sekaligus di Tahap 0 |

## 9. Keputusan

| # | Keputusan | Rekomendasi | Status |
|---|---|---|---|
| D1 | Primitif: Radix atau Base UI | Base UI | ✅ Diputuskan 8 Nov 2026: **Base UI**; Radix jalan keluar bila prototipe gagal |
| D2 | Pembungkus berprops lama, migrasi pemanggil bertahap | Ya | Usulan |
| D3 | Formulir: `Field` + `safeParse`; react-hook-form hanya bila perlu | Ya | Usulan |
| D4 | Toast: Sonner di balik `toast.*` atau Toast bawaan Base UI | Sonner (dukung aksi "Urungkan") | Usulan |
| D5 | Format rupiah | **"Rp1.500.000"** (PUEBI, sumber primer Permendikbud 50/2015) | Rekomendasi kuat; dikonfirmasi pemilik karena terlihat pengguna |
| D6 | Mode gelap | Tunda; token tetap semantik | Usulan |
| D7 | Tanggal: pemilih native; `Calendar` hanya untuk rentang | Ya | Usulan |
| D8 | Fraunces untuk judul Beranda Portal | Ya, hanya di situ | Terbuka |
| D9 | Palet perintah | ⌘K di atas pencarian global yang ada (Tahap 7), bukan navigasi utama | Usulan |
| D10 | Storybook | Tidak; potret Playwright | Usulan |
| D11 | Grafik buatan sendiri, tanpa Recharts | Ya | Usulan |
| D12 | Nada tunggakan klien: nominal, tanggal, aksi, tanpa kata menyalahkan | Diuji di putaran uji Portal | Hipotesis |
| D13 | Rupa: netral hangat (stone) + sidebar netral, merek hanya di logo, tombol utama, fokus, item aktif | Ya (DESIGN.md §4.1, §5) | Terbuka, butuh persetujuan pemilik |
| D14 | Huruf UI: Inter atau Plus Jakarta Sans | Plus Jakarta Sans bila lolos `tnum`/ukuran di prototipe, selain itu Inter | Terbuka |
| D15 | Istilah jadwal: "Jadwal" + "tahapan" di console dan Portal (menggantikan Timeline/Milestone) | Ya, satu istilah per konsep | Terbuka |
| D16 | Endpoint boleh diubah bila alur UI tidak ideal | Tambahan dulu; memutus hanya dengan masa transisi di tahap yang sama | ✅ Diputuskan 9 Nov 2026 (pemilik produk) |
| D17 | Unggah bukti: URL bertanda tangan atau base64 dalam satu permintaan | URL bertanda tangan (progres + batal di HP), fallback base64 untuk penyimpanan lokal | Usulan |
| D18 | Urungkan: soft delete + restore untuk pembayaran, klien, vendor proyek | Ya; pembayaran **terverifikasi** tetap lewat konfirmasi + alasan, bukan urungkan | Usulan |

## 10. Hal yang belum bisa diverifikasi

Ukuran payload bukti/aktivitas dan latensi nyata per panggilan (diukur di Tahap 0); kecocokan izin baca `/staff` dan
`/vendors` dengan tulisan "hanya Owner" di dokumen; perilaku polling `/subscriptions/pending-charge`; fitur `tnum` Plus
Jakarta Sans; ukuran gzip tiap komponen shadcn; perilaku portal Base UI terhadap variabel tenant; nada pesan tunggakan;
bukti adopsi palet perintah. Sumber lengkap dengan tingkat verifikasi di `knowledge/DESIGN.md` §12.
