import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { COMPANY, CONTACT } from "@/modules/homepage/data/contact";
import { TERMS_UPDATED } from "@/modules/homepage/data/legal";
import { usePageMeta } from "@/shared/hooks/usePageMeta";
import { PUBLIC_PAGES } from "@/app/routes/public-pages";

// Terms follow the app glossary (DESIGN.md §7, QA FQ-09): proyek, klien, jadwal/tahapan, progres, baca-saja, Portal Klien,
// anggota tim, workspace. Each is defined once in §1; the Owner keeps its name because it is the locked role in the app.
const SECTIONS: LegalSection[] = [
  {
    id: "definisi",
    title: "1. Definisi",
    body: [
      `"ElProof" adalah layanan perangkat lunak berbasis langganan yang membantu wedding organizer ("WO") mencatat persiapan setiap pernikahan — termasuk vendor, jadwal, pembayaran, dan kendala — serta menunjukkan sebagian informasi tersebut kepada klien. ElProof diselenggarakan oleh ${COMPANY.name} (${COMPANY.brand}), beralamat di ${CONTACT.address} ("kami").`,
      `"Workspace" adalah ruang kerja satu usaha WO di ElProof; data setiap workspace terpisah dari workspace lain. "Owner" adalah orang yang mendaftarkan workspace dan bertanggung jawab atasnya. "Anggota tim" adalah orang yang diundang Owner untuk bekerja di workspace, dengan peran dan akses yang diatur Owner.`,
      `"Proyek" adalah satu pernikahan yang dikelola WO di ElProof, beserta jadwal (daftar tahapan persiapan), vendor, pembayaran, kendala, dan dokumennya. "Klien" adalah pasangan atau keluarga yang pernikahannya dikelola WO. "Portal Klien" adalah tampilan khusus klien untuk memantau progres proyeknya sendiri.`,
    ],
  },
  {
    id: "ruang-lingkup",
    title: "2. Ruang lingkup layanan",
    body: [
      `ElProof menyediakan pencatatan proyek, pengelolaan vendor dan venue, pemantauan jadwal dan pembayaran, riwayat aktivitas, serta Portal Klien. Di Portal Klien, klien melihat progres proyeknya sendiri; selain itu klien hanya dapat mengirim bukti transfer dan menandai selesai tahapan yang ditugaskan WO kepadanya. Bukti transfer dari klien baru dihitung sebagai pembayaran setelah diverifikasi WO.`,
      `ElProof tidak menjadi pihak dalam perjanjian antara WO dan vendor atau kliennya. Seluruh kesepakatan bisnis, kontrak kerja, dan transaksi antara WO dengan pihak ketiga sepenuhnya menjadi tanggung jawab WO.`,
    ],
  },
  {
    id: "pendaftaran-akun",
    title: "3. Pendaftaran dan akun",
    body: [
      `Owner mendaftarkan workspace sendiri melalui halaman Daftar dengan akun Google. Workspace juga dapat dibuatkan oleh tim ElProof atas permintaan WO. Owner menjamin bahwa ia berwenang mewakili usaha WO yang didaftarkan dan bahwa data yang diberikan benar.`,
      `Owner dan anggota tim masuk menggunakan akun Google. Klien masuk menggunakan nama pengguna yang dibuatkan WO dan kata sandi yang dibuat klien sendiri melalui tautan undangan dari WO. Setiap pengguna wajib menjaga keamanan akses akunnya dan segera melapor bila ada dugaan akses tidak sah.`,
      `Setiap workspace hanya dapat mengakses datanya sendiri; data tidak dapat diakses dari workspace lain. Klien hanya dapat melihat data proyeknya sendiri.`,
    ],
  },
  {
    id: "masa-coba",
    title: "4. Masa coba gratis",
    body: [
      `Workspace baru mendapat masa coba gratis selama 14 hari tanpa kartu kredit dan tanpa tagihan otomatis. Selama masa coba, jumlah anggota tim, proyek aktif, dan akun klien dibatasi sesuai batas yang tampil di WO Console. Batas ini hilang setelah Owner membayar paket langganan.`,
      `Masa coba berlaku satu kali per akun Google dan per usaha. Kami dapat menolak atau menghentikan masa coba yang digunakan untuk penyalahgunaan, misalnya membuat workspace berulang untuk menghindari pembayaran.`,
    ],
  },
  {
    id: "langganan-pembayaran",
    title: "5. Langganan dan pembayaran",
    body: [
      `Akses penuh setelah masa coba diberikan per paket langganan dengan masa aktif tertentu. Harga yang berlaku adalah harga yang tampil saat tagihan dibuat. Pembayaran dilakukan oleh Owner melalui menu Langganan di WO Console.`,
      `Pembayaran diproses oleh mitra pembayaran pihak ketiga. ElProof tidak menyimpan data kartu atau rekening pembayaran WO.`,
      `Bila masa coba atau langganan berakhir tanpa pembayaran, akses dikurangi bertahap: 7 hari pertama akses tetap penuh dengan pengingat; 23 hari berikutnya baca-saja (data dapat dilihat dan diekspor, tetapi tidak diubah); setelah itu anggota tim hanya dapat membuka halaman Langganan untuk membayar. Portal Klien tetap dapat dibuka selama masa tersebut. Akses penuh kembali segera setelah pembayaran diterima.`,
    ],
  },
  {
    id: "data-kerahasiaan",
    title: "6. Data dan kerahasiaan",
    body: [
      `Data proyek, vendor, klien, dan dokumen yang diunggah WO adalah milik WO. Untuk data pribadi klien dan vendor, WO bertindak sebagai pengendali data dan kami sebagai prosesor data, sebagaimana diatur dalam Perjanjian pemrosesan data yang menjadi bagian dari ketentuan ini.`,
      `Kami tidak menghapus data workspace secara otomatis karena langganan berakhir. Owner dapat meminta ekspor atau penghapusan data melalui halaman Kontak.`,
    ],
  },
  {
    id: "penggunaan-wajar",
    title: "7. Penggunaan yang dilarang",
    body: [
      `WO dilarang menggunakan ElProof untuk menyimpan konten yang melanggar hukum, mengunggah data pribadi tanpa dasar yang sah, mencoba mengakses data workspace lain, atau mengganggu keamanan dan ketersediaan layanan.`,
      `Kami dapat membekukan workspace yang melanggar ketentuan ini atau digunakan untuk penipuan. Bila memungkinkan, kami memberi tahu Owner beserta alasannya.`,
    ],
  },
  {
    id: "batasan-tanggung-jawab",
    title: "8. Batasan tanggung jawab",
    body: [
      `ElProof disediakan sebagai alat bantu pencatatan dan transparansi, bukan sebagai penjamin terlaksananya perjanjian antara WO dengan vendor atau kliennya. Perselisihan yang timbul dari hubungan bisnis tersebut diselesaikan langsung oleh pihak-pihak terkait.`,
      `WO bertanggung jawab atas keakuratan data yang dimasukkan ke ElProof, termasuk status proyek, nilai kontrak, dan informasi vendor.`,
    ],
  },
  {
    id: "perubahan-ketentuan",
    title: "9. Perubahan ketentuan",
    body: [
      `Ketentuan ini dapat diperbarui mengikuti perkembangan layanan. Perubahan material diberitahukan kepada Owner melalui WO Console sebelum berlaku. Versi yang disetujui setiap Owner saat mendaftar tercatat beserta waktunya.`,
    ],
  },
];

export default function TermsPage() {
  usePageMeta(PUBLIC_PAGES.terms);
  return <LegalDocument title={PUBLIC_PAGES.terms.title} updated={TERMS_UPDATED} sections={SECTIONS} />;
}
