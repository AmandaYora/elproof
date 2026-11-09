import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { FileLock2 } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { DataRequest } from "@/modules/platform-admin/data/console.types";
import { DATA_REQUEST_STATUS, dueText } from "@/modules/platform-admin/lib/data-requests";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { CompleteDataRequestModal } from "@/modules/platform-admin/components/CompleteDataRequestModal";
import { DATA_REQUEST_KIND_LABEL } from "@/modules/platform-admin/lib/data-requests";

export default function DataRequestsPage() {
  const requests = useConsoleStore((s) => s.dataRequests);
  const fetchDataRequests = useConsoleStore((s) => s.fetchDataRequests);
  const completeDataRequest = useConsoleStore((s) => s.completeDataRequest);
  const canClose = usePlatformCan("tenant-manage");
  const [openOnly, setOpenOnly] = useState(true);
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const [closing, setClosing] = useState<DataRequest | undefined>(undefined);
  const [closeError, setCloseError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchDataRequests(openOnly)
      .then(() => setLoaded(true))
      .catch(() => setFailed(true));
  }, [fetchDataRequests, openOnly]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleClose(status: "done" | "rejected", note: string) {
    if (!closing) return;
    setCloseError(null);
    setSaving(true);
    try {
      await completeDataRequest(closing.id, status, note);
      toast.success("Permintaan data ditutup");
      setClosing(undefined);
      load();
    } catch (err) {
      setCloseError(getApiErrorMessage(err, "Permintaan belum bisa ditutup"));
    } finally {
      setSaving(false);
    }
  }

  if (failed && !loaded) return <LoadError onRetry={load} />;

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Data &amp; Privasi</h1>
        <p className="mt-1 text-[13px] text-text-secondary">
          Permintaan salinan atau penghapusan data dari pemilik WO sesuai UU PDP. Setiap permintaan harus dijawab dalam 30 hari. Catat permintaan baru dari halaman detail tenant.
        </p>
      </div>

      <div className="flex gap-2">
        <Button size="sm" variant={openOnly ? "primary" : "secondary"} onClick={() => setOpenOnly(true)}>
          Perlu diproses
        </Button>
        <Button size="sm" variant={openOnly ? "secondary" : "primary"} onClick={() => setOpenOnly(false)}>
          Semua
        </Button>
      </div>

      <Card>
        {!loaded ? (
          <div className="py-12 text-center text-sm text-text-secondary">Memuat permintaan...</div>
        ) : requests.length === 0 ? (
          <EmptyState
            icon={<FileLock2 className="h-8 w-8" />}
            title={openOnly ? "Tidak ada permintaan yang menunggu" : "Belum ada permintaan data"}
            description="Permintaan dari pemilik WO akan muncul di sini setelah dicatat tim dukungan."
          />
        ) : (
          <ul className="divide-y divide-border-light">
            {requests.map((r) => {
              const due = dueText(r);
              const st = DATA_REQUEST_STATUS[r.status];
              return (
                <li key={r.id} className="flex flex-col gap-2 px-5 py-4 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link to={ROUTE_PATHS.platformTenantDetail(String(r.tenantId), "data")} className="text-[14px] font-semibold text-text-primary hover:underline">
                        {r.tenantName || `Tenant #${r.tenantId}`}
                      </Link>
                      <Badge tone={st.tone}>{st.label}</Badge>
                      <Badge tone={r.kind === "delete" ? "danger" : "info"} dot={false}>{DATA_REQUEST_KIND_LABEL[r.kind]}</Badge>
                    </div>
                    <p className="mt-1 text-[13px] text-text-secondary">
                      Diminta oleh {r.requestedBy}, dicatat {formatDate(r.createdAt)}
                      {r.note ? `. ${r.note}` : ""}
                    </p>
                    {r.status === "open" ? (
                      <p className={`mt-1 text-[12.5px] font-semibold ${due.late ? "text-danger-text" : "text-text-secondary"}`}>
                        Batas jawab {formatDate(r.dueAt)}, {due.text}
                      </p>
                    ) : (
                      <p className="mt-1 text-[12.5px] text-text-secondary">
                        Ditutup {formatDate(r.completedAt)} oleh {r.completedBy}: {r.completionNote}
                      </p>
                    )}
                  </div>
                  {r.status === "open" && canClose && (
                    <Button size="sm" variant="secondary" className="shrink-0" onClick={() => { setCloseError(null); setClosing(r); }}>
                      Tutup permintaan
                    </Button>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </Card>

      <CompleteDataRequestModal key={closing?.id ?? "none"} request={closing} onClose={() => setClosing(undefined)} onSubmit={handleClose} error={closeError} isSubmitting={saving} />
    </div>
  );
}
