/** Profil usaha › Penawaran as the API sends it (GET/PUT /tenants/me/quote-settings). */
export interface QuoteSettings {
  documentCode: string;
  isPkp: boolean;
  defaultValidityDays: number;
  defaultTermsText: string;
}

/**
 * What a quotation number made today looks like with this code: MO-PNW-261119-001 (code, "PNW", the date as yymmdd,
 * the day's sequence). The server assigns the real numbers; this only shows the shape.
 */
export function exampleQuoteNumber(code: string, today = new Date()): string {
  const two = (n: number) => String(n).padStart(2, "0");
  const date = `${two(today.getFullYear() % 100)}${two(today.getMonth() + 1)}${two(today.getDate())}`;
  return `${code || "MO"}-PNW-${date}-001`;
}
