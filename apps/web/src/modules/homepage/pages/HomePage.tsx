import { lazy, Suspense } from "react";
import { Link } from "react-router-dom";
import { Check, MessageCircle } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Container } from "@/modules/homepage/components/Container";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { ctaClass } from "@/modules/homepage/components/cta";
import { HeroScreens } from "@/modules/homepage/components/home/HeroScreens";
import { HomeFaq } from "@/modules/homepage/components/home/HomeFaq";
import { Journey } from "@/modules/homepage/components/home/Journey";
import { PreviewFallback } from "@/modules/homepage/components/home/PreviewWindow";
import { Pricing } from "@/modules/homepage/components/home/Pricing";
import { whatsappUrl } from "@/modules/homepage/data/contact";
import { usePageMeta } from "@/shared/hooks/usePageMeta";
import { PUBLIC_PAGES } from "@/app/routes/public-pages";

// The try-it preview is interactive and below the fold: its own chunk; the fallback already carries the frame.
const AppPreview = lazy(() => import("@/modules/homepage/components/home/AppPreview"));

const H2 = "text-[30px] leading-[1.1] font-extrabold tracking-[-0.03em] md:text-[44px]";
const LEAD = "mt-4 text-[17px] text-muted-foreground md:text-[18px]";

/**
 * What the owner gets, under the hero: each line is an outcome backed by a feature that exists (Portal Klien, the
 * dashboard's due terms, the project margin, the shareable portal). No numbers we cannot prove.
 */
const BENEFITS = [
  "Tidak perlu lagi kirim rekap manual ke klien",
  "Tagihan klien yang jatuh tempo langsung terlihat",
  "Margin tiap proyek kelihatan, bukan tebakan",
  "Portal Klien bisa Anda tunjukkan saat presentasi ke calon pasangan",
];

/** The second pillar: trust that sells. Each point says how ElProof gets there. */
const TRUST = [
  {
    title: "Transparansi jadi nilai jual paket Anda",
    body: "Saat presentasi, tunjukkan Portal Klien yang akan dipegang pasangan. WO yang bisa memperlihatkan progres dan bukti terlihat lebih profesional dibanding yang hanya mengandalkan grup WhatsApp.",
  },
  {
    title: "Tidak ada lagi salah paham soal pembayaran",
    body: "Setiap pembayaran tercatat beserta bukti transfernya dan diverifikasi tim. Pasangan melihat status yang sama, jadi pertanyaan \u201csudah masuk belum?\u201d tidak perlu dijawab berulang.",
  },
  {
    title: "Pasangan yang tenang lebih mudah merekomendasikan Anda",
    body: "Mereka tahu apa yang sudah beres dan apa yang sedang ditangani, tanpa harus menagih kabar. Pengalaman seperti itu yang mereka ceritakan ke teman.",
  },
];

function WhatsAppCta({ variant = "outline" }: { variant?: "outline" | "outline-inverted" }) {
  return (
    <a href={whatsappUrl()} target="_blank" rel="noreferrer" className={ctaClass(variant, "md", "w-full sm:w-auto")}>
      <MessageCircle aria-hidden className="size-[18px]" /> Tanya lewat WhatsApp
    </a>
  );
}

export default function HomePage() {
  usePageMeta(PUBLIC_PAGES.home);

  return (
    <>
      {/* Hero: the headline is the LCP element; the two screens below it are plain HTML. */}
      <section className="overflow-hidden pt-12 pb-12 md:pt-20 md:pb-16">
        <Container>
          <div className="grid gap-8 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)] lg:items-end lg:gap-12">
            <h1 className="max-w-[15ch] text-[40px] leading-[1.02] font-extrabold tracking-[-0.04em] sm:text-[60px] lg:text-[72px]">
              Kerja tim lebih ringan, klien makin percaya.
            </h1>
            <div className="lg:pb-2">
              <p className="max-w-[46ch] text-[17px] text-muted-foreground md:text-[18px]">
                Catat tahapan, vendor, dan pembayaran di satu tempat. Pasangan memantau persiapannya sendiri lewat tautan
                WhatsApp, jadi tim Anda berhenti menjawab pertanyaan yang sama dan punya waktu untuk lebih banyak acara.
              </p>
              <div className="mt-7 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
                <Link to={ROUTE_PATHS.signup} className={ctaClass("primary", "md", "w-full sm:w-auto")}>
                  Coba gratis 14 hari
                </Link>
                <WhatsAppCta />
              </div>
              <p className="mt-4 text-[14px] text-muted-foreground">Daftar dengan akun Google. Tanpa kartu kredit.</p>
            </div>
          </div>
          <div className="mt-12 md:mt-16">
            <HeroScreens />
          </div>
        </Container>
      </section>

      {/* Benefit strip: what the owner gets, one line each. */}
      <section aria-label="Yang Anda dapatkan" className="border-y border-border bg-muted">
        <Container>
          <ul className="grid gap-x-8 gap-y-3 py-6 text-[15px] sm:grid-cols-2 lg:grid-cols-4">
            {BENEFITS.map((fact) => (
              <li key={fact} className="flex items-start gap-2.5">
                <Check aria-hidden className="mt-1 size-4 shrink-0 text-seal" strokeWidth={2.8} />
                <span>{fact}</span>
              </li>
            ))}
          </ul>
        </Container>
      </section>

      <section id="cara-kerja" className="scroll-mt-24 py-20 md:py-28">
        <Container>
          <div className="reveal max-w-[680px]">
            <h2 className={H2}>Pekerjaan tim lebih ringan, dari H-180 sampai hari H</h2>
            <p className={LEAD}>
              Yang biasanya tersebar di chat, spreadsheet, dan galeri HP kini ada di satu tempat. Tim mencatat sekali, sisanya
              dihitung dan ditampilkan ElProof di satu beranda.
            </p>
          </div>
          <Journey />
        </Container>
      </section>

      <section aria-labelledby="pratinjau-judul" className="pb-20 md:pb-28">
        <Container>
          <div className="reveal max-w-[680px]">
            <h2 id="pratinjau-judul" className={H2}>
              Coba sendiri, tanpa mendaftar
            </h2>
            <p className={LEAD}>
              Pratinjau ini memakai contoh data. Pilih tab untuk melihat ringkasan proyek, termin beserta buktinya, dan
              tampilan untuk klien.
            </p>
          </div>
          <div className="mt-10">
            <Suspense fallback={<PreviewFallback />}>
              <AppPreview />
            </Suspense>
          </div>
        </Container>
      </section>

      <section className="relative overflow-hidden bg-primary py-20 text-primary-foreground md:py-28">
        <div aria-hidden className="pointer-events-none absolute -right-24 -bottom-28 opacity-[0.16] md:-right-10">
          <ProofSeal size={420} className="text-seal-on-primary" />
        </div>
        <Container className="relative">
          <div className="reveal max-w-[680px]">
            <h2 className={H2}>Klien lebih percaya, penjualan ikut naik</h2>
            <p className="mt-4 text-[17px] text-on-primary-muted md:text-[18px]">
              Kepercayaan pasangan adalah aset terbesar WO. ElProof membuatnya terlihat: setiap tahapan dan pembayaran
              punya bukti yang bisa mereka buka sendiri.
            </p>
          </div>
          <ul className="reveal mt-12 grid gap-10 md:grid-cols-3 md:gap-0 md:divide-x md:divide-on-primary-muted/30">
            {TRUST.map((item, i) => (
              <li key={item.title} className={i === 0 ? "md:pr-8" : i === 1 ? "md:px-8" : "md:pl-8"}>
                <Check aria-hidden className="size-5 text-seal-on-primary" strokeWidth={2.6} />
                <h3 className="mt-3 text-[20px] leading-snug font-bold tracking-[-0.02em]">{item.title}</h3>
                <p className="mt-2.5 text-[15.5px] text-on-primary-muted">{item.body}</p>
              </li>
            ))}
          </ul>
          <div className="mt-12 flex flex-col gap-3 sm:flex-row">
            <Link to={ROUTE_PATHS.signup} className={ctaClass("inverted", "md", "w-full sm:w-auto")}>
              Coba gratis 14 hari
            </Link>
            <WhatsAppCta variant="outline-inverted" />
          </div>
        </Container>
      </section>

      <section id="harga" className="scroll-mt-24 py-20 md:py-28">
        <Container>
          <Pricing headingClassName={H2} leadClassName={LEAD} />
        </Container>
      </section>

      <section aria-labelledby="faq-judul" className="pb-20 md:pb-28">
        <Container className="grid gap-8 lg:grid-cols-[0.8fr_1.2fr] lg:gap-16">
          <div>
            <h2 id="faq-judul" className={H2}>
              Pertanyaan umum
            </h2>
            <p className={LEAD}>Jawaban singkat untuk pertanyaan yang paling sering kami terima.</p>
          </div>
          <HomeFaq />
        </Container>
      </section>

      <section className="border-t border-border bg-muted py-20 md:py-24">
        <Container className="reveal grid items-center gap-8 lg:grid-cols-[1fr_auto]">
          <div>
            <h2 className="max-w-[22ch] text-[32px] leading-[1.08] font-extrabold tracking-[-0.035em] md:text-[48px]">
              Tangani lebih banyak pernikahan, tanpa menambah kerepotan.
            </h2>
            <p className="mt-4 text-[17px] text-muted-foreground">Gratis 14 hari, tanpa kartu kredit. Daftar dengan akun Google.</p>
          </div>
          <div className="flex flex-col gap-3 sm:flex-row lg:flex-col">
            <Link to={ROUTE_PATHS.signup} className={ctaClass("primary", "md", "w-full sm:w-auto")}>
              Coba gratis 14 hari
            </Link>
            <WhatsAppCta />
          </div>
        </Container>
      </section>
    </>
  );
}
