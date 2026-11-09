import { Link } from "react-router-dom";
import { AlertTriangle, Clock, Lock } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { SubscriptionLifecycle } from "@/shared/stores/useSubscriptionStatusStore";

type Tone = "info" | "warning" | "danger";

interface BannerCopy {
  tone: Tone;
  message: string;
}

/**
 * What the banner says for each lifecycle state. Returns null when there is
 * nothing to tell (paid and not close to the end).
 */
export function bannerCopy(lc: SubscriptionLifecycle): BannerCopy | null {
  const end = lc.periodEndsAt ? formatDate(lc.periodEndsAt) : null;
  const days = Math.max(lc.daysLeft, 0);
  switch (lc.status) {
    case "trialing":
      return {
        tone: days <= 3 ? "warning" : "info",
        message:
          days === 0
            ? `Masa coba ElProof berakhir hari ini.`
            : `Masa coba ElProof tersisa ${days} hari, sampai ${end}.`,
      };
    case "expiring_soon":
      return {
        tone: "warning",
        message:
          days === 0
            ? `Langganan berakhir hari ini. Perpanjang agar pekerjaan tim tidak terputus.`
            : `Langganan berakhir dalam ${days} hari, pada ${end}. Perpanjang agar pekerjaan tim tidak terputus.`,
      };
    case "past_due":
      return {
        tone: "warning",
        message: `${lc.isTrial ? "Masa coba" : "Langganan"} sudah berakhir pada ${end}. Semua fitur masih bisa dipakai sampai ${
          lc.readOnlyAt ? formatDate(lc.readOnlyAt) : "masa tenggang selesai"
        }; setelah itu data hanya bisa dilihat.`,
      };
    case "read_only":
    case "pending_payment":
    case "expired":
      return {
        tone: "danger",
        message: `Mode baca-saja: data bisa dilihat dan diekspor, tetapi tidak bisa diubah sampai langganan diperpanjang.${
          lc.suspendAt ? ` Akses ditangguhkan mulai ${formatDate(lc.suspendAt)}.` : ""
        }`,
      };
    default:
      return null;
  }
}

const TONE_CLASSES: Record<Tone, string> = {
  info: "border-info/25 bg-info-soft text-info-text",
  warning: "border-warning/30 bg-warning-soft text-warning-text",
  danger: "border-danger/30 bg-danger-soft text-danger-text",
};

export function SubscriptionBanner({ lifecycle, canManage }: { lifecycle: SubscriptionLifecycle; canManage: boolean }) {
  const copy = bannerCopy(lifecycle);
  if (!copy) return null;
  const Icon = copy.tone === "info" ? Clock : AlertTriangle;
  return (
    <div role="status" className={cn("mb-5 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border px-4 py-3", TONE_CLASSES[copy.tone])}>
      <Icon className="h-4 w-4 shrink-0" />
      <p className="min-w-0 flex-1 text-[13px] font-medium leading-relaxed">{copy.message}</p>
      {canManage ? (
        <Link
          to={ROUTE_PATHS.subscription}
          className="shrink-0 rounded-md bg-white/70 px-3 py-1.5 text-[13px] font-semibold text-text-primary hover:bg-white"
        >
          {lifecycle.isTrial && lifecycle.status === "trialing" ? "Pilih paket" : "Perpanjang langganan"}
        </Link>
      ) : (
        <span className="shrink-0 text-[12.5px] font-medium opacity-80">Hubungi Owner WO untuk memperpanjang.</span>
      )}
    </div>
  );
}

/**
 * Shown instead of the page once the subscription is suspended for
 * non-payment: the API only serves the subscription and export routes then.
 */
export function SubscriptionLockedScreen({ canManage }: { canManage: boolean }) {
  return (
    <div className="mx-auto mt-10 flex max-w-lg flex-col items-center rounded-xl border border-border bg-surface px-6 py-10 text-center shadow-sm">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-danger-soft text-danger-text">
        <Lock className="h-5 w-5" />
      </div>
      <h1 className="text-lg font-bold text-text-primary">Langganan ditangguhkan</h1>
      <p className="mt-2 text-[13.5px] leading-relaxed text-text-secondary">
        Langganan WO ini belum diperpanjang, jadi WO Console dikunci sementara. Data proyek, klien, dan vendor tetap
        tersimpan dan tidak dihapus. Portal klien tetap bisa dibuka oleh pasangan.
      </p>
      {canManage ? (
        <Link
          to={ROUTE_PATHS.subscription}
          className="mt-6 inline-flex h-9 items-center rounded-md bg-navy-900 px-4 text-sm font-semibold text-white hover:bg-navy-800"
        >
          Perpanjang langganan
        </Link>
      ) : (
        <p className="mt-6 text-[13px] font-medium text-text-primary">Hubungi Owner WO Anda untuk memperpanjang langganan.</p>
      )}
    </div>
  );
}
