import { useCallback, useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { toast } from "@/shared/stores/useToastStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { formatDateTime } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { SupportStatus } from "@/modules/platform-admin/data/console.types";

/**
 * Pengaturan: the Owner decides whether ElProof's support team may look at this
 * workspace. Consent lasts 24 hours and can be withdrawn at any time; every
 * session the team starts is listed here with its reason.
 */
export function SupportAccessCard() {
  const canManage = useCan("business_profile.manage");
  const [status, setStatus] = useState<SupportStatus | null>(null);
  const [failed, setFailed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [confirmRevoke, setConfirmRevoke] = useState(false);
  const [revokeError, setRevokeError] = useState<string | null>(null);

  const load = useCallback(() => {
    setFailed(false);
    httpClient
      .get(API.workspace.supportAccess)
      .then((res) => setStatus(res.data.data as SupportStatus))
      .catch(() => setFailed(true));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function grant() {
    setBusy(true);
    try {
      const res = await httpClient.put(API.workspace.supportAccess);
      setStatus(res.data.data as SupportStatus);
      toast.success("Akses dukungan diberikan selama 24 jam");
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Akses dukungan belum bisa diberikan"));
    } finally {
      setBusy(false);
    }
  }

  async function revoke() {
    setRevokeError(null);
    setBusy(true);
    try {
      const res = await httpClient.delete(API.workspace.supportAccess);
      setStatus(res.data.data as SupportStatus);
      setConfirmRevoke(false);
      toast.success("Akses dukungan dicabut");
    } catch (err) {
      setRevokeError(getApiErrorMessage(err, "Akses dukungan belum bisa dicabut"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Card>
      <CardHeader
        title="Akses dukungan ElProof"
        subtitle="Izinkan tim ElProof melihat workspace ini saat Anda butuh bantuan. Mereka hanya bisa melihat, tidak bisa mengubah data."
      />
      <CardContent className="flex flex-col gap-4">
        {failed && !status ? (
          <LoadError message="Status akses dukungan belum bisa dimuat." onRetry={load} />
        ) : !status ? (
          <p className="text-[13px] text-text-secondary">Memuat...</p>
        ) : (
          <>
            {status.grant ? (
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="flex items-start gap-2 text-[13.5px] text-text-primary">
                  <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-success-text" />
                  <span>
                    Tim ElProof boleh melihat workspace ini sampai <span className="font-semibold">{formatDateTime(status.grant.expiresAt)}</span>.
                  </span>
                </p>
                {canManage && (
                  <Button size="sm" variant="secondary" disabled={busy} onClick={() => { setRevokeError(null); setConfirmRevoke(true); }}>
                    Cabut akses
                  </Button>
                )}
              </div>
            ) : (
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="text-[13.5px] text-text-secondary">Saat ini tim ElProof tidak bisa melihat workspace Anda.</p>
                {canManage && (
                  <Button size="sm" disabled={busy} onClick={() => void grant()}>
                    {busy ? "Memproses..." : "Beri izin 24 jam"}
                  </Button>
                )}
              </div>
            )}

            <div>
              <h3 className="text-[13px] font-semibold text-text-primary">Riwayat sesi</h3>
              {status.sessions.length === 0 ? (
                <p className="mt-1 text-[13px] text-text-secondary">Belum pernah ada sesi dukungan.</p>
              ) : (
                <ul className="mt-2 divide-y divide-border-light rounded-md border border-border">
                  {status.sessions.map((s) => (
                    <li key={s.id} className="px-3.5 py-2.5">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-[13px] font-semibold text-text-primary">{s.admin}</span>
                        <Badge tone={!s.endedAt && new Date(s.expiresAt) > new Date() ? "warning" : "neutral"}>
                          {!s.endedAt && new Date(s.expiresAt) > new Date() ? "Sedang melihat" : "Selesai"}
                        </Badge>
                      </div>
                      <p className="mt-0.5 text-[12.5px] text-text-secondary">
                        {formatDateTime(s.startedAt)}. Alasan: {s.reason}
                      </p>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </>
        )}
      </CardContent>

      <ConfirmDialog
        open={confirmRevoke}
        title="Cabut akses dukungan?"
        confirmLabel="Cabut akses"
        isSubmitting={busy}
        error={revokeError}
        onConfirm={() => void revoke()}
        onClose={() => setConfirmRevoke(false)}
      >
        <p>Tim ElProof tidak bisa memulai sesi baru, dan sesi yang sedang berjalan langsung berakhir.</p>
      </ConfirmDialog>
    </Card>
  );
}
