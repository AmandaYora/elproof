import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Plus, Pencil, Send, Zap, UserX, UserCheck, Download, ArrowUpRight } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
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
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { API } from "@/shared/services/api-endpoints";

const SOURCE_LABEL = { self_service: "Daftar mandiri", platform: "Dibuat tim ElProof" } as const;

export default function TenantListPage() {
  const tenants = usePlatformAdminStore((s) => s.tenantPage);
  const meta = usePlatformAdminStore((s) => s.tenantPageMeta);
  const fetchTenantPage = usePlatformAdminStore((s) => s.fetchTenantPage);
  const registerTenant = usePlatformAdminStore((s) => s.registerTenant);
  const uploadTenantLogo = usePlatformAdminStore((s) => s.uploadTenantLogo);
  const [params] = useSearchParams();

  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query);
  const [statusFilter, setStatusFilter] = useState<string>(params.get("status") ?? "Semua");
  const [planFilter, setPlanFilter] = useState("Semua");
  const [sourceFilter, setSourceFilter] = useState("Semua");
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const [exporting, setExporting] = useState(false);

  const canManage = usePlatformCan("tenant-manage");
  const canActivate = usePlatformCan("activate-subscription");
  const canFreeze = usePlatformCan("super-admin");
  const canExport = usePlatformCan("internal-read");
  // The detail page reads a workspace's internals, which Read-only may not.
  const canOpen = canExport;

  const status = statusFilter === "Semua" ? "" : statusFilter;
  const extra = { planId: planFilter === "Semua" ? "" : planFilter, source: sourceFilter === "Semua" ? "" : sourceFilter };

  const refetch = useCallback(async () => {
    setFailed(false);
    try {
      await fetchTenantPage(page, debouncedQuery, status, { planId: extra.planId, source: extra.source });
      setLoaded(true);
    } catch {
      setFailed(true);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchTenantPage, page, debouncedQuery, status, extra.planId, extra.source]);

  const actions = useTenantActions(refetch);

  useEffect(() => {
    setPage(1);
  }, [debouncedQuery, statusFilter, planFilter, sourceFilter]);

  useEffect(() => {
    void refetch();
  }, [refetch]);

  function planName(planId: string | null) {
    return actions.plans.find((p) => p.id === planId)?.name ?? "-";
  }

  async function handleRegister(values: TenantCreateFormValues) {
    actions.clearActionError();
    try {
      const result = await registerTenant(values);
      setCreateOpen(false);
      await refetch();
      actions.showOwnerLink({
        tenant: result.tenant,
        link: result.invitation,
        email: result.email,
        notice: `${result.tenant.businessName} didaftarkan dengan masa coba 14 hari.`,
      });
    } catch (err) {
      actions.setActionError(getApiErrorMessage(err, "Gagal mendaftarkan tenant"));
    }
  }

  async function handleExport() {
    setExporting(true);
    try {
      await downloadFile(API.console.tenantsExport, { search: debouncedQuery || undefined, status: status || undefined, planId: extra.planId || undefined, source: extra.source || undefined }, "tenant.csv");
      toast.success("Daftar tenant diunduh. Pengunduhan ini tercatat di audit log.");
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Daftar tenant belum bisa diunduh"));
    } finally {
      setExporting(false);
    }
  }

  function renderName(tenant: Tenant, className: string) {
    return canOpen ? (
      <Link to={ROUTE_PATHS.platformTenantDetail(tenant.id)} className={`${className} hover:underline`}>
        {tenant.businessName}
      </Link>
    ) : (
      <span className={className}>{tenant.businessName}</span>
    );
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

  if (failed && !loaded) return <LoadError onRetry={() => void refetch()} />;

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

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama WO, pemilik, atau email..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Select className="w-56" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="Semua">Semua Status</option>
          {TENANT_STATUS_FILTERS.map((status) => (
            <option key={status} value={status}>
              {TENANT_STATUS_LABEL[status]}
            </option>
          ))}
        </Select>
        <Select className="w-56" value={planFilter} onChange={(e) => setPlanFilter(e.target.value)}>
          <option value="Semua">Semua Paket</option>
          {actions.plans.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </Select>
        <Select className="w-56" value={sourceFilter} onChange={(e) => setSourceFilter(e.target.value)}>
          <option value="Semua">Semua Sumber Daftar</option>
          <option value="self_service">{SOURCE_LABEL.self_service}</option>
          <option value="platform">{SOURCE_LABEL.platform}</option>
        </Select>
      </div>

      <Card>
        {tenants.length === 0 ? (
          <EmptyState title="Tidak ada tenant ditemukan" description="Ubah kata kunci pencarian atau filter." />
        ) : (
          <>
            <CardList
              className="sm:hidden"
              items={tenants}
              keyFor={(tenant) => tenant.id}
              renderItem={(tenant) => {
                const d = daysUntilExpiry(tenant);
                return (
                  <>
                    <div className="flex items-start justify-between gap-3">
                      <span className="flex flex-col gap-1">
                        {renderName(tenant, "font-semibold text-text-primary")}
                        {tenant.signupSource === "self_service" && <span className="text-[12px] font-normal text-text-secondary">Daftar mandiri</span>}
                      </span>
                      <div className="flex shrink-0 flex-col items-end gap-1">
                        <TenantStatusBadge status={tenant.subscriptionStatus} />
                        {tenant.isSuspended && <Badge tone="danger">Dibekukan</Badge>}
                      </div>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <CardListField label="Owner" value={tenant.ownerName} />
                      <CardListField label="Username" value={`@${tenant.username}`} />
                      <CardListField label="Telepon" value={tenant.phone} />
                      <CardListField label="Email" value={tenant.email} />
                      <CardListField label="Kota" value={tenant.city} />
                      <CardListField label="Paket" value={planName(tenant.planId)} />
                      <CardListField
                        label="Berakhir"
                        value={
                          periodEndsAt(tenant) ? (
                            <>
                              {formatDate(periodEndsAt(tenant))}
                              {tenant.lifecycle?.isTrial && <span className="ml-1 text-text-secondary">(masa coba)</span>}
                              {d !== null && (
                                <span className={`ml-1 ${d < 0 ? "text-danger-text" : "text-text-secondary"}`}>
                                  ({d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`})
                                </span>
                              )}
                            </>
                          ) : (
                            "-"
                          )
                        }
                      />
                    </div>
                    <div className="pt-1">
                      {renderRowActions(tenant)}
                    </div>
                  </>
                );
              }}
            />
            <div className="hidden sm:block">
            <Table>
              <THead>
                <TR>
                  <TH>Nama WO</TH>
                  <TH>Owner</TH>
                  <TH>Kontak</TH>
                  <TH>Kota</TH>
                  <TH>Paket</TH>
                  <TH>Status Langganan</TH>
                  <TH>Berakhir</TH>
                  <TH>Aksi</TH>
                </TR>
              </THead>
              <TBody>
                {tenants.map((tenant) => {
                  const d = daysUntilExpiry(tenant);
                  return (
                    <TR key={tenant.id}>
                      <TD>
                        {renderName(tenant, "block font-semibold text-text-primary")}
                        {tenant.signupSource === "self_service" && <span className="block text-[12px] text-text-secondary">Daftar mandiri</span>}
                      </TD>
                      <TD>
                        <span className="block">{tenant.ownerName}</span>
                        <span className="block text-[12.5px] text-text-secondary">@{tenant.username}</span>
                      </TD>
                      <TD>
                        <span className="block">{tenant.phone}</span>
                        <span className="block text-[12.5px] text-text-secondary">{tenant.email}</span>
                      </TD>
                      <TD>{tenant.city}</TD>
                      <TD>{planName(tenant.planId)}</TD>
                      <TD>
                        <div className="flex flex-col items-start gap-1">
                          <TenantStatusBadge status={tenant.subscriptionStatus} />
                          {tenant.isSuspended && <Badge tone="danger">Dibekukan</Badge>}
                        </div>
                      </TD>
                      <TD>
                        {periodEndsAt(tenant) ? (
                          <>
                            {formatDate(periodEndsAt(tenant))}
                            {tenant.lifecycle?.isTrial && <span className="ml-1 text-[12px] text-text-secondary">(masa coba)</span>}
                            {d !== null && (
                              <span className={`ml-1.5 text-[12px] font-semibold ${d < 0 ? "text-danger-text" : "text-text-secondary"}`}>
                                ({d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`})
                              </span>
                            )}
                          </>
                        ) : (
                          "-"
                        )}
                      </TD>
                      <TD>
                        {renderRowActions(tenant)}
                      </TD>
                    </TR>
                  );
                })}
              </TBody>
            </Table>
            </div>
            <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </>
        )}
      </Card>

      <TenantFormModal
        key="create"
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSubmitCreate={handleRegister}
        onSubmitEdit={() => undefined}
        onUploadLogo={uploadTenantLogo}
      />

      {actions.dialogs}
    </div>
  );
}
