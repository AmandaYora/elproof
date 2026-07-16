import { Badge } from "@/shared/components/ui/Badge";
import type { TenantSubscriptionStatus } from "@/modules/platform-admin/data/types";
import { TENANT_STATUS_LABEL, TENANT_STATUS_TONE } from "@/modules/platform-admin/lib/status";

export function TenantStatusBadge({ status }: { status: TenantSubscriptionStatus }) {
  return <Badge tone={TENANT_STATUS_TONE[status]}>{TENANT_STATUS_LABEL[status]}</Badge>;
}
