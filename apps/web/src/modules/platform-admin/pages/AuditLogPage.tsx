import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Download } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Input, Select } from "@/shared/components/ui/Input";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { toast } from "@/shared/stores/useToastStore";
import { formatDateTime } from "@/shared/lib/formatters";
import { downloadFile } from "@/shared/lib/download";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { API } from "@/shared/services/api-endpoints";
import type { AuditEntry } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";

const ACTOR_LABEL: Record<string, string> = { platform_admin: "Tim ElProof", staff: "Staf WO", client: "Klien" };

function Actor({ e }: { e: AuditEntry }) {
  return (
    <>
      <span className="block">{ACTOR_LABEL[e.actorType] ?? e.actorType}{e.actorRole ? `, ${e.actorRole}` : ""}</span>
      <span className="block text-[12px] text-text-secondary">ID {e.actorId}</span>
    </>
  );
}

function Outcome({ e }: { e: AuditEntry }) {
  return <Badge tone={e.succeeded ? "success" : "danger"}>{e.succeeded ? `Berhasil ${e.statusCode}` : `Gagal ${e.statusCode}`}</Badge>;
}

function TenantLink({ id }: { id: number | null }) {
  if (id === null) return <span className="text-text-secondary">-</span>;
  return (
    <Link to={ROUTE_PATHS.platformTenantDetail(String(id), "aktivitas")} className="hover:underline">
      Tenant {id}
    </Link>
  );
}

const LIST_FILTERS = { actorType: "", outcome: "", from: "", to: "", action: "", tenantId: "" };

/** The exact action label: typed, then applied on Enter or when leaving the field (not on every keystroke). */
function ActionFilter({ value, onCommit }: { value: string; onCommit: (value: string) => void }) {
  const [draft, setDraft] = useState(value);
  useEffect(() => setDraft(value), [value]);
  const commit = () => {
    if (draft.trim() !== value) onCommit(draft.trim());
  };
  return (
    <Input
      className="w-52"
      aria-label="Aksi (nama persis)"
      placeholder="Aksi, cth. activate-subscription"
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") commit();
      }}
    />
  );
}

export default function AuditLogPage() {
  const entries = useConsoleStore((s) => s.auditPage);
  const meta = useConsoleStore((s) => s.auditMeta);
  const fetchAuditPage = useConsoleStore((s) => s.fetchAuditPage);

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back); ?tenantId= still narrows to one tenant.
  const list = useListState({ filters: LIST_FILTERS, searchParam: "q" });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("platform-audit-log");
  const [loadState, setLoadState] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [exporting, setExporting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoadState("loading");
    fetchAuditPage(list.params)
      .then(() => !cancelled && setLoadState("ready"))
      .catch(() => !cancelled && setLoadState("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchAuditPage, list.paramsKey, attempt]);

  async function handleExport() {
    setExporting(true);
    try {
      const f = list.filters;
      await downloadFile(
        API.console.auditLogExport,
        {
          q: list.q || undefined,
          actorType: f.actorType || undefined,
          outcome: f.outcome || undefined,
          action: f.action || undefined,
          tenantId: f.tenantId || undefined,
          from: f.from || undefined,
          to: f.to || undefined,
        },
        "audit-log.csv",
      );
      toast.success("Audit log diunduh. Pengunduhan ini tercatat di audit log.");
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Audit log belum bisa diunduh"));
    } finally {
      setExporting(false);
    }
  }

  const columns: Column<AuditEntry>[] = [
    { id: "time", header: "Waktu", sortKey: "createdAt", className: "whitespace-nowrap", cell: (e) => formatDateTime(e.occurredAt) },
    { id: "actor", header: "Pelaku", sortKey: "actor", cell: (e) => <Actor e={e} /> },
    {
      id: "action",
      header: "Aksi",
      sortKey: "action",
      primary: true,
      cell: (e) => (
        <>
          <span className="block font-medium text-text-primary">{e.action}</span>
          <span className="block max-w-xs truncate font-mono text-[11.5px] font-normal text-muted-foreground" title={e.path}>{e.path}</span>
        </>
      ),
    },
    {
      id: "detail",
      header: "Rincian",
      cell: (e) => (
        <div className="flex flex-col items-start gap-1">
          <Outcome e={e} />
          <span className="text-[12.5px]"><TenantLink id={e.tenantId} /></span>
          <span className="font-mono text-[12px] text-muted-foreground">{e.ip}</span>
        </div>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Audit Log</h1>
          <p className="mt-1 max-w-2xl text-[13px] text-text-secondary">
            Jejak setiap perubahan yang dilakukan tim ElProof dan anggota tim WO: siapa, kapan, dan hasilnya. Isi data yang diubah tidak disimpan di sini.
          </p>
        </div>
        <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={() => void handleExport()} disabled={exporting}>
          {exporting ? "Menyiapkan..." : "Unduh CSV"}
        </Button>
      </div>

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari aksi atau alamat..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Select className="w-44" aria-label="Pelaku" value={list.filters.actorType || "Semua"} onChange={(e) => list.setFilter("actorType", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua pelaku</option>
          <option value="platform_admin">Tim ElProof</option>
          <option value="staff">Anggota tim WO</option>
          <option value="client">Klien</option>
        </Select>
        <Select className="w-40" aria-label="Hasil" value={list.filters.outcome || "Semua"} onChange={(e) => list.setFilter("outcome", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua hasil</option>
          <option value="ok">Berhasil</option>
          <option value="failed">Gagal</option>
        </Select>
        <ActionFilter value={list.filters.action} onCommit={(v) => list.setFilter("action", v)} />
        <Input type="date" className="w-40" aria-label="Dari tanggal" value={list.filters.from} max={list.filters.to || undefined} onChange={(e) => list.setFilter("from", e.target.value)} />
        <Input type="date" className="w-40" aria-label="Sampai tanggal" value={list.filters.to} min={list.filters.from || undefined} onChange={(e) => list.setFilter("to", e.target.value)} />
        {list.filters.tenantId ? (
          <Button variant="secondary" size="sm" onClick={() => list.setFilter("tenantId", "")}>
            Hanya Tenant {list.filters.tenantId}. Tampilkan semua
          </Button>
        ) : null}
      </ListToolbar>

      <DataTable
        label="Daftar audit log"
        columns={columns}
        rows={entries}
        rowId={(e) => String(e.id)}
        status={loadState}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada catatan",
          description: "Setiap perubahan oleh tim ElProof atau anggota tim WO akan tercatat di sini: siapa, kapan, dan hasilnya.",
        }}
        density={density}
      />
    </div>
  );
}
