import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { COMPANY, CONTACT } from "@/modules/homepage/data/contact";
import { TERMS_UPDATED } from "@/modules/homepage/data/legal";
import { usePageMeta } from "@/shared/hooks/usePageMeta";
import { PUBLIC_PAGES } from "@/app/routes/public-pages";

// Words as in the app and in Syarat dan ketentuan §1 (QA FQ-09): workspace, Owner, anggota tim, klien, proyek, Portal Klien.
const SECTIONS: LegalSection[] = [
  {
    id: "peran-kami",
    title: "1. Peran ElProof",
    body: [
      `ElProof diselenggarakan oleh ${COMPANY.name} (${COMPANY.brand}), ${CONTACT.address} ("kami"). Untuk data akun Owner dan anggota tim, kami adalah pengendali data. Untuk data klien, vendor, dan isi proyek yang dimasukkan WO, WO adalah pengendali data dan kami memprosesnya atas nama WO sesuai Perjanjian pemrosesan data. Kebijakan ini disusun mengikuti Undang-Undang Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi.`,
      `Pertanyaan dan permintaan tentang pelindungan data pribadi dapat dikirim ke ${COMPANY.privacyEmail} atau WhatsApp ${CONTACT.phones[0].display}.`,
    ],
  },
  {
    id: "data-dikumpulkan",
    title: "2. Data yang kami kumpulkan",
    body: [
      `Saat mendaftar dengan Google: nama dan alamat email akun Google Anda, serta pengenal akun Google untuk proses masuk berikutnya. Kami tidak menerima kata sandi Google Anda dan tidak mengakses isi Gmail, Drive, atau kontak Anda.`,
      `Profil usaha: nama WO, nomor WhatsApp, kota, dan logo yang Anda isi.`,
      `Bukti persetujuan: versi Syarat dan ketentuan yang disetujui, waktu persetujuan, alamat IP, dan jenis peramban saat mendaftar.`,
      `Data operasional yang dimasukkan WO: proyek pernikahan, klien, vendor, jadwal, pembayaran, kendala, serta dokumen dan foto bukti kerja, termasuk bukti transfer yang dikirim klien lewat Portal Klien.`,
      `Data langganan: paket, periode aktif, dan riwayat pembayaran. Data kartu atau rekening diproses oleh mitra pembayaran pihak ketiga dan tidak kami simpan.`,
    ],
  },
  {
    id: "penggunaan-data",
    title: "3. Tujuan dan dasar pemrosesan",
    body: [
      `Menjalankan layanan sesuai perjanjian dengan Anda: membuat dan mengamankan akun, mencatat data proyek, dan menampilkan progres kepada anggota tim dan klien sesuai perannya.`,
      `Komunikasi tentang akun, masa coba, dan langganan.`,
      `Kabar fitur dan promo hanya bila Anda mencentangnya saat mendaftar. Persetujuan ini dapat ditarik kapan saja melalui halaman Kontak.`,
      `Menjaga keamanan layanan dan memenuhi kewajiban hukum, termasuk penelusuran gangguan dan pencegahan penyalahgunaan.`,
    ],
  },
  {
    id: "pembagian-data",
    title: "4. Pembagian data",
    body: [
      `Kami tidak menjual data pribadi. Data dibagikan hanya kepada penyedia yang membantu menjalankan layanan: penyedia server dan penyimpanan objek, mitra pembayaran, dan Google untuk proses masuk. Mereka terikat kewajiban kerahasiaan.`,
      `Data dapat diungkapkan bila diwajibkan oleh ketentuan hukum yang berlaku.`,
    ],
  },
  {
    id: "penyimpanan-data",
    title: "5. Penyimpanan dan retensi",
    body: [
      `Data disimpan selama workspace ada. Data tidak dihapus otomatis karena langganan berakhir; Owner dapat meminta ekspor atau penghapusan data workspace melalui halaman Kontak. Bukti persetujuan disimpan selama diperlukan untuk membuktikan persetujuan tersebut.`,
    ],
  },
  {
    id: "keamanan-akses",
    title: "6. Keamanan",
    body: [
      `Akses dipisahkan per workspace dan per peran. Koneksi dienkripsi dengan HTTPS. Admin ElProof wajib memakai verifikasi dua langkah dan setiap tindakan admin dicatat. Bila terjadi kegagalan pelindungan data yang berdampak pada Anda, kami memberi tahu Anda dan otoritas terkait sesuai ketentuan hukum.`,
    ],
  },
  {
    id: "hak-atas-data",
    title: "7. Hak Anda",
    body: [
      `Anda berhak meminta akses, salinan, perbaikan, dan penghapusan data pribadi Anda, menarik persetujuan, serta mengajukan keberatan atas pemrosesan tertentu. Owner dan anggota tim dapat menghubungi kami melalui halaman Kontak atau alamat pelindungan data di atas.`,
      `Klien dan vendor mengajukan permintaan tersebut kepada WO yang mengelola datanya, karena WO adalah pengendali data tersebut. Kami membantu WO memenuhi permintaan itu.`,
    ],
  },
  {
    id: "cookie",
    title: "8. Penyimpanan sesi",
    body: [
      `ElProof hanya menyimpan data sesi yang diperlukan agar Anda tetap masuk. Tombol Google di halaman Masuk dan Daftar (mis. "Lanjutkan dengan Google") dimuat dari layanan Google dan tunduk pada kebijakan privasi Google. ElProof tidak memakai cookie iklan atau pelacak pemasaran.`,
    ],
  },
  {
    id: "perubahan-kebijakan",
    title: "9. Perubahan kebijakan",
    body: [
      `Kebijakan ini dapat diperbarui mengikuti perkembangan layanan. Perubahan material diberitahukan melalui WO Console sebelum berlaku.`,
    ],
  },
];

export default function PrivacyPage() {
  usePageMeta(PUBLIC_PAGES.privacy);
  return <LegalDocument title={PUBLIC_PAGES.privacy.title} updated={TERMS_UPDATED} sections={SECTIONS} />;
}
