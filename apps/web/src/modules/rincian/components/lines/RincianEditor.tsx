import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { ArrowDown, ArrowUp, ChevronDown, ChevronRight, CircleAlert, Gift, Info, Package, PanelRight, Plus, ReceiptText, Tag, Trash2 } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Input } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/shared/components/kit/dropdown-menu";
import { cn } from "@/shared/lib/cn";
import { formatCurrency } from "@/shared/lib/formatters";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { LINE_KIND_LABELS } from "@/shared/lib/labels";
import { toast } from "@/shared/stores/useToastStore";
import {
  extraGuestsTotal,
  indexRincian,
  moveRow,
  newKey,
  normalize,
  removeRow,
  restoreRows,
  rowErrors,
  rowsFromRincian,
  sameRows,
  toDrafts,
  topLevel,
  type ComputedIndex,
  type EditorRow,
  type RowGroup,
} from "@/modules/rincian/lib/editor";
import { extraGuestsFormula } from "@/modules/rincian/lib/describe";
import { toRincian, type RawRincian } from "@/modules/rincian/lib/wire";
import { CostSummaryPanel, negative } from "@/modules/rincian/components/lines/CostSummaryPanel";
import { ExtraGuestsHelp } from "@/modules/rincian/components/lines/ExtraGuestsHelp";
import type { LineDraft, NamedChoice, PackageChoice, Rincian, RincianLine, SupplierChoice } from "@/modules/rincian/types";

// Opened only after a click.
const PackagePickerModal = lazy(() => import("@/modules/rincian/components/lines/PackagePickerModal"));
const LineDetailDrawer = lazy(() => import("@/modules/rincian/components/lines/LineDetailDrawer"));

const PREVIEW_DELAY_MS = 500;

export interface RincianEditorProps {
  /** The saved rincian (its linesVersion is the If-Match of a save). */
  rincian: Rincian;
  /** project.edit: rows can be changed; otherwise the rincian is read-only. */
  canEdit: boolean;
  events: NamedChoice[];
  /** Vendors and venues the WO pays: what a line may point at. */
  suppliers: SupplierChoice[];
  categories: NamedChoice[];
  searchPackages: (q: string) => Promise<PackageChoice[]>;
  /** What the drafts come to (server-computed, writes nothing). */
  preview: (lines: LineDraft[], signal: AbortSignal) => Promise<Rincian>;
  /** Saves the whole rincian over `version`; a stale version is a 409 carrying data.current. */
  save: (lines: LineDraft[], version: string) => Promise<Rincian>;
  /** Reads the rincian again ("Muat versi terbaru"). */
  reload: () => Promise<Rincian>;
  /** Where "Margin belum lengkap (n vendor)" leads. */
  missingCostHref?: string;
  /** Shown in "Ringkasan biaya" under the numbers (e.g. the payment schedule no longer adding up). */
  summaryExtra?: ReactNode;
}

type FieldErrors = Record<string, Record<string, string>>;

/**
 * The rincian editor (DESAIN §5.3): packages from the catalog with their isi, the computed "Tambahan tamu", items,
 * bonus and potongan, edited in place and saved together ("Simpan rincian", If-Match). While anything changes, the
 * server previews what it comes to (500 ms after the last change); the screen never prices a line itself.
 */
export function RincianEditor({
  rincian,
  canEdit,
  events,
  suppliers,
  categories,
  searchPackages,
  preview,
  save,
  reload,
  missingCostHref,
  summaryExtra,
}: RincianEditorProps) {
  const savedRows = useMemo(() => rowsFromRincian(rincian), [rincian]);
  const [rows, setRows] = useState<EditorRow[]>(savedRows);
  const dirty = !sameRows(rows, savedRows);

  const [computed, setComputed] = useState<Rincian | null>(null);
  const [computing, setComputing] = useState(false);
  const [previewFailed, setPreviewFailed] = useState(false);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [general, setGeneral] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [conflict, setConflict] = useState<Rincian | null>(null);
  const [showDiff, setShowDiff] = useState(false);
  const [status, setStatus] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [picking, setPicking] = useState(false);
  const [detailKey, setDetailKey] = useState<string | null>(null);
  const [focusKey, setFocusKey] = useState<string | null>(null);

  // A new saved rincian (another tab, an acara whose guests changed, our own save): follow it while nothing is being
  // edited, and say when guests moved the price (DESAIN A4).
  const previousSaved = useRef(rincian);
  const ownSave = useRef(false);
  useEffect(() => {
    const before = previousSaved.current;
    previousSaved.current = rincian;
    if (before === rincian) return;
    if (!ownSave.current && before.projectId === rincian.projectId) {
      const was = extraGuestsTotal(before);
      const now = extraGuestsTotal(rincian);
      if (was !== now && rincian.totals) {
        const message = `Tambahan tamu berubah dari ${formatCurrency(was)} menjadi ${formatCurrency(now)}. Nilai kontrak ${formatCurrency(rincian.totals.total)}.`;
        setStatus(message);
        toast.info(message);
      }
    }
    ownSave.current = false;
    setRows((current) => (sameRows(current, rowsFromRincian(before)) ? rowsFromRincian(rincian) : current));
  }, [rincian]);

  // The preview: 500 ms after the last change, the earlier one cancelled.
  const lastTotal = useRef<number | undefined>(rincian.totals?.total);
  useEffect(() => {
    if (!dirty) {
      setComputed(null);
      setComputing(false);
      setPreviewFailed(false);
      return;
    }
    const controller = new AbortController();
    setComputing(true);
    const timer = setTimeout(() => {
      // An empty description is fine while typing: the preview only needs the numbers.
      const drafts = toDrafts(rows).map((d) => (d.description ? d : { ...d, description: LINE_KIND_LABELS[d.kind] }));
      preview(drafts, controller.signal)
        .then((r) => {
          setComputed(r);
          setPreviewFailed(false);
          setComputing(false);
          setErrors({});
          setGeneral([]);
          const total = r.totals?.total;
          if (total !== undefined && total !== lastTotal.current) {
            lastTotal.current = total;
            setStatus(`Nilai kontrak sekarang ${formatCurrency(total)}`);
          }
        })
        .catch((err) => {
          if (controller.signal.aborted) return;
          setComputing(false);
          setPreviewFailed(true);
          const fields = getApiFieldErrors(err);
          if (Object.keys(fields).length > 0) {
            const mapped = rowErrors(fields, drafts);
            // A missing description is said on save, not while the row is still being typed.
            for (const key of Object.keys(mapped.byRow)) {
              delete mapped.byRow[key].description;
              if (Object.keys(mapped.byRow[key]).length === 0) delete mapped.byRow[key];
            }
            setErrors(mapped.byRow);
            setGeneral(mapped.general);
          }
        });
    }, PREVIEW_DELAY_MS);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [rows, dirty, preview]);

  // A row just added gets the focus on its first field.
  useEffect(() => {
    if (!focusKey) return;
    const el = document.querySelector<HTMLElement>(`[data-line="${focusKey}"] input, [data-line="${focusKey}"] button`);
    el?.focus();
    setFocusKey(null);
  }, [focusKey, rows]);

  const shown = dirty ? (computed ?? rincian) : rincian;
  const index = useMemo(() => indexRincian(shown), [shown]);
  const eventName = useCallback((id: string) => events.find((e) => e.id === id)?.name ?? "", [events]);

  function update(key: string, changes: Partial<EditorRow>) {
    setRows((current) => current.map((r) => (r.key === key ? { ...r, ...changes } : r)));
    setErrors((current) => {
      if (!current[key]) return current;
      const next = { ...current, [key]: { ...current[key] } };
      for (const f of Object.keys(changes)) delete next[key][f];
      return next;
    });
  }

  function add(kind: "Item" | "Bonus" | "Discount") {
    const row: EditorRow = {
      key: newKey(),
      id: "",
      kind: kind === "Discount" ? "Discount" : "Item",
      parentKey: "",
      packageId: "",
      description: "",
      eventId: "",
      categoryId: "",
      supplierRef: "",
      quantity: kind === "Discount" ? 0 : 1,
      unit: "",
      unitPrice: 0,
      amount: 0,
      isBonus: kind === "Bonus",
    };
    setRows((current) => normalize([...current, row]));
    setFocusKey(row.key);
  }

  function addPackage(pkg: PackageChoice) {
    const row: EditorRow = {
      key: newKey(),
      id: "",
      kind: "Package",
      parentKey: "",
      packageId: pkg.id,
      description: pkg.name,
      eventId: "",
      categoryId: "",
      supplierRef: "",
      quantity: 1,
      unit: "",
      unitPrice: 0,
      amount: 0,
      isBonus: false,
    };
    setPicking(false);
    setRows((current) => normalize([...current, row]));
    setStatus(`${pkg.name} ditambahkan ke rincian`);
    setFocusKey(row.key);
  }

  function addComponent(pkgKey: string) {
    const row: EditorRow = {
      key: newKey(),
      id: "",
      kind: "Component",
      parentKey: pkgKey,
      packageId: "",
      description: "",
      eventId: "",
      categoryId: "",
      supplierRef: "",
      quantity: 0,
      unit: "",
      unitPrice: 0,
      amount: 0,
      isBonus: false,
    };
    setExpanded((s) => new Set(s).add(pkgKey));
    setRows((current) => normalize([...current, row]));
    setFocusKey(row.key);
  }

  function move(row: EditorRow, direction: -1 | 1) {
    const moved = moveRow(rows, row.key, direction);
    if (!moved) return;
    setRows(moved.rows);
    setStatus(`${row.description || LINE_KIND_LABELS[row.kind]} dipindah ke posisi ${moved.position} dari ${moved.count}`);
  }

  function remove(row: EditorRow) {
    const { rows: next, removed, index: at } = removeRow(rows, row.key);
    setRows(next);
    const name = row.description || LINE_KIND_LABELS[row.kind];
    toast.success(`${name} dihapus dari rincian`, { undo: () => setRows((current) => restoreRows(current, removed, at)) });
  }

  function discard() {
    setRows(savedRows);
    setErrors({});
    setGeneral([]);
    setConflict(null);
    setShowDiff(false);
  }

  async function handleSave() {
    if (!dirty || saving) return;
    const drafts = toDrafts(rows);
    const missing = rows.filter((r) => !r.description.trim());
    if (missing.length > 0) {
      setErrors(Object.fromEntries(missing.map((r) => [r.key, { description: "Isi uraian baris" }])));
      setGeneral([]);
      setFocusKey(missing[0].key);
      return;
    }
    setSaving(true);
    setGeneral([]);
    try {
      ownSave.current = true;
      const saved = await save(drafts, rincian.linesVersion ?? "");
      setRows(rowsFromRincian(saved));
      setConflict(null);
      setErrors({});
      const total = saved.totals?.total ?? 0;
      lastTotal.current = total;
      const mismatch =
        saved.termsMismatch && saved.termsTotal !== undefined
          ? ` Jadwal termin (${formatCurrency(saved.termsTotal)}) tidak sama dengan nilai kontrak (${formatCurrency(total)}).`
          : "";
      setStatus(`Rincian disimpan. Nilai kontrak ${formatCurrency(total)}.${mismatch}`);
      toast.success("Rincian disimpan");
    } catch (err) {
      ownSave.current = false;
      if (isStaleConflict(err)) {
        const current = (err as { response?: { data?: { data?: { current?: RawRincian } } } }).response?.data?.data?.current;
        setConflict(current ? toRincian(current) : rincian);
        return;
      }
      const fields = getApiFieldErrors(err);
      const mapped = rowErrors(fields, drafts);
      setErrors(mapped.byRow);
      const message = getApiErrorMessage(err, "Rincian belum tersimpan. Periksa koneksi Anda, lalu coba lagi.");
      setGeneral(mapped.general.length > 0 ? mapped.general : Object.keys(mapped.byRow).length > 0 ? [] : [message]);
      if (Object.keys(mapped.byRow).length > 0) toast.error("Rincian belum tersimpan. Periksa isian yang ditandai.");
    } finally {
      setSaving(false);
    }
  }

  async function loadLatest() {
    try {
      const latest = await reload();
      setRows(rowsFromRincian(latest));
      setConflict(null);
      setShowDiff(false);
      setErrors({});
      setGeneral([]);
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Versi terbaru belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  const errorCount = Object.values(errors).reduce((n, e) => n + Object.keys(e).length, 0) + general.length;
  const detailRow = detailKey ? rows.find((r) => r.key === detailKey) : undefined;
  const savedTotal = rincian.totals?.total;
  const shownTotal = shown.totals?.total;
  const delta = dirty && savedTotal !== undefined && shownTotal !== undefined ? shownTotal - savedTotal : 0;
  const empty = rows.length === 0;

  const addMenu = canEdit ? (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button size="sm" icon={<Plus className="size-3.5" />} />}>
        Tambah
        <ChevronDown aria-hidden className="ml-1 size-3.5" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => setPicking(true)}>
          <Package aria-hidden /> Paket dari katalog
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => add("Item")}>
          <ReceiptText aria-hidden /> Item
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => add("Bonus")}>
          <Gift aria-hidden /> Bonus
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => add("Discount")}>
          <Tag aria-hidden /> Potongan
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ) : null;

  return (
    <div className="flex flex-col gap-5 xl:grid xl:grid-cols-[minmax(0,1fr)_300px] xl:items-start">
      <section aria-labelledby="rincian-heading" className="min-w-0 rounded-lg border border-border bg-card">
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border-light px-5 py-4">
          <div className="min-w-0">
            <h3 id="rincian-heading" className="text-[16px] leading-snug font-semibold text-foreground">
              Rincian
            </h3>
            <p className="mt-0.5 text-[13px] text-muted-foreground">Nilai kontrak dihitung dari baris di bawah ini.</p>
          </div>
          {!empty && addMenu}
        </div>

        {/* Below 1280 px the panel sits under the table; this line keeps the total in view (DESAIN §5.4). */}
        {!empty && shown.totals && (
          <p className="flex flex-wrap items-center justify-between gap-2 border-b border-border-light px-5 py-2.5 text-[13px] xl:hidden">
            <span>
              Nilai kontrak{" "}
              <span className="font-semibold tabular-nums text-foreground">{computing ? "Menghitung…" : formatCurrency(shown.totals.total)}</span>
            </span>
            <a href="#ringkasan-biaya" className="inline-flex items-center gap-1 font-medium text-primary hover:underline">
              <PanelRight aria-hidden className="size-3.5" /> Lihat ringkasan
            </a>
          </p>
        )}

        {conflict && (
          <div role="alert" className="m-5 mb-0 rounded-md border border-warning/30 bg-warning-soft px-3.5 py-3 text-[13px] text-warning-text">
            <p className="font-medium">Rincian sudah diubah orang lain. Perubahan Anda belum disimpan.</p>
            <div className="mt-2 flex flex-wrap gap-2">
              <Button size="sm" variant="secondary" onClick={() => setShowDiff((v) => !v)} aria-expanded={showDiff}>
                Lihat perbedaan
              </Button>
              <Button size="sm" variant="secondary" onClick={() => void loadLatest()}>
                Muat versi terbaru
              </Button>
            </div>
            {showDiff && <ConflictDiff mine={rows} mineTotal={shownTotal} theirs={conflict} />}
          </div>
        )}

        {errorCount >= 3 || general.length > 0 ? (
          <div role="alert" className="m-5 mb-0 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] text-danger-text">
            <p className="font-medium">Periksa isian yang ditandai.</p>
            <ul className="mt-1 list-disc pl-5">
              {general.map((g) => (
                <li key={g}>{g}</li>
              ))}
              {Object.entries(errors).flatMap(([key, fields]) =>
                Object.entries(fields).map(([field, message]) => (
                  <li key={`${key}-${field}`}>
                    <button
                      type="button"
                      className="underline underline-offset-2 hover:no-underline"
                      onClick={() => document.querySelector<HTMLElement>(`[data-line="${key}"] [data-field="${field}"]`)?.focus()}
                    >
                      {message}
                    </button>
                  </li>
                )),
              )}
            </ul>
          </div>
        ) : null}

        {empty ? (
          <EmptyState
            title="Belum ada rincian"
            description="Susun rincian agar nilai kontrak dan termin bisa dihitung."
            action={
              canEdit ? (
                <div className="flex flex-wrap justify-center gap-2">
                  <Button size="sm" icon={<Package className="size-3.5" />} onClick={() => setPicking(true)}>
                    Tambah paket
                  </Button>
                  <Button size="sm" variant="secondary" icon={<Plus className="size-3.5" />} onClick={() => add("Item")}>
                    Tambah item
                  </Button>
                </div>
              ) : undefined
            }
          />
        ) : (
          <table className="w-full border-collapse text-[13.5px] max-md:block">
            <caption className="sr-only">Baris rincian</caption>
            <thead className="max-md:sr-only">
              <tr className="border-b border-border-light text-left text-[12.5px] text-muted-foreground">
                <th scope="col" className="px-5 py-2 font-medium">Uraian</th>
                <th scope="col" className="px-3 py-2 font-medium">Acara</th>
                <th scope="col" className="px-3 py-2 font-medium">Jumlah × Harga satuan</th>
                <th scope="col" className="px-3 py-2 text-right font-medium">Nominal</th>
                <th scope="col" className="w-12 px-3 py-2">
                  <span className="sr-only">Aksi</span>
                </th>
              </tr>
            </thead>
            {(["main", "bonus", "discount"] as RowGroup[]).map((group) => (
              <GroupBody
                key={group}
                group={group}
                rows={rows}
                index={index}
                errors={errors}
                canEdit={canEdit && !saving}
                expanded={expanded}
                eventName={eventName}
                onToggle={(key) =>
                  setExpanded((s) => {
                    const next = new Set(s);
                    if (next.has(key)) next.delete(key);
                    else next.add(key);
                    return next;
                  })
                }
                onChange={update}
                onMove={move}
                onRemove={remove}
                onDetail={setDetailKey}
                onAddComponent={addComponent}
              />
            ))}
            {shown.totals && (
              <tfoot className="max-md:block">
                <TotalRow label="Subtotal" value={formatCurrency(shown.totals.subtotal)} />
                {shown.isPkp && <TotalRow label="PPN" value={formatCurrency(shown.totals.tax)} />}
                <TotalRow
                  label="Nilai kontrak"
                  strong
                  value={computing ? "Menghitung…" : formatCurrency(shown.totals.total)}
                  note={previewFailed && !computing ? "belum diperbarui" : undefined}
                />
              </tfoot>
            )}
          </table>
        )}

        {dirty && canEdit && (
          <div className="sticky bottom-0 z-10 flex flex-wrap items-center justify-between gap-3 rounded-b-lg border-t border-border bg-card px-5 py-3 shadow-[0_-4px_12px_-8px_var(--color-foreground)]">
            <p className="text-[13px] text-foreground">
              Perubahan belum disimpan
              {shownTotal !== undefined && (
                <>
                  {" · "}Nilai kontrak{" "}
                  <span className="font-semibold tabular-nums">{computing ? "Menghitung…" : formatCurrency(shownTotal)}</span>
                  {!computing && delta !== 0 && (
                    <span className="tabular-nums text-muted-foreground"> ({delta > 0 ? "+" : ""}{delta > 0 ? formatCurrency(delta) : negative(delta)})</span>
                  )}
                </>
              )}
            </p>
            <div className="flex gap-2">
              <Button size="sm" variant="secondary" onClick={discard} disabled={saving}>
                Batalkan perubahan
              </Button>
              <Button size="sm" onClick={() => void handleSave()} disabled={saving || computing}>
                {saving ? "Menyimpan..." : "Simpan rincian"}
              </Button>
            </div>
          </div>
        )}

        <p role="status" aria-live="polite" className="sr-only">
          {status}
        </p>
      </section>

      <CostSummaryPanel
        id="ringkasan-biaya"
        rincian={shown}
        computing={computing}
        stale={previewFailed}
        missingCostHref={missingCostHref}
        className="xl:sticky xl:top-4"
      >
        {summaryExtra}
      </CostSummaryPanel>

      {picking && (
        <Suspense fallback={null}>
          <PackagePickerModal search={searchPackages} onPick={addPackage} onClose={() => setPicking(false)} />
        </Suspense>
      )}
      {detailRow && (
        <Suspense fallback={null}>
          <LineDetailDrawer
            row={detailRow}
            events={events}
            suppliers={suppliers}
            categories={categories}
            onClose={() => setDetailKey(null)}
            onApply={(changes) => {
              update(detailRow.key, changes);
              setDetailKey(null);
            }}
          />
        </Suspense>
      )}
    </div>
  );
}

/* ---------------------------------------------------------------- table body -------------------------------------- */

interface GroupBodyProps {
  group: RowGroup;
  rows: EditorRow[];
  index: ComputedIndex;
  errors: FieldErrors;
  canEdit: boolean;
  expanded: Set<string>;
  eventName: (id: string) => string;
  onToggle: (key: string) => void;
  onChange: (key: string, changes: Partial<EditorRow>) => void;
  onMove: (row: EditorRow, direction: -1 | 1) => void;
  onRemove: (row: EditorRow) => void;
  onDetail: (key: string) => void;
  onAddComponent: (pkgKey: string) => void;
}

const GROUP_TITLE: Record<RowGroup, string | null> = { main: null, bonus: "Bonus", discount: "Potongan" };

function GroupBody(props: GroupBodyProps) {
  const { group, rows, index, expanded } = props;
  const top = topLevel(rows, group);
  if (top.length === 0) return null;
  const title = GROUP_TITLE[group];
  return (
    <tbody className="max-md:block">
      {title && (
        <tr className="max-md:block">
          <th scope="rowgroup" colSpan={5} className="bg-muted/60 px-5 py-1.5 text-left text-[12.5px] font-semibold text-foreground max-md:block">
            {title}
          </th>
        </tr>
      )}
      {top.map((row, i) => {
        if (row.kind !== "Package") {
          return <LineRow key={row.key} {...props} row={row} position={i} siblings={top.length} />;
        }
        const savedComponents = rows.filter((r) => r.kind === "Component" && r.parentKey === row.key);
        // A package added in this session shows the isi the catalog will give it (from the preview).
        const catalogComponents = row.id ? [] : (index.componentsOf.get(row.key) ?? []).filter((l) => !l.id);
        const componentCount = savedComponents.length + catalogComponents.length;
        const open = expanded.has(row.key);
        const extra = index.extraOf.get(row.key);
        return [
          <LineRow
            key={row.key}
            {...props}
            row={row}
            position={i}
            siblings={top.length}
            componentCount={componentCount}
            open={open}
          />,
          ...(open
            ? [
                ...savedComponents.map((c, ci) => (
                  <LineRow key={c.key} {...props} row={c} position={ci} siblings={savedComponents.length} />
                )),
                ...catalogComponents.map((c, ci) => <CatalogComponentRow key={`${row.key}-c${ci}`} line={c} />),
              ]
            : []),
          extra ? <ExtraGuestsRow key={`${row.key}-extra`} line={extra} packageName={row.description} /> : null,
        ];
      })}
    </tbody>
  );
}

const ROW = "border-b border-border-light align-top max-md:flex max-md:flex-wrap max-md:items-start max-md:gap-x-3 max-md:px-4 max-md:py-3";
const CELL = "px-3 py-2.5 max-md:p-0";

function LineRow({
  row,
  index,
  errors,
  canEdit,
  eventName,
  onToggle,
  onChange,
  onMove,
  onRemove,
  onDetail,
  onAddComponent,
  position,
  siblings,
  componentCount = 0,
  open = false,
}: GroupBodyProps & { row: EditorRow; position: number; siblings: number; componentCount?: number; open?: boolean }) {
  const line: RincianLine | undefined = index.byKey.get(row.key);
  const rowErrorsOf = errors[row.key] ?? {};
  const name = row.description || LINE_KIND_LABELS[row.kind];
  const isComponent = row.kind === "Component";

  const amountCell = (() => {
    if (isComponent) return <span className="text-muted-foreground">Termasuk paket</span>;
    if (row.kind === "Item" && row.isBonus) return <Badge tone="brand" dot={false}>Bonus</Badge>;
    if (row.kind === "Discount") {
      return canEdit ? (
        <span className="flex flex-col items-end gap-1">
          <CurrencyInput
            value={row.amount}
            onChange={(v) => onChange(row.key, { amount: v })}
            className="w-36 text-right"
            aria-invalid={Boolean(rowErrorsOf.amount)}
            data-field="amount"
            aria-label={`Nominal potongan ${name} (rupiah)`}
          />
          <FieldError message={rowErrorsOf.amount} />
        </span>
      ) : (
        <span className="tabular-nums">{negative(line?.amount ?? row.amount)}</span>
      );
    }
    const amount = line?.amount;
    return amount === undefined ? <span className="text-muted-foreground">Menghitung…</span> : <span className="tabular-nums font-medium">{formatCurrency(amount)}</span>;
  })();

  return (
    <tr data-line={row.key} className={cn(ROW, isComponent && "bg-muted/30")}>
      <td className={cn(CELL, "pl-5 md:min-w-[11rem] max-md:w-full max-md:pl-0", isComponent && "pl-12 max-md:pl-4")}>
        <div className="flex items-start gap-1.5">
          {row.kind === "Package" && (
            <button
              type="button"
              onClick={() => onToggle(row.key)}
              aria-expanded={open}
              aria-label={`${open ? "Sembunyikan" : "Tampilkan"} ${componentCount} isi ${name}`}
              className="mt-1.5 inline-flex size-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              {open ? <ChevronDown aria-hidden className="size-4" /> : <ChevronRight aria-hidden className="size-4" />}
            </button>
          )}
          <div className="min-w-0 flex-1">
            {canEdit && row.kind !== "Package" ? (
              <Input
                value={row.description}
                onChange={(e) => onChange(row.key, { description: e.target.value })}
                onDoubleClick={() => onDetail(row.key)}
                placeholder={row.kind === "Discount" ? "Mis. Potongan harga paket" : isComponent ? "Isi paket" : "Mis. Souvenir kipas"}
                aria-label={`Uraian ${LINE_KIND_LABELS[row.kind].toLowerCase()}`}
                aria-invalid={Boolean(rowErrorsOf.description)}
                data-field="description"
                className="h-9"
              />
            ) : (
              <p className={cn("pt-1.5 text-foreground", row.kind === "Package" && "font-semibold")}>{name}</p>
            )}
            <FieldError message={rowErrorsOf.description} />
            {row.kind === "Package" && (
              <p className="mt-0.5 flex flex-wrap items-center gap-x-2 text-[12.5px] text-muted-foreground">
                {line?.packageRule && line.packageRule.includedGuests > 0 && <span>Termasuk {line.packageRule.includedGuests.toLocaleString("id-ID")} tamu</span>}
                <span>{componentCount} isi paket</span>
                {!row.id && <Badge tone="info" dot={false}>Baru dari katalog</Badge>}
              </p>
            )}
          </div>
        </div>
      </td>
      <td className={cn(CELL, "text-muted-foreground max-md:order-3 max-md:text-[12.5px]", !eventName(row.eventId) && "max-md:hidden")}>{eventName(row.eventId) || (isComponent ? "" : "—")}</td>
      <td className={cn(CELL, "max-md:order-4")}>
        {row.kind === "Item" &&
          (canEdit ? (
            <div className="flex flex-wrap items-center gap-1.5">
              <Input
                type="number"
                inputMode="numeric"
                min={1}
                value={row.quantity || ""}
                onChange={(e) => onChange(row.key, { quantity: Math.max(0, Math.trunc(Number(e.target.value) || 0)) })}
                aria-label={`Jumlah ${name}`}
                aria-invalid={Boolean(rowErrorsOf.quantity)}
                data-field="quantity"
                className="h-9 w-16"
              />
              <Input
                value={row.unit}
                onChange={(e) => onChange(row.key, { unit: e.target.value })}
                placeholder="pcs"
                aria-label={`Satuan ${name}`}
                data-field="unit"
                className="h-9 w-20"
              />
              {!row.isBonus && (
                <>
                  <span aria-hidden className="text-muted-foreground">×</span>
                  <CurrencyInput
                    value={row.unitPrice}
                    onChange={(v) => onChange(row.key, { unitPrice: v })}
                    aria-label={`Harga satuan ${name} (rupiah)`}
                    aria-invalid={Boolean(rowErrorsOf.unitPrice)}
                    data-field="unitPrice"
                    className="h-9 w-32"
                  />
                </>
              )}
              <FieldError message={rowErrorsOf.quantity ?? rowErrorsOf.unitPrice} />
            </div>
          ) : (
            <span className="tabular-nums text-muted-foreground">
              {row.quantity.toLocaleString("id-ID")} {row.unit}
              {!row.isBonus && line?.unitPrice !== undefined ? ` × ${formatCurrency(line.unitPrice)}` : ""}
            </span>
          ))}
      </td>
      <td className={cn(CELL, "text-right whitespace-nowrap max-md:order-2 max-md:ml-auto")}>{amountCell}</td>
      <td className={cn(CELL, "pr-5 text-right max-md:order-5 max-md:pr-0")}>
        {canEdit && (
          <ActionMenu
            variant="ghost"
            label={`Aksi untuk ${name}`}
            items={[
              { label: "Detail baris", icon: <Info aria-hidden />, onSelect: () => onDetail(row.key) },
              { label: "Tambah isi paket", icon: <Plus aria-hidden />, onSelect: () => onAddComponent(row.key), hidden: row.kind !== "Package" || !row.id },
              { label: "Pindah ke atas", icon: <ArrowUp aria-hidden />, onSelect: () => onMove(row, -1), hidden: position === 0 },
              { label: "Pindah ke bawah", icon: <ArrowDown aria-hidden />, onSelect: () => onMove(row, 1), hidden: position >= siblings - 1 },
              { label: row.kind === "Package" ? "Hapus paket" : "Hapus baris", icon: <Trash2 aria-hidden />, tone: "danger", onSelect: () => onRemove(row) },
            ]}
          />
        )}
      </td>
    </tr>
  );
}

/** An isi a package added in this session gets from the catalog (read-only until the package is saved). */
function CatalogComponentRow({ line }: { line: RincianLine }) {
  return (
    <tr className={cn(ROW, "bg-muted/30")}>
      <td className={cn(CELL, "pl-12 text-foreground max-md:w-full max-md:pl-4")}>{line.description}</td>
      <td className={CELL} />
      <td className={CELL} />
      <td className={cn(CELL, "text-right whitespace-nowrap text-muted-foreground max-md:ml-auto")}>Termasuk paket</td>
      <td className={CELL} />
    </tr>
  );
}

/** "Tambahan tamu": computed from the acara's guests, never typed (DESAIN D5). */
function ExtraGuestsRow({ line, packageName }: { line: RincianLine; packageName: string }) {
  const info = line.extraGuests;
  return (
    <tr className={ROW}>
      <td className={cn(CELL, "pl-12 max-md:w-full max-md:pl-4")}>
        <p className="flex items-center gap-1 text-foreground">
          Tambahan tamu
          {info && <ExtraGuestsHelp info={info} packageName={packageName} />}
          {info?.noGuests && (
            <Badge tone="warning" dot={false}>
              Belum ada tamu
            </Badge>
          )}
        </p>
        {info && <p className="mt-0.5 text-[12.5px] text-muted-foreground">{extraGuestsFormula(info)}</p>}
      </td>
      <td className={cn(CELL, "text-muted-foreground max-md:order-3 max-md:text-[12.5px]")}>{info?.eventKindName ?? ""}</td>
      <td className={CELL} />
      <td className={cn(CELL, "text-right whitespace-nowrap tabular-nums max-md:order-2 max-md:ml-auto")}>
        {line.amount === undefined ? "" : formatCurrency(line.amount)}
      </td>
      <td className={CELL} />
    </tr>
  );
}

function TotalRow({ label, value, strong = false, note }: { label: string; value: string; strong?: boolean; note?: string }) {
  return (
    <tr className="max-md:flex max-md:justify-between max-md:px-4 max-md:py-1.5">
      <th scope="row" colSpan={3} className={cn("px-3 py-1.5 text-right font-normal text-muted-foreground max-md:p-0", strong && "font-semibold text-foreground")}>
        {label}
      </th>
      <td className={cn("px-3 py-1.5 text-right whitespace-nowrap tabular-nums max-md:p-0", strong && "text-[15px] font-semibold text-foreground")}>
        {value}
        {note && <span className="ml-1.5 text-[12px] font-normal text-warning-text">({note})</span>}
      </td>
      <td className="max-md:hidden" />
    </tr>
  );
}

function FieldError({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <span className="mt-1 flex items-start gap-1 text-xs font-medium text-danger-text">
      <CircleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
      {message}
    </span>
  );
}

/** "Lihat perbedaan": what the newer version has that the edit does not, and the other way round. */
function ConflictDiff({ mine, mineTotal, theirs }: { mine: EditorRow[]; mineTotal: number | undefined; theirs: Rincian }) {
  const theirNames = theirs.lines.filter((l) => l.kind !== "Component" && l.kind !== "ExtraGuests").map((l) => l.description);
  const myNames = mine.filter((r) => r.kind !== "Component").map((r) => r.description);
  const onlyTheirs = theirNames.filter((n) => !myNames.includes(n));
  const onlyMine = myNames.filter((n) => !theirNames.includes(n));
  return (
    <div className="mt-3 flex flex-col gap-2 rounded-md bg-card px-3.5 py-3 text-foreground">
      {theirs.totals && mineTotal !== undefined && (
        <p>
          Nilai kontrak versi terbaru {formatCurrency(theirs.totals.total)}, perubahan Anda {formatCurrency(mineTotal)}.
        </p>
      )}
      <p>{onlyTheirs.length > 0 ? `Hanya di versi terbaru: ${onlyTheirs.join(", ")}.` : "Versi terbaru tidak punya baris yang tidak ada di perubahan Anda."}</p>
      <p>{onlyMine.length > 0 ? `Hanya di perubahan Anda: ${onlyMine.join(", ")}.` : "Perubahan Anda tidak menambah baris baru."}</p>
      <p className="text-muted-foreground">Muat versi terbaru, lalu ulangi perubahan Anda yang masih perlu.</p>
    </div>
  );
}
