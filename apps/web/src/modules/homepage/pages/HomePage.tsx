import { Link } from "react-router-dom";
import { ClipboardList, Handshake, Eye, CalendarCheck, Wallet, ShieldCheck, Receipt, ArrowRight, CheckCircle2 } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { Reveal } from "@/modules/homepage/components/Reveal";
import { cn } from "@/shared/lib/cn";

const PARTIES = [
  {
    icon: ClipboardList,
    title: "Tim WO Console",
    body: "Catat setiap project, vendor, dan pembayaran secara real-time — tidak ada lagi laporan manual yang tercecer.",
    color: "bg-blue-50 text-blue-600 ring-blue-100",
  },
  {
    icon: Handshake,
    title: "Vendor Eksternal",
    body: "Rekam jejak keterlibatan yang jelas: status booking, tenggat kerja, dan riwayat pembayaran per project.",
    color: "bg-amber-50 text-amber-600 ring-amber-100",
  },
  {
    icon: Eye,
    title: "Pasangan Client",
    body: "Portal khusus read-only. Pasangan memantau progress tanpa perlu bertanya berulang kali ke tim Anda.",
    color: "bg-emerald-50 text-emerald-600 ring-emerald-100",
  },
];

const FEATURES = [
  {
    icon: CalendarCheck,
    title: "Timeline & Milestone",
    body: "Setiap tahap persiapan punya milestone dan tenggat yang dipantau bersama.",
    className: "md:col-span-2 md:row-span-1",
  },
  {
    icon: Wallet,
    title: "Manajemen Vendor",
    body: "Kelola kontrak dan pembayaran vendor dalam satu dashboard.",
    className: "md:col-span-1 md:row-span-1",
  },
  {
    icon: ShieldCheck,
    title: "Portal Transparansi",
    body: "Client melihat bukti kerja secara langsung tanpa akses ke data rahasia.",
    className: "md:col-span-1 md:row-span-1",
  },
  {
    icon: Receipt,
    title: "Otomatisasi Tagihan",
    body: "Satu paket langganan dengan pembayaran otomatis untuk seluruh tim.",
    className: "md:col-span-2 md:row-span-1",
  },
];

export default function HomePage() {
  return (
    <div className="flex flex-col bg-background">
      {/* HERO SECTION */}
      <section className="relative overflow-hidden bg-gradient-to-b from-navy-950 to-navy-900 px-6 pt-20 pb-24 md:pt-32 md:pb-36 lg:pt-40">
        {/* Subtle background glow */}
        <div className="pointer-events-none absolute -right-20 -top-20 z-0 h-[500px] w-[500px] rounded-full bg-blue-500/10 opacity-60 blur-[100px]" />
        <div className="pointer-events-none absolute -left-20 bottom-0 z-0 h-[400px] w-[400px] rounded-full bg-emerald-500/10 opacity-60 blur-[100px]" />
        
        <div className="mx-auto flex max-w-7xl flex-col gap-16 lg:grid lg:grid-cols-2 lg:items-center lg:gap-12 xl:gap-20">
          {/* LEFT SIDE: TEXT */}
          <Reveal className="relative z-10 flex flex-col items-center text-center lg:items-start lg:text-left">
            <div className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/10 px-4 py-1.5 text-[13px] font-semibold text-white shadow-sm backdrop-blur-sm transition-transform hover:scale-105">
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-warning opacity-75"></span>
                <span className="relative inline-flex h-2 w-2 rounded-full bg-warning"></span>
              </span>
              Portal Kerja & Transparansi WO
            </div>
            
            <h1 className="mt-8 font-display text-[42px] font-semibold leading-[1.15] tracking-tight text-white md:text-[56px] xl:text-[68px]">
              Setiap detail pernikahan, <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-300 to-indigo-300">tercatat dan terbukti.</span>
            </h1>
            
            <p className="mt-6 max-w-xl text-[16px] leading-relaxed text-white/80 md:text-[18px]">
              ElProof mendokumentasikan progress, vendor, dan pembayaran untuk tim WO — 
              sementara pasangan client memantau langsung persiapan hari bahagia mereka secara mandiri.
            </p>
            
            <div className="mt-10 flex w-full flex-col items-center gap-4 sm:w-auto sm:flex-row lg:justify-start">
              <Link
                to={ROUTE_PATHS.homepageContact}
                className="group flex h-13 w-full items-center justify-center gap-2 rounded-full bg-white px-8 text-[15px] font-bold text-navy-950 shadow-lg shadow-navy-900/20 transition-all hover:-translate-y-1 hover:bg-surface-muted hover:shadow-xl sm:w-auto"
              >
                Jadwalkan Demo <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
              </Link>
              <a
                href="#fitur"
                className="flex h-13 w-full items-center justify-center rounded-full border border-white/20 bg-white/5 px-8 text-[15px] font-semibold text-white shadow-sm backdrop-blur-sm transition-all hover:-translate-y-1 hover:border-white/30 hover:bg-white/10 sm:w-auto"
              >
                Jelajahi Fitur
              </a>
            </div>
          </Reveal>

          {/* RIGHT SIDE: RELEVANT MOCKUP / ANIMATION */}
          <Reveal delay={200} className="relative z-10 mx-auto w-full max-w-lg lg:mx-0 lg:max-w-none">
            <div className="relative flex h-[400px] w-full items-center justify-center lg:h-[500px]">
              
              {/* Decorative background glow */}
              <div className="absolute inset-0 z-0 rounded-full bg-gradient-to-tr from-white/5 to-transparent opacity-30 blur-2xl"></div>
              
              {/* Card 1: Project Header (Top Left Floating) */}
              <div className="absolute left-0 top-6 z-20 w-[260px] animate-pulse rounded-2xl border border-white/10 bg-white/10 p-5 shadow-2xl backdrop-blur-md transition-transform duration-500 hover:scale-105 sm:left-4 lg:-left-4" style={{animationDuration: "4s"}}>
                <div className="flex items-center justify-between border-b border-white/10 pb-3">
                  <span className="text-[12px] font-semibold tracking-wider text-white/50 uppercase">Project Aktif</span>
                  <span className="flex h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]"></span>
                </div>
                <h3 className="mt-4 font-display text-[20px] font-bold text-white">Raka & Dinda</h3>
                <p className="text-[13px] text-white/70">Resepsi • 12 Nov 2026</p>
                <div className="mt-5 flex items-center gap-3">
                  <div className="h-1.5 flex-1 rounded-full bg-white/10 overflow-hidden">
                    <div className="h-full w-[75%] rounded-full bg-gradient-to-r from-blue-400 to-emerald-400 shadow-[0_0_10px_rgba(52,211,153,0.5)]"></div>
                  </div>
                  <span className="text-[12px] font-bold text-white">75%</span>
                </div>
              </div>

              {/* Card 2: Checklist Milestone (Center Right Floating) */}
              <div className="absolute right-0 top-32 z-30 w-[280px] rounded-2xl border border-white/10 bg-white/10 p-5 shadow-2xl backdrop-blur-xl transition-transform duration-500 hover:scale-105 sm:right-4 lg:-right-4">
                <div className="mb-4 flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/20 text-blue-300">
                    <CalendarCheck className="h-4 w-4" />
                  </div>
                  <span className="text-[14px] font-bold text-white">Milestone Persiapan</span>
                </div>
                <div className="flex flex-col gap-3">
                  <div className="flex items-center gap-3 rounded-xl bg-white/5 p-2.5 border border-white/5">
                    <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400"><CheckCircle2 className="h-3 w-3" /></div>
                    <span className="text-[13px] font-medium text-white/90">Venue & Gedung</span>
                  </div>
                  <div className="flex items-center gap-3 rounded-xl bg-white/5 p-2.5 border border-white/5">
                    <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400"><CheckCircle2 className="h-3 w-3" /></div>
                    <span className="text-[13px] font-medium text-white/90">DP Katering Lunas</span>
                  </div>
                  <div className="flex items-center gap-3 rounded-xl bg-white/5 p-2.5 border border-white/5 opacity-60">
                    <div className="h-5 w-5 shrink-0 rounded-full border-2 border-white/20"></div>
                    <span className="text-[13px] font-medium text-white/90">Fitting Baju Pengantin</span>
                  </div>
                </div>
              </div>

              {/* Card 3: Vendor Payment (Bottom Left Floating) */}
              <div className="absolute bottom-6 left-8 z-40 w-[240px] animate-pulse rounded-2xl border border-white/10 bg-white/10 p-5 shadow-2xl backdrop-blur-md transition-transform duration-500 hover:scale-105 sm:left-12 lg:left-4" style={{animationDuration: "5s", animationDelay: "1s"}}>
                <div className="mb-3 flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-500/20 text-amber-300">
                    <Wallet className="h-4 w-4" />
                  </div>
                  <span className="text-[14px] font-bold text-white">Vendor Katering</span>
                </div>
                <div className="flex items-end justify-between mt-3">
                  <div className="flex flex-col">
                    <span className="text-[11px] text-white/50 uppercase tracking-wider">Status</span>
                    <span className="mt-1 text-[16px] font-bold text-emerald-400">LUNAS</span>
                  </div>
                  <div className="flex h-7 w-7 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400 border border-emerald-500/30">
                    <CheckCircle2 className="h-4 w-4" />
                  </div>
                </div>
              </div>

            </div>
          </Reveal>
        </div>
      </section>

      {/* PARTIES SECTION */}
      <section className="px-6 py-24 bg-white">
        <div className="mx-auto max-w-6xl">
          <Reveal className="text-center max-w-2xl mx-auto">
            <p className="text-[13px] font-bold uppercase tracking-widest text-navy-900/60">Satu Platform, Tiga Pihak</p>
            <h2 className="mt-4 font-display text-[32px] font-semibold tracking-tight text-navy-950 md:text-[40px]">
              Menjembatani semua yang terlibat.
            </h2>
          </Reveal>

          <div className="mt-16 grid gap-8 md:grid-cols-3">
            {PARTIES.map((party, i) => (
              <Reveal key={party.title} delay={i * 100}>
                <div className="group h-full flex flex-col rounded-3xl border border-border bg-surface p-8 transition-all hover:-translate-y-2 hover:shadow-xl hover:shadow-navy-900/5">
                  <div className={cn("flex h-14 w-14 items-center justify-center rounded-2xl ring-4 mb-6 transition-transform group-hover:scale-110", party.color)}>
                    <party.icon className="h-6 w-6" />
                  </div>
                  <h3 className="text-[18px] font-bold text-navy-950">{party.title}</h3>
                  <p className="mt-3 text-[14.5px] leading-relaxed text-text-secondary flex-1">{party.body}</p>
                </div>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* FITUR SECTION (BENTO GRID) */}
      <section id="fitur" className="px-6 py-24 bg-surface-muted/30">
        <div className="mx-auto max-w-6xl">
          <Reveal>
            <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
              <div className="max-w-2xl">
                <p className="text-[13px] font-bold uppercase tracking-widest text-navy-900/60">Fitur Utama</p>
                <h2 className="mt-4 font-display text-[32px] font-semibold tracking-tight text-navy-950 md:text-[40px]">
                  Bekerja rapi, client tenang.
                </h2>
              </div>
            </div>
          </Reveal>

          <div className="mt-12 grid gap-4 auto-rows-[220px] md:grid-cols-3">
            {FEATURES.map((feature, i) => (
              <Reveal key={feature.title} delay={i * 100} className={cn("relative overflow-hidden rounded-3xl border border-border bg-white p-8 transition-all hover:shadow-lg hover:shadow-navy-900/5 group", feature.className)}>
                <div className="absolute right-0 top-0 -mt-8 -mr-8 h-32 w-32 rounded-full bg-gradient-to-br from-navy-50 to-transparent opacity-50 transition-transform group-hover:scale-150" />
                <div className="relative z-10 flex h-full flex-col">
                  <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-navy-900 text-white shadow-md shadow-navy-900/20 mb-auto">
                    <feature.icon className="h-5 w-5" />
                  </div>
                  <div className="mt-6">
                    <h3 className="text-[18px] font-bold text-navy-950">{feature.title}</h3>
                    <p className="mt-2 text-[14px] leading-relaxed text-text-secondary">{feature.body}</p>
                  </div>
                </div>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* TRUST STATEMENT */}
      <section className="px-6 py-32 bg-white">
        <Reveal className="mx-auto flex max-w-4xl flex-col items-center text-center">
          <div className="mb-8 rounded-full bg-navy-50 p-4 ring-8 ring-navy-50/50">
            <ProofSeal size={64} className="text-navy-900" />
          </div>
          <h2 className="font-display text-[28px] font-medium italic leading-snug text-navy-950 md:text-[36px]">
            "Kepercayaan pasangan client dibangun dari apa yang bisa mereka lihat sendiri —
            bukan sekadar yang mereka dengar dari tim WO."
          </h2>
          <div className="mt-8 flex items-center gap-3 text-[14px] font-semibold uppercase tracking-widest text-navy-900/60">
            <span className="h-px w-8 bg-navy-200"></span>
            Prinsip ElProof
            <span className="h-px w-8 bg-navy-200"></span>
          </div>
        </Reveal>
      </section>

      {/* CTA */}
      <section className="px-6 py-24 mb-12">
        <div className="mx-auto max-w-5xl rounded-[2.5rem] bg-navy-950 px-8 py-20 text-center shadow-2xl md:px-20 relative overflow-hidden">
          <div className="absolute inset-0 bg-[linear-gradient(to_right,#ffffff0a_1px,transparent_1px),linear-gradient(to_bottom,#ffffff0a_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]"></div>
          
          <Reveal className="relative z-10 mx-auto max-w-2xl">
            <h2 className="font-display text-[32px] font-semibold tracking-tight text-white md:text-[44px]">
              Siap meningkatkan standar pelayanan WO Anda?
            </h2>
            <p className="mt-6 text-[16px] leading-relaxed text-white/70">
              Bergabunglah dengan puluhan Wedding Organizer yang telah beralih ke cara kerja yang lebih transparan dan terukur bersama ElProof.
            </p>
            <div className="mt-10 flex flex-col items-center justify-center gap-4 sm:flex-row">
              <Link
                to={ROUTE_PATHS.homepageContact}
                className="group flex h-14 w-full items-center justify-center gap-2 rounded-full bg-white px-8 text-[15px] font-bold text-navy-950 transition-all hover:scale-105 hover:bg-surface-muted sm:w-auto"
              >
                Mulai Sekarang <ArrowRight className="h-5 w-5 transition-transform group-hover:translate-x-1" />
              </Link>
            </div>
            
            <div className="mt-10 flex flex-wrap items-center justify-center gap-6 text-[13px] font-medium text-white/50">
              <span className="flex items-center gap-2"><CheckCircle2 className="h-4 w-4 text-emerald-400" /> Setup Cepat</span>
              <span className="flex items-center gap-2"><CheckCircle2 className="h-4 w-4 text-emerald-400" /> Support Dedicated</span>
              <span className="flex items-center gap-2"><CheckCircle2 className="h-4 w-4 text-emerald-400" /> Keamanan Data</span>
            </div>
          </Reveal>
        </div>
      </section>
    </div>
  );
}
