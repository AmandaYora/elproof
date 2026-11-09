import { useEffect, useState } from "react";
import { Plus, KeyRound, Ban, CheckCircle2, Info, Copy, Check } from "lucide-react";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Badge } from "@/shared/components/ui/Badge";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { AppFormModal } from "@/modules/platform-admin/components/AppFormModal";
import { AppDetailModal } from "@/modules/platform-admin/components/AppDetailModal";
import { useAppsStore, type PaymentApp } from "@/modules/platform-admin/stores/useAppsStore";
import type { CreateAppFormValues } from "@/modules/platform-admin/schemas/app.schema";
import { formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";

type SecretReveal =
  | { kind: "create"; appId: string; secret: string }
  | { kind: "reset"; appId: string; secret: string };

export default function AppListPage() {
  const apps = useAppsStore((s) => s.apps);
  const isLoading = useAppsStore((s) => s.isLoading);
  const fetchApps = useAppsStore((s) => s.fetchApps);
  const createApp = useAppsStore((s) => s.createApp);
  const resetAppSecret = useAppsStore((s) => s.resetAppSecret);
  const toggleAppActive = useAppsStore((s) => s.toggleAppActive);

  const [modalOpen, setModalOpen] = useState(false);
  const [secretReveal, setSecretReveal] = useState<SecretReveal | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [detailApp, setDetailApp] = useState<PaymentApp | undefined>(undefined);
  const [resetTarget, setResetTarget] = useState<PaymentApp | undefined>(undefined);
  const [resetError, setResetError] = useState<string | null>(null);
  const [isResetting, setIsResetting] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);

  async function copy(label: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(label);
      setTimeout(() => setCopied(null), 2000);
    } catch {
      setCopied(null);
    }
  }

  useEffect(() => {
    void fetchApps();
  }, [fetchApps]);

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleCreate(values: CreateAppFormValues) {
    setActionError(null);
    const result = await createApp(values);
    setModalOpen(false);
    setSecretReveal({ kind: "create", appId: result.appId, secret: result.secret });
  }

  async function handleConfirmReset() {
    if (!resetTarget) return;
    setResetError(null);
    setIsResetting(true);
    try {
      const result = await resetAppSecret(resetTarget.appId);
      setResetTarget(undefined);
      setSecretReveal({ kind: "reset", appId: result.appId, secret: result.secret });
    } catch (err) {
      setResetError(getApiErrorMessage(err, "Gagal mereset secret aplikasi"));
    } finally {
      setIsResetting(false);
    }
  }

  async function handleToggleActive(app: PaymentApp) {
    setActionError(null);
    try {
      await toggleAppActive(app.appId);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status aplikasi"));
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Manajemen Aplikasi</h1>
          <p className="mt-1 max-w-2xl text-[13px] text-text-secondary">
            Kelola aplikasi yang diizinkan memproses pembayaran melalui ElProof.{" "}
            <span className="font-medium text-text-primary">ElProof Billing</span> mewakili sistem langganan
            ElProof sendiri dan selalu aktif secara otomatis; aplikasi lain di bawah adalah pihak eksternal yang
            Anda daftarkan sendiri.
          </p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={() => setModalOpen(true)}>
          Tambah Aplikasi Eksternal
        </Button>
      </div>

      {actionError && (
        <p className="max-w-2xl rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {actionError}
        </p>
      )}

      <Card>
        {isLoading && apps.length === 0 ? (
          <div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>
        ) : apps.length === 0 ? (
          <EmptyState title="Belum ada aplikasi" description="Daftarkan aplikasi eksternal pertama untuk mulai membuat charge lewat dompet ElProof." />
        ) : (
          <>
          <CardList
            className="sm:hidden"
            items={apps}
            keyFor={(app) => app.appId}
            renderItem={(app) => (
              <>
                <div className="flex items-start justify-between gap-3">
                  <span className="font-medium text-text-primary">{app.name}</span>
                  <Badge tone={app.isActive ? "success" : "neutral"}>{app.isActive ? "Aktif" : "Nonaktif"}</Badge>
                </div>
                <div className="flex flex-col gap-1.5">
                  <CardListField label="App ID" value={<span className="font-mono">{app.appId}</span>} />
                  <CardListField label="Jenis" value={<Badge tone={app.kind === "internal" ? "navy" : "info"}>{app.kind === "internal" ? "Sistem ElProof" : "Eksternal"}</Badge>} />
                  <CardListField label="Callback URL" value={app.callbackUrl || "-"} />
                  <CardListField label="Terdaftar" value={formatDate(app.createdAt)} />
                </div>
                {app.kind === "external" && (
                  <div className="flex items-center gap-1.5 pt-1">
                    <IconActionButton icon={Info} label="Detail" onClick={() => setDetailApp(app)} />
                    <IconActionButton icon={KeyRound} label="Reset Secret" onClick={() => {
                      setResetError(null);
                      setResetTarget(app);
                    }} />
                    <IconActionButton
                      icon={app.isActive ? Ban : CheckCircle2}
                      label={app.isActive ? "Nonaktifkan" : "Aktifkan"}
                      tone={app.isActive ? "danger" : "success"}
                      onClick={() => void handleToggleActive(app)}
                    />
                  </div>
                )}
              </>
            )}
          />
          <div className="hidden sm:block">
          <Table>
            <THead>
              <TR>
                <TH>Nama</TH>
                <TH>Jenis</TH>
                <TH>Status</TH>
                <TH className="text-right">Aksi</TH>
              </TR>
            </THead>
            <TBody>
              {apps.map((app) => (
                <TR key={app.appId}>
                  <TD className="font-medium">{app.name}</TD>
                  <TD>
                    <Badge tone={app.kind === "internal" ? "navy" : "info"}>
                      {app.kind === "internal" ? "Sistem ElProof" : "Eksternal"}
                    </Badge>
                  </TD>
                  <TD>
                    <Badge tone={app.isActive ? "success" : "neutral"}>{app.isActive ? "Aktif" : "Nonaktif"}</Badge>
                  </TD>
                  <TD>
                    <div className="flex justify-end gap-1.5">
                      {app.kind === "external" && (
                        <>
                          <IconActionButton icon={Info} label="Detail" onClick={() => setDetailApp(app)} />
                          <IconActionButton icon={KeyRound} label="Reset Secret" onClick={() => {
                      setResetError(null);
                      setResetTarget(app);
                    }} />
                          <IconActionButton
                            icon={app.isActive ? Ban : CheckCircle2}
                            label={app.isActive ? "Nonaktifkan" : "Aktifkan"}
                            tone={app.isActive ? "danger" : "success"}
                            onClick={() => void handleToggleActive(app)}
                          />
                        </>
                      )}
                    </div>
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
          </div>
          </>
        )}
      </Card>

      <AppFormModal open={modalOpen} onClose={() => setModalOpen(false)} onSubmit={handleCreate} />

      <AppDetailModal open={detailApp !== undefined} onClose={() => setDetailApp(undefined)} app={detailApp} />

      <Modal
        open={secretReveal !== null}
        onClose={() => setSecretReveal(null)}
        title={secretReveal?.kind === "create" ? "Aplikasi Berhasil Didaftarkan" : "Secret Berhasil Direset"}
        description="Secret ini hanya ditampilkan sekali dan tidak dapat dilihat ulang — salin dan simpan sekarang."
        footer={<Button onClick={() => setSecretReveal(null)}>Selesai</Button>}
      >
        {secretReveal && (
          <div className="flex flex-col gap-3 text-[13.5px]">
            <div className="flex items-center justify-between rounded-md bg-surface-muted px-3 py-2">
              <span className="text-text-secondary">App ID</span>
              <span className="font-mono font-semibold text-text-primary">{secretReveal.appId}</span>
            </div>
            <div className="flex items-center justify-between gap-3 rounded-md bg-surface-muted px-3 py-2">
              <span className="text-text-secondary">Secret</span>
              <span className="min-w-0 break-all font-mono font-semibold text-text-primary">{secretReveal.secret}</span>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                variant="secondary"
                size="sm"
                icon={copied === "id" ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                onClick={() => void copy("id", secretReveal.appId)}
              >
                {copied === "id" ? "App ID tersalin" : "Salin App ID"}
              </Button>
              <Button
                variant="secondary"
                size="sm"
                icon={copied === "secret" ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                onClick={() => void copy("secret", secretReveal.secret)}
              >
                {copied === "secret" ? "Secret tersalin" : "Salin Secret"}
              </Button>
            </div>
            <p className="text-[13px] leading-relaxed text-text-secondary">
              Berikan App ID dan Secret ini kepada tim teknis aplikasi eksternal tersebut untuk menyambungkan
              sistem mereka ke pembayaran ElProof. Jika Secret hilang, gunakan tombol "Reset Secret" — nilai lama
              langsung tidak berlaku lagi.
            </p>
          </div>
        )}
      </Modal>
      <ConfirmDialog
        open={resetTarget !== undefined}
        title="Reset secret aplikasi?"
        confirmLabel="Reset secret"
        confirmText={resetTarget?.appId}
        isSubmitting={isResetting}
        error={resetError}
        onConfirm={() => void handleConfirmReset()}
        onClose={() => setResetTarget(undefined)}
      >
        <p>
          Secret lama <span className="font-semibold text-text-primary">{resetTarget?.name}</span> langsung tidak berlaku.
          Integrasi aplikasi itu akan gagal sampai tim teknisnya memasang secret baru.
        </p>
      </ConfirmDialog>
    </div>
  );
}
