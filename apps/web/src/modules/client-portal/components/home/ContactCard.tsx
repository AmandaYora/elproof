import { MessageCircle, Phone } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { whatsappNumber, whatsappUrl } from "@/shared/lib/access-link";
import { greetingMessage } from "@/modules/client-portal/lib/payment-message";

/** Who to talk to, one tap away by WhatsApp or phone. */
export function ContactCard({
  picName,
  phone,
  businessName,
  clientName,
  projectName,
}: {
  picName: string;
  /** The number a button should use (PIC's, else the business's); undefined when there is none. */
  phone: string | undefined;
  businessName: string;
  clientName: string;
  projectName: string;
}) {
  const digits = whatsappNumber(phone);
  const name = picName || businessName || "Wedding organizer Anda";
  return (
    <section aria-labelledby="contact-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="contact-heading" className="text-[15px] font-semibold text-foreground">Hubungi kami</h2>
      <div className="mt-3 flex items-center gap-3">
        <span aria-hidden className="flex size-11 shrink-0 items-center justify-center rounded-full bg-navy-900 text-[15px] font-semibold text-white">
          {name.trim().charAt(0).toUpperCase()}
        </span>
        <div className="min-w-0">
          <p className="truncate text-[14px] font-semibold text-foreground">{name}</p>
          <p className="truncate text-[12.5px] text-muted-foreground">{picName ? `Penanggung jawab${businessName ? ` · ${businessName}` : ""}` : "Wedding organizer"}</p>
        </div>
      </div>
      {digits ? (
        <div className="mt-4 grid grid-cols-2 gap-2.5">
          <Button
            size="lg"
            nativeButton={false}
            render={<a href={whatsappUrl(phone, greetingMessage(businessName, clientName, projectName))} target="_blank" rel="noopener noreferrer" />}
          >
            <MessageCircle aria-hidden /> WhatsApp
          </Button>
          <Button size="lg" variant="outline" nativeButton={false} render={<a href={`tel:+${digits}`} />}>
            <Phone aria-hidden /> Telepon
          </Button>
        </div>
      ) : (
        <p className="mt-3 text-[13px] text-muted-foreground">Nomor kontak belum tersedia. Hubungi wedding organizer Anda langsung.</p>
      )}
    </section>
  );
}
