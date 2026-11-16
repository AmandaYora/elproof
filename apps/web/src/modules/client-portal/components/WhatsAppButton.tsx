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
  /** Without a number: show "Kontak WO belum tersedia" (default) or nothing. */
  quietWhenMissing?: boolean;
  /** Spacing for that quiet line (className styles the button only). */
  quietClassName?: string;
}

/**
 * A WhatsApp button with the message already typed for the moment it sits in (a term, a kendala, a missing bank
 * account). Without a usable number it never shows a dead button: a quiet line instead, or nothing.
 */
export function WhatsAppButton({ contact, message, label, tone = "outline", size = "default", className, quietWhenMissing = true, quietClassName }: Props) {
  if (!contact.phone) {
    return quietWhenMissing ? <p className={cn("text-[13px] text-muted-foreground", quietClassName)}>Kontak WO belum tersedia</p> : null;
  }
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
