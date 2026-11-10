import { useCallback, useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { CardListField } from "@/shared/components/ui/CardList";
import { cn } from "@/shared/lib/cn";
import { formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useAppsStore, type PaymentApp, type AppSubscription } from "@/modules/platform-admin/stores/useAppsStore";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { PlanFormModal } from "@/modules/platform-admin/components/PlanFormModal";
import { PlanTable } from "@/modules/platform-admin/components/PlanTable";
import type { PlanFormValues } from "@/modules/platform-admin/schemas/plan.schema";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";

type DetailTab = "info" | "paket" | "langganan";

const TABS: { key: DetailTab; label: string }[] = [
  { key: "info", label: "Info" },
  { key: "paket", label: "Paket" },
  { key: "langganan", label: "Langganan" },
];

interface AppDetailModalProps {
  open: boolean;
  onClose: () => void;
  app?: PaymentApp;
}

// Bertab dengan state lokal, BUKAN TabNav (T21/A28) — TabNav berbasis
// NavLink/route dan akan mengubah URL lalu menutup modalnya sendiri.
// Kelas aktif/nonaktif tab bar di bawah sengaja menyamai TabNav.tsx supaya
// bahasa visualnya tetap satu.
export function AppDetailModal({ open, onClose, app }: AppDetailModalProps) {
  const fetchPlansForApp = useSubscriptionPlanStore((s) => s.fetchPlansForApp);
  const createPlan = useSubscriptionPlanStore((s) => s.createPlan);
  const updatePlan = useSubscriptionPlanStore((s) => s.updatePlan);
  const togglePlanActive = useSubscriptionPlanStore((s) => s.togglePlanActive);
  const apps = useAppsStore((s) => s.apps);
  const fetchAppSubscriptions = useAppsStore((s) => s.fetchAppSubscriptions);

  const [tab, setTab] = useState<DetailTab>("info");
  const [appPlans, setAppPlans] = useState<SubscriptionPlan[]>([]);
  const [isLoadingPlans, setIsLoadingPlans] = useState(false);
  const [plansError, setPlansError] = useState<string | null>(null);
  const [planFormOpen, setPlanFormOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<SubscriptionPlan | undefined>(undefined);

  const [subscription, setSubscription] = useState<AppSubscription | undefined>(undefined);
  const [isLoadingSubscription, setIsLoadingSubscription] = useState(false);

  const appId = app?.appId;

  // The plans this App owns (ADR-0040), narrowed by the server.
  const loadPlans = useCallback(
    (id: string) => {
      setIsLoadingPlans(true);
      setPlansError(null);
      return fetchPlansForApp(id)
        .then(setAppPlans)
        .catch((err) => setPlansError(getApiErrorMessage(err, "Gagal memuat paket aplikasi ini")))
        .finally(() => setIsLoadingPlans(false));
    },
    [fetchPlansForApp]
  );

  useEffect(() => {
    if (!open || !appId) return;
    setTab("info");
    setAppPlans([]);
    void loadPlans(appId);

    setIsLoadingSubscription(true);
    fetchAppSubscriptions()
      .then((subs) => setSubscription(subs.find((s) => s.appId === appId)))
      .finally(() => setIsLoadingSubscription(false));
  }, [open, appId, loadPlans, fetchAppSubscriptions]);

  if (!app) return null;

  function openPlanForm(plan?: SubscriptionPlan) {
    setEditingPlan(plan);
    setPlanFormOpen(true);
  }

  function closePlanForm() {
    setPlanFormOpen(false);
    setEditingPlan(undefined);
  }

  // Errors propagate to the form (422 on its fields). A plan moved to another App leaves this list on reload.
  async function handlePlanSubmit(values: PlanFormValues) {
    if (editingPlan) {
      await updatePlan(editingPlan.id, values);
    } else {
      await createPlan(values);
    }
    closePlanForm();
    await loadPlans(appId as string);
  }

  async function handleToggleActive(planId: string) {
    setPlansError(null);
    try {
      await togglePlanActive(planId);
      await loadPlans(appId as string);
    } catch (err) {
      setPlansError(getApiErrorMessage(err, "Gagal mengubah status paket"));
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={app.name}
      description={app.appId}
      size="lg"
      footer={
        <Button variant="secondary" onClick={onClose}>
          Tutup
        </Button>
      }
    >
      <div className="flex flex-col gap-4">
        <div className="flex gap-6 border-b border-border">
          {TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              onClick={() => setTab(t.key)}
              className={cn(
                "shrink-0 border-b-2 py-2.5 text-[13.5px] font-medium transition-colors",
                tab === t.key
                  ? "border-primary font-semibold text-primary"
                  : "border-transparent text-text-secondary hover:text-text-primary"
              )}
            >
              {t.label}
            </button>
          ))}
        </div>

        {tab === "info" && (
          <div className="flex flex-col gap-2.5">
            <CardListField label="App ID" value={<span className="font-mono">{app.appId}</span>} />
            <CardListField
              label="Jenis"
              value={<Badge tone={app.kind === "internal" ? "brand" : "info"}>{app.kind === "internal" ? "Sistem ElProof" : "Eksternal"}</Badge>}
            />
            {/* Sengaja TIDAK memakai CardListField untuk Callback URL — nilainya
               ditruncate paksa di sana, padahal bebas dari truncate itulah
               justru alasan field ini dipindahkan ke tab Info (§9.1). */}
            <div className="flex items-start justify-between gap-3 text-[12.5px]">
              <span className="shrink-0 text-text-secondary">Callback URL</span>
              <span className="min-w-0 break-all text-right font-medium text-text-primary">{app.callbackUrl || "-"}</span>
            </div>
            <CardListField label="Status" value={<Badge tone={app.isActive ? "success" : "neutral"}>{app.isActive ? "Aktif" : "Nonaktif"}</Badge>} />
            <CardListField label="Terdaftar" value={formatDate(app.createdAt)} />
          </div>
        )}

        {tab === "paket" && (
          <div className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-[13px] text-text-secondary">
                {app.kind === "internal"
                  ? "Paket milik ElProof: tampil di halaman Langganan tenant WO."
                  : "Paket milik aplikasi ini: hanya aplikasi ini yang melihat dan menjualnya lewat API."}
              </p>
              <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => openPlanForm()}>
                Tambah paket
              </Button>
            </div>
            {plansError && (
              <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{plansError}</p>
            )}
            {isLoadingPlans && appPlans.length === 0 ? (
              <p className="py-8 text-center text-sm text-text-secondary">Memuat...</p>
            ) : appPlans.length === 0 ? (
              <p className="py-8 text-center text-sm text-text-secondary">Aplikasi ini belum punya paket.</p>
            ) : (
              <div className="-mx-6 border-y border-border-light">
                <PlanTable plans={appPlans} onEdit={(plan) => openPlanForm(plan)} onToggleActive={(id) => void handleToggleActive(id)} />
              </div>
            )}
            {/* Inside the detail dialog so the two stack: Escape closes the plan form first. */}
            <PlanFormModal
              key={editingPlan?.id ?? "new"}
              open={planFormOpen}
              onClose={closePlanForm}
              onSubmit={handlePlanSubmit}
              initialPlan={editingPlan}
              apps={apps}
              defaultAppId={app.appId}
              lockApp={!editingPlan}
            />
          </div>
        )}

        {tab === "langganan" && (
          <div className="flex flex-col gap-2.5">
            {isLoadingSubscription ? (
              <p className="py-8 text-center text-sm text-text-secondary">Memuat...</p>
            ) : subscription ? (
              <>
                <CardListField label="Paket" value={subscription.planName} />
                <CardListField label="Berlaku sampai" value={formatDate(subscription.subscriptionUntil)} />
                <CardListField label="Status" value={<Badge tone="success">{subscription.status}</Badge>} />
              </>
            ) : (
              <div className="flex flex-col items-center gap-1 py-8 text-center">
                <p className="text-sm font-semibold text-text-primary">—</p>
                <p className="text-[12.5px] text-text-secondary">Belum ada pembayaran lewat endpoint langganan.</p>
              </div>
            )}
          </div>
        )}
      </div>
    </Modal>
  );
}
