import { ShieldAlert } from "lucide-react";
import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { TERMS_UPDATED } from "@/modules/homepage/data/legal";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const SECTIONS: LegalSection[] = [
  {
    id: "peran-kami",
    title: "1. Peran ElProof",
    body: [
      `Untuk data akun Owner dan Staff, ElProof adalah pengendali data. Untuk data Client, vendor, dan isi project yang dimasukkan WO, WO adalah pengendali data dan ElProof memprosesnya atas nama WO sesuai Perjanjian Pemrosesan Data. Kebijakan ini disusun mengikuti Undang-Undang Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi.`,
    ],
  },
  {
    id: "data-dikumpulkan",
    title: "2. Data yang Kami Kumpulkan",
    body: [
      `Saat mendaftar dengan Google: nama dan alamat email akun Google Anda, serta pengenal akun Google untuk proses masuk berikutnya. ElProof tidak menerima kata sandi Google Anda dan tidak mengakses isi Gmail, Drive, atau kontak Anda.`,
      `Profil usaha: nama WO, nomor WhatsApp, kota, dan logo yang Anda isi.`,
      `Bukti persetujuan: versi Syarat & Ketentuan yang disetujui, waktu persetujuan, alamat IP, dan jenis peramban saat mendaftar.`,
      `Data operasional yang dimasukkan WO: project pernikahan, client, vendor, timeline, pembayaran, kendala, serta dokumen dan foto bukti kerja.`,
      `Data langganan: paket, periode aktif, dan riwayat pembayaran. Data kartu atau rekening diproses oleh mitra payment gateway dan tidak disimpan ElProof.`,
    ],
  },
  {
    id: "penggunaan-data",
    title: "3. Tujuan dan Dasar Pemrosesan",
    body: [
      `Menjalankan layanan sesuai perjanjian dengan Anda: membuat dan mengamankan akun, mencatat data project, dan menampilkan progress kepada Staff dan Client sesuai perannya.`,
      `Komunikasi tentang akun, masa coba, dan langganan.`,
      `Kabar fitur dan promo hanya bila Anda mencentangnya saat mendaftar. Persetujuan ini dapat ditarik kapan saja melalui halaman Kontak.`,
      `Menjaga keamanan layanan dan memenuhi kewajiban hukum, termasuk penelusuran gangguan dan pencegahan penyalahgunaan.`,
    ],
  },
  {
    id: "pembagian-data",
    title: "4. Pembagian Data",
    body: [
      `ElProof tidak menjual data pribadi. Data dibagikan hanya kepada penyedia yang membantu menjalankan layanan: penyedia server dan penyimpanan objek, mitra payment gateway, dan Google untuk proses masuk. Mereka terikat kewajiban kerahasiaan.`,
      `Data dapat diungkapkan bila diwajibkan oleh ketentuan hukum yang berlaku.`,
    ],
  },
  {
    id: "penyimpanan-data",
    title: "5. Penyimpanan dan Retensi",
    body: [
      `Data disimpan selama Workspace ada. Data tidak dihapus otomatis karena langganan berakhir; Owner dapat meminta ekspor atau penghapusan data Workspace melalui halaman Kontak. Bukti persetujuan disimpan selama diperlukan untuk membuktikan persetujuan tersebut.`,
    ],
  },
  {
    id: "keamanan-akses",
    title: "6. Keamanan",
    body: [
      `Akses dipisahkan per Tenant dan per peran. Koneksi dienkripsi dengan HTTPS. Admin ElProof wajib memakai verifikasi dua langkah dan setiap tindakan admin dicatat. Bila terjadi kegagalan pelindungan data yang berdampak pada Anda, ElProof memberi tahu Anda dan otoritas terkait sesuai ketentuan hukum.`,
    ],
  },
  {
    id: "hak-atas-data",
    title: "7. Hak Anda",
    body: [
      `Anda berhak meminta akses, salinan, perbaikan, dan penghapusan data pribadi Anda, menarik persetujuan, serta mengajukan keberatan atas pemrosesan tertentu. Owner dan Staff dapat menghubungi tim ElProof melalui halaman Kontak.`,
      `Client dan vendor mengajukan permintaan tersebut kepada WO yang mengelola datanya, karena WO adalah pengendali data tersebut. ElProof membantu WO memenuhi permintaan itu.`,
    ],
  },
  {
    id: "cookie",
    title: "8. Penyimpanan Sesi",
    body: [
      `ElProof hanya menyimpan data sesi yang diperlukan agar Anda tetap masuk. Tombol "Masuk dengan Google" dimuat dari layanan Google dan tunduk pada kebijakan privasi Google. ElProof tidak memakai cookie iklan atau pelacak pemasaran.`,
    ],
  },
  {
    id: "perubahan-kebijakan",
    title: "9. Perubahan Kebijakan",
    body: [
      `Kebijakan ini dapat diperbarui mengikuti perkembangan layanan. Perubahan material diberitahukan melalui WO Console sebelum berlaku.`,
    ],
  },
];

export default function PrivacyPage() {
  usePageMeta({ title: "Kebijakan privasi", description: "Cara ElProof mengumpulkan, memakai, dan melindungi data pribadi sesuai UU Pelindungan Data Pribadi." });
  return <LegalDocument icon={ShieldAlert} title="Kebijakan Privasi" updated={TERMS_UPDATED} sections={SECTIONS} />;
}
