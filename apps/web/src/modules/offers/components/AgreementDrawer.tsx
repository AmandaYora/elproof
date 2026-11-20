import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { GitCompare } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Drawer } from "@/shared/components/ui/Drawer";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ACCEPT_CHANNEL_LABELS, LINE_KIND_LABELS, SUPPLIER_PAYER_LABELS } from "@/shared/lib/labels";
import { useCan } from "@/shared/stores/usePermissionStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { formatEventDay, type LineKind } from "@/modules/rincian";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { termDueText } from "@/modules/offers/lib/display";
import type { ProjectAgreement, SnapshotContent } from "@/modules/offers/types";

/** What the project holds now, in the few fields the comparison reads (the project screen passes its own rows). */
export interface AgreementCurrent {
  /** null = not loaded or not visible to this person. */
  total: number | null;
  lines: Array<{ kind: string; description: string; amount?: number; parentId: string }>;
  events: Array<{ kindName: string; eventDate: string }>;
}

interface Props {
  projectId: string;
  current: AgreementCurrent | null;
  onClose: () => void;
}

function lineName(l: { kind: string; description: string }): string {
  return l.description || LINE_KIND_LABELS[l.kind as LineKind] || "Baris";
}

/**
 * "Kesepakatan awal" (DESAIN §6.10): the accepted quotation revision a project was made from, read-only, as it was
 * agreed. [Bandingkan dengan rincian sekarang] sets it beside the project's rincian and events today.
 */
export default function AgreementDrawer({ projectId, current, onClose }: Props) {
  const fetchAgreement = useQuotationStore((s) => s.fetchAgreement);
  const canViewQuotation = useCan("quotation.view");
  const [agreement, setAgreement] = useState<ProjectAgreement | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [comparing, setComparing] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    fetchAgreement(projectId).then(
      (a) => !cancelled && setAgreement(a),
      (err) => !cancelled && setError(getApiErrorMessage(err, "Kesepakatan awal belum bisa dimuat. Coba lagi.")),
    );
    return () => {
      cancelled = true;
    };
  }, [fetchAgreement, projectId, attempt]);

  const description = agreement
    ? [
        `${agreement.quotationNumber} Revisi ${agreement.revisionN}`,
        agreement.acceptedOn ? `diterima ${formatDate(agreement.acceptedOn)}` : "",
        agreement.acceptedChannel ? `lewat ${ACCEPT_CHANNEL_LABELS[agreement.acceptedChannel]}` : "",
      ]
        .filter(Boolean)
        .join(" · ")
    : "Penawaran yang diterima klien saat proyek dibuat.";

  return (
    <Drawer
      open
      onClose={onClose}
      size="lg"
      title="Kesepakatan awal"
      description={description}
      footer={
        // Three controls do not fit one row on a phone: they wrap, the link stays first.
        <div className="flex w-full flex-wrap items-center justify-end gap-2">
          {agreement && canViewQuotation && (
            <Link to={ROUTE_PATHS.quotationDetail(agreement.quotationId)} className="mr-auto self-center text-[13px] font-medium text-primary underline-offset-2 hover:underline">
              Buka penawaran
            </Link>
          )}
          {agreement && current && (
            <Button variant="secondary" icon={<GitCompare className="h-4 w-4" />} aria-pressed={comparing} onClick={() => setComparing((v) => !v)}>
              {comparing ? "Tampilkan kesepakatan" : "Bandingkan dengan rincian sekarang"}
            </Button>
          )}
          <Button variant="secondary" onClick={onClose}>
            Tutup
          </Button>
        </div>
      }
    >
      {error ? (
        <LoadError message={error} onRetry={() => setAttempt((n) => n + 1)} />
      ) : !agreement ? (
        <SkeletonRows rows={6} label="Memuat kesepakatan awal" />
      ) : comparing && current ? (
        <AgreementComparison agreement={agreement} current={current} />
      ) : (
        <AgreementContent agreement={agreement} />
      )}
    </Drawer>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-2 border-t border-border-light pt-4 first:border-t-0 first:pt-0">
      <h3 className="text-[13.5px] font-semibold text-foreground">{title}</h3>
      {children}
    </section>
  );
}

function AgreementContent({ agreement }: { agreement: ProjectAgreement }) {
  const c: SnapshotContent = agreement.content;
  const mainName = c.events[0]?.kindName ?? "acara utama";
  return (
    <div className="flex flex-col gap-5 text-[13.5px]">
      <p className="flex flex-wrap items-baseline justify-between gap-2 rounded-md bg-surface-muted px-3.5 py-2.5">
        <span className="font-semibold text-foreground">Nilai kontrak</span>
        <span className="font-semibold tabular-nums">{formatCurrency(agreement.contractTotal)}</span>
      </p>

      <Section title="Acara">
        {c.events.length === 0 ? (
          <p className="text-muted-foreground">Tidak ada acara.</p>
        ) : (
          <ul className="flex flex-col divide-y divide-border-light">
            {c.events.map((e, i) => {
              const venue = e.venueIndex !== null ? c.suppliers[e.venueIndex]?.name : undefined;
              const guests = e.sessions.reduce((n, s) => n + s.guests, 0);
              return (
                <li key={i} className="py-2 first:pt-0 last:pb-0">
                  <p className="font-medium text-foreground">{e.kindName}</p>
                  <p className="text-[12.5px] text-muted-foreground">
                    {formatEventDay(e.eventDate)}
                    {venue ? ` · ${venue}` : ""}
                    {guests > 0 ? ` · ${guests} tamu` : ""}
                  </p>
                </li>
              );
            })}
          </ul>
        )}
      </Section>

      <Section title="Vendor & venue">
        {c.suppliers.length === 0 ? (
          <p className="text-muted-foreground">Tidak ada vendor & venue.</p>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {c.suppliers.map((s, i) => (
              <li key={i} className="flex flex-wrap justify-between gap-x-3">
                <span className="text-foreground">{s.name}</span>
                <span className="text-[12.5px] text-muted-foreground">
                  {s.kind === "venue" ? "Venue" : "Vendor"} · Dibayar oleh {SUPPLIER_PAYER_LABELS[s.paidBy]}
                  {s.agreedCost !== null ? ` · biaya rekanan ${formatCurrency(s.agreedCost)}` : ""}
                </span>
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Section title="Rincian">
        {c.lines.length === 0 ? (
          <p className="text-muted-foreground">Tidak ada baris rincian.</p>
        ) : (
          <ul className="flex flex-col divide-y divide-border-light">
            {c.lines.map((l, i) => {
              const component = l.kind === "Component";
              return (
                <li key={i} className={`flex flex-wrap items-baseline justify-between gap-x-3 py-1.5 ${component ? "pl-5 text-[12.5px] text-text-secondary" : ""}`}>
                  <span className="min-w-0">
                    {lineName(l)}
                    {l.isBonus && <span className="text-muted-foreground"> · bonus</span>}
                    {l.quantity !== null && l.quantity > 1 && l.unit && (
                      <span className="text-muted-foreground">
                        {" "}
                        · {l.quantity} {l.unit}
                      </span>
                    )}
                  </span>
                  {!component && <span className={`tabular-nums ${l.kind === "Discount" ? "text-success-text" : ""}`}>{formatCurrency(l.amount)}</span>}
                </li>
              );
            })}
          </ul>
        )}
        {c.totals && (
          <dl className="mt-1 flex flex-col gap-1 text-[12.5px] text-text-secondary">
            {c.totals.discount !== 0 && (
              <div className="flex justify-between">
                <dt>Potongan</dt>
                <dd className="tabular-nums">{formatCurrency(-Math.abs(c.totals.discount))}</dd>
              </div>
            )}
            {c.isPkp && (
              <div className="flex justify-between">
                <dt>PPN</dt>
                <dd className="tabular-nums">{formatCurrency(c.totals.tax)}</dd>
              </div>
            )}
          </dl>
        )}
      </Section>

      <Section title="Termin">
        {c.terms.length === 0 ? (
          <p className="text-muted-foreground">Tidak ada termin.</p>
        ) : (
          <ol className="flex flex-col divide-y divide-border-light">
            {c.terms.map((t, i) => (
              <li key={i} className="flex flex-wrap items-baseline justify-between gap-x-3 py-1.5">
                <span>
                  {t.label}
                  {t.percent !== null && <span className="text-text-secondary"> · {t.percent}%</span>}
                  <span className="block text-[12.5px] text-muted-foreground">
                    {termDueText(t, t.dueEventIndex !== null ? (c.events[t.dueEventIndex]?.kindName ?? mainName) : mainName)}
                  </span>
                </span>
                <span className="tabular-nums">{formatCurrency(t.computed)}</span>
              </li>
            ))}
          </ol>
        )}
      </Section>

      <Section title="Syarat & ketentuan">
        {c.termsText ? (
          <p className="text-[13px] leading-relaxed whitespace-pre-line text-text-secondary">{c.termsText}</p>
        ) : (
          <p className="text-muted-foreground">Tidak ada syarat & ketentuan.</p>
        )}
      </Section>
    </div>
  );
}

type Change = { name: string; kind: "Ditambah" | "Diubah" | "Dihapus"; detail: string };

const CHANGE_TONE: Record<Change["kind"], string> = {
  Ditambah: "text-success-text",
  Diubah: "text-warning-text",
  Dihapus: "text-danger-text",
};

/** Lines are matched by name and kind (packages' components are left out: the package line carries the price). */
function lineChanges(agreed: SnapshotContent["lines"], now: AgreementCurrent["lines"]): Change[] {
  const key = (l: { kind: string; description: string }) => `${l.kind}|${lineName(l).trim().toLowerCase()}`;
  const before = new Map<string, { name: string; amount: number }[]>();
  for (const l of agreed) {
    if (l.kind === "Component") continue;
    const list = before.get(key(l)) ?? [];
    list.push({ name: lineName(l), amount: l.amount });
    before.set(key(l), list);
  }
  const changes: Change[] = [];
  for (const l of now) {
    if (l.kind === "Component" || l.parentId) continue;
    const list = before.get(key(l));
    const match = list?.shift();
    const amount = l.amount ?? 0;
    if (!match) changes.push({ name: lineName(l), kind: "Ditambah", detail: formatCurrency(amount) });
    else if (l.amount !== undefined && match.amount !== amount) changes.push({ name: lineName(l), kind: "Diubah", detail: `${formatCurrency(match.amount)} → ${formatCurrency(amount)}` });
  }
  for (const list of before.values()) for (const l of list) changes.push({ name: l.name, kind: "Dihapus", detail: formatCurrency(l.amount) });
  return changes;
}

function eventChanges(agreed: SnapshotContent["events"], now: AgreementCurrent["events"]): Change[] {
  const before = new Map<string, string[]>();
  for (const e of agreed) before.set(e.kindName, [...(before.get(e.kindName) ?? []), e.eventDate]);
  const changes: Change[] = [];
  for (const e of now) {
    const date = before.get(e.kindName)?.shift();
    if (date === undefined) changes.push({ name: e.kindName, kind: "Ditambah", detail: formatEventDay(e.eventDate) });
    else if (date && date !== e.eventDate) changes.push({ name: e.kindName, kind: "Diubah", detail: `${formatEventDay(date)} → ${formatEventDay(e.eventDate)}` });
  }
  for (const [name, dates] of before) for (const d of dates) changes.push({ name, kind: "Dihapus", detail: formatEventDay(d) });
  return changes;
}

function ChangeList({ changes, empty }: { changes: Change[]; empty: string }) {
  if (changes.length === 0) return <p className="text-muted-foreground">{empty}</p>;
  return (
    <ul className="flex flex-col divide-y divide-border-light">
      {changes.map((c, i) => (
        <li key={i} className="flex flex-wrap items-baseline justify-between gap-x-3 py-1.5">
          <span className="min-w-0">
            <span className={`font-medium ${CHANGE_TONE[c.kind]}`}>{c.kind}</span> {c.name}
          </span>
          <span className="tabular-nums text-text-secondary">{c.detail}</span>
        </li>
      ))}
    </ul>
  );
}

function AgreementComparison({ agreement, current }: { agreement: ProjectAgreement; current: AgreementCurrent }) {
  const lines = useMemo(() => lineChanges(agreement.content.lines, current.lines), [agreement, current.lines]);
  const events = useMemo(() => eventChanges(agreement.content.events, current.events), [agreement, current.events]);
  const delta = current.total !== null ? current.total - agreement.contractTotal : null;
  return (
    <div className="flex flex-col gap-5 text-[13.5px]">
      <p className="text-[13px] text-text-secondary">Kesepakatan awal dibandingkan dengan acara dan rincian proyek saat ini.</p>
      <dl className="grid grid-cols-1 gap-2 rounded-md bg-surface-muted px-3.5 py-2.5 sm:grid-cols-3">
        <div>
          <dt className="text-[12.5px] text-muted-foreground">Nilai kontrak awal</dt>
          <dd className="font-semibold tabular-nums">{formatCurrency(agreement.contractTotal)}</dd>
        </div>
        <div>
          <dt className="text-[12.5px] text-muted-foreground">Nilai kontrak sekarang</dt>
          <dd className="font-semibold tabular-nums">{current.total !== null ? formatCurrency(current.total) : "-"}</dd>
        </div>
        <div>
          <dt className="text-[12.5px] text-muted-foreground">Selisih</dt>
          <dd className="font-semibold tabular-nums">
            {delta === null ? "-" : delta === 0 ? "Tidak berubah" : `${delta > 0 ? "+" : "-"}${formatCurrency(Math.abs(delta))}`}
          </dd>
        </div>
      </dl>
      <Section title="Acara">
        <ChangeList changes={events} empty="Acara sama dengan kesepakatan awal." />
      </Section>
      <Section title="Rincian">
        <ChangeList changes={lines} empty="Rincian sama dengan kesepakatan awal." />
      </Section>
    </div>
  );
}
