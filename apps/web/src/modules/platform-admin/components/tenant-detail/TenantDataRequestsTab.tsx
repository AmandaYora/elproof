import { useCallback, useEffect, useRef, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Field, Input, Select, Textarea } from "@/shared/components/ui/Input";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { formatDate } from "@/shared/lib/formatters";
import type { DataRequest } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { dataRequestSchema } from "@/modules/platform-admin/schemas/console.schema";
import { DATA_REQUEST_KIND_LABEL, DATA_REQUEST_STATUS, dueText } from "@/modules/platform-admin/lib/data-requests";
import { CompleteDataRequestModal } from "@/modules/platform-admin/components/CompleteDataRequestModal";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { kind: "Jenis permintaan", requestedBy: "Diminta oleh", note: "Catatan" };

export function TenantDataRequestsTab({ tenantId, ownerEmail, onChanged }: { tenantId: string; ownerEmail: string; onChanged: () => void }) {
  const fetchTenantDataRequests = useConsoleStore((s) => s.fetchTenantDataRequests);
  const addTenantDataRequest = useConsoleStore((s) => s.addTenantDataRequest);
  const completeDataRequest = useConsoleStore((s) => s.completeDataRequest);
  const canWrite = usePlatformCan("tenant-manage");
  const [requests, setRequests] = useState<DataRequest[] | null>(null);
  const [failed, setFailed] = useState(false);
  const form = useZodForm({ schema: dataRequestSchema, initial: { kind: "export", requestedBy: ownerEmail, note: "" } });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [closing, setClosing] = useState<DataRequest | undefined>(undefined);

  const load = useCallback(() => {
    setFailed(false);
    fetchTenantDataRequests(tenantId)
      .then(setRequests)
      .catch(() => setFailed(true));
  }, [fetchTenantDataRequests, tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  // The person, the kind and the requester stay for the next entry; only the note is cleared.
  const handleAdd = form.handleSubmit(
    async (parsed) => {
      await addTenantDataRequest(tenantId, parsed);
      form.reset({ kind: parsed.kind, requestedBy: parsed.requestedBy, note: "" });
      toast.success("Permintaan data dicatat. Batas jawab 30 hari.");
      load();
      onChanged();
    },
    { root: () => formRef.current, fallback: "Permintaan belum bisa dicatat. Periksa koneksi lalu coba lagi.", known: (key) => key in FIELD_LABELS },
  );

  // Errors are not caught here: they propagate to the modal, which shows them inside.
  async function handleClose(status: "done" | "rejected", closeNote: string) {
    if (!closing) return;
    await completeDataRequest(closing.id, status, closeNote);
    toast.success("Permintaan data ditutup");
    setClosing(undefined);
    load();
    onChanged();
  }

  if (failed) return <LoadError onRetry={load} />;

  return (
    <div className="flex flex-col gap-5">
      {canWrite && (
        <Card>
          <CardHeader title="Catat permintaan data" subtitle="Saat pemilik WO meminta salinan atau penghapusan datanya (UU PDP), catat di sini. Batas menjawab 30 hari sejak dicatat." />
          <CardContent>
            <form ref={formRef} noValidate onSubmit={(e) => void handleAdd(e)} className="flex flex-col gap-4">
              <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <Field label="Jenis permintaan" name="kind" error={errors.kind}>
                  <Select value={values.kind} onChange={(e) => set("kind", e.target.value as "export" | "delete")}>
                    <option value="export">{DATA_REQUEST_KIND_LABEL.export}</option>
                    <option value="delete">{DATA_REQUEST_KIND_LABEL.delete}</option>
                  </Select>
                </Field>
                <Field label="Diminta oleh" name="requestedBy" error={errors.requestedBy}>
                  <Input value={values.requestedBy} onChange={(e) => set("requestedBy", e.target.value)} onBlur={() => blur("requestedBy")} />
                </Field>
                <div className="sm:col-span-2">
                  <Field label="Catatan" name="note" error={errors.note}>
                    <Textarea rows={2} value={values.note} onChange={(e) => set("note", e.target.value)} onBlur={() => blur("note")} />
                  </Field>
                </div>
              </div>
              <div>
                <Button type="submit" disabled={form.submitting}>
                  {form.submitting ? "Menyimpan..." : "Catat permintaan"}
                </Button>
              </div>
            </form>
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
                    <Button size="sm" variant="secondary" className="shrink-0" onClick={() => setClosing(r)}>
                      Tutup permintaan
                    </Button>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </Card>

      <CompleteDataRequestModal key={closing?.id ?? "none"} request={closing} onClose={() => setClosing(undefined)} onSubmit={handleClose} />
    </div>
  );
}
