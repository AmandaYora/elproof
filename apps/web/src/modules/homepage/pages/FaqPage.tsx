import { useState } from "react";
import { Link } from "react-router-dom";
import { ChevronDown, HelpCircle, ArrowRight } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Reveal } from "@/modules/homepage/components/Reveal";
import { cn } from "@/shared/lib/cn";

const FAQS = [
  {
    question: "Apa itu ElProof?",
    answer: `ElProof adalah platform berbasis langganan (Software as a Service) yang membantu wedding organizer (WO) mendokumentasikan persiapan project pernikahan — vendor, milestone, pembayaran, dan kendala — serta membagikan sebagian informasi tersebut kepada pasangan client terkait lewat Client Portal.`,
  },
  {
    question: "Siapa yang bisa menggunakan ElProof?",
    answer: `ElProof ditujukan untuk bisnis wedding organizer. Setiap bisnis WO mendaftar sebagai satu Tenant, dengan akun Owner yang dapat menambahkan Staff lain sesuai kebutuhan. Pasangan client mendapat akses baca terbatas melalui Client Portal, bukan akun berlangganan tersendiri.`,
  },
  {
    question: "Bagaimana cara mulai berlangganan?",
    answer: `Aktivasi akun Tenant dibantu langsung oleh tim kami. Hubungi kami melalui halaman Kontak untuk menceritakan kebutuhan bisnis Anda, dan tim kami akan membuatkan akun Owner setelah proses aktivasi selesai.`,
  },
  {
    question: "Metode pembayaran apa saja yang didukung?",
    answer: `Pembayaran langganan diproses melalui mitra payment gateway pihak ketiga. ElProof tidak menyimpan data kartu atau rekening pembayaran Anda — seluruh proses otorisasi pembayaran ditangani langsung oleh mitra tersebut.`,
  },
  {
    question: "Apa yang terjadi jika langganan saya berakhir?",
    answer: `Apabila langganan berakhir tanpa perpanjangan, akses Tenant terhadap ElProof dapat dinonaktifkan sementara hingga langganan diperpanjang kembali. Data yang sudah tercatat tidak dihapus akibat keterlambatan perpanjangan.`,
  },
  {
    question: "Apakah saya bisa mengajukan refund?",
    answer: `Ya, dalam kondisi tertentu — misalnya kesalahan penagihan sistem. Lihat halaman Kebijakan Refund untuk syarat, kondisi yang tidak dapat direfund, dan proses pengajuannya.`,
  },
  {
    question: "Apakah data project saya aman?",
    answer: `Data antar-Tenant terpisah secara penuh dan tidak dapat diakses lintas akun bisnis. ElProof menerapkan pemisahan akses berbasis peran (Staff, Client, admin platform) sehingga setiap pihak hanya melihat data yang menjadi haknya. Lihat Kebijakan Privasi untuk detail lengkap.`,
  },
  {
    question: "Bagaimana cara menghubungi dukungan pelanggan?",
    answer: `Tim kami dapat dihubungi melalui telepon, WhatsApp, atau email pada jam layanan yang tertera di halaman Kontak. Di luar jam tersebut, pesan Anda tetap akan kami balas sesegera mungkin.`,
  },
];

export default function FaqPage() {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  const toggleOpen = (index: number) => {
    setOpenIndex(openIndex === index ? null : index);
  };

  return (
    <div className="bg-background min-h-screen">
      {/* HEADER PAGE */}
      <section className="relative overflow-hidden border-b border-border bg-white px-6 py-20 md:py-28">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1px,transparent_1px),linear-gradient(to_bottom,#f1f5f9_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] opacity-60"></div>
        <div className="mx-auto max-w-3xl text-center relative z-10">
          <Reveal>
            <div className="inline-flex items-center gap-2 rounded-full bg-blue-50 px-3.5 py-1.5 text-[13px] font-bold uppercase tracking-widest text-blue-600">
              <HelpCircle className="h-4 w-4" /> Bantuan
            </div>
            <h1 className="mt-6 font-display text-[36px] font-semibold tracking-tight text-navy-950 md:text-[48px]">
              Pertanyaan yang Sering Diajukan
            </h1>
            <p className="mx-auto mt-6 max-w-xl text-[16px] leading-relaxed text-text-secondary">
              Belum menemukan jawaban yang Anda cari? Kami siap membantu menjawab keraguan Anda.
            </p>
          </Reveal>
        </div>
      </section>

      {/* FAQ SECTION */}
      <section className="px-6 py-20 bg-white">
        <div className="mx-auto max-w-3xl">
          <Reveal className="flex flex-col gap-3">
            {FAQS.map((item, index) => {
              const isOpen = openIndex === index;
              return (
                <div
                  key={index}
                  className={cn(
                    "overflow-hidden rounded-2xl border transition-all duration-200",
                    isOpen 
                      ? "border-navy-900/10 bg-white shadow-lg shadow-navy-900/5" 
                      : "border-border bg-surface hover:border-navy-900/20"
                  )}
                >
                  <button
                    type="button"
                    onClick={() => toggleOpen(index)}
                    className="flex w-full items-center justify-between gap-4 px-6 py-5 text-left outline-none focus-visible:bg-surface-muted"
                  >
                    <span className={cn(
                      "text-[16px] font-semibold transition-colors",
                      isOpen ? "text-navy-950" : "text-text-primary"
                    )}>
                      {item.question}
                    </span>
                    <span className={cn(
                      "flex h-8 w-8 shrink-0 items-center justify-center rounded-full transition-all duration-300",
                      isOpen ? "bg-navy-50 text-navy-900" : "bg-surface-muted text-text-secondary"
                    )}>
                      <ChevronDown className={cn("h-4 w-4 transition-transform duration-300", isOpen && "rotate-180")} />
                    </span>
                  </button>
                  
                  <div
                    className={cn(
                      "grid transition-all duration-300 ease-in-out",
                      isOpen ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"
                    )}
                  >
                    <div className="overflow-hidden">
                      <p className="px-6 pb-6 text-[15px] leading-relaxed text-text-secondary">
                        {item.answer}
                      </p>
                    </div>
                  </div>
                </div>
              );
            })}
          </Reveal>

          <Reveal delay={200} className="mt-16 rounded-3xl bg-surface-muted/50 p-8 text-center border border-border">
            <h3 className="text-[18px] font-bold text-navy-950">Masih punya pertanyaan lain?</h3>
            <p className="mt-2 text-[15px] text-text-secondary">Tim kami selalu siap membantu Anda kapan saja.</p>
            <Link
              to={ROUTE_PATHS.homepageContact}
              className="mt-6 inline-flex h-11 items-center justify-center gap-2 rounded-full bg-white px-6 text-[14px] font-semibold text-navy-950 shadow-sm border border-border transition-all hover:-translate-y-0.5 hover:shadow-md"
            >
              Hubungi Kami <ArrowRight className="h-4 w-4" />
            </Link>
          </Reveal>
        </div>
      </section>
    </div>
  );
}
