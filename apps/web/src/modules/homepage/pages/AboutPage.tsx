import { Link } from "react-router-dom";
import { ClipboardList, Handshake, Eye, ShieldCheck, Layers, ArrowRight } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { Reveal } from "@/modules/homepage/components/Reveal";

const PARTIES = [
  {
    icon: ClipboardList,
    title: "Tim WO Console",
    body: "Mencatat setiap project, vendor, milestone, dan pembayaran di satu tempat, alih-alih tercecer di grup chat dan spreadsheet terpisah.",
  },
  {
    icon: Handshake,
    title: "Vendor",
    body: "Punya rekam jejak keterlibatan yang jelas per project — status booking, tenggat kerja, dan riwayat pembayaran.",
  },
  {
    icon: Eye,
    title: "Pasangan Client",
    body: "Memantau progress persiapan pernikahan mereka sendiri lewat Client Portal yang read-only, tanpa perlu bertanya berulang kali ke tim WO.",
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
    <div>
      <div className="border-b border-border bg-surface px-6 py-14">
        <div className="mx-auto max-w-3xl">
          <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Kenali Kami</p>
          <h1 className="mt-3 font-display text-[32px] font-semibold text-text-primary md:text-[38px]">
            Tentang ElProof
          </h1>
          <p className="mt-3 max-w-xl text-[14.5px] leading-relaxed text-text-secondary">
            ElProof dibangun untuk satu masalah spesifik: persiapan pernikahan melibatkan banyak pihak, tapi
            informasinya sering tercecer. Kami membuat semuanya tercatat, dan bisa dibuktikan.
          </p>
        </div>
      </div>

      <div className="px-6 py-14">
        <div className="mx-auto flex max-w-3xl flex-col gap-12">
          <Reveal>
            <h2 className="text-[17px] font-semibold text-text-primary">Apa itu ElProof</h2>
            <p className="mt-3 text-[14.5px] leading-relaxed text-text-secondary">
              ElProof adalah platform perangkat lunak berbasis langganan (SaaS) untuk bisnis wedding organizer
              (WO). Lewat WO Console, tim WO mendokumentasikan project pernikahan yang mereka tangani — mulai
              dari kontrak vendor, timeline milestone, hingga status pembayaran — dan membagikan sebagian
              informasi tersebut kepada pasangan client terkait lewat Client Portal.
            </p>
          </Reveal>

          <Reveal delay={60}>
            <h2 className="text-[17px] font-semibold text-text-primary">Mengapa Kami Ada</h2>
            <p className="mt-3 text-[14.5px] leading-relaxed text-text-secondary">
              Persiapan pernikahan biasanya melibatkan banyak vendor dan koordinasi yang panjang, dan
              informasinya kerap tersebar di grup chat, catatan pribadi, atau ingatan masing-masing pihak.
              Akibatnya, pasangan client sering harus bertanya berulang kali hanya untuk tahu sejauh mana
              persiapan mereka berjalan. ElProof menggantikan cara kerja itu dengan satu sumber informasi yang
              sama-sama bisa diakses oleh tim WO dan client-nya, sesuai peran masing-masing.
            </p>
          </Reveal>

          <Reveal delay={90}>
            <h2 className="text-[17px] font-semibold text-text-primary">Satu Platform, Tiga Pihak</h2>
            <div className="mt-5 grid gap-4 sm:grid-cols-3">
              {PARTIES.map((party) => (
                <div key={party.title} className="rounded-xl border border-border bg-surface p-5">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-navy-900/[0.07] text-navy-900">
                    <party.icon className="h-[17px] w-[17px]" />
                  </div>
                  <h3 className="mt-3.5 text-[14.5px] font-semibold text-text-primary">{party.title}</h3>
                  <p className="mt-1.5 text-[13px] leading-relaxed text-text-secondary">{party.body}</p>
                </div>
              ))}
            </div>
          </Reveal>

          <Reveal delay={120}>
            <h2 className="text-[17px] font-semibold text-text-primary">Prinsip yang Kami Pegang</h2>
            <div className="mt-5 flex flex-col gap-4">
              {PRINCIPLES.map((principle) => (
                <div key={principle.title} className="flex gap-4 rounded-xl border border-border bg-surface p-5">
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-navy-900 text-white">
                    <principle.icon className="h-[16px] w-[16px]" />
                  </div>
                  <div>
                    <h3 className="text-[14.5px] font-semibold text-text-primary">{principle.title}</h3>
                    <p className="mt-1 text-[13px] leading-relaxed text-text-secondary">{principle.body}</p>
                  </div>
                </div>
              ))}
            </div>
          </Reveal>
        </div>
      </div>

      <section className="bg-navy-950 px-6 py-16">
        <Reveal className="mx-auto flex max-w-2xl flex-col items-center gap-5 text-center">
          <ProofSeal size={48} className="text-warning" />
          <p className="font-display text-[21px] italic leading-relaxed text-white md:text-[24px]">
            "Kepercayaan pasangan client dibangun dari apa yang bisa mereka lihat sendiri — bukan sekadar yang
            mereka dengar dari tim WO."
          </p>
          <p className="text-[13px] text-white/50">Prinsip di balik cara ElProof dirancang</p>
          <Link
            to={ROUTE_PATHS.homepageContact}
            className="mt-2 inline-flex h-10 items-center justify-center gap-2 rounded-md bg-warning px-6 text-[13.5px] font-semibold text-navy-950 transition-colors hover:bg-warning/90"
          >
            Hubungi Kami <ArrowRight className="h-4 w-4" />
          </Link>
        </Reveal>
      </section>
    </div>
  );
}
