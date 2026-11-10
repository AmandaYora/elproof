import { useState } from "react";
import { Check, Copy, MessageCircle } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import {
  accessLinkMessage,
  accessLinkUrl,
  formatLinkExpiry,
  whatsappNumber,
  whatsappUrl,
  type AccessLink,
  type AccessLinkRecipient,
} from "@/shared/lib/access-link";

const PURPOSE_COPY: Record<AccessLink["purpose"], { title: string; body: string }> = {
  staff_invite: {
    title: "Link undangan siap dikirim",
    body: "Penerima membuka link ini lalu masuk dengan akun Google yang emailnya sama dengan yang diundang. Link yang diteruskan ke orang lain tidak bisa dipakai.",
  },
  client_invite: {
    title: "Link undangan klien siap dikirim",
    body: "Klien membuka link ini dan membuat kata sandinya sendiri. Anda tidak perlu mengetahui kata sandinya.",
  },
  client_reset: {
    title: "Link atur ulang kata sandi siap dikirim",
    body: "Klien membuka link ini dan membuat kata sandi baru. Kata sandi lama dan sesi yang masih terbuka langsung berhenti setelahnya.",
  },
};

interface AccessLinkDialogProps {
  open: boolean;
  onClose: () => void;
  link: AccessLink | null;
  recipient: AccessLinkRecipient;
  /** WO name shown in the message. */
  woName: string;
  /** Optional line above the link, e.g. what just happened. */
  notice?: string;
}

/**
 * Shows a freshly issued one-time link with "Salin link" and "Kirim via
 * WhatsApp" (wa.me with a ready message — free, sent from the WO's own
 * WhatsApp). The link is shown only now; issuing a new one later revokes it.
 */
export function AccessLinkDialog({ open, onClose, link, recipient, woName, notice }: AccessLinkDialogProps) {
  const [copied, setCopied] = useState<"link" | "message" | null>(null);
  if (!link) return null;
  const url = accessLinkUrl(link.token);
  const message = accessLinkMessage(link, recipient, woName);
  const copy = PURPOSE_COPY[link.purpose];
  const hasWa = whatsappNumber(recipient.phone) !== null;

  async function copyText(text: string, what: "link" | "message") {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(what);
      window.setTimeout(() => setCopied(null), 2000);
    } catch {
      setCopied(null);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={copy.title}
      description={`Untuk ${recipient.name}. Berlaku sampai ${formatLinkExpiry(link.expiresAt)}, sekali pakai.`}
      size="md"
      footer={
        <Button variant="secondary" onClick={onClose}>
          Selesai
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        {notice && (
          <p role="status" className="rounded-md border border-success/30 bg-success-soft px-3.5 py-2.5 text-[13px] font-medium text-success-text">
            {notice}
          </p>
        )}
        <p className="text-[13px] leading-relaxed text-text-secondary">{copy.body}</p>

        <div className="flex items-center gap-2 rounded-md border border-border bg-surface-muted px-3 py-2">
          <code className="min-w-0 flex-1 truncate text-[12.5px] text-text-primary" title={url}>
            {url}
          </code>
          <Button
            variant="secondary"
            size="sm"
            icon={copied === "link" ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            onClick={() => void copyText(url, "link")}
          >
            {copied === "link" ? "Tersalin" : "Salin link"}
          </Button>
        </div>

        <div className="flex flex-col gap-1.5">
          <span className="text-[13px] font-medium text-text-primary">Pesan siap kirim</span>
          <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-md border border-border bg-white px-3 py-2.5 font-sans text-[13px] leading-relaxed text-text-primary">
            {message}
          </pre>
        </div>

        <div className="flex flex-col gap-2 sm:flex-row">
          <a
            href={whatsappUrl(recipient.phone, message)}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex h-9 flex-1 items-center justify-center gap-2 rounded-md bg-[#1f7a4d] px-4 text-sm font-medium text-white hover:bg-[#19663f]"
          >
            <MessageCircle className="h-4 w-4" />
            {hasWa ? "Kirim via WhatsApp" : "Pilih chat WhatsApp"}
          </a>
          <Button
            variant="secondary"
            className="flex-1 justify-center"
            icon={copied === "message" ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            onClick={() => void copyText(message, "message")}
          >
            {copied === "message" ? "Pesan tersalin" : "Salin pesan"}
          </Button>
        </div>
        {!hasWa && (
          <p className="text-[12px] text-text-secondary">Nomor WhatsApp penerima belum ada, jadi WhatsApp akan meminta Anda memilih chat.</p>
        )}
      </div>
    </Modal>
  );
}
