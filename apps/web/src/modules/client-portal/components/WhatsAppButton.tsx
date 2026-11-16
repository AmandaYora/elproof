import { MessageCircle } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { cn } from "@/shared/lib/cn";
import { whatsappUrl } from "@/shared/lib/access-link";
import type { PortalContact } from "@/modules/client-portal/lib/usePortalContact";

interface Props {
  contact: PortalContact;
  /** The whole prefilled message. */
  message: string;
  label: string;
  /** solid = the green WhatsApp button (one per card at most); outline = a quieter one beside another action. */
  tone?: "solid" | "outline";
  size?: "sm" | "default" | "lg";
  className?: string;
}

/**
 * A WhatsApp button with the message already typed for the moment it sits in (a term, a kendala, a missing bank
 * account). Without a usable number it renders nothing (never a dead button); the "Hubungi kami" card and the
 * bank-account note say once that no contact is available.
 */
export function WhatsAppButton({ contact, message, label, tone = "outline", size = "default", className }: Props) {
  if (!contact.phone) return null;
  return (
    <Button
      size={size}
      variant={tone === "solid" ? "default" : "outline"}
      className={cn(tone === "solid" && "bg-whatsapp text-whatsapp-foreground hover:bg-whatsapp-hover", className)}
      nativeButton={false}
      render={<a href={whatsappUrl(contact.phone, message)} target="_blank" rel="noopener noreferrer" />}
    >
      <MessageCircle aria-hidden className={tone === "outline" ? "text-whatsapp" : undefined} />
      {label}
    </Button>
  );
}
