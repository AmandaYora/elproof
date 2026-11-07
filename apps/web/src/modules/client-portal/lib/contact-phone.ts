import { whatsappNumber } from "@/shared/lib/access-link";

/**
 * The number a WhatsApp button should use: the PIC's when it is a usable
 * number, otherwise the business's. (A staff phone is free text; "-" or a
 * half-typed number must not hide a perfectly good business number.)
 */
export function pickContactPhone(picPhone: string | undefined, businessPhone: string | undefined): string | undefined {
  if (whatsappNumber(picPhone)) return picPhone;
  if (whatsappNumber(businessPhone)) return businessPhone;
  return undefined;
}
