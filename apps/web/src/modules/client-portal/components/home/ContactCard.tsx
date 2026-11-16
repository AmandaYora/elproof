import { Phone } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { whatsappNumber } from "@/shared/lib/access-link";
import { WhatsAppButton } from "@/modules/client-portal/components/WhatsAppButton";
import { askMessage, type PortalContact } from "@/modules/client-portal/lib/usePortalContact";

/** Who to talk to, one tap away by WhatsApp or phone. */
export function ContactCard({ contact }: { contact: PortalContact }) {
  const digits = whatsappNumber(contact.phone);
  return (
    <section aria-labelledby="contact-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="contact-heading" className="text-base font-semibold text-foreground">Hubungi kami</h2>
      <div className="mt-3 flex items-center gap-3">
        <span aria-hidden className="flex size-11 shrink-0 items-center justify-center rounded-full bg-primary text-[15px] font-semibold text-primary-foreground">
          {contact.name.trim().charAt(0).toUpperCase()}
        </span>
        <div className="min-w-0">
          <p className="truncate text-[14px] font-semibold text-foreground">{contact.isPic ? contact.name : contact.businessName || "Wedding organizer Anda"}</p>
          <p className="truncate text-[13px] text-muted-foreground">
            {contact.isPic ? `Penanggung jawab${contact.businessName ? ` · ${contact.businessName}` : ""}` : "Wedding organizer"}
          </p>
        </div>
      </div>
      {digits ? (
        <div className="mt-4 grid grid-cols-2 gap-2.5">
          <WhatsAppButton contact={contact} tone="solid" size="lg" label="WhatsApp" message={askMessage(contact)} />
          <Button size="lg" variant="outline" nativeButton={false} render={<a href={`tel:+${digits}`} />}>
            <Phone aria-hidden /> Telepon
          </Button>
        </div>
      ) : (
        <p className="mt-3 text-[13px] text-muted-foreground">Kontak WO belum tersedia. Hubungi wedding organizer Anda langsung.</p>
      )}
    </section>
  );
}
