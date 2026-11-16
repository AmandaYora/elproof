export const CONTACT = {
  email: "cs@elcodelabs.com",
  phones: [
    { display: "0851-7347-1146", number: "6285173471146" }, // digits only, for tel: and wa.me links
    { display: "0851-5825-1146", number: "6285158251146" },
  ],
  website: "elproof.elcodelabs.com",
  address: "Jl. Ciptakarya II No.33, Bandung Kota, 40292, Jawa Barat",
  hours: "Setiap hari, 08.00–21.00 WIB",
};

/**
 * Who provides ElProof: the company named in the footer and, as provider and data controller, at the top of every legal
 * document (QA FQ-12). The correspondence address and contact are CONTACT's.
 */
export const COMPANY = {
  name: "PT Prayora Karya Pratama",
  brand: "Elcodelabs",
  // TODO(FQ-12): no dedicated data-protection address exists yet; the general customer-service email stands in. Replace
  // it here when the company names a data-protection officer or mailbox (e.g. privasi@…), and every legal page follows.
  privacyEmail: CONTACT.email,
};

/** A wa.me link to the first ElProof number with a ready message (the visitor still presses send). */
export function whatsappUrl(text = "Halo ElProof, saya ingin bertanya tentang ElProof untuk WO saya."): string {
  return `https://wa.me/${CONTACT.phones[0].number}?text=${encodeURIComponent(text)}`;
}
