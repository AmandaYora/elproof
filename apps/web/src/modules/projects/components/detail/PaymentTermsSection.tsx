import { useEffect, useMemo, useState } from "react";
import { CalendarClock, Pencil, Plus } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Drawer } from "@/shared/components/ui/Drawer";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { TERM_STATUS_TONE } from "@/modules/projects/lib/status";
import { defaultTermTemplate } from "@/modules/projects/lib/terms";
import { PaymentTermsEditor, validateTermRows } from "@/modules/projects/components/PaymentTermsEditor";
import type { PaymentTerm, PaymentTermInput } from "@/modules/projects/types";

// Overdue first, then what is coming up by date, paid last (FLOW-IDEAL 2.4).
const GROUP_ORDER: Record<PaymentTerm["status"], number> = { overdue: 0, partial: 1, due_soon: 1, upcoming: 1, paid: 2 };

/** The client's payment schedule: what is due when, and how each installment stands. */
export function PaymentTermsSection({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const terms = useProjectStore((s) => s.paymentTerms);
  const termsFor = useProjectStore((s) => s.paymentTermsFor);
  const fetchPaymentTerms = useProjectStore((s) => s.fetchPaymentTerms);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const canRecord = useCan("finance.record");
  const [editing, setEditing] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  // The editor replaces the whole schedule, so it must never open before this
  // project's own schedule has loaded (it would offer a blank or foreign one).
  const loaded = termsFor === projectId;

  useEffect(() => {
    setLoadFailed(false);
    fetchPaymentTerms(projectId).catch(() => setLoadFailed(true));
    // A payment recorded in the section below changes how installments stand.
  }, [projectId, fetchPaymentTerms, clientPayments, attempt]);

  const sorted = useMemo(
    () => [...terms].sort((a, b) => GROUP_ORDER[a.status] - GROUP_ORDER[b.status] || a.dueDate.localeCompare(b.dueDate)),
    [terms],
  );
  const contractValue = project?.contractValue ?? 0;

  return (
    <>
      <Card>
        <CardHeader
          title="Jadwal Termin"
          subtitle="Kapan klien harus membayar berapa, dan sudah sampai mana."
          action={
            canRecord && loaded && terms.length > 0 ? (
              <Button size="sm" variant="secondary" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => setEditing(true)}>
                Atur jadwal
              </Button>
            ) : undefined
          }
        />
        <CardContent>
          {!loaded ? (
            loadFailed ? (
              <LoadError message="Jadwal termin belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />
            ) : (
              <SkeletonRows rows={3} label="Memuat jadwal termin" />
            )
          ) : terms.length === 0 ? (
            <EmptyState
              title="Belum ada jadwal termin"
              description="Dengan jadwal, Anda langsung tahu termin mana yang terlambat tanpa menghitung sendiri."
              action={
                canRecord ? (
                  <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setEditing(true)}>
                    Buat jadwal termin
                  </Button>
                ) : undefined
              }
            />
          ) : (
            <ul className="flex flex-col divide-y divide-border-light">
              {sorted.map((t) => (
                <li key={t.id} className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 py-3 first:pt-0 last:pb-0">
                  <div className="min-w-0">
                    <p className="text-[13.5px] font-semibold text-text-primary">{t.label}</p>
                    <p className="mt-0.5 flex items-center gap-1.5 text-[12.5px] text-text-secondary">
                      <CalendarClock className="h-3.5 w-3.5" aria-hidden />
                      Jatuh tempo {formatDate(t.dueDate)}
                    </p>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-right">
                      <p className="text-[13.5px] font-semibold tabular-nums text-text-primary">{formatCurrency(t.amount)}</p>
                      {t.status !== "paid" && t.paid > 0 && (
                        <p className="text-[12px] tabular-nums text-text-secondary">Sisa {formatCurrency(t.remaining)}</p>
                      )}
                    </div>
                    <Badge tone={TERM_STATUS_TONE[t.status]}>{TERM_STATUS_LABELS.staff[t.status]}</Badge>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      {editing && loaded && project && (
        <PaymentTermsDrawer
          projectId={projectId}
          contractValue={contractValue}
          bookingDate={project.prepStartDate}
          eventDate={project.eventDate}
          initial={terms.map((t) => ({ label: t.label, amount: t.amount, dueDate: t.dueDate }))}
          onClose={() => setEditing(false)}
        />
      )}
    </>
  );
}

function PaymentTermsDrawer({
  projectId,
  contractValue,
  bookingDate,
  eventDate,
  initial,
  onClose,
}: {
  projectId: string;
  contractValue: number;
  bookingDate: string;
  eventDate: string;
  initial: PaymentTermInput[];
  onClose: () => void;
}) {
  const save = useProjectStore((s) => s.savePaymentTerms);
  const [rows, setRows] = useState<PaymentTermInput[]>(() =>
    initial.length > 0 ? initial : defaultTermTemplate(contractValue, bookingDate, eventDate),
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [rowErrors, setRowErrors] = useState<Record<number, string>>({});

  const total = rows.reduce((sum, r) => sum + r.amount, 0);
  const over = contractValue > 0 && total > contractValue;

  async function handleSave() {
    const problems = validateTermRows(rows);
    setRowErrors(problems);
    if (Object.keys(problems).length > 0 || over) {
      setError(over ? `Jumlah termin melebihi nilai kontrak ${formatCurrency(contractValue)}.` : "Lengkapi dulu bagian yang ditandai.");
      return;
    }
    setError(null);
    setSaving(true);
    try {
      await save(projectId, rows);
      toast.success("Jadwal termin disimpan");
      onClose();
    } catch (err) {
      setError(getApiErrorMessage(err, "Jadwal belum tersimpan. Periksa koneksi lalu coba lagi."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Drawer
      open
      onClose={onClose}
      busy={saving}
      size="lg"
      title="Atur jadwal termin"
      description="Jumlah semua termin tidak boleh melebihi nilai kontrak."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button onClick={() => void handleSave()} disabled={saving}>{saving ? "Menyimpan..." : "Simpan jadwal"}</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <PaymentTermsEditor
          rows={rows}
          setRows={setRows}
          rowErrors={rowErrors}
          contractValue={contractValue}
          bookingDate={bookingDate}
          eventDate={eventDate}
        />

        {error && (
          <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
            {error}
          </p>
        )}
      </div>
    </Drawer>
  );
}
