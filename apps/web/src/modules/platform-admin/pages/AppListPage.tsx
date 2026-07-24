import { useEffect, useState } from "react";
import { Plus, KeyRound, Ban, CheckCircle2 } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Badge } from "@/shared/components/ui/Badge";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { AppFormModal } from "@/modules/platform-admin/components/AppFormModal";
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

  useEffect(() => {
    void fetchApps();
  }, [fetchApps]);

  async function handleCreate(values: CreateAppFormValues) {
    setActionError(null);
    try {
      const result = await createApp(values);
      setModalOpen(false);
      setSecretReveal({ kind: "create", appId: result.appId, secret: result.secret });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mendaftarkan aplikasi"));
    }
  }

  async function handleResetSecret(appId: string) {
    setActionError(null);
    try {
      const result = await resetAppSecret(appId);
      setSecretReveal({ kind: "reset", appId: result.appId, secret: result.secret });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mereset secret aplikasi"));
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
        <p className="max-w-2xl rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
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
                    <IconActionButton icon={KeyRound} label="Reset Secret" onClick={() => void handleResetSecret(app.appId)} />
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
                <TH>App ID</TH>
                <TH>Jenis</TH>
                <TH>Callback URL</TH>
                <TH>Status</TH>
                <TH>Terdaftar</TH>
                <TH className="text-right">Aksi</TH>
              </TR>
            </THead>
            <TBody>
              {apps.map((app) => (
                <TR key={app.appId}>
                  <TD className="font-medium">{app.name}</TD>
                  <TD className="font-mono text-[12.5px] text-text-secondary">{app.appId}</TD>
                  <TD>
                    <Badge tone={app.kind === "internal" ? "navy" : "info"}>
                      {app.kind === "internal" ? "Sistem ElProof" : "Eksternal"}
                    </Badge>
                  </TD>
                  <TD className="max-w-[220px] truncate text-[12.5px] text-text-secondary" title={app.callbackUrl}>
                    {app.callbackUrl || "-"}
                  </TD>
                  <TD>
                    <Badge tone={app.isActive ? "success" : "neutral"}>{app.isActive ? "Aktif" : "Nonaktif"}</Badge>
                  </TD>
                  <TD className="text-[12.5px] text-text-secondary">{formatDate(app.createdAt)}</TD>
                  <TD>
                    <div className="flex justify-end gap-1.5">
                      {app.kind === "external" && (
                        <>
                          <IconActionButton icon={KeyRound} label="Reset Secret" onClick={() => void handleResetSecret(app.appId)} />
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

      <AppFormModal open={modalOpen} onClose={() => setModalOpen(false)} onSubmit={(values) => void handleCreate(values)} />

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
            <div className="flex items-center justify-between rounded-md bg-surface-muted px-3 py-2">
              <span className="text-text-secondary">Secret</span>
              <span className="font-mono font-semibold text-text-primary">{secretReveal.secret}</span>
            </div>
            <p className="text-[13px] leading-relaxed text-text-secondary">
              Berikan App ID dan Secret ini kepada tim teknis aplikasi eksternal tersebut untuk menyambungkan
              sistem mereka ke pembayaran ElProof. Jika Secret hilang, gunakan tombol "Reset Secret" — nilai lama
              langsung tidak berlaku lagi.
            </p>
          </div>
        )}
      </Modal>
    </div>
  );
}
