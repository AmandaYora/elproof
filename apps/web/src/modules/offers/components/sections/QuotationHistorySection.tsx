import { useCallback, useEffect, useState } from "react";
import { Check, Download, Eye, GitCompare, Share2 } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { formatCurrency, formatDateTime } from "@/shared/lib/formatters";
import { API } from "@/shared/services/api-endpoints";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { PageSection } from "@/modules/offers/components/sections/QuotationSections";
import type { QuotationDialog } from "@/modules/offers/lib/page-dialog";
import type { QuotationActivity, QuotationDetail, QuotationRevision } from "@/modules/offers/types";

const NO_ACTIVITY: QuotationActivity[] = [];

interface Props {
  detail: QuotationDetail;
  /** Newest first. */
  revisions: QuotationRevision[];
  revisionsStatus: "loading" | "error" | "ready";
  onRetryRevisions: () => void;
  canShare: boolean;
  open: (dialog: QuotationDialog) => void;
  onShare: (n: number) => void;
  onDownload: (n: number) => void;
}

/** "Riwayat" (DESAIN §6.2): every revision the client was sent, with its PDF and a comparison, then what happened. */
export function QuotationHistorySection({ detail, revisions, revisionsStatus, onRetryRevisions, canShare, open, onShare, onDownload }: Props) {
  const q = detail.quotation;
  const fetchActivity = useQuotationStore((s) => s.fetchActivity);
  const activity = useQuotationStore((s) => (s.activityFor === q.id ? s.activity : NO_ACTIVITY));
  const [activityFailed, setActivityFailed] = useState(false);

  const loadActivity = useCallback(() => {
    setActivityFailed(false);
    fetchActivity(q.id).catch(() => setActivityFailed(true));
  }, [fetchActivity, q.id]);
  // Each change of the quotation is a new entry in its history.
  useEffect(loadActivity, [loadActivity, q.changedAt]);

  return (
    <PageSection id="riwayat" title="Riwayat">
      <div className="flex flex-col gap-6">
        <div className="flex flex-col gap-2">
          <h3 className="text-[13.5px] font-semibold text-foreground">Revisi untuk klien</h3>
          {revisionsStatus === "error" ? (
            <LoadError message="Daftar revisi belum bisa dimuat." onRetry={onRetryRevisions} />
          ) : revisionsStatus === "loading" && revisions.length === 0 ? (
            <SkeletonRows rows={2} label="Memuat revisi" />
          ) : revisions.length === 0 ? (
            <p className="text-[13px] text-muted-foreground">Belum ada revisi. Revisi dibuat dengan "Buat revisi untuk klien" dan menjadi PDF yang dibagikan.</p>
          ) : (
            <ol className="flex flex-col divide-y divide-border-light">
              {revisions.map((r, i) => {
                const previous = revisions[i + 1];
                return (
                  <li key={r.n} className="flex flex-wrap items-center justify-between gap-3 py-2.5">
                    <div className="min-w-0 text-[13.5px]">
                      <p className="flex flex-wrap items-center gap-2">
                        <span className="font-semibold text-foreground">Revisi {r.n}</span>
                        {r.isAccepted && (
                          <Badge tone="success" dot={false}>
                            <Check aria-hidden />
                            Diterima
                          </Badge>
                        )}
                        {r.matchesWorkingCopy && !r.isAccepted && <Badge tone="info">Sama dengan sekarang</Badge>}
                      </p>
                      <p className="mt-0.5 text-[12.5px] text-muted-foreground">
                        {formatDateTime(r.createdAt)} · {r.createdByName || "-"} · <span className="tabular-nums">{formatCurrency(r.contractTotal)}</span>
                      </p>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <Button size="sm" variant="secondary" icon={<Download className="h-3.5 w-3.5" />} onClick={() => onDownload(r.n)} aria-label={`Unduh PDF Revisi ${r.n}`}>
                        Unduh PDF
                      </Button>
                      <ActionMenu
                        label={`Aksi lain Revisi ${r.n}`}
                        items={[
                          {
                            label: "Lihat PDF",
                            icon: <Eye className="h-4 w-4" />,
                            onSelect: () =>
                              open({ kind: "pdf", title: `${q.number} Revisi ${r.n}`, url: `${API.quotations.revisionPdf(q.id, r.n)}?inline=true`, revisionN: r.n }),
                          },
                          { label: "Bagikan PDF", icon: <Share2 className="h-4 w-4" />, onSelect: () => onShare(r.n), hidden: !canShare },
                          {
                            label: `Bandingkan dengan Revisi ${previous?.n ?? ""}`,
                            icon: <GitCompare className="h-4 w-4" />,
                            onSelect: () => open({ kind: "compare", from: previous?.n ?? r.n, to: r.n }),
                            hidden: !previous,
                          },
                          {
                            label: "Bandingkan dengan yang sekarang",
                            icon: <GitCompare className="h-4 w-4" />,
                            onSelect: () => open({ kind: "compare", from: r.n, to: 0 }),
                            hidden: !(i === 0 && detail.hasUnsentChanges),
                          },
                        ]}
                      />
                    </div>
                  </li>
                );
              })}
            </ol>
          )}
        </div>

        <div className="flex flex-col gap-2">
          <h3 className="text-[13.5px] font-semibold text-foreground">Aktivitas</h3>
          {activityFailed ? (
            <LoadError message="Aktivitas belum bisa dimuat." onRetry={loadActivity} />
          ) : activity.length === 0 ? (
            <p className="text-[13px] text-muted-foreground">Belum ada aktivitas.</p>
          ) : (
            <ol className="flex flex-col gap-2 text-[13px]">
              {activity.map((a) => (
                <li key={a.id} className="flex flex-wrap gap-x-2">
                  <span className="text-muted-foreground tabular-nums">{formatDateTime(a.createdAt)}</span>
                  <span className="text-foreground">{a.summary}</span>
                  {a.actorName && <span className="text-muted-foreground">· {a.actorName}</span>}
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </PageSection>
  );
}
