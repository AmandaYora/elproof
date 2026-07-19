import { Mail, Phone, MessageCircle, MapPin, Clock, ArrowRight, Sparkles } from "lucide-react";
import { CONTACT } from "@/modules/homepage/data/contact";
import { Reveal } from "@/modules/homepage/components/Reveal";

export default function ContactPage() {
  return (
    <div className="bg-background min-h-screen">
      <div className="mx-auto max-w-7xl px-6 py-20 md:py-32">
        <div className="grid gap-16 lg:grid-cols-2 lg:gap-24 items-start">
          
          {/* LEFT SIDE: Heading & Context */}
          <Reveal className="lg:sticky lg:top-32 flex flex-col gap-6">
            <div className="inline-flex items-center gap-2 rounded-full bg-blue-50 px-3.5 py-1.5 text-[13px] font-bold uppercase tracking-widest text-blue-600 w-fit">
              <Sparkles className="h-4 w-4" /> Mulai Bersama Kami
            </div>
            
            <h1 className="font-display text-[40px] font-semibold tracking-tight text-navy-950 md:text-[56px] leading-[1.1]">
              Mari diskusi tentang kebutuhan WO Anda.
            </h1>
            
            <p className="max-w-md text-[16px] leading-relaxed text-text-secondary md:text-[18px]">
              Aktivasi akun ElProof dibantu langsung secara personal. Ceritakan skala project dan kebutuhan Anda, 
              tim kami siap membantu proses onboarding bisnis Anda.
            </p>

            <div className="mt-8 rounded-2xl border border-border bg-surface-muted/50 p-6 shadow-sm">
              <h3 className="text-[15px] font-bold text-navy-950">Sudah punya akun WO Console?</h3>
              <p className="mt-2 text-[14px] leading-relaxed text-text-secondary">
                Anda tidak perlu menghubungi kami untuk masuk sehari-hari — langsung saja login dengan kredensial yang 
                sudah didaftarkan oleh owner / tim Anda.
              </p>
            </div>
          </Reveal>

          {/* RIGHT SIDE: Contact Cards (Bento Style) */}
          <div className="grid gap-4 sm:grid-cols-2">
            {/* Phone Card */}
            <Reveal delay={100} className="col-span-1 sm:col-span-2">
              <div className="group flex flex-col sm:flex-row gap-6 rounded-3xl border border-border bg-white p-8 transition-all hover:border-navy-900/20 hover:shadow-xl hover:shadow-navy-900/5">
                <div className="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-blue-50 text-blue-600 ring-4 ring-blue-50/50 transition-transform group-hover:scale-110">
                  <Phone className="h-6 w-6" />
                </div>
                <div className="flex flex-1 flex-col">
                  <h2 className="text-[18px] font-bold text-navy-950">Telepon & WhatsApp</h2>
                  <p className="mt-2 text-[14px] leading-relaxed text-text-secondary mb-6">
                    Bisa dihubungi lewat telepon langsung atau chat WhatsApp untuk respon tercepat.
                  </p>
                  
                  <div className="flex flex-col gap-4 mt-auto">
                    {CONTACT.phones.map((phone) => (
                      <div key={phone.number} className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-xl bg-surface-muted/50 border border-border">
                        <span className="font-semibold text-navy-900">{phone.display}</span>
                        <div className="flex items-center gap-3">
                          <a href={`tel:+${phone.number}`} className="flex h-10 w-10 items-center justify-center rounded-full bg-white shadow-sm hover:text-navy-900 hover:bg-navy-50 transition-colors" title="Telepon">
                            <Phone className="h-4 w-4" />
                          </a>
                          <a href={`https://wa.me/${phone.number}`} target="_blank" rel="noreferrer" className="flex h-10 items-center gap-2 rounded-full bg-emerald-500 px-4 text-[13px] font-bold text-white shadow-sm hover:bg-emerald-600 transition-colors">
                            <MessageCircle className="h-4 w-4" /> WhatsApp
                          </a>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </Reveal>

            {/* Email Card */}
            <Reveal delay={200} className="flex flex-col rounded-3xl border border-border bg-white p-8 transition-all hover:-translate-y-1 hover:border-navy-900/20 hover:shadow-lg hover:shadow-navy-900/5">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-50 text-amber-600 mb-6">
                <Mail className="h-5 w-5" />
              </div>
              <h2 className="text-[18px] font-bold text-navy-950">Email</h2>
              <p className="mt-2 text-[14px] leading-relaxed text-text-secondary mb-4 flex-1">
                Untuk pertanyaan detail, proposal kerja sama, atau pengiriman dokumen.
              </p>
              <a href={`mailto:${CONTACT.email}`} className="mt-auto group/btn inline-flex items-center justify-between rounded-xl bg-surface-muted/50 p-4 font-semibold text-navy-900 transition-colors hover:bg-surface-muted">
                {CONTACT.email}
                <ArrowRight className="h-4 w-4 transition-transform group-hover/btn:translate-x-1" />
              </a>
            </Reveal>

            {/* Hours Card */}
            <Reveal delay={300} className="flex flex-col rounded-3xl border border-border bg-white p-8 transition-all hover:-translate-y-1 hover:border-navy-900/20 hover:shadow-lg hover:shadow-navy-900/5">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-50 text-emerald-600 mb-6">
                <Clock className="h-5 w-5" />
              </div>
              <h2 className="text-[18px] font-bold text-navy-950">Jam Layanan</h2>
              <p className="mt-2 text-[14px] leading-relaxed text-text-secondary mb-4 flex-1">
                Di luar jam ini, pesan Anda tetap akan kami proses sesegera mungkin.
              </p>
              <div className="mt-auto rounded-xl bg-surface-muted/50 p-4 font-semibold text-navy-900">
                {CONTACT.hours}
              </div>
            </Reveal>

            {/* Address Card */}
            <Reveal delay={400} className="col-span-1 sm:col-span-2 flex flex-col sm:flex-row gap-6 rounded-3xl border border-border bg-white p-8 transition-all hover:border-navy-900/20 hover:shadow-lg hover:shadow-navy-900/5">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-purple-50 text-purple-600">
                <MapPin className="h-5 w-5" />
              </div>
              <div className="flex flex-col flex-1">
                <h2 className="text-[18px] font-bold text-navy-950">Kantor Kami</h2>
                <p className="mt-2 text-[14px] leading-relaxed text-text-secondary">
                  Kunjungan langsung disarankan untuk membuat janji terlebih dahulu agar tim kami dapat bersiap.
                </p>
                <div className="mt-6 flex flex-col sm:flex-row sm:items-center justify-between gap-4 rounded-xl bg-surface-muted/50 p-4">
                  <span className="font-semibold text-navy-900 max-w-[250px] text-[14px] leading-snug">{CONTACT.address}</span>
                  <a href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(CONTACT.address)}`} target="_blank" rel="noreferrer" className="inline-flex h-10 items-center justify-center gap-2 rounded-full bg-white px-5 text-[13px] font-bold text-navy-900 shadow-sm border border-border transition-colors hover:bg-navy-50 shrink-0">
                    Buka di Peta <ArrowRight className="h-3.5 w-3.5" />
                  </a>
                </div>
              </div>
            </Reveal>

          </div>
        </div>
      </div>
    </div>
  );
}
