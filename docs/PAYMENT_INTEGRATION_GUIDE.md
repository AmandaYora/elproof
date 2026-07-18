# Panduan Integrasi Payment Gateway ElProof (untuk Aplikasi Eksternal)

> Audiens dokumen ini: **tim teknis (manusia atau AI agent) di luar ElProof** yang ingin membuat
> charge (QRIS/VA/dll.) lewat satu dompet merchant (Tripay) yang sama dipakai ElProof sendiri,
> tanpa pernah menyentuh kredensial Tripay itu sendiri.
>
> **Dokumen ini berdiri sendiri** — semua yang dibutuhkan untuk integrasi yang benar ada di sini,
> tidak ada dokumen lain yang perlu dibaca. Dokumen ini dikirim berpasangan dengan satu file lain:
> collection Postman siap pakai `ElProof-Payment-Gateway.postman_collection.json` (folder
> "Autentikasi" dan "Pembayaran", tinggal isi `appId`/`secret` lalu jalankan).

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
| `orderRef` | ya | ID transaksi **milik sistem Anda sendiri**, harus unik selamanya (lihat §7 — idempotensi). Bebas format, disarankan mengandung prefix sistem Anda supaya tidak pernah bentrok dengan App lain. |
| `amount` | ya | Rupiah **utuh**, bukan sen (`150000` = Rp150.000). Harus `> 0`. Tripay sendiri mungkin punya minimum per kanal — lihat §9. |
| `channel` | tidak | Kode kanal Tripay, mis. `QRIS`, `BRIVA`, `BNIVA`. **Default `QRIS`** kalau dikosongkan. Jangan hardcode daftar kanal — ambil dinamis dari §6, karena bergantung konfigurasi merchant ElProof. |
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

## 6. Melihat kanal yang tersedia — `GET /external/payments/channels`

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

`feeCustomer`/`feeMerchant` di sini adalah **komponen flat saja** — lihat §9 soal skema biaya
persentase yang belum ikut ditampilkan endpoint ini. Panggil endpoint ini untuk menampilkan pilihan
metode bayar ke pelanggan Anda secara dinamis, bukan hardcode.

## 7. Mengecek status charge — `GET /external/payments/charges/{orderRef}/status`

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
(bukan cuma mengandalkan webhook di §8) sebagai fallback, terutama kalau webhook Anda tidak pernah
menerima notifikasi (jaringan, server Anda down saat itu, dll).

## 8. Notifikasi webhook (best-effort, bukan satu-satunya sumber kebenaran)

Saat charge Anda berubah status di Tripay (lunas, kedaluwarsa, gagal, refund), ElProof mencoba
mengirim **satu kali** POST ke `callbackUrl` yang Anda daftarkan — **fire-and-forget**: tidak ada
retry otomatis, timeout 5 detik. Kalau server Anda sedang down/lambat saat itu, relay hilang begitu
saja — karena itu §7 (polling status) tetap wajib ada di sistem Anda, webhook ini murni percepatan.

```http
POST <callbackUrl Anda>
Content-Type: application/json
X-Webhook-Signature: <hex HMAC-SHA256 dari raw body, di-key dengan SECRET Anda sendiri>

{"orderRef":"INV-2026-0001","paid":true,"amount":150000,"paidAt":"2026-07-19T10:15:00+07:00"}
```

- **`X-Webhook-Signature` ditandatangani dengan `secret` yang sama** yang Anda pakai untuk tukar
  token di §4 — bukan secret lain, bukan kredensial Tripay ElProof.
- `paid` cuma boolean — untuk status lengkap (`expired`/`failed`/`refund`), panggil §7.
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

## 9. Format error & kode yang bisa Anda tangani secara programatik

Semua error mengikuti envelope `{ success: false, message, errors }`, dengan **tambahan** field
`errors.code` khusus di keempat endpoint eksternal ini (endpoint lain ElProof tidak punya field
ini):

| `errors.code` | Kapan muncul | Contoh HTTP status |
|---|---|---|
| `bad_request` | Body tidak valid / field wajib kosong / gateway ElProof sendiri belum dikonfigurasi | `400` **atau** `422` — lihat catatan di bawah |
| `unauthorized` | `appId`/`secret` salah, atau token tidak valid/kedaluwarsa | `401` |
| `forbidden` | Token bukan milik App (principal type salah), atau App Anda dinonaktifkan | `403` |
| `not_found` | `orderRef` tidak ada / bukan milik App Anda | `404` |
| `conflict` | `orderRef` duplikat | `409` |
| `rate_limited` | Melebihi limit `/auth/app/token` | `429` |
| `internal` | Kesalahan tak terduga di sisi ElProof (termasuk error dari Tripay yang belum dipetakan jadi kode spesifik — lihat §11) | `500` |

**Penting:** `bad_request` bisa datang dengan status `400` (kesalahan bentuk request, terdeteksi
sebelum memanggil layanan apa pun) **atau** `422` (kegagalan validasi bisnis, mis. gateway ElProof
sedang tidak aktif). **Cek `errors.code`, jangan hanya switch berdasarkan status HTTP**, untuk
membedakan kelas kegagalan secara andal.

## 10. Ringkasan alur lengkap

```
1. POST /auth/app/token           { appId, secret }              → simpan accessToken, jam kedaluwarsanya
2. GET  /external/payments/channels                              → (opsional) tampilkan pilihan metode bayar
3. POST /external/payments/charges { orderRef, amount, channel } → tampilkan QR/VA/redirect ke pelanggan
4. (tunggu) terima webhook DAN/ATAU polling berkala:
   GET /external/payments/charges/{orderRef}/status
5. Saat status = "paid" → proses pesanan di sistem Anda
6. Saat accessToken kedaluwarsa (~1 jam) → ulangi langkah 1
```

## 11. Keterbatasan yang perlu diketahui

- **Error dari Tripay sendiri yang belum dipetakan** (mis. kanal tidak didukung untuk nominal
  tersebut, kredensial merchant ElProof bermasalah) saat ini muncul sebagai `500`/`internal` yang
  generik, bukan kode spesifik — kalau Anda mendapat ini berulang untuk kombinasi channel+amount
  yang sama, hubungi tim ElProof untuk investigasi lebih lanjut, jangan retry otomatis tanpa batas.
- `feeCustomer`/`feeMerchant` di §6 baru menampilkan komponen flat Tripay; komponen persentase
  belum diekspos lewat endpoint ini (ada di kontrak internal ElProof, `QuoteFee`, tapi belum
  dipetakan ke `/external/payments/channels`).
- Tidak ada endpoint untuk membatalkan charge yang sudah dibuat — biarkan kedaluwarsa sesuai
  `expiresAt` kalau pelanggan Anda batal membayar.
- Kanal yang tersedia bergantung konfigurasi merchant Tripay ElProof sendiri, bisa berubah
  sewaktu-waktu — selalu ambil dinamis lewat §6, jangan hardcode.

## 12. Kontak

Untuk pendaftaran App baru, reset secret, atau laporan masalah integrasi, hubungi tim ElProof lewat
halaman Kontak di <https://elproof.elcodelabs.com/homepage/kontak>.
