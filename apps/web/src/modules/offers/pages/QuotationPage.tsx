import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import axios from "axios";
import {
  AlertTriangle, ArrowRight, CalendarPlus, Check, Copy, Download, Eye, FilePlus2, FolderOpen, GitBranch, GitCompare, Info, Paperclip,
  RotateCcw, Share2, Trash2, Undo2, Upload, X,
} from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { ActionMenu, type ActionMenuItem } from "@/shared/components/ui/ActionMenu";
import { Breadcrumbs } from "@/shared/components/ui/Breadcrumbs";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { Skeleton, SkeletonRows } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { cn } from "@/shared/lib/cn";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ACCEPT_CHANNEL_LABELS } from "@/shared/lib/labels";
import { API } from "@/shared/services/api-endpoints";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { QuotationStatusBadge } from "@/modules/offers/components/QuotationStatusBadge";
import { EventsSection, InfoSection, SuppliersSection, TermsSection, TermsTextSection } from "@/modules/offers/components/sections/QuotationSections";
import { QuotationRincianSection } from "@/modules/offers/components/sections/QuotationRincianSection";
import { QuotationHistorySection } from "@/modules/offers/components/sections/QuotationHistorySection";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { rincianEventsFromDetail } from "@/modules/offers/lib/rincian-adapter";
import { revisionWord, validityText, waNumber } from "@/modules/offers/lib/display";
import type { QuotationDialog } from "@/modules/offers/lib/page-dialog";
import type { QuotationDetail, QuotationRevision } from "@/modules/offers/types";

// Everything opened by a click is loaded by that click.
const QuotationHeaderModal = lazy(() => import("@/modules/offers/components/QuotationHeaderModal"));
const QuotationEventDrawer = lazy(() => import("@/modules/offers/components/QuotationEventDrawer"));
const QuotationSupplierModal = lazy(() => import("@/modules/offers/components/QuotationSupplierModal"));
const QuotationTermsDrawer = lazy(() => import("@/modules/offers/components/QuotationTermsDrawer"));
const QuotationTermsTextModal = lazy(() => import("@/modules/offers/components/QuotationTermsTextModal"));
const AcceptDialog = lazy(() => import("@/modules/offers/components/AcceptDialog"));
const RejectDialog = lazy(() => import("@/modules/offers/components/RejectDialog"));
const ExtendDialog = lazy(() => import("@/modules/offers/components/ExtendDialog"));
const PdfPreviewModal = lazy(() => import("@/modules/offers/components/PdfPreviewModal"));
const CompareDrawer = lazy(() => import("@/modules/offers/components/CompareDrawer"));

const NO_REVISIONS: QuotationRevision[] = [];
const ACCEPTANCE_FILES = ["image/jpeg", "image/png", "image/webp", "application/pdf"];

const JUMPS: Array<{ id: string; label: string }> = [
  { id: "info", label: "Info" },
  { id: "acara", label: "Acara" },
  { id: "vendor", label: "Vendor & venue" },
  { id: "rincian", label: "Rincian" },
  { id: "termin", label: "Termin" },
  { id: "sk", label: "S&K" },
  { id: "riwayat", label: "Riwayat" },
];

interface PrimaryAction {
  label: string;
  icon: ReactNode;
  run?: () => void;
  to?: string;
  busy?: boolean;
}

/**
 * Halaman Penawaran (DESAIN §6.2–6.3): one page in parts (info, acara, vendor & venue, rincian, termin, syarat &
 * ketentuan, riwayat) with one main action for the state the quotation is in. Read-only once accepted, rejected,
 * superseded or made into a project; the banner says which and links on.
 */
export default function QuotationPage() {
  const { quotationId = "" } = useParams<{ quotationId: string }>();
  const navigate = useNavigate();
  const canManage = useCan("quotation.manage");
  const canCreateProject = useCan("project.create");
  const canMargin = useCan("finance.margin.view");
  const detail = useQuotationStore((s) => (s.detailFor === quotationId ? s.detail : null));
  const fetchDetail = useQuotationStore((s) => s.fetchDetail);
  const revisions = useQuotationStore((s) => (s.revisionsFor === quotationId ? s.revisions : NO_REVISIONS));
  const fetchRevisions = useQuotationStore((s) => s.fetchRevisions);
  const createRevision = useQuotationStore((s) => s.createRevision);
  const fetchPdf = useQuotationStore((s) => s.fetchPdf);
  const shareRevision = useQuotationStore((s) => s.shareRevision);
  const downloadRevisionPdf = useQuotationStore((s) => s.downloadRevisionPdf);
  const setStatus = useQuotationStore((s) => s.setStatus);
  const copyQuotation = useQuotationStore((s) => s.copyQuotation);
  const createAlternative = useQuotationStore((s) => s.createAlternative);
  const deleteQuotation = useQuotationStore((s) => s.deleteQuotation);
  const deleteEvent = useQuotationStore((s) => s.deleteEvent);
  const deleteSupplier = useQuotationStore((s) => s.deleteSupplier);
  const uploadAcceptance = useQuotationStore((s) => s.uploadAcceptance);
  const downloadAcceptance = useQuotationStore((s) => s.downloadAcceptance);

  const [loadState, setLoadState] = useState<"loading" | "error" | "missing" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [revisionsStatus, setRevisionsStatus] = useState<"loading" | "error" | "ready">("loading");
  const [dialog, setDialog] = useState<QuotationDialog | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let cancelled = false;
    setLoadState("loading");
    fetchDetail(quotationId).then(
      () => !cancelled && setLoadState("ready"),
      (err) => !cancelled && setLoadState(axios.isAxiosError(err) && err.response?.status === 404 ? "missing" : "error"),
    );
    return () => {
      cancelled = true;
    };
  }, [fetchDetail, quotationId, attempt]);

  const status = detail?.quotation.status;
  const acceptedN = detail?.quotation.acceptedRevisionN;
  const loadRevisions = useCallback(() => {
    setRevisionsStatus("loading");
    fetchRevisions(quotationId).then(
      () => setRevisionsStatus("ready"),
      () => setRevisionsStatus("error"),
    );
  }, [fetchRevisions, quotationId]);
  // A status change marks a revision accepted (or not any more).
  useEffect(loadRevisions, [loadRevisions, status, acceptedN]);

  const events = useMemo(() => (detail ? rincianEventsFromDetail(detail) : []), [detail]);
  const sortedRevisions = useMemo(() => [...revisions].sort((a, b) => b.n - a.n), [revisions]);

  const open = useCallback((d: QuotationDialog) => {
    setConfirmError(null);
    setDialog(d);
  }, []);
  const close = useCallback(() => setDialog(null), []);

  if (!detail) {
    if (loadState === "missing") {
      return (
        <div className="flex flex-col gap-4">
          <Breadcrumbs items={[{ label: "Penawaran", to: ROUTE_PATHS.quotations }, { label: "Tidak ditemukan" }]} />
          <LoadError message="Penawaran ini tidak ada atau bukan bagian Anda." onRetry={() => navigate(ROUTE_PATHS.quotations)} />
        </div>
      );
    }
    return (
      <div className="flex flex-col gap-4">
        <Breadcrumbs items={[{ label: "Penawaran", to: ROUTE_PATHS.quotations }, { label: "Memuat" }]} />
        {loadState === "error" ? (
          <LoadError message="Penawaran tidak bisa dimuat." onRetry={() => setAttempt((n) => n + 1)} />
        ) : (
          <div role="status" aria-label="Memuat penawaran" className="flex flex-col gap-4">
            <Skeleton className="h-9 w-2/5" />
            <Skeleton className="h-5 w-3/5" />
            <SkeletonRows rows={5} label="Memuat penawaran" />
          </div>
        )}
      </div>
    );
  }

  const d: QuotationDetail = detail;
  const q = d.quotation;
  const converted = Boolean(q.convertedProjectId) && !q.projectDeleted;
  const canEdit = canManage && q.editable;
  const matching = d.matchingRevisionN;
  const latest = sortedRevisions[0]?.n ?? d.latestRevisionN;
  const pdfN = q.acceptedRevisionN ?? matching ?? latest ?? null;

  async function act(key: string, run: () => Promise<void>, fallback: string) {
    setBusy(key);
    try {
      await run();
    } catch (err) {
      toast.error(getApiErrorMessage(err, fallback));
    } finally {
      setBusy(null);
    }
  }

  async function makeRevision() {
    try {
      const res = await createRevision(q.id);
      if (res.created) toast.success(res.message || `Revisi ${res.revision.n} dibuat`);
      else toast.info(res.message || `Tidak ada perubahan sejak Revisi ${res.revision.n}`);
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Revisi belum bisa dibuat. Periksa koneksi lalu coba lagi."));
      throw err;
    }
  }

  async function share(n: number) {
    const name = `${q.number}-Rev${n}.pdf`;
    const text = `Halo ${q.coupleNames}, berikut penawaran ${q.number} Revisi ${n} dari kami.`;
    const blob = await fetchPdf(API.quotations.revisionPdf(q.id, n));
    const file = new File([blob], name, { type: "application/pdf" });
    // A phone: the device's share sheet with the file (WhatsApp is in it).
    if (typeof navigator.canShare === "function" && navigator.canShare({ files: [file] })) {
      try {
        await navigator.share({ files: [file], title: name, text });
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return;
        throw err;
      }
    } else {
      // A computer: the PDF is saved, and WhatsApp Web opens to the client with the message ready.
      const url = URL.createObjectURL(file);
      const link = document.createElement("a");
      link.href = url;
      link.download = name;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 10_000);
      const wa = waNumber(q.contactWhatsapp);
      const opened = wa ? window.open(`https://wa.me/${wa}?text=${encodeURIComponent(`${text} PDF-nya kami lampirkan.`)}`, "_blank", "noopener") : null;
      if (!opened) toast.info("PDF sudah diunduh. Lampirkan di chat WhatsApp dengan klien.");
    }
    await shareRevision(q.id, n);
    toast.success("PDF siap dibagikan");
  }

  const doShare = (n: number) => void act(`share-${n}`, () => share(n), "PDF belum bisa dibagikan. Periksa koneksi lalu coba lagi.");
  const doDownload = (n: number) => void act(`download-${n}`, () => downloadRevisionPdf(q.id, n, q.number), "PDF belum bisa diunduh. Coba lagi.");

  async function duplicate(kind: "copy" | "alternative") {
    const made = kind === "copy" ? await copyQuotation(q.id) : await createAlternative(q.id);
    toast.success(kind === "copy" ? `Penawaran ${made.quotation.number} dibuat` : `Alternatif ${made.quotation.number} dibuat`);
    navigate(ROUTE_PATHS.quotationDetail(made.quotation.id));
  }

  async function reopen() {
    await setStatus(q.id, { status: "Draft" });
    toast.success("Penawaran dibuka lagi");
  }

  async function confirmed(run: () => Promise<void>, fallback: string) {
    setBusy("confirm");
    setConfirmError(null);
    try {
      await run();
      setDialog(null);
    } catch (err) {
      setConfirmError(getApiErrorMessage(err, fallback));
    } finally {
      setBusy(null);
    }
  }

  function pickAcceptance(file: File | undefined) {
    if (!file) return;
    if (!ACCEPTANCE_FILES.includes(file.type)) {
      toast.error("Pilih foto (JPG, PNG, WEBP) atau PDF");
      return;
    }
    if (file.size > 10 * 1024 * 1024) {
      toast.error("Ukuran berkas paling besar 10 MB");
      return;
    }
    void act(
      "attachment",
      async () => {
        await uploadAcceptance(q.id, file, q.updatedAt);
        toast.success("Lampiran diunggah");
      },
      "Lampiran belum terunggah. Coba lagi.",
    );
  }

  /* --------------------------------------------------------------------------------- one main action per state --- */
  const previewItem: ActionMenuItem = {
    label: "Pratinjau PDF",
    icon: <Eye className="h-4 w-4" />,
    onSelect: () => open({ kind: "pdf", title: `Pratinjau ${q.number}`, description: "Bertanda DRAF dan tidak membuat revisi.", url: API.quotations.previewPdf(q.id) }),
  };
  const downloadItem: ActionMenuItem = {
    label: pdfN ? `Unduh PDF Revisi ${pdfN}` : "Unduh PDF",
    icon: <Download className="h-4 w-4" />,
    onSelect: () => pdfN && doDownload(pdfN),
    hidden: !pdfN,
  };
  const copyItem: ActionMenuItem = {
    label: "Salin jadi penawaran baru",
    icon: <Copy className="h-4 w-4" />,
    onSelect: () => void act("copy", () => duplicate("copy"), "Penawaran belum bisa disalin. Coba lagi."),
    hidden: !canManage,
  };
  const alternativeItem: ActionMenuItem = {
    label: "Buat alternatif",
    icon: <GitBranch className="h-4 w-4" />,
    onSelect: () => void act("alternative", () => duplicate("alternative"), "Alternatif belum bisa dibuat. Coba lagi."),
    hidden: !canManage || Boolean(q.alternativeOfId),
  };
  const rejectItem: ActionMenuItem = { label: "Tolak", icon: <X className="h-4 w-4" />, onSelect: () => open({ kind: "reject" }), hidden: !canManage };

  let primary: PrimaryAction | null = null;
  let secondary: PrimaryAction | null = null;
  let menu: ActionMenuItem[] = [];
  const s = q.displayStatus;
  if (s === "Draft") {
    if (!matching) {
      primary = canManage ? { label: "Buat revisi untuk klien", icon: <FilePlus2 className="h-4 w-4" />, run: () => void act("revision", () => makeRevision().catch(() => undefined), ""), busy: busy === "revision" } : null;
      secondary = { label: "Pratinjau PDF", icon: <Eye className="h-4 w-4" />, run: previewItem.onSelect };
      menu = [
        { label: "Tandai diterima", icon: <Check className="h-4 w-4" />, onSelect: () => open({ kind: "accept" }), hidden: !canManage },
        alternativeItem,
        copyItem,
        downloadItem,
        rejectItem,
        {
          label: "Hapus draf",
          icon: <Trash2 className="h-4 w-4" />,
          tone: "danger",
          onSelect: () => open({ kind: "delete" }),
          hidden: !canManage || q.revisionCount > 0 || d.group.length > 1,
        },
      ];
    } else {
      primary = canManage
        ? { label: "Bagikan PDF", icon: <Share2 className="h-4 w-4" />, run: () => doShare(matching), busy: busy === `share-${matching}` }
        : { label: `Unduh PDF Revisi ${matching}`, icon: <Download className="h-4 w-4" />, run: () => doDownload(matching), busy: busy === `download-${matching}` };
      secondary = canManage ? { label: "Tandai diterima", icon: <Check className="h-4 w-4" />, run: () => open({ kind: "accept" }) } : null;
      menu = [{ ...downloadItem, hidden: !canManage }, previewItem, alternativeItem, copyItem, rejectItem];
    }
  } else if (s === "Expired") {
    primary = canManage ? { label: "Perpanjang masa berlaku", icon: <CalendarPlus className="h-4 w-4" />, run: () => open({ kind: "extend" }) } : null;
    secondary = canManage ? { label: "Tandai diterima", icon: <Check className="h-4 w-4" />, run: () => open({ kind: "accept" }) } : null;
    menu = [copyItem, downloadItem, previewItem, rejectItem];
  } else if (s === "Accepted") {
    if (converted) {
      primary = { label: "Buka proyek", icon: <FolderOpen className="h-4 w-4" />, to: ROUTE_PATHS.projectDetail(q.convertedProjectId) };
      menu = [downloadItem, copyItem];
    } else {
      primary = canManage && canCreateProject ? { label: "Jadikan proyek", icon: <ArrowRight className="h-4 w-4" />, to: ROUTE_PATHS.quotationConvert(q.id) } : null;
      menu = [
        { label: "Batalkan terima", icon: <Undo2 className="h-4 w-4" />, onSelect: () => open({ kind: "unaccept" }), hidden: !canManage },
        downloadItem,
        copyItem,
      ];
    }
  } else if (s === "Rejected") {
    primary = canManage ? { label: "Buka lagi", icon: <RotateCcw className="h-4 w-4" />, run: () => void act("reopen", reopen, "Penawaran belum bisa dibuka lagi. Coba lagi."), busy: busy === "reopen" } : null;
    menu = [copyItem, downloadItem];
  } else {
    menu = [downloadItem, copyItem];
  }
  const visibleMenu = menu.filter((m) => !m.hidden);

  const supersededBy = d.group.find((m) => m.id === q.supersededById);
  const alternativeOf = d.group.find((m) => m.id === q.alternativeOfId);
  const others = d.group.filter((m) => !m.isCurrent);

  const primaryButton = (p: PrimaryAction, variant: "primary" | "secondary" = "primary", className?: string) =>
    p.to ? (
      <Button variant={variant} className={className} icon={p.icon} onClick={() => navigate(p.to as string)}>
        {p.label}
      </Button>
    ) : (
      <Button variant={variant} className={className} icon={p.icon} onClick={p.run} disabled={p.busy}>
        {p.busy ? "Memproses..." : p.label}
      </Button>
    );

  return (
    <div className="flex flex-col gap-5 pb-20 md:pb-0">
      <Breadcrumbs items={[{ label: "Penawaran", to: ROUTE_PATHS.quotations }, { label: q.number }]} />

      <header className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="text-[13px] font-medium text-muted-foreground tabular-nums">{q.number}</p>
          <h1 className="mt-0.5 text-[22px] leading-tight font-semibold text-foreground">{q.coupleNames}</h1>
          <p className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px] text-text-secondary">
            <QuotationStatusBadge status={q.displayStatus} />
            <span>{revisionWord(q.acceptedRevisionN ?? matching ?? latest)}</span>
            <span aria-hidden>·</span>
            <span>{validityText(q)}</span>
            {d.totals && (
              <>
                <span aria-hidden>·</span>
                <span className="tabular-nums">Nilai kontrak {formatCurrency(d.totals.total)}</span>
              </>
            )}
          </p>
        </div>
        <div className="hidden flex-wrap items-center gap-2 md:flex">
          {secondary && primaryButton(secondary, "secondary")}
          {primary && primaryButton(primary)}
          {visibleMenu.length > 0 && <ActionMenu items={visibleMenu} label="Aksi lain penawaran" />}
        </div>
      </header>

      <div className="flex flex-col gap-3">
        {s === "Draft" && d.hasUnsentChanges && d.latestRevisionN && (
          <Banner tone="warning" icon={<AlertTriangle />}>
            Ada perubahan yang belum dikirim ke klien (sejak Revisi {d.latestRevisionN}).
            <Button size="sm" variant="ghost" icon={<GitCompare className="h-3.5 w-3.5" />} onClick={() => open({ kind: "compare", from: d.latestRevisionN ?? 1, to: 0 })}>
              Lihat perubahan
            </Button>
          </Banner>
        )}
        {s === "Expired" && (
          <Banner tone="warning" icon={<AlertTriangle />}>
            Masa berlaku lewat sejak {formatDate(q.validUntil)}. Perpanjang dulu untuk mengubah isinya; menandai diterima tetap bisa dengan perpanjangan.
          </Banner>
        )}
        {s === "Accepted" && (
          <Banner tone="success" icon={<Check />}>
            Diterima {formatDate(q.acceptedOn)}
            {q.acceptedChannel ? ` lewat ${ACCEPT_CHANNEL_LABELS[q.acceptedChannel]}` : ""} · Revisi {q.acceptedRevisionN}.
            {q.hasAcceptanceAttachment ? (
              <Button
                size="sm"
                variant="ghost"
                icon={<Paperclip className="h-3.5 w-3.5" />}
                onClick={() => void act("attachment", () => downloadAcceptance(q.id, q.acceptanceAttachmentName), "Lampiran belum bisa diunduh. Coba lagi.")}
              >
                {q.acceptanceAttachmentName || "Lampiran"}
              </Button>
            ) : (
              canManage && (
                <>
                  <input ref={fileRef} type="file" accept={ACCEPTANCE_FILES.join(",")} className="sr-only" tabIndex={-1} aria-hidden onChange={(e) => pickAcceptance(e.target.files?.[0])} />
                  <Button size="sm" variant="ghost" icon={<Upload className="h-3.5 w-3.5" />} onClick={() => fileRef.current?.click()} disabled={busy === "attachment"}>
                    {busy === "attachment" ? "Mengunggah..." : "Unggah lampiran"}
                  </Button>
                </>
              )
            )}
          </Banner>
        )}
        {converted && (
          <Banner tone="info" icon={<FolderOpen />}>
            Sudah dijadikan proyek{q.convertedAt ? ` pada ${formatDate(q.convertedAt)}` : ""}.{" "}
            <Link to={ROUTE_PATHS.projectDetail(q.convertedProjectId)} className="font-semibold underline underline-offset-2">
              Buka proyek {q.coupleNames}
            </Link>
          </Banner>
        )}
        {q.projectDeleted && (
          <Banner tone="neutral" icon={<Info />}>
            Proyek yang dibuat dari penawaran ini sudah dihapus. Penawaran bisa dijadikan proyek lagi.
          </Banner>
        )}
        {s === "Rejected" && (
          <Banner tone="danger" icon={<X />}>
            Ditolak{q.rejectedAt ? ` ${formatDate(q.rejectedAt)}` : ""}
            {q.rejectedReason ? `: ${q.rejectedReason}` : "."}
          </Banner>
        )}
        {s === "Superseded" && (
          <Banner tone="neutral" icon={<Info />}>
            Digantikan oleh{" "}
            {supersededBy ? (
              <Link to={ROUTE_PATHS.quotationDetail(supersededBy.id)} className="font-semibold underline underline-offset-2">
                {supersededBy.number}
              </Link>
            ) : q.supersededById ? (
              <Link to={ROUTE_PATHS.quotationDetail(q.supersededById)} className="font-semibold underline underline-offset-2">
                penawaran lain
              </Link>
            ) : (
              "penawaran lain"
            )}
            , yang diterima klien.
          </Banner>
        )}
        {others.length > 0 && (
          <div className="flex flex-col gap-1.5 rounded-md border border-border px-3.5 py-2.5 text-[13px]">
            <p className="font-medium text-foreground">
              {alternativeOf ? (
                <>
                  Alternatif dari{" "}
                  <Link to={ROUTE_PATHS.quotationDetail(alternativeOf.id)} className="underline underline-offset-2">
                    {alternativeOf.number}
                  </Link>
                </>
              ) : (
                `${others.length} alternatif`
              )}
            </p>
            <ul className="flex flex-col gap-1">
              {others.map((m) => (
                <li key={m.id} className="flex flex-wrap items-center gap-2 text-text-secondary">
                  <Link to={ROUTE_PATHS.quotationDetail(m.id)} className="font-medium text-foreground tabular-nums underline-offset-2 hover:underline">
                    {m.number}
                  </Link>
                  <QuotationStatusBadge status={m.displayStatus} />
                  <span className="tabular-nums">{formatCurrency(m.contractTotal)}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <nav aria-label="Bagian penawaran" className="sticky top-16 z-10 -mx-1 flex gap-1 overflow-x-auto border-b border-border bg-background/95 px-1 py-1.5 backdrop-blur">
        {JUMPS.map((j) => (
          <a
            key={j.id}
            href={`#${j.id}`}
            onClick={(e) => {
              e.preventDefault();
              const el = document.getElementById(j.id);
              const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
              el?.scrollIntoView({ behavior: reduce ? "auto" : "smooth", block: "start" });
            }}
            className="shrink-0 rounded-md px-2.5 py-1.5 text-[13px] font-medium text-text-secondary hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
          >
            {j.label}
          </a>
        ))}
      </nav>

      <InfoSection detail={d} canEdit={canEdit} open={open} />
      <EventsSection detail={d} events={events} canEdit={canEdit} open={open} />
      <SuppliersSection detail={d} canEdit={canEdit} canMargin={canMargin} open={open} />
      <QuotationRincianSection detail={d} canEdit={canEdit} canMargin={canMargin} onEditTerms={() => open({ kind: "terms" })} />
      <TermsSection detail={d} canEdit={canEdit} open={open} />
      <TermsTextSection detail={d} canEdit={canEdit} open={open} />
      <QuotationHistorySection
        detail={d}
        revisions={sortedRevisions}
        revisionsStatus={revisionsStatus}
        onRetryRevisions={loadRevisions}
        canShare={canManage && (s === "Draft" || s === "Expired" || s === "Accepted")}
        open={open}
        onShare={doShare}
        onDownload={doDownload}
      />

      {/* On a phone the main action stays at hand at the bottom (DESAIN §6.2). */}
      {(primary || visibleMenu.length > 0) && (
        <div className="fixed inset-x-0 bottom-0 z-20 flex items-center gap-2 border-t border-border bg-background/95 px-4 py-3 backdrop-blur md:hidden">
          {primary && primaryButton(primary, "primary", "flex-1")}
          {secondary && !primary && primaryButton(secondary, "secondary", "flex-1")}
          {visibleMenu.length > 0 && (
            <ActionMenu
              items={secondary && primary ? [{ label: secondary.label, icon: secondary.icon, onSelect: () => (secondary?.to ? navigate(secondary.to) : secondary?.run?.()) }, ...visibleMenu] : visibleMenu}
              label="Aksi lain penawaran"
            />
          )}
        </div>
      )}

      <Suspense fallback={null}>
        {dialog?.kind === "header" && <QuotationHeaderModal quotation={q} onClose={close} />}
        {dialog?.kind === "event" && <QuotationEventDrawer quotationId={q.id} events={events} event={dialog.event} onClose={close} />}
        {dialog?.kind === "supplier" && <QuotationSupplierModal quotationId={q.id} suppliers={d.suppliers} supplier={dialog.supplier} canMargin={canMargin} onClose={close} />}
        {dialog?.kind === "terms" && <QuotationTermsDrawer detail={d} onClose={close} />}
        {dialog?.kind === "termsText" && <QuotationTermsTextModal detail={d} onClose={close} />}
        {dialog?.kind === "accept" && <AcceptDialog detail={d} revisions={sortedRevisions} onClose={close} onCreateRevision={makeRevision} />}
        {dialog?.kind === "reject" && <RejectDialog quotation={q} onClose={close} />}
        {dialog?.kind === "extend" && <ExtendDialog quotation={q} onClose={close} />}
        {dialog?.kind === "pdf" && (
          <PdfPreviewModal
            title={dialog.title}
            description={dialog.description}
            url={dialog.url}
            onClose={close}
            onDownload={dialog.revisionN ? () => doDownload(dialog.revisionN as number) : undefined}
          />
        )}
        {dialog?.kind === "compare" && <CompareDrawer quotationId={q.id} revisions={[...sortedRevisions].reverse()} from={dialog.from} to={dialog.to} onClose={close} />}
      </Suspense>

      <ConfirmDialog
        open={dialog?.kind === "deleteEvent"}
        title={dialog?.kind === "deleteEvent" ? `Hapus acara ${dialog.event.kindName}?` : "Hapus acara?"}
        confirmLabel="Hapus acara"
        tone="danger"
        isSubmitting={busy === "confirm"}
        error={confirmError}
        onConfirm={() =>
          dialog?.kind === "deleteEvent" &&
          void confirmed(async () => {
            await deleteEvent(q.id, dialog.event.id);
            toast.success("Acara dihapus");
          }, "Acara belum bisa dihapus. Coba lagi.")
        }
        onClose={close}
      >
        <p>Sesi dan tamunya ikut terhapus. Baris rincian yang terikat ke acara ini dilepas dari acaranya.</p>
      </ConfirmDialog>

      <ConfirmDialog
        open={dialog?.kind === "deleteSupplier"}
        title={dialog?.kind === "deleteSupplier" ? `Hapus ${dialog.supplier.name} dari penawaran?` : "Hapus vendor?"}
        confirmLabel="Hapus dari penawaran"
        tone="danger"
        isSubmitting={busy === "confirm"}
        error={confirmError}
        onConfirm={() =>
          dialog?.kind === "deleteSupplier" &&
          void confirmed(async () => {
            await deleteSupplier(q.id, dialog.supplier.id);
            toast.success("Vendor dihapus dari penawaran");
          }, "Vendor belum bisa dihapus. Coba lagi.")
        }
        onClose={close}
      >
        <p>Vendor tetap ada di direktori. Baris rincian yang menunjuk vendor ini dilepas darinya.</p>
      </ConfirmDialog>

      <ConfirmDialog
        open={dialog?.kind === "unaccept"}
        title="Batalkan penerimaan?"
        confirmLabel="Batalkan terima"
        tone="primary"
        isSubmitting={busy === "confirm"}
        error={confirmError}
        onConfirm={() =>
          void confirmed(async () => {
            await setStatus(q.id, { status: "Draft" });
            toast.success("Penerimaan dibatalkan");
          }, "Penerimaan belum bisa dibatalkan. Coba lagi.")
        }
        onClose={close}
      >
        <p>Penawaran kembali menjadi Draf dan bisa diubah lagi. Alternatif yang menjadi Digantikan tidak ikut kembali.</p>
      </ConfirmDialog>

      <ConfirmDialog
        open={dialog?.kind === "delete"}
        title={`Hapus draf ${q.number}?`}
        confirmLabel="Hapus draf"
        tone="danger"
        isSubmitting={busy === "confirm"}
        error={confirmError}
        onConfirm={() =>
          void confirmed(async () => {
            await deleteQuotation(q.id);
            toast.success("Draf penawaran dihapus");
            navigate(ROUTE_PATHS.quotations);
          }, "Draf belum bisa dihapus. Coba lagi.")
        }
        onClose={close}
      >
        <p>Draf ini belum pernah dikirim ke klien. Isinya dihapus permanen dan tidak bisa dikembalikan.</p>
      </ConfirmDialog>
    </div>
  );
}

const BANNER: Record<"warning" | "success" | "info" | "danger" | "neutral", string> = {
  warning: "border-warning/30 bg-warning-soft text-warning-text",
  success: "border-success/30 bg-success-soft text-success-text",
  info: "border-info/30 bg-info-soft text-info-text",
  danger: "border-danger/30 bg-danger-soft text-danger-text",
  neutral: "border-border bg-muted text-text-secondary",
};

function Banner({ tone, icon, children }: { tone: keyof typeof BANNER; icon: ReactNode; children: ReactNode }) {
  return (
    <div role="status" className={cn("flex items-start gap-2 rounded-md border px-3.5 py-2.5 text-[13px] [&>svg]:mt-0.5 [&>svg]:h-4 [&>svg]:w-4 [&>svg]:shrink-0", BANNER[tone])}>
      {icon}
      <div className="flex flex-1 flex-wrap items-center gap-x-2 gap-y-1">{children}</div>
    </div>
  );
}
