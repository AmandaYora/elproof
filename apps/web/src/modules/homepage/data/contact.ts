export const CONTACT = {
  email: "cs@elcodelabs.com",
  phones: [
    { display: "0851-7347-1146", number: "6285173471146" }, // digits only, for tel: and wa.me links
    { display: "0851-5825-1146", number: "6285158251146" },
  ],
  website: "elproof.elcodelabs.com",
  address: "Jl. Ciptakarya II No.33, Bandung Kota, 40292, Jawa Barat",
  hours: "Setiap Hari, 08.00–21.00 WIB",
};

/** A wa.me link to the first ElProof number with a ready message (the visitor still presses send). */
export function whatsappUrl(text = "Halo ElProof, saya ingin bertanya tentang ElProof untuk WO saya."): string {
  return `https://wa.me/${CONTACT.phones[0].number}?text=${encodeURIComponent(text)}`;
}
