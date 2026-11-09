import { useEffect, useState } from "react";
import { Plus, Pencil, CheckCircle2, Ban } from "lucide-react";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { PlanFormModal } from "@/modules/platform-admin/components/PlanFormModal";
import { TrialLimitsCard } from "@/modules/platform-admin/components/TrialLimitsCard";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { useAppsStore } from "@/modules/platform-admin/stores/useAppsStore";
import type { PlanFormValues } from "@/modules/platform-admin/schemas/plan.schema";
import { formatCurrency } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";

interface PlanTableProps {
  plans: SubscriptionPlan[];
  showPrivateBadge: boolean;
  onEdit: (plan: SubscriptionPlan) => void;
  onToggleActive: (planId: string) => void;
}

// Reused by both the public-catalog and private-per-App sections (D13/§9.2)
// — the row shape is identical, only the "Privat" badge differs.
function PlanTable({ plans, showPrivateBadge, onEdit, onToggleActive }: PlanTableProps) {
  return (
    <>
      <CardList
        className="sm:hidden"
        items={plans}
        keyFor={(plan) => plan.id}
        renderItem={(plan) => (
          <>
            <div className="flex items-start justify-between gap-3">
              <span className="font-semibold text-text-primary">{plan.name}</span>
              <div className="flex items-center gap-1.5">
                {showPrivateBadge && <Badge tone="info">Privat</Badge>}
                {plan.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <CardListField label="Durasi" value={`${plan.durationMonths} bulan`} />
              <CardListField label="Harga" value={formatCurrency(plan.price)} />
              <CardListField label="Fitur" value={`${plan.features.length} fitur`} />
            </div>
            <div className="flex items-center gap-1.5 pt-1">
              <IconActionButton icon={Pencil} label="Ubah Paket" tone="neutral" onClick={() => onEdit(plan)} />
              {plan.isActive ? (
                <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => onToggleActive(plan.id)} />
              ) : (
                <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => onToggleActive(plan.id)} />
              )}
            </div>
          </>
        )}
      />
      <div className="hidden sm:block">
        <Table>
          <THead>
            <TR>
              <TH>Nama Paket</TH>
              <TH>Durasi</TH>
              <TH>Harga</TH>
              <TH>Fitur</TH>
              <TH>Status</TH>
              <TH>Aksi</TH>
            </TR>
          </THead>
          <TBody>
            {plans.map((plan) => (
              <TR key={plan.id}>
                <TD className="font-semibold text-text-primary">
                  <div className="flex items-center gap-2">
                    {plan.name}
                    {showPrivateBadge && <Badge tone="info">Privat</Badge>}
                  </div>
                </TD>
                <TD>{plan.durationMonths} bulan</TD>
                <TD className="tabular-nums">{formatCurrency(plan.price)}</TD>
                <TD>
                  <span className="text-[12.5px] text-text-secondary" title={plan.features.join("\n")}>
                    {plan.features.length} fitur
                  </span>
                </TD>
                <TD>{plan.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}</TD>
                <TD>
                  <div className="flex items-center gap-1.5">
                    <IconActionButton icon={Pencil} label="Ubah Paket" tone="neutral" onClick={() => onEdit(plan)} />
                    {plan.isActive ? (
                      <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => onToggleActive(plan.id)} />
                    ) : (
                      <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => onToggleActive(plan.id)} />
                    )}
                  </div>
                </TD>
              </TR>
            ))}
          </TBody>
        </Table>
      </div>
    </>
  );
}

export default function PlanListPage() {
  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const createPlan = useSubscriptionPlanStore((s) => s.createPlan);
  const updatePlan = useSubscriptionPlanStore((s) => s.updatePlan);
  const togglePlanActive = useSubscriptionPlanStore((s) => s.togglePlanActive);
  const apps = useAppsStore((s) => s.apps);
  const fetchApps = useAppsStore((s) => s.fetchApps);

  const [modalOpen, setModalOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<SubscriptionPlan | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchPlans();
    void fetchApps();
  }, [fetchPlans, fetchApps]);

  function openCreateModal() {
    setEditingPlan(undefined);
    setModalOpen(true);
  }

  function openEditModal(plan: SubscriptionPlan) {
    setEditingPlan(plan);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingPlan(undefined);
  }

  async function handleSubmit(values: PlanFormValues) {
    setActionError(null);
    try {
      if (editingPlan) {
        await updatePlan(editingPlan.id, values);
      } else {
        await createPlan(values);
      }
      closeModal();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan paket"));
    }
  }

  async function handleToggleActive(planId: string) {
    setActionError(null);
    try {
      await togglePlanActive(planId);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status paket"));
    }
  }

  // Pemisahnya HANYA appIds dari planResponse (D12) — tidak ada kolom
  // is_public di database (ditolak eksplisit di D6).
  const publicPlans = plans.filter((p) => p.appIds.length === 0);
  const privateByApp = new Map<string, SubscriptionPlan[]>();
  for (const plan of plans) {
    for (const appId of plan.appIds) {
      const list = privateByApp.get(appId) ?? [];
      list.push(plan);
      privateByApp.set(appId, list);
    }
  }

  function appNameFor(appId: string) {
    return apps.find((a) => a.appId === appId)?.name ?? appId;
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Paket</h1>
          <p className="mt-1 text-[13px] text-text-secondary">
            Katalog publik ElProof untuk tenant WO, dan paket privat per aplikasi standalone.
          </p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
          Tambah Paket
        </Button>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <TrialLimitsCard />

      {plans.length === 0 ? (
        <Card>
          <EmptyState title="Belum ada paket" description="Tambahkan paket langganan pertama untuk platform ElProof." />
        </Card>
      ) : (
        <>
          <Card>
            <CardHeader title="Katalog Publik ElProof" subtitle="Terlihat oleh tenant WO ElProof — dipakai demo dan kebutuhan internal" />
            {publicPlans.length === 0 ? (
              <p className="px-5 py-6 text-center text-[13px] text-text-secondary">Tidak ada paket publik.</p>
            ) : (
              <PlanTable plans={publicPlans} showPrivateBadge={false} onEdit={openEditModal} onToggleActive={(id) => void handleToggleActive(id)} />
            )}
          </Card>

          {/* Sembunyikan seluruh seksi kalau belum ada paket privat sama
             sekali (10.19) — sebelum migrasi seed jalan, ElProof belum
             punya App standalone; kotak kosong di sini tidak berguna. */}
          {privateByApp.size > 0 && (
            <Card>
              <CardHeader title="Paket Privat per App" subtitle="Hanya terlihat oleh aplikasi standalone yang dipetakan" />
              <div className="flex flex-col divide-y divide-border-light">
                {Array.from(privateByApp.entries()).map(([appId, appPlans]) => (
                  <div key={appId} className="px-5 py-3">
                    <p className="pb-2 text-[13px] font-semibold text-text-primary">{appNameFor(appId)}</p>
                    <div className="-mx-5">
                      <PlanTable
                        plans={appPlans}
                        showPrivateBadge={true}
                        onEdit={openEditModal}
                        onToggleActive={(id) => void handleToggleActive(id)}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </>
      )}

      <PlanFormModal key={editingPlan?.id ?? "new"} open={modalOpen} onClose={closeModal} onSubmit={handleSubmit} initialPlan={editingPlan} />
    </div>
  );
}
