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
      body: "Nomor WhatsApp, kota, dan logo untuk WO Console dan portal klien.",
      done: s.profileComplete,
      action: { label: "Lengkapi", onClick: openProfile },
    },
    {
      key: "project",
      title: "Buat proyek pertama",
      body: "Tahapan dari jadwal bawaan langsung terpasang. Sesuaikan di menu Jadwal bawaan bila perlu.",
      done: s.usage.activeProjects > 0,
      action: { label: "Buat proyek", to: ROUTE_PATHS.projects },
    },
    {
      key: "client",
      title: "Tambahkan klien ke portal",
      body: "Pasangan bisa memantau persiapan tanpa bertanya berulang kali.",
      done: s.usage.clients > 0,
      action: { label: "Tambah klien", to: ROUTE_PATHS.clients },
    },
    {
      key: "staff",
      title: "Ajak tim Anda",
      body: "Tambahkan admin, wedding planner, atau sales di menu Tim.",
      done: s.usage.staff > 0,
      action: { label: "Tambah anggota tim", to: ROUTE_PATHS.users },
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
    <div className="flex min-w-0 flex-1 items-center gap-2">
      <span className="shrink-0 text-[12px] text-muted-foreground">{label}</span>
      <div aria-hidden className="h-1.5 min-w-8 flex-1 overflow-hidden rounded-full bg-muted">
        <div className={cn("h-full rounded-full", full ? "bg-warning" : "bg-primary")} style={{ width: `${Math.min(100, (used / Math.max(1, limit)) * 100)}%` }} />
      </div>
      <span className={cn("shrink-0 text-[12px] font-semibold tabular-nums", full ? "text-warning-text" : "text-foreground")}>
        {used} / {limit}
      </span>
    </div>
  );
}

const actionClass =
  "inline-flex h-8 shrink-0 items-center justify-center rounded-md px-3 text-[13px] font-semibold outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring pointer-coarse:h-10";

/**
 * The Owner's "Mulai di sini" checklist (PLAN.md §3 step 4) plus the free-trial usage caps, kept compact so the
 * Beranda numbers stay near the top: the five steps on one row, only the next step explained with its action.
 * Hidden once every step is done or the Owner dismisses it ("Sembunyikan panduan").
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
  const next = steps.find((s) => !s.done);

  return (
    <section aria-labelledby={showChecklist ? "onboarding-title" : "trial-limits-title"} className="rounded-lg border border-border bg-card">
      {showChecklist && (
        <div className="px-5 py-4">
          <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
            <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
              <h2 id="onboarding-title" className="text-[16px] font-semibold text-foreground">
                {welcome ? "Workspace Anda siap. Mulai dari sini." : "Mulai di sini"}
              </h2>
              <span className="flex items-center gap-2 text-[13px] text-muted-foreground">
                <span aria-hidden className="h-1.5 w-20 overflow-hidden rounded-full bg-muted">
                  <span className="block h-full rounded-full bg-primary" style={{ width: `${(doneCount / steps.length) * 100}%` }} />
                </span>
                {doneCount} dari {steps.length} langkah selesai
              </span>
            </div>
            <button
              type="button"
              onClick={() => void dismiss()}
              className="rounded-md px-2 py-1 text-[13px] font-medium text-muted-foreground outline-none hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-ring"
            >
              Sembunyikan panduan
            </button>
          </div>

          <ol className="mt-3 grid grid-cols-1 gap-x-4 gap-y-1.5 sm:grid-cols-2 lg:grid-cols-5">
            {steps.map((step, i) => (
              <li key={step.key} className="flex min-w-0 items-center gap-2">
                <span
                  aria-hidden
                  className={cn(
                    "flex size-6 shrink-0 items-center justify-center rounded-full text-[12px] font-semibold",
                    step.done ? "bg-success text-white" : step === next ? "bg-primary text-primary-foreground" : "border border-border text-muted-foreground",
                  )}
                >
                  {step.done ? <Check className="size-3.5" /> : i + 1}
                </span>
                <span className={cn("truncate text-[13px]", step.done ? "text-muted-foreground line-through decoration-muted-foreground/40" : step === next ? "font-semibold text-foreground" : "text-foreground")}>
                  {step.title}
                  <span className="sr-only">{step.done ? " (selesai)" : step === next ? " (berikutnya)" : ""}</span>
                </span>
              </li>
            ))}
          </ol>

          {next && (
            <div className="mt-3 flex flex-col gap-3 rounded-md bg-muted px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
              <p className="min-w-0 text-[13px] text-muted-foreground">
                <span className="font-semibold text-foreground">Berikutnya: {next.title}.</span> {next.body}
              </p>
              {next.action.to ? (
                <Link to={next.action.to} className={cn(actionClass, "bg-primary text-primary-foreground hover:bg-primary-hover")}>
                  {next.action.label}
                </Link>
              ) : (
                <button type="button" onClick={next.action.onClick} className={cn(actionClass, "bg-primary text-primary-foreground hover:bg-primary-hover")}>
                  {next.action.label}
                </button>
              )}
            </div>
          )}
        </div>
      )}

      {limits && (
        <div className={cn("flex flex-col gap-2 px-5 py-3 lg:flex-row lg:items-center lg:gap-6", showChecklist && "border-t border-border-light")}>
          <h3 id="trial-limits-title" className="shrink-0 text-[13px] font-semibold text-foreground">
            Batas masa coba
          </h3>
          <div className="flex flex-1 flex-col gap-2 sm:flex-row sm:gap-6">
            <UsageMeter label="Anggota tim selain Owner" used={status.usage.staff} limit={limits.maxStaff} />
            <UsageMeter label="Proyek aktif" used={status.usage.activeProjects} limit={limits.maxActiveProjects} />
            <UsageMeter label="Akun klien" used={status.usage.clients} limit={limits.maxClients} />
          </div>
          <Link to={ROUTE_PATHS.subscription} className="shrink-0 text-[13px] font-semibold text-primary underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring">
            Pilih paket untuk tanpa batas
          </Link>
        </div>
      )}

      <CompleteProfileModal open={profileOpen} onClose={() => setProfileOpen(false)} onSaved={() => void fetch()} />
    </section>
  );
}
