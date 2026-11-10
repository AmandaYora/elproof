import { Badge } from "@/shared/components/ui/Badge";
import type { PlatformAdminRole } from "@/modules/platform-admin/data/types";

import type { BadgeTone } from "@/shared/components/ui/Badge";

const ROLE_TONE: Record<PlatformAdminRole, BadgeTone> = {
  "Super Admin": "brand",
  Billing: "info",
  Support: "neutral",
  "Read-only": "neutral",
};

export function PlatformAdminRoleBadge({ role }: { role: PlatformAdminRole }) {
  return <Badge tone={ROLE_TONE[role] ?? "neutral"}>{role}</Badge>;
}
