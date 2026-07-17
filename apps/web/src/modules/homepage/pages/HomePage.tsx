import { Link } from "react-router-dom";
import { ClipboardList, Handshake, Eye, CalendarCheck, Wallet, ShieldCheck, Receipt, ArrowRight } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { Reveal } from "@/modules/homepage/components/Reveal";

const PARTIES = [
  {
    icon: ClipboardList,
    title: "Tim WO Console",
    body: "Catat setiap project, vendor, dan pembayaran secara real-time — tidak ada lagi laporan manual yang tercecer di grup chat.",
  },
  {
    icon: Handshake,
    title: "Vendor",
    body: "Setiap vendor punya rekam jejak keterlibatan yang jelas: status booking, tenggat kerja, dan riwayat pembayaran per project.",
  },
  {
    icon: Eye,
    title: "Pasangan Client",
    body: "Portal khusus, read-only: pasangan memantau progress dan kendala persiapan tanpa perlu bertanya berulang kali ke tim WO.",
  },
];

const FEATURES = [
  {
    icon: CalendarCheck,
    title: "Timeline & Milestone Project",
    body: "Dari tanda tangan kontrak hingga hari-H, setiap tahap persiapan punya milestone dan tenggat yang bisa dipantau bersama tim dan client.",
  },
  {
    icon: Wallet,
    title: "Manajemen Vendor & Pembayaran",
    body: "Kelola kontrak, DP, pelunasan, dan status keterlibatan setiap vendor — katering, dekorasi, dokumentasi — dalam satu dashboard.",
  },
  {
    icon: ShieldCheck,
    title: "Portal Transparansi Client",
    body: "Pasangan client melihat progress, bukti kerja, dan kendala project secara langsung, tanpa akses ke data project WO lainnya.",
  },
  {
    icon: Receipt,
    title: "Langganan & Tagihan",
    body: "Satu paket langganan tahunan untuk seluruh tim WO Anda, dengan pembayaran QRIS/VA otomatis lewat gateway pembayaran terpercaya.",
  },
];

export default function HomePage() {
  return (
    <div>
      {/* HERO */}
      <section className="relative overflow-hidden bg-gradient-to-br from-navy-950 to-navy-900 px-6 pb-20 pt-16 md:pb-28 md:pt-20">
        <div className="pointer-events-none absolute -right-32 -top-32 h-96 w-96 rounded-full bg-navy-800/40 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-40 -left-20 h-80 w-80 rounded-full bg-navy-800/30 blur-3xl" />

        <div className="relative mx-auto flex max-w-6xl flex-col items-center gap-10 md:flex-row md:items-center md:justify-between">
          <div className="max-w-xl text-center md:text-left">
            <p className="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/5 px-3.5 py-1.5 text-[12.5px] font-medium text-white/70">
              Portal kerja &amp; transparansi wedding organizer
            </p>
            <h1 className="mt-6 font-display text-[36px] font-semibold leading-[1.15] text-white md:text-[46px]">
              Setiap detail persiapan pernikahan, tercatat dan terbukti.
            </h1>
            <p className="mt-5 text-[15.5px] leading-relaxed text-white/70">
              ElProof mendokumentasikan progress, vendor, dan pembayaran dalam satu tempat untuk tim WO —
              sementara pasangan client memantau langsung persiapan hari bahagia mereka, tanpa perlu bertanya
              berulang kali.
            </p>
            <div className="mt-8 flex flex-col items-center gap-3 sm:flex-row md:justify-start">
              <Link
                to={ROUTE_PATHS.homepageContact}
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-md bg-warning px-6 text-[14.5px] font-semibold text-navy-950 transition-colors hover:bg-warning/90 sm:w-auto"
              >
                Hubungi Kami <ArrowRight className="h-4 w-4" />
              </Link>
              <a
                href="#fitur"
                className="inline-flex h-11 w-full items-center justify-center rounded-md border border-white/20 px-6 text-[14.5px] font-semibold text-white transition-colors hover:bg-white/10 sm:w-auto"
              >
                Lihat Fitur
              </a>
            </div>
          </div>

          <div className="flex shrink-0 items-center justify-center">
            <ProofSeal size={200} animate className="text-warning" />
          </div>
        </div>
      </section>

      {/* TIGA PIHAK */}
      <section className="px-6 py-20 md:py-24">
        <div className="mx-auto max-w-6xl">
          <Reveal className="max-w-2xl">
            <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Satu platform, tiga pihak</p>
            <h2 className="mt-3 font-display text-[28px] font-semibold text-text-primary md:text-[32px]">
              Dibangun untuk menjembatani semua yang terlibat dalam satu pernikahan.
            </h2>
          </Reveal>

          <div className="mt-12 grid gap-6 md:grid-cols-3">
            {PARTIES.map((party, i) => (
              <Reveal key={party.title} delay={i * 90}>
                <div className="h-full rounded-xl border border-border bg-surface p-7">
                  <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-navy-900/[0.07] text-navy-900">
                    <party.icon className="h-5 w-5" />
                  </div>
                  <h3 className="mt-5 text-[16px] font-semibold text-text-primary">{party.title}</h3>
                  <p className="mt-2 text-[13.5px] leading-relaxed text-text-secondary">{party.body}</p>
                </div>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* FITUR */}
      <section id="fitur" className="bg-surface-muted/60 px-6 py-20 md:py-24">
        <div className="mx-auto max-w-6xl">
          <Reveal className="max-w-2xl">
            <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Yang bisa dilakukan ElProof</p>
            <h2 className="mt-3 font-display text-[28px] font-semibold text-text-primary md:text-[32px]">
              Semua yang dibutuhkan tim WO untuk bekerja rapi — dan client untuk tenang.
            </h2>
          </Reveal>

          <div className="mt-12 grid gap-5 md:grid-cols-2">
            {FEATURES.map((feature, i) => (
              <Reveal key={feature.title} delay={i * 80}>
                <div className="flex h-full gap-4 rounded-xl border border-border bg-surface p-6">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-navy-900 text-white">
                    <feature.icon className="h-[18px] w-[18px]" />
                  </div>
                  <div>
                    <h3 className="text-[15.5px] font-semibold text-text-primary">{feature.title}</h3>
                    <p className="mt-1.5 text-[13.5px] leading-relaxed text-text-secondary">{feature.body}</p>
                  </div>
                </div>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* THESIS / TRUST STATEMENT */}
      <section className="bg-navy-950 px-6 py-20 md:py-24">
        <Reveal className="mx-auto flex max-w-3xl flex-col items-center gap-6 text-center">
          <ProofSeal size={56} className="text-warning" />
          <p className="font-display text-[24px] italic leading-relaxed text-white md:text-[28px]">
            "Kepercayaan pasangan client dibangun dari apa yang bisa mereka lihat sendiri —
            bukan sekadar yang mereka dengar dari tim WO."
          </p>
          <p className="text-[13.5px] text-white/50">Prinsip di balik cara ElProof dirancang</p>
        </Reveal>
      </section>

      {/* CTA */}
      <section className="bg-gradient-to-br from-navy-900 to-navy-950 px-6 py-20 text-center md:py-24">
        <Reveal className="mx-auto max-w-2xl">
          <h2 className="font-display text-[28px] font-semibold text-white md:text-[34px]">
            Siap membuat persiapan pernikahan klien Anda lebih terjaga?
          </h2>
          <p className="mt-4 text-[15px] leading-relaxed text-white/70">
            Ceritakan kebutuhan bisnis WO Anda — tim kami akan membantu proses aktivasi akun secara langsung.
          </p>
          <Link
            to={ROUTE_PATHS.homepageContact}
            className="mt-8 inline-flex h-11 items-center justify-center gap-2 rounded-md bg-warning px-7 text-[14.5px] font-semibold text-navy-950 transition-colors hover:bg-warning/90"
          >
            Hubungi Kami <ArrowRight className="h-4 w-4" />
          </Link>
        </Reveal>
      </section>
    </div>
  );
}
