import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import { pickContactPhone } from "@/modules/client-portal/lib/contact-phone";

/** Who the couple talks to, and how a message to them starts. Every WhatsApp button in the portal reads this. */
export interface PortalContact {
  /** The PIC's number when usable, else the business's; undefined when neither is (the buttons then hide). */
  phone: string | undefined;
  /** "Rina" (the PIC) or the WO's name: who the greeting and the button labels address. */
  name: string;
  /** Whether `name` is the PIC (a person) rather than the business. */
  isPic: boolean;
  businessName: string;
  clientName: string;
  projectName: string;
}

export function usePortalContact(): PortalContact {
  const home = useClientHomeStore((s) => s.home);
  const clientName = useAuthStore((s) => s.session?.displayName ?? "");
  const businessName = useTenantBrandingStore((s) => s.businessName) ?? "";
  const picName = home?.pic.name || home?.project.picName || "";
  return {
    phone: pickContactPhone(home?.pic.phone || home?.project.picPhone, home?.contact?.whatsapp),
    name: picName || businessName || "wedding organizer Anda",
    isPic: Boolean(picName),
    businessName,
    clientName,
    projectName: home?.project.name ?? "",
  };
}

/** "Halo Rina, saya Citra (Citra & Bima). Saya ingin menanyakan Termin 2 Rp40.000.000." */
export function askMessage(c: PortalContact, subject?: string): string {
  const who = c.clientName ? `saya ${c.clientName}${c.projectName ? ` (${c.projectName})` : ""}` : `kami dari ${c.projectName || "portal klien"}`;
  const ask = subject ? `Saya ingin menanyakan ${subject}.` : "Saya ingin bertanya tentang persiapan pernikahan kami.";
  return `Halo ${c.name}, ${who}. ${ask}`;
}
