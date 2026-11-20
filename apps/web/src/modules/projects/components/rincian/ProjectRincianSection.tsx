import { useCallback, useEffect, useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import { RincianEditor, supplierKindLabel, type LineDraft, type PackageChoice, type Rincian } from "@/modules/rincian";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { SUPPLIER_FILTER_PARAM } from "@/modules/projects/components/suppliers/supplier-helpers";
import { usePackageStore } from "@/modules/packages/stores/usePackageStore";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ProjectEvent } from "@/modules/projects/types";

// One empty list, so the selector below returns the same value until the events load (a fresh [] would re-render forever).
const NO_EVENTS: ProjectEvent[] = [];

/**
 * The project's rincian on "Acara & rincian" (DESAIN §6.10): the generic editor from modules/rincian, fed by this
 * project's store (lines, events, the vendors and venues the WO pays, the catalog), plus the payment schedule notice.
 */
export function ProjectRincianSection({ projectId }: { projectId: string }) {
  const canEdit = useCan("project.edit");
  const rincian = useProjectStore((s) => (s.rincianFor === projectId ? s.rincian : null));
  const fetchRincian = useProjectStore((s) => s.fetchRincian);
  const saveRincian = useProjectStore((s) => s.saveRincian);
  const previewRincian = useProjectStore((s) => s.previewRincian);
  const events = useProjectStore((s) => (s.eventsFor === projectId ? s.events : NO_EVENTS));
  const suppliers = useProjectStore((s) => s.suppliers);
  const fetchSuppliers = useProjectStore((s) => s.fetchSuppliers);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);
  const searchActivePackages = usePackageStore((s) => s.searchActivePackages);
  const [failed, setFailed] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchRincian(projectId).catch(() => setFailed(true));
  }, [projectId, fetchRincian]);

  useEffect(load, [load]);
  useEffect(() => {
    // What a line may point at: not needed to show the rincian, so a failure only leaves the pickers short.
    fetchSuppliers(projectId).catch(() => undefined);
    if (useVendorCategoryStore.getState().categories.length === 0) fetchCategories().catch(() => undefined);
  }, [projectId, fetchSuppliers, fetchCategories]);

  const eventChoices = useMemo(() => events.map((e) => ({ id: e.id, name: e.kindName })), [events]);
  const supplierChoices = useMemo(
    () =>
      suppliers
        .filter((s) => s.paidBy === "WO" && s.engagementStatus !== "Cancelled")
        .map((s) => ({ id: s.id, name: s.name, kindLabel: supplierKindLabel(s) })),
    [suppliers],
  );
  const categoryChoices = useMemo(() => categories.filter((c) => c.isActive).map((c) => ({ id: c.id, name: c.name })), [categories]);

  const searchPackages = useCallback(
    async (q: string): Promise<PackageChoice[]> =>
      (await searchActivePackages(q)).map((p) => ({
        id: p.id,
        name: p.name,
        price: p.price,
        includedGuests: p.includedGuests ?? 0,
        componentCount: p.componentCount,
        eventKindNames: p.eventKinds.map((k) => k.name),
      })),
    [searchActivePackages],
  );
  const preview = useCallback((lines: LineDraft[], signal: AbortSignal) => previewRincian(projectId, lines, signal), [projectId, previewRincian]);
  const save = useCallback((lines: LineDraft[], version: string) => saveRincian(projectId, lines, version), [projectId, saveRincian]);
  const reload = useCallback(() => fetchRincian(projectId), [projectId, fetchRincian]);

  if (!rincian) {
    return (
      <section aria-label="Rincian" className="rounded-lg border border-border bg-card p-5">
        {failed ? (
          <LoadError message="Rincian belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={load} />
        ) : (
          <SkeletonRows rows={4} label="Memuat rincian" />
        )}
      </section>
    );
  }

  return (
    <RincianEditor
      rincian={rincian}
      canEdit={canEdit}
      events={eventChoices}
      suppliers={supplierChoices}
      categories={categoryChoices}
      searchPackages={searchPackages}
      preview={preview}
      save={save}
      reload={reload}
      missingCostHref={`${ROUTE_PATHS.projectDetail(projectId, "vendor")}?${SUPPLIER_FILTER_PARAM}=tanpa-biaya`}
      summaryExtra={<TermsMismatchNotice projectId={projectId} rincian={rincian} />}
    />
  );
}

/**
 * The schedule no longer adds up to the contract value (PLAN §2.2, ADR-0042): said here, and fitted only when the
 * person asks ("Hitung ulang termin"), never silently.
 */
function TermsMismatchNotice({ projectId, rincian }: { projectId: string; rincian: Rincian }) {
  const canRecord = useCan("finance.record");
  const recalculate = useProjectStore((s) => s.recalculatePaymentTerms);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  if (!rincian.termsMismatch || !rincian.hasTerms || rincian.termsTotal === undefined || !rincian.totals) return null;
  const total = rincian.totals.total;

  async function handle() {
    setBusy(true);
    setError(null);
    try {
      await recalculate(projectId);
      setConfirming(false);
      toast.success("Jadwal termin dihitung ulang");
    } catch (err) {
      setError(getApiErrorMessage(err, "Jadwal termin belum dihitung ulang. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div role="status" className="mt-3 rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] text-warning-text">
      <p>
        Jadwal termin ({formatCurrency(rincian.termsTotal)}) tidak sama dengan nilai kontrak ({formatCurrency(total)}).
      </p>
      {canRecord && (
        <Button size="sm" variant="secondary" className="mt-2" icon={<RefreshCw className="h-3.5 w-3.5" />} onClick={() => setConfirming(true)}>
          Hitung ulang termin
        </Button>
      )}
      <ConfirmDialog
        open={confirming}
        title="Hitung ulang jadwal termin?"
        confirmLabel="Hitung ulang termin"
        tone="primary"
        isSubmitting={busy}
        error={error}
        onConfirm={() => void handle()}
        onClose={() => {
          setConfirming(false);
          setError(null);
        }}
      >
        <p>
          Termin yang sudah ada pembayarannya tidak berubah. Termin yang belum dibayar dihitung ulang sesuai porsinya
          sampai jumlah semua termin sama dengan nilai kontrak {formatCurrency(total)}. Nama dan tanggal jatuh tempo tetap.
        </p>
      </ConfirmDialog>
    </div>
  );
}
