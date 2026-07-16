import { Badge } from "@/shared/components/ui/Badge";
import type { PlatformAdminRole } from "@/modules/platform-admin/data/types";

export function PlatformAdminRoleBadge({ role }: { role: PlatformAdminRole }) {
  return <Badge tone={role === "Super Admin" ? "navy" : "neutral"}>{role}</Badge>;
}
