import { useCallback, useEffect, useRef, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Field, Textarea } from "@/shared/components/ui/Input";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { formatDateTime } from "@/shared/lib/formatters";
import type { TenantNote } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { noteSchema } from "@/modules/platform-admin/schemas/console.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { body: "Catatan" };

export function TenantNotesTab({ tenantId }: { tenantId: string }) {
  const fetchTenantNotes = useConsoleStore((s) => s.fetchTenantNotes);
  const addTenantNote = useConsoleStore((s) => s.addTenantNote);
  const canWrite = usePlatformCan("tenant-manage");
  const [notes, setNotes] = useState<TenantNote[] | null>(null);
  const [failed, setFailed] = useState(false);
  const form = useZodForm({ schema: noteSchema, initial: { body: "" } });
  const formRef = useRef<HTMLFormElement>(null);

  const load = useCallback(() => {
    setFailed(false);
    fetchTenantNotes(tenantId)
      .then(setNotes)
      .catch(() => setFailed(true));
  }, [fetchTenantNotes, tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  const handleAdd = form.handleSubmit(
    async (parsed) => {
      await addTenantNote(tenantId, parsed.body);
      form.reset({ body: "" });
      toast.success("Catatan disimpan");
      load();
    },
    { root: () => formRef.current, fallback: "Catatan belum bisa disimpan. Periksa koneksi lalu coba lagi.", known: (key) => key in FIELD_LABELS },
  );

  if (failed) return <LoadError onRetry={load} />;

  return (
    <div className="flex flex-col gap-5">
      {canWrite && (
        <Card>
          <CardHeader title="Tambah catatan" subtitle="Hanya terlihat oleh tim ElProof. Catat hasil percakapan, janji, atau konteks yang perlu diketahui rekan lain." />
          <CardContent>
            <form ref={formRef} noValidate onSubmit={(e) => void handleAdd(e)} className="flex flex-col gap-3">
              <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
              <Field label="Catatan" name="body" error={form.errors.body}>
                <Textarea rows={3} value={form.values.body} onChange={(e) => form.set("body", e.target.value)} onBlur={() => form.blur("body")} />
              </Field>
              <div>
                <Button type="submit" disabled={form.submitting}>
                  {form.submitting ? "Menyimpan..." : "Simpan catatan"}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
      <Card>
        <CardHeader title="Riwayat catatan" />
        {!notes ? (
          <div className="py-10 text-center text-sm text-text-secondary">Memuat catatan...</div>
        ) : notes.length === 0 ? (
          <EmptyState title="Belum ada catatan" description="Catatan tentang tenant ini akan muncul di sini." />
        ) : (
          <ul className="divide-y divide-border-light">
            {notes.map((n) => (
              <li key={n.id} className="px-5 py-3.5">
                <p className="whitespace-pre-line text-[13.5px] text-text-primary">{n.body}</p>
                <p className="mt-1 text-[12.5px] text-text-secondary">
                  {n.admin}, {formatDateTime(n.createdAt)}
                </p>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
