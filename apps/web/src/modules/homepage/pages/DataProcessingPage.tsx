import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { TERMS_UPDATED } from "@/modules/homepage/data/legal";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const SECTIONS: LegalSection[] = [
  {
    id: "para-pihak",
    title: "1. Para Pihak dan Peran",
    body: [
      `Perjanjian ini berlaku antara WO yang mendaftarkan Workspace ("Pengendali") dan ElProof ("Prosesor"), dan menjadi bagian dari Syarat & Ketentuan. Perjanjian ini mengatur pemrosesan data pribadi Client, keluarga Client, vendor, dan pihak lain yang dimasukkan WO ke ElProof, sesuai Undang-Undang Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi.`,
    ],
  },
  {
    id: "instruksi",
    title: "2. Pemrosesan Berdasarkan Instruksi",
    body: [
      `ElProof memproses data hanya untuk menyediakan layanan ElProof kepada WO dan sesuai instruksi WO melalui penggunaan fitur layanan. ElProof tidak memakai data tersebut untuk tujuan lain, termasuk pemasaran.`,
      `WO menjamin memiliki dasar pemrosesan yang sah untuk setiap data pribadi yang dimasukkan, termasuk memberi tahu Client bahwa datanya dikelola menggunakan ElProof.`,
    ],
  },
  {
    id: "kerahasiaan-keamanan",
    title: "3. Kerahasiaan dan Keamanan",
    body: [
      `ElProof menjaga kerahasiaan data dan menerapkan langkah teknis dan organisasi yang wajar: pemisahan data per Tenant, akses berdasarkan peran, enkripsi koneksi, verifikasi dua langkah untuk admin ElProof, dan pencatatan aktivitas admin. Personel ElProof hanya mengakses data Tenant bila diperlukan untuk dukungan dan dengan izin Owner.`,
    ],
  },
  {
    id: "subprosesor",
    title: "4. Subprosesor",
    body: [
      `WO menyetujui ElProof menggunakan subprosesor untuk menjalankan layanan: penyedia server dan basis data, penyedia penyimpanan objek untuk dokumen dan foto, dan mitra payment gateway untuk pembayaran langganan. ElProof memastikan subprosesor terikat kewajiban pelindungan data yang setara dan memberi tahu WO sebelum menambah subprosesor baru.`,
    ],
  },
  {
    id: "bantuan",
    title: "5. Bantuan kepada Pengendali",
    body: [
      `ElProof membantu WO memenuhi permintaan subjek data (akses, perbaikan, penghapusan) melalui fitur layanan atau atas permintaan WO.`,
      `Bila terjadi kegagalan pelindungan data yang menyangkut data WO, ElProof memberi tahu WO tanpa penundaan yang tidak wajar, paling lambat dalam 3 x 24 jam setelah mengetahuinya, beserta informasi yang diperlukan WO untuk memenuhi kewajibannya.`,
    ],
  },
  {
    id: "akhir-layanan",
    title: "6. Pengembalian dan Penghapusan Data",
    body: [
      `Data tidak dihapus otomatis saat langganan berakhir. Atas permintaan WO, ElProof menyerahkan ekspor data dan/atau menghapus data Workspace, kecuali yang wajib disimpan menurut hukum. Salinan cadangan terhapus mengikuti siklus pencadangan.`,
    ],
  },
  {
    id: "audit",
    title: "7. Informasi dan Audit",
    body: [
      `ElProof menyediakan informasi yang wajar untuk menunjukkan kepatuhan terhadap perjanjian ini atas permintaan tertulis WO.`,
    ],
  },
];

export default function DataProcessingPage() {
  usePageMeta({ title: "Perjanjian pemrosesan data", description: "Perjanjian pemrosesan data antara wedding organizer sebagai pengendali data dan ElProof sebagai prosesor." });
  return (
    <LegalDocument
      title="Perjanjian Pemrosesan Data"
      updated={TERMS_UPDATED}
      intro="Mengatur cara ElProof memproses data pribadi client dan vendor yang Anda kelola di ElProof."
      sections={SECTIONS}
    />
  );
}
