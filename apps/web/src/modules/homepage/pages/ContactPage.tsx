import { Mail, Phone, MessageCircle, Globe, MapPin, Clock, ArrowRight } from "lucide-react";
import { CONTACT } from "@/modules/homepage/data/contact";

export default function ContactPage() {
  return (
    <div>
      <div className="border-b border-border bg-surface px-6 py-14">
        <div className="mx-auto max-w-3xl">
          <p className="text-[12.5px] font-semibold uppercase tracking-wide text-navy-900/70">Mulai Bersama Kami</p>
          <h1 className="mt-3 font-display text-[32px] font-semibold text-text-primary md:text-[38px]">Hubungi Kami</h1>
          <p className="mt-3 max-w-xl text-[14.5px] leading-relaxed text-text-secondary">
            Tertarik menggunakan ElProof untuk bisnis wedding organizer Anda? Aktivasi akun dibantu langsung oleh
            tim kami — ceritakan kebutuhan Anda lewat salah satu kanal berikut.
          </p>
        </div>
      </div>

      <div className="px-6 py-14">
        <div className="mx-auto grid max-w-4xl gap-5 sm:grid-cols-2">
          <div className="flex flex-col rounded-xl border border-border bg-surface p-6">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-navy-900/[0.07] text-navy-900">
              <Phone className="h-[18px] w-[18px]" />
            </div>
            <h2 className="mt-4 text-[15px] font-semibold text-text-primary">Telepon</h2>
            <p className="mt-1 text-[14px] font-medium text-navy-900">{CONTACT.phoneDisplay}</p>
            <p className="mt-2 flex-1 text-[13px] leading-relaxed text-text-secondary">
              Bisa dihubungi lewat telepon langsung atau chat WhatsApp.
            </p>
            <div className="mt-4 flex items-center gap-4">
              <a
                href={`tel:+${CONTACT.phoneNumber}`}
                className="inline-flex items-center gap-1.5 text-[13.5px] font-semibold text-navy-900 hover:underline"
              >
                Telepon <ArrowRight className="h-3.5 w-3.5" />
              </a>
              <a
                href={`https://wa.me/${CONTACT.phoneNumber}`}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1.5 text-[13.5px] font-semibold text-navy-900 hover:underline"
              >
                <MessageCircle className="h-3.5 w-3.5" /> WhatsApp
              </a>
            </div>
          </div>

          <div className="flex flex-col rounded-xl border border-border bg-surface p-6">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-navy-900/[0.07] text-navy-900">
              <Mail className="h-[18px] w-[18px]" />
            </div>
            <h2 className="mt-4 text-[15px] font-semibold text-text-primary">Email</h2>
            <p className="mt-1 text-[14px] font-medium text-navy-900">{CONTACT.email}</p>
            <p className="mt-2 flex-1 text-[13px] leading-relaxed text-text-secondary">
              Untuk pertanyaan detail, kerja sama, atau kebutuhan dokumentasi tertulis.
            </p>
            <a
              href={`mailto:${CONTACT.email}`}
              className="mt-4 inline-flex items-center gap-1.5 text-[13.5px] font-semibold text-navy-900 hover:underline"
            >
              Kirim Email <ArrowRight className="h-3.5 w-3.5" />
            </a>
          </div>

          <div className="flex flex-col rounded-xl border border-border bg-surface p-6">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-navy-900/[0.07] text-navy-900">
              <MapPin className="h-[18px] w-[18px]" />
            </div>
            <h2 className="mt-4 text-[15px] font-semibold text-text-primary">Alamat Kantor</h2>
            <p className="mt-1 text-[14px] font-medium text-navy-900">{CONTACT.address}</p>
            <p className="mt-2 flex-1 text-[13px] leading-relaxed text-text-secondary">Kunjungan disarankan membuat janji terlebih dahulu.</p>
            <a
              href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(CONTACT.address)}`}
              target="_blank"
              rel="noreferrer"
              className="mt-4 inline-flex items-center gap-1.5 text-[13.5px] font-semibold text-navy-900 hover:underline"
            >
              Lihat di Peta <ArrowRight className="h-3.5 w-3.5" />
            </a>
          </div>

          <div className="flex flex-col rounded-xl border border-border bg-surface p-6">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-navy-900/[0.07] text-navy-900">
              <Clock className="h-[18px] w-[18px]" />
            </div>
            <h2 className="mt-4 text-[15px] font-semibold text-text-primary">Jam Layanan</h2>
            <p className="mt-1 text-[14px] font-medium text-navy-900">{CONTACT.hours}</p>
            <p className="mt-2 flex-1 text-[13px] leading-relaxed text-text-secondary">
              Di luar jam ini, pesan Anda tetap kami balas sesegera mungkin.
            </p>
          </div>

          <div className="flex flex-col rounded-xl border border-border bg-surface p-6 sm:col-span-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-navy-900/[0.07] text-navy-900">
              <Globe className="h-[18px] w-[18px]" />
            </div>
            <h2 className="mt-4 text-[15px] font-semibold text-text-primary">Website</h2>
            <p className="mt-1 text-[14px] font-medium text-navy-900">{CONTACT.website}</p>
            <a
              href={`https://${CONTACT.website}`}
              target="_blank"
              rel="noreferrer"
              className="mt-4 inline-flex items-center gap-1.5 text-[13.5px] font-semibold text-navy-900 hover:underline"
            >
              Kunjungi Website <ArrowRight className="h-3.5 w-3.5" />
            </a>
          </div>
        </div>

        <div className="mx-auto mt-6 max-w-4xl rounded-xl border border-border bg-surface-muted/60 p-6">
          <h3 className="text-[14.5px] font-semibold text-text-primary">Sudah punya akun WO Console?</h3>
          <p className="mt-1.5 text-[13.5px] leading-relaxed text-text-secondary">
            Anda tidak perlu menghubungi kami untuk masuk sehari-hari — langsung saja masuk dengan akun yang
            sudah dibuatkan tim Anda.
          </p>
        </div>
      </div>
    </div>
  );
}
