# Panduan Integrasi Payment Gateway ElProof (untuk Aplikasi Eksternal)

> Audiens dokumen ini: **tim teknis (manusia atau AI agent) di luar ElProof** yang ingin membuat
> charge (QRIS/VA/dll.) lewat satu dompet merchant (Tripay) yang sama dipakai ElProof sendiri,
> tanpa pernah menyentuh kredensial Tripay itu sendiri.
>
> **Dokumen ini berdiri sendiri** — semua yang dibutuhkan untuk integrasi yang benar ada di sini,
> tidak ada dokumen lain yang perlu dibaca. Dokumen ini dikirim berpasangan dengan satu file lain:
> collection Postman siap pakai `ElProof-Payment-Gateway.postman_collection.json` (folder
> "Autentikasi", "Pembayaran", dan "Langganan" — untuk berlangganan ElProof sendiri, §6/§7 —
> tinggal isi `appId`/`secret` lalu jalankan).

---

## 1. Konsep dasar

Aplikasi Anda didaftarkan sebagai satu **App eksternal** di ElProof — diberi `appId` (client ID)
dan `secret` sekali, oleh tim ElProof lewat Platform Console. Selanjutnya, alur integrasinya:

```
1. Tukar appId + secret  →  dapat access token (berlaku 1 jam)
2. Pakai token itu untuk: buat charge, cek status charge, lihat daftar kanal pembayaran
3. (opsional, best-effort) Terima notifikasi webhook saat charge lunas
```

Anda **tidak pernah** melihat/menyimpan kredensial Tripay ElProof sendiri — semua transaksi
diproses lewat satu dompet merchant yang sama, diisolasi per App oleh ElProof.

## 2. Sebelum mulai: pendaftaran App

Anda **tidak bisa mendaftarkan diri sendiri**. Hubungi tim ElProof untuk didaftarkan lewat Platform
Console → Manajemen Aplikasi. Anda akan menerima:

| Field | Contoh | Catatan |
|---|---|---|
| `appId` | `app_e135647f4d10` | client ID Anda, boleh dicatat/di-log |
| `secret` | string acak ~43 karakter | **ditampilkan sekali saja, saat itu juga** — tidak bisa dilihat ulang lewat API/Platform Console manapun. Simpan langsung di secret manager Anda. |

Jika `secret` hilang, minta tim ElProof mereset (`Reset Secret` di Platform Console) — secret lama
langsung berhenti berfungsi begitu direset, dan yang baru juga hanya tampil sekali.

**Jangan pernah** menaruh `secret` di kode sisi client (browser/mobile) — ini kredensial
server-to-server, perlakukan seperti API key rahasia.

## 3. Base URL

```
https://elproof.elcodelabs.com/api/v1
```

Semua path di dokumen ini relatif terhadap base URL di atas — hanya ada satu environment
(production), tidak ada sandbox terpisah untuk API ini.

## 4. Autentikasi — `POST /auth/app/token`

Tukar `appId` + `secret` untuk access token. **Tidak ada refresh token** — kalau token
kedaluwarsa, ulangi saja langkah ini dari awal.

```http
POST /auth/app/token
Content-Type: application/json

{
  "appId": "app_e135647f4d10",
  "secret": "I7WNsjJQNlG9w_uyDFvu6qgqncUeT868qJQSP5rt0uM"
}
```

Sukses (`200`):
```json
{
  "success": true,
  "message": "ok",
  "data": {
    "accessToken": "eyJhbGciOi...",
    "tokenType": "Bearer",
    "expiresIn": 3600
  }
}
```

Gagal — `appId`/`secret` salah, atau App sudah dinonaktifkan (`401`):
```json
{ "success": false, "message": "appId atau secret tidak valid", "errors": { "code": "unauthorized" } }
```

**Rate limit: 10 percobaan/menit per alamat IP.** Melebihi itu → `429`, dengan header
`Retry-After` (detik) dan `errors.code = "rate_limited"`. Ini sengaja ketat (mencegah tebak-tebakan
secret) — jangan retry token exchange dalam loop rapat; cache token yang masih valid dan pakai
ulang sampai `expiresIn` habis.

Pakai token di setiap request berikutnya:
```
Authorization: Bearer eyJhbGciOi...
```

## 5. Membuat charge — `POST /external/payments/charges`

```http
POST /external/payments/charges
Authorization: Bearer <access token>
Content-Type: application/json

{
  "orderRef": "INV-2026-0001",
  "amount": 150000,
  "channel": "QRIS",
  "customerName": "Budi Santoso",
  "customerEmail": "budi@example.com",
  "customerPhone": "081234567890"
}
```

| Field | Wajib? | Catatan |
|---|---|---|
| `orderRef` | ya | ID transaksi **milik sistem Anda sendiri**, harus unik selamanya (lihat §9 — idempotensi). Bebas format, disarankan mengandung prefix sistem Anda supaya tidak pernah bentrok dengan App lain. |
| `amount` | ya | Rupiah **utuh**, bukan sen (`150000` = Rp150.000). Harus `> 0`. Tripay sendiri mungkin punya minimum per kanal — lihat §11. |
| `channel` | tidak | Kode kanal Tripay, mis. `QRIS`, `BRIVA`, `BNIVA`. **Default `QRIS`** kalau dikosongkan. Jangan hardcode daftar kanal — ambil dinamis dari §8, karena bergantung konfigurasi merchant ElProof. |
| `customerName`/`customerEmail`/`customerPhone` | tidak | Diteruskan ke Tripay, bisa muncul di halaman pembayaran/struk. |

Sukses (`201`):
```json
{
  "success": true,
  "message": "ok",
  "data": {
    "orderRef": "INV-2026-0001",
    "providerRef": "T1234567890ABCDE",
    "channel": "QRIS",
    "qrImageUrl": "https://tripay.co.id/.../qr.png",
    "amount": 150000,
    "feeAmount": 750,
    "expiresAt": "2026-07-19T22:13:20+07:00",
    "status": "unpaid"
  }
}
```

**Field yang muncul di response bergantung kanal yang dipilih** — `qrImageUrl`, `payCode`, dan
`checkoutUrl` sama-sama `omitempty`; hanya salah satu yang relevan untuk kanal tersebut yang akan
terisi:

| Kanal (tipe) | Field yang terisi | Yang harus Anda lakukan |
|---|---|---|
| QRIS | `qrImageUrl` | Tampilkan gambar QR ke pelanggan untuk discan |
| Virtual Account (BRIVA, BNIVA, dll.) | `payCode` | Tampilkan nomor VA ke pelanggan |
| E-wallet/checkout (kalau tersedia) | `checkoutUrl` | Redirect pelanggan ke URL ini |

`status` awal selalu `unpaid`. Nilai yang mungkin: `unpaid` / `paid` / `expired` / `failed` /
`refund`.

### Duplikat `orderRef` → `409`

```json
{ "success": false, "message": "order_ref \"INV-2026-0001\" sudah pernah dipakai", "errors": { "code": "conflict" } }
```

Ini **aman untuk retry logic Anda** — kalau request pertama gagal di jaringan sebelum Anda sempat
membaca responsnya, ulangi saja dengan `orderRef` yang sama: kalau charge pertama benar-benar
berhasil dibuat, Anda akan dapat `409` (bukan charge kedua yang tidak sengaja) — panggil
`GET .../status` untuk tahu hasil charge yang pertama.

## 6. Katalog paket Anda sendiri — `GET /external/subscription-plans`

Setiap App eksternal punya katalog paket langganannya **sendiri**, terisolasi dari App lain —
paket App lain tidak pernah terlihat lewat endpoint ini, dan sebaliknya.

```http
GET /external/subscription-plans
Authorization: Bearer <access token>
```

Sukses (`200`):
```json
{
  "success": true,
  "message": "ok",
  "data": [
    {
      "id": 7,
      "name": "Paket 1 Tahun",
      "durationMonths": 12,
      "price": 2000000,
      "active": true,
      "features": ["VPS Regular", "Domain Website", "SSL Security"]
    }
  ]
}
```

| Field | Catatan |
|---|---|
| `id` | Dipakai sebagai `planId` saat membuat charge langganan (§7). |
| `active` | Paket **non-aktif tetap dikirim** (bukan disaring) — supaya Anda bisa menandai paket yang sedang dipakai pelanggan meski sudah tidak dijual lagi. Saring sendiri di sisi Anda kalau hanya ingin menampilkan yang aktif. |
| `features` | Rincian paket, ditampilkan apa adanya ke pelanggan Anda. Bisa `[]` (paket tanpa rincian) — tidak pernah `null`. |

**`data: []` bukan galat** — artinya App Anda **belum dipetakan ke paket mana pun**. Ini keadaan
sah (bukan kekurangan sistem — App yang memakai ElProof murni sebagai gateway pembayaran umum,
bukan untuk berlangganan ElProof sendiri, memang tidak perlu punya katalog), hubungi tim ElProof
kalau Anda menyangka seharusnya sudah punya paket.

## 7. Berlangganan ElProof — `POST /external/subscriptions/charges`

**Ini cara yang benar untuk membeli/memperpanjang langganan ElProof** — berbeda dari
`POST /external/payments/charges` (§5), yang untuk transaksi **umum** lewat dompet ElProof dan
tidak menautkan apa pun ke katalog paket. Kalau charge Anda dibuat lewat §5, ElProof tidak bisa
mencatat langganan Anda sama sekali.

```http
POST /external/subscriptions/charges
Authorization: Bearer <access token>
Content-Type: application/json

{
  "orderRef": "SUB-2026-0001",
  "planId": 7,
  "customerName": "Budi Santoso",
  "customerEmail": "budi@example.com",
  "customerPhone": "081234567890",
  "voucherCode": "HEMAT10"
}
```

| Field | Wajib? | Catatan |
|---|---|---|
| `orderRef` | ya | Sama aturannya dengan §5 — unik selamanya milik sistem Anda. |
| `planId` | ya | `id` dari katalog Anda sendiri (§6). **Tidak ada `amount`** — nominal diambil ElProof dari harga paket, Anda tidak bisa menentukannya sendiri. |
| `customerName`/`customerEmail`/`customerPhone` | tidak | Sama seperti §5. |
| `voucherCode` | tidak | Kode voucher potongan harga dari tim ElProof (§7a). Potongan dihitung ElProof; `amount` di respons sudah dipotong. Kosong / tidak dikirim = harga paket penuh. |

`planId` yang bukan milik App Anda (App lain, atau tidak ada) ditolak dengan **`422`**, walaupun
Anda tahu id-nya — endpoint katalog (§6) saja bukan penjaga yang cukup karena `planId` bisa
ditebak:
```json
{ "success": false, "message": "Paket ini tidak tersedia untuk aplikasi Anda", "errors": { "code": "bad_request" } }
```

Sukses (`201`) — bentuk response **identik** dengan §5, jadi parser yang sudah Anda punya untuk
charge biasa tidak perlu diubah:
```json
{
  "success": true,
  "message": "ok",
  "data": {
    "orderRef": "SUB-2026-0001",
    "providerRef": "T1234567890ABCDE",
    "channel": "QRIS",
    "qrImageUrl": "https://tripay.co.id/.../qr.png",
    "amount": 2000000,
    "feeAmount": 10000,
    "expiresAt": "2026-07-19T22:13:20+07:00",
    "status": "unpaid"
  }
}
```

`orderRef` duplikat → `409`, sama persis seperti §5. Mengecek status charge langganan memakai
endpoint **yang sama** dengan §9 — `GET /external/payments/charges/{orderRef}/status` **tidak
dipindahkan**, jadi satu polling/parser status melayani kedua jenis charge.

### 7a. Voucher potongan harga — `POST /external/subscriptions/voucher-check`

Tim ElProof bisa membuat voucher untuk App Anda (potongan persen atau nominal; bisa dibatasi ke satu paket,
masa berlaku, kuota, dan sekali per pelanggan). Sebelum membuat charge, Anda boleh memeriksa kode lebih dulu
untuk menampilkan harga ke pelanggan — **tidak ada yang dipesan** di langkah ini:

```http
POST /external/subscriptions/voucher-check
Authorization: Bearer <access token>
Content-Type: application/json

{ "planId": 7, "voucherCode": "HEMAT10" }
```

```json
{ "success": true, "message": "Voucher dapat dipakai",
  "data": { "voucherCode": "HEMAT10", "planId": 7, "originalAmount": 2000000, "discountAmount": 200000, "finalAmount": 1800000 } }
```

Lalu kirim `voucherCode` yang sama di §7. Charge memeriksa kodenya **lagi** (kuota bisa habis di antara dua
panggilan); pemakaian baru tercatat saat charge dibuat, dan kembali ke kuota bila charge kedaluwarsa/gagal.
Respons §7 lalu membawa `originalAmount`, `discountAmount`, dan `voucherCode`; `amount` = yang ditagih.

Kode yang ditolak (di 7a maupun §7) → **`422`**, tanpa charge, dengan alasan yang bisa Anda tangani di
`errors.reason` dan kalimat siap tampil di `message`:
```json
{ "success": false, "message": "Voucher ini sudah kedaluwarsa (berlaku sampai 31 Desember 2026).",
  "errors": { "code": "bad_request", "reason": "voucher_expired" } }
```

| `reason` | Arti |
|---|---|
| `voucher_not_found` | Kode tidak ada — **atau milik aplikasi lain** (dijawab sama persis) |
| `voucher_inactive` | Dinonaktifkan tim ElProof |
| `voucher_not_started` / `voucher_expired` | Di luar masa berlaku (tanggal WIB) |
| `voucher_exhausted` | Kuota total habis |
| `voucher_already_used` | Voucher sekali-per-pelanggan sudah dipakai App Anda (pelanggan = App Anda, pemilik langganan di ledger ElProof) |
| `voucher_wrong_plan` | Voucher hanya untuk paket lain |
| `voucher_too_large` | Potongan akan menghabiskan seluruh harga — harga akhir tidak pernah Rp0 |

## 8. Melihat kanal yang tersedia — `GET /external/payments/channels`

```http
GET /external/payments/channels
Authorization: Bearer <access token>
```

```json
{
  "success": true,
  "message": "ok",
  "data": [
    { "code": "QRIS", "name": "QRIS", "type": "General", "feeCustomer": 750, "feeMerchant": 0, "iconUrl": "https://...", "active": true }
  ]
}
```

`feeCustomer`/`feeMerchant` di sini adalah **komponen flat saja** — lihat §11 soal skema biaya
persentase yang belum ikut ditampilkan endpoint ini. Panggil endpoint ini untuk menampilkan pilihan
metode bayar ke pelanggan Anda secara dinamis, bukan hardcode.

Respons endpoint ini **di-cache di sisi ElProof selama 5 menit** (bukan selalu memanggil Tripay
langsung) — kalau Anda baru saja mengaktifkan/menonaktifkan kanal di dashboard Tripay, perubahan itu
bisa butuh sampai 5 menit untuk terlihat di sini. Ini bukan bug: kanal jarang berubah, jadi ini
trade-off yang disengaja demi kecepatan. Kalau Anda butuh kepastian instan atas status aktif suatu
kanal, jangan andalkan endpoint ini semata — status charge (§5, §9) selalu live, tidak pernah
di-cache.

## 9. Mengecek status charge — `GET /external/payments/charges/{orderRef}/status`

```http
GET /external/payments/charges/INV-2026-0001/status
Authorization: Bearer <access token>
```

Bentuk response **sama persis** dengan response pembuatan charge (§5), dengan `status` ter-update.

`orderRef` yang bukan milik App Anda (atau tidak pernah ada) → `404`, bukan bocor "ada tapi bukan
milikmu":
```json
{ "success": false, "message": "order_ref tidak ditemukan", "errors": { "code": "not_found" } }
```

**Ini adalah sumber kebenaran paling andal** untuk status charge — pakai polling ke endpoint ini
(bukan cuma mengandalkan webhook di §10) sebagai fallback, terutama kalau webhook Anda tidak pernah
menerima notifikasi (jaringan, server Anda down saat itu, dll).

## 10. Notifikasi webhook (best-effort, bukan satu-satunya sumber kebenaran)

Saat charge Anda berubah status (lunas, kedaluwarsa, gagal, refund), ElProof mencoba mengirim
**satu kali** POST ke `callbackUrl` yang Anda daftarkan — **fire-and-forget**: tidak ada retry
otomatis, timeout 5 detik. Kalau server Anda sedang down/lambat tepat saat itu, pengiriman itu
hilang begitu saja — karena itu §9 (polling status) tetap wajib ada di sistem Anda, webhook ini
murni percepatan, bukan pengganti.

ElProof secara berkala juga memeriksa ulang charge yang belum selesai langsung ke gateway (jaga-jaga
kalau notifikasi awalnya tidak pernah sampai ke ElProof sama sekali), jadi Anda tetap akan menerima
percobaan pengiriman webhook untuk charge yang sempat "terlewat" — tapi ini tidak mengubah saran di
atas: percobaan pengiriman ke server Anda sendiri tetap cuma sekali per notifikasi, tanpa retry.
Polling `GET .../status` (§9) tetap satu-satunya cara yang benar-benar andal.

```http
POST <callbackUrl Anda>
Content-Type: application/json
X-Webhook-Signature: <hex HMAC-SHA256 dari raw body, di-key dengan SECRET Anda sendiri>

{"orderRef":"INV-2026-0001","paid":true,"amount":150000,"paidAt":"2026-07-19T10:15:00+07:00"}
```

- **`X-Webhook-Signature` ditandatangani dengan `secret` yang sama** yang Anda pakai untuk tukar
  token di §4 — bukan secret lain, bukan kredensial Tripay ElProof.
- `paid` cuma boolean — untuk status lengkap (`expired`/`failed`/`refund`), panggil §9.
- **Selalu verifikasi signature sebelum memproses payload** — jangan proses body mentah-mentah.

Contoh verifikasi (Node.js):
```js
const crypto = require("crypto");
function isValidSignature(rawBody, signatureHeader, secret) {
  const expected = crypto.createHmac("sha256", secret).update(rawBody).digest("hex");
  return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signatureHeader));
}
```

Contoh verifikasi (Python):
```python
import hmac, hashlib

def is_valid_signature(raw_body: bytes, signature_header: str, secret: str) -> bool:
    expected = hmac.new(secret.encode(), raw_body, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature_header)
```

Gunakan `raw_body` **sebelum** di-parse JSON — signature dihitung atas byte mentah, bukan hasil
`json.dumps` ulang (yang bisa berbeda urutan/spasi dari body asli).

## 11. Format error & kode yang bisa Anda tangani secara programatik

Semua error mengikuti envelope `{ success: false, message, errors }`. Di keenam endpoint eksternal
ini — `/auth/app/token`, ketiga `/external/payments/*` (§5, §8, §9), dan kedua endpoint langganan
(§6, §7) — `errors.code` **biasanya** ada (endpoint lain ElProof tidak punya field ini sama
sekali) — **kecuali satu kasus penting, lihat peringatan di bawah tabel:**

**`/external/subscription-plans` (§6) dan `/external/subscriptions/charges` (§7) memakai
kosakata `errors.code` yang SAMA PERSIS dengan tabel di bawah** — tidak ada kode baru yang perlu
Anda tangani khusus untuk keduanya, parser galat yang sudah Anda buat untuk `/external/payments/*`
langsung berlaku.

| `errors.code` | Kapan muncul | Contoh HTTP status |
|---|---|---|
| `bad_request` | Body tidak valid / field wajib kosong / gateway ElProof sendiri belum dikonfigurasi / `planId` tidak dipetakan ke App Anda (§7) / metode HTTP salah **pada kelima endpoint eksternal ber-token-App** (metode salah spesifik di `/auth/app/token` sendiri direspons `405` tanpa `errors.code` — kasus langka, seharusnya tidak pernah terjadi kalau Anda selalu mengirim `POST` sesuai §4) | `400` **atau** `422` — lihat catatan di bawah |
| `unauthorized` | `appId`/`secret` salah saat tukar token di `/auth/app/token` | `401` |
| `forbidden` | Token bukan milik App (principal type salah), atau App Anda dinonaktifkan | `403` |
| `not_found` | `orderRef` tidak ada / bukan milik App Anda | `404` |
| `conflict` | `orderRef` duplikat (§5 maupun §7) | `409` |
| `rate_limited` | Melebihi limit `/auth/app/token` | `429` |
| `internal` | Kesalahan tak terduga di sisi ElProof (termasuk error dari Tripay yang belum dipetakan jadi kode spesifik — lihat §13) | `500` |

**Penting #1:** `bad_request` bisa datang dengan status `400` (kesalahan bentuk request, terdeteksi
sebelum memanggil layanan apa pun) **atau** `422` (kegagalan validasi bisnis, mis. gateway ElProof
sedang tidak aktif). Cek `errors.code`, jangan hanya switch berdasarkan status HTTP, untuk
membedakan keduanya.

**Penting #2 — pengecualian nyata, bukan sekadar catatan kaki:** kalau `accessToken` yang Anda
kirim ke salah satu dari **kelima** endpoint ber-token-App (ketiga `/external/payments/*` DAN
kedua endpoint langganan §6/§7 — semuanya diverifikasi lapisan yang sama) **kosong/tidak ada,
salah format, atau sudah kedaluwarsa**, penolakannya terjadi di lapisan verifikasi token yang sama
untuk seluruh ElProof (bukan kode khusus modul pembayaran) — responsnya `401`, **tanpa field
`errors` sama sekali** (bukan `errors: null`, benar-benar tidak ada key-nya):
```json
{ "success": false, "message": "Token tidak valid atau sudah kedaluwarsa" }
```
Ini justru kasus yang **paling sering terjadi** dalam praktik, karena `accessToken` kedaluwarsa tiap
~1 jam (§4). **Untuk mendeteksi token kedaluwarsa di kelima endpoint ini, cek status HTTP
`401` secara langsung — jangan bergantung pada `errors.code` untuk kasus ini, karena field itu bisa
saja tidak ada.** `errors.code = "unauthorized"` hanya muncul untuk kegagalan `appId`/`secret` saat
menukar token di `/auth/app/token` sendiri (§4) — bukan untuk token yang sudah kedaluwarsa di
endpoint lain.

## 12. Ringkasan alur lengkap

Charge umum (§5):
```
1. POST /auth/app/token           { appId, secret }              → simpan accessToken, jam kedaluwarsanya
2. GET  /external/payments/channels                              → (opsional) tampilkan pilihan metode bayar
3. POST /external/payments/charges { orderRef, amount, channel } → tampilkan QR/VA/redirect ke pelanggan
4. (tunggu) terima webhook DAN/ATAU polling berkala:
   GET /external/payments/charges/{orderRef}/status
5. Saat status = "paid" → proses pesanan di sistem Anda
6. Saat accessToken kedaluwarsa (~1 jam) → ulangi langkah 1
```

Berlangganan ElProof (§6/§7) — sama persis, hanya langkah 2-3 yang beda:
```
1. POST /auth/app/token                { appId, secret }         → simpan accessToken
2. GET  /external/subscription-plans                              → ambil planId milik App Anda
   (opsional) POST /external/subscriptions/voucher-check { planId, voucherCode } → tampilkan harga akhir
3. POST /external/subscriptions/charges { orderRef, planId }      → tampilkan QR/VA/redirect ke pelanggan
   (dengan voucher: { orderRef, planId, voucherCode })
4. (tunggu) polling berkala: GET /external/payments/charges/{orderRef}/status — endpoint status
   SAMA dengan charge umum, tidak dipindahkan
5. Saat status = "paid" → langganan Anda tercatat di ElProof sampai tanggal yang dihitung dari planId
```

## 13. Keterbatasan yang perlu diketahui

- **Error dari Tripay sendiri yang belum dipetakan** (mis. kanal tidak didukung untuk nominal
  tersebut, kredensial merchant ElProof bermasalah) saat ini muncul sebagai `500`/`internal` yang
  generik, bukan kode spesifik — kalau Anda mendapat ini berulang untuk kombinasi channel+amount
  yang sama, hubungi tim ElProof untuk investigasi lebih lanjut, jangan retry otomatis tanpa batas.
- `feeCustomer`/`feeMerchant` di §8 baru menampilkan komponen flat Tripay; komponen persentase
  belum diekspos lewat endpoint ini (ada di kontrak internal ElProof, `QuoteFee`, tapi belum
  dipetakan ke `/external/payments/channels`).
- Tidak ada endpoint untuk membatalkan charge yang sudah dibuat — biarkan kedaluwarsa sesuai
  `expiresAt` kalau pelanggan Anda batal membayar.
- Kanal yang tersedia bergantung konfigurasi merchant Tripay ElProof sendiri, bisa berubah
  sewaktu-waktu — selalu ambil dinamis lewat §8, jangan hardcode.

## 14. Kontak

Untuk pendaftaran App baru, reset secret, atau laporan masalah integrasi, hubungi tim ElProof lewat
halaman Kontak di <https://elproof.elcodelabs.com/homepage/kontak>.
