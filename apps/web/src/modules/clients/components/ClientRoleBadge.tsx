import { Badge, type BadgeTone } from "@/shared/components/ui/Badge";
import type { ClientRole } from "@/modules/clients/types";

const ROLE_LABEL: Record<ClientRole, string> = {
  Bride: "Pengantin Wanita",
  Groom: "Pengantin Pria",
  "Family Representative": "Wedding Representative",
};

const ROLE_TONE: Record<ClientRole, BadgeTone> = {
  Bride: "info",
  Groom: "navy",
  "Family Representative": "neutral",
};

export function ClientRoleBadge({ role }: { role: ClientRole }) {
  return <Badge tone={ROLE_TONE[role]}>{ROLE_LABEL[role]}</Badge>;
}
