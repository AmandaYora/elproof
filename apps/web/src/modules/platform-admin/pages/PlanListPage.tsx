import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Plus } from "lucide-react";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { PlanFormModal } from "@/modules/platform-admin/components/PlanFormModal";
import { TrialLimitsCard } from "@/modules/platform-admin/components/TrialLimitsCard";
import { Field, Select } from "@/shared/components/ui/Input";
import { appLabel, ELPROOF_APP_ID, type SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { useAppsStore } from "@/modules/platform-admin/stores/useAppsStore";
import { PlanTable } from "@/modules/platform-admin/components/PlanTable";
import type { PlanFormValues } from "@/modules/platform-admin/schemas/plan.schema";
import { getApiErrorMessage } from "@/shared/lib/api-error";

/** The URL keeps the App filter (?aplikasi=), so the view survives a reload and the Back button. */
const APP_PARAM = "aplikasi";

export default function PlanListPage() {
  const plans = useSubscriptionPlanStore((s) => s.plans);
  const isLoading = useSubscriptionPlanStore((s) => s.isLoading);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const createPlan = useSubscriptionPlanStore((s) => s.createPlan);
  const updatePlan = useSubscriptionPlanStore((s) => s.updatePlan);
  const togglePlanActive = useSubscriptionPlanStore((s) => s.togglePlanActive);
  const apps = useAppsStore((s) => s.apps);
  const fetchApps = useAppsStore((s) => s.fetchApps);

  const [searchParams, setSearchParams] = useSearchParams();
  const appFilter = searchParams.get(APP_PARAM) ?? "";

  const [modalOpen, setModalOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<SubscriptionPlan | undefined>(undefined);
  const [newPlanAppId, setNewPlanAppId] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchApps();
  }, [fetchApps]);

  // The server narrows the list to one App: the browser never loads every plan just to hide most of them.
  useEffect(() => {
    void fetchPlans(appFilter);
  }, [fetchPlans, appFilter]);

  function setAppFilter(appId: string) {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (appId) next.set(APP_PARAM, appId);
        else next.delete(APP_PARAM);
        return next;
      },
      { replace: true }
    );
  }

  function openCreateModal(appId: string) {
    setEditingPlan(undefined);
    setNewPlanAppId(appId);
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

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleSubmit(values: PlanFormValues) {
    setActionError(null);
    if (editingPlan) {
      await updatePlan(editingPlan.id, values);
    } else {
      await createPlan(values);
    }
    closeModal();
  }

  async function handleToggleActive(planId: string) {
    setActionError(null);
    try {
      await togglePlanActive(planId);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status paket"));
    }
  }

  // Sections in the order of Manajemen Aplikasi, ElProof's own catalog first. With a filter there is one section,
  // even when it is empty; without one, an App with no plans yet has no section.
  const appOrder = [ELPROOF_APP_ID, ...apps.map((a) => a.appId).filter((id) => id !== ELPROOF_APP_ID)];
  for (const plan of plans) {
    if (!appOrder.includes(plan.appId)) appOrder.push(plan.appId);
  }
  const sections = (appFilter ? [appFilter] : appOrder)
    .map((appId) => ({ appId, plans: plans.filter((p) => p.appId === appId) }))
    .filter((section) => appFilter !== "" || section.plans.length > 0);

  function nameOf(appId: string) {
    return appLabel(apps.find((a) => a.appId === appId), appId);
  }

  const filterOptions = [ELPROOF_APP_ID, ...apps.map((a) => a.appId).filter((id) => id !== ELPROOF_APP_ID)];
  if (appFilter && !filterOptions.includes(appFilter)) filterOptions.push(appFilter);

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Paket</h1>
          <p className="mt-1 max-w-2xl text-[13px] text-text-secondary">
            Setiap paket milik satu aplikasi dan hanya dijual oleh aplikasi itu. Paket ElProof tampil untuk tenant WO;
            paket aplikasi lain hanya terlihat oleh aplikasi tersebut.
          </p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={() => openCreateModal(appFilter)}>
          Tambah Paket
        </Button>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <TrialLimitsCard />

      <div className="w-full sm:max-w-xs">
        <Field label="Aplikasi" name="aplikasi">
          <Select value={appFilter} onChange={(e) => setAppFilter(e.target.value)} aria-label="Saring paket per aplikasi">
            <option value="">Semua aplikasi</option>
            {filterOptions.map((appId) => (
              <option key={appId} value={appId}>
                {nameOf(appId)}
              </option>
            ))}
          </Select>
        </Field>
      </div>

      {sections.length === 0 ? (
        <Card>
          <EmptyState
            title={isLoading ? "Memuat paket..." : "Belum ada paket"}
            description={isLoading ? undefined : "Tambahkan paket pertama dan pilih aplikasi yang menjualnya."}
          />
        </Card>
      ) : (
        sections.map((section) => (
          <Card key={section.appId} data-app={section.appId}>
            <CardHeader
              title={nameOf(section.appId)}
              subtitle={
                section.appId === ELPROOF_APP_ID
                  ? "Tampil di halaman Langganan tenant WO dan beranda ElProof"
                  : "Hanya terlihat oleh aplikasi ini lewat API"
              }
              action={
                <Button variant="secondary" size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => openCreateModal(section.appId)}>
                  Tambah paket
                </Button>
              }
            />
            {section.plans.length === 0 ? (
              <p className="px-5 py-6 text-center text-[13px] text-text-secondary">
                {isLoading ? "Memuat paket..." : "Belum ada paket untuk aplikasi ini."}
              </p>
            ) : (
              <PlanTable plans={section.plans} onEdit={openEditModal} onToggleActive={(id) => void handleToggleActive(id)} />
            )}
          </Card>
        ))
      )}

      <PlanFormModal
        key={editingPlan?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmit={handleSubmit}
        initialPlan={editingPlan}
        apps={apps}
        defaultAppId={newPlanAppId}
      />
    </div>
  );
}
