import { Badge } from "@/shared/components/ui/Badge";
import { OWNER_ROLE_NAME } from "@/modules/users/types";

/** The member's workspace role (names are set by each WO). */
export function UserRoleBadge({ role }: { role: string }) {
  return <Badge tone={role === OWNER_ROLE_NAME ? "brand" : "neutral"}>{role}</Badge>;
}
