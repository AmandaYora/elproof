import { useEffect, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { CardListField } from "@/shared/components/ui/CardList";
import { cn } from "@/shared/lib/cn";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useAppsStore, type PaymentApp, type AppSubscription } from "@/modules/platform-admin/stores/useAppsStore";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";

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
  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const apps = useAppsStore((s) => s.apps);
  const fetchAppPlans = useAppsStore((s) => s.fetchAppPlans);
  const setAppPlans = useAppsStore((s) => s.setAppPlans);
  const fetchAppSubscriptions = useAppsStore((s) => s.fetchAppSubscriptions);

  const [tab, setTab] = useState<DetailTab>("info");
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [isLoadingPlans, setIsLoadingPlans] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const [subscription, setSubscription] = useState<AppSubscription | undefined>(undefined);
  const [isLoadingSubscription, setIsLoadingSubscription] = useState(false);

  const appId = app?.appId;

  useEffect(() => {
    if (!open || !appId) return;
    setTab("info");
    setSaveError(null);
    void fetchPlans();

    setIsLoadingPlans(true);
    fetchAppPlans(appId)
      .then((ids) => setSelectedIds(new Set(ids)))
      .finally(() => setIsLoadingPlans(false));

    setIsLoadingSubscription(true);
    fetchAppSubscriptions()
      .then((subs) => setSubscription(subs.find((s) => s.appId === appId)))
      .finally(() => setIsLoadingSubscription(false));
  }, [open, appId, fetchPlans, fetchAppPlans, fetchAppSubscriptions]);

  if (!app) return null;

  function toggle(id: number) {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  async function handleSave() {
    setIsSaving(true);
    setSaveError(null);
    try {
      // Replace-set semantics (A8): kirim SELURUH id yang tercentang, bukan
      // delta — mengirim delta akan menghapus pemetaan yang tidak ikut terkirim.
      await setAppPlans(appId as string, Array.from(selectedIds));
      onClose();
    } catch (err) {
      setSaveError(getApiErrorMessage(err, "Gagal menyimpan pemetaan paket"));
    } finally {
      setIsSaving(false);
    }
  }

  function appNameFor(id: string) {
    return apps.find((a) => a.appId === id)?.name ?? id;
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={app.name}
      description={app.appId}
      size="lg"
      footer={
        tab === "paket" ? (
          <>
            <Button variant="secondary" onClick={onClose} disabled={isSaving}>
              Batal
            </Button>
            <Button onClick={() => void handleSave()} disabled={isSaving || isLoadingPlans}>
              {isSaving ? "Menyimpan..." : "Simpan Pemetaan"}
            </Button>
          </>
        ) : (
          <Button variant="secondary" onClick={onClose}>
            Tutup
          </Button>
        )
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
                  ? "border-navy-900 font-semibold text-navy-900"
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
              value={<Badge tone={app.kind === "internal" ? "navy" : "info"}>{app.kind === "internal" ? "Sistem ElProof" : "Eksternal"}</Badge>}
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
          <div className="flex flex-col gap-2.5">
            {saveError && (
              <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{saveError}</p>
            )}
            {isLoadingPlans ? (
              <p className="py-8 text-center text-sm text-text-secondary">Memuat...</p>
            ) : plans.length === 0 ? (
              <p className="py-8 text-center text-sm text-text-secondary">Belum ada paket. Tambahkan dulu di menu Paket.</p>
            ) : (
              <ul className="flex flex-col gap-2">
                {plans.map((plan) => {
                  const otherApps = plan.appIds.filter((id) => id !== appId);
                  return (
                    <li key={plan.id}>
                      <label className="flex items-start gap-2.5 rounded-md border border-border px-3 py-2.5">
                        <input
                          type="checkbox"
                          className="mt-0.5"
                          checked={selectedIds.has(Number(plan.id))}
                          onChange={() => toggle(Number(plan.id))}
                        />
                        <span className="flex-1">
                          <span className="block text-[13.5px] font-medium text-text-primary">
                            {plan.name} — {formatCurrency(plan.price)} / {plan.durationMonths} bulan
                          </span>
                          {otherApps.length > 0 && (
                            <span className="block text-[12px] text-text-secondary">Juga dipetakan ke: {otherApps.map(appNameFor).join(", ")}</span>
                          )}
                        </span>
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}
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
