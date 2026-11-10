# UJI-MANUAL.md — Yang hanya bisa dibuktikan manusia (PLAN Tahap 9)

Semua yang bisa diotomatisasi sudah menjadi gerbang (lihat ADR-0037 dan `.github/workflows/ci.yml`): axe ketat di 68 halaman × 2 viewport, tes perilaku, tes tulis,
pemeriksa istilah, anggaran bundel dan panggilan API. **Axe hanya menangkap sebagian masalah aksesibilitas** (urutan fokus yang masuk akal, kalimat yang dibacakan, dan apakah sebuah label *benar* tidak bisa dinilainya), dan tidak ada tes otomatis yang bisa
menggantikan jari di layar HP, pembaca layar yang berbicara, atau pengguna yang bingung. Dokumen ini adalah naskah untuk tiga hal itu, dan tempat hasilnya dicatat.
Uji kegunaan punya protokol sendiri: [`UJI-KEGUNAAN.md`](UJI-KEGUNAAN.md) (putaran 4 = akhir, tugas sama dengan putaran 1).

## 0. Menyiapkan

```bash
cd apps/web && npm run e2e:stack      # MySQL, MinIO, API, web produksi; sesi login di e2e/.tmp/auth/*.json
npm run e2e:down                      # mematikan dan membuang basis data sementara
```

Data benih dan akun uji ada di `apps/web/e2e/README.md` ("Data benih"). Untuk HP nyata: buka alamat tumpukan dari jaringan yang sama atau pakai lingkungan staging
(`E2E_WEB_PORT` dan `E2E_API_PORT` tetap; origin sesi login ikut port, jadi masuk ulang di perangkat). Tidak ada kata sandi sungguhan di benih.

## 1. Papan ketik saja (tanpa tetikus, tanpa layar sentuh)

Cabut tetikus atau jangan sentuh. Lulus bila **setiap langkah bisa dicapai dan fokus terlihat selalu**, tidak ada perangkap fokus, dan Escape menutup hanya lapisan teratas.

| # | Alur | Langkah kunci | Harus terjadi |
|---|---|---|---|
| K1 | Masuk | Tab ke nama pengguna, kata sandi, tombol mata, "Masuk". Salah sandi lalu benar. | Galat dibaca di dekat kolom dan fokus pindah ke kolom bermasalah; setelah masuk fokus di halaman baru, bukan di `<body>` |
| K2 | Buat proyek | Proyek → "Tambah proyek" → isi semua kolom (tanggal native lewat papan ketik) → tambah termin → simpan | Satu panggilan; kolom salah dikenali, fokus ke yang pertama; sukses menuju halaman proyek |
| K3 | Catat dan verifikasi pembayaran | Tab Keuangan → "Catat pembayaran" → unggah berkas dengan papan ketik → simpan → "Verifikasi" | Drawer menjaga fokus, Escape tertahan saat menyimpan; verifikasi memunculkan cap dan fokus kembali ke pemicu |
| K4 | Lihat bukti | Dokumen → buka penampil → "Ukuran asli" → "Unduh" → Escape | Fokus masuk ke penampil dan kembali ke tombol pembuka |
| K5 | Bayar langganan | Pengaturan → Langganan → pilih paket → bayar | Setiap kontrol terjangkau; pesan status dibaca otomatis |
| K6 | Menu dan pencarian | `Ctrl/Cmd+B`, `Ctrl/Cmd+K`, ketik "Citra", Enter | Menu menciut dan ingat pilihan; palet menemukan dan membuka proyek; Escape mengembalikan fokus |
| K7 | Daftar | Urutkan kolom dengan Enter dan Spasi, filter, halaman berikutnya, pilih baris, aksi massal | `aria-sort` berubah, URL ikut, seleksi diumumkan |

## 2. Pembaca layar

Satu kali penuh per kombinasi: **VoiceOver + Safari (macOS)**, **VoiceOver + Safari (iPhone)**, **NVDA + Firefox atau Chrome (Windows)**, **TalkBack + Chrome (Android)**. Ulangi K1–K5 dengan layar dimatikan atau ditutup.

Lulus bila, untuk setiap alur: pengguna tahu **di mana** ia berada (judul halaman, landmark "Menu utama", "Jejak halaman"), **apa** yang bisa ditekan (setiap ikon punya nama, tidak
ada "tombol" tanpa nama), **apa yang baru terjadi** (toast dibacakan sopan, tanpa mencuri fokus; dialog 409 dibacakan), dan **apa yang salah** (galat kolom terhubung ke kolom).

Daftar periksa tambahan (tandai ✔/✘ dan catat kutipan yang dibacakan):
- Judul halaman berbeda di tiap rute; `h1` tunggal.
- Tabel daftar: kepala kolom bisa diurutkan dibacakan beserta arah urutan; jumlah total dibacakan.
- Grafik tren dibacakan sebagai **tabel data** (bukan gambar bisu); nilai nol tetap dibacakan.
- Dialog: judul dan deskripsi dibacakan saat terbuka; fokus awal masuk akal; setelah ditutup, fokus kembali.
- Cap bukti (`ProofStamp`): dibacakan siapa memverifikasi dan kapan.
- Status berwarna tidak hanya warna: ada teks atau ikon bernama.
- Pesan "Data diubah orang lain" (dua sesi mengubah proyek yang sama): dibacakan dan "Muat ulang" terjangkau.

## 3. Perangkat nyata

Minimal: satu **iPhone** (Safari, iOS terbaru dan satu versi sebelumnya bila ada) dan satu **Android kelas menengah** (Chrome; yang dipakai klien nyata, bukan flagship).

| Periksa | Cara | Lulus bila |
|---|---|---|
| Target sentuh | Jempol satu tangan di Portal dan WO: tombol aksi baris, ikon, mata kata sandi, menu | Tidak ada salah tekan pada aksi yang berdekatan; tombol utama ≥ 44 px; minimal 24 px dengan jarak (WCAG 2.5.8) |
| Tanpa gulir samping | Putar ke tegak dan mendatar di Beranda, Detail Proyek, daftar | Tidak ada gulir horizontal halaman (tabel boleh bergulir di dalam kartunya) |
| Papan ketik layar | Isi form pembayaran dan buat proyek | Kolom aktif tidak tertutup papan ketik; tombol simpan terjangkau |
| Menu | Buka sheet menu, pilih halaman, kembali | Sheet menutup setelah pindah; gulir latar terkunci selama terbuka |
| Banner sesi dukungan | Platform → Tenant → Dukungan → "lihat sebagai" lalu buka layar WO, menu, dialog, toast | Banner **selalu teratas** (z-index 70), tidak tertutup menu, dialog, atau toast; sesi hanya-baca menolak ubah data dengan pesan |
| Unggah bukti | Foto dari kamera, lalu dari berkas | Progres terlihat, bisa dibatalkan, berkas besar dikompres; gagal di tengah tidak menyisakan pembayaran setengah |
| Jaringan buruk | Mode pesawat di tengah simpan; lalu 3G throttle | Pesan galat jelas dan data yang diketik tidak hilang; layar tidak terkunci di keadaan memuat |
| Teks dan kontras | Cahaya matahari, ukuran font sistem besar | Teks tetap terbaca; tata letak tidak patah di font 130% |
| Pasang ke layar utama | Petunjuk pasang di Beranda Portal (`InstallHint`) | Perilakunya jelas bagi klien: bisa ditutup dan tidak mengganggu; **catat apa yang sebenarnya terjadi** (apakah muncul lagi) |
| Performa terasa | Muat Beranda Portal dari data seluler | LCP terasa ≤ 2,5 dtk; catat waktu stopwatch bila meragukan (angka lab ada di `e2e/reports/BASELINE.md`) |

## 4. Lembar hasil

Salin tabel ini ke bagian bawah dokumen (atau berkas bertanggal di folder ini), satu baris per pengujian. Kegagalan **tidak** dihapus dari tabel setelah diperbaiki; tulis nomor komit perbaikannya.

| Tanggal | Penguji | Perangkat dan perangkat lunak | Butir (K1…, S, P-…) | ✔/✘ | Yang terjadi (kutipan) | Perbaikan (komit) |
|---|---|---|---|---|---|---|
| | | | | | | |

## 5. Kapan Tahap 9 benar-benar selesai

Gerbang otomatis hijau **dan** bagian 1 (K1–K7), bagian 2 (empat kombinasi), bagian 3 (iPhone dan Android) terisi tanpa ✘ yang belum diperbaiki, **dan** putaran uji kegunaan 4 selesai dengan
keberhasilan ≥ 90% per tugas (`UJI-KEGUNAAN.md` §7). Sebelum itu catatan Tahap 9 di `PLAN.md` harus tetap menyebut bagian mana yang belum dikerjakan.
