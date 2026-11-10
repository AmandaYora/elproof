import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Pencil, Send, Zap, UserX, UserCheck, Download, ArrowUpRight } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Select } from "@/shared/components/ui/Input";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { TenantStatusBadge } from "@/modules/platform-admin/components/TenantStatusBadge";
import { TenantFormModal } from "@/modules/platform-admin/components/TenantFormModal";
import { daysUntilExpiry, periodEndsAt } from "@/modules/platform-admin/data";
import type { Tenant } from "@/modules/platform-admin/data";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { useTenantActions } from "@/modules/platform-admin/hooks/useTenantActions";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import type { TenantCreateFormValues } from "@/modules/platform-admin/schemas/tenant.schema";
import { TENANT_STATUS_FILTERS, TENANT_STATUS_LABEL } from "@/modules/platform-admin/lib/status";
import { formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { downloadFile } from "@/shared/lib/download";
import { toast } from "@/shared/stores/useToastStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { API } from "@/shared/services/api-endpoints";

const SOURCE_LABEL = { self_service: "Daftar mandiri", platform: "Dibuat tim ElProof" } as const;

const LIST_FILTERS = { status: "", planId: "", source: "" };

export default function TenantListPage() {
  const tenants = usePlatformAdminStore((s) => s.tenantPage);
  const meta = usePlatformAdminStore((s) => s.tenantPageMeta);
  const fetchTenantPage = usePlatformAdminStore((s) => s.fetchTenantPage);
  const registerTenant = usePlatformAdminStore((s) => s.registerTenant);
  const uploadTenantLogo = usePlatformAdminStore((s) => s.uploadTenantLogo);

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState({ filters: LIST_FILTERS });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("platform-tenants");
  const [loadState, setLoadState] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [createOpen, setCreateOpen] = useState(false);
  const [exporting, setExporting] = useState(false);

  const canManage = usePlatformCan("tenant-manage");
  const canActivate = usePlatformCan("activate-subscription");
  const canFreeze = usePlatformCan("super-admin");
  const canExport = usePlatformCan("internal-read");
  // The detail page reads a workspace's internals, which Read-only may not.
  const canOpen = canExport;

  useEffect(() => {
    let cancelled = false;
    setLoadState("loading");
    fetchTenantPage(list.params)
      .then(() => !cancelled && setLoadState("ready"))
      .catch(() => !cancelled && setLoadState("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchTenantPage, list.paramsKey, attempt]);

  const reload = useCallback(() => fetchTenantPage(list.params), [fetchTenantPage, list.params]);

  const actions = useTenantActions(reload);

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleRegister(values: TenantCreateFormValues) {
    actions.clearActionError();
    const result = await registerTenant(values);
    setCreateOpen(false);
    await reload();
    actions.showOwnerLink({
      tenant: result.tenant,
      link: result.invitation,
      email: result.email,
      notice: `${result.tenant.businessName} didaftarkan dengan masa coba 14 hari.`,
    });
  }

  async function handleExport() {
    setExporting(true);
    try {
      await downloadFile(
        API.console.tenantsExport,
        { search: list.q || undefined, status: list.filters.status || undefined, planId: list.filters.planId || undefined, source: list.filters.source || undefined },
        "tenant.csv",
      );
      toast.success("Daftar tenant diunduh. Pengunduhan ini tercatat di audit log.");
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Daftar tenant belum bisa diunduh"));
    } finally {
      setExporting(false);
    }
  }

  function planName(planId: string | null) {
    return actions.plans.find((p) => p.id === planId)?.name ?? "-";
  }

  function renderRowActions(tenant: Tenant) {
    return (
      <div className="flex items-center gap-1.5">
        {canOpen && (
          <Link
            to={ROUTE_PATHS.platformTenantDetail(tenant.id)}
            title="Buka detail tenant"
            aria-label="Buka detail tenant"
            className="inline-flex h-8 w-8 items-center justify-center rounded-md text-text-secondary hover:bg-surface-muted hover:text-text-primary pointer-coarse:h-11 pointer-coarse:w-11"
          >
            <ArrowUpRight className="h-4 w-4" />
          </Link>
        )}
        {canManage && <IconActionButton icon={Pencil} label="Ubah Tenant" tone="neutral" onClick={() => actions.openEdit(tenant)} />}
        {canManage && <IconActionButton icon={Send} label="Undang ulang Owner" tone="info" onClick={() => void actions.reinvite(tenant)} />}
        {canActivate && <IconActionButton icon={Zap} label="Aktifkan Langganan" tone="success" onClick={() => actions.openActivation(tenant)} />}
        {canFreeze &&
          (tenant.isSuspended ? (
            <IconActionButton icon={UserCheck} label="Aktifkan Kembali" tone="success" onClick={() => actions.openSuspension(tenant)} />
          ) : (
            <IconActionButton icon={UserX} label="Bekukan Tenant" tone="danger" onClick={() => actions.openSuspension(tenant)} />
          ))}
      </div>
    );
  }

  const columns: Column<Tenant>[] = [
    {
      id: "name",
      header: "Tenant",
      sortKey: "name",
      primary: true,
      cell: (t) => (
        <>
          {canOpen ? (
            <Link to={ROUTE_PATHS.platformTenantDetail(t.id)} className="block font-semibold text-text-primary hover:underline">
              {t.businessName}
            </Link>
          ) : (
            <span className="block font-semibold text-text-primary">{t.businessName}</span>
          )}
          <span className="block text-[12.5px] font-normal text-muted-foreground">
            {t.ownerName} · {t.email}
          </span>
          {t.signupSource === "self_service" && <span className="block text-[12px] font-normal text-muted-foreground">Daftar mandiri</span>}
        </>
      ),
    },
    { id: "plan", header: "Paket", sortKey: "planName", cell: (t) => planName(t.planId) },
    {
      id: "status",
      header: "Status",
      sortKey: "status",
      cell: (t) => (
        <div className="flex flex-col items-start gap-1">
          <TenantStatusBadge status={t.subscriptionStatus} />
          {t.isSuspended && <Badge tone="danger">Dibekukan</Badge>}
        </div>
      ),
    },
    { id: "trial", header: "Masa coba berakhir", sortKey: "trialEndsAt", cell: (t) => (t.trialEndsAt ? formatDate(t.trialEndsAt) : <span className="text-muted-foreground">-</span>) },
    {
      id: "ends",
      header: "Langganan berakhir",
      cell: (t) => {
        const end = periodEndsAt(t);
        if (!end) return <span className="text-muted-foreground">-</span>;
        const d = daysUntilExpiry(t);
        return (
          <>
            {formatDate(end)}
            {t.lifecycle?.isTrial && <span className="ml-1 text-[12px] text-muted-foreground">(masa coba)</span>}
            {d !== null && (
              <span className={`ml-1.5 text-[12px] font-semibold ${d < 0 ? "text-danger-text" : "text-muted-foreground"}`}>
                ({d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`})
              </span>
            )}
          </>
        );
      },
    },
    { id: "joined", header: "Bergabung", sortKey: "createdAt", cell: (t) => formatDate(t.joinedAt) },
  ];

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Tenant</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Seluruh WO yang terdaftar dan berlangganan platform ElProof.</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {canExport && (
            <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={() => void handleExport()} disabled={exporting}>
              {exporting ? "Menyiapkan..." : "Unduh CSV"}
            </Button>
          )}
          {canManage && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => setCreateOpen(true)}>
              Tambah Tenant
            </Button>
          )}
        </div>
      </div>

      {actions.actionError && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actions.actionError}</p>
      )}

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama WO, pemilik, atau email..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Select className="w-52" aria-label="Status" value={list.filters.status || "Semua"} onChange={(e) => list.setFilter("status", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua status</option>
          {TENANT_STATUS_FILTERS.map((status) => (
            <option key={status} value={status}>
              {TENANT_STATUS_LABEL[status]}
            </option>
          ))}
        </Select>
        <Select className="w-52" aria-label="Paket" value={list.filters.planId || "Semua"} onChange={(e) => list.setFilter("planId", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua paket</option>
          {actions.plans.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </Select>
        <Select className="w-56" aria-label="Sumber daftar" value={list.filters.source || "Semua"} onChange={(e) => list.setFilter("source", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua sumber daftar</option>
          <option value="self_service">{SOURCE_LABEL.self_service}</option>
          <option value="platform">{SOURCE_LABEL.platform}</option>
        </Select>
      </ListToolbar>

      <DataTable
        label="Daftar tenant"
        columns={columns}
        rows={tenants}
        rowId={(t) => t.id}
        status={loadState}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada tenant",
          description: "Tenant muncul di sini setelah WO mendaftar sendiri atau tim ElProof menambahkannya.",
          action: canManage ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => setCreateOpen(true)}>
              Tambah tenant
            </Button>
          ) : undefined,
        }}
        density={density}
        actions={renderRowActions}
      />

      <TenantFormModal
        key="create"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSubmitCreate={handleRegister}
        onSubmitEdit={async () => undefined}
        onUploadLogo={uploadTenantLogo}
      />

      {actions.dialogs}
    </div>
  );
}
