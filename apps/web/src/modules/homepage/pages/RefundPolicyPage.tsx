import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { REFUND_UPDATED } from "@/modules/homepage/data/legal";
import { usePageMeta } from "@/shared/hooks/usePageMeta";
import { PUBLIC_PAGES } from "@/app/routes/public-pages";

// Words as in the app and in Syarat dan ketentuan §1 (QA FQ-09): WO, workspace, Owner, anggota tim, proyek.
const SECTIONS: LegalSection[] = [
  {
    id: "ketentuan-umum",
    title: "1. Ketentuan umum",
    body: [
      `ElProof adalah layanan perangkat lunak berbasis langganan, bukan barang fisik. Kebijakan refund (pengembalian dana) ini mengatur kondisi di mana WO berhak mengajukan pengembalian dana atas pembayaran langganan yang telah diproses melalui mitra pembayaran pihak ketiga.`,
    ],
  },
  {
    id: "kondisi-berhak",
    title: "2. Kondisi yang berhak direfund",
    body: [
      `Kesalahan penagihan sistem — misalnya WO tertagih dua kali (duplikat) untuk paket langganan yang sama pada periode yang sama.`,
      `Layanan tidak dapat diakses sama sekali akibat gangguan di pihak ElProof selama lebih dari 3 (tiga) hari kerja berturut-turut sejak pembayaran diterima, dan belum sempat digunakan sama sekali oleh WO.`,
      `Pembayaran berhasil diproses namun langganan workspace tidak diaktifkan dalam waktu wajar akibat kesalahan di pihak ElProof.`,
    ],
  },
  {
    id: "kondisi-tidak-berhak",
    title: "3. Kondisi yang tidak dapat direfund",
    body: [
      `Langganan yang sudah digunakan sebagian atau seluruhnya untuk mengelola proyek, sejak Owner atau anggota tim pertama kali masuk ke workspace tersebut.`,
      `Berubah pikiran setelah pembayaran berhasil, tanpa adanya kegagalan layanan di pihak ElProof.`,
      `Akses yang dibatasi sementara karena keterlambatan perpanjangan langganan — ini bukan pemutusan layanan, dan akses penuh kembali kapan saja setelah perpanjangan dibayar.`,
      `Kesalahan input data, kesalahan pemilihan paket, atau kendala penggunaan yang berasal dari pihak WO sendiri.`,
    ],
  },
  {
    id: "cara-mengajukan",
    title: "4. Cara mengajukan refund",
    body: [
      `Ajukan permintaan refund melalui salah satu kanal pada halaman Kontak (telepon, WhatsApp, atau email), selambat-lambatnya 7 (tujuh) hari kalender sejak tanggal pembayaran, dengan menyertakan bukti pembayaran dan alasan pengajuan.`,
      `Tim kami akan memverifikasi kelayakan pengajuan berdasarkan kondisi pada Bagian 2 dan 3 di atas, dan memberikan keputusan selambat-lambatnya 5 (lima) hari kerja setelah pengajuan diterima lengkap.`,
    ],
  },
  {
    id: "proses-waktu",
    title: "5. Proses dan waktu pengembalian dana",
    body: [
      `Refund yang disetujui diproses melalui mitra pembayaran yang sama dengan metode pembayaran awal, dan diteruskan ke rekening atau instrumen pembayaran WO selambat-lambatnya 14 (empat belas) hari kerja sejak persetujuan — mengikuti waktu proses mitra pembayaran terkait.`,
      `ElProof tidak memungut biaya tambahan atas proses refund yang disetujui; biaya administrasi yang mungkin dikenakan oleh mitra pembayaran di luar kendali ElProof.`,
    ],
  },
  {
    id: "perubahan-kebijakan",
    title: "6. Perubahan kebijakan",
    body: [
      `Kebijakan ini dapat diperbarui sewaktu-waktu mengikuti perkembangan layanan. Perubahan berlaku sejak dipublikasikan pada halaman ini dan tidak berlaku surut terhadap pengajuan refund yang sudah diputuskan sebelumnya.`,
    ],
  },
];

export default function RefundPolicyPage() {
  usePageMeta(PUBLIC_PAGES.refund);
  return (
    <LegalDocument
      title={PUBLIC_PAGES.refund.title}
      updated={REFUND_UPDATED}
      sections={SECTIONS}
      closing={{
        title: "Ingin mengajukan pengembalian dana?",
        body: (
          <>
            Pastikan Anda memenuhi syarat-syarat di atas, lalu hubungi kami melalui{" "}
            <Link to={ROUTE_PATHS.homepageContact} className="font-semibold text-foreground underline underline-offset-4 hover:text-primary-hover">
              halaman Kontak
            </Link>{" "}
            dan sertakan bukti pembayaran Anda.
          </>
        ),
      }}
    />
  );
}
