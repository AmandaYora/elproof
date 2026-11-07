import { MessageCircle, Phone } from "lucide-react";
import { whatsappNumber, whatsappUrl } from "@/shared/lib/access-link";
import { greetingMessage } from "@/modules/client-portal/lib/payment-message";
import type { TenantContact } from "@/shared/stores/useContactStore";

/** Who to talk to, one tap away by WhatsApp or phone. */
export function ContactCard({
  picName,
  picPhone,
  contact,
  clientName,
  projectName,
}: {
  picName: string;
  picPhone: string | undefined;
  contact: TenantContact | null;
  clientName: string;
  projectName: string;
}) {
  const phone = picPhone;
  const digits = whatsappNumber(phone);
  const name = picName || contact?.businessName || "Wedding organizer Anda";
  return (
    <section aria-labelledby="contact-heading" className="rounded-2xl border border-border bg-white p-5 sm:p-6">
      <h2 id="contact-heading" className="text-[15px] font-semibold text-text-primary">Hubungi kami</h2>
      <div className="mt-3 flex items-center gap-3">
        <span aria-hidden className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-navy-900 text-[15px] font-semibold text-white">
          {name.trim().charAt(0).toUpperCase()}
        </span>
        <div className="min-w-0">
          <p className="truncate text-[14px] font-semibold text-text-primary">{name}</p>
          <p className="truncate text-[12.5px] text-text-secondary">{picName ? `Penanggung jawab · ${contact?.businessName ?? ""}` : "Wedding organizer"}</p>
        </div>
      </div>
      {digits ? (
        <div className="mt-4 grid grid-cols-2 gap-2.5">
          <a
            href={whatsappUrl(phone, greetingMessage(contact?.businessName ?? "", clientName, projectName))}
            target="_blank"
            rel="noopener noreferrer"
            className="flex min-h-12 items-center justify-center gap-2 rounded-xl bg-navy-900 px-3 text-[14px] font-semibold text-white hover:bg-navy-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy-900"
          >
            <MessageCircle className="h-4 w-4" aria-hidden /> WhatsApp
          </a>
          <a
            href={`tel:+${digits}`}
            className="flex min-h-12 items-center justify-center gap-2 rounded-xl border border-border bg-white px-3 text-[14px] font-semibold text-navy-900 hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy-900"
          >
            <Phone className="h-4 w-4" aria-hidden /> Telepon
          </a>
        </div>
      ) : (
        <p className="mt-3 text-[13px] text-text-secondary">Nomor kontak belum tersedia. Hubungi wedding organizer Anda langsung.</p>
      )}
    </section>
  );
}
