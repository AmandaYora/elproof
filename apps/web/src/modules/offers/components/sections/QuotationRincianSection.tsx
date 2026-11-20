import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { toast } from "@/shared/stores/useToastStore";
import { RincianEditor, supplierKindLabel, type LineDraft, type PackageChoice } from "@/modules/rincian";
import { usePackageStore } from "@/modules/packages/stores/usePackageStore";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { rincianFromDetail } from "@/modules/offers/lib/rincian-adapter";
import type { QuotationDetail, QuotationLine } from "@/modules/offers/types";

interface Props {
  detail: QuotationDetail;
  canEdit: boolean;
  canMargin: boolean;
  onEditTerms: () => void;
}

/**
 * The quotation's rincian (DESAIN §5.3): the shared editor from modules/rincian, fed by the quotation API through the
 * offers adapter. "Ringkasan biaya" is the editor's own panel; the quotation adds the revision it matches and a
 * schedule that no longer adds up.
 */
export function QuotationRincianSection({ detail, canEdit, canMargin, onEditTerms }: Props) {
  const id = detail.quotation.id;
  const saveLines = useQuotationStore((s) => s.saveLines);
  const previewLines = useQuotationStore((s) => s.previewLines);
  const fetchDetail = useQuotationStore((s) => s.fetchDetail);
  const refreshPackage = useQuotationStore((s) => s.refreshPackage);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);
  const searchActivePackages = usePackageStore((s) => s.searchActivePackages);
  const [refreshing, setRefreshing] = useState<QuotationLine | null>(null);
  const [refreshBusy, setRefreshBusy] = useState(false);
  const [refreshError, setRefreshError] = useState<string | null>(null);

  useEffect(() => {
    if (useVendorCategoryStore.getState().categories.length === 0) fetchCategories().catch(() => undefined);
  }, [fetchCategories]);

  // One Rincian per detail: the editor follows a new saved rincian by identity.
  const rincian = useMemo(() => rincianFromDetail(detail, canMargin), [detail, canMargin]);
  const eventChoices = useMemo(() => detail.events.map((e) => ({ id: e.id, name: e.kindName })), [detail.events]);
  const supplierChoices = useMemo(
    () => detail.suppliers.filter((s) => s.paidBy === "WO").map((s) => ({ id: s.id, name: s.name, kindLabel: supplierKindLabel(s) })),
    [detail.suppliers],
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
  const preview = useCallback((lines: LineDraft[], signal: AbortSignal) => previewLines(id, lines, signal, canMargin), [id, previewLines, canMargin]);
  const save = useCallback((lines: LineDraft[], version: string) => saveLines(id, lines, version, canMargin), [id, saveLines, canMargin]);
  const reload = useCallback(async () => rincianFromDetail(await fetchDetail(id), canMargin), [id, fetchDetail, canMargin]);

  const changedPackages = detail.lines.filter((l) => l.kind === "Package" && l.catalogChanged);

  async function confirmRefresh() {
    if (!refreshing) return;
    setRefreshBusy(true);
    setRefreshError(null);
    try {
      await refreshPackage(id, refreshing.id);
      toast.success(`${refreshing.description} diperbarui dari katalog`);
      setRefreshing(null);
    } catch (err) {
      setRefreshError(getApiErrorMessage(err, "Paket belum bisa diperbarui dari katalog. Coba lagi."));
    } finally {
      setRefreshBusy(false);
    }
  }

  const matching = detail.matchingRevisionN;
  const summaryExtra = (
    <div className="mt-3 flex flex-col gap-2 text-[12.5px] text-text-secondary">
      {matching ? (
        <p>Sama dengan Revisi {matching}.</p>
      ) : detail.latestRevisionN ? (
        <p>Ada perubahan sejak Revisi {detail.latestRevisionN}.</p>
      ) : (
        <p>Belum ada revisi untuk klien.</p>
      )}
      {detail.termsMismatch && detail.totals && (
        <div role="status" className="rounded-md border border-warning/30 bg-warning-soft px-3 py-2 text-warning-text">
          <p>Jumlah termin tidak sama dengan nilai kontrak ({formatCurrency(detail.totals.total)}).</p>
          {canEdit && (
            <Button size="sm" variant="secondary" className="mt-2" onClick={onEditTerms}>
              Ubah termin
            </Button>
          )}
        </div>
      )}
    </div>
  );

  return (
    <section id="rincian" aria-labelledby="rincian-judul" className="flex scroll-mt-32 flex-col gap-3">
      <h2 id="rincian-judul" className="sr-only">
        Rincian
      </h2>
      {detail.totalsProblem && (
        <p role="alert" className="flex items-start gap-2 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] text-danger-text">
          <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
          {detail.totalsProblem}
        </p>
      )}
      {canEdit && changedPackages.length > 0 && (
        <div role="status" className="flex flex-col gap-2 rounded-md border border-info/30 bg-info-soft px-3.5 py-2.5 text-[13px] text-info-text">
          {changedPackages.map((l) => (
            <div key={l.id} className="flex flex-wrap items-center justify-between gap-2">
              <span>
                {l.description} berubah di katalog{l.copiedAt ? ` sejak disalin ${formatDate(l.copiedAt)}` : ""}.
              </span>
              <Button size="sm" variant="secondary" icon={<RefreshCw className="h-3.5 w-3.5" />} onClick={() => setRefreshing(l)}>
                Perbarui dari katalog
              </Button>
            </div>
          ))}
        </div>
      )}
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
        summaryExtra={summaryExtra}
      />
      <ConfirmDialog
        open={refreshing !== null}
        title={refreshing ? `Perbarui ${refreshing.description} dari katalog?` : "Perbarui dari katalog?"}
        confirmLabel="Perbarui dari katalog"
        tone="primary"
        isSubmitting={refreshBusy}
        error={refreshError}
        onConfirm={() => void confirmRefresh()}
        onClose={() => {
          setRefreshing(null);
          setRefreshError(null);
        }}
      >
        <p>
          Nama, harga, isi paket, dan aturan tamu diambil lagi dari katalog. Simpan dulu perubahan rincian lain yang belum
          tersimpan: rincian dibaca ulang setelah paket diperbarui.
        </p>
      </ConfirmDialog>
    </section>
  );
}
