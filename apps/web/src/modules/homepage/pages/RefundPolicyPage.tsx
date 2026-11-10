import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { LegalDocument, type LegalSection } from "@/modules/homepage/components/LegalDocument";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

// Dated separately from TERMS_UPDATED (data/legal.ts): this policy has its own revision.
const UPDATED = "18 Juli 2026";

const SECTIONS: LegalSection[] = [
  {
    id: "ketentuan-umum",
    title: "1. Ketentuan Umum",
    body: [
      `ElProof adalah layanan berbasis langganan (Software as a Service), bukan barang fisik. Kebijakan refund ini mengatur kondisi di mana Tenant berhak mengajukan pengembalian dana atas pembayaran langganan yang telah diproses melalui mitra payment gateway.`,
    ],
  },
  {
    id: "kondisi-berhak",
    title: "2. Kondisi yang Berhak Direfund",
    body: [
      `Kesalahan penagihan sistem — misalnya Tenant tertagih dua kali (duplikat) untuk paket langganan yang sama pada periode yang sama.`,
      `Layanan tidak dapat diakses sama sekali akibat gangguan di pihak ElProof selama lebih dari 3 (tiga) hari kerja berturut-turut sejak pembayaran diterima, dan belum sempat digunakan sama sekali oleh Tenant.`,
      `Pembayaran berhasil diproses namun aktivasi akun Tenant tidak dilakukan dalam waktu wajar akibat kesalahan di pihak ElProof.`,
    ],
  },
  {
    id: "kondisi-tidak-berhak",
    title: "3. Kondisi yang Tidak Dapat Direfund",
    body: [
      `Langganan yang sudah digunakan sebagian atau seluruhnya untuk mengelola project, sejak Owner atau Staff pertama kali login ke akun tersebut.`,
      `Perubahan pikiran (change of mind) setelah pembayaran berhasil, tanpa adanya kegagalan layanan di pihak ElProof.`,
      `Akses yang dinonaktifkan sementara karena keterlambatan perpanjangan langganan — ini bukan pemutusan layanan, dan dapat diaktifkan kembali kapan saja setelah perpanjangan dilakukan.`,
      `Kesalahan input data, kesalahan pemilihan paket, atau kendala penggunaan yang berasal dari pihak Tenant sendiri.`,
    ],
  },
  {
    id: "cara-mengajukan",
    title: "4. Cara Mengajukan Refund",
    body: [
      `Ajukan permintaan refund melalui salah satu kanal pada halaman Kontak (telepon, WhatsApp, atau email), selambat-lambatnya 7 (tujuh) hari kalender sejak tanggal pembayaran, dengan menyertakan bukti pembayaran dan alasan pengajuan.`,
      `Tim kami akan memverifikasi kelayakan pengajuan berdasarkan kondisi pada Bagian 2 dan 3 di atas, dan memberikan keputusan selambat-lambatnya 5 (lima) hari kerja setelah pengajuan diterima lengkap.`,
    ],
  },
  {
    id: "proses-waktu",
    title: "5. Proses dan Waktu Pengembalian Dana",
    body: [
      `Refund yang disetujui diproses melalui mitra payment gateway yang sama dengan metode pembayaran awal, dan diteruskan ke rekening atau instrumen pembayaran Tenant selambat-lambatnya 14 (empat belas) hari kerja sejak persetujuan — mengikuti waktu proses mitra pembayaran terkait.`,
      `ElProof tidak memungut biaya tambahan atas proses refund yang disetujui; biaya administrasi yang mungkin dikenakan oleh mitra payment gateway di luar kendali ElProof.`,
    ],
  },
  {
    id: "perubahan-kebijakan",
    title: "6. Perubahan Kebijakan",
    body: [
      `Kebijakan ini dapat diperbarui sewaktu-waktu mengikuti perkembangan layanan. Perubahan berlaku sejak dipublikasikan pada halaman ini dan tidak berlaku surut terhadap pengajuan refund yang sudah diputuskan sebelumnya.`,
    ],
  },
];

export default function RefundPolicyPage() {
  usePageMeta({ title: "Kebijakan refund", description: "Ketentuan pengembalian dana untuk langganan ElProof." });
  return (
    <LegalDocument
      title="Kebijakan Refund"
      updated={UPDATED}
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

