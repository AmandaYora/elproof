import { Badge } from "@/shared/components/ui/Badge";
import type { BadgeTone } from "@/shared/components/ui/Badge";
import type { StaffRole } from "@/modules/users/types";

const ROLE_TONE: Record<StaffRole, BadgeTone> = {
  Owner: "navy",
  Admin: "info",
  Staff: "neutral",
};

export function UserRoleBadge({ role }: { role: StaffRole }) {
  return <Badge tone={ROLE_TONE[role]}>{role}</Badge>;
}
