import { useEffect, useState, type ReactNode } from "react";
import { Minus, Pencil, Plus } from "lucide-react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { cn } from "@/shared/lib/cn";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { SUPPLIER_PAYER_LABELS } from "@/shared/lib/labels";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import type { CompareChange, CompareRow, QuotationCompare, QuotationRevision, SnapshotEvent, SnapshotLine, SnapshotSupplier } from "@/modules/offers/types";

const CHANGE: Record<CompareChange, { label: string; icon: ReactNode; className: string }> = {
  added: { label: "Ditambah", icon: <Plus aria-hidden className="h-3.5 w-3.5" />, className: "text-success-text" },
  changed: { label: "Diubah", icon: <Pencil aria-hidden className="h-3.5 w-3.5" />, className: "text-info-text" },
  removed: { label: "Dihapus", icon: <Minus aria-hidden className="h-3.5 w-3.5" />, className: "text-danger-text" },
};

const LINE_KIND: Record<SnapshotLine["kind"], string> = {
  Package: "Paket",
  Component: "Isi paket",
  ExtraGuests: "Tambahan tamu",
  Item: "Item",
  Discount: "Potongan",
};

function lineText(l: SnapshotLine): string {
  const name = l.description || LINE_KIND[l.kind];
  if (l.kind === "Item" && l.quantity !== null && l.unitPrice !== null) {
    return `${name} ${l.quantity.toLocaleString("id-ID")} ${l.unit} × ${formatCurrency(l.unitPrice)}${l.isBonus ? " (bonus)" : ""}`;
  }
  return l.isBonus ? `${name} (bonus)` : name;
}

function lineAmount(l: SnapshotLine): string {
  if (l.kind === "Component") return "";
  const amount = l.kind === "Discount" ? -Math.abs(l.amount) : l.amount;
  return formatCurrency(amount);
}

function eventText(e: SnapshotEvent): string {
  const guests = e.sessions.reduce((s, x) => s + x.guests, 0);
  const sessions = e.sessions.length > 1 ? ` · ${e.sessions.length} sesi` : "";
  return `${e.kindName} · ${e.eventDate ? formatDate(e.eventDate) : "tanggal belum ada"} · ${guests.toLocaleString("id-ID")} tamu${sessions}`;
}

function supplierText(s: SnapshotSupplier): string {
  return `${s.name} · dibayar oleh ${SUPPLIER_PAYER_LABELS[s.paidBy]}`;
}

interface Props {
  quotationId: string;
  revisions: QuotationRevision[];
  /** The pair to open with; to = 0 is the working copy (the changes not sent yet). */
  from: number;
  to: number;
  onClose: () => void;
}

/** "Bandingkan revisi" (DESAIN §6.4): what was added, changed and removed between two revisions, and the contract value before → after. */
export default function CompareDrawer({ quotationId, revisions, from: initialFrom, to: initialTo, onClose }: Props) {
  const compare = useQuotationStore((s) => s.compare);
  const [from, setFrom] = useState(initialFrom);
  const [to, setTo] = useState(initialTo);
  const [result, setResult] = useState<QuotationCompare | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setResult(null);
    setFailed(false);
    compare(quotationId, from, to).then(
      (r) => !cancelled && setResult(r),
      () => !cancelled && setFailed(true),
    );
    return () => {
      cancelled = true;
    };
  }, [compare, quotationId, from, to, attempt]);

  const name = (n: number) => (n === 0 ? "Perubahan yang belum dikirim" : `Revisi ${n}`);

  return (
    <Drawer open onClose={onClose} size="xl" title={`${name(from)} → ${name(to)}`} description="Bandingkan revisi" footer={<Button onClick={onClose}>Tutup</Button>}>
      <div className="flex flex-col gap-6">
        <div className="flex flex-wrap items-center gap-2 text-[13px] text-text-secondary">
          <Select aria-label="Revisi awal" className="w-44" value={String(from)} onChange={(e) => setFrom(Number(e.target.value))}>
            {revisions.map((r) => (
              <option key={r.n} value={String(r.n)}>
                {`Revisi ${r.n}`}
              </option>
            ))}
          </Select>
          <span aria-hidden>→</span>
          <Select aria-label="Revisi pembanding" className="w-60" value={String(to)} onChange={(e) => setTo(Number(e.target.value))}>
            {revisions.map((r) => (
              <option key={r.n} value={String(r.n)}>
                {`Revisi ${r.n}`}
              </option>
            ))}
            <option value="0">Perubahan yang belum dikirim</option>
          </Select>
        </div>

        {failed ? (
          <LoadError message="Perbandingan belum bisa dimuat." onRetry={() => setAttempt((n) => n + 1)} />
        ) : !result ? (
          <SkeletonRows rows={5} label="Memuat perbandingan" />
        ) : (
          <>
            <Section title="Rincian" unchanged={result.unchangedLines} unit="baris">
              {result.lines.map((row, i) => (
                <ChangeRow key={i} change={row.change} label={row.parent ? `${row.parent} › ` : ""} {...lineCells(row)} />
              ))}
            </Section>
            <Section title="Acara" unchanged={result.unchangedEvents} unit="acara">
              {result.events.map((row, i) => (
                <ChangeRow key={i} change={row.change} {...cells(row, eventText)} />
              ))}
            </Section>
            <Section title="Vendor & venue" unchanged={result.unchangedSuppliers} unit="vendor & venue">
              {result.suppliers.map((row, i) => (
                <ChangeRow key={i} change={row.change} {...cells(row, supplierText)} />
              ))}
            </Section>
            {(result.termsChanged || result.termsTextChanged || result.headerChanged) && (
              <ul className="flex flex-col gap-1 text-[13px] text-text-secondary">
                {result.headerChanged && <li>Data pasangan atau kontak diubah.</li>}
                {result.termsChanged && <li>Termin pembayaran diubah.</li>}
                {result.termsTextChanged && <li>Syarat & ketentuan diubah.</li>}
              </ul>
            )}
            <div className="flex flex-wrap items-baseline justify-between gap-2 border-t border-border pt-4">
              <span className="text-[13.5px] font-semibold text-foreground">Nilai kontrak</span>
              <span className="text-[13.5px] tabular-nums">
                {result.totalsBefore ? formatCurrency(result.totalsBefore.total) : "-"}
                <span aria-hidden> → </span>
                <span className="sr-only"> menjadi </span>
                <span className="font-semibold">{result.totalsAfter ? formatCurrency(result.totalsAfter.total) : "-"}</span>
              </span>
            </div>
          </>
        )}
      </div>
    </Drawer>
  );
}

function cells<T>(row: CompareRow<T>, text: (v: T) => string): { before: string; after: string } {
  return { before: row.before ? text(row.before) : "", after: row.after ? text(row.after) : "" };
}

function lineCells(row: CompareRow<SnapshotLine>): { before: string; after: string; amountBefore: string; amountAfter: string } {
  return {
    before: row.before ? lineText(row.before) : "",
    after: row.after ? lineText(row.after) : "",
    amountBefore: row.before ? lineAmount(row.before) : "",
    amountAfter: row.after ? lineAmount(row.after) : "",
  };
}

function Section({ title, unchanged, unit, children }: { title: string; unchanged: number; unit: string; children: ReactNode[] }) {
  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-[13.5px] font-semibold text-foreground">{title}</h3>
      {children.length > 0 ? <ul className="flex flex-col divide-y divide-border-light">{children}</ul> : <p className="text-[13px] text-muted-foreground">Tidak ada yang berubah.</p>}
      {unchanged > 0 && children.length > 0 && (
        <p className="text-xs text-muted-foreground">
          {unchanged} {unit} tidak berubah
        </p>
      )}
    </section>
  );
}

function ChangeRow({
  change,
  label = "",
  before,
  after,
  amountBefore = "",
  amountAfter = "",
}: {
  change: CompareChange;
  label?: string;
  before: string;
  after: string;
  amountBefore?: string;
  amountAfter?: string;
}) {
  const look = CHANGE[change];
  const text = change === "removed" ? before : after;
  return (
    <li className="grid grid-cols-[6.5rem_1fr_auto] items-start gap-3 py-2 text-[13px]">
      <span className={cn("inline-flex items-center gap-1.5 font-medium", look.className)}>
        {look.icon}
        {look.label}
      </span>
      <span className="min-w-0">
        <span className="text-muted-foreground">{label}</span>
        {change === "changed" && before !== after ? (
          <>
            <span className="text-muted-foreground line-through">{before}</span>
            <span className="block text-foreground">{after}</span>
          </>
        ) : (
          <span className={cn(change === "removed" && "line-through text-muted-foreground")}>{text}</span>
        )}
      </span>
      <span className="text-right whitespace-nowrap tabular-nums">
        {change === "changed" && amountBefore !== amountAfter ? (
          <>
            <span className="text-muted-foreground">{amountBefore}</span>
            <span aria-hidden> → </span>
            {amountAfter}
          </>
        ) : change === "removed" ? (
          amountBefore
        ) : (
          amountAfter
        )}
      </span>
    </li>
  );
}
