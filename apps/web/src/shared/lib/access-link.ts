// One-time access links (PLAN.md Fase 3): staff/Owner invitations accepted
// with Google, and client invitations / password resets where the client
// sets their own password. The WO sends them by WhatsApp or copies them;
// ElProof sends nothing itself (no e-mail/WhatsApp API yet).

export type AccessLinkPurpose = "staff_invite" | "client_invite" | "client_reset";

export interface AccessLink {
  token: string;
  purpose: AccessLinkPurpose;
  /** RFC 3339 */
  expiresAt: string;
}

export interface AccessLinkRecipient {
  name: string;
  /** WhatsApp number in any common spelling; omitted = choose the chat in WhatsApp. */
  phone?: string;
  /** Google account that must accept a staff invitation. */
  email?: string;
  /** Client login name. */
  username?: string;
  /** e.g. "Admin", "Wedding Planner", "Owner". */
  roleLabel?: string;
}

export function accessLinkUrl(token: string): string {
  return `${window.location.origin}/undangan/${encodeURIComponent(token)}`;
}

export function formatLinkExpiry(expiresAt: string): string {
  return new Intl.DateTimeFormat("id-ID", { day: "numeric", month: "long", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(
    new Date(expiresAt),
  );
}

/** wa.me wants digits with the country code: 0812… -> 62812…. */
export function whatsappNumber(phone: string | undefined): string | null {
  if (!phone) return null;
  let d = phone.replace(/\D/g, "");
  if (d.startsWith("0")) d = "62" + d.slice(1);
  else if (d.startsWith("8")) d = "62" + d;
  return /^62[1-9]\d{7,12}$/.test(d) ? d : null;
}

export function whatsappUrl(phone: string | undefined, text: string): string {
  const n = whatsappNumber(phone);
  return `https://wa.me/${n ?? ""}?text=${encodeURIComponent(text)}`;
}

/** The ready-to-send message for each kind of link. */
export function accessLinkMessage(link: AccessLink, to: AccessLinkRecipient, woName: string): string {
  const url = accessLinkUrl(link.token);
  const until = formatLinkExpiry(link.expiresAt);
  const greeting = `Halo ${to.name},`;
  switch (link.purpose) {
    case "staff_invite":
      return [
        greeting,
        `Anda diundang bergabung di ${woName} di ElProof${to.roleLabel ? ` sebagai ${to.roleLabel}` : ""}.`,
        `Buka link ini lalu masuk dengan akun Google${to.email ? ` ${to.email}` : ""}:`,
        url,
        `Link berlaku sampai ${until} dan hanya bisa dipakai sekali.`,
      ].join("\n");
    case "client_invite":
      return [
        greeting,
        `${woName} mengundang Anda ke portal ElProof untuk memantau persiapan pernikahan.`,
        to.username ? `Nama pengguna Anda: ${to.username}` : "",
        "Buka link ini untuk membuat kata sandi Anda sendiri:",
        url,
        `Link berlaku sampai ${until} dan hanya bisa dipakai sekali.`,
      ]
        .filter(Boolean)
        .join("\n");
    case "client_reset":
      return [
        greeting,
        `Ini link untuk mengatur ulang kata sandi portal ElProof Anda bersama ${woName}.`,
        to.username ? `Nama pengguna Anda: ${to.username}` : "",
        url,
        `Link berlaku sampai ${until} dan hanya bisa dipakai sekali.`,
      ]
        .filter(Boolean)
        .join("\n");
  }
}
