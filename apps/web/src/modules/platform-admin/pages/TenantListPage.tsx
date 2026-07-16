import { useEffect, useState } from "react";
import { Plus, Pencil, KeyRound, Zap, UserX, UserCheck } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Badge } from "@/shared/components/ui/Badge";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { TenantStatusBadge } from "@/modules/platform-admin/components/TenantStatusBadge";
import { TenantFormModal } from "@/modules/platform-admin/components/TenantFormModal";
import { TenantResetPasswordModal } from "@/modules/platform-admin/components/TenantResetPasswordModal";
import { ActivateSubscriptionModal } from "@/modules/platform-admin/components/ActivateSubscriptionModal";
import { daysUntilExpiry } from "@/modules/platform-admin/data";
import type { Tenant, TenantSubscriptionStatus } from "@/modules/platform-admin/data";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import type { TenantCreateFormValues, TenantFormValues, ResetTenantPasswordFormValues } from "@/modules/platform-admin/schemas/tenant.schema";
import type { ActivateSubscriptionFormValues } from "@/modules/platform-admin/schemas/activate-subscription.schema";
import { TENANT_STATUS_LABEL } from "@/modules/platform-admin/lib/status";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";

const STATUS_FILTERS: TenantSubscriptionStatus[] = ["active", "expiring_soon", "expired", "pending_payment"];

type CredentialReveal =
  | { kind: "create"; tenant: Tenant; username: string; password: string }
  | { kind: "reset"; tenant: Tenant; username: string; password: string };

export default function TenantListPage() {
  const tenants = usePlatformAdminStore((s) => s.tenantPage);
  const meta = usePlatformAdminStore((s) => s.tenantPageMeta);
  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchTenantPage = usePlatformAdminStore((s) => s.fetchTenantPage);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const registerTenant = usePlatformAdminStore((s) => s.registerTenant);
  const updateTenant = usePlatformAdminStore((s) => s.updateTenant);
  const toggleTenantSuspension = usePlatformAdminStore((s) => s.toggleTenantSuspension);
  const resetTenantCredential = usePlatformAdminStore((s) => s.resetTenantCredential);
  const activateTenantSubscription = usePlatformAdminStore((s) => s.activateTenantSubscription);

  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("Semua");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | undefined>(undefined);
  const [resetPasswordTenant, setResetPasswordTenant] = useState<Tenant | undefined>(undefined);
  const [activatingTenant, setActivatingTenant] = useState<Tenant | undefined>(undefined);
  const [credentialReveal, setCredentialReveal] = useState<CredentialReveal | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchPlans();
  }, [fetchPlans]);

  useEffect(() => {
    setPage(1);
  }, [query, statusFilter]);

  useEffect(() => {
    void fetchTenantPage(page, query, statusFilter === "Semua" ? "" : statusFilter);
  }, [fetchTenantPage, page, query, statusFilter]);

  function refetch() {
    return fetchTenantPage(page, query, statusFilter === "Semua" ? "" : statusFilter);
  }

  function planName(planId: string | null) {
    return plans.find((p) => p.id === planId)?.name ?? "-";
  }

  function openCreateModal() {
    setEditingTenant(undefined);
    setModalOpen(true);
  }

  function openEditModal(tenant: Tenant) {
    setEditingTenant(tenant);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingTenant(undefined);
  }

  async function handleRegister(values: TenantCreateFormValues) {
    setActionError(null);
    try {
      const result = await registerTenant(values);
      closeModal();
      await refetch();
      setCredentialReveal({ kind: "create", tenant: result.tenant, username: result.username, password: result.password });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mendaftarkan tenant"));
    }
  }

  async function handleEditSubmit(values: TenantFormValues) {
    if (!editingTenant) return;
    setActionError(null);
    try {
      await updateTenant(editingTenant.id, values);
      closeModal();
      await refetch();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui tenant"));
    }
  }

  async function handleResetPasswordSubmit(values: ResetTenantPasswordFormValues) {
    if (!resetPasswordTenant) return;
    setActionError(null);
    try {
      const result = await resetTenantCredential(resetPasswordTenant.id, values.password);
      setCredentialReveal({ kind: "reset", tenant: resetPasswordTenant, username: result.username, password: result.password });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mereset password tenant"));
    }
    setResetPasswordTenant(undefined);
  }

  async function handleActivateSubmit(values: ActivateSubscriptionFormValues) {
    if (!activatingTenant) return;
    setActionError(null);
    try {
      await activateTenantSubscription(activatingTenant.id, values.planId);
      await refetch();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengaktifkan langganan tenant"));
    }
    setActivatingTenant(undefined);
  }

  async function handleToggleSuspension(tenantId: string) {
    setActionError(null);
    try {
      await toggleTenantSuspension(tenantId);
      await refetch();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status tenant"));
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Tenant</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Seluruh WO yang terdaftar dan berlangganan platform ElProof.</p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
          Tambah Tenant
        </Button>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama WO, pemilik, atau kota..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Select className="w-56" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="Semua">Semua Status</option>
          {STATUS_FILTERS.map((status) => (
            <option key={status} value={status}>
              {TENANT_STATUS_LABEL[status]}
            </option>
          ))}
        </Select>
      </div>

      <Card>
        {tenants.length === 0 ? (
          <EmptyState title="Tidak ada tenant ditemukan" description="Ubah kata kunci pencarian atau filter status." />
        ) : (
          <>
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
                      <TD className="font-semibold text-text-primary">{tenant.businessName}</TD>
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
                          {tenant.isSuspended && <Badge tone="danger">Disuspend</Badge>}
                        </div>
                      </TD>
                      <TD>
                        {tenant.subscriptionExpiresAt ? (
                          <>
                            {formatDate(tenant.subscriptionExpiresAt)}
                            {d !== null && (
                              <span className={`ml-1.5 text-[12px] font-semibold ${d < 0 ? "text-danger" : "text-text-secondary"}`}>
                                ({d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`})
                              </span>
                            )}
                          </>
                        ) : (
                          "-"
                        )}
                      </TD>
                      <TD>
                        <div className="flex items-center gap-1.5">
                          <IconActionButton icon={Pencil} label="Ubah Tenant" tone="neutral" onClick={() => openEditModal(tenant)} />
                          <IconActionButton
                            icon={KeyRound}
                            label="Reset Password"
                            tone="info"
                            onClick={() => setResetPasswordTenant(tenant)}
                          />
                          <IconActionButton
                            icon={Zap}
                            label="Aktifkan Langganan"
                            tone="success"
                            onClick={() => setActivatingTenant(tenant)}
                          />
                          {tenant.isSuspended ? (
                            <IconActionButton
                              icon={UserCheck}
                              label="Aktifkan Kembali"
                              tone="success"
                              onClick={() => void handleToggleSuspension(tenant.id)}
                            />
                          ) : (
                            <IconActionButton
                              icon={UserX}
                              label="Suspend Tenant"
                              tone="danger"
                              onClick={() => void handleToggleSuspension(tenant.id)}
                            />
                          )}
                        </div>
                      </TD>
                    </TR>
                  );
                })}
              </TBody>
            </Table>
            <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </>
        )}
      </Card>

      <TenantFormModal
        key={editingTenant?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        initialTenant={editingTenant}
        plans={plans}
        onSubmitCreate={handleRegister}
        onSubmitEdit={handleEditSubmit}
      />

      <TenantResetPasswordModal
        key={resetPasswordTenant?.id ?? "reset-none"}
        open={resetPasswordTenant !== undefined}
        onClose={() => setResetPasswordTenant(undefined)}
        tenantName={resetPasswordTenant?.businessName}
        onSubmit={handleResetPasswordSubmit}
      />

      <ActivateSubscriptionModal
        key={activatingTenant?.id ?? "activate-none"}
        open={activatingTenant !== undefined}
        onClose={() => setActivatingTenant(undefined)}
        tenant={activatingTenant}
        plans={plans}
        onSubmit={handleActivateSubmit}
      />

      <Modal
        open={credentialReveal !== null}
        onClose={() => setCredentialReveal(null)}
        title={credentialReveal?.kind === "create" ? "Tenant Berhasil Didaftarkan" : "Password Berhasil Direset"}
        description={
          credentialReveal
            ? credentialReveal.kind === "create"
              ? `${credentialReveal.tenant.businessName} kini dapat masuk ke WO Console dengan kredensial berikut.`
              : `Kredensial login owner ${credentialReveal.tenant.businessName} telah diperbarui.`
            : undefined
        }
        size="sm"
        footer={<Button onClick={() => setCredentialReveal(null)}>Selesai</Button>}
      >
        {credentialReveal && (
          <div className="flex flex-col gap-3 text-[13px]">
            <div className="flex items-center gap-2 text-text-secondary">
              <KeyRound className="h-3.5 w-3.5 shrink-0" />
              Kredensial akun owner:
            </div>
            <div className="flex items-center justify-between rounded-md bg-surface-muted px-3.5 py-2.5">
              <span className="text-text-secondary">Username</span>
              <span className="font-semibold text-text-primary">{credentialReveal.username}</span>
            </div>
            <div className="flex items-center justify-between rounded-md bg-surface-muted px-3.5 py-2.5">
              <span className="text-text-secondary">Password</span>
              <span className="font-mono font-semibold text-text-primary">{credentialReveal.password}</span>
            </div>
            {credentialReveal.kind === "create" ? (
              <p className="text-[12px] text-text-secondary">
                Kredensial ini juga otomatis dikirim ke email owner. Tagihan paket {planName(credentialReveal.tenant.planId)} sebesar{" "}
                {formatCurrency(plans.find((p) => p.id === credentialReveal.tenant.planId)?.price ?? 0)} telah diterbitkan dan menunggu
                pembayaran.
              </p>
            ) : (
              <p className="text-[12px] text-text-secondary">Kredensial baru ini juga otomatis dikirim ke email owner.</p>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
