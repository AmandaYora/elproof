import { Badge } from "@/shared/components/ui/Badge";
import type { TenantSubscriptionStatus } from "@/modules/platform-admin/data/types";
import { TENANT_FROZEN_LABEL, TENANT_STATUS_LABEL, TENANT_STATUS_TONE } from "@/modules/platform-admin/lib/status";

/**
 * A tenant's status. A tenant frozen by the platform team reads "Dibekukan" whatever its subscription dates say (EP-09);
 * the subscription it returns to when unfrozen is kept and named in the tooltip.
 */
export function TenantStatusBadge({ status, frozen = false }: { status: TenantSubscriptionStatus; frozen?: boolean }) {
  if (frozen) {
    return (
      <span title={`Langganan tersimpan: ${TENANT_STATUS_LABEL[status]}`}>
        <Badge tone="danger">{TENANT_FROZEN_LABEL}</Badge>
      </span>
    );
  }
  return <Badge tone={TENANT_STATUS_TONE[status]}>{TENANT_STATUS_LABEL[status]}</Badge>;
}
