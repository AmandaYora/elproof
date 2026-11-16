import { useEffect, useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Archive, ArchiveRestore, CalendarDays, ListFilter, Plus, Wallet } from "lucide-react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { PROJECT_STATUS_LABELS } from "@/shared/lib/labels";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Input, Select } from "@/shared/components/ui/Input";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { toast } from "@/shared/stores/useToastStore";
import { cn } from "@/shared/lib/cn";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { PROJECT_STATUS_OPTIONS } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectListItem } from "@/modules/projects/types";
import { daysUntil } from "@/modules/projects/lib/dates";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { formatCurrency, formatDate, formatDayCountdown } from "@/shared/lib/formatters";

// The status is the list's main filter, so it is a row of tabs above the table rather than a dropdown (RISET §7.1).
const STATUS_TABS: Array<{ value: string; label: string }> = [
  { value: "", label: "Semua" },
  ...PROJECT_STATUS_OPTIONS.map((s) => ({ value: s, label: PROJECT_STATUS_LABELS.staff[s] })),
];

type Filters = { status: string; archived: string; picStaffId: string; eventFrom: string; eventTo: string };

export default function ProjectListPage() {
  // Wedding Planner ("Staff" role) never creates a project — only Owner/Admin
  // do, then assign it to a planner (see PLAN.md's RBAC section). The
  // backend already rejects POST /projects for this role; hiding the button
  // too avoids a confusing 403 after filling out the whole form.
  const canCreate = useCan("project.create");
  const canArchive = useCan("project.archive");
  const projects = useProjectStore((s) => s.projectPage);
  const meta = useProjectStore((s) => s.projectPageMeta);
  const fetchProjectPage = useProjectStore((s) => s.fetchProjectPage);
  const setProjectArchived = useProjectStore((s) => s.setProjectArchived);
  const staffSummaries = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);
  const navigate = useNavigate();
  const isWide = useMediaQuery("(min-width: 768px)");

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState<Filters>({ filters: { status: "", archived: "", picStaffId: "", eventFrom: "", eventTo: "" } });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("projects");
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [filtersOpen, setFiltersOpen] = useState(false);

  // The PIC filter needs the names of the team: a small list, loaded once.
  useEffect(() => {
    void fetchStaffSummaries().catch(() => undefined);
  }, [fetchStaffSummaries]);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchProjectPage(list.params)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchProjectPage, list.paramsKey, attempt]);

  async function handleArchive(p: ProjectListItem, archived: boolean) {
    try {
      await setProjectArchived(p.id, archived);
      toast.success(archived ? "Proyek diarsipkan" : "Proyek dipulihkan dari arsip");
      setAttempt((n) => n + 1);
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Status arsip proyek belum bisa diubah. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  // Filters other than the status tabs and the search: what the phone's "Saring (n)" button counts.
  const extraFilters = [list.filters.picStaffId, list.filters.eventFrom || list.filters.eventTo, list.filters.archived].filter(Boolean).length;

  const columns: Column<ProjectListItem>[] = [
    {
      id: "name",
      header: "Proyek",
      sortKey: "name",
      primary: true,
      truncate: true,
      className: "w-[30%] min-w-56",
      cell: (p) => {
        const couple = [p.brideName, p.groomName].filter(Boolean).join(" & ");
        return (
          <span className="block min-w-0 leading-tight" title={couple ? `${p.name} · ${couple}` : p.name}>
            <span className="block truncate font-semibold">{p.name}</span>
            {couple && <span className="block truncate text-[12.5px] font-normal text-muted-foreground">{couple}</span>}
          </span>
        );
      },
    },
    {
      id: "eventDate",
      header: "Tanggal acara",
      sortKey: "eventDate",
      card: true,
      className: "whitespace-nowrap",
      cell: (p) => {
        const open = p.status !== "Completed" && p.status !== "Cancelled";
        return (
          <span className="whitespace-nowrap tabular-nums">
            {formatDate(p.eventDate)}
            {open && <span className="ml-1.5 font-semibold text-highlight-text">{formatDayCountdown(daysUntil(p.eventDate))}</span>}
          </span>
        );
      },
    },
    {
      id: "status",
      header: "Status",
      sortKey: "status",
      card: true,
      className: "whitespace-nowrap",
      cell: (p) => (
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap">
          <ProjectStatusBadge status={p.status} />
          {p.isArchived && <Badge tone="neutral">Diarsipkan</Badge>}
        </span>
      ),
    },
    { id: "contractValue", header: "Nilai kontrak", sortKey: "contractValue", align: "right", cell: (p) => formatCurrency(p.contractValue) },
    {
      id: "remaining",
      header: "Sisa tagihan",
      sortKey: "remaining",
      align: "right",
      card: true,
      cell: (p) => (p.remaining > 0 ? formatCurrency(p.remaining) : p.contractValue > 0 ? "Lunas" : "-"),
    },
    {
      id: "nextDueDate",
      header: "Termin berikut",
      sortKey: "nextDueDate",
      className: "whitespace-nowrap",
      cell: (p) => {
        if (!p.nextDueDate) return <span className="text-muted-foreground">-</span>;
        // Past due: the red date also says so in words (never color alone).
        return daysUntil(p.nextDueDate) < 0 ? (
          <span className="font-medium whitespace-nowrap text-danger-text">{formatDate(p.nextDueDate)} · terlambat</span>
        ) : (
          formatDate(p.nextDueDate)
        );
      },
    },
    {
      id: "progress",
      header: "Progres",
      align: "right",
      cell: (p) => (p.progress ? `${p.progress.overallPercent}%` : <span className="text-muted-foreground">-</span>),
    },
  ];

  const pickers = (
    <>
      <Select className="w-full md:w-48" aria-label="PIC" value={list.filters.picStaffId || "Semua"} onChange={(e) => list.setFilter("picStaffId", e.target.value === "Semua" ? "" : e.target.value)}>
        <option value="Semua">Semua PIC</option>
        {staffSummaries.map((m) => (
          <option key={m.id} value={m.id}>
            {m.name}
          </option>
        ))}
      </Select>
      <label className="flex items-center justify-between gap-2 text-[13px] text-muted-foreground">
        Acara dari
        <Input type="date" className="w-40" value={list.filters.eventFrom} max={list.filters.eventTo || undefined} onChange={(e) => list.setFilter("eventFrom", e.target.value)} />
      </label>
      <label className="flex items-center justify-between gap-2 text-[13px] text-muted-foreground">
        sampai
        <Input type="date" className="w-40" value={list.filters.eventTo} min={list.filters.eventFrom || undefined} onChange={(e) => list.setFilter("eventTo", e.target.value)} />
      </label>
      <label className="flex h-9 items-center gap-2 rounded-md border border-border px-3 text-[13px] text-text-secondary">
        <input type="checkbox" checked={list.filters.archived === "true"} onChange={(e) => list.setFilter("archived", e.target.checked ? "true" : "")} />
        Diarsipkan
      </label>
    </>
  );

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Proyek"
        className="mb-1"
        actions={
          canCreate && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => navigate(ROUTE_PATHS.projectNew)}>
              Tambah proyek
            </Button>
          )
        }
      />

      <StatusTabs value={list.filters.status} onChange={(v) => list.setFilter("status", v)} />

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama proyek, pasangan, atau venue..." }}
        density={isWide ? { value: density, onChange: setDensity } : undefined}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
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
        label="Daftar proyek"
        columns={columns}
        rows={projects}
        rowId={(p) => p.id}
        rowHref={(p) => ROUTE_PATHS.projectDetail(p.id)}
        rowLabel={(p) => p.name}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada proyek",
          description: "Satu proyek adalah satu pernikahan yang Anda tangani: jadwal, vendor, termin pembayaran, dan klien ada di dalamnya.",
          action: canCreate ? (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => navigate(ROUTE_PATHS.projectNew)}>
              Tambah proyek
            </Button>
          ) : undefined,
        }}
        density={density}
        actions={(p) => (
          <RowActions
            rowLabel={p.name}
            items={[
              { label: "Buka keuangan", icon: <Wallet className="h-4 w-4" />, onSelect: () => navigate(ROUTE_PATHS.projectDetail(p.id, "keuangan/klien")) },
              { label: "Buka jadwal", icon: <CalendarDays className="h-4 w-4" />, onSelect: () => navigate(ROUTE_PATHS.projectDetail(p.id, "jadwal")) },
              {
                label: p.isArchived ? "Pulihkan dari arsip" : "Arsipkan",
                icon: p.isArchived ? <ArchiveRestore className="h-4 w-4" /> : <Archive className="h-4 w-4" />,
                onSelect: () => void handleArchive(p, !p.isArchived),
                hidden: !canArchive,
              },
            ]}
          />
        )}
      />

      {!isWide && (
        <Drawer
          open={filtersOpen}
          onClose={() => setFiltersOpen(false)}
          title="Saring proyek"
          footer={
            <>
              {extraFilters > 0 && (
                <Button
                  variant="ghost"
                  onClick={() => {
                    // One URL change (several setFilter calls in a row could overwrite each other).
                    const next = new URLSearchParams(list.queryString);
                    for (const key of ["picStaffId", "eventFrom", "eventTo", "archived", "page"]) next.delete(key);
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
    </div>
  );
}

/** Semua · Draf · Persiapan · Siap · Selesai · Dibatalkan: one press filters the table. */
function StatusTabs({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <div role="group" aria-label="Saring menurut status" className="flex gap-1 overflow-x-auto border-b border-border">
      {STATUS_TABS.map((tab) => (
        <StatusTab key={tab.value || "all"} active={value === tab.value} onClick={() => onChange(tab.value)}>
          {tab.label}
        </StatusTab>
      ))}
    </div>
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
