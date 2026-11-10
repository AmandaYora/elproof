import { lazy, Suspense } from "react";
import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Container } from "@/modules/homepage/components/Container";
import { ctaClass } from "@/modules/homepage/components/cta";
import { HeroPhone } from "@/modules/homepage/components/home/HeroPhone";
import { HomeFaq } from "@/modules/homepage/components/home/HomeFaq";
import { PreviewFallback } from "@/modules/homepage/components/home/PreviewWindow";
import { Pricing } from "@/modules/homepage/components/home/Pricing";
import { StoryStatic } from "@/modules/homepage/components/home/StoryStatic";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

// Interactive parts below the fold load in their own chunks; their fallbacks already carry the text.
const Story = lazy(() => import("@/modules/homepage/components/home/Story"));
const AppPreview = lazy(() => import("@/modules/homepage/components/home/AppPreview"));

const H2 = "text-[30px] leading-[1.1] font-extrabold tracking-[-0.03em] md:text-[42px]";
const LEAD = "mt-4 text-[17px] text-muted-foreground md:text-[18px]";

const FACTS = [
  { term: "Jadwal", text: "dipantau tim dan klien" },
  { term: "Termin", text: "dengan bukti transfer" },
  { term: "Vendor", text: "dan venue per proyek" },
  { term: "Portal", text: "klien dari tautan WhatsApp" },
];

const SCREENS = [
  {
    who: "Untuk tim WO",
    name: "WO Console",
    points: [
      "Semua proyek dan tenggatnya di satu beranda",
      "Vendor, venue, dan kendala per proyek",
      "Verifikasi bukti pembayaran klien",
      "Peran dan akses untuk setiap anggota tim",
    ],
  },
  {
    who: "Untuk pasangan",
    name: "Portal Klien",
    points: [
      "Hitung mundur dan tahapan persiapan",
      "Tagihan, jatuh tempo, dan buktinya",
      "Dibuka dari tautan WhatsApp, tanpa memasang aplikasi",
      "Hanya yang Anda pilih untuk ditampilkan",
    ],
  },
];

export default function HomePage() {
  usePageMeta({
    title: "Bukti kerja WO yang bisa dilihat klien",
    description:
      "ElProof membantu wedding organizer mencatat tahapan, vendor, dan termin pembayaran beserta buktinya, lalu menunjukkannya ke klien lewat Portal Klien.",
  });

  return (
    <>
      {/* Hero: the headline is the LCP element; the phone is plain HTML. */}
      <section className="pt-12 pb-16 md:pt-20 md:pb-24">
        <Container className="grid items-center gap-14 lg:grid-cols-[1.25fr_0.75fr] lg:gap-16">
          <div>
            <h1 className="max-w-[13ch] text-[40px] leading-[1.04] font-extrabold tracking-[-0.035em] sm:text-[54px] lg:text-[64px]">
              Klien percaya karena bisa melihat buktinya.
            </h1>
            <p className="mt-6 max-w-[540px] text-[18px] text-muted-foreground md:text-[19px]">
              ElProof membantu wedding organizer mencatat setiap tahapan, vendor, dan pembayaran, lalu menunjukkannya ke
              pasangan dengan rapi. Lebih sedikit pertanyaan berulang, lebih banyak kepercayaan.
            </p>
            <div className="mt-8 flex flex-col gap-3 sm:flex-row">
              <Link to={ROUTE_PATHS.signup} className={ctaClass("primary", "md", "w-full sm:w-auto")}>
                Coba gratis 14 hari
              </Link>
              <a href="#cara-kerja" className={ctaClass("outline", "md", "w-full sm:w-auto")}>
                Lihat cara kerjanya
              </a>
            </div>
            <p className="mt-4 text-[14px] text-muted-foreground">Daftar dengan akun Google. Tanpa kartu kredit.</p>
          </div>
          <HeroPhone />
        </Container>
      </section>

      {/* Fact strip: four plain statements, not cards. */}
      <section aria-label="Yang dicatat ElProof" className="bg-muted py-7">
        <Container>
          <ul className="grid gap-x-8 gap-y-2.5 text-[15px] text-muted-foreground sm:grid-cols-2 lg:flex lg:justify-between">
            {FACTS.map((fact) => (
              <li key={fact.term}>
                <strong className="mr-1.5 text-[16px] font-bold text-foreground">{fact.term}</strong>
                {fact.text}
              </li>
            ))}
          </ul>
        </Container>
      </section>

      <section id="cara-kerja" className="scroll-mt-24 py-20 md:py-28">
        <Container>
          <div className="reveal max-w-[640px]">
            <h2 className={H2}>Satu pernikahan, dari kontrak sampai hari H</h2>
            <p className={LEAD}>
              Ikuti satu proyek dari awal. Di setiap tahap terlihat apa yang dicatat tim Anda dan apa yang dilihat
              pasangan.
            </p>
          </div>
          <Suspense fallback={<StoryStatic />}>
            <Story />
          </Suspense>
        </Container>
      </section>

      <section aria-labelledby="pratinjau-judul" className="pb-20 md:pb-28">
        <Container>
          <div className="reveal max-w-[640px]">
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

      <section className="bg-primary py-20 text-primary-foreground md:py-28">
        <Container>
          <div className="reveal max-w-[640px]">
            <h2 className={H2}>Dua layar, satu catatan yang sama</h2>
            <p className="mt-4 text-[17px] text-on-primary-muted md:text-[18px]">
              Tim bekerja di WO Console. Pasangan membuka Portal Klien dari HP. Keduanya membaca data yang sama.
            </p>
          </div>
          <div className="reveal mt-12 grid gap-10 md:grid-cols-2 md:gap-0 md:divide-x md:divide-on-primary-muted/30">
            {SCREENS.map((screen, i) => (
              <div key={screen.name} className={i === 0 ? "md:pr-10" : "md:pl-10"}>
                <p className="text-[14px] font-semibold text-on-primary-muted">{screen.who}</p>
                <h3 className="mt-1 text-[24px] font-bold tracking-[-0.02em]">{screen.name}</h3>
                <ul className="mt-4 grid list-disc gap-2 pl-5 marker:text-on-primary-muted">
                  {screen.points.map((point) => (
                    <li key={point}>{point}</li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </Container>
      </section>

      <section id="harga" className="scroll-mt-24 py-20 md:py-28">
        <Container>
          <div className="max-w-[640px]">
            <h2 className={H2}>Harga terbuka, pilih lamanya</h2>
            <p className={LEAD}>Mulai dengan coba gratis 14 hari. Setelah itu pilih paket sesuai lama yang Anda butuhkan.</p>
          </div>
          <div className="mt-10">
            <Pricing />
          </div>
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

      <section className="bg-muted py-20 text-center md:py-28">
        <Container className="reveal">
          <h2 className="mx-auto max-w-[20ch] text-[32px] leading-[1.08] font-extrabold tracking-[-0.035em] md:text-[50px]">
            Tunjukkan kerja tim Anda, bukan sekadar janjinya.
          </h2>
          <p className="mt-4 text-[17px] text-muted-foreground">Gratis 14 hari, tanpa kartu kredit. Daftar dengan akun Google.</p>
          <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
            <Link to={ROUTE_PATHS.signup} className={ctaClass("primary", "md", "w-full sm:w-auto")}>
              Coba gratis 14 hari
            </Link>
            <Link to={ROUTE_PATHS.homepageContact} className={ctaClass("outline", "md", "w-full sm:w-auto")}>
              Hubungi kami
            </Link>
          </div>
        </Container>
      </section>
    </>
  );
}
