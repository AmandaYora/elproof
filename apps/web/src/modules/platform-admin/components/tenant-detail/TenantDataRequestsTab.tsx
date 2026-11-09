import { useCallback, useEffect, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Field, Input, Select, Textarea } from "@/shared/components/ui/Input";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { DataRequest } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { dataRequestSchema } from "@/modules/platform-admin/schemas/console.schema";
import { DATA_REQUEST_KIND_LABEL, DATA_REQUEST_STATUS, dueText } from "@/modules/platform-admin/lib/data-requests";
import { CompleteDataRequestModal } from "@/modules/platform-admin/components/CompleteDataRequestModal";

export function TenantDataRequestsTab({ tenantId, ownerEmail, onChanged }: { tenantId: string; ownerEmail: string; onChanged: () => void }) {
  const fetchTenantDataRequests = useConsoleStore((s) => s.fetchTenantDataRequests);
  const addTenantDataRequest = useConsoleStore((s) => s.addTenantDataRequest);
  const completeDataRequest = useConsoleStore((s) => s.completeDataRequest);
  const canWrite = usePlatformCan("tenant-manage");
  const [requests, setRequests] = useState<DataRequest[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [kind, setKind] = useState<"export" | "delete">("export");
  const [requestedBy, setRequestedBy] = useState(ownerEmail);
  const [note, setNote] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [closing, setClosing] = useState<DataRequest | undefined>(undefined);
  const [closeError, setCloseError] = useState<string | null>(null);
  const [closingBusy, setClosingBusy] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchTenantDataRequests(tenantId)
      .then(setRequests)
      .catch(() => setFailed(true));
  }, [fetchTenantDataRequests, tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleAdd() {
    const parsed = dataRequestSchema.safeParse({ kind, requestedBy, note });
    if (!parsed.success) {
      const next: Record<string, string> = {};
      for (const issue of parsed.error.issues) next[String(issue.path[0])] ??= issue.message;
      setErrors(next);
      return;
    }
    setErrors({});
    setSaving(true);
    try {
      await addTenantDataRequest(tenantId, parsed.data);
      setNote("");
      toast.success("Permintaan data dicatat. Batas jawab 30 hari.");
      load();
      onChanged();
    } catch (err) {
      setErrors({ note: getApiErrorMessage(err, "Permintaan belum bisa dicatat") });
    } finally {
      setSaving(false);
    }
  }

  async function handleClose(status: "done" | "rejected", closeNote: string) {
    if (!closing) return;
    setCloseError(null);
    setClosingBusy(true);
    try {
      await completeDataRequest(closing.id, status, closeNote);
      toast.success("Permintaan data ditutup");
      setClosing(undefined);
      load();
      onChanged();
    } catch (err) {
      setCloseError(getApiErrorMessage(err, "Permintaan belum bisa ditutup"));
    } finally {
      setClosingBusy(false);
    }
  }

  if (failed) return <LoadError onRetry={load} />;

  return (
    <div className="flex flex-col gap-5">
      {canWrite && (
        <Card>
          <CardHeader title="Catat permintaan data" subtitle="Saat pemilik WO meminta salinan atau penghapusan datanya (UU PDP), catat di sini. Batas menjawab 30 hari sejak dicatat." />
          <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Jenis permintaan" error={errors.kind}>
              <Select value={kind} onChange={(e) => setKind(e.target.value as "export" | "delete")}>
                <option value="export">{DATA_REQUEST_KIND_LABEL.export}</option>
                <option value="delete">{DATA_REQUEST_KIND_LABEL.delete}</option>
              </Select>
            </Field>
            <Field label="Diminta oleh" error={errors.requestedBy}>
              <Input value={requestedBy} onChange={(e) => setRequestedBy(e.target.value)} aria-invalid={errors.requestedBy ? true : undefined} />
            </Field>
            <div className="sm:col-span-2">
              <Field label="Catatan" error={errors.note}>
                <Textarea rows={2} value={note} onChange={(e) => setNote(e.target.value)} aria-invalid={errors.note ? true : undefined} />
              </Field>
            </div>
            <div className="sm:col-span-2">
              <Button onClick={() => void handleAdd()} disabled={saving}>
                {saving ? "Menyimpan..." : "Catat permintaan"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader title="Riwayat permintaan" />
        {!requests ? (
          <div className="py-10 text-center text-sm text-text-secondary">Memuat permintaan...</div>
        ) : requests.length === 0 ? (
          <EmptyState title="Belum ada permintaan data" description="Permintaan salinan atau penghapusan data dari tenant ini akan muncul di sini." />
        ) : (
          <ul className="divide-y divide-border-light">
            {requests.map((r) => {
              const due = dueText(r);
              const st = DATA_REQUEST_STATUS[r.status];
              return (
                <li key={r.id} className="flex flex-col gap-2 px-5 py-3.5 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-[13.5px] font-semibold text-text-primary">{DATA_REQUEST_KIND_LABEL[r.kind]}</span>
                      <Badge tone={st.tone}>{st.label}</Badge>
                    </div>
                    <p className="mt-1 text-[12.5px] text-text-secondary">
                      Oleh {r.requestedBy}, dicatat {formatDate(r.createdAt)}
                      {r.note ? `. ${r.note}` : ""}
                    </p>
                    <p className={`mt-0.5 text-[12.5px] ${r.status === "open" && due.late ? "font-semibold text-danger-text" : "text-text-secondary"}`}>
                      {r.status === "open" ? `Batas jawab ${formatDate(r.dueAt)}, ${due.text}` : `Ditutup ${formatDate(r.completedAt)} oleh ${r.completedBy}: ${r.completionNote}`}
                    </p>
                  </div>
                  {r.status === "open" && canWrite && (
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

      <CompleteDataRequestModal key={closing?.id ?? "none"} request={closing} onClose={() => setClosing(undefined)} onSubmit={handleClose} error={closeError} isSubmitting={closingBusy} />
    </div>
  );
}
