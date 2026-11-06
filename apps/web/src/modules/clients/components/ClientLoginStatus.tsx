import { Badge } from "@/shared/components/ui/Badge";
import type { Client } from "@/modules/clients/types";

/** Status of a client account: invitation pending, active or inactive. */
export function ClientLoginStatusBadge({ client }: { client: Client }) {
  if (!client.isActive) return <Badge tone="neutral">Nonaktif</Badge>;
  if (client.loginStatus === "invited") return <Badge tone="warning">Menunggu undangan</Badge>;
  return <Badge tone="success">Aktif</Badge>;
}

/** Label for the "send a link" action, which depends on the account state. */
export function clientLinkActionLabel(client: Client): string {
  return client.loginStatus === "invited" ? "Kirim ulang undangan" : "Buat link atur ulang sandi";
}
