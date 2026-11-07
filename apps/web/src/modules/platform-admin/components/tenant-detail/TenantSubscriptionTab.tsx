import { Zap } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { formatDate } from "@/shared/lib/formatters";
import { TenantStatusBadge } from "@/modules/platform-admin/components/TenantStatusBadge";
import type { Tenant } from "@/modules/platform-admin/data/types";
import type { ManualActivation } from "@/modules/platform-admin/data/console.types";
import { DefinitionList } from "@/modules/platform-admin/components/tenant-detail/TenantProfileTab";

interface Props {
  tenant: Tenant;
  planName: string;
  activations: ManualActivation[];
  onActivate?: () => void;
}

export function TenantSubscriptionTab({ tenant, planName, activations, onActivate }: Props) {
  const lc = tenant.lifecycle;
  const accessText = { full: "Penuh", read_only: "Hanya melihat data", billing_only: "Hanya halaman pembayaran", blocked: "Tidak bisa masuk" }[lc.access];
  return (
    <div className="flex flex-col gap-5">
      <Card>
        <CardHeader
          title="Status langganan"
          action={
            onActivate && (
              <Button size="sm" icon={<Zap className="h-3.5 w-3.5" />} onClick={onActivate}>
                Aktifkan langganan
              </Button>
            )
          }
        />
        <CardContent>
          <DefinitionList
            rows={[
              { label: "Status", value: <TenantStatusBadge status={tenant.subscriptionStatus} /> },
              { label: "Paket", value: planName || (lc.isTrial ? "Masa coba" : "-") },
              { label: lc.isTrial ? "Masa coba berakhir" : "Langganan berakhir", value: formatDate(lc.periodEndsAt) },
              { label: "Sisa waktu", value: lc.periodEndsAt ? (lc.daysLeft >= 0 ? `${lc.daysLeft} hari` : `Lewat ${Math.abs(lc.daysLeft)} hari`) : "-" },
              { label: "Menjadi baca-saja pada", value: formatDate(lc.readOnlyAt) },
              { label: "Ditangguhkan pada", value: formatDate(lc.suspendAt) },
              { label: "Akses staf WO saat ini", value: accessText },
              { label: "Dibekukan tim ElProof", value: lc.frozen ? "Ya" : "Tidak" },
            ]}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Aktivasi manual" subtitle="Langganan yang diaktifkan tim tanpa pembayaran lewat gateway. Alasan dan bukti wajib diisi." />
        {activations.length === 0 ? (
          <CardContent>
            <p className="text-[13px] text-text-secondary">Belum pernah ada aktivasi manual untuk tenant ini.</p>
          </CardContent>
        ) : (
          <ul className="divide-y divide-border-light">
            {activations.map((a) => (
              <li key={a.id} className="px-5 py-3.5">
                <p className="text-[13.5px] font-semibold text-text-primary">
                  {a.reasonLabel} <span className="font-normal text-text-secondary">oleh {a.admin}, {formatDate(a.createdAt)}</span>
                </p>
                <p className="mt-0.5 text-[12.5px] text-text-secondary">
                  Bukti: {a.proofReference}. Berlaku sampai {formatDate(a.expiresAt)}.{a.note ? ` ${a.note}` : ""}
                </p>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
