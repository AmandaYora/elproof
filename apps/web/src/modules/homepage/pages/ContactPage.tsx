import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { Mail, MessageCircle, Phone } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Container } from "@/modules/homepage/components/Container";
import { ctaClass } from "@/modules/homepage/components/cta";
import { CONTACT } from "@/modules/homepage/data/contact";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const GREETING = "Halo ElProof, saya ingin bertanya tentang ElProof untuk usaha WO saya: ";

const whatsappUrl = (number: string) => `https://wa.me/${number}?text=${encodeURIComponent(GREETING)}`;
const mapUrl = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(CONTACT.address)}`;

const textLink =
  "inline-flex min-h-11 items-center gap-1.5 rounded-md font-semibold text-foreground underline underline-offset-4 outline-hidden hover:text-primary-hover focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring";

function Row({ term, children }: { term: string; children: ReactNode }) {
  return (
    <div className="border-t border-border py-4 first:border-t-0 first:pt-0 last:pb-0 sm:grid sm:grid-cols-[132px_minmax(0,1fr)] sm:gap-4">
      <dt className="text-[14px] font-medium text-muted-foreground sm:pt-[11px]">{term}</dt>
      <dd className="mt-1 text-[16px] text-foreground sm:mt-0">{children}</dd>
    </div>
  );
}

export default function ContactPage() {
  usePageMeta({ title: "Hubungi kami", description: "Hubungi tim ElProof untuk pertanyaan, bantuan, atau permintaan terkait data Anda." });
  const [main, second] = CONTACT.phones;

  return (
    <div className="bg-background">
      <Container className="py-12 md:py-20">
        <div className="grid gap-12 lg:grid-cols-[minmax(0,1fr)_460px] lg:gap-20">
          <div className="max-w-[32rem]">
            <h1 className="text-[40px] font-extrabold leading-[1.04] tracking-[-0.035em] text-foreground md:text-[56px]">
              Hubungi kami
            </h1>
            <p className="mt-5 max-w-[34rem] text-[17px] leading-relaxed text-muted-foreground md:text-[18px]">
              Tanya soal ElProof, minta dibantu menyiapkan workspace, atau ajukan permintaan data dan refund. Paling cepat
              lewat WhatsApp; kami membalas pada jam layanan.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <a href={whatsappUrl(main.number)} target="_blank" rel="noreferrer" className={ctaClass("primary", "md", "focus-visible:outline-solid")}>
                <MessageCircle aria-hidden className="h-[18px] w-[18px]" /> Tanya lewat WhatsApp
              </a>
              <a href={`mailto:${CONTACT.email}`} className={ctaClass("outline", "md", "focus-visible:outline-solid")}>
                <Mail aria-hidden className="h-[18px] w-[18px]" /> Kirim email
              </a>
            </div>
            <p className="mt-4 text-[14.5px] text-muted-foreground">
              Ke WhatsApp {main.display}. Pesan pembuka sudah terisi; tinggal lengkapi pertanyaan Anda.
            </p>

            <div className="mt-12 border-t-2 border-seal pt-6">
              <h2 className="text-[18px] font-bold tracking-[-0.02em] text-foreground">Sudah memakai ElProof?</h2>
              <p className="mt-2 text-[16px] leading-relaxed text-muted-foreground">
                Anda tidak perlu menghubungi kami untuk masuk sehari-hari. Buka{" "}
                <Link to={ROUTE_PATHS.login} className="font-semibold text-foreground underline underline-offset-4 hover:text-primary-hover">
                  halaman Masuk
                </Link>{" "}
                dengan akun yang didaftarkan pemilik usaha atau tim Anda.
              </p>
            </div>
          </div>

          <section aria-labelledby="kontak-judul" className="rounded-2xl bg-muted p-5 sm:p-7 lg:self-start">
            <h2 id="kontak-judul" className="mb-5 text-[20px] font-extrabold tracking-[-0.03em] text-foreground">
              Kontak ElProof
            </h2>
            <dl>
              <Row term="WhatsApp utama">
                <span className="block font-semibold sm:pt-[11px]">{main.display}</span>
                <span className="flex flex-wrap gap-x-5">
                  <a href={whatsappUrl(main.number)} target="_blank" rel="noreferrer" className={textLink}>
                    <MessageCircle aria-hidden className="h-4 w-4" /> WhatsApp
                  </a>
                  <a href={`tel:+${main.number}`} className={textLink}>
                    <Phone aria-hidden className="h-4 w-4" /> Telepon
                  </a>
                </span>
              </Row>
              {second && (
                <Row term="Nomor kedua">
                  <span className="block font-semibold sm:pt-[11px]">{second.display}</span>
                  <span className="flex flex-wrap gap-x-5">
                    <a href={whatsappUrl(second.number)} target="_blank" rel="noreferrer" className={textLink}>
                      <MessageCircle aria-hidden className="h-4 w-4" /> WhatsApp
                    </a>
                    <a href={`tel:+${second.number}`} className={textLink}>
                      <Phone aria-hidden className="h-4 w-4" /> Telepon
                    </a>
                  </span>
                </Row>
              )}
              <Row term="Email">
                <a href={`mailto:${CONTACT.email}`} className={`${textLink} break-all`}>
                  {CONTACT.email}
                </a>
              </Row>
              <Row term="Jam layanan">
                <span className="block sm:pt-[11px]">{CONTACT.hours}</span>
                <span className="mt-1 block text-[14.5px] text-muted-foreground">
                  Di luar jam ini, pesan Anda tetap kami balas sesegera mungkin.
                </span>
              </Row>
              <Row term="Alamat">
                <span className="block leading-relaxed sm:pt-[9px]">{CONTACT.address}</span>
                <span className="mt-1 block text-[14.5px] text-muted-foreground">
                  Ingin berkunjung? Buat janji dulu agar tim kami bisa bersiap.
                </span>
                <a href={mapUrl} target="_blank" rel="noreferrer" className={textLink}>
                  Buka di peta
                </a>
              </Row>
            </dl>
          </section>
        </div>
      </Container>
    </div>
  );
}
