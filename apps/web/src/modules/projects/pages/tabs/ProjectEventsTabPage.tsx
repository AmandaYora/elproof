import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useOutletContext, useSearchParams } from "react-router-dom";
import { FileCheck2, Plus } from "lucide-react";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectEvent } from "@/modules/projects/types";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";
import { EventList, formatEventDay, type EventRowAction } from "@/modules/rincian";
import { DueShiftPrompt } from "@/modules/projects/components/events/DueShiftPrompt";
import { ProjectRincianSection } from "@/modules/projects/components/rincian/ProjectRincianSection";

// The drawer (form, pickers, venue directory) is only needed after a click.
const EventDrawer = lazy(() => import("@/modules/projects/components/events/EventDrawer"));
// "Kesepakatan awal" (a project made from a quotation): read-only, opened with ?kesepakatan=1.
const AgreementDrawer = lazy(() => import("@/modules/offers/components/AgreementDrawer"));

/**
 * "Acara & rincian": the project's events and their sessions, then the rincian (price lines) with "Ringkasan biaya"
 * (DESAIN §5.1–5.4, §6.10). The rincian's "Tambahan tamu" follows the guests of these events, so an event write
 * refetches it (useProjectStore.afterEventWrite).
 */
export default function ProjectEventsTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const canEdit = useCan("project.edit");
  const events = useProjectStore((s) => s.events);
  const eventsFor = useProjectStore((s) => s.eventsFor);
  const fetchEvents = useProjectStore((s) => s.fetchEvents);
  const setMainEvent = useProjectStore((s) => s.setMainEvent);
  const moveEvent = useProjectStore((s) => s.moveEvent);
  const duplicateEvent = useProjectStore((s) => s.duplicateEvent);
  const deleteEvent = useProjectStore((s) => s.deleteEvent);
  const quotation = useProjectStore((s) => (s.currentProject?.id === projectId ? s.currentProject.quotation : null));
  const rincian = useProjectStore((s) => (s.rincianFor === projectId ? s.rincian : null));
  const [params, setParams] = useSearchParams();
  const agreementOpen = quotation !== null && params.get("kesepakatan") === "1";

  function setAgreementOpen(open: boolean) {
    setParams(
      (p) => {
        const next = new URLSearchParams(p);
        if (open) next.set("kesepakatan", "1");
        else next.delete("kesepakatan");
        return next;
      },
      { replace: true },
    );
  }

  const [loadError, setLoadError] = useState(false);
  // undefined = closed; null = a new event.
  const [editing, setEditing] = useState<ProjectEvent | null | undefined>(undefined);
  const [deleting, setDeleting] = useState<ProjectEvent | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoadError(false);
    fetchEvents(projectId).catch(() => setLoadError(true));
  }, [projectId, fetchEvents]);

  useEffect(load, [load]);

  const loaded = eventsFor === projectId;

  async function run(event: ProjectEvent, action: () => Promise<void>, done: string | null, fallback: string) {
    setBusyId(event.id);
    try {
      await action();
      if (done) toast.success(done);
    } catch (err) {
      toast.error(getApiErrorMessage(err, fallback));
    } finally {
      setBusyId(null);
    }
  }

  function onAction(event: ProjectEvent, action: EventRowAction) {
    switch (action) {
      case "edit":
        setEditing(event);
        return;
      case "main":
        void run(event, () => setMainEvent(projectId, event.id), `${event.kindName} menjadi acara utama`, "Acara utama belum berganti. Periksa koneksi Anda, lalu coba lagi.");
        return;
      case "up":
      case "down":
        void run(event, () => moveEvent(projectId, event.id, action), null, "Urutan acara belum berubah. Periksa koneksi Anda, lalu coba lagi.");
        return;
      case "duplicate":
        void run(event, () => duplicateEvent(projectId, event.id), "Acara diduplikat", "Acara belum diduplikat. Periksa koneksi Anda, lalu coba lagi.");
        return;
      case "delete":
        setDeleteError(null);
        setDeleting(event);
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    setDeleteError(null);
    try {
      await deleteEvent(projectId, deleting.id);
      setDeleting(null);
      toast.success("Acara dihapus");
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Acara belum dihapus. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setDeleteBusy(false);
    }
  }

  const addButton = canEdit ? (
    <Button size="sm" icon={<Plus className="size-3.5" />} onClick={() => setEditing(null)}>
      Tambah acara
    </Button>
  ) : undefined;

  return (
    <div className="flex flex-col gap-5">
      {quotation && (
        <p className="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-md border border-border-light bg-surface-muted px-3.5 py-2.5 text-[13px] text-text-secondary">
          <FileCheck2 aria-hidden className="h-4 w-4 shrink-0" />
          <span>
            Dibuat dari {quotation.number} Revisi {quotation.revisionN}.
          </span>
          <Button size="sm" variant="ghost" onClick={() => setAgreementOpen(true)}>
            Lihat kesepakatan awal
          </Button>
        </p>
      )}
      <Card>
        <CardHeader
          title={<span id="acara-heading">Acara</span>}
          subtitle={loaded && events.length > 1 ? "Tanggal acara utama menjadi Hari H proyek." : undefined}
          action={loaded && events.length > 0 ? addButton : undefined}
        />
        <section aria-labelledby="acara-heading">
          {loadError ? (
            <div className="p-5">
              <LoadError message="Acara belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={load} />
            </div>
          ) : !loaded ? (
            <div className="p-5">
              <SkeletonRows rows={3} label="Memuat acara" />
            </div>
          ) : events.length === 0 ? (
            <EmptyState title="Belum ada acara" description="Tambahkan acara agar tamu dan kapasitas venue bisa dicek." action={addButton} />
          ) : (
            <EventList events={events} canEdit={canEdit} busyId={busyId} onAction={onAction} />
          )}
        </section>
      </Card>

      <ProjectRincianSection projectId={projectId} />

      {editing !== undefined && (
        <Suspense fallback={null}>
          <EventDrawer projectId={projectId} event={editing} onClose={() => setEditing(undefined)} />
        </Suspense>
      )}

      {agreementOpen && (
        <Suspense fallback={null}>
          <AgreementDrawer
            projectId={projectId}
            current={loaded && rincian ? { total: rincian.totals?.total ?? null, lines: rincian.lines, events } : null}
            onClose={() => setAgreementOpen(false)}
          />
        </Suspense>
      )}

      <DueShiftPrompt projectId={projectId} />

      <ConfirmDialog
        open={deleting !== null}
        title={deleting ? `Hapus acara ${deleting.kindName}?` : "Hapus acara?"}
        confirmLabel="Hapus acara"
        isSubmitting={deleteBusy}
        error={deleteError}
        onConfirm={() => void confirmDelete()}
        onClose={() => {
          setDeleting(null);
          setDeleteError(null);
        }}
      >
        {deleting && (
          <>
            <p>
              {deleting.kindName} pada {formatEventDay(deleting.eventDate)} beserta sesinya dihapus dan tidak bisa dipulihkan.
              {deleting.isMain && " Acara paling awal yang tersisa menjadi acara utama, dan Hari H proyek ikut berpindah."}
            </p>
            {deleting.venue && (
              <p className="mt-2">
                {deleting.venue.name} dilepas dari Vendor & venue bila tidak dipakai acara lain dan belum punya pembayaran.
              </p>
            )}
          </>
        )}
      </ConfirmDialog>
    </div>
  );
}
