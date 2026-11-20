import { lazy, Suspense, useEffect, useMemo, useState, type ReactNode } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { CornerDownRight, ExternalLink, FolderOpen, ListFilter, Plus } from "lucide-react";
import { Avatar } from "@/shared/components/ui/Avatar";
import { Button } from "@/shared/components/ui/Button";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Select } from "@/shared/components/ui/Input";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { useCan } from "@/shared/stores/usePermissionStore";
import { cn } from "@/shared/lib/cn";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { QuotationStatusBadge } from "@/modules/offers/components/QuotationStatusBadge";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { expiresSoon } from "@/modules/offers/lib/display";
import type { Quotation, QuotationSummary, QuotationTab } from "@/modules/offers/types";

const QuotationCreateModal = lazy(() => import("@/modules/offers/components/QuotationCreateModal"));

type Filters = { tab: string; expiringSoon: string; picStaffId: string };
const FILTERS: Filters = { tab: "", expiringSoon: "", picStaffId: "" };

const TABS: Array<{ value: QuotationTab; label: string; count: keyof QuotationSummary }> = [
  { value: "", label: "Semua", count: "all" },
  { value: "Draft", label: "Draf", count: "draft" },
  { value: "Expired", label: "Kedaluwarsa", count: "expired" },
  { value: "Accepted", label: "Diterima", count: "accepted" },
  { value: "Rejected", label: "Ditolak", count: "rejected" },
  { value: "Superseded", label: "Digantikan", count: "superseded" },
];

/** A table row: a quotation, or one of its alternatives right under it. */
type Row = Quotation & { childOf: string };

/** Penawaran (DESAIN §6.1): every quotation with its alternatives, filtered, sorted and paged on the server. */
export default function QuotationListPage() {
  const navigate = useNavigate();
  const canManage = useCan("quotation.manage");
  const rows = useQuotationStore((s) => s.rows);
  const meta = useQuotationStore((s) => s.meta);
  const summary = useQuotationStore((s) => s.summary);
  const fetchQuotations = useQuotationStore((s) => s.fetchQuotations);
  const fetchSummary = useQuotationStore((s) => s.fetchSummary);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);
  const isWide = useMediaQuery("(min-width: 768px)");
  const [search, setSearch] = useSearchParams();

  const list = useListState<Filters>({ filters: FILTERS });
  const { density, setDensity } = useListDensity();
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [filtersOpen, setFiltersOpen] = useState(false);

  // "Baru ▾ › Penawaran" lands here with ?baru=1: the dialog opens over the list.
  const createOpen = search.get("baru") === "1" && canManage;
  function closeCreate() {
    const next = new URLSearchParams(search);
    next.delete("baru");
    setSearch(next, { replace: true });
  }
  function openCreate() {
    const next = new URLSearchParams(search);
    next.set("baru", "1");
    setSearch(next);
  }

  useEffect(() => {
    void fetchStaffSummaries().catch(() => undefined);
  }, [fetchStaffSummaries]);

  useEffect(() => {
    void fetchSummary().catch(() => undefined);
  }, [fetchSummary, attempt]);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchQuotations(list.params)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchQuotations, list.paramsKey, attempt]);

  const tableRows = useMemo<Row[]>(
    () => rows.flatMap((q) => [{ ...q, childOf: "" }, ...q.alternatives.map((a) => ({ ...a, childOf: q.id }))]),
    [rows],
  );

  const extraFilters = [list.filters.picStaffId, list.filters.expiringSoon].filter(Boolean).length;

  const columns: Column<Row>[] = [
    {
      id: "number",
      header: "Nomor",
      sortKey: "number",
      primary: true,
      // The number is how a quotation is named on the phone and in the PDF: it is never cut.
      className: "whitespace-nowrap",
      cell: (q) => (
        <span className={cn("flex min-w-0 items-start gap-1.5", q.childOf && "pl-4")}>
          {q.childOf && <CornerDownRight aria-hidden className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />}
          <span className="block min-w-0 leading-tight">
            <Link
              to={ROUTE_PATHS.quotationDetail(q.id)}
              className="block whitespace-nowrap font-semibold tabular-nums underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
            >
              {q.number}
            </Link>
            {/* The couple has its own column; under the number only what relates it to other quotations. */}
            {(q.childOf || altCount(rows, q.id) > 0) && (
              <span className="block truncate text-[12.5px] font-normal text-muted-foreground">
                {q.childOf ? "Alternatif" : `${altCount(rows, q.id)} alternatif`}
              </span>
            )}
          </span>
        </span>
      ),
    },
    { id: "coupleNames", header: "Pasangan", sortKey: "coupleNames", card: true, className: "whitespace-nowrap", cell: (q) => q.coupleNames },
    {
      id: "mainEventDate",
      header: "Acara utama",
      sortKey: "mainEventDate",
      className: "whitespace-nowrap",
      cell: (q) => (q.mainEventDate ? formatDate(q.mainEventDate) : <span className="text-muted-foreground">Belum ada tanggal</span>),
    },
    {
      id: "contractTotal",
      header: "Nilai kontrak",
      sortKey: "contractTotal",
      align: "right",
      card: true,
      className: "whitespace-nowrap tabular-nums",
      cell: (q) => (q.revisionCount > 0 || q.contractTotal > 0 ? formatCurrency(q.contractTotal) : <span className="text-muted-foreground">Belum disusun</span>),
    },
    {
      id: "status",
      header: "Status",
      card: true,
      className: "whitespace-nowrap",
      cell: (q) => (
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap md:flex-col md:items-start md:gap-0.5">
          <QuotationStatusBadge status={q.displayStatus} />
          {q.revisionCount > 0 && <span className="text-[12.5px] text-muted-foreground">Revisi {q.revisionCount}</span>}
        </span>
      ),
    },
    {
      id: "validUntil",
      header: "Berlaku sampai",
      sortKey: "validUntil",
      card: true,
      className: "whitespace-nowrap",
      cell: (q) => <ValidityCell q={q} />,
    },
    {
      id: "pic",
      header: "PIC",
      // Initials keep the column narrow; the name is read out and shown on hover.
      cell: (q) =>
        q.picName ? (
          <span title={q.picName} className="inline-flex">
            <Avatar name={q.picName} />
            <span className="sr-only">{q.picName}</span>
          </span>
        ) : (
          <span className="text-muted-foreground">-</span>
        ),
    },
  ];

  const pickers = (
    <>
      <Select
        className="w-full md:w-48"
        aria-label="PIC"
        value={list.filters.picStaffId || "Semua"}
        onChange={(e) => list.setFilter("picStaffId", e.target.value === "Semua" ? "" : e.target.value)}
      >
        <option value="Semua">Semua PIC</option>
        {staff.map((m) => (
          <option key={m.id} value={m.id}>
            {m.name}
          </option>
        ))}
      </Select>
      <label className="flex h-9 items-center gap-2 rounded-md border border-border px-3 text-[13px] text-text-secondary">
        <input type="checkbox" checked={list.filters.expiringSoon === "true"} onChange={(e) => list.setFilter("expiringSoon", e.target.checked ? "true" : "")} />
        Segera kedaluwarsa (≤ 7 hari)
        {summary && summary.expiringSoon > 0 && <span className="tabular-nums text-muted-foreground">{summary.expiringSoon}</span>}
      </label>
    </>
  );

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Penawaran"
        className="mb-1"
        description="Susun penawaran untuk calon klien, bagikan sebagai PDF, lalu jadikan proyek setelah diterima."
        actions={
          canManage && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
              Penawaran baru
            </Button>
          )
        }
      />

      <div role="group" aria-label="Saring menurut status" className="flex gap-1 overflow-x-auto border-b border-border">
        {TABS.map((tab) => (
          <StatusTab key={tab.value || "all"} active={list.filters.tab === tab.value} onClick={() => list.setFilter("tab", tab.value)}>
            {tab.label}
            {summary && tab.value !== "" && summary[tab.count] > 0 && <span className="ml-1.5 tabular-nums text-muted-foreground">{summary[tab.count]}</span>}
          </StatusTab>
        ))}
      </div>

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nomor, pasangan, atau WhatsApp..." }}
        density={isWide ? { value: density, onChange: setDensity } : undefined}
      >
        {isWide ? (
          pickers
        ) : (
          <Button variant="secondary" size="sm" icon={<ListFilter className="h-3.5 w-3.5" />} onClick={() => setFiltersOpen(true)} aria-haspopup="dialog">
            {extraFilters > 0 ? `Saring (${extraFilters})` : "Saring"}
          </Button>
        )}
      </ListToolbar>

      <DataTable
        label="Daftar penawaran"
        columns={columns}
        rows={tableRows}
        rowId={(q) => q.id}
        rowHref={(q) => ROUTE_PATHS.quotationDetail(q.id)}
        rowLabel={(q) => `${q.number} ${q.coupleNames}`}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        density={density}
        empty={{
          title: "Belum ada penawaran",
          description: "Susun penawaran untuk calon klien dan bagikan sebagai PDF.",
          action: canManage ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
              Penawaran baru
            </Button>
          ) : undefined,
        }}
        actions={(q) => (
          <RowActions
            rowLabel={q.number}
            items={[
              { label: "Buka penawaran", icon: <ExternalLink className="h-4 w-4" />, onSelect: () => navigate(ROUTE_PATHS.quotationDetail(q.id)) },
              {
                label: "Buka proyek",
                icon: <FolderOpen className="h-4 w-4" />,
                onSelect: () => navigate(ROUTE_PATHS.projectDetail(q.convertedProjectId)),
                hidden: !q.convertedProjectId || q.projectDeleted,
              },
            ]}
          />
        )}
      />

      {!isWide && (
        <Drawer
          open={filtersOpen}
          onClose={() => setFiltersOpen(false)}
          title="Saring penawaran"
          footer={
            <>
              {extraFilters > 0 && (
                <Button
                  variant="ghost"
                  onClick={() => {
                    const next = new URLSearchParams(list.queryString);
                    for (const key of ["picStaffId", "expiringSoon", "page"]) next.delete(key);
                    list.applyQueryString(next.toString());
                  }}
                >
                  Hapus saringan
                </Button>
              )}
              <Button onClick={() => setFiltersOpen(false)}>Tampilkan hasil</Button>
            </>
          }
        >
          <div className="flex flex-col gap-4">{pickers}</div>
        </Drawer>
      )}

      {createOpen && (
        <Suspense fallback={null}>
          <QuotationCreateModal open={createOpen} onClose={closeCreate} />
        </Suspense>
      )}
    </div>
  );
}

function altCount(rows: Array<{ id: string; alternatives: Quotation[] }>, id: string): number {
  return rows.find((r) => r.id === id)?.alternatives.length ?? 0;
}

/** "30 Nov 2026 · 11 hari lagi", in the warning colour only within a week; a past date says so in words. */
function ValidityCell({ q }: { q: Quotation }) {
  if (q.convertedProjectId && !q.projectDeleted) {
    return (
      <Link
        to={ROUTE_PATHS.projectDetail(q.convertedProjectId)}
        className="font-medium text-primary underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
      >
        Sudah jadi proyek
      </Link>
    );
  }
  if (!q.validUntil) return <span className="text-muted-foreground">-</span>;
  if (q.displayStatus === "Expired") return <span className="font-medium whitespace-nowrap text-warning-text">{formatDate(q.validUntil)} · lewat</span>;
  if (q.displayStatus !== "Draft") return <span className="whitespace-nowrap text-muted-foreground">{formatDate(q.validUntil)}</span>;
  const soon = expiresSoon(q);
  return (
    <span className={cn("block whitespace-nowrap tabular-nums leading-tight", soon && "font-medium text-warning-text")}>
      {formatDate(q.validUntil)}
      <span className={cn("block text-[12.5px]", !soon && "text-muted-foreground")}>{q.daysLeft === 0 ? "hari ini" : `${q.daysLeft} hari lagi`}</span>
    </span>
  );
}

function StatusTab({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        "-mb-px shrink-0 border-b-2 px-2.5 py-2.5 text-[13.5px] font-medium transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
        active ? "border-primary font-semibold text-primary" : "border-transparent text-text-secondary hover:text-text-primary",
      )}
    >
      {children}
    </button>
  );
}
