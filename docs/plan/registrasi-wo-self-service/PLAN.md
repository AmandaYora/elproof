# PLAN.md — Registrasi WO Self-Service + Perbaikan Flow & UI/UX Menyeluruh

> Status: **DISETUJUI ARAHNYA — K1–K5 diputuskan 5 Nov 2026; K6 & K9 masih terbuka (§6)**. Disusun 5 Nov 2026 dari audit kode
> (3 console + backend) dan riset eksternal (sumber di §8). Belum ada kode yang diubah.

## 0. Ringkasan

ElProof saat ini **tidak punya registrasi mandiri**: tenant WO didaftarkan manual oleh tim
platform (`POST /api/v1/tenants`, khusus platform admin), dan password Owner, staf, maupun
klien diketik oleh orang lain lalu dikirim di luar sistem. Model ini tidak skalabel untuk
"banyak WO × banyak klien". Audit juga menemukan bahwa langganan, suspend, dan nonaktif akun
**tidak pernah ditegakkan**. Registrasi self-service tanpa penegakan langganan sama saja
membuka akses gratis selamanya, jadi fondasi penegakan dikerjakan lebih dulu.

## 1. Kondisi saat ini (terverifikasi di kode)

| # | Fakta | Lokasi |
|---|---|---|
| 1 | Registrasi tenant hanya oleh platform admin. Urutannya tenant → seed → owner → credential **tanpa transaksi/kompensasi**, sehingga kegagalan meninggalkan tenant yatim | `platform/application/tenant_service.go:164-216` |
| 2 | Validasi backend hanya format username. Aturan min 8 karakter untuk password hanya ada di frontend | `tenant_service.go:165`, `platform-admin/schemas/tenant.schema.ts:41` |
| 3 | `username` unik secara global lintas tenant. Login mencoba username lalu email, tanpa pemilihan tenant | `migrations/000001`, `identity/application/auth_service.go:106-125` |
| 4 | Login **tidak di-rate-limit**. Limiter hanya dipasang di `/auth/app/token` | `payment/payment.module.go:53,105` |
| 5 | Status `expired`/`expiring_soon` tidak pernah ditulis. `is_suspended` tidak dicek di login maupun middleware | `platform/domain/tenant.go:12-13`, `shared/middleware/auth.go` |
| 6 | Menonaktifkan klien/staf tidak menonaktifkan credential-nya, jadi akun nonaktif tetap bisa login | `clients/application/client_service.go:229`, `staff_service.go:195` |
| 7 | Email kosong disimpan sebagai `''` (bukan NULL) di kolom UNIQUE | `identity/infrastructure/mysql_credential_repository.go:65` |
| 8 | Tidak ada infrastruktur email/WA, tabel token, maupun lupa password | `go.mod` |
| 9 | Tidak ada kuota paket (proyek, staf, klien, storage) | `migrations/000004` |
| 10 | Password reset dikembalikan dalam plaintext di response | `platform/presentation/tenant_handler.go:443` |
| 11 | Sertifikat TLS produksi kedaluwarsa sejak 28 Okt 2026 | ops/server |
| 12 | Akun klien bisa GET semua sub-resource proyek: pembayaran vendor/venue, log aktivitas, semua dokumen + file (hanya dibatasi metode GET) | `projects/presentation/handler.go:149-191, 271-274` |
| 13 | Role Super Admin & Support platform tidak dibedakan di guard mana pun; tidak ada 2FA & audit log | `platform/domain/platform_admin.go:8-9` |

## 2. Prinsip rancangan (dari riset)

1. **Form daftar pendek**, data lain ditanyakan setelah masuk (progressive profiling). Pola
   SaaS UMKM Indonesia (Kledo, Talenta): nama, nama usaha, WhatsApp, email, password.
2. **Trial tanpa kartu** 14–30 hari. Ini norma pesaing (HoneyBook 30, Planning Pod 14,
   Talenta 14, Accurate 30). Tripay QRIS memang tidak bisa auto-debit.
3. **Tidak ada yang menetapkan password orang lain** (OWASP ASVS 5.0 §6.4.1, §6.4.6). Staf
   dan klien masuk lewat **undangan** sekali pakai yang kedaluwarsa.
4. **Identitas WO dari Google** (`sub` unik global, email sudah terverifikasi Google).
   ElProof tidak menyimpan kata sandi Owner/staf. Otorisasi tetap memakai ID internal,
   bukan email, karena email bisa berganti.
5. **Downgrade bertahap, bukan kunci total**: trial → masa tenggang → baca-saja → suspend.
   Portal klien tetap bisa dibaca supaya pengantin tidak terkena dampak tunggakan WO.
6. **Anti-abuse**: akun Google sudah menyaring sebagian besar bot. Ditambah rate limit per IP
   untuk signup dan login, batas jumlah workspace trial per akun Google, dan tombol bekukan
   tenant di Platform Console. Cloudflare Turnstile disiapkan bila spam muncul.
7. **WhatsApp sebagai kanal utama**: tombol "Kirim via WhatsApp" (`wa.me` + pesan terisi)
   **tanpa biaya API** untuk undangan dan pengingat yang dikirim manual oleh staf WO. API
   WhatsApp resmi (±Rp396/pesan termasuk PPN) hanya untuk pesan otomatis. Gateway tidak
   resmi (Fonnte dan sejenisnya) **tidak dipakai** karena risiko nomor diblokir.
8. **UU PDP / PP 33/2026** (berlaku 16 Jan 2027): pemberitahuan privasi tampil sebelum tombol
   daftar. Centang S&K + DPA (WO = pengendali data klien, ElProof = prosesor) **terpisah**
   dari centang pemasaran yang opsional. Bukti persetujuan disimpan (waktu, versi teks, IP).
9. **Arsitektur**: tetap modular monolith. Transaksi tidak boleh lintas modul, jadi
   registrasi diorkestrasi sebagai **saga dengan kompensasi + idempotensi** di modul `platform`.

## 3. Flow registrasi WO (diputuskan 5 Nov 2026)

> **Keputusan pemilik**: pendaftaran WO **hanya lewat Sign in with Google (OAuth)**. ElProof
> tidak mengelola email/kata sandi Owner baru dan tidak perlu OTP email. Email dari Google
> sudah terverifikasi (`email_verified`). Nomor WhatsApp diminta belakangan di langkah
> "Lengkapi Profil", bukan saat daftar.

```
[Landing "/"] → "Coba Gratis 14 Hari"  |  "Tanya via WhatsApp / Jadwalkan Demo"
      │
[1] /daftar → tombol "Daftar dengan Google" (Google Identity Services)
    Pemberitahuan privasi ringkas di atas tombol
      │  frontend → ID token Google → POST /api/v1/public/signup/google
      │  backend verifikasi ID token (signature JWKS Google, aud = client ID ElProof,
      │  iss, exp, email_verified = true) → simpan `google_sub` + email + nama
      │  email/sub sudah terhubung ke akun → langsung masuk (bukan daftar ulang)
[2] /daftar/workspace (1 layar, hanya untuk akun Google baru)
    Nama WO · alamat/slug (otomatis dari nama, bisa diedit, cek unik)
    ☐ S&K + Perjanjian Pemrosesan Data (wajib) · ☐ Info & promo (opsional)
      │  → `signup_requests` (idempoten per google_sub), bukti persetujuan (waktu, versi, IP)
[3] Pembuatan workspace (saga, idempoten per signup_request):
    tenant (slug otomatis dari nama WO, unik + daftar kata terlarang)
    → seed kategori vendor + Timeline Default → staff Owner
    → credential Owner (tanpa password, `google_sub` terhubung)
    → subscription_status = trial, trial_ends_at = now + 14 hari
    gagal di langkah mana pun → kompensasi mundur, signup_request bisa diulang
[4] Masuk otomatis → Dashboard + Checklist "Mulai di sini":
    ① Lengkapi profil WO (No. WhatsApp, kota, logo)  ② Atur Timeline Default
    ③ Buat proyek pertama  ④ Undang klien ke portal  ⑤ Undang staf  ⑥ Pilih paket
    (+ opsional "Proyek Contoh" yang bisa dihapus sekali klik)
[5] Trial: banner sisa hari untuk semua role; pengingat H-7, H-3, H-0 lewat banner
    (email/WA otomatis menyusul setelah ada kanal kirim)
[6] Trial/langganan habis (diputuskan: bertahap):
    Masa tenggang 7 hari (akses penuh + banner merah)
    → baca-saja (lihat & ekspor, tidak bisa membuat/mengubah; portal klien tetap terbaca)
    → suspend setelah X hari (login staf ditolak dengan pesan jelas + CTA bayar)
    → data tidak dihapus tanpa pemberitahuan
```

**Jalur sales-assisted tetap ada**: Platform Console masih bisa membuat tenant. Bedanya,
Owner menerima **undangan set-password** alih-alih password yang diketik admin.

### Cara masuk per jenis pengguna (diputuskan 5 Nov 2026)

| Pengguna | Cara masuk | Cara mendapatkan akses |
|---|---|---|
| Owner WO | **Google saja** | Daftar sendiri (§3), atau diundang tim platform (jalur sales-assisted) |
| Admin / Staff / Sales | **Google saja** | Undangan dari Owner |
| Klien (pengantin/keluarga) | **Username + kata sandi** | Akun dibuat WO; **kata sandi diatur klien sendiri** lewat link undangan |
| Admin Platform (internal) | Username + kata sandi (tidak berubah) | Dibuat Super Admin |

Data saat ini adalah data dummy/testing, jadi **tidak ada migrasi akun lama**. Skema dan flow
dibangun bersih sesuai tabel di atas; seed/dummy dibuat ulang bila perlu.

### Undangan staf & klien (skala "banyak klien per WO")

**Tanpa penyedia email.** Semua link dikirim staf lewat tombol **"Kirim via WhatsApp"**
(`wa.me/62…?text=` dengan pesan siap kirim, gratis) atau **"Salin link"**.

- **Staf**: Pengaturan → Pengguna → "Undang Staf" (nama, email Google, peran). Link berlaku
  7 hari, sekali pakai, bisa dikirim ulang dan dicabut. Penerima membuka link, lalu "Masuk
  dengan Google". Email Google **harus sama** dengan email yang diundang, supaya link yang
  diteruskan ke orang lain tidak bisa dipakai.
- **Klien**: tab Client di proyek → "Tambah Klien" (nama, peran, No. WhatsApp; username
  dibuat otomatis dan bisa diedit). Sistem membuat link undangan **set kata sandi** (7 hari,
  sekali pakai). WO **tidak pernah mengetik atau melihat** kata sandi klien.
- **Klien lupa kata sandi**: dari halaman login, "Lupa kata sandi? Hubungi WO Anda". Staf WO
  menekan "Buat link atur ulang sandi" (berlaku 24 jam, sekali pakai) lalu mengirimnya via
  WhatsApp. Lupa sandi mandiri lewat email/WA otomatis menyusul setelah ada kanal kirim.
- **Staf/Owner**: tidak punya kata sandi di ElProof, jadi pemulihan akun ditangani Google.
- **Nonaktif/hapus**: menonaktifkan staf atau klien langsung menonaktifkan akses masuknya dan
  mencabut sesi aktif.
- **Kuota paket**: `max_active_projects`, `max_staff`, `max_clients_per_project`,
  `storage_mb` di `subscription_plans`. Dicek di service saat create, dengan pesan
  "Batas paket tercapai → Upgrade".

## 4. Perubahan teknis utama

| Area | Perubahan |
|---|---|
| `platform` | `signup_requests`, kolom `tenants.slug` (UNIQUE), `trial_ends_at`, status `trial`/`grace`/`read_only`; job harian penulis status; kuota di `subscription_plans`; saga registrasi; "Lengkapi Profil" (No. WhatsApp, kota) |
| `identity` | Verifikasi ID token Google (JWKS, `aud`, `iss`, `exp`, `email_verified`) → kolom `google_sub` (UNIQUE) di `credentials`, `password_hash` nullable; endpoint `POST /auth/google`; login username+sandi **hanya untuk klien & admin platform**; tabel `auth_tokens` (invite_staff, set_password_client, reset_password_client — disimpan hash, sekali pakai, kedaluwarsa); rate limit login per IP + per identifier; cek status tenant (suspend) saat login/refresh; email NULL bukan `''`; sinkronisasi `is_active` + pencabutan refresh token; pembersihan refresh token kedaluwarsa |
| `shared` | Middleware `TenantAccessGate` (trial/grace → penuh, read_only → tolak metode tulis dengan kode `subscription_read_only`), helper link `wa.me`. **Tidak ada mailer di fase ini.** |
| `staff`, `clients` | Ganti "buat akun + password" dengan undangan; cek kuota; constraint UNIQUE `(project_id, role)` bila diputuskan satu orang per peran |
| `web` | Tombol Google (Google Identity Services) di `/login` & `/daftar`; `/daftar/workspace`; `/undangan/staf/:token`; `/undangan/klien/:token` & `/atur-sandi/:token`; checklist onboarding; banner langganan global; `/` → landing |
| Konfigurasi | `GOOGLE_CLIENT_ID` (backend & frontend), origin produksi + lokal di Google Cloud OAuth consent screen (scope dasar `openid email profile` saja, tanpa scope sensitif) |
| Platform Console | Antrean/daftar pendaftar baru, tombol bekukan tenant, metrik trial → bayar |

## 5. Rencana perbaikan menyeluruh (bertahap)

Urutan dipilih supaya setiap fase bisa dirilis sendiri dan tidak membuka celah baru.

**Fase 0 — Ops (di server, bukan kode)** — ✅ **SELESAI 6 Nov 2026**
- Akar masalah: sertifikat elproof juga memuat `journey.jwswedding.com` (custom domain tenant) yang
  DNS-nya sudah pindah ke server lain → tantangan HTTP-01 gagal → certbot gagal memperpanjang
  seluruh sertifikat sejak ±28 Sep; kedaluwarsa 28 Okt. `certbot.timer` sendiri sehat.
- Perbaikan: domain JWS dihapus dari site nginx elproof (backup `elproof.bak-20261106`), sertifikat
  diterbitkan ulang hanya untuk `elproof.elcodelabs.com` (berlaku s.d. 3 Feb 2027), deploy hook
  `reload-nginx.sh` ditambahkan, `certbot renew --dry-run` sukses untuk ke-5 sertifikat di VPS.
- Verifikasi eksternal: HTTPS valid, `/api/v1/health` 200, redirect http→https 301.
- Dicatat di `/opt/elcodelabs/SERVER_PLAYBOOK.md`, `docs/DEPLOYMENT.md` §6, ADR-0015 "Deprecation".
- Temuan di luar lingkup (tidak diubah): `stevoapp.my.id` mengarah ke VPS ini tanpa site nginx,
  sehingga tersaji sertifikat elkasir (mismatch).

> Rancangan "seperti apa yang ideal" per peran (Platform Owner, WO, Klien) ada di
> [FLOW-IDEAL.md](FLOW-IDEAL.md). Fase di bawah merujuk kode kesenjangan G1–G11 di sana.

**Fase 1 — Fondasi keamanan & penegakan langganan** (prasyarat registrasi mandiri; G1–G3) — ✅ **SELESAI 6 Nov 2026** (branch `revamp/flow-ideal`; ADR-0022, ADR-0023; diverifikasi unit test + 89 langkah end-to-end pada MySQL sementara + cek visual)
- **Tutup kebocoran data klien di API (G1)**: whitelist endpoint yang boleh diakses principal `client` (bukan "semua GET"), DTO khusus klien untuk proyek/vendor/venue, dokumen difilter `client_visible`, unduh file dicek visibilitasnya.
- Rate limit login; cek suspend di login/refresh; sinkronisasi `is_active` ke credential + cabut refresh token; perbaikan `''` → NULL.
- **State machine langganan** (FLOW-IDEAL §1.4) + job harian + `TenantAccessGate` + banner global di WO Console.
- Platform: peran Super Admin / Billing / Support / Read-only ditegakkan di backend, **2FA (TOTP) admin platform**, **audit log** aksi sensitif, aktivasi manual wajib alasan + bukti, hapus password plaintext dari response, konfirmasi ketik-slug untuk Suspend / Reset Secret / ganti gateway, kunci form gateway saat gagal dimuat.

**Fase 2 — Registrasi WO self-service (Google)** — ✅ **SELESAI 6 Nov 2026** (branch `revamp/flow-ideal`; ADR-0024; diverifikasi unit test + 103 langkah end-to-end pada MySQL sementara dengan Google palsu lokal + cek visual dengan tombol Google asli + review independen)
- Sign in with Google (verifikasi ID token sendiri), `signup_requests` + saga yang bisa dilanjutkan, trial 14 hari, Lengkapi Profil (WhatsApp, kota, logo), checklist onboarding, batas masa coba (K6) yang bisa diubah dari Platform Console, halaman legal (S&K + Perjanjian Pemrosesan Data + Privasi sesuai UU PDP), `/` → landing + CTA daftar.
- **Berbeda dari rencana:** tanpa slug (custom domain dihentikan, K15). Staf lama dengan email yang sama **tidak** ditautkan otomatis ke Google; harus konfirmasi kata sandi ElProof sekali (mencegah "pendudukan" email oleh Owner lain). Login kata sandi untuk staf tetap ada sampai undangan Fase 3.

**Fase 3 — Undangan & akun** — ✅ **SELESAI 6 Nov 2026** (branch `revamp/flow-ideal`; ADR-0025; diverifikasi unit test + 58 langkah end-to-end Fase 3 dan 101 langkah regresi Fase 2 pada MySQL sementara + cek visual + review independen)
- Undangan staf (Google, email harus cocok) dan klien (set sandi sendiri) lewat tombol WhatsApp/salin link, link reset sandi klien oleh WO, redirect ke halaman asal, pesan "sesi berakhir", error jaringan dibedakan dari salah sandi, `<form>` + `autocomplete`.
- Ikut dikerjakan: Owner yang didaftarkan tim ElProof juga menerima undangan Google (tanpa kata sandi dari admin), "Undang ulang Owner" menggantikan reset password di Platform Console, **login kata sandi untuk staf ditutup** (K5), ganti Perwakilan Keluarga langsung mencabut akses orang lama dan membuat nama pengguna baru, status "Menunggu undangan" di daftar staf/client.

**Fase 3a — RBAC per tenant (G12)** — ✅ **SELESAI 6 Nov 2026** (branch `revamp/flow-ideal`; ADR-0026 menggantikan ADR-0017; diverifikasi unit test + 83 langkah end-to-end RBAC (termasuk pencegahan eskalasi peran), 68 langkah Fase 3 dan 101 langkah regresi Fase 2 pada MySQL sementara + uji migrasi data lama naik/turun + cek visual per peran + review independen)
- Tabel `permissions` (katalog tetap), `tenant_roles` + `tenant_role_permissions` (+ cakupan semua/proyek saya), peran klien di tabel yang sama dengan `principal_kind = client`. Seed templat bawaan saat workspace dibuat (Owner terkunci; Admin, Wedding Planner, Sales, Keuangan; Pengantin, Keluarga).
- Middleware/guard izin di backend menggantikan cek string role (ADR-0017 diganti ADR baru). Klaim JWT membawa `role_id`; izin di-resolve dari DB dengan cache singkat.
- Halaman Pengaturan → Peran & Akses (matriks izin per grup, deskripsi awam). Sidebar & tombol mengikuti izin.
- Fase 1 menutup kebocoran klien dengan whitelist dulu; Fase 3a memindahkannya ke izin `portal.*`.
- **Berbeda dari rencana:** katalog izin di kode Go (bukan tabel `permissions`) agar izin baru selalu datang bersama pengecekannya; diuji cocok dengan seed migrasi. Peran lama "Staff" menjadi Wedding Planner. Perubahan peran tercatat di `audit_log` (kini dengan `tenant_id`). Peran bawaan bisa diubah tetapi tidak dihapus; peran kustom hanya bisa dihapus jika tidak dipakai.

**Fase 3b — Fondasi UI (G11)** — ✅ **SELESAI 6 Nov 2026** (bersama Fase 3a)
- Token design system dua kepadatan (console padat, portal lega), komponen Drawer, Toast, Skeleton, ConfirmDialog (ketik nama), label status terpusat (FLOW-IDEAL §4). Memakai skill `frontend-design` sesuai `.claude/rules/frontend-react.md`.
- Hasil: `Drawer`, `Toaster`/`toast`, `Skeleton`, token `--density-*` (console `compact`, portal `comfortable`), `shared/lib/labels.ts` (label staf vs klien), `StatusBadges audience="client"` di portal. `ConfirmDialog` ketik-nama sudah ada sejak Fase 1. Penerapan pola baru ke seluruh halaman lama dilanjutkan di Fase 4.

**Fase 4 — WO Console UX (G7, G9)** — ✅ **SELESAI 7 Nov 2026** (branch `revamp/flow-ideal`; ADR-0027 dan ADR-0028; diverifikasi unit test backend, build web, uji di browser pada MySQL sementara, dan review independen; temuan review yang terbukti (celah Refund pada batas lebih bayar, status ditulis ulang oleh Update, nominal negatif/raksasa, state project lama terbawa, dialog penutupan muncul lagi) sudah diperbaiki)
- **4a:** pola modal standar (`Modal`/`Drawer` + `useDialogBehavior`: tunggu hasil simpan, galat di dalam modal, toast, focus trap, Escape hanya menutup dialog teratas); bug form "Ubah Project"; Escape di Combobox; galat form merah + `aria-invalid`; siklus proyek eksplisit (`/transition`, tombol "Mulai persiapan / Tandai siap / Selesaikan project / Buka kembali", menu "⋯") + checklist penutupan; validasi lebih bayar di backend dan form (klien, vendor, venue) + badge "Lebih bayar"; mode baca-saja menyembunyikan tombol tulis (`useCan`, `useWriteLocked`, `IconActionButton`).
- **4b:** Detail Proyek dua kolom (panel samping) dengan tab Ringkasan pertama, tab Kendala, tab Keuangan dengan **jadwal termin** (tabel `client_payment_terms`, status diturunkan dari pembayaran asli, editor drawer), header diperbaiki.
- **4c:** Beranda berbasis tindakan (angka tagihan terlambat, antrean tindakan, acara 14 hari, tugas minggu ini, pembayaran 30 hari, project per tahap), buat project 2 langkah di `/projects/baru` (venue dari direktori + jadwal termin), venue tunggal (nama venue disinkronkan server), halaman **Profil Usaha** (data usaha, logo, pilihan warna merek oleh Owner), state loading/error/empty dibedakan, target sentuh 44px di perangkat sentuh, sidebar mobile `inert`, istilah "Bukti" seragam.
- **Belum, dipindah ke fase lain:**
  - Beranda "Proyek saya" untuk peran dengan cakupan *proyek saya* (Wedding Planner/Sales): beranda masih khusus peran yang menjangkau semua project (ADR-0026). Masuk Fase 5.
  - Arus kas 8 minggu (butuh jadwal pembayaran vendor/venue, belum ada).
  - Alamat usaha dan rekening tujuan pembayaran klien: baru berguna saat portal klien menampilkannya, jadi dikerjakan di Fase 5.
  - Sakelar "Terlihat klien" per kendala: belum ada kolom di backend.
  - `paidAmount` vendor di daftar vendor masih menghitung semua jenis pembayaran (Tambahan ikut).
  - Nilai `engagementStatus` yang salah di API masih berujung error 500 (bukan 422); temuan lama di luar lingkup.
- Rencana awal fase ini:
- Beranda berbasis tindakan, Detail Proyek dua kolom dengan tab Ringkasan pertama, tab Kendala & Keuangan (jadwal termin), siklus proyek eksplisit + checklist penutupan, buat proyek 2 langkah, Profil Usaha self-service.
- Pola modal standar (await submit, error di dalam modal, tombol nonaktif saat menyimpan) + sistem toast.
- Saat langganan baca-saja: sembunyikan/nonaktifkan tombol tulis di semua halaman (Fase 1 baru menampilkan banner; API sudah menolak dengan pesan jelas).
- Bug form "Ubah Project"; Escape di Combobox; validasi lebih bayar (frontend + backend) + badge "Lebih bayar".
- Transisi status project yang eksplisit; arahkan ke detail setelah membuat project; venue tunggal.
- State loading/error/empty yang dibedakan; label status berbahasa Indonesia; istilah seragam ("Dokumen").
- Aksesibilitas: focus trap Modal, error merah + `aria-invalid`, target sentuh 44px, sidebar mobile `inert`.

**Fase 5 — Portal Klien (G8)** — ✅ **SELESAI 7 Nov 2026** (branch `revamp/flow-ideal`; ADR-0029; diverifikasi unit test, build web, uji di browser (HP dan desktop, peran Pengantin dan Keluarga, server mati) dan review independen)
- Beranda mobile-first dengan tab bawah (Beranda, Jadwal, Pembayaran, Vendor, Lainnya; navigasi atas di layar lebar), hitung mundur termasuk setelah hari H, kartu "Yang perlu Anda lakukan" (merah hanya di sini), tagihan berikutnya + rekening + tombol konfirmasi WhatsApp, progres empat tahap, kabar terbaru, kontak PIC (WhatsApp/telepon).
- Profil Usaha: alamat dan rekening pembayaran (tabel `tenant_business_details`, bank all-or-nothing); `GET /tenants/me/contact` memberi rekening hanya ke staf atau klien ber-`portal.finance.view`.
- Klien membaca jadwal termin (`payment-terms`, label ramah klien); tab Pembayaran memakai sisa tagihan yang benar (Tambahan tidak ikut). Kendala selalu menjawab: apa yang terjadi, apa yang kami lakukan, kapan kabar berikutnya; lencana status klien tidak pernah merah.
- Halaman galat "Gagal memuat / Coba lagi / Hubungi WO" dan "Akun belum terhubung" menggantikan lemparan ke login; PWA (manifest, ikon, service worker hanya untuk halaman offline, panduan pasang termasuk iOS).
- Peran Pengantin vs Keluarga ditegakkan di UI dan API (Keluarga: tanpa tab Pembayaran, tanpa nominal, tanpa rekening).
- "Proyek saya": `GET /dashboard/mine` dan halaman Beranda untuk peran berlingkup proyek (Wedding Planner, Sales).
- **Belum:** ganti kata sandi oleh klien sendiri; "Bagikan ke keluarga"; sakelar "Terlihat klien" per kendala; manifest PWA per WO (nama/ikon masih generik); foto PIC.

**Fase 6 — Platform Owner, Langganan, Marketing (G5, G6, G10)** — ✅ **SELESAI 7 Nov 2026** (branch `revamp/flow-ideal`; ADR-0030; diverifikasi unit test, build web, uji langsung di MySQL sementara dan browser (desktop dan HP) dan review independen; temuan review yang terbukti sudah diperbaiki)
- Platform Console baru: Ringkasan dari `GET /platform/metrics` (MRR, ARPA, konversi trial, churn, pendapatan 30 hari), daftar tenant dengan filter paket/sumber dan ekspor CSV, Detail Tenant 360° (profil + persetujuan pendaftaran, langganan + aktivasi manual, tagihan, pemakaian, pengguna, aktivitas, catatan internal, permintaan data, dukungan), Tagihan, Pengumuman (banner di WO Console), Data & Privasi (permintaan PDP, batas 30 hari), Audit Log dengan filter dan ekspor, Kebijakan Langganan yang bisa diubah tanpa deploy, Tampilan Login (ganti 3 foto). Menu dan tombol mengikuti peran (Super Admin, Billing, Support, Read-only).
- Sesi dukungan baca-saja ("Lihat sebagai tenant"): izin Owner 24 jam yang bisa dicabut, alasan wajib, token 60 menit tanpa refresh, semua non-GET ditolak, banner di WO Console, riwayat terlihat oleh Owner. Token dicek ke sesi, izin, dan admin di setiap permintaan, jadi mengakhiri sesi, mencabut izin, atau menonaktifkan admin menghentikannya seketika. Mengganti email Owner hanya Super Admin.
- Langganan: kode QR di penyiapan 2FA; QRIS dengan hitung mundur, periode baru yang akan dibeli, pesan berbeda untuk kedaluwarsa dan dibatalkan; kuitansi langganan yang bisa dicetak/disimpan sebagai PDF lewat dialog cetak browser.
- Marketing/SEO: judul dan deskripsi per halaman (`usePageMeta`), `robots.txt`, halaman legal dan kredibilitas yang sudah ada tetap dipakai.
- Custom domain tenant dihapus (K15): kolom, API, form, `useDomainBranding()`, injeksi meta; migrasi 62. Migrasi 63 menambah enam tabel konsol.
- **Belum / sengaja ditunda:** versi harga dan entitlement paket (belum ada pelanggan berbayar, K13); MRR ekspansi/kontraksi (belum ada riwayat ganti paket); sitemap (domain produksi belum tercatat); tab Aktivitas tenant hanya untuk Super Admin; pembacaan halaman 360° belum masuk audit; pembayaran di luar gateway masuk MRR tetapi tidak ke angka pendapatan 30 hari; hitung mundur QRIS dan unggah foto login belum diuji dengan gateway/penyimpanan sungguhan.
- Kwitansi/invoice **klien** WO **tidak** termasuk di sini; sudah direncanakan di `docs/plan/invoice-kwitansi-client/`.

**Fase 7 — Skala**
- Pagination server-side di Platform Console & GlobalSearch (hapus `all=true`), tes isolasi tenant otomatis (token tenant A → ID tenant B = 404), indeks komposit `(tenant_id, …)`, presigned URL untuk file.

## 6. Keputusan produk yang dibutuhkan

| # | Keputusan | Status |
|---|---|---|
| K1 | Model coba & lama trial | ✅ **Trial 14 hari** tanpa kartu |
| K2 | Kanal verifikasi pendaftaran | ✅ **Sign in with Google saja** (tanpa email/password untuk Owner baru). WhatsApp diminta di kelengkapan profil |
| K3 | Pendaftaran langsung aktif atau ditinjau tim | ✅ **Langsung aktif**; tim bisa membekukan tenant |
| K4 | Setelah trial/langganan habis | ✅ **Bertahap**: tenggang 7 hari → baca-saja → suspend; portal klien tetap terbaca |
| K5 | Cara masuk staf & klien; akun lama | ✅ Staf **Google saja**; klien **username + sandi yang diatur sendiri lewat undangan**; data lama dummy, **tanpa migrasi** |
| K6 | Batas tiap paket | ✅ **Batas hanya saat masa coba**: 3 staf, 5 proyek aktif, 10 klien. Paket berbayar tanpa batas (paket dibedakan durasi, bukan tingkat). Angka bisa diubah dari Platform Console (diputuskan 6 Nov 2026) |
| K7 | Alamat per WO | Slug disimpan sekarang; subdomain `namawo.elproof…` ditunda |
| K8 | Penyedia email | Ditunda; tidak dibutuhkan selama login WO via Google dan link dikirim via WhatsApp |
| K9 | Akun Google Cloud OAuth | ✅ Client ID `145222647838-e9nhkpkaoiupcd0ume68qg2bqmlhcq58.apps.googleusercontent.com` (publik). Authorized JavaScript origins: `https://elproof.elcodelabs.com`, `http://localhost`, `http://localhost:5173`; redirect URI kosong (alur ID token Google Identity Services, tanpa client secret). Consent screen harus **In production** sebelum WO sungguhan mendaftar |
| K10 | Durasi tiap tahap downgrade | ✅ **Tanpa penghapusan otomatis**: tenggang → baca-saja → suspend, bertahan sampai bayar atau Owner minta hapus. Durasi tenggang/baca-saja diatur di Kebijakan (usulan 7 & 23 hari) |
| K11 | Impersonasi oleh tim ElProof | ✅ **Dengan izin Owner tenant** (24 jam), baca-saja, maks 60 menit, tercatat |
| K12 | Siapa boleh melihat/melakukan apa (staf & klien) | ✅ **RBAC diatur tiap WO**: katalog izin tetap + peran yang bisa diedit/dibuat WO, untuk staf maupun klien (FLOW-IDEAL §2.5) |
| K13 | Perubahan harga paket untuk tenant lama | ✅ Tidak ada pelanggan riil (data dummy) → harga di-snapshot per invoice saja; versi harga ditunda |
| K14 | Peran admin platform | Usulan: Super Admin / Billing / Support / Read-only + 2FA wajib |
| K15 | Custom domain per tenant | ✅ **Dihentikan** — semua tenant & klien memakai `elproof.elcodelabs.com`. Kode custom domain (ADR-0015) dihapus di Fase 6 |

## 7. Risiko & ketidakpastian

- **Tripay menutup pendaftaran merchant baru** (sejak ±27 Agu 2026). ElProof sudah merchant aktif, jadi tidak terdampak. Tripay juga tidak punya recurring, sehingga penagihan tetap manual per periode.
- **Ketergantungan pada Google**: WO tanpa akun Google tidak bisa mendaftar. Ini risiko kecil di Indonesia (mayoritas Android), dan jalur sales-assisted tetap memakai Google. Bila Google OAuth bermasalah, login WO ikut terganggu; admin platform tetap bisa masuk dengan sandi.
- **Undangan staf diteruskan ke orang lain**: dicegah dengan mewajibkan email Google sama dengan email yang diundang.
- **Kewajiban hukum ElProof**: pendaftaran PSE Lingkup Privat di OSS dan dokumen privasi/DPA perlu ditinjau konsultan hukum sebelum 16 Jan 2027.
- **Banyak instance**: reconciler & rate limiter masih in-process (ADR-0011). Aman untuk satu instance; perlu diubah bila di-scale.

## 8. Sumber utama (diakses 5 Nov 2026)

- OWASP ASVS 5.0 V6; OWASP Authentication & Forgot Password Cheat Sheet; NIST SP 800-63B-4
- Microsoft Learn — multitenant identity & domain names; AWS SaaS Lens; Clerk Organizations
- First Page Sage — SaaS free trial conversion benchmarks (Sep 2025)
- honeybook.com/pricing, dubsado.com/pricing, planningpod.com, help.aisleplanner.com
- kledo.com/harga & /daftar, talenta.co/harga, accurate.id/harga, mokapos.com/harga, gatera.id
- developers.facebook.com (WhatsApp pricing), zenziva.id/harga, fonnte.com, resend.com/pricing, aws.amazon.com/ses/pricing, support.google.com/a/answer/81126
- tripay.co.id & /developer, docs.xendit.co/recurring, docs.midtrans.com
- siplawfirm.id (PP 33/2026), teknologi.bisnis.com (PP 33/2026, PSE), NN/g (empty state, confirmation dialog), Cloudflare Turnstile plans
