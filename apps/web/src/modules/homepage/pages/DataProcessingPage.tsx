import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { COMPANY } from "@/modules/homepage/data/contact";
import { TERMS_UPDATED } from "@/modules/homepage/data/legal";
import { usePageMeta } from "@/shared/hooks/usePageMeta";
import { PUBLIC_PAGES } from "@/app/routes/public-pages";

// Words as in the app and in Syarat dan ketentuan §1 (QA FQ-09): workspace, Owner, klien, Portal Klien.
const SECTIONS: LegalSection[] = [
  {
    id: "para-pihak",
    title: "1. Para pihak dan peran",
    body: [
      `Perjanjian ini berlaku antara WO yang mendaftarkan workspace ("Pengendali") dan ${COMPANY.name} (${COMPANY.brand}) selaku penyelenggara ElProof ("Prosesor"), dan menjadi bagian dari Syarat dan ketentuan. Perjanjian ini mengatur pemrosesan data pribadi klien, keluarga klien, vendor, dan pihak lain yang dimasukkan WO ke ElProof, sesuai Undang-Undang Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi.`,
    ],
  },
  {
    id: "instruksi",
    title: "2. Pemrosesan berdasarkan instruksi",
    body: [
      `ElProof memproses data hanya untuk menyediakan layanan ElProof kepada WO dan sesuai instruksi WO melalui penggunaan fitur layanan. ElProof tidak memakai data tersebut untuk tujuan lain, termasuk pemasaran.`,
      `WO menjamin memiliki dasar pemrosesan yang sah untuk setiap data pribadi yang dimasukkan, termasuk memberi tahu klien bahwa datanya dikelola menggunakan ElProof.`,
    ],
  },
  {
    id: "kerahasiaan-keamanan",
    title: "3. Kerahasiaan dan keamanan",
    body: [
      `ElProof menjaga kerahasiaan data dan menerapkan langkah teknis dan organisasi yang wajar: pemisahan data per workspace, akses berdasarkan peran, enkripsi koneksi, verifikasi dua langkah untuk admin ElProof, dan pencatatan aktivitas admin. Personel ElProof hanya mengakses data workspace bila diperlukan untuk dukungan dan dengan izin Owner.`,
    ],
  },
  {
    id: "subprosesor",
    title: "4. Subprosesor",
    body: [
      `WO menyetujui ElProof menggunakan subprosesor untuk menjalankan layanan: penyedia server dan basis data, penyedia penyimpanan objek untuk dokumen dan foto, dan mitra pembayaran yang memproses pembayaran langganan. ElProof memastikan subprosesor terikat kewajiban pelindungan data yang setara dan memberi tahu WO sebelum menambah subprosesor baru.`,
    ],
  },
  {
    id: "bantuan",
    title: "5. Bantuan kepada pengendali",
    body: [
      `ElProof membantu WO memenuhi permintaan subjek data (akses, perbaikan, penghapusan) melalui fitur layanan atau atas permintaan WO.`,
      `Bila terjadi kegagalan pelindungan data yang menyangkut data WO, ElProof memberi tahu WO tanpa penundaan yang tidak wajar, paling lambat dalam 3 x 24 jam setelah mengetahuinya, beserta informasi yang diperlukan WO untuk memenuhi kewajibannya. Pemberitahuan dikirim ke Owner; WO dapat menghubungi kami di ${COMPANY.privacyEmail}.`,
    ],
  },
  {
    id: "akhir-layanan",
    title: "6. Pengembalian dan penghapusan data",
    body: [
      `Data tidak dihapus otomatis saat langganan berakhir. Atas permintaan WO, ElProof menyerahkan ekspor data dan/atau menghapus data workspace, kecuali yang wajib disimpan menurut hukum. Salinan cadangan terhapus mengikuti siklus pencadangan.`,
    ],
  },
  {
    id: "audit",
    title: "7. Informasi dan audit",
    body: [
      `ElProof menyediakan informasi yang wajar untuk menunjukkan kepatuhan terhadap perjanjian ini atas permintaan tertulis WO.`,
    ],
  },
];

export default function DataProcessingPage() {
  usePageMeta(PUBLIC_PAGES.dataProcessing);
  return (
    <LegalDocument
      title={PUBLIC_PAGES.dataProcessing.title}
      updated={TERMS_UPDATED}
      intro="Mengatur cara ElProof memproses data pribadi klien dan vendor yang Anda kelola di ElProof."
      sections={SECTIONS}
    />
  );
}
