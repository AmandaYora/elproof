import { ArrowRightLeft, Ban, Check, Clock, X, type LucideIcon } from "lucide-react";
import { Badge, type BadgeTone } from "@/shared/components/ui/Badge";
import { QUOTATION_STATUS_LABELS } from "@/shared/lib/labels";
import type { QuotationDisplayStatus } from "@/modules/offers/types";

// Each status has its own symbol next to its word (DESAIN §6.1): never colour alone.
const LOOK: Record<QuotationDisplayStatus, { tone: BadgeTone; icon: LucideIcon }> = {
  Draft: { tone: "info", icon: Clock },
  Expired: { tone: "warning", icon: Ban },
  Accepted: { tone: "success", icon: Check },
  Rejected: { tone: "danger", icon: X },
  Superseded: { tone: "neutral", icon: ArrowRightLeft },
};

export function QuotationStatusBadge({ status, className }: { status: QuotationDisplayStatus; className?: string }) {
  const { tone, icon: Icon } = LOOK[status];
  return (
    <Badge tone={tone} dot={false} className={className}>
      <Icon aria-hidden />
      {QUOTATION_STATUS_LABELS[status]}
    </Badge>
  );
}
