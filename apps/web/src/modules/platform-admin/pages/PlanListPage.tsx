import { useEffect, useState } from "react";
import { Plus, Pencil, CheckCircle2, Ban } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { PlanFormModal } from "@/modules/platform-admin/components/PlanFormModal";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import type { PlanFormValues } from "@/modules/platform-admin/schemas/plan.schema";
import { formatCurrency } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export default function PlanListPage() {
  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const createPlan = useSubscriptionPlanStore((s) => s.createPlan);
  const updatePlan = useSubscriptionPlanStore((s) => s.updatePlan);
  const togglePlanActive = useSubscriptionPlanStore((s) => s.togglePlanActive);

  const [modalOpen, setModalOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<SubscriptionPlan | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchPlans();
  }, [fetchPlans]);

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

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Paket</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola paket langganan yang tersedia untuk tenant.</p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
          Tambah Paket
        </Button>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      <Card>
        {plans.length === 0 ? (
          <EmptyState title="Belum ada paket" description="Tambahkan paket langganan pertama untuk platform ElProof." />
        ) : (
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
                  <TD className="font-semibold text-text-primary">{plan.name}</TD>
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
                      <IconActionButton icon={Pencil} label="Ubah Paket" tone="neutral" onClick={() => openEditModal(plan)} />
                      {plan.isActive ? (
                        <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(plan.id)} />
                      ) : (
                        <IconActionButton
                          icon={CheckCircle2}
                          label="Aktifkan"
                          tone="success"
                          onClick={() => void handleToggleActive(plan.id)}
                        />
                      )}
                    </div>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <PlanFormModal key={editingPlan?.id ?? "new"} open={modalOpen} onClose={closeModal} onSubmit={handleSubmit} initialPlan={editingPlan} />
    </div>
  );
}
