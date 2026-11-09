import { useCallback, useEffect, useRef, useState } from "react";
import { Eye, ShieldCheck } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Field, Textarea } from "@/shared/components/ui/Input";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { formatDateTime } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { enterSupportSession } from "@/shared/lib/impersonation";
import type { SupportStatus } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { supportReasonSchema } from "@/modules/platform-admin/schemas/console.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { reason: "Alasan melihat workspace ini" };

export function TenantSupportTab({ tenantId, tenantName }: { tenantId: string; tenantName: string }) {
  const fetchTenantSupport = useConsoleStore((s) => s.fetchTenantSupport);
  const startSupportSession = useConsoleStore((s) => s.startSupportSession);
  const canStart = usePlatformCan("support-access");
  const [status, setStatus] = useState<SupportStatus | null>(null);
  const [failed, setFailed] = useState(false);
  const form = useZodForm({ schema: supportReasonSchema, initial: { reason: "" } });
  const formRef = useRef<HTMLFormElement>(null);
  // The page is being replaced by the tenant's console: keep the button busy until it goes.
  const [redirecting, setRedirecting] = useState(false);
  const starting = form.submitting || redirecting;

  const load = useCallback(() => {
    setFailed(false);
    fetchTenantSupport(tenantId)
      .then(setStatus)
      .catch(() => setFailed(true));
  }, [fetchTenantSupport, tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  const handleStart = form.handleSubmit(
    async ({ reason }) => {
      try {
        const result = await startSupportSession(tenantId, reason);
        setRedirecting(true);
        enterSupportSession(result, tenantName);
      } catch (err) {
        toast.error(getApiErrorMessage(err, "Sesi dukungan belum bisa dimulai"));
        load();
        throw err; // shown above the form as well
      }
    },
    { root: () => formRef.current, fallback: "Sesi dukungan belum bisa dimulai", known: (key) => key in FIELD_LABELS },
  );

  if (failed) return <LoadError onRetry={load} />;
  if (!status) return <div className="py-12 text-center text-sm text-text-secondary">Memuat status akses...</div>;

  return (
    <div className="flex flex-col gap-5">
      <Card>
        <CardHeader title="Lihat sebagai tenant" subtitle="Membuka WO Console sebagai Owner, hanya untuk dilihat. Tidak ada data yang bisa diubah, dan Owner bisa melihat riwayatnya." />
        <CardContent className="flex flex-col gap-4">
          {status.grant ? (
            <p className="flex items-start gap-2 text-[13.5px] text-text-primary">
              <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-success-text" />
              <span>
                Owner memberi izin akses dukungan sampai <span className="font-semibold">{formatDateTime(status.grant.expiresAt)}</span>.
              </span>
            </p>
          ) : (
            <p className="rounded-md border border-border bg-surface-muted px-3.5 py-3 text-[13px] text-text-secondary">
              Owner belum memberi izin, atau izinnya sudah berakhir. Minta Owner membuka Pengaturan, bagian Akses dukungan, lalu memberi izin 24 jam. Tanpa izin itu, sesi tidak bisa dimulai.
            </p>
          )}
          {canStart ? (
            <form ref={formRef} noValidate onSubmit={(e) => void handleStart(e)} className="flex flex-col gap-4">
              <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
              <Field label="Alasan melihat workspace ini" required name="reason" error={form.errors.reason} hint="Dicatat di riwayat dan terlihat oleh Owner. Contoh: membantu Owner mengatur timeline default sesuai tiket #123.">
                <Textarea rows={3} value={form.values.reason} disabled={!status.grant || starting} onChange={(e) => form.set("reason", e.target.value)} onBlur={() => form.blur("reason")} />
              </Field>
              <div>
                <Button type="submit" icon={<Eye className="h-4 w-4" />} disabled={!status.grant || starting}>
                  {starting ? "Membuka..." : "Mulai sesi dukungan 60 menit"}
                </Button>
              </div>
            </form>
          ) : (
            <p className="text-[13px] text-text-secondary">Peran Anda tidak dapat memulai sesi dukungan. Hanya Support dan Super Admin yang bisa.</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Riwayat sesi dukungan" />
        {status.sessions.length === 0 ? (
          <EmptyState title="Belum ada sesi dukungan" description="Setiap kali tim melihat workspace ini, tercatat di sini." />
        ) : (
          <ul className="divide-y divide-border-light">
            {status.sessions.map((s) => (
              <li key={s.id} className="px-5 py-3.5">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-[13.5px] font-semibold text-text-primary">{s.admin}</span>
                  <Badge tone={s.endedAt ? "neutral" : new Date(s.expiresAt) > new Date() ? "warning" : "neutral"}>
                    {s.endedAt ? "Selesai" : new Date(s.expiresAt) > new Date() ? "Berjalan" : "Kedaluwarsa"}
                  </Badge>
                </div>
                <p className="mt-1 text-[12.5px] text-text-secondary">
                  {formatDateTime(s.startedAt)}. Alasan: {s.reason}
                </p>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
