import { useCallback, useEffect, useState } from "react";
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

export function TenantSupportTab({ tenantId, tenantName }: { tenantId: string; tenantName: string }) {
  const fetchTenantSupport = useConsoleStore((s) => s.fetchTenantSupport);
  const startSupportSession = useConsoleStore((s) => s.startSupportSession);
  const canStart = usePlatformCan("support-access");
  const [status, setStatus] = useState<SupportStatus | null>(null);
  const [failed, setFailed] = useState(false);
  const [reason, setReason] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [starting, setStarting] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchTenantSupport(tenantId)
      .then(setStatus)
      .catch(() => setFailed(true));
  }, [fetchTenantSupport, tenantId]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleStart() {
    const parsed = supportReasonSchema.safeParse({ reason });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message);
      return;
    }
    setError(undefined);
    setStarting(true);
    try {
      const result = await startSupportSession(tenantId, parsed.data.reason);
      enterSupportSession(result, tenantName);
    } catch (err) {
      const message = getApiErrorMessage(err, "Sesi dukungan belum bisa dimulai");
      setError(message);
      toast.error(message);
      setStarting(false);
      load();
    }
  }

  if (failed) return <LoadError onRetry={load} />;
  if (!status) return <div className="py-12 text-center text-sm text-text-secondary">Memuat status akses...</div>;

  return (
    <div className="flex flex-col gap-5">
      <Card>
        <CardHeader title="Lihat sebagai tenant" subtitle="Membuka WO Console sebagai Owner, hanya untuk dilihat. Tidak ada data yang bisa diubah, dan Owner bisa melihat riwayatnya." />
        <CardContent className="flex flex-col gap-4">
          {status.grant ? (
            <p className="flex items-start gap-2 text-[13.5px] text-text-primary">
              <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-success" />
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
            <>
              <Field label="Alasan melihat workspace ini" required error={error} hint="Dicatat di riwayat dan terlihat oleh Owner. Contoh: membantu Owner mengatur timeline default sesuai tiket #123.">
                <Textarea rows={3} value={reason} disabled={!status.grant || starting} onChange={(e) => setReason(e.target.value)} aria-invalid={error ? true : undefined} />
              </Field>
              <div>
                <Button icon={<Eye className="h-4 w-4" />} disabled={!status.grant || starting} onClick={() => void handleStart()}>
                  {starting ? "Membuka..." : "Mulai sesi dukungan 60 menit"}
                </Button>
              </div>
            </>
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
