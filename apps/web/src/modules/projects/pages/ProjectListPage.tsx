import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus } from "lucide-react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { PROJECT_STATUS_LABELS } from "@/shared/lib/labels";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Input, Select } from "@/shared/components/ui/Input";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { PROJECT_STATUS_OPTIONS } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectListItem } from "@/modules/projects/types";
import { daysUntil } from "@/modules/projects/lib/dates";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

export default function ProjectListPage() {
  // Wedding Planner ("Staff" role) never creates a project — only Owner/Admin
  // do, then assign it to a planner (see PLAN.md's RBAC section). The
  // backend already rejects POST /projects for this role; hiding the button
  // too avoids a confusing 403 after filling out the whole form.
  const canCreate = useCan("project.create");
  const projects = useProjectStore((s) => s.projectPage);
  const meta = useProjectStore((s) => s.projectPageMeta);
  const fetchProjectPage = useProjectStore((s) => s.fetchProjectPage);
  const staffSummaries = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);
  const navigate = useNavigate();

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState({ filters: { status: "", archived: "", picStaffId: "", eventFrom: "", eventTo: "" } });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("projects");
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);

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

  const columns: Column<ProjectListItem>[] = [
    { id: "name", header: "Proyek", sortKey: "name", primary: true, className: "min-w-44", cell: (p) => <span className="font-semibold">{p.name}</span> },
    { id: "couple", header: "Pasangan", cell: (p) => [p.brideName, p.groomName].filter(Boolean).join(" & ") || "-" },
    { id: "eventDate", header: "Tanggal acara", sortKey: "eventDate", className: "whitespace-nowrap", cell: (p) => formatDate(p.eventDate) },
    {
      id: "status",
      header: "Status",
      sortKey: "status",
      cell: (p) => (
        <span className="inline-flex flex-wrap items-center gap-1.5">
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
      cell: (p) => (p.remaining > 0 ? formatCurrency(p.remaining) : p.contractValue > 0 ? "Lunas" : "-"),
    },
    {
      id: "nextDueDate",
      header: "Termin berikut",
      sortKey: "nextDueDate",
      cell: (p) => {
        if (!p.nextDueDate) return <span className="text-muted-foreground">-</span>;
        // Past due: the red date also says so in words (never color alone).
        return daysUntil(p.nextDueDate) < 0 ? (
          <span className="font-medium text-danger-text">{formatDate(p.nextDueDate)} · terlambat</span>
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

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Project</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola seluruh project pernikahan yang sedang dan pernah ditangani.</p>
        </div>
        {canCreate && (
          <Button icon={<Plus className="h-4 w-4" />} onClick={() => navigate(ROUTE_PATHS.projectNew)}>
            Tambah Project
          </Button>
        )}
      </div>

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama proyek, pasangan, atau venue..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Select className="w-48" aria-label="Status" value={list.filters.status || "Semua"} onChange={(e) => list.setFilter("status", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua status</option>
          {PROJECT_STATUS_OPTIONS.map((s) => (
            <option key={s} value={s}>
              {PROJECT_STATUS_LABELS.staff[s]}
            </option>
          ))}
        </Select>
        <Select className="w-52" aria-label="PIC" value={list.filters.picStaffId || "Semua"} onChange={(e) => list.setFilter("picStaffId", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua PIC</option>
          {staffSummaries.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </Select>
        <label className="flex items-center gap-2 text-[13px] text-muted-foreground">
          Acara dari
          <Input type="date" className="w-40" value={list.filters.eventFrom} max={list.filters.eventTo || undefined} onChange={(e) => list.setFilter("eventFrom", e.target.value)} />
        </label>
        <label className="flex items-center gap-2 text-[13px] text-muted-foreground">
          sampai
          <Input type="date" className="w-40" value={list.filters.eventTo} min={list.filters.eventFrom || undefined} onChange={(e) => list.setFilter("eventTo", e.target.value)} />
        </label>
        <label className="flex h-9 items-center gap-2 rounded-md border border-border px-3 text-[13px] text-text-secondary">
          <input
            type="checkbox"
            checked={list.filters.archived === "true"}
            onChange={(e) => list.setFilter("archived", e.target.checked ? "true" : "")}
          />
          Diarsipkan
        </label>
      </ListToolbar>

      <DataTable
        label="Daftar proyek"
        columns={columns}
        rows={projects}
        rowId={(p) => p.id}
        rowHref={(p) => ROUTE_PATHS.projectDetail(p.id)}
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
      />
    </div>
  );
}
