# Riset Portal Klien — apakah sudah profesional dan modern?

Tech Lead · 2026-11-16 · Mengikuti skill `rudis-riset-desain` (langkah 1–3). Belum ada kode yang diubah. Bahan: potret e2e
Portal Klien terbaru (HP 375 px dan desktop 1280 px, data contoh, palet navy) di `riset/portal-sekarang-*.webp`, kode
`apps/web/src/modules/client-portal`, dan riset luar (sumber di akhir; lembar kontak pembanding di `riset/ref-portal.webp`).

## 1. Kasus

- **Pengguna:** pasangan pengantin (dan perwakilan keluarga), hampir selalu di HP, membuka dari tautan WhatsApp yang dikirim
  WO; jarang membuka, tetapi saat membuka ingin jawaban cepat.
- **Pertanyaan yang harus terjawab dalam 5 detik:** "Persiapan sampai mana? Apakah ada yang harus saya lakukan atau bayar?
  Pembayaran saya sudah tercatat? Ada masalah? Bagaimana menghubungi WO?"
- **Peran portal bagi WO:** alat kepercayaan dan nilai jual (lihat copy situs publik "Klien lebih percaya, penjualan ikut naik").

## 2. Yang sudah baik (pertahankan)

| Aspek | Bukti |
|---|---|
| Mobile-first yang benar | Bilah bawah 5 tujuan berlabel (Beranda, Jadwal, Pembayaran, Vendor, Lainnya), sesuai pedoman 3–5 item dengan label selalu terlihat [material-nav][uxplanet-tab] |
| Hero yang emosional dan jelas | Nama pasangan + hitung mundur "H-118" + tanggal dan venue; satu-satunya blok gelap di halaman |
| "Yang perlu Anda lakukan" | Daftar tindakan di atas (tagihan terlambat, kendala yang sedang ditangani) — pola dashboard portal Aisle Planner ("tasks, deadlines, next steps at a glance") [aisle-portal] |
| Transparansi kendala | "Apa yang terjadi / Yang sedang kami lakukan / Kabar berikutnya" — jarang ada di portal pesaing, kuat untuk kepercayaan |
| Bukti dan verifikasi | Segel emas "Terverifikasi … oleh Rina" di pembayaran dan tahapan; biaya vendor tidak pernah tampil |
| Kontrol WO | Hanya yang dipilih WO yang tampil (dokumen, kendala), sama seperti kontrol akses portal HoneyBook/Aisle Planner [honeybook-portal][aisle-portal] |
| Aksesibilitas & warna | Lolos axe ketat; palet navy + emas; status selalu ikon + teks; bisa dipasang di layar utama (PWA) |

**Kesimpulan singkat:** dasarnya **sudah profesional** dan lebih transparan daripada banyak pesaing. Yang membuatnya belum
terasa **modern dan hangat** adalah rupa beberapa layar dan dua kemampuan yang di portal sejenis sudah dianggap wajar.

## 3. Temuan → rekomendasi

Prioritas: **P1** paling terasa bagi pasangan · **P2** kerapian/kehangatan · **P3** fitur berikutnya (perlu keputusan).

| # | Temuan | Rekomendasi | Prio |
|---|---|---|---|
| 1 | **Menghubungi WO tersembunyi.** Kartu "Hubungi kami" (WhatsApp) ada di bawah Beranda, di bawah lipatan, dan hilang bila WO belum mengisi nomor; tagihan terlambat hanya berkata "hubungi PIC" tanpa tombol | Tombol WhatsApp PIC selalu terjangkau: ikon di header + tombol di setiap item "perlu dilakukan" dengan pesan terisi sesuai konteks ("Halo, soal Termin 2…"). Bila nomor kosong: arahkan ke nomor usaha, dan di WO Console beri pengingat "lengkapi WhatsApp" (sudah ada di "Mulai di sini") | P1 |
| 2 | **Ringkasan Pembayaran memakan satu layar penuh:** 3 kartu bertumpuk (Nilai kontrak, Sudah dibayar hijau, Sisa tagihan oranye). Warna hijau/oranye dipakai untuk angka biasa, padahal oranye = peringatan | Satu kartu: "Sudah dibayar Rp45 jt dari Rp150 jt" + bilah progres tersegmen per termin + "Berikutnya: Termin 2, 25 Okt" + tombol rekening/WhatsApp; warna status hanya bila memang terlambat [pay-progress] | P1 |
| 3 | **Rekening pembayaran sering kosong** ("belum dibagikan") sehingga pasangan tidak tahu ke mana membayar | Saat kosong: tombol "Minta nomor rekening lewat WhatsApp" (bukan teks pasif); di WO Console, Beranda menandai "Rekening belum diisi" sebagai tindakan | P1 |
| 4 | **Hero terasa berat dan seragam** (blok navy penuh, tanpa sentuhan personal); pesaing konsumen (Zola) dan portal Aisle Planner memakai foto dan nuansa hangat [zola][aisle-portal] | Foto sampul opsional per proyek (diunggah WO) di belakang hero dengan lapisan navy hanya di bawah teks (pola layar masuk); tanpa foto: navy + motif segel emas tipis. Tetap satu hero per halaman | P2 |
| 5 | **Jadwal berupa daftar panjang** "Sedang dan akan berjalan / Sudah selesai" | Kelompokkan per fase waktu seperti checklist Zola ("Bulan ini", "1–3 bulan lagi", "Menjelang hari H") dengan H-N di tiap baris; tahapan selesai diringkas (bisa dibuka) [zola] | P2 |
| 6 | **Desktop hanya satu kolom sempit** (~580 px) di layar lebar | ≥ 1024 px: dua kolom (kiri: tindakan + tagihan; kanan: progres + kabar terbaru + kontak), bilah atas tetap | P2 |
| 7 | **Kendala hanya di "Lainnya"** tanpa penanda jumlah | Lencana jumlah kendala terbuka di tab Lainnya; item kendala tetap muncul di "Yang perlu Anda lakukan" (sudah) | P2 |
| 8 | **Konsistensi kecil:** judul halaman Dokumen memakai ikon di depan (halaman lain tidak); judul kartu 15 px; label "Lewat jatuh tempo — hubungi PIC" berupa teks oranye tanpa aksi | Samakan header halaman (judul 20/600 tanpa ikon), judul kartu 16/600 (DESIGN.md §4.5), aksi selalu berupa tombol | P2 |
| 9 | **Tidak ada percakapan di dalam portal**; HoneyBook dan Aisle Planner punya pesan/komentar per item dan notifikasi [honeybook-portal][aisle-portal] | Keputusan produk: WhatsApp sudah jadi kebiasaan di Indonesia, jadi rekomendasi **tidak** membangun chat; cukup tombol WhatsApp kontekstual (#1). Nanti: notifikasi PWA untuk tagihan jatuh tempo dan kabar baru | P3 |
| 10 | **Pasangan tidak bisa mengirim bukti transfer sendiri** (WO yang mencatat); HoneyBook memungkinkan klien membayar tagihan di portal [honeybook-portal][fast-io] | Keputusan produk: "Kirim bukti transfer" dari portal → masuk antrean "Bukti menunggu verifikasi" di Keuangan WO (sudah ada) → WO memverifikasi. Mengurangi kerja admin WO dan mempercepat segel | P3 |
| 11 | **Tidak ada tugas untuk pasangan** (mis. "kirim daftar tamu", "pilih menu tasting"); Aisle Planner menugaskan item ke klien [aisle-portal] | Keputusan produk: tahapan dengan penanda "untuk klien" yang tampil di "Yang perlu Anda lakukan" dan bisa ditandai selesai oleh pasangan | P3 |

## 4. Wireframe (HP)

```
Beranda (HP)                              Pembayaran (HP)
┌─────────────────────────┐               ┌─────────────────────────┐
│ [logo] WO Contoh   [WA] │ ← WA PIC       │ Pembayaran         [WA] │
│ ┌─────────────────────┐ │               │ ┌─────────────────────┐ │
│ │ (foto sampul/navy)  │ │               │ │ Sudah dibayar        │ │
│ │ 14 Mar 2027 · Venue │ │               │ │ Rp45 jt dari Rp150 jt│ │
│ │ Citra & Bima        │ │               │ │ ▇▇▇▁▁▁▁▁▁ (3 termin) │ │
│ │ H-118               │ │               │ │ Berikutnya: Termin 2 │ │
│ └─────────────────────┘ │               │ │ 25 Okt · terlambat ⚠ │ │
│ Yang perlu Anda lakukan │               │ │ [Lihat rekening][WA] │ │
│ ⚠ Termin 2 terlambat    │               │ └─────────────────────┘ │
│   [Tanya lewat WhatsApp]│               │ Jadwal pembayaran …     │
│ ◎ 1 kendala ditangani › │               │ Riwayat (segel emas) …  │
│ Progres persiapan …     │               └─────────────────────────┘
│ Kabar terbaru …         │
│ [Beranda Jadwal Bayar Vendor Lainnya(1)]│
└─────────────────────────┘
```

## 5. Penilaian akhir

- **Profesional:** ya — struktur, transparansi kendala, bukti terverifikasi, dan aksesibilitasnya di atas rata-rata portal sejenis.
- **Modern:** sebagian — rupa dasar sudah satu sistem dengan WO Console, tetapi ringkasan pembayaran, jadwal, hero, dan tata
  letak desktop masih terasa "daftar kartu"; kontak WO yang tersembunyi adalah celah terbesar untuk pasangan.
- **Usulan tahapan:** **A** (#1, #2, #3, #7, #8 — cepat, tanpa API baru kecuali tidak ada) → **B** (#4 foto sampul butuh
  unggah di WO Console + kolom proyek, #5, #6) → **C** (#9–#11, keputusan produk dan API baru).

## Sumber (diakses 16 Nov 2026)

- [aisle-portal] Aisle Planner, "A Portal Your Clients Will Love" — https://aisleplanner.com/products/client-portal/
- [honeybook-portal] HoneyBook Help, what clients can see and do in the client portal — https://help.honeybook.com/en/articles/6428603
- [fast-io] Fast.io, HoneyBook client portal features & limits (pihak ketiga) — https://fast.io/resources/honeybook-client-portal/
- [zola] Zola, wedding checklist (fase waktu) — https://www.zola.com/wedding-planning/checklist
- [material-nav] Material Design, bottom navigation (3–5 tujuan) — https://material.io/components/bottom-navigation
- [uxplanet-tab] UX Planet, bottom tab bar best practices (label selalu terlihat) — https://uxplanet.org/bottom-tab-bar-navigation-design-best-practices-48d46a3b0c36
- [pay-progress] Contoh pola ringkasan cicilan (kartu + progres + jadwal) — https://www.shadcn.io/blocks/checkout-installments ; https://kombai.com/gallery/inspirations/card-ios-11/
- Praktik portal klien umum (vendor blog, arah saja): https://assembly.com/blog/client-portal-best-practices ; https://www.jetadmin.io/blog/client-portal-design-how-to-build-a-secure-intuitive-portal-in-2026/
