import { Badge } from "@/shared/components/ui/Badge";

export function VendorStatusBadge({ isActive }: { isActive: boolean }) {
  return isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>;
}
