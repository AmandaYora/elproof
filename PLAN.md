# PLAN.md — Rencana & Checkpoint: Backend ElProof + Integrasi Frontend

> **Status dokumen:** Living plan. Update checkbox & status tiap fase setiap kali ada progres —
> jangan buat dokumen baru, edit file ini. Dibuat berdasarkan audit menyeluruh terhadap
> `apps/web/src` (frontend, React 19 + Tailwind 4) per 2026-07-16, mengikuti standar
> `.claude/rules/*` dan skill `monorepo-standard`.

**Legenda status fase:** 🔴 Belum mulai · 🟡 Berjalan · 🟢 Selesai · ⏸️ Diblokir (butuh keputusan)

---

## Daftar Isi

1. [Ringkasan Eksekutif](#1-ringkasan-eksekutif)
2. [Temuan Audit Frontend (Mendalam)](#2-temuan-audit-frontend-mendalam)
3. [Peta Modul: Frontend → Backend](#3-peta-modul-frontend--backend)
4. [Keputusan Arsitektur yang Harus Dikunci Dulu](#4-keputusan-arsitektur-yang-harus-dikunci-dulu)
5. [Rencana Fase per Fase](#5-rencana-fase-per-fase)
6. [Keputusan Terbuka (Butuh Konfirmasi User)](#6-keputusan-terbuka-butuh-konfirmasi-user)
7. [Cara Memperbarui Dokumen Ini](#7-cara-memperbarui-dokumen-ini)

---

## 1. Ringkasan Eksekutif

**Apakah frontend sudah siap dibuatkan backend?** Ya, secara model domain — tetapi ini adalah
proyek **greenfield backend**, bukan "tinggal sambung API". `apps/api` saat ini hanya scaffold
generik kosong (modul `auth`/`order`/`users` isinya cuma `.gitkeep`, satu migration tabel `users`
generik, satu endpoint health-check). Tidak ada satupun handler bisnis yang sudah ada.

Kabar baiknya: sepanjang sesi pembangunan frontend, model data yang dipakai (interface TypeScript
di `mock/types.ts`, `modules/*/data/types.ts`) **sudah konsisten memakai primitive-ID untuk relasi
lintas-entitas** (persis pola yang diwajibkan `.claude/rules/database.md`), dan Zod schema per
modul sudah mendefinisikan boundary validasi input. Ini pekerjaan pemodelan domain yang sudah
separuh jalan.

Kabar yang perlu diwaspadai: **hanya 3 dari 10 modul frontend punya Zustand store nyata**
(`useAuthStore`, `useSubscriptionPlanStore`, `usePlatformAdminStore`). Tujuh modul lainnya —
termasuk `projects`, `clients`, `vendors`, `vendor-categories`, `users` (WO staff) — memakai
`useState` lokal yang di-seed ulang dari `mock/seed.ts` setiap kali halaman dimuat. Artinya modul
inti bisnis (Project & turunannya) belum punya lapisan state yang bisa langsung "disambung" ke
API — bagian ini butuh refactor frontend yang setara besarnya dengan pekerjaan backend-nya sendiri.

Gap paling kritis yang ditemukan (detail di §2):
- **Tidak ada satupun route guard** di seluruh aplikasi — semua halaman (termasuk `/platform/*`)
  bisa diakses langsung lewat URL tanpa login.
- **Auth 100% palsu**: password plaintext di array/store, tanpa token/session/expiry, dan tiga
  "sesi" (staff/client/platform admin) hidup bersamaan dalam satu store datar.
- **Tidak ada konsep `tenantId`** di modul WO Console manapun — semua data merepresentasikan
  satu tenant hardcode.
- **Upload file itu sendiri belum pernah dibangun** — viewer bukti transaksi/dokumen selalu
  menampilkan salah satu dari 2 file demo statis, terlepas dari data aslinya.
- **Tidak ada library data-fetching terpasang** (react-query/SWR/react-hook-form nihil di
  `package.json`) — kalau pola ini mau dipakai, itu penambahan baru, bukan sesuatu yang tinggal
  diaktifkan.
- `docs/*.md` dan `knowledge/*.md` semuanya masih placeholder `_TODO._` — belum ada keputusan
  arsitektur backend yang terekam di mana pun.

**Kesimpulan:** frontend cukup matang untuk mulai merancang backend (modelnya jelas), tapi
"integrasi" di sini berarti membangun backend dari nol *sekaligus* merombak state-management
frontend untuk 7 modul yang belum punya store. Rencana di §5 mengurutkan pekerjaan ini supaya
tidak dikerjakan sekaligus dan tetap bisa diverifikasi per tahap.

---

## 2. Temuan Audit Frontend (Mendalam)

### 2.1 State management — realita per modul

| Modul frontend | Store Zustand? | Sumber data saat ini | Implikasi backend |
|---|---|---|---|
| `auth` | ❌ (nulis ke `shared/useAuthStore`) | 3 daftar hardcode dibaca berurutan | Auth module baru, bukan refactor |
| `dashboard` | ❌ (read-only) | agregasi dari `mock/selectors.ts` | Endpoint agregasi, tanpa tabel sendiri |
| `projects` (+ 6 sub-entity) | ❌ | `useState` per halaman, seed dari `mock/seed.ts` | Refactor state frontend + backend baru, sekaligus |
| `clients` | ❌ | `useState`, seed dari `mock/seed.ts` | idem |
| `vendors` | ❌ | `useState`, seed dari `mock/seed.ts` | idem |
| `vendor-categories` | ❌ | `useState`, seed dari `mock/seed.ts` | idem |
| `users` (staff WO) | ❌ | `useState`, seed dari `mock/seed.ts` | idem |
| `client-portal` | ❌ (read-only) | selectors `mock/*`, scoping via `useCurrentClientProjectId()` | Perlu row-level scoping asli di backend |
| `subscription` | ✅ `useSubscriptionPlanStore` (shared) | `plan` dicari dari store; histori transaksi masih lokal | Sebagian sudah siap arsitekturnya |
| `platform-admin` | ✅ `usePlatformAdminStore` | store lengkap dgn actions CRUD | Paling siap — struktur action sudah dekat dgn bentuk service backend |

ID di 7 modul tanpa store dibuat lewat pola `let nextXId = 1000` per file — direset setiap reload,
bukan ID asli. Ini penanda "mock" yang konsisten dan mudah dilacak saat migrasi.

### 2.2 `mock/` adalah kontrak data implisit yang sudah ada

`apps/web/src/mock/{types.ts, seed.ts, selectors.ts}` adalah **model domain WO Console yang
sesungguhnya** — 218 baris interface, 574 baris seed, 399 baris fungsi query murni. Beberapa fungsi
di `selectors.ts` mengenkode **aturan bisnis nyata** yang harus dipindahkan ke backend sebagai
service/DB view, bukan ditulis ulang dari nol:

- `computeProjectProgress` / `computeMilestoneStats` → derivasi status "On Track / Attention / At
  Risk".
- `isPaymentEvidenceComplete` / `getMilestoneEvidenceCompleteness` → aturan: pembayaran dianggap
  lengkap jika ada `invoiceEvidenceId` **dan** `proofEvidenceId`, kecuali tipe `Refund` yang cukup
  `proofEvidenceId` saja.
- `getDashboardStats`, `getProjectTrend`, `getRevenueTrend` → agregasi dashboard.
- `TODAY = "2026-07-12"` dipakai sebagai pengganti waktu nyata di seluruh aplikasi (termasuk
  `modules/platform-admin/lib/trend.ts`) — backend butuh strategi jam/timezone asli untuk
  menggantikan ini.

### 2.3 Auth & routing — tidak ada penegakan apa pun

- Tidak ada `ProtectedRoute`/`RequireAuth`/`AuthGuard` di manapun (dikonfirmasi lewat pencarian
  menyeluruh — nol hasil). `AppLayout`, `PlatformLayout` mengizinkan akses langsung via URL tanpa
  cek apa pun. `ProjectDetailLayout`/`ClientPortalLayout` hanya redirect kalau **entitasnya** tidak
  ditemukan (`getProject(id) == null`) — itu validasi data, bukan otorisasi.
- `useAuthStore` menyimpan **tiga sesi sekaligus** (`currentStaffId`, `currentClientId`,
  `currentPlatformAdminId`) dengan default yang selalu valid (`st1`/`cl-p1-bride`/`pa1`) bahkan
  sebelum ada yang login. Tidak ada konsep "logged out".
- Password di `shared/constants/demo-accounts.ts` dan `platformAdminCredentials` disimpan
  **plaintext**; komentar di `usePlatformAdminStore.ts` sudah secara eksplisit menandai ini sebagai
  jalan pintas mock ("a real backend would hash it").
- Tidak ada token/JWT, tidak ada refresh, tidak ada invalidasi saat logout (logout = `navigate()`
  saja, store tidak direset).

### 2.4 Kapabilitas yang belum pernah dibangun sama sekali (bukan sekadar "belum disambung")

- **Upload & penyimpanan file**: `shared/lib/evidence-file.ts` selalu mengembalikan salah satu dari
  2 file demo statis (`foto-contoh.svg` / `dokumen-contoh.pdf`), apa pun record `Evidence`-nya. Ini
  kapabilitas baru total, bukan "tinggal disambung ke endpoint yang sudah ada UI-nya".
- **Multi-tenant data scoping**: nol modul WO Console punya `tenantId`. `subscription/SubscriptionPage.tsx`
  hardcode `CURRENT_PLAN_ID = "plan2"` dengan komentar eksplisit yang mengaitkannya ke satu tenant
  seed (`Anisa Wedding Organizer`) di `platform-admin/data/seed.ts`.
- **Activity log yang benar-benar mencatat**: `ActivityLogEntry` di mock hanya data seed statis —
  UI tidak pernah menulis entri baru ke sana (komponen section punya `sessionHistory` lokal
  sendiri-sendiri yang terpisah).
- **Audit trail untuk perubahan sensitif**: mis. "ganti perwakilan client" (`Family Representative`)
  meng-overwrite record yang sama tanpa menyimpan riwayat sebelumnya.

### 2.5 Dependency & tooling

- `apps/web/package.json`: **tidak ada** `@tanstack/react-query`, `swr`, `react-hook-form`, atau
  library data-fetching apa pun — semuanya harus ditambah baru kalau dipilih. `axios` sudah
  terpasang tapi cuma dipakai `shared/services/http-client.ts` yang **tidak diimpor di mana pun
  lagi** (nol referensi lain). `shared/services/api-endpoints.ts` cuma `{ base: "/api/v1" }`, tidak
  ada peta endpoint per resource.
- `apps/api`: `go.mod` (`module elproof`, go 1.22), `main.go` cuma `net/http` mux + 1 endpoint
  health-check, `sqlc.yaml` sudah mengarah ke `internal/modules` + `internal/database/sqlc`,
  1 migration (`users` generik, bukan skema ElProof), modul `order` adalah **nama placeholder
  generik yang tidak relevan** dengan domain ElProof dan belum pernah diganti.
- `docs/{PRD,SYSTEM_DESIGN,DB_SCHEMA,API_CONTRACT,DEPLOYMENT}.md` dan seluruh `knowledge/*.md`
  (termasuk 3 ADR) **masih placeholder `_TODO._`** — dikonfirmasi dengan membaca langsung tiap
  file, bukan asumsi.

---

## 3. Peta Modul: Frontend → Backend

Modul backend diusulkan mengikuti aturan modular monolith (`contracts/` saja yang publik, tanpa
join/FK lintas modul). `dashboard` dan `client-portal` **bukan modul backend tersendiri** — keduanya
murni lapisan komposisi/read yang memanggil kontrak modul lain.

| Modul backend (usulan) | Tabel utama yang dimiliki | Diturunkan dari frontend module(s) | Catatan boundary |
|---|---|---|---|
| `identity` (auth) | `credentials`, `sessions`/`refresh_tokens` | `auth`, + bagian login dari `platform-admin` | Satu skema JWT untuk 3 tipe principal: `staff`, `client`, `platform_admin` (klaim `principal_type`, `tenant_id` nullable, `role`) |
| `platform` | `tenants`, `platform_admins` | `platform-admin` (Tenant + PlatformAdmin) | Orkestrasi registrasi tenant memanggil kontrak `staff` (buat Owner) + `identity` (buat kredensial) |
| `billing` | `subscription_plans`, `subscription_transactions` | `subscription` (WO side) + bagian paket/transaksi dari `platform-admin` | Single source of truth katalog paket — dibaca oleh WO Console & Platform Console via kontrak yang sama |
| `staff` (users) | `staff_members` | `users` (WO staff) | Tenant-scoped; role Owner/Admin/Staff; baris Owner hanya bisa dibuat via orkestrasi `platform` |
| `clients` | `clients` | `clients` | Tenant + project scoped (FK primitif ke `projects.id`) |
| `vendors` | `vendors`, `vendor_categories` | `vendors`, `vendor-categories` | Digabung satu modul — relasi kategori↔vendor sepenuhnya internal |
| `projects` | `projects`, `project_milestones`, `project_vendors`, `vendor_milestones`, `vendor_payments`, `vendor_issues`, `evidence`, `activity_log` | `projects` (7 tab + semua sub-entity) | Modul terbesar; `vendorId`/`categoryId`/`picStaffId` disimpan sebagai ID primitif, diresolusi lewat kontrak `vendors`/`staff` |
| `shared` (bukan modul bisnis) | — | — | Utilitas teknis saja: file storage abstraction, response envelope, pagination, error, logger — **bukan** tempat Evidence/domain logic |

---

## 4. Keputusan Arsitektur yang Harus Dikunci Dulu

Dikerjakan di Fase 0 sebelum baris kode backend pertama ditulis:

1. **Strategi multi-tenant**: setiap tabel tenant-scoped (`staff_members`, `clients`, `vendors`,
   `vendor_categories`, `projects`, dan turunannya) mendapat kolom `tenant_id`; middleware auth
   menyuntikkan `tenant_id` dari klaim JWT, repository selalu filter berdasarkan itu.
2. **Model auth**: JWT access + refresh token, satu skema untuk 3 principal type, hash password
   dengan bcrypt, revocation list untuk refresh token saat logout.
3. **Strategi file/evidence storage**: ~~mulai dari disk lokal~~ — **direvisi (2026-07-16, ADR-0006):**
   IDCloudHost Object Storage (S3-compatible, via `minio-go/v7`), atas permintaan eksplisit user —
   di balik satu interface storage yang sama, jadi keputusan ini tetap tidak pernah membocor ke
   domain logic `projects`. Kompresi (JPEG lossy/PNG lossless, lihat ADR-0010) berjalan di kedua
   sisi (frontend Canvas API + backend Go), transfer HTTP-nya base64-in-JSON, bukan multipart.
4. **Waktu server**: hilangkan seluruh ketergantungan pada `TODAY` hardcode; backend jadi satu-satunya
   sumber waktu "sekarang" (dengan timezone eksplisit, mis. `Asia/Jakarta`).
5. **Penamaan ulang scaffold `apps/api`**: modul `order` (generik, tidak relevan) dibuang; modul
   `auth`/`users` di-scaffold ulang sesuai peta di §3.

---

## 5. Rencana Fase per Fase

> Setiap fase punya **Checkpoint** berupa checklist — centang saat kriteria benar-benar terverifikasi
> (bukan saat kode ditulis). Jangan mulai fase berikutnya kalau checkpoint fase sebelumnya belum
> tercentang semua, kecuali ada alasan eksplisit yang dicatat di sini.

### Fase 0 — Fondasi & Kontrak Tertulis
**Status:** 🟢 Selesai

Tujuan: mengubah pengetahuan implisit di frontend (§2–§3) menjadi dokumen tertulis, sebelum kode
backend ditulis, supaya `knowledge/*` dan `docs/*` benar-benar jadi sumber kebenaran (bukan
placeholder `_TODO._` selamanya).

Lingkup:
- Isi `docs/DB_SCHEMA.md` (tabel, kolom, kepemilikan modul, relasi sebagai ID primitif) berdasarkan
  peta §3 dan interface di `mock/types.ts` + `modules/*/data/types.ts`.
- Isi `docs/API_CONTRACT.md` (daftar endpoint per modul, format request/response sesuai
  `.claude/rules/api-standard.md`).
- Isi `docs/SYSTEM_DESIGN.md`, `docs/PRD.md` (ringkas, merujuk knowledge/ bukan duplikasi).
- Isi `knowledge/MODULE_MAP.md`, `ARCHITECTURE_OVERVIEW.md`, `DATABASE_GUIDE.md`, `API_GUIDE.md`,
  `DOMAIN_GLOSSARY.md`, `BACKEND_GUIDE.md`, `PRODUCT_REQUIREMENTS.md` (saat ini semuanya kosong).
- Tulis ADR baru untuk 5 keputusan di §4 (strategi tenant, auth, file storage, waktu server,
  restrukturisasi `apps/api`).
- Rapikan `apps/api/internal/modules/*`: hapus `order`, buat folder skeleton untuk `identity`,
  `platform`, `billing`, `staff`, `clients`, `vendors`, `projects` (masing-masing dengan
  `application/contracts/domain/infrastructure/presentation`).

Checkpoint:
- [x] `docs/DB_SCHEMA.md` mencantumkan seluruh tabel di §3 dengan kolom + relasi ID primitif
- [x] `docs/API_CONTRACT.md` mencantumkan endpoint untuk seluruh modul (`identity`, `platform`, `billing`, `staff`, `clients`, `vendors`, `projects`)
- [x] 6 ADR baru dibuat (0004–0009, termasuk keputusan Zustand-only) dan berstatus "Accepted"
- [x] Seluruh `knowledge/*.md` (`MODULE_MAP`, `ARCHITECTURE_OVERVIEW`, `DATABASE_GUIDE`, `API_GUIDE`, `DOMAIN_GLOSSARY`, `BACKEND_GUIDE`, `FRONTEND_GUIDE`, `PROJECT_BRIEF`, `PRODUCT_REQUIREMENTS`) sudah diisi, tidak ada lagi `_TODO._`
- [x] Struktur folder `apps/api/internal/modules/` cocok dengan peta §3 (`identity`, `platform`, `billing`, `staff`, `clients`, `vendors`, `projects`), `order` sudah dihapus, `auth`→`identity`, `users`→`staff`
- [x] `.env` lokal dibuat (JWT secret acak, `elproof_db` dibuat di MySQL host)

---

### Fase 1 — Infra Inti & Identity (Auth)
**Status:** 🟢 Selesai (backend + wiring frontend, terverifikasi end-to-end lewat Playwright)

Tujuan: aplikasi Go bisa connect ke MySQL, migration jalan, dan satu modul auth nyata berdiri —
ini fondasi yang dipakai semua fase berikutnya.

Lingkup:
- Router (bisa tetap `net/http` + `ServeMux` Go 1.22 seperti sekarang, atau `chi` — putuskan di
  Fase 0 kalau perlu; jangan tambah framework besar tanpa alasan).
- Config loader dari `.env` (`DB_*`, `JWT_SECRET`, `JWT_EXPIRES_IN` — sudah ada di `.env.example`).
- `shared/` teknis: response envelope (`{success, message, data}` / error), pagination helper
  (`meta: {page, limit, total, total_pages}`), logger, error mapper → HTTP status.
- Modul `identity`: tabel `credentials` (principal_type, principal_id, username, password_hash) +
  `refresh_tokens`; endpoint `POST /api/v1/auth/login`, `POST /api/v1/auth/refresh`,
  `POST /api/v1/auth/logout`; middleware `RequireAuth` yang mem-parse JWT dan menyuntik
  `tenant_id`/`principal_type`/`role` ke context request.
- Seeder migration untuk kredensial demo (menggantikan `DEMO_STAFF_ACCOUNTS` dkk di frontend) dengan
  password ter-hash.

Checkpoint:
- [x] `npm run dev:api` menghasilkan server yang connect ke MySQL host tanpa error (`elproof_db`, verifikasi manual via `go run ./cmd/server` — health check `200 OK`)
- [x] `POST /api/v1/auth/login` berhasil untuk ketiga principal type (staff/client/platform_admin) melawan data ter-seed (`anisa.putri`/`aurelia.wijaya`/`reza.hakim`, verifikasi curl)
- [x] Endpoint terproteksi (`/api/v1/auth/me`) mengembalikan 401 tanpa token, dan `tenant_id`/`role`/`principal_type` di klaim token sesuai baris yang login
- [x] Response semua endpoint mengikuti envelope `.claude/rules/api-standard.md`
- [x] Refresh token rotasi berfungsi (token lama otomatis revoked setelah dipakai; reuse ditolak); logout merevoke refresh token secara eksplisit
- [x] Password salah ditolak (`401`), password di-hash bcrypt di database (bukan plaintext)

**Checkpoint frontend (bagian dari Fase 1 yang sama):**
- [x] `shared/services/http-client.ts`: interceptor request menempelkan `Authorization: Bearer`, interceptor response mencoba refresh token sekali saat 401 lalu retry, logout otomatis kalau refresh gagal
- [x] `shared/services/api-endpoints.ts` diisi path nyata (`/api/v1/auth/{login,refresh,logout,me}`)
- [x] `shared/stores/useAuthStore.ts` menyimpan session asli (token, principalType, principalId, tenantId, role, displayName), persist ke `localStorage`, tetap Zustand murni (ADR-0009) — field lama (`currentStaffId` dkk.) dipertahankan sebagai derived value demi kompatibilitas mundur dengan puluhan komponen yang sudah ada, fallback ke ID demo lama saat belum ada sesi (logged-out sungguhan baru ditegakkan penuh di Fase 7 lewat route guard)
- [x] `LoginPage.tsx` memanggil API sungguhan untuk ketiga jenis akun, bukan lagi pencocokan plaintext lokal; tombol akun demo tetap ada sebagai quick-fill saja
- [x] Tombol "Keluar" di WO Console/Client Portal/Platform Console memanggil `logoutAndRedirect()` (shared/lib/auth-actions.ts) — clear session lokal + revoke refresh token di server
- [x] Diverifikasi end-to-end via Playwright: login 3 jenis akun, sesi bertahan setelah reload, logout menghapus sesi, password salah menampilkan pesan dari backend

**Catatan implementasi (batasan lingkungan, bukan penyimpangan desain):**
- `go.mod` diturunkan dari `go 1.22`→`go 1.21` untuk memakai toolchain yang sudah ter-install di
  lingkungan ini (percobaan mengunduh toolchain 1.22 gagal) — `infra/docker/Dockerfile` disamakan ke
  `golang:1.21-alpine` supaya image Docker konsisten dengan `go.mod`, bukan diam-diam beda versi.
  Konsekuensinya, routing pakai helper `shared/httpx.Method()` alih-alih sintaks `"POST /path"`
  bawaan `net/http.ServeMux` Go 1.22, supaya tetap portable ke Go 1.21+.
- Query `identity` module ditulis manual dengan `database/sql`, bukan sqlc — instalasi CLI `sqlc`
  gagal di lingkungan ini (`cc1.exe: sorry, unimplemented: 64-bit mode not compiled in`, toolchain
  CGO 32-bit lokal). Lihat `knowledge/DATABASE_GUIDE.md` § "Known local-environment limitation".
  Interface repository tetap sama sehingga bisa diganti ke sqlc-generated code kapan saja tanpa
  mengubah `application`/`domain`.
- CORS middleware (`shared/middleware/cors.go`) ditambahkan — tidak direncanakan eksplisit di draf
  awal PLAN.md ini, tapi diperlukan supaya `npm run dev:web` (Vite, port berbeda) bisa memanggil
  `npm run dev:api` (port 8080) saat development; middleware ini no-op di production karena satu
  container menyajikan keduanya dari origin yang sama.

---

### Fase 2 — Platform Console: `platform` + `billing`
**Status:** 🟢 Selesai (backend + frontend, terverifikasi end-to-end lewat Playwright)

Tujuan: modul yang paling dekat dengan bentuk akhirnya di frontend (`usePlatformAdminStore`,
`useSubscriptionPlanStore`) dipindahkan ke backend nyata terlebih dulu — risiko refactor
frontend paling rendah karena sudah ada store yang jelas.

Lingkup backend:
- `billing`: tabel `subscription_plans` (+ fitur sebagai kolom JSON atau tabel anak
  `plan_features`), `subscription_transactions` (status `unpaid|paid|expired|granted`).
- `platform`: tabel `tenants`, `platform_admins`; endpoint registrasi tenant yang mengorkestrasi
  pembuatan baris Owner di modul `staff` + kredensial di modul `identity` (via app service, bukan
  import lintas modul langsung); endpoint `activate-subscription` (bypass Tripay, status `granted`).

Lingkup frontend:
- Ganti `usePlatformAdminStore` dan `useSubscriptionPlanStore` dari state lokal jadi pemanggil
  `httpClient` sungguhan (isi `shared/services/api-endpoints.ts` dengan path nyata).
- 5 halaman Platform Console (`Dashboard`, `Tenant`, `Paket`, `Transaksi`, `Pengguna`) dan halaman
  `Langganan` WO Console disambungkan ke API ini.

Checkpoint:
- [x] CRUD tenant, paket, transaksi, platform admin berjalan lewat API sungguhan (bukan Zustand lokal)
- [x] Registrasi tenant baru benar-benar membuat baris Owner yang bisa login (via modul `staff` + `identity`) — diverifikasi login sebagai Owner tenant baru dgn `tenant_id`/`role` benar di JWT
- [x] Aktivasi manual paket menghasilkan transaksi berstatus `granted` dan tidak masuk hitungan "Pendapatan Terbayar"
- [x] Refresh halaman tidak lagi mereset data (state persisten di server, bukan seed in-memory)
- [x] Playwright regression: registrasi tenant, aktivasi manual, CRUD paket, CRUD admin platform, self-lockout, dan alur bayar mandiri Owner (`/subscriptions/pay`) semua lulus

**Endpoint tambahan di luar draf awal (ditambahkan karena benar-benar dibutuhkan, lihat
`docs/API_CONTRACT.md` untuk detail lengkap):**
- `GET /tenants/me` — self-service Owner untuk membaca tenant miliknya sendiri (halaman `Langganan`
  WO Console butuh ini; endpoint `/tenants/{id}` yang ada tetap platform_admin-only)
- `POST /subscriptions/pay` — self-service Owner untuk "Bayar Sekarang", orkestrasi yang sama
  dengan `activate-subscription` tapi status `paid` bukan `granted`; kedua endpoint ini sengaja
  ditaruh di modul `platform` (bukan `billing`) karena keduanya juga harus mengubah baris `tenants`
  yang dimiliki `platform`, bukan `billing`
- Toggle tunggal (`toggle-suspension`, `toggle-active`) dipakai alih-alih pasangan
  suspend/reactivate dan deactivate/reactivate terpisah — lebih sesuai dengan perilaku UI yang
  sudah ada (satu tombol yang membalik status)

**Catatan implementasi (bug ditemukan & diperbaiki lewat Playwright, bukan sekadar catatan gaya):**
- **Bug pemilihan paket default**: `TenantFormModal`/`ActivateSubscriptionModal` di-mount sekali di
  awal render `TenantListPage` (selalu ada di tree, cuma disembunyikan lewat prop `open`), sehingga
  `useState` default `planId` sempat membeku ke array `plans` yang masih kosong (belum selesai
  fetch). Diperbaiki dengan `useEffect` yang mengisi `planId` begitu daftar plan asli tersedia,
  tanpa menimpa pilihan user.
- **Bug tipe `planId`**: form mengirim `planId` sebagai string (id Zustand/HTML select selalu
  string), tapi Go men-decode ke `int64` — menyebabkan `json.Decode` gagal dengan pesan generik
  "Body permintaan tidak valid". Diperbaiki dengan `Number(values.planId)` di titik pemanggilan API
  pada store, konsisten dengan pola yang sudah dipakai di `activateTenantSubscription`.
- **Ketidakcocokan ID staff (Fase 1 vs Fase 2)**: `mock/seed.ts` masih memakai ID lama `"st1"`/`"st2"`/`"st3"`
  untuk Anisa/Bagas/Citra, sementara backend nyata (Fase 1) memberi ID numerik asli `"1"`/`"2"`/`"3"`.
  Setelah login sungguhan, `currentStaffId` di `useAuthStore` menjadi `"1"` tapi `mock/seed.ts` masih
  mencari `"st1"` — `getStaff()` mengembalikan `undefined`, membuat Sidebar/`SubscriptionPage` gagal
  mendeteksi Owner. Diperbaiki dengan menyelaraskan ID di `mock/seed.ts` (`"st1"→"1"`, dst., ~94
  referensi `picStaffId`/`actorStaffId` ikut berubah) serta `FALLBACK_STAFF_ID` di `useAuthStore.ts`.
  Ini murni penyelarasan sementara — `mock/seed.ts` akan digantikan sepenuhnya oleh store `staff`
  asli di Fase 3, saat itu ID ini tidak relevan lagi.
- `usePlatformAdminStore.verifyPlatformAdminLogin` (fungsi lama, dipakai `LoginPage` sebelum Fase 1)
  dihapus karena sudah jadi dead code — auth sekarang selalu lewat `identity` module.

---

### Fase 3 — WO Console Inti: `staff` (CRUD penuh) + `vendors`/`vendor-categories`
**Status:** 🟢 Selesai (backend + frontend, terverifikasi end-to-end lewat curl + Playwright)

**Penyesuaian cakupan dari draf awal:** `clients` **dipindahkan ke Fase 4**, digabung dengan
`projects`. Alasannya ditemukan saat mulai implementasi: `clients.project_id` adalah FK primitif
ke `projects.projects.id`, dan frontend (`ClientListPage.tsx`) memang **tidak punya alur "buat
client baru"** sama sekali — client selalu dibuat implisit saat project dibuat (fitur di modul
`projects`, Fase 4). Tanpa tabel `projects` nyata, tidak ada cara membuat baris `clients` yang sah
(tidak ada `project_id` valid untuk direferensikan). Membangun `clients` sekarang berarti
membangunnya dengan data uji buatan yang tidak representatif, lalu membangunnya ulang lagi di Fase
4 begitu `projects` ada. `staff` dan `vendors`/`vendor-categories` tidak punya masalah ini — keduanya
murni referensi tenant-scoped, tanpa dependensi ke modul yang belum ada.

Lingkup backend (keduanya diverifikasi lewat curl — lihat catatan implementasi):
- `staff`: diperluas dari Fase 2 (yang hanya punya `CreateOwner`) ke CRUD penuh — `List`, `Create`
  (menolak `role=Owner`), `Update` (menolak mengubah ke/dari Owner), `SetActive` (menolak
  menonaktifkan Owner). Semua tenant-scoped via klaim JWT (`middleware.Claims.TenantIDInt()`,
  helper baru).
- `vendors`: modul baru, memiliki tabel `vendor_categories` + `vendors`. CRUD penuh untuk
  keduanya; `vendors.categoryId` divalidasi terhadap `vendor_categories` tenant yang sama saat
  create/update (menolak kategori yang tidak ada atau milik tenant lain).

Lingkup frontend (bagian yang lebih besar dari kelihatannya, sesuai dugaan draf awal):
- 3 store Zustand baru: `modules/users/stores/useStaffStore.ts`,
  `modules/vendor-categories/stores/useVendorCategoryStore.ts`,
  `modules/vendors/stores/useVendorStore.ts` — mengikuti pola persis `usePlatformAdminStore`
  (fetch-then-set, tanpa cache otomatis).
- `UserListPage`, `VendorCategoryListPage`, `VendorListPage` diganti dari `useState(() => seedX)` +
  `let nextXId` jadi pemanggilan store di atas.
- **Keputusan §6 soal React Query vs Zustand sudah diputuskan (ADR-0009) sebelum fase ini mulai** —
  tetap Zustand manual, tidak ada penyimpangan.
- Tipe `StaffMember`/`Vendor`/`VendorCategory` **tetap diimpor dari `@/mock`** (bukan didefinisikan
  ulang) — modul `projects` (masih 100% mock, Fase 4) memakai array mock yang sama untuk PIC/vendor
  picker-nya, jadi ini murni pemakaian ulang shape TypeScript, bukan berbagi data. Konsekuensinya:
  data staff/vendor yang tampil di halaman Pengguna/Vendor (real, dari API) akan **berbeda** dari
  yang muncul di dropdown pemilihan PIC/vendor di dalam Project (masih data mock lama) — perbedaan
  ini disengaja dan akan hilang begitu Fase 4 memigrasikan `projects`.

Checkpoint:
- [x] Data staff/vendor/kategori bertahan setelah reload halaman (dibaca dari server, bukan seed di memori)
- [x] Baris Owner tidak bisa dibuat/diubah/dinonaktifkan dari halaman Pengguna WO Console (ditolak backend 422/403, bukan cuma disembunyikan UI)
- [x] Akses langsung ke `/staff`, `/vendors`, `/vendor-categories` tanpa token ditolak 401; token `platform_admin` (tanpa `tenant_id`) ditolak 403 — diverifikasi lewat curl langsung, bukan cuma UI
- [x] Pola store/fetch di fase ini konsisten dengan `knowledge/FRONTEND_GUIDE.md` (sudah ditulis di Fase 0, dikonfirmasi masih akurat di fase ini — tidak ada pola baru yang menyimpang)

**Catatan implementasi:**
- Bug ditemukan lewat Playwright yang sama seperti Fase 2: `VendorFormModal` punya default
  `categoryId` yang bisa membeku ke daftar kosong (mount sekali, `categories` datang async) —
  diperbaiki dengan pola `useEffect` backfill yang sama seperti `TenantFormModal` di Fase 2, kali
  ini sekaligus dijadikan prop (`categories: VendorCategory[]`) alih-alih impor array mock langsung.
- `docs/DB_SCHEMA.md`/`API_CONTRACT.md` diperbarui menambahkan `staff` dan `vendors` yang sekarang
  benar-benar terimplementasi (lihat bagian masing-masing).

---

### Fase 4 — Projects (Modul Terbesar) + `clients`
**Status:** 🟢 Selesai (backend + frontend, terverifikasi end-to-end lewat curl + Playwright, termasuk upload evidence ke bucket IDCloudHost sungguhan)

Tujuan: modul inti bisnis — `projects` beserta 6 sub-entity, upload file sungguhan, **dan**
`clients` (dipindahkan dari Fase 3 — lihat catatan penyesuaian cakupan di Fase 3 — karena baris
`clients` hanya bisa dibuat secara sah begitu ada `project_id` nyata untuk direferensikan; alur
"buat client" di frontend memang selalu implisit lewat pembuatan project, tidak ada form
"tambah client" berdiri sendiri).

Lingkup backend:
- Tabel: `projects`, `project_milestones`, `project_vendors`, `vendor_milestones`,
  `vendor_payments`, `vendor_issues`, `evidence`, `activity_log` — semua tenant-scoped via
  `projects.tenant_id`, turunannya via FK ke `project_id`.
- `clients`: tabel `clients` (FK primitif `project_id`), dibuat sebagai bagian dari orkestrasi
  pembuatan project (bukan endpoint create client berdiri sendiri, mengikuti perilaku frontend yang
  sudah ada); endpoint CRUD kontak, reset kredensial, ganti perwakilan (putuskan apakah butuh tabel
  histori — lihat §6).
- Endpoint upload untuk `evidence`: JSON + base64 (bukan multipart — lihat ADR-0010), dengan
  kompresi backend via `internal/shared/compress` dan penyimpanan ke IDCloudHost Object Storage via
  `internal/shared/storage` — **kedua utilitas ini sudah dibangun & diverifikasi lebih dulu**
  (lihat "Groundwork" di bawah), tinggal dipakai oleh handler `evidence` saat modul ini digarap.
- Service layer memindahkan logic dari `mock/selectors.ts`: `computeProjectProgress`,
  `computeMilestoneStats`, `isPaymentEvidenceComplete` (termasuk pengecualian tipe `Refund`),
  `getMilestoneEvidenceCompleteness`.
- `activity_log` benar-benar di-append oleh backend pada setiap mutasi (create project, ubah status
  milestone, catat pembayaran, buka/tutup issue, upload evidence) — bukan hanya seed statis.

Lingkup frontend:
- Store baru untuk `projects` (dan sub-entitasnya) menggantikan seluruh `useState` lokal di
  `ProjectListPage` dan 7 tab detail.
- Normalisasi 3 skema inline (`payment`, `issue`, `evidence` — saat ini didefinisikan ad-hoc di
  komponen section, bukan di folder `schemas/`) jadi file `schemas/*.schema.ts` yang konsisten
  dengan modul lain.
- Komponen upload evidence sungguhan menggantikan `getEvidenceFileUrl()` yang selama ini selalu
  mengembalikan 2 file demo statis. Sebelum dikirim, file gambar dikompres via
  `shared/lib/image-compression.ts` (Canvas API, tanpa dependency npm baru) lalu di-base64.

**Groundwork sudah selesai duluan (2026-07-16), sebelum sisa modul `projects` digarap** — lihat
ADR-0006 (revisi) dan ADR-0010:
- [x] Object storage provider diputuskan: IDCloudHost Object Storage (S3-compatible), kredensial di
  `.env` (`S3_ENDPOINT`/`S3_BUCKET`/`S3_ACCESS_KEY`/`S3_SECRET_KEY`/`S3_USE_SSL`), tidak pernah
  di-hardcode atau di-log.
- [x] `internal/shared/storage` (klien S3 via `minio-go/v7`) dibangun — `Save`/`Open`/`Delete`/
  `BuildKey` (`elproof/upload/{tenantId}/{projectId}/{category}/{filename}`).
- [x] `internal/shared/compress` (JPEG kualitas 82 lossy, PNG lossless, downscale maks 2000px sisi
  terpanjang, format lain dilewatkan apa adanya) dibangun.
- [x] `shared/lib/image-compression.ts` (frontend, Canvas API, aturan identik dengan backend)
  dibangun.
- [x] **Diverifikasi langsung ke bucket IDCloudHost sungguhan** (bukan mock/simulasi): upload →
  download → cek integritas byte-per-byte → re-decode sebagai gambar valid → hapus objek uji.
  Semua tahap berhasil. Objek uji sudah dibersihkan dari bucket.
- [x] `go.mod` naik ke `go 1.25.0` (otomatis, dependency `minio-go/v7` mensyaratkan ≥1.25) —
  `infra/docker/Dockerfile` disamakan ke `golang:1.25-alpine`. Percobaan sebelumnya mengunduh
  toolchain 1.22 gagal (lihat catatan Fase 1), tapi 1.25 berhasil diunduh otomatis — jadi batasan
  itu spesifik ke versi 1.22, bukan blokir total pada mekanisme auto-download toolchain.

Checkpoint:
- [x] Alur lengkap project → milestone → vendor engagement → payment → issue → evidence berjalan end-to-end lewat API sungguhan (curl) **dan** lewat UI sungguhan (Playwright, browser nyata)
- [x] Upload evidence menyimpan & menampilkan file asli (bukan `foto-contoh.svg`/`dokumen-contoh.pdf`), tersimpan di IDCloudHost dengan path sesuai konvensi ADR-0006; viewer mengambil file lewat `httpClient` terautentikasi (blob + object URL), bukan `<img src>` langsung, karena endpoint download ada di belakang Bearer token
- [x] `activity_log` terisi otomatis dari mutasi nyata, terlihat di tab Aktivitas (diverifikasi 6 jenis entri: project_created, vendor_added, milestone_updated ×2, payment_recorded, issue_created, issue_updated, evidence_uploaded)
- [x] Progress project & status milestone dihitung backend, angka cocok dengan hasil `computeProjectProgress` versi mock sebagai baseline regresi (diverifikasi manual saat curl-test: 1 milestone project selesai + 1 milestone vendor belum → 50%, kondisi "Attention" karena evidence pembayaran belum lengkap — sama seperti mock)
- [x] Client baru bisa dibuat dari tab Client project (form baru, bukan cuma implisit dari pembuatan project — lihat catatan implementasi) dan bisa langsung login ke Client Portal dengan kredensial yang baru dibuat

**Catatan implementasi (backend):**
- Cakupan backend (migrasi, domain, application, infrastructure, presentation, wiring `main.go`,
  `cmd/seed`) dibangun dan diverifikasi penuh lewat skrip curl end-to-end sebelum frontend disentuh
  sama sekali — lihat ringkasan di bagian "Groundwork" dan checklist di atas.
- `net/http.ServeMux` tidak bisa mendaftarkan pattern prefix yang tumpang-tindih, jadi seluruh
  subtree `/api/v1/projects/` ditangani oleh satu method dispatcher (`Handler.Item`) yang mem-parse
  segment path dan switch ke ~20 method handler privat tersebar di beberapa file dalam package
  `presentation` yang sama — pola yang sama dipakai `clients`.
- `ReplaceRepresentative` (ganti Wedding Representative) menimpa baris client yang sama tanpa
  riwayat — ini keputusan eksplisit (bukan lupa), lihat §6.3.

**Catatan implementasi (frontend — bagian terbesar dari fase ini):**
- Store baru: `modules/projects/stores/useProjectStore.ts` (satu store untuk project + 7
  sub-entitas: milestone, vendor engagement, vendor milestone, payment, issue, evidence, activity —
  semua fetch-then-set per slice, tidak ada namespacing per-project di dalam store, jadi setiap tab
  selalu fetch ulang saat projectId berubah) dan `modules/clients/stores/useClientStore.ts`
  (di-cache per `projectId` karena endpoint list client memang di-scope ke satu project).
- `picStaffId`/`vendorId` picker di form `projects` sekarang memakai store `staff`/`vendors` **asli**
  dari Fase 3 (`useStaffStore`/`useVendorStore`), bukan lagi array `@/mock` seperti catatan
  penyimpangan yang didokumentasikan di Fase 3 — penyimpangan itu sudah hilang seperti yang
  diperkirakan.
- Endpoint `GET /projects` (list) sengaja tidak menyertakan `progress` (baru dihitung di `GET
  /projects/{id}`) — `ProjectListPage`/`ProjectCard` melakukan fetch progress per-project setelah
  fetch list (N+1, diterima untuk skala data WO saat ini) agar kartu project tetap menampilkan
  persentase & kondisi tanpa endpoint agregat baru.
- Fitur reorder milestone (naik/turunkan urutan) dihapus dari UI — backend tidak punya endpoint
  untuk mengubah `sort_order` di luar waktu pembuatan (`NextSortOrder` auto-increment saja), jadi
  tombol panah yang ada di mock dihapus alih-alih dipalsukan di frontend saja.
- Bug ditemukan lewat Playwright (baru, tidak ada di Fase 2/3, karena baru kali ini form beririsan
  dengan field yang tak terlihat di UI atau selector Zustand yang tidak stabil):
  1. **`POST /projects/{id}/vendors` mengembalikan 422** — form tambah vendor tidak punya field
     tanggal acara, tapi backend mewajibkan `eventDate` valid saat create. Diperbaiki dengan mengisi
     `eventDate` otomatis dari `currentProject.eventDate` di `useProjectStore.createVendorEngagement`
     (masuk akal secara domain: tanggal butuh vendor = tanggal acara project).
  2. **`categoryId` project-vendor tersimpan `0`** — komentar awal saya di kode ("server menurunkan
     categoryId dari vendor") salah; backend menyimpan apa pun yang dikirim body. Diperbaiki dengan
     mengambil `categoryId` asli dari vendor terpilih (`useVendorStore`) sebelum POST/PATCH.
  3. **`POST /projects/{id}/payments` dan `.../issues` mengembalikan 400** — `projectVendorId` yang
     dikirim masih `string` (dari form), sementara body JSON backend mengharapkan angka; Go gagal
     decode lalu balas "Body permintaan tidak valid". Diperbaiki dengan `Number(values.projectVendorId)`
     di kedua action store.
  4. **Modal tambah pembayaran/kendala/evidence kadang gagal submit tanpa pesan error** — modal-modal
     ini dulu selalu ter-mount (hanya `Modal` di dalamnya yang cek `open`), dan `useState` default
     untuk pilihan vendor/relatedId dihitung sekali saat mount pertama — kalau data vendor/milestone
     belum sempat ke-fetch di titik itu, defaultnya kosong selamanya (racy, tidak reaktif). Diperbaiki
     dengan me-mount modal-modal ini secara kondisional (`{modalOpen && <Modal .../>}`) supaya state
     awal selalu dihitung ulang tepat saat user membuka modal, bukan saat parent section mount.
  5. **Navigasi ke tab Client meng-crash seluruh halaman** (React error #185, "Maximum update depth
     exceeded") — selector Zustand `(s) => s.clientsByProject[projectId] ?? []` mengembalikan array
     `[]` baru di setiap pemanggilan ketika key belum ada, yang merusak pengecekan referensi
     `useSyncExternalStore` dan memicu render tanpa henti. Diperbaiki dengan konstanta `EMPTY_CLIENTS`
     modul-level sebagai fallback yang stabil — kelas bug ini diperiksa ulang di seluruh frontend
     (`grep` pola `?? []` di dalam pemanggilan `useStore`) untuk memastikan tidak ada instance lain.
  6. **`apps/web` tidak pernah bisa mencapai backend lewat `npm run dev:web`** — `VITE_API_BASE_URL`
     hanya ada di `.env` root, tapi Vite hanya membaca `.env` dari root project-nya sendiri
     (`apps/web`), bukan root monorepo, dan `vite.config.ts` tidak mengatur `envDir`. Ini bukan bug
     baru dari fase ini, tapi baru ketahuan sekarang karena baru kali ini frontend benar-benar
     memanggil backend saat dites lewat `npm run dev:web` murni (fase-fase sebelumnya kemungkinan
     dites dengan env var di-export manual di shell). Diperbaiki permanen dengan menambah
     `apps/web/.env` + `.env.example` (`VITE_API_BASE_URL=http://localhost:8080`).
- Evidence lama tampil lewat file demo statis (`foto-contoh.svg`/`dokumen-contoh.pdf`) yang sama
  untuk semua evidence — sekarang `EvidenceViewerModal` mengambil file evidence **sungguhan** lewat
  `httpClient` (dengan Bearer token) sebagai blob lalu menampilkannya via object URL sementara
  (di-revoke saat modal ditutup), karena endpoint download ada di belakang autentikasi staff dan
  `<img src>`/`<a href>` polos tidak bisa membawa header itu. Jenis file (gambar vs PDF vs lainnya)
  ditentukan dari ekstensi nama file asli, bukan dari `EvidenceType` (satu jenis evidence seperti
  "Transfer Proof" bisa saja berupa PDF atau foto).
- Client sekarang bisa dibuat dari tab Client project lewat form baru (role + kontak + password),
  bukan hanya lewat seed manual — form "reset kredensial" juga diperbarui untuk meminta password
  baru (bukan cuma menekan tombol), karena endpoint reset backend butuh password baru, bukan sekadar
  menimpa `lastCredentialResetAt`.
- Diverifikasi lewat Playwright end-to-end (browser sungguhan, bukan curl): login staff → buat
  project → tambah vendor engagement → tambah milestone vendor → tambah milestone project → tandai
  selesai → catat pembayaran → catat & selesaikan kendala → upload evidence sungguhan ke bucket
  IDCloudHost → lihat evidence lewat viewer (gambar tampil, tanpa error) → cek tab Aktivitas → cek
  progress di header → buat client baru → logout → login sebagai client baru, berhasil. Nol error
  console browser di sepanjang alur. Data uji (≈14 project percobaan, 6 evidence sungguhan di
  bucket, 1 vendor/kategori uji) dibersihkan setelahnya: objek bucket dihapus lewat command sekali
  pakai, database di-reset lewat `cmd/seed` (aman diulang, sesuai desainnya).

---

### Fase 5 — Dashboard & Agregasi
**Status:** 🟢 Selesai (backend + frontend, terverifikasi lewat curl + Playwright)

Tujuan: endpoint read-only untuk kedua dashboard (WO Console & Platform Console), menggantikan
agregasi client-side.

Lingkup yang benar-benar dikerjakan (lihat catatan implementasi untuk penyesuaian dari draf awal):
- Endpoint agregasi WO dashboard baru: `GET /api/v1/dashboard`, dibangun di dalam modul `projects`
  (bukan modul terpisah — lihat `knowledge/MODULE_MAP.md`, semua tabel yang dibutuhkan sudah dimiliki
  modul ini). Logic dipindah dari `mock/selectors.ts` (`getDashboardStats`, `getProjectTrend`,
  `getRevenueTrend`, `getCurrentMonthRevenue`, `getUpcomingProjects`, `getRecentActivity`) ke
  `application/dashboard_service.go` + `domain/dashboard.go`, dengan jam server asli (`time.Now()`)
  sebagai referensi "hari ini" — bukan `TODAY` hardcode.
- Platform dashboard **tidak dapat endpoint baru** — datanya (tenant + transaksi) sudah 100% real
  sejak Fase 2 (`usePlatformAdminStore`), jadi satu-satunya gap adalah referensi tanggal `TODAY` yang
  masih diimpor dari `@/mock` di `modules/platform-admin/lib/trend.ts` dan `data/selectors.ts`.
  Diperbaiki dengan `todayISO()` (jam nyata) — bukan endpoint agregasi baru, karena data yang
  dihitung sudah sepenuhnya ada di client (menambah endpoint di sini hanya memindahkan komputasi
  yang sudah benar tanpa mengubah hasilnya).
- Pencarian global (`GlobalSearch.tsx`) tidak mendapat endpoint `/search` baru — cukup dipindah untuk
  memanggil 3 endpoint list yang sudah ada (`GET /projects`, `GET /vendors`, `GET /clients`) secara
  paralel lalu filter substring di client seperti sebelumnya, sama seperti sebelumnya secara UX tapi
  di atas data sungguhan. `GET /clients` diperluas: jika query param `projectId` tidak diisi, sekarang
  mengembalikan seluruh client tenant (bukan wajib 400) — perubahan backward-compatible kecil yang
  dibutuhkan karena pencarian perlu mencari lintas semua project sekaligus, bukan satu per satu.

Checkpoint:
- [x] Kedua dashboard menampilkan angka yang dihitung dari data sungguhan (bukan mock), diverifikasi manual lewat curl (progress/lagging/revenue/trend cocok dengan perhitungan manual atas data uji)
- [x] Toggle "Bulan Ini"/"Tahun Ini" (Platform dashboard) memakai tanggal server asli, bukan konstanta `TODAY` yang di-hardcode
- [x] Pencarian global memanggil endpoint backend (3 store fetch), bukan iterasi array `@/mock` di memori browser

**Catatan implementasi:**
- `application/dashboard_service.go`'s `DashboardService` bergantung langsung pada `*ProjectService`
  (bukan cuma repository interface) untuk menghitung `laggingProjects` — ini aman karena keduanya ada
  di package `application` yang sama (satu modul), bukan pelanggaran boundary lintas-modul.
- 3 query SQL baru (`ListOpenIssues`, `ListOverdueVendorMilestones`, `ListPaymentCandidates`) di
  `infrastructure/mysql_dashboard_repository.go` melakukan JOIN ke `projects`/`project_vendors` untuk
  scoping tenant dan resolusi `vendorId` — semua tabel yang di-JOIN dimiliki modul `projects` sendiri,
  jadi tidak melanggar aturan "no cross-module join". Nama vendor/staff **tidak** diresolusi di
  backend (hanya `vendorId` primitif dikirim) — frontend me-resolve nama lewat `useVendorStore` yang
  sudah ter-fetch, konsisten dengan pola yang sudah dipakai di seluruh modul `projects` sejak Fase 4.
- Bug ditemukan lewat Playwright: response `GET /dashboard` mengembalikan `null` (bukan `[]`) untuk
  setiap array yang kosong (perilaku default Go untuk slice nil yang di-marshal ke JSON), sementara
  konvensi yang sudah dipakai di seluruh modul lain (`make([]X, 0, len(list))`) sengaja menghindari
  ini. Frontend crash (`Cannot read properties of null (reading 'map')`) karena kode konversi memakai
  `raw.field.map(...)` tanpa null-guard. Diperbaiki di sisi backend (bukan menambah null-check di
  frontend) — `toDashboardResponse` sekarang mem-pre-allocate setiap slice dengan `make(..., 0, ...)`,
  sesuai konvensi yang sudah ada, supaya kontrak API tetap konsisten "array kosong, bukan null" di
  seluruh endpoint.
- `activityResponse` (dipakai baik oleh endpoint aktivitas per-project maupun dashboard) mendapat
  field baru `projectId` — endpoint per-project sebelumnya tidak butuh field ini (implisit dari URL),
  tapi feed aktivitas lintas-project di dashboard butuh tahu project mana yang dituju setiap entri.
- `shared/lib/formatters.ts` mendapat `todayISO()` sebagai satu-satunya sumber "hari ini" di seluruh
  frontend — `modules/projects/lib/dates.ts` sekarang re-export dari sini alih-alih mendefinisikan
  ulang, supaya `platform-admin` (modul lain) tidak perlu impor lintas-modul frontend ke `projects`.

---

### Fase 6 — Client Portal
**Status:** 🟢 Selesai (backend + frontend, terverifikasi lewat curl + Playwright, termasuk percobaan akses lintas-client langsung ke API)

Tujuan: memastikan client hanya bisa melihat project miliknya sendiri — ini satu-satunya tempat
di aplikasi yang butuh row-level scoping berbasis identitas client, bukan cuma tenant.

Lingkup yang benar-benar dikerjakan:
- `/api/v1/projects/{id}/...` (subtree yang sama persis dipakai staff, bukan endpoint terpisah)
  sekarang menerima prinsipal `client` juga: dibatasi hanya `GET` (percobaan `POST`/`PATCH`
  langsung ditolak 403 sebelum menyentuh logic apa pun), dan `{id}` wajib sama dengan satu-satunya
  project milik client tersebut — diverifikasi lewat pemanggilan baru `clients.Contracts`
  (`ProjectIDForClient`), bukan konvensi frontend seperti `useCurrentClientProjectId()` yang lama.
- `GET /api/v1/projects/me` (baru) — titik masuk Client Portal, mengikuti pola `GET /tenants/me`
  dari Fase 2. Client login tidak pernah menerima `projectId` di response-nya (dan `identity`
  sengaja profile-agnostic, ADR-0005, jadi tidak akan pernah menyimpannya), sehingga frontend genap
  tidak punya cara lain untuk tahu project miliknya sendiri kecuali lewat endpoint ini.

Checkpoint:
- [x] Login sebagai client demo A tidak bisa melihat data project client demo B lewat manipulasi URL/ID langsung ke API (diuji lewat `fetch` manual dari sesi browser client A ke `/projects/{id milik B}`)
- [x] Percobaan akses project lain menghasilkan 403 dari backend, bukan hanya redirect halaman di frontend
- [x] Percobaan client melakukan `POST`/`PATCH` ke project miliknya sendiri juga ditolak 403 (client selalu read-only, bahkan untuk data miliknya sendiri)

**Catatan implementasi:**
- Masalah wiring melingkar: `clients.NewModule` butuh `projects.Contracts()` (validasi `project_id`
  saat create — sudah ada sejak Fase 4), tapi `projects` sekarang juga butuh sesuatu dari `clients`
  (resolusi "project mana milik client ini"). Constructor kedua arah sekaligus tidak mungkin (urutan
  pembangunan modul di `main.go` jadi saling menunggu). Diselesaikan dengan **interface lokal** —
  `projects/presentation` mendefinisikan `ClientAccessResolver` miliknya sendiri (bentuknya persis
  cocok dengan satu method di `clients.Contracts`), sehingga `projects` **tidak pernah** meng-`import`
  package `clients` sama sekali. `main.go` (composition root, sudah meng-`import` keduanya) menjembatani
  lewat `projectsModule.SetClientAccessResolver(clientsModule.Contracts())` — dipanggil setelah
  KEDUA modul selesai dibangun. Ini pola dependency-inversion standar Go untuk memutus siklus
  konstruksi modular monolith, bukan pelanggaran terhadap "no cross-module import".
- `clients` module mendapat package `contracts` untuk pertama kalinya (sebelumnya hanya
  mengonsumsi, tidak pernah mengekspos) — `ProjectIDForClient(tenantID, clientID) (projectID, error)`,
  juga menolak (403) jika client sudah dinonaktifkan.
- Route `/projects/me` **tidak** didaftarkan sebagai `mux.Handle` terpisah — mengikuti pola
  `tenants/me` yang sudah ada persis: segment `"me"` dicek secara eksplisit di awal dispatcher
  `Item` sebelum segment lain di-parse sebagai ID numerik, supaya satu subtree tetap punya satu
  titik masuk dispatcher, konsisten dengan pola yang sudah dipakai di seluruh modul ini.
- Endpoint GET yang dipakai bersama staff & client (milestones, vendor engagements + milestone-nya,
  payments, issues, evidence + file download, activity) **tidak diduplikasi** — satu switch
  dispatcher yang sama dipakai kedua prinsipal, karena tak satu pun cabang `GET` di dalamnya
  membaca `staffID`/`role` (hanya `tenantID`, yang bisa diisi baik dari klaim staff maupun client).
  Cabang tulis (`POST`/`PATCH`) otomatis tidak pernah tercapai client karena sudah ditolak lebih
  awal di `resolveProjectAccess`.
- Frontend: `ClientPortalLayout` sekarang memanggil `fetchMyProject()` (bukan `getClient()`/`getProject()`
  dari `@/mock`) untuk resolusi project + render header; ke-4 tab (`Ringkasan`, `Vendor & Progress`,
  `Pembayaran`, `Kendala`) dan `IssueCard` dipindah dari selector mock ke `useProjectStore`/
  `useVendorStore`/`useVendorCategoryStore`, memakai pola `useOutletContext` yang sama dengan
  `ProjectDetailLayout` di WO Console. Hook lama `useCurrentClientProjectId()` (mock-based) dihapus.
- Diverifikasi lewat Playwright: login sebagai 2 client berbeda (masing-masing pemilik project
  berbeda) di sesi browser terpisah, memastikan masing-masing hanya melihat data project miliknya
  sendiri di ke-4 tab, **dan** memanggil `fetch()` manual dari console sesi client pertama langsung
  ke API untuk project client kedua — mengonfirmasi backend menolak 403 meski request datang
  langsung ke API (bukan cuma dicegah oleh UI/routing frontend).

---

### Fase 7 — Pengerasan Lintas-Modul (Hardening)
**Status:** 🟢 Selesai (backend + frontend, terverifikasi lewat curl + Playwright)

Tujuan: menutup gap keamanan/kualitas yang sengaja ditunda dari fase-fase sebelumnya supaya tidak
memperlambat setiap fase individual.

Lingkup yang benar-benar dikerjakan:
- `RequireAuth` (baru, `shared/components/RequireAuth.tsx`) dipasang sebagai wrapper route tanpa
  parent di ketiga pohon route (`protected.routes.tsx` untuk WO Console, `platform.routes.tsx`
  untuk Platform Console, `client-portal.routes.tsx` untuk Client Portal), masing-masing dibatasi
  ke `principalType` yang sesuai lewat prop `allow`. Layout yang sudah ada (`AppLayout`,
  `PlatformLayout`, `ClientPortalLayout`) jadi child dari `RequireAuth`, tidak diubah sama sekali.
- Server-side pagination penuh (opsi paling menyeluruh yang dipilih user lewat `AskUserQuestion` —
  bukan skip atau partial) untuk **9 endpoint list** lintas semua modul: `staff`, `vendors`,
  `vendor-categories`, `clients` (khusus list tenant-wide, bukan `ListByProject`), `projects`,
  `tenants`, `platform-admins`, `plans`, `subscription-transactions`. Setiap endpoint kini:
  - default ke respons terpaginasi (`response.OKPaginated` + `pagination.Meta{page, limit, total,
    total_pages}`), dibangun dari `pagination.FromRequest(r)` + `ListPaginated` baru di tiap
    repository (COUNT query + `LIMIT`/`OFFSET`), dan
  - tetap menyediakan array penuh lewat `?all=true` (`response.OK`, tanpa `meta`) — dipakai oleh
    setiap dropdown/picker/global-search/dashboard-aggregation yang butuh seluruh data sekaligus,
    tanpa mengubah kontrak lama mereka sama sekali.
  - Filter facet tambahan (`search`, dan per-modul: `role` untuk staff & platform-admins,
    `categoryId` untuk vendors, `status` untuk tenants/projects/transactions) ikut diterapkan di
    level SQL, bukan cuma di query pertama lalu difilter di memori — supaya kombinasi
    "cari + filter + halaman N" tetap benar, bukan sekadar filter atas hasil halaman yang sudah
    terpotong.
- Frontend: setiap store yang sebelumnya cuma punya satu field array + satu `fetch()` sekarang
  punya **field/actions kedua** yang paralel (`xxxPage` + `xxxPageMeta` + `fetchXxxPage(page, ...)`),
  sementara field/aksi lama (`xxx` + `fetchXxx()`) tetap ada apa adanya — cuma ditambah
  `{ params: { all: true } }` di panggilan `httpClient.get`-nya supaya perilakunya sama persis
  seperti sebelum ada pagination. 8 halaman list dipindah dari `usePagination` (client-side slicing)
  ke pola baru ini: `UserListPage`, `VendorListPage`, `VendorCategoryListPage`, `TenantListPage`,
  `PlatformUserListPage`, `PlatformTransactionsPage`, `SubscriptionPage` (tabel riwayat transaksi),
  `ProjectListPage` (sebelumnya malah *tidak* punya pagination sama sekali — sekarang dapat).
  `ClientListPage` **sengaja tidak diubah**: halaman itu mengelompokkan client per-project lalu
  memaginasi kelompok itu sendiri di client-side, tidak pernah memanggil endpoint list tenant-wide
  yang dipaginasi backend — jadi tidak ada yang perlu (atau bisa) dikonversi di sana.
- Kredensial demo dihapus total dari source frontend: `demo-accounts.ts` sekarang cuma berisi
  `{ username, label }` (tanpa `password`, dan tanpa `staffId`/`clientId` yang ternyata sudah dead
  code sejak awal). Login demo tetap ada, tapi memanggil endpoint backend baru,
  `POST /api/v1/auth/demo-login`, yang menerbitkan sesi asli untuk sebuah username **tanpa
  mengecek password sama sekali** — endpoint ini hanya didaftarkan ke mux sama sekali kalau
  `APP_ENV=development` (lihat `identity.module.go`), jadi di luar development endpoint itu
  benar-benar tidak ada, bukan cuma disembunyikan. Panel "Akun Demo" di `LoginPage` juga dibungkus
  `{!import.meta.env.PROD && (...)}` — Vite mengganti `import.meta.env.PROD` jadi literal boolean
  saat build produksi, jadi seluruh panel (termasuk daftar akun) ikut ter-tree-shake habis dari
  bundle produksi, bukan sekadar disembunyikan lewat CSS.
- Sesi/logout: diperiksa ulang, ternyata **sudah memadai sejak Fase 1** —
  `logoutAndRedirect` (`shared/lib/auth-actions.ts`) sudah membersihkan sesi lokal DAN memanggil
  `POST /auth/logout` untuk mencabut refresh token di server. Tidak ada perubahan yang diperlukan
  di sini.
- Item "tabel histori ganti perwakilan client" (§6 poin 3) — sudah diputuskan di Fase 4
  (overwrite-only, tanpa tabel audit terpisah), jadi tidak ada pekerjaan baru di fase ini.

Checkpoint:
- [x] Tidak ada satu route pun yang bisa diakses tanpa token valid + role yang sesuai (diuji langsung, bukan hanya lewat UI)
- [x] Tidak ada password/secret plaintext tersisa di source frontend
- [x] Semua list page yang relevan memakai pagination berbasis server (`ClientListPage` dikecualikan dengan alasan struktural di atas)

**Catatan implementasi:**
- Keputusan lingkup pagination (opsi 1/2/3) sengaja ditanyakan ke user lewat `AskUserQuestion`
  karena checkpoint literal PLAN ("semua list endpoint dipaginasi") berbenturan langsung dengan
  banyak konsumen yang butuh data penuh (dropdown, global search, agregasi dashboard) — user
  memilih opsi 3 (paling menyeluruh: paginasi default + endpoint/parameter `?all=true` terpisah),
  bukan skip atau partial.
- Backend build + `go vet` dijalankan ulang setelah **setiap** modul selesai dikonversi (bukan cuma
  di akhir) untuk mengisolasi kesalahan sedini mungkin di codebase seukuran ini.
- Verifikasi end-to-end: backend diverifikasi lewat curl asli (login staff/platform-admin sungguhan,
  cek `meta` + `?all=true` + kombinasi `search`/`role`/`status` di 9 endpoint), frontend lewat
  Playwright asli (klik tombol demo login, ketik di search box lalu konfirmasi request jaringan
  sungguhan `?search=...` terkirim dan tabel ter-update, screenshot tiap list page) — bukan cuma
  `tsc`/`go build` lolos.

---

### Fase 8 — Deployment
**Status:** 🟢 Selesai (konfigurasi + Dockerfile diperbaiki dan dinalar habis; **`docker compose up` itu sendiri belum benar-benar dieksekusi** — Docker tidak terpasang di environment ini, lihat catatan di bawah)

Tujuan: satu container app (frontend+backend sesuai standar) + database di host, siap dijalankan
dari clone bersih.

Lingkup yang benar-benar dikerjakan:
- `docker-compose.yml` satu app container (sesuai `.claude/rules/monorepo.md`) — sudah ada sejak
  awal, tapi ternyata punya bug nyata yang baru ketahuan saat ditelusuri ulang di fase ini (lihat
  Catatan implementasi).
- `.env.example` (root + `apps/web/`) diberi komentar yang menjelaskan pembagian tanggung jawab
  `DATABASE_URL` (dev lokal vs override Docker) dan kenapa `VITE_API_BASE_URL` sengaja tidak dipakai
  di build Docker.
- `docs/DEPLOYMENT.md` ditulis ulang total: clone → konfigurasi env → migrate + seed → run lokal →
  Docker, termasuk penjelasan kenapa override `DATABASE_URL` di compose diperlukan dan bagaimana
  demo-login/panel "Akun Demo" berperilaku beda di `APP_ENV=production`.

Checkpoint:
- [x] Konfigurasi dinalar dan diperbaiki habis sampai *seharusnya* jalan end-to-end dari clone bersih tanpa langkah manual tambahan — **tapi belum diverifikasi lewat `docker compose up` sungguhan** (Docker tidak tersedia di environment ini)
- [x] `docs/DEPLOYMENT.md` mencerminkan langkah yang benar-benar dinalar/diuji sebagian (lihat rincian per langkah), dengan bagian "Known limitation" yang secara eksplisit mengakui bagian yang belum diuji, bukan mengklaim semuanya sudah diverifikasi

**Catatan implementasi (3 bug nyata ditemukan & diperbaiki, bukan cuma tinjauan kosmetik):**
1. **`DATABASE_URL` di `docker-compose.yml` tidak akan pernah bisa connect ke MySQL di host.**
   `.env` (dipakai lewat `env_file:`) berisi `DATABASE_URL=...tcp(localhost:3306)...` untuk run
   lokal non-Docker — tapi di dalam container, `localhost` merujuk ke container itu sendiri, bukan
   mesin host, walaupun `extra_hosts: host.docker.internal:host-gateway` sudah benar dipasang.
   Diperbaiki dengan menambah blok `environment:` di `docker-compose.yml` yang meng-override
   `DATABASE_URL` khusus untuk container, menyusun ulang dari `DB_USER`/`DB_PASSWORD`/`DB_PORT`/
   `DB_NAME` (Compose otomatis membaca `.env` di root untuk interpolasi `${...}` ini) tapi dengan
   host `host.docker.internal`.
2. **Server Go tidak pernah menyajikan file statis frontend sama sekali.** `Dockerfile` sudah
   meng-copy hasil build frontend ke `./public` di image final, tapi `cmd/server/main.go` tidak
   pernah punya route untuk itu — mengunjungi `/` di container akan 404 total. Ini bug yang benar-
   benar menghalangi checkpoint "aplikasi berjalan end-to-end", bukan cuma kekurangan kosmetik.
   Ditambahkan `spaFileServer` (helper baru di `main.go`): menyajikan file statis dari `./public`,
   fallback ke `index.html` untuk path apa pun yang bukan file asli (perlu supaya route React
   Router seperti `/dashboard` tetap jalan saat direct-link/hard-refresh) — no-op aman di dev lokal
   karena direktori `./public` memang tidak ada di sana (frontend jalan lewat Vite di port 5173).
3. **Efek samping dari bug #2 saat diperbaiki: fallback SPA awalnya "menelan" request `/api/v1/...`
   yang tidak match jadi respons 200 HTML palsu**, ditemukan lewat pengujian manual (jalankan
   binary Go dengan `./public` diisi hasil build asli): `POST /api/v1/auth/demo-login` di
   `APP_ENV=production` (di mana endpoint itu sengaja tidak didaftarkan) balikin status 200 alih-alih
   404 — karena mux jatuh ke handler `"/"` yang tidak peduli prefix path atau method HTTP. Diperbaiki
   dengan mengecek prefix `/api/` di awal `spaFileServer` dan langsung membalas 404 JSON
   (`response.Error`) untuk itu, sebelum logic fallback index.html dieksekusi sama sekali.
4. `Dockerfile`: `npm install --workspaces --include-workspace-root=false || true` di stage frontend
   diganti jadi `npm install --workspace=apps/web --include-workspace-root=false` (tanpa `|| true`)
   — `packages/api-contract` tidak punya `package.json` sama sekali (cuma `README.md`), jadi
   `--workspaces` generik berisiko galat di glob `packages/*`; `|| true` sebelumnya menyembunyikan
   kegagalan install apa pun alih-alih membuat build gagal cepat dan jelas. Karena `apps/web` sudah
   dikonfirmasi tidak bergantung ke `packages/shared` sama sekali (`grep` kosong), mempersempit scope
   instalasi ke `apps/web` saja aman dan lebih deterministik. Stage final juga ditambah user non-root
   (`addgroup`/`adduser` + `USER app`) — Alpine berjalan sebagai root secara default, yang tidak
   perlu untuk proses yang cuma menyajikan HTTP.
5. **Verifikasi yang benar-benar dilakukan** (mengingat Docker sendiri tidak terpasang di environment
   ini): build frontend produksi asli (`npm run build -w apps/web`), copy hasilnya ke
   `apps/api/public`, compile+jalankan binary Go asli dengan `APP_ENV=production` persis meniru
   kondisi di dalam container, lalu curl manual ke `/`, `/dashboard` (fallback SPA), asset statis
   asli, `/api/v1/health`, `/api/v1/auth/login` (harus tetap jalan), dan
   `/api/v1/auth/demo-login` + endpoint API acak yang tidak ada (harus 404 JSON, bukan HTML) — semua
   sesuai ekspektasi setelah perbaikan bug #3. Yang **tidak** diverifikasi: `docker build`/
   `docker compose up` itu sendiri (image build multi-stage, resolusi `host.docker.internal` yang
   sungguhan dari dalam container Docker Desktop) — didisclose eksplisit ke user, bukan diklaim
   sudah teruji end-to-end.

---

### Fase 9 — Modul `payment` (Mode Internal) + Integrasi Tripay Sungguhan
**Status:** 🟡 Berjalan (backend + frontend selesai dan terverifikasi lewat curl + boot manual; charge/webhook Tripay **belum diuji dengan kredensial sandbox sungguhan** — lihat catatan)

Tujuan: mengganti `platform`'s `Pay` yang sebelumnya murni simulasi (langsung menulis transaksi
`paid` dan mengaktifkan langganan tanpa proses pembayaran nyata sama sekali) dengan pembayaran QRIS
sungguhan lewat Tripay, mengikuti pola "satu dompet merchant" di `MODULE_PAYMENT.md`. Fase ini
**hanya mode internal** — `platform` sebagai satu-satunya App, dipanggil sebagai fungsi Go langsung,
tanpa App registry/HTTP eksternal/webhook relay (itu Fase 10).

Lingkup yang benar-benar dikerjakan:
- `modules/payment` baru — struktur folder persis §3 `MODULE_PAYMENT.md` (`contracts/`, `domain/`,
  `infrastructure/` — `client.go`, `gateway.go`, `tripay.go`, 4 repository file — `presentation/`,
  `payment.module.go`). Sesuai catatan desain dokumen itu, modul ini **tidak punya lapisan
  `application/` terpisah** — `infrastructure.PaymentService` satu struct yang mengimplementasikan
  `contracts.Client` (App internal), `contracts.Dispatcher` (composition root), dan admin CRUD
  (`GetConfig`/`UpdateConfig`) sekaligus.
- 4 tabel baru (bukan ledger — lihat §4), migration `000009`: `payment_gateway_config` (baris
  tunggal, kredensial Tripay terenkripsi AES-256-GCM, kunci dari env var baru
  `PAYMENT_ENCRYPTION_KEY`), `payment_apps` (App Registry — sudah termasuk kolom `kind=external`-only
  sejak sekarang supaya Fase 10 tidak butuh migration lagi), `payment_charge_dispatch`,
  `payment_webhook_events`.
- Kontrak publik internal (§5): `Enabled`, `QuoteFee`, `CreateCharge`/`CreateChannelCharge`,
  `ListChannels`, `CheckStatus`. Provider Tripay (`infrastructure/tripay.go`) mengimplementasikan
  interface `gateway` yang provider-agnostic.
- **`platform` (bukan `billing`) yang jadi App internal + webhook consumer** — lihat Catatan
  implementasi #1 untuk kenapa rencana awal ("`billing` App-nya") tidak bisa dipakai begitu saja.
  `TenantService.Pay` sekarang memanggil `payment.Client.CreateCharge` dan mengembalikan charge
  (bukan tenant yang sudah aktif); `TenantService.ApplyWebhookEvent` (metode baru, mengimplementasikan
  `contracts.WebhookConsumer`) yang benar-benar mengaktifkan langganan, dipanggil oleh
  `payment.Dispatcher` saat webhook masuk. `ActivateSubscription` (bypass admin) **tidak berubah
  sama sekali** — tetap sinkron, tidak pernah menyentuh gateway.
- `billing` dapat status transaksi baru, `pending` (charge dibuat, menunggu konfirmasi — beda dari
  `unpaid` yang berarti belum pernah ada percobaan charge sama sekali), plus method kontrak baru
  `UpdateTransactionStatus` (dipanggil `platform` saat webhook mengonfirmasi/menggagalkan charge).
- Route webhook baru, satu-satunya, tidak pernah berubah: `POST /webhooks/payment` (tanpa auth JWT —
  kepercayaan dari verifikasi signature HMAC).
- Platform Console: halaman baru **Konfigurasi Gateway** (`/platform/pembayaran`) — pilih provider
  (kosong = mode simulasi, atau Tripay), sandbox/production, merchant code, API key + private key
  (write-only, form kosong = tidak diubah).
- WO Console: `SubscriptionPage`'s "Bayar Sekarang" sekarang membuka modal QRIS/checkout-link,
  lalu polling riwayat transaksi tiap 4 detik mencari baris dengan `paymentReference` yang cocok
  sampai statusnya bukan lagi `pending` — bukan sukses instan.

Checkpoint:
- [x] `platform.Pay` menghasilkan charge lewat pemanggilan gateway sungguhan (bukan baris `paid` instan) — diverifikasi: endpoint memanggil Tripay asli dan gagal terhormat (500 bukan sukses palsu) saat kredensial dummy dipakai
- [ ] Webhook Tripay sungguhan (sandbox) berhasil diverifikasi, diproses idempoten (dikirim ulang 2x → diproses sekali), dan mengubah status langganan tenant jadi aktif — **belum bisa diuji, butuh kredensial sandbox nyata (lihat catatan)**
- [x] Kredensial Tripay tersimpan terenkripsi at-rest, tidak pernah ter-echo balik ke Platform Console dalam bentuk plaintext — diverifikasi lewat curl (`GET gateway-config` hanya mengembalikan `hasTripayApiKey`/`hasTripayPrivateKey` boolean)
- [x] `ActivateSubscription` (bypass admin) tetap berfungsi persis seperti sebelumnya, tidak tersentuh perubahan gateway — kode path-nya sama sekali tidak memanggil `payment` module

**Catatan implementasi:**
1. **Rencana awal PLAN.md menyebut "`billing` App internal-nya" — ternyata tidak bisa, `platform`
   yang harus jadi App-nya.** Alasan: begitu webhook mengonfirmasi charge, yang perlu terjadi bukan
   cuma "tandai transaksi paid" (itu memang `billing`'s job) tapi juga "aktifkan langganan tenant"
   (mengubah `tenants.subscription_status`/`subscription_expires_at`, tabel yang dimiliki
   `platform`, bukan `billing`). `billing` tidak boleh (dan secara struktur tidak bisa, tanpa
   membuat cycle) meng-`import` `platform` untuk melakukan itu — sementara `platform` **sudah**
   meng-`import` `billing.Contracts` sejak awal (orchestrator yang sudah ada, lihat `Register`/
   `grantOrPay` lama). Jadi `platform` yang didaftarkan sebagai App internal
   (`paymentcontracts.InternalAppBilling = "platform-billing"`) dan yang mengimplementasikan
   `ApplyWebhookEvent` — bukan `billing`. Dijembatani lewat mapping baru `platform`'s sendiri,
   `pending_subscription_charges` (order_ref → tenant_id+plan_id), karena `payment` module sendiri
   sengaja tidak pernah tahu apa itu "tenant" atau "plan" (lihat non-goals `MODULE_PAYMENT.md`).
2. **Ditemukan lewat verifikasi manual sebelum sesi ini: `apps/web/tsconfig.json` tidak punya
   `noEmit`, jadi `npm run build -w apps/web` (dijalankan untuk uji Docker di Fase 8) menulis 153
   file `.js` hasil kompilasi tersebar di `apps/web/src`, persis di sebelah source `.tsx`-nya.
   Ditemukan saat menelusuri struktur modul `platform-admin` untuk fase ini, dikonfirmasi user,
   dihapus semua, dan `noEmit: true` ditambahkan ke tsconfig supaya tidak terulang.**
3. Endpoint path (`/merchant/payment-channel`), base URL sandbox (`tripay.co.id/api-sandbox`), dan
   skema auth (`Authorization: Bearer <api_key>`) di `tripay.go` **sempat diverifikasi langsung**
   lewat curl ke Tripay sandbox sungguhan (tanpa kredensial nyata) — responsnya persis format Tripay
   (`{"success":false,"message":"Invalid API Key"}`), mengonfirmasi base URL/path/skema auth benar.
   Yang **belum** diverifikasi: bentuk request/response `/transaction/create` dan
   `/transaction/detail`, serta bentuk body+signature webhook callback — itu ditulis mengikuti
   dokumentasi publik Tripay, bukan hasil curl langsung (butuh kredensial merchant sungguhan).
4. **Kebutuhan input user sebelum checkpoint webhook bisa benar-benar dicentang:** akun Tripay
   sandbox sungguhan (merchant code + API key + private key) — sama seperti kredensial IDCloudHost
   S3 yang dulu diperlukan Fase 4, ini dependency eksternal yang tidak bisa disimulasikan penuh.
   Setelah kredensial tersedia: input lewat Platform Console → Konfigurasi Gateway, lalu uji
   `Pay` sungguhan dari WO Console dan bayar QR-nya langsung dari HP.
5. `QuoteFee` (§5) mengasumsikan skema biaya Tripay flat+persentase (`domain.Channel.QuoteFee`) —
   belum diverifikasi terhadap skema biaya QRIS Tripay yang sesungguhnya (juga menunggu kredensial
   nyata), tapi tidak dipakai di jalur kritis `Pay` (cuma metode pelengkap di kontrak).
6. **Bug nyata ditemukan lewat Playwright, bukan cuma `tsc`/`go build` lolos:** endpoint konfigurasi
   gateway awalnya didaftarkan sebagai `PUT /api/v1/payment/gateway-config` — begitu diklik dari
   Platform Console (browser sungguhan, port 5173 ke API port 8080), request gagal total kena
   CORS preflight, karena `middleware.CORS`'s `Access-Control-Allow-Methods` cuma mengizinkan
   `GET, POST, PATCH, DELETE, OPTIONS` (tidak ada `PUT`) — daftar itu ditulis sebelum modul ini ada
   dan tidak pernah diantisipasi butuh method baru. Diperbaiki dengan mengganti endpoint ini jadi
   `PATCH` (bukan menambah `PUT` ke whitelist CORS bersama) supaya konsisten dengan konvensi
   endpoint update di seluruh modul lain (`staff`/`vendors`/`clients`/`projects`/`tenants`/
   `platform-admins`/`plans` semuanya `PATCH`, tak satu pun pakai `PUT`) — sekaligus tidak menambah
   scope perubahan ke middleware bersama untuk kasus yang sebenarnya cukup diselesaikan di modul ini
   sendiri.
7. Diverifikasi lewat Playwright sungguhan (bukan cuma curl): login sebagai platform_admin → buka
   Konfigurasi Gateway → pilih Tripay, isi kredensial dummy, simpan → status berubah jadi "Aktif" +
   pesan sukses tampil. Login sebagai staff Owner → klik "Perpanjang Langganan" → "Bayar Sekarang" →
   backend mencoba charge sungguhan ke Tripay sandbox dengan kredensial dummy, gagal (500), pesan
   error tampil di UI (bukan gagal diam-diam), modal konfirmasi tetap terbuka untuk retry — semua
   sesuai ekspektasi. Data uji (kredensial dummy) direset ke kosong setelah verifikasi.

---

### Fase 10 — Modul `payment` (Mode Eksternal) — Payment Gateway as a Service untuk SaaS Lain
**Status:** 🔴 Belum mulai

Tujuan: membuka dompet merchant yang sama (dibangun Fase 9) sebagai API server-to-server, sehingga
SaaS lain di luar ElProof — codebase terpisah, server terpisah — bisa membuat charge QRIS/VA lewat
dompet ini tanpa pernah menyentuh kredensial Tripay-nya. Sesuai §2.2 & §9 `MODULE_PAYMENT.md`: fase
ini murni **aditif** di atas infrastruktur Fase 9 (registry App, dispatcher, tabel config yang
sama) — tidak mengubah cara `platform` (App internal, `platform-billing`) bekerja sama sekali.

Lingkup:
- Aktor baru di `identity`: **App** — client-credentials, tidak terikat tenant/role, sejajar dengan
  `staff`/`client`/`platform_admin` yang sudah ada. Reuse `TokenIssuer` yang sudah ada di
  `identity` untuk menerbitkan access token App (tanpa refresh token — App cukup menukar ulang
  `appId`+`secret` saat token kedaluwarsa, lihat §7.1).
- `POST /auth/app/token` — tukar `{appId, secret}` dengan access token. Status App dicek **live**
  di setiap request berikutnya (bukan cuma saat token terbit) — menonaktifkan App langsung memutus
  token yang sedang berjalan. Rate limit ketat per-IP.
- Tiga route baru, digerbangi aktor App (§7.2), `appId` selalu diresolusi dari token (tak pernah
  dari body — App tidak bisa menyamar App lain):
  - `POST /external/payments/charges`
  - `GET /external/payments/charges/{orderRef}/status`
  - `GET /external/payments/channels`
- App Registry diperluas untuk mendukung baris `kind=external`: `secret` disimpan dua bentuk (hash
  bcrypt untuk verifikasi masuk saat token exchange, salinan terenkripsi reversibel untuk
  menandatangani relay webhook keluar — lihat §7.5, ini keputusan kriptografi paling penting di
  pola ini). Uniqueness `order_ref` di Charge Dispatch Index otomatis memberi idempotensi (percobaan
  ulang dengan `order_ref` sama → `409 Conflict`), tanpa tabel tambahan.
- Dispatcher webhook (sudah ada sejak Fase 9) diperluas: cabang `kind=external` mengirim relay
  fire-and-forget (tepat satu percobaan, timeout pendek, tanpa retry) ke `callback_url` App itu,
  ditandatangani HMAC-SHA256 atas raw body. Fallback App eksternal untuk relay yang hilang: polling
  lewat `GET .../status`.
- Platform Console: halaman **Manajemen Aplikasi** (baru) — daftar App (internal + eksternal),
  tombol **Tambah Aplikasi eksternal** (generate `client_id`+`secret`, secret **cuma tampil satu
  kali**, persis pola reveal-password yang sudah ada di Tenant/Staff/Platform-Admin), **Reset
  Secret**, **Aktifkan/Nonaktifkan**.
- Kode error acuan §7.6 (`bad_request`/`unauthorized`/`forbidden`/`not_found`/`conflict`/
  `rate_limited`/`internal`) dipakai konsisten di ketiga route eksternal.

Checkpoint:
- [ ] SaaS eksternal (disimulasikan lewat curl/Postman, tanpa akses ke codebase ElProof sama sekali) berhasil menukar `appId`+`secret` jadi token, membuat charge, dan menerima relay webhook tertandatangani saat charge lunas
- [ ] Menonaktifkan App eksternal lewat Platform Console langsung menolak token yang sedang berjalan (bukan menunggu expiry)
- [ ] Percobaan membuat charge dengan `order_ref` yang sama dua kali menghasilkan `409 Conflict`, bukan charge duplikat
- [ ] Menambah App eksternal baru tidak mengubah/mengganggu perilaku `platform` (App internal) sama sekali — diverifikasi dengan menjalankan ulang alur `Pay` Fase 9 setelah App eksternal pertama didaftarkan
- [ ] Secret App eksternal tidak pernah bisa dibaca ulang lewat API/Platform Console setelah dibuat — hanya reset (nilai baru, ditampilkan sekali lagi)

---

### Catatan Operasional — Reset Data ke Minimal (di luar penomoran fase)

Atas permintaan eksplisit user, `cmd/seed` (dan database dev yang sudah dijalankan) diubah dari
seed demo yang kaya (3 paket, 1 tenant contoh, 3 staff, 1 project, 1 client — dipakai sepanjang
Fase 1–9 untuk verifikasi Playwright/curl) menjadi **minimal**: hanya satu akun Platform Console
(`superadmin` / `superadmin`, email literal `superadmin`) dan satu paket langganan ("Paket 1 Tahun",
12 bulan, Rp 2.000.000) — tabel lain (tenants, staff_members, clients, vendors, vendor_categories,
projects + seluruh sub-entity, subscription_transactions, tabel `payment` Fase 9) benar-benar
kosong. Diverifikasi lewat curl + Playwright: login `superadmin` berhasil, Platform Dashboard
menampilkan semua statistik nol, `payment_gateway_config` kembali ke baris tunggal default
(`active_provider=NULL`), `payment_apps` cuma berisi App internal (`platform-billing`, otomatis
di-bootstrap ulang oleh `cmd/server` — tidak perlu dimasukkan `cmd/seed`).

Konsekuensi untuk siapa pun yang melanjutkan sesi ini: tidak ada tenant/staff/client sampai dibuat
manual lewat Platform Console → Tenant → Tambah Tenant. Menjalankan ulang `go run ./cmd/seed` akan
selalu kembali ke state minimal ini, bukan dataset demo lama. (Mekanisme demo-login yang disebut di
paragraf ini sejak itu **dihapus total** — lihat catatan operasional berikutnya di bawah.)

---

### Catatan Operasional — Penghapusan Total Mock/Dummy/Demo (di luar penomoran fase)

Atas permintaan eksplisit user lanjutan ("pastikan semua terintegrasi tanpa mock, dummy, dan
hal-hal berkaitan dengan demo"), dikerjakan sampai tuntas, bukan sekadar audit:

1. **Fitur nyata terakhir yang masih bergantung ke `apps/web/src/mock` dibangun dulu ke backend,
   baru mock-nya dihapus** — bukan sebaliknya (melepas fitur demi bisa hapus lebih cepat).
   `VendorListPage`'s "Lihat Project" sebelumnya memanggil `getVendorProjectHistory()` dari
   `mock/selectors.ts` (satu-satunya pemakaian **runtime** mock di seluruh frontend; sisanya cuma
   `import type`, hilang saat build). Sekarang: `GET /vendors/{id}/project-history` (baru) —
   `vendors` module memanggil `projects.Contracts.ListVendorEngagementHistory` (metode baru) karena
   `project_vendors` dimiliki `projects`, bukan `vendors` — konsisten dengan aturan "no cross-module
   join", diselesaikan lewat contracts, bukan join lintas modul. `vendors.NewModule` sekarang
   menerima `projects.Contracts` sebagai constructor arg (dependency satu-arah biasa, bukan siklik —
   `main.go` cukup membangun `projects` sebelum `vendors`, lihat `knowledge/MODULE_MAP.md`).
2. **Seluruh `apps/web/src/mock/` dihapus** setelah dipastikan nol pemakaian tersisa (`grep` kosong
   total). Tipe yang sebelumnya cuma hidup di `mock/types.ts` (`StaffMember`, `StaffRole`, `Vendor`,
   `VendorCategory` — `ClientRole`/project-related types sudah punya rumah asli sejak Fase 4/6)
   dipindah ke `modules/users/types.ts`, `modules/vendors/types.ts`,
   `modules/vendor-categories/types.ts` yang baru dibuat. Beberapa komentar kode yang sudah usang
   ("separate dataset from mock.X, dipakai `projects` yang masih full-mock sampai Fase 4 selesai")
   ikut dibersihkan — proyek sudah lama melewati titik itu.
3. **Mekanisme demo-login (Fase 7) dihapus total**, bukan cuma disembunyikan — dianggap "hal
   berkaitan dengan demo" sesuai instruksi user, dan sudah kurang relevan setelah database direset
   ke minimal (§ di atas): `POST /api/v1/auth/demo-login` (handler, service method, route
   registration, `demoLoginEnabled` flag di `identity.module.go`) dihapus dari backend; panel "Akun
   Demo" dan `loginAsDemoAccount` dihapus dari `LoginPage.tsx`; `demo-accounts.ts` dihapus. Login
   sekarang cuma satu jalur: username + password asli (`superadmin`/`superadmin` atau akun tenant
   yang sudah dibuat).
4. Diverifikasi: `go build`/`go vet` bersih, `tsc -b --noEmit` bersih (termasuk memastikan nol sisa
   `import ... from "@/mock"` sebelum folder-nya benar-benar dihapus, bukan setelahnya).

---

### Catatan Operasional — Deploy Produksi Pertama (di luar penomoran fase)

ElProof sekarang **live** di <https://elproof.elcodelabs.com> (VPS "Elcodelabs", 2026-07-17).
Detail teknis lengkap ada di `docs/DEPLOYMENT.md` §6 dan `knowledge/decisions/ADR-0011-deployment-target.md`
— ringkasan:

1. VPS ini sudah punya konvensi multi-app baku sendiri (`elkasir` adalah app pertama): CI (GitHub
   Actions) yang build image → push ke GHCR, VPS cuma `docker pull` + jalankan (server 2GB RAM
   tidak pernah build). ElProof mengikuti pola ini persis, bukan `docker compose up --build`
   langsung di server seperti asumsi awal.
2. `cmd/server` sekarang punya subcommand (`migrate up/down/force`, `seed`, `healthcheck`) supaya
   satu binary/image sudah cukup untuk semua kebutuhan operasional — tidak perlu CLI migrate/seed
   binary terpisah di VPS.
3. Deploy pertama menemukan 2 bug nyata (keduanya sudah diperbaiki): Dockerfile pakai `CMD` bukan
   `ENTRYPOINT` (bikin `docker run image migrate up` gagal exec), dan migrator awalnya reuse shared
   `*sql.DB` tanpa `multiStatements` (bikin file migrasi multi-statement gagal). Bug kedua sempat
   meninggalkan `schema_migrations` dalam status "dirty" — dipulihkan lewat subcommand baru
   `migrate force <version>`, bukan SQL manual ke database produksi.
4. Domain `elproof.elcodelabs.com` awalnya salah arah (ALIAS record sisa hosting Hostinger lain di
   registrar yang sama) — diperbaiki jadi A record ke IP VPS sebelum nginx/certbot dipasang.
5. Status akhir terverifikasi: container sehat, 25 tabel ter-migrasi lengkap, data seed minimal
   (`superadmin`/`superadmin` + "Paket 1 Tahun") ada, login dan frontend teruji end-to-end lewat
   HTTPS publik.

---

### Catatan Operasional — Halaman Marketing Publik `/homepage` (di luar penomoran fase)

Ditambahkan modul frontend baru `modules/homepage` — **murni frontend, tanpa modul backend maupun
panggilan API sama sekali** (lihat baris `homepage` di `knowledge/MODULE_MAP.md`):

1. Tiga route baru: `/homepage` (landing), `/homepage/syarat-ketentuan`, `/homepage/kontak` — dengan
   `MarketingLayout` sendiri (nav atas + footer), terpisah dari layout WO Console/Client Portal/
   Platform Console yang sudah ada.
2. Desain memakai `frontend-design` skill: tetap 100% palet token yang sudah ada (navy 950/900/800,
   aksen `warning` yang sudah ada), tambah font `Fraunces` khusus judul (token baru `--font-display`
   di `theme.css`, dipakai lewat utility Tailwind `font-display`) — body tetap Inter.
3. Elemen tanda tangan visual: segel SVG "ElProof · Satu Pintu, Semua Terbukti"
   (`modules/homepage/components/ProofSeal.tsx`), dianimasikan sekali di hero.
4. Bug kecil ditemukan & diperbaiki saat verifikasi: token font kustom awalnya salah namespace
   (`--font-family-display`, bukan `--font-display`) sehingga Tailwind v4 tidak membuatkan utility
   `font-display` — judul sempat ikut Inter, bukan Fraunces.
5. CTA utama "Hubungi Kami" (bukan "Daftar Sekarang") — konsisten dengan arsitektur ElProof yang
   *sales-assisted* (tenant didaftarkan admin platform, bukan self-service signup).
6. Data kontak di `modules/homepage/data/contact.ts` — awalnya placeholder jelas-palsu, sudah diganti
   data asli dari user (email `cs@elcodelabs.com`, telepon/WhatsApp `0851-7347-1146`, alamat kantor,
   jam layanan, website `elkasir.elcodelabs.com`).
7. Karena tidak menyentuh backend/database, deploy fitur ini hanya perlu lewat pipeline CI→GHCR→VPS
   yang sudah ada (Fase 9 operasional di atas) — tidak ada migration, tidak ada perubahan `.env`,
   tidak ada perubahan infra baru.

---

## 6. Keputusan Terbuka (Butuh Konfirmasi User)

Item berikut **sengaja tidak diputuskan sepihak** dalam dokumen ini karena menyentuh preferensi
atau menyimpang dari default proyek — konfirmasi sebelum fase terkait dimulai:

1. ~~**React Query/SWR vs pola Zustand manual**~~ — **DIPUTUSKAN (2026-07-16): Zustand murni,
   konsisten di seluruh modul.** User mengonfirmasi eksplisit: semua state — termasuk 7 modul yang
   sebelumnya belum punya store (`staff`/`users`, `clients`, `vendors`, `vendor-categories`,
   `projects` + sub-entity) — harus memakai pola Zustand store yang sama seperti
   `usePlatformAdminStore`/`useSubscriptionPlanStore`/`useAuthStore`. Tidak ada React Query/SWR di
   proyek ini. Setiap modul mendapat satu store di `modules/<m>/stores/use<Modul>Store.ts` dengan
   state + actions (create/update/delete/toggle), dipanggil dari komponen via selector hook, dan
   actions memanggil `httpClient` lalu meng-update state lokal store (pola "fetch-then-set", sama
   persis seperti `usePlatformAdminStore` sekarang) — bukan cache otomatis ala React Query.
2. ~~**Router Go**~~ — **DIPUTUSKAN (default dipertahankan sampai akhir Fase 4): tetap `net/http`
   `ServeMux` bawaan, tanpa `chi`/`gorilla/mux`.** Terbukti cukup bahkan untuk subtree ternested
   dalam di `projects` (`/projects/{id}/vendors/{pvId}/milestones/{id}`) lewat satu method
   dispatcher per subtree (`Handler.Item`) yang mem-parse segment path — lihat catatan implementasi
   Fase 4. Tidak ada kebutuhan nyata yang muncul untuk router pihak ketiga.
3. ~~**Riwayat "ganti perwakilan client"**~~ — **DIPUTUSKAN (Fase 4): tetap overwrite-only, tanpa
   tabel audit terpisah.** `POST /clients/{id}/replace-representative` menimpa baris `clients` yang
   sama, konsisten dengan perilaku mock sebelumnya — lihat `docs/DB_SCHEMA.md` §`clients` dan
   catatan implementasi Fase 4 di atas.
4. **Fitur paket sebagai kolom JSON vs tabel `plan_features`** terpisah — berdampak ke Fase 2,
   tergantung apakah fitur perlu di-query/di-filter individual di masa depan atau cukup ditampilkan
   sebagai daftar.

---

## 7. Cara Memperbarui Dokumen Ini

- Update **Status** di header tiap fase (🔴/🟡/🟢/⏸️) begitu mulai/selesai — jangan tunggu semua
  fase selesai baru diupdate sekali.
- Centang item Checkpoint hanya setelah benar-benar diverifikasi (idealnya via Playwright atau
  pengujian API langsung), bukan saat kode "kelihatannya sudah benar".
- Kalau ada keputusan baru yang menyimpang dari §4/§6, tambahkan ADR baru di `knowledge/decisions/`
  dan tautkan dari sini — jangan biarkan keputusan hanya hidup di riwayat chat.
- Dokumen ini adalah pelengkap `knowledge/` dan `docs/`, bukan pengganti — detail teknis final tetap
  tinggal di `docs/DB_SCHEMA.md`/`API_CONTRACT.md` begitu Fase 0 selesai.
