// Mirrors apps/api's domain.AppliedToContract / OverpaymentExcess: only DP,
// Termin and Pelunasan use up the contract value (Refund gives room back);
// Tambahan is money for work outside the contract and never counts.
// Keeping the rule in one place makes the form's hint, the summary figures
// and the badge agree with what the API will accept.

export interface PaymentLike {
  type: string;
  amount: number;
}

const COUNTS_AGAINST_CONTRACT = new Set(["DP", "Termin", "Pelunasan"]);

export function countsAgainstContract(type: string): boolean {
  return COUNTS_AGAINST_CONTRACT.has(type);
}

/** How much of the contract value has been paid: DP + Termin + Pelunasan − Refund. */
export function appliedToContract(payments: PaymentLike[]): number {
  return payments.reduce((sum, p) => {
    if (countsAgainstContract(p.type)) return sum + p.amount;
    if (p.type === "Refund") return sum - p.amount;
    return sum;
  }, 0);
}

/** What is still owed. Negative means the contract has been overpaid. */
export function remainingToPay(contractValue: number, payments: PaymentLike[]): number {
  return contractValue - appliedToContract(payments);
}

/**
 * Message for the Nominal field when saving `next` would take what is paid
 * past what is owed, or null when it is fine. `others` are the other payments
 * of the same owed amount; `original` is the saved version of the payment being
 * edited (omit when adding). Same rule as the API: refused only when the change
 * both exceeds the contract and raises the paid total, so editing a reference
 * number on an already-overpaid project, or lowering an amount, stays allowed.
 * A contract of 0 (not set yet) is never limited.
 */
export function overpaymentMessage(
  contractValue: number,
  others: PaymentLike[],
  next: PaymentLike,
  formatMoney: (n: number) => string,
  owed = "sisa tagihan",
  original?: PaymentLike,
): string | null {
  if (contractValue <= 0) return null;
  const was = appliedToContract(original ? [...others, original] : others);
  const now = appliedToContract([...others, next]);
  if (now <= was || now <= contractValue) return null;
  const remaining = remainingToPay(contractValue, others);
  return `Melebihi ${owed} (${formatMoney(Math.max(remaining, 0))}). Kurangi nominalnya, atau pilih jenis Tambahan untuk biaya di luar kontrak.`;
}
