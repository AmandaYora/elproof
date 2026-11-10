import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ScrollText } from "lucide-react";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { formatDateTime } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { AuditEntry } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";

const ACTOR_LABEL: Record<string, string> = { platform_admin: "Tim ElProof", staff: "Staf WO", client: "Klien" };

export function TenantActivityTab({ tenantId }: { tenantId: string }) {
  const canSee = usePlatformCan("super-admin");
  const fetchTenantAudit = useConsoleStore((s) => s.fetchTenantAudit);
  const [entries, setEntries] = useState<AuditEntry[] | null>(null);
  const [failed, setFailed] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchTenantAudit(tenantId)
      .then(setEntries)
      .catch(() => setFailed(true));
  }, [fetchTenantAudit, tenantId]);

  useEffect(() => {
    if (canSee) load();
  }, [canSee, load]);

  if (!canSee) {
    return (
      <Card>
        <EmptyState icon={<ScrollText className="h-8 w-8" />} title="Aktivitas hanya untuk Super Admin" description="Audit log memuat jejak seluruh pengguna, jadi hanya Super Admin yang dapat membukanya." />
      </Card>
    );
  }
  if (failed) return <LoadError onRetry={load} />;
  if (!entries) return <div className="py-12 text-center text-sm text-text-secondary">Memuat aktivitas...</div>;

  return (
    <Card>
      <CardHeader
        title="Aktivitas terbaru"
        subtitle="20 perubahan terakhir di workspace ini."
        action={
          <Link to={`${ROUTE_PATHS.platformAudit}?tenantId=${tenantId}`} className="text-[13px] font-semibold text-primary hover:underline">
            Buka di Audit Log
          </Link>
        }
      />
      {entries.length === 0 ? (
        <EmptyState title="Belum ada aktivitas" description="Perubahan data di workspace ini akan tercatat di sini." />
      ) : (
        <ul className="divide-y divide-border-light">
          {entries.map((e) => (
            <li key={e.id} className="flex flex-wrap items-center justify-between gap-2 px-5 py-3">
              <span className="min-w-0">
                <span className="block text-[13.5px] font-medium text-text-primary">{e.action}</span>
                <span className="block text-[12.5px] text-text-secondary">
                  {ACTOR_LABEL[e.actorType] ?? e.actorType}
                  {e.actorRole ? `, ${e.actorRole}` : ""} · {formatDateTime(e.occurredAt)}
                </span>
              </span>
              <Badge tone={e.succeeded ? "success" : "danger"}>{e.succeeded ? "Berhasil" : `Gagal ${e.statusCode}`}</Badge>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
