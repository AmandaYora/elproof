import { useState, type ReactNode } from "react";
import { AlertTriangle, EyeOff, Pencil, Plus, RefreshCw, Trash2 } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { RowActions } from "@/shared/components/ui/RowActions";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { SUPPLIER_PAYER_LABELS } from "@/shared/lib/labels";
import { toast } from "@/shared/stores/useToastStore";
import { EventList, supplierKindLabel, toEventInput, draftFromEvent, type EventRowAction, type RincianEvent } from "@/modules/rincian";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { termDueText, validityText } from "@/modules/offers/lib/display";
import type { QuotationDialog } from "@/modules/offers/lib/page-dialog";
import type { QuotationDetail } from "@/modules/offers/types";

interface SectionProps {
  detail: QuotationDetail;
  /** The working copy can be changed by this person now (a draft, quotation.manage, not read-only). */
  canEdit: boolean;
  open: (dialog: QuotationDialog) => void;
}

/** One part of the quotation page, with an id the jump list points at. */
export function PageSection({ id, title, subtitle, action, children }: { id: string; title: string; subtitle?: ReactNode; action?: ReactNode; children: ReactNode }) {
  return (
    <Card id={id} className="scroll-mt-32">
      <CardHeader title={<span id={`${id}-judul`}>{title}</span>} subtitle={subtitle} action={action} />
      <CardContent>{children}</CardContent>
    </Card>
  );
}

function EditButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <Button size="sm" variant="secondary" icon={<Pencil className="h-3.5 w-3.5" />} onClick={onClick} aria-label={label}>
      Ubah
    </Button>
  );
}

/* ------------------------------------------------------------------------------------------------------- Info --- */

export function InfoSection({ detail, canEdit, open }: SectionProps) {
  const q = detail.quotation;
  const rows: Array<[string, ReactNode]> = [
    ["Pasangan", q.coupleNames],
    ["WhatsApp", q.contactWhatsapp || <span className="text-muted-foreground">Belum diisi</span>],
    ["PIC", q.picName || "-"],
    ["Tanggal dokumen", formatDate(q.docDate)],
    ["Masa berlaku", validityText(q)],
    ["Catatan internal", q.notes ? <span className="whitespace-pre-line">{q.notes}</span> : <span className="text-muted-foreground">-</span>],
  ];
  return (
    <PageSection id="info" title="Info" action={canEdit && <EditButton label="Ubah data penawaran" onClick={() => open({ kind: "header" })} />}>
      <dl className="grid grid-cols-1 gap-x-6 gap-y-3 text-[13.5px] sm:grid-cols-2">
        {rows.map(([label, value]) => (
          <div key={label} className="min-w-0">
            <dt className="text-[12.5px] text-muted-foreground">{label}</dt>
            <dd className="mt-0.5 break-words text-foreground">{value}</dd>
          </div>
        ))}
      </dl>
    </PageSection>
  );
}

/* ------------------------------------------------------------------------------------------------------ Acara --- */

export function EventsSection({ detail, events, canEdit, open }: SectionProps & { events: RincianEvent[] }) {
  const orderEvents = useQuotationStore((s) => s.orderEvents);
  const createEvent = useQuotationStore((s) => s.createEvent);
  const [busyId, setBusyId] = useState<string | null>(null);
  const id = detail.quotation.id;

  async function run(eventId: string, action: () => Promise<unknown>, done: string, fallback: string) {
    setBusyId(eventId);
    try {
      await action();
      toast.success(done);
    } catch (err) {
      toast.error(getApiErrorMessage(err, fallback));
    } finally {
      setBusyId(null);
    }
  }

  function onAction(event: RincianEvent, action: EventRowAction) {
    const index = events.findIndex((e) => e.id === event.id);
    switch (action) {
      case "edit":
        open({ kind: "event", event });
        return;
      case "delete":
        open({ kind: "deleteEvent", event });
        return;
      case "duplicate":
        void run(event.id, () => createEvent(id, toEventInput(draftFromEvent(event))), "Acara disalin", "Acara belum bisa disalin. Coba lagi.");
        return;
      case "up":
      case "down": {
        const ids = events.map((e) => e.id);
        const j = action === "up" ? index - 1 : index + 1;
        if (j < 0 || j >= ids.length) return;
        [ids[index], ids[j]] = [ids[j], ids[index]];
        void run(event.id, () => orderEvents(id, ids), `${event.kindName} dipindah ke urutan ${j + 1}`, "Urutan acara belum bisa diubah. Coba lagi.");
        return;
      }
      default:
        return;
    }
  }

  const add = canEdit && (
    <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => open({ kind: "event", event: null })}>
      Tambah acara
    </Button>
  );

  return (
    <PageSection id="acara" title="Acara" subtitle={events.length > 0 ? `${events.length} acara` : undefined} action={events.length > 0 ? add : undefined}>
      {events.length === 0 ? (
        <EmptyState title="Belum ada acara" description="Tambahkan acara agar tamu dan kapasitas venue bisa dicek." action={add || undefined} />
      ) : (
        <EventList events={events} canEdit={canEdit} busyId={busyId} onAction={onAction} hide={["main"]} />
      )}
    </PageSection>
  );
}

/* --------------------------------------------------------------------------------------------- Vendor & venue --- */

export function SuppliersSection({ detail, canEdit, canMargin, open }: SectionProps & { canMargin: boolean }) {
  const refreshSupplier = useQuotationStore((s) => s.refreshSupplier);
  const suppliers = detail.suppliers;
  const id = detail.quotation.id;

  async function refresh(supplierId: string, name: string) {
    try {
      await refreshSupplier(id, supplierId);
      toast.success(`${name} diperbarui dari direktori`);
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Belum bisa diperbarui dari direktori. Coba lagi."));
    }
  }

  const add = canEdit && (
    <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => open({ kind: "supplier", supplier: null })}>
      Tambah vendor
    </Button>
  );

  return (
    <PageSection id="vendor" title="Vendor & venue" subtitle="Venue masuk otomatis dari acara." action={suppliers.length > 0 ? add : undefined}>
      {suppliers.length === 0 ? (
        <EmptyState
          title="Belum ada vendor & venue"
          description="Venue tercatat saat dipilih di acara. Tambahkan vendor yang ikut dalam penawaran ini."
          action={add || undefined}
        />
      ) : (
        <ul className="flex flex-col divide-y divide-border-light">
          {suppliers.map((s) => (
            <li key={s.id} className="flex items-start justify-between gap-3 py-3 first:pt-0 last:pb-0">
              <div className="min-w-0 text-[13.5px]">
                <p className="font-semibold text-foreground">{s.name}</p>
                <p className="mt-0.5 text-[12.5px] text-muted-foreground">
                  {supplierKindLabel(s)} · Dibayar oleh {SUPPLIER_PAYER_LABELS[s.paidBy]}
                </p>
                {!s.showName && s.kind === "vendor" && (
                  <p className="mt-1 flex items-center gap-1.5 text-[12.5px] text-muted-foreground">
                    <EyeOff aria-hidden className="h-3.5 w-3.5" />
                    Nama tidak tampil di PDF
                  </p>
                )}
                {canMargin && s.paidBy === "WO" && s.agreedCost !== undefined && (
                  <p className="mt-1 text-[12.5px] text-text-secondary">
                    Biaya rekanan:{" "}
                    {s.agreedCost === null ? <span className="font-medium text-warning-text">belum diisi</span> : <span className="tabular-nums">{formatCurrency(s.agreedCost)}</span>}
                  </p>
                )}
                {s.notes && <p className="mt-1 text-[12.5px] whitespace-pre-line text-text-secondary">{s.notes}</p>}
              </div>
              {canEdit && (
                <RowActions
                  rowLabel={s.name}
                  primary={{ label: "Ubah", icon: <Pencil />, onClick: () => open({ kind: "supplier", supplier: s }) }}
                  items={[
                    { label: "Perbarui dari direktori", icon: <RefreshCw />, onSelect: () => void refresh(s.id, s.name) },
                    { label: "Hapus dari penawaran", icon: <Trash2 />, tone: "danger", onSelect: () => open({ kind: "deleteSupplier", supplier: s }), hidden: s.kind === "venue" },
                  ]}
                />
              )}
            </li>
          ))}
        </ul>
      )}
    </PageSection>
  );
}

/* ----------------------------------------------------------------------------------------------------- Termin --- */

export function TermsSection({ detail, canEdit, open }: SectionProps) {
  const events = new Map(detail.events.map((e) => [e.id, e.kindName]));
  const mainName = detail.events[0]?.kindName ?? "acara utama";
  const total = detail.totals?.total;
  return (
    <PageSection id="termin" title="Termin" action={canEdit && <EditButton label="Ubah termin" onClick={() => open({ kind: "terms" })} />}>
      {detail.termsMismatch && total !== undefined && (
        <p role="status" className="mb-3 flex items-start gap-2 rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] text-warning-text">
          <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
          <span>
            Jumlah termin tidak sama dengan nilai kontrak ({formatCurrency(total)}). Sesuaikan termin sebelum membuat revisi.
          </span>
        </p>
      )}
      {detail.terms.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">Belum ada termin.</p>
      ) : (
        <ol className="flex flex-col divide-y divide-border-light">
          {detail.terms.map((t) => (
            <li key={t.id} className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-2.5 text-[13.5px] first:pt-0 last:pb-0">
              <span className="min-w-0">
                <span className="font-semibold text-foreground">{t.label}</span>
                {t.percent !== null && <span className="text-text-secondary"> · {t.percent}%</span>}
                <span className="block text-[12.5px] text-muted-foreground">{termDueText(t, t.dueEventId ? (events.get(t.dueEventId) ?? mainName) : mainName)}</span>
              </span>
              <span className="tabular-nums text-foreground">{detail.totals ? formatCurrency(t.computed) : "-"}</span>
            </li>
          ))}
        </ol>
      )}
    </PageSection>
  );
}

/* --------------------------------------------------------------------------------------- Syarat & ketentuan --- */

export function TermsTextSection({ detail, canEdit, open }: SectionProps) {
  const [all, setAll] = useState(false);
  const long = detail.termsText.length > 600;
  return (
    <PageSection id="sk" title="Syarat & ketentuan" action={canEdit && <EditButton label="Ubah syarat & ketentuan" onClick={() => open({ kind: "termsText" })} />}>
      {detail.termsText ? (
        <>
          <p className="text-[13px] leading-relaxed whitespace-pre-line text-text-secondary">{long && !all ? `${detail.termsText.slice(0, 600)}…` : detail.termsText}</p>
          {long && (
            <Button size="sm" variant="ghost" className="mt-2" aria-expanded={all} onClick={() => setAll((v) => !v)}>
              {all ? "Tampilkan lebih sedikit" : "Tampilkan semua"}
            </Button>
          )}
        </>
      ) : (
        <p className="text-[13px] text-muted-foreground">Belum ada syarat & ketentuan. Isi bawaannya diatur di Profil usaha › Penawaran.</p>
      )}
    </PageSection>
  );
}
