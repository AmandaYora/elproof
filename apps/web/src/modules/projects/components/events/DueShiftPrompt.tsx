import { useEffect, useRef, useState } from "react";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { daysBetween } from "@/shared/lib/formatters";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { hasUnpaidTerms, shiftUnpaidTerms } from "@/modules/projects/lib/terms";

/**
 * Hari H is the main event's date, so an event write can move it (a new date, another main event, the main event
 * deleted). The installments do not follow it on their own (QA RT-04): this offers to move the unpaid ones with it.
 * Mounted on the Acara tab, the only place an event is written.
 */
export function DueShiftPrompt({ projectId }: { projectId: string }) {
  const canRecordFinance = useCan("finance.record");
  const eventDate = useProjectStore((s) => (s.currentProject?.id === projectId ? s.currentProject.eventDate : null));
  const fetchPaymentTerms = useProjectStore((s) => s.fetchPaymentTerms);
  const savePaymentTerms = useProjectStore((s) => s.savePaymentTerms);
  const seen = useRef<string | null>(eventDate);
  const [days, setDays] = useState<number | null>(null);
  const [shifting, setShifting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const before = seen.current;
    seen.current = eventDate;
    if (!before || !eventDate || before === eventDate || !canRecordFinance) return;
    const moved = daysBetween(before, eventDate);
    if (moved === 0) return;
    let live = true;
    fetchPaymentTerms(projectId)
      .then(() => {
        if (live && hasUnpaidTerms(useProjectStore.getState().paymentTerms)) setDays(moved);
      })
      .catch(() => undefined); // the schedule can still be edited by hand on the Keuangan tab
    return () => {
      live = false;
    };
  }, [eventDate, canRecordFinance, fetchPaymentTerms, projectId]);

  async function shift() {
    if (days === null) return;
    setError(null);
    setShifting(true);
    try {
      // Read the schedule as it is now, not as it was when the question opened.
      await savePaymentTerms(projectId, shiftUnpaidTerms(useProjectStore.getState().paymentTerms, days));
      setDays(null);
      toast.success("Jatuh tempo termin digeser");
    } catch (err) {
      setError(getApiErrorMessage(err, "Jatuh tempo termin belum digeser. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setShifting(false);
    }
  }

  const way = days !== null && days > 0 ? "mundur" : "maju";
  return (
    <ConfirmDialog
      open={days !== null}
      title="Geser jatuh tempo termin?"
      confirmLabel="Geser jatuh tempo"
      cancelLabel="Biarkan"
      tone="primary"
      isSubmitting={shifting}
      error={error}
      onConfirm={() => void shift()}
      onClose={() => {
        setDays(null);
        setError(null);
      }}
    >
      {days !== null && (
        <p>
          Hari H {way} {Math.abs(days)} hari. Geser jatuh tempo termin yang belum lunas {way} {Math.abs(days)} hari juga? Termin yang
          sudah lunas tidak berubah. Pilih Biarkan untuk mengatur jatuh tempo sendiri di tab Keuangan.
        </p>
      )}
    </ConfirmDialog>
  );
}
