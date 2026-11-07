import { formatCurrency } from "@/shared/lib/formatters";

interface ConfirmationInput {
  businessName: string;
  clientName: string;
  projectName: string;
  termLabel: string;
  amount: number;
  bankName?: string;
  bankAccountNumber?: string;
}

/** The WhatsApp message a client sends to confirm a transfer, already filled in. */
export function paymentConfirmationMessage(i: ConfirmationInput): string {
  const to = i.bankName && i.bankAccountNumber ? ` ke rekening ${i.bankName} ${i.bankAccountNumber}` : "";
  return `Halo ${i.businessName}, saya ${i.clientName} (${i.projectName}). Saya sudah transfer ${i.termLabel} sebesar ${formatCurrency(i.amount)}${to}. Mohon dikonfirmasi. Terima kasih.`;
}

/** A general greeting when the client just wants to reach their WO. */
export function greetingMessage(businessName: string, clientName: string, projectName: string): string {
  return `Halo ${businessName}, saya ${clientName} (${projectName}). Saya ingin bertanya tentang persiapan pernikahan kami.`;
}
