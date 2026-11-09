import { useCallback, useEffect, useState } from "react";
import { Link, Navigate, useParams } from "react-router-dom";
import { ArrowLeft, Pencil, Send, UserCheck, UserX } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { TabNav } from "@/shared/components/ui/TabNav";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { TenantStatusBadge } from "@/modules/platform-admin/components/TenantStatusBadge";
import { TenantProfileTab } from "@/modules/platform-admin/components/tenant-detail/TenantProfileTab";
import { TenantSubscriptionTab } from "@/modules/platform-admin/components/tenant-detail/TenantSubscriptionTab";
import { TenantBillingTab } from "@/modules/platform-admin/components/tenant-detail/TenantBillingTab";
import { TenantUsageTab } from "@/modules/platform-admin/components/tenant-detail/TenantUsageTab";
import { TenantMembersTab } from "@/modules/platform-admin/components/tenant-detail/TenantMembersTab";
import { TenantActivityTab } from "@/modules/platform-admin/components/tenant-detail/TenantActivityTab";
import { TenantNotesTab } from "@/modules/platform-admin/components/tenant-detail/TenantNotesTab";
import { TenantDataRequestsTab } from "@/modules/platform-admin/components/tenant-detail/TenantDataRequestsTab";
import { TenantSupportTab } from "@/modules/platform-admin/components/tenant-detail/TenantSupportTab";
import type { Tenant, SubscriptionTransaction } from "@/modules/platform-admin/data/types";
import type { TenantOverview } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { toTenant, type RawTenant } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { useTenantActions } from "@/modules/platform-admin/hooks/useTenantActions";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";

const TABS = [
  { key: "profil", label: "Profil" },
  { key: "langganan", label: "Langganan" },
  { key: "tagihan", label: "Tagihan" },
  { key: "pemakaian", label: "Pemakaian" },
  { key: "pengguna", label: "Pengguna" },
  { key: "aktivitas", label: "Aktivitas" },
  { key: "catatan", label: "Catatan" },
  { key: "data", label: "Permintaan data" },
  { key: "dukungan", label: "Dukungan" },
] as const;

type TabKey = (typeof TABS)[number]["key"];

export default function TenantDetailPage() {
  const { id = "", tab } = useParams();
  const fetchTenantOverview = useConsoleStore((s) => s.fetchTenantOverview);
  const fetchTenantTransactions = useConsoleStore((s) => s.fetchTenantTransactions);
  const [overview, setOverview] = useState<TenantOverview | null>(null);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [transactions, setTransactions] = useState<SubscriptionTransaction[]>([]);
  const [failed, setFailed] = useState(false);

  const canManage = usePlatformCan("tenant-manage");
  const canActivate = usePlatformCan("activate-subscription");
  const canFreeze = usePlatformCan("super-admin");

  const load = useCallback(async () => {
    setFailed(false);
    try {
      const [ov, tx] = await Promise.all([fetchTenantOverview(id), fetchTenantTransactions(id)]);
      setOverview(ov);
      setTenant(toTenant(ov.tenant as RawTenant));
      setTransactions(tx);
    } catch {
      setFailed(true);
    }
  }, [fetchTenantOverview, fetchTenantTransactions, id]);

  const actions = useTenantActions(load);

  useEffect(() => {
    setOverview(null);
    setTenant(null);
    void load();
  }, [load]);

  const activeTab = TABS.find((t) => t.key === tab)?.key as TabKey | undefined;
  if (tab !== undefined && !activeTab) return <Navigate to={ROUTE_PATHS.platformTenantDetail(id)} replace />;
  const current: TabKey = activeTab ?? "profil";

  if (failed) {
    return (
      <div className="flex flex-col gap-4">
        <Link to={ROUTE_PATHS.platformTenants} className="inline-flex items-center gap-1.5 text-[13px] font-medium text-text-secondary hover:text-text-primary">
          <ArrowLeft className="h-4 w-4" /> Semua tenant
        </Link>
        <LoadError message="Tenant tidak ditemukan atau belum bisa dimuat." onRetry={() => void load()} />
      </div>
    );
  }
  if (!overview || !tenant) return <div className="py-20 text-center text-sm text-text-secondary">Memuat tenant...</div>;

  const tabItems = TABS.map((t) => ({
    to: ROUTE_PATHS.platformTenantDetail(id, t.key),
    label: t.key === "data" && overview.openDataRequests > 0 ? `${t.label} (${overview.openDataRequests})` : t.label,
    end: true,
  }));

  return (
    <div className="flex flex-col gap-5">
      <Link to={ROUTE_PATHS.platformTenants} className="inline-flex items-center gap-1.5 text-[13px] font-medium text-text-secondary hover:text-text-primary">
        <ArrowLeft className="h-4 w-4" /> Semua tenant
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2.5">
            <h1 className="text-xl font-bold text-text-primary">{tenant.businessName}</h1>
            <TenantStatusBadge status={tenant.subscriptionStatus} />
            {tenant.isSuspended && <Badge tone="danger">Dibekukan</Badge>}
          </div>
          <p className="mt-1 text-[13px] text-text-secondary">
            {tenant.ownerName} · {tenant.email} · {tenant.city}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {canManage && (
            <Button variant="secondary" size="sm" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => actions.openEdit(tenant)}>
              Ubah
            </Button>
          )}
          {canManage && (
            <Button variant="secondary" size="sm" icon={<Send className="h-3.5 w-3.5" />} onClick={() => void actions.reinvite(tenant)}>
              Undang ulang Owner
            </Button>
          )}
          {canFreeze && (
            <Button variant={tenant.isSuspended ? "secondary" : "danger"} size="sm" icon={tenant.isSuspended ? <UserCheck className="h-3.5 w-3.5" /> : <UserX className="h-3.5 w-3.5" />} onClick={() => actions.openSuspension(tenant)}>
              {tenant.isSuspended ? "Aktifkan kembali" : "Bekukan"}
            </Button>
          )}
        </div>
      </div>

      {actions.actionError && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actions.actionError}</p>
      )}

      <TabNav items={tabItems} />

      {current === "profil" && <TenantProfileTab tenant={tenant} consent={overview.consent} />}
      {current === "langganan" && (
        <TenantSubscriptionTab tenant={tenant} planName={overview.planName} activations={overview.activations} onActivate={canActivate ? () => actions.openActivation(tenant) : undefined} />
      )}
      {current === "tagihan" && <TenantBillingTab transactions={transactions} />}
      {current === "pemakaian" && <TenantUsageTab isTrial={tenant.lifecycle.isTrial} staff={overview.staff} projects={overview.activeProjects} clients={overview.clients} />}
      {current === "pengguna" && <TenantMembersTab members={overview.members} />}
      {current === "aktivitas" && <TenantActivityTab tenantId={id} />}
      {current === "catatan" && <TenantNotesTab tenantId={id} />}
      {current === "data" && <TenantDataRequestsTab tenantId={id} ownerEmail={tenant.email} onChanged={() => void load()} />}
      {current === "dukungan" && <TenantSupportTab tenantId={id} tenantName={tenant.businessName} />}

      {actions.dialogs}
    </div>
  );
}
