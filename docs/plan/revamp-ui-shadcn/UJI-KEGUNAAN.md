# UJI-KEGUNAAN.md — Protokol uji kegunaan revamp UI

> Dibuat 9 Nov 2026 untuk `PLAN.md` §7. **Pengujian ini harus dijalankan oleh manusia dengan pengguna nyata**; kode
> tidak bisa menggantikannya. Putaran 1 (garis dasar) dilakukan **sebelum** perubahan visual pertama dirilis ke
> pengguna, yaitu pada versi aplikasi yang sekarang berjalan (cabang `revamp/flow-ideal`).

## 1. Putaran dan tujuan

| Putaran | Kapan | Yang diuji | Tujuan |
|---|---|---|---|
| 1 Garis dasar | Tahap 0 | WO Console + Portal Klien versi sekarang | angka pembanding |
| 2 | Akhir Tahap 3 | Portal Klien baru | memastikan janji merek (jelas, cepat, bukti) |
| 3 | Akhir Tahap 6 | WO Console baru | memastikan alur kerja harian |
| 4 Akhir | Tahap 9 | Keduanya | membandingkan dengan garis dasar |

**Target akhir** (PLAN §2): keberhasilan ≥ 90% **per tugas** (bukan rata-rata), waktu dan salah langkah turun dari garis
dasar, SEQ ≥ 6 dari 7 pada tugas Portal.

## 2. Peserta

- **5 staf WO** per putaran: campuran Owner/Admin dan Wedding Planner/Sales; usahakan ≥ 2 yang bukan pengguna harian
  ElProof. 5 orang cukup untuk menemukan sebagian besar masalah kualitatif; baca angka **per tugas**, jangan rata-rata.
- **5 klien** per putaran (pengantin atau keluarga) memakai **HP Android mereka sendiri**, sinyal apa adanya.
- Jangan memakai tim pengembang atau staf ElProof sebagai peserta. Catat umur, perangkat, dan seberapa sering memakai aplikasi
  serupa. Beri imbalan kecil yang pantas. Minta izin merekam layar dan suara; data hanya untuk perbaikan produk.

## 3. Persiapan (sekali per putaran)

1. Workspace uji berisi data realistis tetapi **fiktif**: 3 proyek (satu punya termin terlambat, satu selesai, satu draf),
   vendor dan venue, satu klien dengan akun masuk. Pakai kunci benih yang sama tiap putaran supaya adil.
2. Akun uji per peserta; kata sandi sementara; HP peserta sudah bisa membuka alamat aplikasi.
3. Alat rekam: rekam layar + suara (HP: perekam layar bawaan; laptop: QuickTime). Stopwatch. Lembar catat (§6).
4. Fasilitator **tidak memberi petunjuk**. Pengamat tambahan boleh mencatat, tidak boleh berbicara.

## 4. Naskah fasilitator

> "Kami menguji aplikasinya, bukan Anda. Tidak ada jawaban salah. Mohon bicara keras apa yang Anda pikirkan. Saya akan
> memberi beberapa tugas; kalau buntu, bilang saja, dan kita lanjut. Boleh berhenti kapan saja."

Per tugas: bacakan tugas **persis** seperti di §5, mulai stopwatch saat peserta mulai, berhenti saat peserta menyatakan
selesai atau menyerah (batas 5 menit untuk staf, 3 menit untuk klien). Setelah tiap tugas tanyakan SEQ:
"Dari 1 (sangat sulit) sampai 7 (sangat mudah), seberapa mudah tugas tadi?" lalu "Apa yang membingungkan?"

## 5. Tugas tetap

**Staf WO (WO Console)**

| # | Tugas (dibacakan) | Berhasil bila |
|---|---|---|
| W1 | "Buat proyek pernikahan baru untuk Citra dan Bima tanggal 20 Desember 2026, nilai kontrak Rp50.000.000, dan atur jadwal pembayaran DP 30% dan pelunasan." | proyek tersimpan dengan termin yang jumlahnya = nilai kontrak |
| W2 | "Catat bahwa klien sudah membayar DP Rp15.000.000 hari ini, lengkap dengan foto bukti transfernya." | pembayaran tersimpan, bukti terlampir, sisa tagihan benar |
| W3 | "Cari proyek mana saja yang tagihannya sudah terlambat." | menyebut proyek yang benar tanpa menghitung manual |
| W4 | "Undang klien proyek ini lewat WhatsApp agar bisa melihat perkembangannya." | tautan undangan terkirim/disalin dengan benar |
| W5 | "Ada masalah dengan vendor katering. Catat kendalanya dan pastikan klien bisa melihatnya." | kendala tersimpan dan terlihat klien |

**Klien (Portal, di HP)**

| # | Tugas (dibacakan) | Berhasil bila |
|---|---|---|
| K1 | "Berapa yang harus Anda bayar berikutnya dan kapan batasnya?" | menyebut nominal dan tanggal yang benar |
| K2 | "Ke rekening mana Anda harus mentransfer?" | menemukan rekening yang benar |
| K3 | "Apakah DP Anda sudah diterima WO? Tunjukkan buktinya." | menemukan status "diterima" dan membuka bukti |
| K4 | "Apa yang akan terjadi minggu depan dalam persiapan pernikahan Anda?" | menyebut tahapan yang benar |
| K5 | "Anda ingin bertanya sesuatu kepada penanggung jawab dari WO. Hubungi dia." | membuka WhatsApp/telepon ke PIC |

## 6. Lembar catat (satu baris per peserta per tugas)

Salin ke spreadsheet dengan kolom berikut (CSV siap pakai di `uji-kegunaan-lembar.csv` di folder ini).

`putaran, peserta, peran, perangkat, tugas, berhasil (ya/tidak/dibantu), detik, salah_langkah, seq (1-7), kutipan_kebingungan, catatan`

- **Salah langkah**: tiap klik/ketukan ke tempat yang tidak membawa ke tujuan, atau tiap galat yang dilihat peserta.
- **Dibantu** dihitung gagal. Berhenti karena batas waktu = gagal.
- Catat juga: hal yang membuat peserta ragu, kata yang tidak dipahami, tombol yang tidak ditemukan.

## 7. Cara membaca hasil

- Hitung per tugas: tingkat keberhasilan (n berhasil dari 5), median detik, rata-rata salah langkah, rata-rata SEQ.
- Kumpulkan kutipan kebingungan, kelompokkan per layar. Temuan yang muncul pada ≥ 2 peserta jadi perbaikan wajib.
- Bandingkan dengan putaran 1. Tulis ringkasan satu halaman di `docs/plan/revamp-ui-shadcn/hasil-putaran-N.md`:
  tabel per tugas, tiga temuan terbesar, keputusan (perbaiki sekarang/ditunda/diterima).
- Catatan metode: sampel 5 orang bukan bukti statistik; ia menemukan masalah, bukan membuktikan persentase. Angka
  "≥ 90%" dibaca sebagai "tidak ada tugas yang gagal pada lebih dari satu peserta".
