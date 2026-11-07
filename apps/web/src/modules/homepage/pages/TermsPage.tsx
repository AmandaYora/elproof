import { Scale } from "lucide-react";
import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { TERMS_UPDATED } from "@/modules/homepage/data/legal";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const SECTIONS: LegalSection[] = [
  {
    id: "definisi",
    title: "1. Definisi",
    body: [
      `"ElProof" adalah platform perangkat lunak berbasis langganan (Software as a Service) yang membantu wedding organizer ("WO") mendokumentasikan persiapan project pernikahan — termasuk vendor, timeline, pembayaran, dan kendala — serta membagikan sebagian informasi tersebut kepada pasangan client terkait.`,
      `"Tenant" atau "Workspace" adalah satu akun bisnis WO di ElProof. "Owner" adalah orang yang mendaftarkan Workspace dan bertanggung jawab atasnya. "Staff" adalah pengguna yang bekerja di bawah satu Tenant. "Client" adalah pasangan atau keluarga yang project pernikahannya dikelola Tenant, dengan akses baca terbatas melalui Client Portal.`,
    ],
  },
  {
    id: "ruang-lingkup",
    title: "2. Ruang Lingkup Layanan",
    body: [
      `ElProof menyediakan pencatatan project, manajemen vendor dan venue, pelacakan timeline dan pembayaran, log aktivitas, serta portal bagi Client untuk memantau progress project mereka sendiri secara read-only.`,
      `ElProof tidak menjadi pihak dalam perjanjian antara Tenant dan vendor atau Client-nya. Seluruh kesepakatan bisnis, kontrak kerja, dan transaksi antara Tenant dengan pihak ketiga sepenuhnya menjadi tanggung jawab Tenant.`,
    ],
  },
  {
    id: "pendaftaran-akun",
    title: "3. Pendaftaran dan Akun",
    body: [
      `Owner mendaftarkan Workspace sendiri melalui halaman Daftar dengan akun Google. Workspace juga dapat dibuatkan oleh tim ElProof atas permintaan WO. Owner menjamin bahwa ia berwenang mewakili usaha WO yang didaftarkan dan bahwa data yang diberikan benar.`,
      `Owner dan Staff masuk menggunakan akun Google. Client masuk menggunakan nama pengguna dan kata sandi yang diberikan Tenant. Setiap pengguna wajib menjaga keamanan akses akunnya dan segera melapor bila ada dugaan akses tidak sah.`,
      `Setiap Tenant hanya dapat mengakses data miliknya sendiri. Data antar-Tenant terpisah dan tidak dapat diakses lintas akun bisnis. Client hanya dapat melihat data project-nya sendiri.`,
    ],
  },
  {
    id: "masa-coba",
    title: "4. Masa Coba Gratis",
    body: [
      `Workspace baru mendapat masa coba gratis selama 14 hari tanpa kartu kredit dan tanpa tagihan otomatis. Selama masa coba, penggunaan dibatasi jumlah staf, project aktif, dan akun client sesuai batas yang tampil di WO Console. Batas ini hilang setelah Tenant membayar paket langganan.`,
      `Masa coba berlaku satu kali per akun Google dan per usaha. ElProof dapat menolak atau menghentikan masa coba yang digunakan untuk penyalahgunaan, misalnya membuat Workspace berulang untuk menghindari pembayaran.`,
    ],
  },
  {
    id: "langganan-pembayaran",
    title: "5. Langganan dan Pembayaran",
    body: [
      `Akses penuh setelah masa coba diberikan per paket langganan dengan masa aktif tertentu. Harga yang berlaku adalah harga yang tampil saat tagihan dibuat. Pembayaran dilakukan oleh Owner melalui menu Langganan di WO Console.`,
      `Pembayaran diproses oleh mitra payment gateway pihak ketiga. ElProof tidak menyimpan data kartu atau rekening pembayaran Tenant.`,
      `Bila masa coba atau langganan berakhir tanpa pembayaran, akses dikurangi bertahap: 7 hari pertama akses tetap penuh dengan pengingat; 23 hari berikutnya hanya baca dan ekspor data; setelah itu Staff hanya dapat membuka halaman Langganan untuk membayar. Client Portal tetap dapat dibaca selama masa tersebut. Akses penuh kembali segera setelah pembayaran diterima.`,
    ],
  },
  {
    id: "data-kerahasiaan",
    title: "6. Data dan Kerahasiaan",
    body: [
      `Data project, vendor, client, dan dokumen yang diunggah Tenant adalah milik Tenant. Untuk data pribadi Client dan vendor, Tenant bertindak sebagai pengendali data dan ElProof sebagai prosesor data, sebagaimana diatur dalam Perjanjian Pemrosesan Data yang menjadi bagian dari ketentuan ini.`,
      `ElProof tidak menghapus data Tenant secara otomatis karena langganan berakhir. Tenant dapat meminta ekspor atau penghapusan data melalui halaman Kontak.`,
    ],
  },
  {
    id: "penggunaan-wajar",
    title: "7. Penggunaan yang Dilarang",
    body: [
      `Tenant dilarang menggunakan ElProof untuk menyimpan konten yang melanggar hukum, mengunggah data pribadi tanpa dasar yang sah, mencoba mengakses data Tenant lain, atau mengganggu keamanan dan ketersediaan layanan.`,
      `ElProof dapat membekukan Workspace yang melanggar ketentuan ini atau digunakan untuk penipuan. Bila memungkinkan, ElProof memberi tahu Owner beserta alasannya.`,
    ],
  },
  {
    id: "batasan-tanggung-jawab",
    title: "8. Batasan Tanggung Jawab",
    body: [
      `ElProof disediakan sebagai alat bantu dokumentasi dan transparansi, bukan sebagai penjamin terlaksananya perjanjian antara Tenant dengan vendor atau Client-nya. Perselisihan yang timbul dari hubungan bisnis tersebut diselesaikan langsung oleh pihak-pihak terkait.`,
      `Tenant bertanggung jawab atas keakuratan data yang dimasukkan ke ElProof, termasuk status project, nilai kontrak, dan informasi vendor.`,
    ],
  },
  {
    id: "perubahan-ketentuan",
    title: "9. Perubahan Ketentuan",
    body: [
      `Ketentuan ini dapat diperbarui mengikuti perkembangan layanan. Perubahan material diberitahukan kepada Owner melalui WO Console sebelum berlaku. Versi yang disetujui setiap Owner saat mendaftar tercatat beserta waktunya.`,
    ],
  },
];

export default function TermsPage() {
  usePageMeta({ title: "Syarat dan ketentuan", description: "Syarat dan ketentuan penggunaan layanan ElProof untuk wedding organizer dan klien mereka." });
  return <LegalDocument icon={Scale} title="Syarat & Ketentuan" updated={TERMS_UPDATED} sections={SECTIONS} />;
}
