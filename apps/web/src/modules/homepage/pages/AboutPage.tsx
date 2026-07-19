import { Link } from "react-router-dom";
import { ClipboardList, Handshake, Eye, ShieldCheck, Layers, ArrowUpRight, CheckCircle2 } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { Reveal } from "@/modules/homepage/components/Reveal";
import { cn } from "@/shared/lib/cn";

const PARTIES = [
  {
    icon: ClipboardList,
    title: "Tim WO Console",
    body: "Mencatat setiap project, vendor, milestone, dan pembayaran di satu tempat, alih-alih tercecer di grup chat.",
    color: "bg-blue-50 text-blue-600",
  },
  {
    icon: Handshake,
    title: "Vendor",
    body: "Punya rekam jejak keterlibatan yang jelas per project — status booking, tenggat kerja, dan riwayat.",
    color: "bg-amber-50 text-amber-600",
  },
  {
    icon: Eye,
    title: "Pasangan Client",
    body: "Memantau progress pernikahan mereka lewat Client Portal yang read-only secara mandiri.",
    color: "bg-emerald-50 text-emerald-600",
  },
];

const PRINCIPLES = [
  {
    icon: ShieldCheck,
    title: "Transparansi yang Terukur",
    body: "Setiap progress, bukti kerja, dan status pembayaran dicatat sebagai data — bukan janji lisan — sehingga bisa dilihat kembali kapan saja oleh pihak yang berhak.",
  },
  {
    icon: Layers,
    title: "Data Terpisah per Tenant",
    body: "Setiap bisnis WO yang berlangganan ElProof memiliki ruang data sendiri. Client hanya melihat project miliknya, tidak pernah data project atau tenant lain.",
  },
];

export default function AboutPage() {
  return (
    <div className="bg-background">
      {/* HEADER PAGE */}
      <section className="relative overflow-hidden border-b border-border bg-white px-6 py-20 md:py-28">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1px,transparent_1px),linear-gradient(to_bottom,#f1f5f9_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] opacity-60"></div>
        <div className="mx-auto max-w-4xl text-center relative z-10">
          <Reveal>
            <div className="inline-flex items-center gap-2 rounded-full bg-blue-50 px-3.5 py-1.5 text-[13px] font-bold uppercase tracking-widest text-blue-600">
              Kenali Kami
            </div>
            <h1 className="mt-6 font-display text-[40px] font-semibold tracking-tight text-navy-950 md:text-[56px]">
              Menghadirkan transparansi dalam setiap persiapan.
            </h1>
            <p className="mx-auto mt-6 max-w-2xl text-[16px] leading-relaxed text-text-secondary md:text-[18px]">
              ElProof dibangun untuk satu masalah spesifik: persiapan pernikahan melibatkan banyak pihak, tapi
              informasinya sering tercecer. Kami membuat semuanya tercatat, dan bisa dibuktikan.
            </p>
          </Reveal>
        </div>
      </section>

      {/* STORY SECTION */}
      <section className="px-6 py-24 bg-white">
        <div className="mx-auto max-w-6xl">
          <div className="grid gap-16 md:grid-cols-2 md:items-center">
            <Reveal className="order-2 md:order-1">
              <div className="relative aspect-square w-full max-w-md mx-auto overflow-hidden rounded-3xl border border-border bg-surface md:mx-0 flex flex-col items-center justify-center p-8">
                {/* Background decorative */}
                <div className="absolute inset-0 bg-gradient-to-br from-blue-50/50 to-emerald-50/50"></div>
                
                {/* Visual: from messy chat to organized card */}
                <div className="relative z-10 w-full flex flex-col items-center">
                  
                  {/* Messy chat bubbles (faded/background) */}
                  <div className="absolute left-0 top-0 w-36 rounded-2xl rounded-bl-none bg-white p-3 shadow-sm opacity-60 -rotate-6 animate-pulse" style={{animationDuration: '3s'}}>
                    <div className="h-2 w-16 bg-red-100 rounded mb-2"></div>
                    <div className="h-2 w-24 bg-surface-muted rounded"></div>
                  </div>
                  <div className="absolute right-0 top-8 w-36 rounded-2xl rounded-br-none bg-white p-3 shadow-sm opacity-60 rotate-6 animate-pulse" style={{animationDuration: '4s', animationDelay: '1s'}}>
                    <div className="h-2 w-20 bg-amber-100 rounded mb-2"></div>
                    <div className="h-2 w-16 bg-surface-muted rounded"></div>
                  </div>

                  {/* Organized ElProof Card (Foreground) */}
                  <div className="relative z-20 w-full max-w-[280px] transform transition-transform duration-700 hover:scale-105 rounded-2xl border border-border bg-white shadow-2xl shadow-navy-900/10 p-5 mt-16">
                    <div className="flex items-center gap-3 mb-4 border-b border-border pb-4">
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-navy-50 text-navy-900">
                        <ShieldCheck className="h-5 w-5" />
                      </div>
                      <div>
                        <h4 className="text-[14.5px] font-bold text-navy-950">Single Source of Truth</h4>
                        <p className="text-[11.5px] text-text-secondary mt-0.5">Data terpusat & terverifikasi</p>
                      </div>
                    </div>
                    <div className="flex flex-col gap-3">
                      <div className="flex items-center gap-3 rounded-lg bg-surface-muted/30 p-2 border border-border/50">
                        <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600"><CheckCircle2 className="h-3 w-3" /></div>
                        <div className="h-2 w-32 rounded bg-surface-muted"></div>
                      </div>
                      <div className="flex items-center gap-3 rounded-lg bg-surface-muted/30 p-2 border border-border/50">
                        <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600"><CheckCircle2 className="h-3 w-3" /></div>
                        <div className="h-2 w-24 rounded bg-surface-muted"></div>
                      </div>
                      <div className="flex items-center gap-3 rounded-lg bg-surface-muted/30 p-2 border border-border/50 opacity-50">
                        <div className="h-5 w-5 shrink-0 rounded-full border-2 border-border"></div>
                        <div className="h-2 w-28 rounded bg-surface-muted"></div>
                      </div>
                    </div>
                  </div>

                </div>
              </div>
            </Reveal>
            
            <div className="order-1 flex flex-col justify-center md:order-2">
              <Reveal>
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-navy-900 text-white shadow-md shadow-navy-900/20 mb-6">
                  <ShieldCheck className="h-6 w-6" />
                </div>
                <h2 className="font-display text-[28px] font-semibold tracking-tight text-navy-950 md:text-[36px]">
                  Mengapa Kami Ada
                </h2>
                <div className="mt-6 space-y-5 text-[15.5px] leading-relaxed text-text-secondary">
                  <p>
                    Persiapan pernikahan melibatkan banyak vendor dan koordinasi yang panjang. Seringkali, informasi tersebut tersebar di grup chat, catatan pribadi, atau sebatas ingatan masing-masing pihak.
                  </p>
                  <p>
                    Akibatnya, pasangan client sering harus bertanya berulang kali hanya untuk tahu sejauh mana persiapan mereka berjalan. Ketidakpastian ini memicu kecemasan di momen yang seharusnya membahagiakan.
                  </p>
                  <p className="font-medium text-navy-900">
                    ElProof menggantikan cara kerja itu dengan satu sumber kebenaran (Single Source of Truth).
                  </p>
                </div>
              </Reveal>
            </div>
          </div>
        </div>
      </section>

      {/* PARTIES SECTION */}
      <section className="px-6 py-24 bg-surface-muted/50 border-y border-border">
        <div className="mx-auto max-w-6xl">
          <Reveal className="text-center">
            <h2 className="font-display text-[28px] font-semibold tracking-tight text-navy-950 md:text-[36px]">
              Satu Platform, Tiga Pihak
            </h2>
            <p className="mx-auto mt-4 max-w-2xl text-[16px] text-text-secondary">
              Masing-masing pihak memiliki akses yang sesuai dengan perannya.
            </p>
          </Reveal>

          <div className="mt-16 grid gap-6 md:grid-cols-3">
            {PARTIES.map((party, i) => (
              <Reveal key={party.title} delay={i * 100}>
                <div className="flex h-full flex-col rounded-3xl border border-border bg-white p-8 transition-all hover:-translate-y-1 hover:shadow-lg hover:shadow-navy-900/5">
                  <div className={cn("flex h-12 w-12 items-center justify-center rounded-xl mb-6", party.color)}>
                    <party.icon className="h-6 w-6" />
                  </div>
                  <h3 className="text-[18px] font-bold text-navy-950">{party.title}</h3>
                  <p className="mt-3 text-[14.5px] leading-relaxed text-text-secondary">{party.body}</p>
                </div>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* PRINCIPLES SECTION */}
      <section className="px-6 py-24 bg-white">
        <div className="mx-auto max-w-4xl">
          <Reveal className="text-center mb-16">
            <h2 className="font-display text-[28px] font-semibold tracking-tight text-navy-950 md:text-[36px]">
              Prinsip yang Kami Pegang
            </h2>
          </Reveal>
          
          <div className="flex flex-col gap-6">
            {PRINCIPLES.map((principle, i) => (
              <Reveal key={principle.title} delay={i * 100}>
                <div className="group flex flex-col md:flex-row gap-6 rounded-3xl border border-border bg-surface-muted/30 p-8 transition-colors hover:bg-white hover:shadow-xl hover:shadow-navy-900/5 hover:border-navy-900/10">
                  <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-white text-navy-900 shadow-sm border border-border group-hover:bg-navy-900 group-hover:text-white transition-colors">
                    <principle.icon className="h-6 w-6" />
                  </div>
                  <div>
                    <h3 className="text-[20px] font-bold text-navy-950">{principle.title}</h3>
                    <p className="mt-3 text-[15.5px] leading-relaxed text-text-secondary">{principle.body}</p>
                  </div>
                </div>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      {/* BOTTOM CTA */}
      <section className="px-6 py-24 mb-12">
        <div className="mx-auto max-w-5xl rounded-[2.5rem] bg-navy-950 px-8 py-20 text-center shadow-2xl md:px-20 relative overflow-hidden">
          <div className="absolute inset-0 bg-[linear-gradient(to_right,#ffffff0a_1px,transparent_1px),linear-gradient(to_bottom,#ffffff0a_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)]"></div>
          
          <Reveal className="relative z-10 mx-auto flex max-w-2xl flex-col items-center gap-8 text-center">
            <div className="rounded-full bg-white/10 p-5 text-white ring-8 ring-white/5 backdrop-blur-sm">
              <ProofSeal size={56} />
            </div>
            <h2 className="font-display text-[32px] font-semibold tracking-tight text-white md:text-[40px]">
              Siap untuk bekerja lebih profesional?
            </h2>
            <p className="text-[16px] text-white/70 max-w-xl">
              Tinggalkan cara lama yang rentan miskomunikasi. Mulai kelola proyek WO Anda dengan standar baru bersama ElProof.
            </p>
            <Link
              to={ROUTE_PATHS.homepageContact}
              className="group mt-4 inline-flex h-14 items-center justify-center gap-2 rounded-full bg-white px-8 text-[15px] font-bold text-navy-950 transition-all hover:scale-105 hover:bg-surface-muted"
            >
              Hubungi Tim Kami <ArrowUpRight className="h-5 w-5 transition-transform group-hover:translate-x-1 group-hover:-translate-y-1" />
            </Link>
          </Reveal>
        </div>
      </section>
    </div>
  );
}
