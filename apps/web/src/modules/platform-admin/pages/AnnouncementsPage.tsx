import { useCallback, useEffect, useState } from "react";
import { Megaphone, Pencil, Plus, Trash2 } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge, type BadgeTone } from "@/shared/components/ui/Badge";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { toast } from "@/shared/stores/useToastStore";
import { formatDateTime } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { ELPROOF_APP_ID } from "@/shared/data/subscriptionPlans";
import type { Announcement } from "@/modules/platform-admin/data/console.types";
import { useConsoleStore, type AnnouncementInput } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { AnnouncementFormModal } from "@/modules/platform-admin/components/AnnouncementFormModal";
import { ANNOUNCEMENT_KINDS } from "@/modules/platform-admin/schemas/console.schema";
import { TENANT_STATUS_LABEL } from "@/modules/platform-admin/lib/status";
import type { TenantSubscriptionStatus } from "@/modules/platform-admin/data/types";

function phase(a: Announcement): { label: string; tone: BadgeTone } {
  if (a.active) return { label: "Tayang", tone: "success" };
  if (new Date(a.startsAt).getTime() > Date.now()) return { label: "Terjadwal", tone: "info" };
  return { label: "Selesai", tone: "neutral" };
}

export default function AnnouncementsPage() {
  const announcements = useConsoleStore((s) => s.announcements);
  const fetchAnnouncements = useConsoleStore((s) => s.fetchAnnouncements);
  const saveAnnouncement = useConsoleStore((s) => s.saveAnnouncement);
  const deleteAnnouncement = useConsoleStore((s) => s.deleteAnnouncement);
  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const canEdit = usePlatformCan("super-admin");
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);
  const [editing, setEditing] = useState<Announcement | null | undefined>(undefined); // undefined: closed, null: new
  const [deleting, setDeleting] = useState<Announcement | undefined>(undefined);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const load = useCallback(() => {
    setFailed(false);
    fetchAnnouncements()
      .then(() => setLoaded(true))
      .catch(() => setFailed(true));
    // Announcements and tenants concern WO tenants: only ElProof's own plans apply (ADR-0040).
    void fetchPlans(ELPROOF_APP_ID).catch(() => undefined);
  }, [fetchAnnouncements, fetchPlans]);

  useEffect(() => {
    load();
  }, [load]);

  function audienceLabel(a: Announcement): string {
    if (a.audience === "status") return `Status: ${TENANT_STATUS_LABEL[a.audienceValue as TenantSubscriptionStatus] ?? a.audienceValue}`;
    if (a.audience === "plan") return `Paket: ${plans.find((p) => p.id === a.audienceValue)?.name ?? "paket #" + a.audienceValue}`;
    return "Semua tenant";
  }

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleSubmit(input: AnnouncementInput) {
    await saveAnnouncement(editing?.id ?? null, input);
    toast.success(editing ? "Pengumuman diperbarui" : "Pengumuman diterbitkan");
    setEditing(undefined);
  }

  async function handleDelete() {
    if (!deleting) return;
    setDeleteError(null);
    try {
      await deleteAnnouncement(deleting.id);
      toast.success("Pengumuman dihapus");
      setDeleting(undefined);
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Pengumuman belum bisa dihapus"));
    }
  }

  if (failed && !loaded) return <LoadError onRetry={load} />;

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Pengumuman</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kabar untuk tenant: pemeliharaan, perubahan harga, atau informasi penting lainnya.</p>
        </div>
        {canEdit && (
          <Button icon={<Plus className="h-4 w-4" />} onClick={() => setEditing(null)}>
            Buat pengumuman
          </Button>
        )}
      </div>

      <Card>
        {!loaded ? (
          <div className="py-12 text-center text-sm text-text-secondary">Memuat pengumuman...</div>
        ) : announcements.length === 0 ? (
          <EmptyState
            icon={<Megaphone className="h-8 w-8" />}
            title="Belum ada pengumuman"
            description={canEdit ? "Buat pengumuman pertama, misalnya jadwal pemeliharaan." : "Super Admin dapat membuat pengumuman untuk tenant."}
          />
        ) : (
          <ul className="divide-y divide-border-light">
            {announcements.map((a) => {
              const p = phase(a);
              return (
                <li key={a.id} className="flex flex-col gap-2 px-5 py-4 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-[14px] font-semibold text-text-primary">{a.title}</span>
                      <Badge tone={p.tone}>{p.label}</Badge>
                      <Badge tone="neutral" dot={false}>{ANNOUNCEMENT_KINDS.find((k) => k.value === a.kind)?.label ?? a.kind}</Badge>
                    </div>
                    <p className="mt-1 whitespace-pre-line text-[13px] text-text-secondary">{a.body}</p>
                    <p className="mt-2 text-[12.5px] text-text-secondary">
                      {audienceLabel(a)} · {formatDateTime(a.startsAt)}
                      {a.endsAt ? ` sampai ${formatDateTime(a.endsAt)}` : ", tanpa batas akhir"}
                    </p>
                  </div>
                  {canEdit && (
                    <div className="flex shrink-0 items-center gap-1.5">
                      <IconActionButton icon={Pencil} label="Ubah pengumuman" tone="neutral" onClick={() => setEditing(a)} />
                      <IconActionButton icon={Trash2} label="Hapus pengumuman" tone="danger" onClick={() => { setDeleteError(null); setDeleting(a); }} />
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </Card>

      <AnnouncementFormModal
        key={editing?.id ?? "new"}
        open={editing !== undefined}
        onClose={() => setEditing(undefined)}
        initial={editing ?? undefined}
        plans={plans}
        onSubmit={handleSubmit}
      />

      <ConfirmDialog
        open={deleting !== undefined}
        title="Hapus pengumuman?"
        confirmLabel="Hapus pengumuman"
        error={deleteError}
        onConfirm={() => void handleDelete()}
        onClose={() => setDeleting(undefined)}
      >
        <p>
          &quot;{deleting?.title}&quot; langsung hilang dari WO Console semua tenant. Untuk menyembunyikan sementara, ubah waktu selesai tayangnya.
        </p>
      </ConfirmDialog>
    </div>
  );
}
