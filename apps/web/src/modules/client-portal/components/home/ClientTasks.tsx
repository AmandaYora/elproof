import { useState } from "react";
import { Check, CircleCheck, ClipboardList } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { toast } from "@/shared/stores/useToastStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate, formatEventOffset } from "@/shared/lib/formatters";
import { daysUntil } from "@/modules/projects/lib/dates";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import type { ClientTask } from "@/modules/client-portal/types";

/** Tasks due within this many days (or already late) are lifted into "Yang perlu Anda lakukan". */
export const URGENT_TASK_DAYS = 14;

export const isOpenTask = (t: ClientTask) => t.status !== "Completed" && t.status !== "Cancelled";
export const isUrgentTask = (t: ClientTask) => isOpenTask(t) && daysUntil(t.targetDate) <= URGENT_TASK_DAYS;

/** "Tenggat 20 Nov 2026 · 4 hari lagi", or that it is past, said without blame. */
export function taskDueText(t: ClientTask): string {
  const d = daysUntil(t.targetDate);
  const when = d < 0 ? `lewat ${Math.abs(d)} hari` : d === 0 ? "hari ini" : `${d} hari lagi`;
  return `Tenggat ${formatDate(t.targetDate)} · ${when}`;
}

/** "Tandai selesai": shown done at once, put back with a message when the server refuses. */
export function CompleteTaskButton({ task, size = "sm" }: { task: ClientTask; size?: "sm" | "default" }) {
  const can = useCan("portal.tasks.complete");
  const completeTask = useClientHomeStore((s) => s.completeTask);
  if (!can || !isOpenTask(task)) return null;
  return (
    <Button
      size={size}
      onClick={() => {
        completeTask(task.id).then(
          () => toast.success(`"${task.name}" ditandai selesai`),
          (err) => toast.error(getApiErrorMessage(err, "Tugas belum bisa ditandai selesai. Periksa koneksi, lalu coba lagi.")),
        );
      }}
    >
      <Check aria-hidden /> Tandai selesai
    </Button>
  );
}

/**
 * "Tugas Anda": what the WO asked the couple to do (send the guest list, choose the tasting menu). Tasks due soon are
 * already in "Yang perlu Anda lakukan", so this lists the later ones; finished tasks fold into "Sudah selesai (n)".
 */
export function ClientTasksCard({ tasks, eventDate }: { tasks: ClientTask[]; eventDate: string }) {
  const [showDone, setShowDone] = useState(false);
  const later = tasks.filter((t) => isOpenTask(t) && !isUrgentTask(t)).sort((a, b) => a.targetDate.localeCompare(b.targetDate));
  const lifted = tasks.filter(isUrgentTask).length;
  const done = tasks.filter((t) => t.status === "Completed").sort((a, b) => (b.completedDate ?? "").localeCompare(a.completedDate ?? ""));
  if (later.length === 0 && done.length === 0) return null;

  return (
    <section aria-labelledby="tasks-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="tasks-heading" className="text-base font-semibold text-foreground">Tugas Anda</h2>
      {lifted > 0 && <p className="mt-1 text-[13px] text-muted-foreground">Tugas yang tenggatnya dekat ada di daftar "Yang perlu Anda lakukan".</p>}

      {later.length > 0 && (
        <ul className="mt-3 flex flex-col divide-y divide-border-light">
          {later.map((t) => (
            <li key={t.id} className="flex flex-wrap items-center gap-x-3 gap-y-2 py-3 first:pt-1">
              <ClipboardList className="size-4 shrink-0 text-muted-foreground" aria-hidden />
              <span className="min-w-0 flex-1">
                <span className="block text-[14px] font-medium text-foreground">{t.name}</span>
                <span className="block text-[13px] text-muted-foreground">
                  {formatEventOffset(t.targetDate, eventDate)} · {formatDate(t.targetDate)}
                </span>
              </span>
              <CompleteTaskButton task={t} />
            </li>
          ))}
        </ul>
      )}

      {done.length > 0 && (
        <div className="mt-3 border-t border-border-light pt-3">
          <button
            type="button"
            aria-expanded={showDone}
            onClick={() => setShowDone((v) => !v)}
            className="flex min-h-11 w-full items-center justify-between rounded-md text-[14px] font-medium text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
          >
            Sudah selesai ({done.length})
            <span className="text-[13px] text-muted-foreground">{showDone ? "Sembunyikan" : "Lihat"}</span>
          </button>
          {showDone && (
            <ul className="mt-1 flex flex-col gap-2">
              {done.map((t) => (
                <li key={t.id} className="flex items-start gap-3 text-[14px]">
                  <CircleCheck className="mt-0.5 size-4 shrink-0 text-success" aria-hidden />
                  <span className="min-w-0">
                    <span className="block text-foreground">{t.name}</span>
                    <span className="block text-[13px] text-muted-foreground">Selesai {formatDate(t.completedDate)}</span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  );
}

