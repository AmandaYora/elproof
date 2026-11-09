import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { Download, ScrollText } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Field, Input, Select } from "@/shared/components/ui/Input";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { formatDateTime } from "@/shared/lib/formatters";
import { downloadFile } from "@/shared/lib/download";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { API } from "@/shared/services/api-endpoints";
import type { AuditEntry, AuditFilter } from "@/modules/platform-admin/data/console.types";
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

export default function AuditLogPage() {
  const entries = useConsoleStore((s) => s.auditPage);
  const meta = useConsoleStore((s) => s.auditMeta);
  const fetchAuditPage = useConsoleStore((s) => s.fetchAuditPage);
  const [params] = useSearchParams();
  const [q, setQ] = useState("");
  const debouncedQ = useDebouncedValue(q, 300);
  const [filter, setFilter] = useState<AuditFilter>({ tenantId: params.get("tenantId") ?? "" });
  const [page, setPage] = useState(1);
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const [exporting, setExporting] = useState(false);

  const effective: AuditFilter = { ...filter, q: debouncedQ };

  const latest = useRef(0);
  const load = useCallback(() => {
    const mine = ++latest.current;
    setFailed(false);
    fetchAuditPage(page, { ...filter, q: debouncedQ })
      .then(() => {
        if (mine === latest.current) setLoaded(true);
      })
      .catch(() => {
        if (mine === latest.current) setFailed(true);
      });
  }, [fetchAuditPage, page, filter, debouncedQ]);

  useEffect(() => {
    load();
  }, [load]);

  // Changing a filter returns to page 1 in the same update, so one fetch
  // follows (a separate effect resetting the page would fire a second one
  // whose answer could land first and be overwritten by the older page).
  function set<K extends keyof AuditFilter>(key: K, value: AuditFilter[K]) {
    setPage(1);
    setFilter((prev) => ({ ...prev, [key]: value }));
  }

  async function handleExport() {
    setExporting(true);
    try {
      await downloadFile(
        API.console.auditLogExport,
        { ...effective, actorType: effective.actorType || undefined, outcome: effective.outcome || undefined, q: effective.q || undefined, tenantId: effective.tenantId || undefined, from: effective.from || undefined, to: effective.to || undefined, actorId: undefined },
        "audit-log.csv"
      );
      toast.success("Audit log diunduh. Pengunduhan ini tercatat di audit log.");
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Audit log belum bisa diunduh"));
    } finally {
      setExporting(false);
    }
  }

  if (failed && !loaded) return <LoadError onRetry={load} />;

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Audit Log</h1>
          <p className="mt-1 max-w-2xl text-[13px] text-text-secondary">
            Jejak setiap perubahan yang dilakukan tim ElProof dan staf WO: siapa, kapan, dan hasilnya. Isi data yang diubah tidak disimpan di sini.
          </p>
        </div>
        <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={() => void handleExport()} disabled={exporting}>
          {exporting ? "Menyiapkan..." : "Unduh CSV"}
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-6">
        <div className="sm:col-span-2">
          <Field label="Cari aksi atau alamat">
            <SearchInput placeholder="cth. activate-subscription" value={q} onChange={(e) => {
                setPage(1);
                setQ(e.target.value);
              }} />
          </Field>
        </div>
        <Field label="Pelaku">
          <Select value={filter.actorType ?? ""} onChange={(e) => set("actorType", e.target.value)}>
            <option value="">Semua pelaku</option>
            <option value="platform_admin">Tim ElProof</option>
            <option value="staff">Staf WO</option>
            <option value="client">Klien</option>
          </Select>
        </Field>
        <Field label="Hasil">
          <Select value={filter.outcome ?? ""} onChange={(e) => set("outcome", e.target.value as AuditFilter["outcome"])}>
            <option value="">Semua hasil</option>
            <option value="ok">Berhasil</option>
            <option value="failed">Gagal</option>
          </Select>
        </Field>
        <Field label="Dari tanggal">
          <Input type="date" value={filter.from ?? ""} onChange={(e) => set("from", e.target.value)} />
        </Field>
        <Field label="Sampai tanggal">
          <Input type="date" value={filter.to ?? ""} onChange={(e) => set("to", e.target.value)} />
        </Field>
        {filter.tenantId ? (
          <div className="flex items-end sm:col-span-2">
            <Button variant="secondary" size="sm" onClick={() => set("tenantId", "")}>
              Hanya Tenant {filter.tenantId}. Tampilkan semua
            </Button>
          </div>
        ) : null}
      </div>

      {failed && loaded && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          Audit log belum bisa dimuat dengan filter ini. Data di bawah mungkin tidak sesuai filter; coba ubah filter lagi.
        </p>
      )}

      <Card>
        {!loaded ? (
          <div className="py-12 text-center text-sm text-text-secondary">Memuat audit log...</div>
        ) : entries.length === 0 ? (
          <EmptyState icon={<ScrollText className="h-8 w-8" />} title="Tidak ada catatan" description="Ubah filter atau rentang tanggal untuk melihat catatan lain." />
        ) : (
          <>
            <CardList
              className="sm:hidden"
              items={entries}
              keyFor={(e) => String(e.id)}
              renderItem={(e) => (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className="font-semibold text-text-primary">{e.action}</span>
                    <Outcome e={e} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Waktu" value={formatDateTime(e.occurredAt)} />
                    <CardListField label="Pelaku" value={<Actor e={e} />} />
                    <CardListField label="Tenant" value={<TenantLink id={e.tenantId} />} />
                    <CardListField label="IP" value={e.ip} />
                  </div>
                </>
              )}
            />
            <div className="hidden sm:block">
              <Table>
                <THead>
                  <TR>
                    <TH>Waktu</TH>
                    <TH>Pelaku</TH>
                    <TH>Tenant</TH>
                    <TH>Aksi</TH>
                    <TH>Hasil</TH>
                    <TH>IP</TH>
                  </TR>
                </THead>
                <TBody>
                  {entries.map((e) => (
                    <TR key={e.id}>
                      <TD className="whitespace-nowrap">{formatDateTime(e.occurredAt)}</TD>
                      <TD><Actor e={e} /></TD>
                      <TD><TenantLink id={e.tenantId} /></TD>
                      <TD>
                        <span className="block font-medium text-text-primary">{e.action}</span>
                        <span className="block max-w-xs truncate font-mono text-[11.5px] text-text-secondary" title={e.path}>{e.path}</span>
                      </TD>
                      <TD><Outcome e={e} /></TD>
                      <TD className="font-mono text-[12.5px]">{e.ip}</TD>
                    </TR>
                  ))}
                </TBody>
              </Table>
            </div>
            <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </>
        )}
      </Card>
    </div>
  );
}
