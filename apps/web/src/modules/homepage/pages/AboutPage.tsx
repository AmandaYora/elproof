import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Container } from "@/modules/homepage/components/Container";
import { ctaClass } from "@/modules/homepage/components/cta";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { CONTACT } from "@/modules/homepage/data/contact";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const H2 = "text-[26px] leading-[1.15] font-extrabold tracking-[-0.03em] md:text-[32px]";

const PRINCIPLES = [
  {
    title: "Bukti di setiap angka penting",
    body: "Pembayaran, tahapan, dan dokumen dicatat sebagai data beserta buktinya, bukan janji lisan. Semuanya bisa dilihat kembali kapan saja oleh pihak yang berhak.",
  },
  {
    title: "Klien melihat yang perlu dilihat",
    body: "Pasangan melihat jadwal, tagihan, dan bukti yang Anda pilih untuk ditampilkan. Biaya vendor dan catatan internal tim tetap rahasia. Data setiap usaha juga terpisah: klien hanya melihat proyeknya sendiri, tidak pernah data proyek atau usaha lain.",
  },
  {
    title: "Dibuat untuk cara kerja WO Indonesia",
    body: "Istilahnya istilah WO: termin, vendor, venue, H\u201130. Uang ditulis dalam Rupiah, dan klien membuka Portal Klien dari tautan WhatsApp tanpa memasang aplikasi.",
  },
];

export default function AboutPage() {
  usePageMeta({
    title: "Tentang kami",
    description:
      "Kenapa ElProof dibuat: agar wedding organizer bisa menunjukkan pembayaran, vendor, dan progres persiapan kepada klien, lengkap dengan buktinya.",
  });

  return (
    <article className="py-14 md:py-24">
      <Container>
        <div className="max-w-[68ch]">
          <ProofSeal size={56} className="text-seal" />
          <h1 className="mt-6 text-[38px] leading-[1.06] font-extrabold tracking-[-0.035em] md:text-[52px]">
            Dibuat agar kerja wedding organizer bisa dibuktikan.
          </h1>

          <div className="mt-8 grid gap-5 text-[17px] text-muted-foreground md:text-[18px]">
            <p>
              Sebagian besar wedding organizer bekerja lewat grup WhatsApp dan spreadsheet. Bukti transfer tersimpan di
              galeri HP seseorang, kabar dari vendor ada di obrolan yang berbeda, dan jadwal persiapan sering hanya
              diingat oleh PIC-nya.
            </p>
            <p>
              Akibatnya pasangan bertanya hal yang sama berulang kali: sudah lunas belum, vendor mana yang sudah pasti,
              apa yang perlu disiapkan minggu ini. Setiap jawaban harus dicari ulang oleh tim.
            </p>
            <p className="text-foreground">
              ElProof mengumpulkan semua itu di satu tempat untuk tim, lalu menunjukkan bagian yang perlu dilihat
              pasangan, lengkap dengan buktinya.
            </p>
          </div>

          <section aria-labelledby="prinsip" className="reveal mt-16 md:mt-20">
            <h2 id="prinsip" className={H2}>
              Yang kami pegang
            </h2>
            <div className="mt-6 grid gap-8">
              {PRINCIPLES.map((principle) => (
                <div key={principle.title} className="border-t border-border pt-6">
                  <h3 className="text-[20px] font-bold tracking-[-0.02em]">{principle.title}</h3>
                  <p className="mt-2 text-muted-foreground">{principle.body}</p>
                </div>
              ))}
            </div>
          </section>

          <section aria-labelledby="siapa" className="reveal mt-16 md:mt-20">
            <h2 id="siapa" className={H2}>
              Siapa di balik ElProof
            </h2>
            <p className="mt-4 text-muted-foreground">
              ElProof dikembangkan dan dijalankan oleh tim yang bisa Anda hubungi langsung, lewat email{" "}
              <a href={`mailto:${CONTACT.email}`} className="font-semibold whitespace-nowrap text-foreground underline underline-offset-4">
                {CONTACT.email}
              </a>{" "}
              atau WhatsApp{" "}
              <a
                href={`https://wa.me/${CONTACT.phones[0].number}`}
                className="font-semibold whitespace-nowrap text-foreground underline underline-offset-4"
              >
                {CONTACT.phones[0].display}
              </a>
              . Jam layanan: {CONTACT.hours}.
            </p>
            <p className="mt-3 text-muted-foreground">Alamat: {CONTACT.address}.</p>
          </section>

          <section aria-labelledby="mulai" className="reveal mt-16 rounded-3xl bg-muted p-6 md:mt-20 md:p-10">
            <h2 id="mulai" className={H2}>
              Ingin mencobanya di usaha Anda?
            </h2>
            <p className="mt-3 text-muted-foreground">Gratis 14 hari, tanpa kartu kredit. Daftar dengan akun Google.</p>
            <div className="mt-6 flex flex-col gap-3 sm:flex-row">
              <Link to={ROUTE_PATHS.signup} className={ctaClass("primary", "md", "w-full sm:w-auto")}>
                Coba gratis 14 hari
              </Link>
              <Link to={ROUTE_PATHS.homepageContact} className={ctaClass("outline", "md", "w-full sm:w-auto")}>
                Hubungi kami
              </Link>
            </div>
          </section>
        </div>
      </Container>
    </article>
  );
}
