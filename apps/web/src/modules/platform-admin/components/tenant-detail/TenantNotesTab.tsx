import { useCallback, useEffect, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Field, Textarea } from "@/shared/components/ui/Input";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { formatDateTime } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { TenantNote } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { noteSchema } from "@/modules/platform-admin/schemas/console.schema";

export function TenantNotesTab({ tenantId }: { tenantId: string }) {
  const fetchTenantNotes = useConsoleStore((s) => s.fetchTenantNotes);
  const addTenantNote = useConsoleStore((s) => s.addTenantNote);
  const canWrite = usePlatformCan("tenant-manage");
  const [notes, setNotes] = useState<TenantNote[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [body, setBody] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchTenantNotes(tenantId)
      .then(setNotes)
      .catch(() => setFailed(true));
  }, [fetchTenantNotes, tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleAdd() {
    const parsed = noteSchema.safeParse({ body });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message);
      return;
    }
    setError(undefined);
    setSaving(true);
    try {
      await addTenantNote(tenantId, parsed.data.body);
      setBody("");
      toast.success("Catatan disimpan");
      load();
    } catch (err) {
      setError(getApiErrorMessage(err, "Catatan belum bisa disimpan"));
    } finally {
      setSaving(false);
    }
  }

  if (failed) return <LoadError onRetry={load} />;

  return (
    <div className="flex flex-col gap-5">
      {canWrite && (
        <Card>
          <CardHeader title="Tambah catatan" subtitle="Hanya terlihat oleh tim ElProof. Catat hasil percakapan, janji, atau konteks yang perlu diketahui rekan lain." />
          <CardContent className="flex flex-col gap-3">
            <Field label="Catatan" error={error}>
              <Textarea rows={3} value={body} onChange={(e) => setBody(e.target.value)} aria-invalid={error ? true : undefined} />
            </Field>
            <div>
              <Button onClick={() => void handleAdd()} disabled={saving}>
                {saving ? "Menyimpan..." : "Simpan catatan"}
              </Button>
            </div>
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
