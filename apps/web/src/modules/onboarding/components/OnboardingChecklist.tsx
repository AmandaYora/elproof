import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Check } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { cn } from "@/shared/lib/cn";
import { useOnboardingStore, type OnboardingStatus } from "@/modules/onboarding/stores/useOnboardingStore";
import { CompleteProfileModal } from "@/modules/onboarding/components/CompleteProfileModal";

interface Step {
  key: string;
  title: string;
  body: string;
  done: boolean;
  action: { label: string; to?: string; onClick?: () => void };
}

function buildSteps(s: OnboardingStatus, openProfile: () => void): Step[] {
  return [
    {
      key: "profile",
      title: "Lengkapi profil WO",
      body: "Nomor WhatsApp, kota, dan logo untuk WO Console dan portal client.",
      done: s.profileComplete,
      action: { label: "Lengkapi", onClick: openProfile },
    },
    {
      key: "project",
      title: "Buat project pertama",
      body: "Tahapan dari Timeline Default langsung terpasang. Sesuaikan di menu Timeline Default bila perlu.",
      done: s.usage.activeProjects > 0,
      action: { label: "Buat project", to: ROUTE_PATHS.projects },
    },
    {
      key: "client",
      title: "Tambahkan client ke portal",
      body: "Pasangan bisa memantau persiapan tanpa bertanya berulang kali.",
      done: s.usage.clients > 0,
      action: { label: "Tambah client", to: ROUTE_PATHS.clients },
    },
    {
      key: "staff",
      title: "Ajak tim Anda",
      body: "Tambahkan admin, wedding planner, atau sales di menu Pengguna.",
      done: s.usage.staff > 0,
      action: { label: "Tambah staf", to: ROUTE_PATHS.users },
    },
    {
      key: "plan",
      title: "Pilih paket langganan",
      body: "Aktifkan sebelum masa coba berakhir agar tim tetap bisa bekerja tanpa batas.",
      done: s.hasPaidPlan,
      action: { label: "Lihat paket", to: ROUTE_PATHS.subscription },
    },
  ];
}

function UsageMeter({ label, used, limit }: { label: string; used: number; limit: number }) {
  const full = used >= limit;
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1.5">
      <div className="flex items-baseline justify-between gap-2 text-[12.5px]">
        <span className="text-text-secondary">{label}</span>
        <span className={cn("font-semibold tabular-nums", full ? "text-warning-text" : "text-text-primary")}>
          {used} / {limit}
        </span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-surface-muted">
        <div
          className={cn("h-full rounded-full", full ? "bg-warning" : "bg-navy-900")}
          style={{ width: `${Math.min(100, (used / Math.max(1, limit)) * 100)}%` }}
        />
      </div>
    </div>
  );
}

/**
 * The Owner's "Mulai di sini" checklist (PLAN.md §3 step 4) plus the
 * free-trial usage caps. Hidden once every step is done or the Owner
 * dismisses it.
 */
export function OnboardingChecklist({ welcome = false }: { welcome?: boolean }) {
  const status = useOnboardingStore((s) => s.status);
  const fetch = useOnboardingStore((s) => s.fetch);
  const dismiss = useOnboardingStore((s) => s.dismiss);
  const [profileOpen, setProfileOpen] = useState(false);

  useEffect(() => {
    void fetch();
  }, [fetch]);

  if (!status) return null;
  const steps = buildSteps(status, () => setProfileOpen(true));
  const doneCount = steps.filter((s) => s.done).length;
  const showChecklist = !status.dismissed && doneCount < steps.length;
  const limits = status.trialLimits;
  if (!showChecklist && !limits) return null;

  return (
    <section aria-labelledby="onboarding-title" className="rounded-xl border border-border bg-surface shadow-sm">
      {showChecklist && (
        <>
          <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border-light px-5 py-4 sm:px-6">
            <div>
              <h2 id="onboarding-title" className="text-[15px] font-semibold text-text-primary">
                {welcome ? "Workspace Anda siap. Mulai dari sini." : "Mulai di sini"}
              </h2>
              <p className="mt-0.5 text-[13px] text-text-secondary">
                {doneCount} dari {steps.length} langkah selesai
              </p>
            </div>
            <button
              type="button"
              onClick={() => void dismiss()}
              className="rounded-md px-2 py-1 text-[12.5px] font-medium text-text-secondary hover:bg-surface-muted hover:text-text-primary"
            >
              Sembunyikan panduan
            </button>
          </div>
          <ol className="divide-y divide-border-light">
            {steps.map((step, i) => (
              <li key={step.key} className="flex flex-col gap-3 px-5 py-3.5 sm:flex-row sm:items-center sm:gap-4 sm:px-6">
                <span
                  aria-hidden
                  className={cn(
                    "flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-[12.5px] font-semibold",
                    step.done ? "bg-success text-white" : "border border-border text-text-secondary",
                  )}
                >
                  {step.done ? <Check className="h-4 w-4" /> : i + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <p className={cn("text-[14px] font-medium", step.done ? "text-text-secondary line-through decoration-text-secondary/40" : "text-text-primary")}>
                    {step.title}
                    <span className="sr-only">{step.done ? " (selesai)" : ""}</span>
                  </p>
                  {!step.done && <p className="mt-0.5 text-[12.5px] leading-relaxed text-text-secondary">{step.body}</p>}
                </div>
                {!step.done &&
                  (step.action.to ? (
                    <Link
                      to={step.action.to}
                      className="inline-flex h-8 shrink-0 items-center justify-center rounded-md border border-border bg-white px-3 text-[13px] font-medium text-text-primary hover:bg-surface-muted"
                    >
                      {step.action.label}
                    </Link>
                  ) : (
                    <button
                      type="button"
                      onClick={step.action.onClick}
                      className="inline-flex h-8 shrink-0 items-center justify-center rounded-md bg-navy-900 px-3 text-[13px] font-medium text-white hover:bg-navy-800"
                    >
                      {step.action.label}
                    </button>
                  ))}
              </li>
            ))}
          </ol>
        </>
      )}

      {limits && (
        <div className={cn("px-5 py-4 sm:px-6", showChecklist && "border-t border-border-light bg-surface-muted/40")}>
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h3 className="text-[13px] font-semibold text-text-primary">Batas masa coba</h3>
            <Link to={ROUTE_PATHS.subscription} className="text-[12.5px] font-medium text-navy-900 hover:underline">
              Pilih paket untuk tanpa batas
            </Link>
          </div>
          <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:gap-6">
            <UsageMeter label="Staf selain Owner" used={status.usage.staff} limit={limits.maxStaff} />
            <UsageMeter label="Project aktif" used={status.usage.activeProjects} limit={limits.maxActiveProjects} />
            <UsageMeter label="Akun client" used={status.usage.clients} limit={limits.maxClients} />
          </div>
        </div>
      )}

      <CompleteProfileModal open={profileOpen} onClose={() => setProfileOpen(false)} onSaved={() => void fetch()} />
    </section>
  );
}
