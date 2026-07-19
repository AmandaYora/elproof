import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Receipt } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Reveal } from "@/modules/homepage/components/Reveal";
import { cn } from "@/shared/lib/cn";

const SECTIONS = [
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
  const [activeId, setActiveId] = useState("");

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
          }
        });
      },
      { rootMargin: "-20% 0px -80% 0px" }
    );

    SECTIONS.forEach((s) => {
      const el = document.getElementById(s.id);
      if (el) observer.observe(el);
    });

    return () => observer.disconnect();
  }, []);

  return (
    <div className="bg-background min-h-screen">
      {/* HEADER */}
      <section className="relative overflow-hidden border-b border-border bg-white px-6 py-20 md:py-24">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1px,transparent_1px),linear-gradient(to_bottom,#f1f5f9_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] opacity-60"></div>
        <div className="mx-auto max-w-4xl text-center relative z-10">
          <Reveal>
            <div className="inline-flex items-center gap-2 rounded-full bg-blue-50 px-3.5 py-1.5 text-[13px] font-bold uppercase tracking-widest text-blue-600">
              <Receipt className="h-4 w-4" /> Dokumen Layanan
            </div>
            <h1 className="mt-6 font-display text-[36px] font-semibold tracking-tight text-navy-950 md:text-[48px]">
              Kebijakan Refund
            </h1>
            <p className="mt-4 text-[15px] font-medium text-text-secondary">
              Terakhir diperbarui: 18 Juli 2026
            </p>
          </Reveal>
        </div>
      </section>

      {/* CONTENT */}
      <div className="mx-auto max-w-6xl px-6 py-16 md:py-24">
        <div className="flex flex-col md:flex-row gap-12 lg:gap-20">
          
          {/* SIDEBAR NAVIGATION */}
          <div className="hidden md:block w-64 shrink-0">
            <div className="sticky top-32 flex flex-col gap-2">
              <h3 className="text-[12px] font-bold uppercase tracking-wider text-text-secondary mb-4">Daftar Isi</h3>
              {SECTIONS.map((section) => (
                <a
                  key={section.id}
                  href={`#${section.id}`}
                  className={cn(
                    "text-[14px] font-medium transition-all duration-200 py-1.5 border-l-2 pl-4",
                    activeId === section.id
                      ? "border-navy-900 text-navy-950 bg-navy-50/50 rounded-r-md"
                      : "border-border text-text-secondary hover:border-navy-400 hover:text-navy-900"
                  )}
                >
                  {section.title}
                </a>
              ))}
            </div>
          </div>

          {/* DOCUMENT BODY */}
          <div className="flex-1 max-w-3xl rounded-3xl bg-white p-8 md:p-12 shadow-sm border border-border">
            <Reveal className="flex flex-col gap-12">
              {SECTIONS.map((section) => (
                <div key={section.id} id={section.id} className="scroll-mt-32">
                  <h2 className="text-[20px] font-bold text-navy-950 tracking-tight">{section.title}</h2>
                  <div className="mt-4 flex flex-col gap-4">
                    {section.body.map((paragraph, i) => (
                      <p key={i} className="text-[15.5px] leading-relaxed text-text-secondary">
                        {paragraph}
                      </p>
                    ))}
                  </div>
                </div>
              ))}

              <div className="mt-8 rounded-2xl border border-border bg-surface-muted/60 p-8">
                <h3 className="text-[16px] font-bold text-navy-950">Ingin mengajukan pengembalian dana?</h3>
                <p className="mt-3 text-[14.5px] leading-relaxed text-text-secondary">
                  Pastikan Anda memenuhi syarat-syarat di atas, lalu hubungi kami melalui{" "}
                  <Link to={ROUTE_PATHS.homepageContact} className="font-semibold text-navy-900 hover:underline">
                    halaman Kontak
                  </Link>{" "}
                  dan sertakan bukti pembayaran Anda.
                </p>
              </div>
            </Reveal>
          </div>
        </div>
      </div>
    </div>
  );
}
