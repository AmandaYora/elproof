import { useEffect, useRef, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { CheckCircle2, ShieldCheck, Clock, Lock, QrCode, ExternalLink, AlertTriangle, XCircle, Receipt, CircleAlert, TicketPercent } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { cn } from "@/shared/lib/cn";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { formatCurrency, formatDate, daysBetween, addMonths, todayISO } from "@/shared/lib/formatters";
import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { TenantStatusBadge } from "@/modules/platform-admin/components/TenantStatusBadge";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { usePlatformAdminStore, type PaymentCharge } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { APP_NAME } from "@/shared/constants/brand";
import { TRANSACTION_STATUS_TONE, TRANSACTION_STATUS_LABEL, TRANSACTION_TYPE_LABEL } from "@/modules/platform-admin/lib/status";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";
import { FieldLabel, Input } from "@/shared/components/ui/Input";
import type { VoucherQuote } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { AppLogoMark } from "@/shared/components/AppLogoMark";

const POLL_INTERVAL_MS = 4000;

const RENEWAL_WARNING_DAYS = 30;

// Local calendar day (WIB for the people using this), not the UTC day.
function today(): string {
  return todayISO();
}

/**
 * The paid period a payment buys, as the API will grant it: it starts where
 * current access ends (remaining paid time or trial is kept), or today when
 * access has already lapsed.
 */
function newPeriod(periodEnd: string | null, durationMonths: number): { start: string; end: string } {
  const now = today();
  const start = periodEnd && periodEnd > now ? periodEnd : now;
  return { start, end: addMonths(start, durationMonths) };
}

/** Time left on a QRIS code, as mm:ss. Calls onElapsed once when it reaches zero. */
function ChargeCountdown({ expiresAt, onElapsed }: { expiresAt: string; onElapsed: () => void }) {
  const [secondsLeft, setSecondsLeft] = useState(() => Math.max(0, Math.floor((new Date(expiresAt).getTime() - Date.now()) / 1000)));

  useEffect(() => {
    const id = setInterval(() => {
      setSecondsLeft(Math.max(0, Math.floor((new Date(expiresAt).getTime() - Date.now()) / 1000)));
    }, 1000);
    return () => clearInterval(id);
  }, [expiresAt]);

  useEffect(() => {
    if (secondsLeft === 0) onElapsed();
    // onElapsed only needs to fire when the time runs out.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [secondsLeft === 0]);

  const mm = String(Math.floor(secondsLeft / 60)).padStart(2, "0");
  const ss = String(secondsLeft % 60).padStart(2, "0");
  return (
    <p className={`text-[13px] ${secondsLeft <= 60 ? "font-semibold text-danger-text" : "text-text-secondary"}`}>
      Kode QRIS berlaku <span className="font-mono tabular-nums">{mm}:{ss}</span> lagi
    </p>
  );
}

export default function SubscriptionPage() {
  // Paying and billing history need subscription.manage (ADR-0026).
  const isOwner = useCan("subscription.manage");
  const businessName = useTenantBrandingStore((s) => s.businessName) ?? APP_NAME;

  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const tenant = usePlatformAdminStore((s) => s.myTenant);
  const fetchMyTenant = usePlatformAdminStore((s) => s.fetchMyTenant);
  const transactions = usePlatformAdminStore((s) => s.transactionPage);
  const meta = usePlatformAdminStore((s) => s.transactionPageMeta);
  const fetchTransactionPage = usePlatformAdminStore((s) => s.fetchTransactionPage);
  const paySubscription = usePlatformAdminStore((s) => s.paySubscription);
  const checkVoucher = usePlatformAdminStore((s) => s.checkVoucher);
  const fetchPendingCharge = usePlatformAdminStore((s) => s.fetchPendingCharge);
  const cancelPendingCharge = usePlatformAdminStore((s) => s.cancelPendingCharge);

  // The plan the Owner just clicked a card for — independent of
  // tenant.planId, since a tenant may have no plan yet (never subscribed) or
  // may click a *different* card than the one they currently have (upgrade/
  // switch), not just renew.
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [activeCharge, setActiveCharge] = useState<PaymentCharge | null>(null);
  const [chargeOutcome, setChargeOutcome] = useState<"paid" | "expired" | "cancelled" | null>(null);
  // The period the open charge will buy, fixed when the charge was created.
  const [chargePeriod, setChargePeriod] = useState<{ start: string; end: string } | null>(null);
  // The Owner's one pending self-service charge, if any — backs the "Lihat
  // QR" row action in Riwayat Transaksi (for its live QR/pay code/checkout
  // URL, which the transaction row itself doesn't carry) and gates the plan
  // cards' subscribe buttons. Not rendered as its own banner — the pending
  // transaction row already shows amount/date/status.
  const [pendingCharge, setPendingCharge] = useState<PaymentCharge | null>(null);
  const [confirmingCancel, setConfirmingCancel] = useState(false);
  const [isCancelling, setIsCancelling] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // Voucher in the "Bayar sekarang" dialog. Only a code the server accepted (quote) is sent with the payment; editing the
  // text drops it, so a code is always checked again before it counts.
  const [voucherCode, setVoucherCode] = useState("");
  const [voucherQuote, setVoucherQuote] = useState<VoucherQuote | null>(null);
  const [voucherError, setVoucherError] = useState<string | null>(null);
  const [isCheckingVoucher, setIsCheckingVoucher] = useState(false);
  // The check whose answer is awaited: an answer for an older code, another plan or a closed dialog is dropped.
  const voucherSeq = useRef(0);

  useEffect(() => {
    if (!isOwner) return;
    void fetchPlans();
    void fetchMyTenant();
    void fetchPendingCharge().then(setPendingCharge).catch(() => {});
  }, [isOwner, fetchPlans, fetchMyTenant, fetchPendingCharge]);

  useEffect(() => {
    if (!isOwner) return;
    void fetchTransactionPage(page, "");
  }, [isOwner, fetchTransactionPage, page]);

  const activePlans = plans.filter((p) => p.isActive);
  const selectedPlan = plans.find((p) => p.id === selectedPlanId);
  const lifecycle = tenant?.lifecycle;
  const isActive = tenant?.subscriptionStatus === "active" || tenant?.subscriptionStatus === "expiring_soon";
  // A pending charge whose live gateway status couldn't be fetched degrades
  // to just orderRef + amount (see GetPendingCharge on the backend) — channel
  // is only ever set by a real gateway response, so its absence is the
  // signal to hide QR-specific UI (there's nothing to show) while still
  // allowing "Batalkan".
  const isPendingChargeDegraded = pendingCharge !== null && !pendingCharge.channel;

  const periodEnd = lifecycle?.periodEndsAt ?? tenant?.subscriptionExpiresAt ?? null;
  const daysLeft = periodEnd ? daysBetween(today(), periodEnd) : null;
  const isExpiringSoon = tenant?.subscriptionStatus === "expiring_soon";
  // Label for the period line, by lifecycle state.
  const periodLabel = lifecycle?.status === "trialing"
    ? "Masa coba hingga"
    : isActive
      ? "Berlaku hingga"
      : "Berakhir pada";

  // Stop polling on unmount so a left-open tab doesn't keep hitting the API.
  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  function stopPolling() {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }

  // The charge just created is `pending` until the gateway's webhook
  // confirms it (Fase 9) — there is no synchronous "paid" response, so the
  // frontend polls the transaction history for this specific order ref
  // instead of waiting on the Pay call itself.
  function startPolling(orderRef: string) {
    stopPolling();
    pollRef.current = setInterval(() => {
      void (async () => {
        await fetchTransactionPage(1, "");
        const match = usePlatformAdminStore.getState().transactionPage.find((tx) => tx.paymentReference === orderRef);
        if (!match || match.status === "pending") return;
        stopPolling();
        if (match.status === "paid") {
          setChargeOutcome("paid");
          await fetchMyTenant();
          void useSubscriptionStatusStore.getState().fetch();
        } else if (match.status === "cancelled") {
          setChargeOutcome("cancelled");
        } else {
          setChargeOutcome("expired");
        }
      })();
    }, POLL_INTERVAL_MS);
  }

  function resetVoucher(code = "") {
    voucherSeq.current++;
    setVoucherCode(code);
    setVoucherQuote(null);
    setVoucherError(null);
    setIsCheckingVoucher(false);
  }

  function handleSelectPlan(planId: string) {
    setActionError(null);
    setSelectedPlanId(planId);
    resetVoucher(); // a code checked for one plan says nothing about another
    setConfirmOpen(true);
  }

  async function handleApplyVoucher() {
    if (!selectedPlan) return;
    const code = voucherCode.trim().toUpperCase();
    if (!code) {
      setVoucherError("Masukkan kode voucher.");
      return;
    }
    const seq = ++voucherSeq.current;
    const planId = selectedPlan.id;
    setVoucherError(null);
    setVoucherQuote(null);
    setIsCheckingVoucher(true);
    try {
      const quote = await checkVoucher(planId, code);
      if (seq !== voucherSeq.current) return;
      setVoucherCode(quote.voucherCode);
      setVoucherQuote(quote);
    } catch (err) {
      if (seq !== voucherSeq.current) return;
      setVoucherError(getApiFieldErrors(err).voucherCode ?? getApiErrorMessage(err, "Kode voucher belum bisa diperiksa. Periksa koneksi lalu coba lagi."));
    } finally {
      if (seq === voucherSeq.current) setIsCheckingVoucher(false);
    }
  }

  // The voucher counts only for the plan it was checked against.
  const appliedVoucher = voucherQuote && selectedPlan && String(voucherQuote.planId) === selectedPlan.id ? voucherQuote : null;

  async function handleConfirmSubscribe() {
    if (!selectedPlan) return;
    // A typed code that was never checked is not silently dropped: the Owner either uses it or clears it.
    if (voucherCode.trim() && !appliedVoucher) {
      setVoucherError("Tekan Pakai untuk memeriksa kode ini, atau kosongkan kolom kode voucher.");
      return;
    }
    setActionError(null);
    setIsSubmitting(true);
    try {
      const charge = await paySubscription(selectedPlan.id, appliedVoucher?.voucherCode);
      setConfirmOpen(false);
      setActiveCharge(charge);
      setPendingCharge(charge);
      setChargeOutcome(null);
      setChargePeriod(newPeriod(periodEnd, selectedPlan.durationMonths));
      startPolling(charge.orderRef);
    } catch (err) {
      // The server checks the code again: refused at the last second (quota just ran out), it says so on the voucher field
      // and the price goes back to normal, with the dialog still open.
      const refusal = appliedVoucher ? getApiFieldErrors(err).voucherCode : undefined;
      if (refusal) {
        voucherSeq.current++;
        setVoucherQuote(null);
        setVoucherError(refusal);
      } else {
        setActionError(getApiErrorMessage(err, "Gagal memproses pembayaran"));
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  function closeChargeModal() {
    stopPolling();
    setActiveCharge(null);
    setChargeOutcome(null);
    void fetchTransactionPage(page, "");
    // Re-check rather than assume: the charge may have just resolved (paid
    // or expired) while its modal was open, in which case there's nothing
    // pending anymore and the row action should disappear too.
    void fetchPendingCharge().then(setPendingCharge).catch(() => {});
  }

  // Re-opens the QR modal for the Owner's one pending charge — triggered
  // from its "Lihat QR" row action in Riwayat Transaksi (e.g. after
  // accidentally closing the modal in an earlier visit).
  function handleViewPendingCharge() {
    if (!pendingCharge) return;
    setActionError(null);
    setActiveCharge(pendingCharge);
    setChargeOutcome(null);
    setChargePeriod(null);
    startPolling(pendingCharge.orderRef);
  }

  async function handleCancelPendingCharge() {
    setActionError(null);
    setIsCancelling(true);
    try {
      await cancelPendingCharge();
      setPendingCharge(null);
      setConfirmingCancel(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membatalkan tagihan"));
    } finally {
      setIsCancelling(false);
    }
  }

  if (!isOwner) {
    return (
      <div className="flex flex-col gap-5">
        <PageHeader className="mb-1" title="Langganan" description={`Paket ${APP_NAME} untuk tim ${businessName}, pembayaran, dan kuitansinya.`} />
        <Card className="max-w-md">
          <CardContent className="py-5">
            <EmptyState
              icon={<Lock className="h-8 w-8 text-muted-foreground" />}
              title="Hanya untuk Owner"
              description={`Langganan ${APP_NAME} hanya bisa dilihat dan diatur oleh Owner. Hubungi Owner WO Anda bila perlu.`}
            />
          </CardContent>
        </Card>
      </div>
    );
  }

  // Shared row-level actions for the one "Menunggu Konfirmasi" (pending)
  // transaction row, if any — reused by both the mobile CardList and desktop
  // Table renderings below.
  function pendingRowActions(tx: (typeof transactions)[number]) {
    if (tx.status !== "pending") return null;
    return (
      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap gap-1.5">
          {!isPendingChargeDegraded && (
            <Button size="sm" onClick={handleViewPendingCharge}>
              Lihat QR
            </Button>
          )}
          <Button size="sm" variant="secondary" onClick={() => setConfirmingCancel(true)} disabled={isCancelling}>
            Batalkan
          </Button>
        </div>
        {confirmingCancel && (
          <div className="max-w-[240px] rounded-md border border-danger/30 bg-danger-soft px-3 py-2">
            <p className="flex items-start gap-1.5 text-[12px] font-medium text-danger-text">
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              Yakin ingin membatalkan tagihan ini?
            </p>
            <div className="mt-2 flex justify-end gap-2">
              <Button size="sm" variant="secondary" onClick={() => setConfirmingCancel(false)} disabled={isCancelling}>
                Batal
              </Button>
              <Button size="sm" variant="danger" onClick={() => void handleCancelPendingCharge()} disabled={isCancelling}>
                {isCancelling ? "Memproses..." : "Batalkan tagihan"}
              </Button>
            </div>
          </div>
        )}
      </div>
    );
  }

  // Paid payments have a printable receipt.
  function receiptLink(tx: (typeof transactions)[number]) {
    if (tx.status !== "paid") return null;
    return (
      <Link to={ROUTE_PATHS.subscriptionReceipt(tx.id)} className="inline-flex items-center gap-1.5 text-[13px] font-semibold text-primary hover:underline">
        <Receipt className="h-3.5 w-3.5" /> Kuitansi
      </Link>
    );
  }

  const currentPlan = plans.find((p) => p.id === tenant?.planId);
  const showDaysLeft =
    (isExpiringSoon || lifecycle?.status === "trialing") && daysLeft !== null && daysLeft >= 0 && daysLeft <= RENEWAL_WARNING_DAYS;

  return (
    <div className="flex flex-col gap-5">
      <PageHeader className="mb-1" title="Langganan" description={`Paket ${APP_NAME} untuk tim ${businessName}, pembayaran, dan kuitansinya.`} />

      {actionError && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {actionError}
        </p>
      )}

      {/* ≥ 1024 px: where the subscription stands on the left, the plans to choose from on the right. */}
      <div className="grid gap-5 lg:grid-cols-[300px_minmax(0,1fr)] lg:items-start lg:gap-8">
        <Card className="lg:sticky lg:top-20">
          <CardHeader title="Langganan Anda" />
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <span className="text-[12.5px] text-muted-foreground">Status</span>
              <div>{tenant ? <TenantStatusBadge status={tenant.subscriptionStatus} /> : <span className="text-[13px] text-muted-foreground">Memuat...</span>}</div>
            </div>
            <div className="flex flex-col gap-0.5">
              <span className="text-[12.5px] text-muted-foreground">Paket</span>
              <span className="text-[14px] font-medium text-foreground">
                {currentPlan?.name ?? (lifecycle?.status === "trialing" ? "Masa coba" : "Belum ada paket")}
              </span>
            </div>
            {periodEnd && (
              <div className="flex flex-col gap-0.5">
                <span className="text-[12.5px] text-muted-foreground">{periodLabel}</span>
                <span className="flex items-center gap-1.5 text-[14px] font-medium text-foreground">
                  <Clock aria-hidden className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  {formatDate(periodEnd)}
                </span>
                {showDaysLeft && <span className="text-[13px] font-semibold text-warning-text">Tersisa {daysLeft} hari</span>}
              </div>
            )}
            {pendingCharge && (
              <p className="rounded-md bg-muted px-3.5 py-2.5 text-[12.5px] text-muted-foreground">
                Ada pembayaran yang belum selesai. Selesaikan atau batalkan di riwayat transaksi sebelum memilih paket lagi.
              </p>
            )}
            <p className="flex items-start gap-1.5 border-t border-border-light pt-4 text-[12.5px] text-muted-foreground">
              <ShieldCheck aria-hidden className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              Pembayaran lewat QRIS diproses otomatis. Langganan aktif segera setelah pembayaran diterima.
            </p>
          </CardContent>
        </Card>

        <section aria-labelledby="pilih-paket" className="flex min-w-0 flex-col gap-3">
          <h2 id="pilih-paket" className="text-[16px] font-semibold text-foreground">
            Pilih paket
          </h2>
          {activePlans.length === 0 ? (
            <Card>
              <CardContent className="py-5">
                <EmptyState title="Belum ada paket tersedia" description={`Hubungi ${APP_NAME} untuk informasi paket langganan.`} />
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              {activePlans.map((p) => {
                const isCurrentPlan = p.id === tenant?.planId;
                // One plan: a wide card, price and button on the left and what it includes on the right.
                const wide = activePlans.length === 1;
                return (
                  <Card key={p.id} className={cn("flex flex-col", wide && "sm:col-span-2", isCurrentPlan && "border-primary ring-1 ring-ring")}>
                    <CardContent className={cn("flex flex-1 flex-col gap-4 py-5", wide && "md:grid md:grid-cols-2 md:grid-rows-[auto_auto_1fr] md:gap-x-10")}>
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex min-w-0 items-center gap-3">
                          <AppLogoMark className="size-10 shrink-0" />
                          <p className="text-[16px] font-semibold break-words text-foreground">{p.name}</p>
                        </div>
                        {isCurrentPlan && <Badge tone="success">Paket Anda</Badge>}
                      </div>

                      <div className="flex items-baseline gap-1.5">
                        <span className="text-[28px] leading-none font-bold tabular-nums text-foreground">{formatCurrency(p.price)}</span>
                        <span className="text-[13px] text-muted-foreground">/ {p.durationMonths} bulan</span>
                      </div>

                      <ul className={cn("flex flex-1 flex-col gap-2 border-t border-border-light pt-4", wide && "md:col-start-2 md:row-span-3 md:row-start-1 md:border-t-0 md:border-l md:pt-0 md:pl-10")}>
                        {p.features.map((feature) => (
                          <li key={feature} className="flex items-start gap-2 text-[13.5px] text-foreground">
                            <CheckCircle2 aria-hidden className="mt-0.5 h-4 w-4 shrink-0 text-success-text" />
                            {feature}
                          </li>
                        ))}
                      </ul>

                      {pendingCharge ? (
                        <p className="rounded-md bg-muted px-3.5 py-2.5 text-[12.5px] text-muted-foreground">Selesaikan pembayaran yang tertunda dulu.</p>
                      ) : (
                        <Button className={cn("w-full", wide && "md:self-end")} variant={isCurrentPlan ? "primary" : "secondary"} onClick={() => handleSelectPlan(p.id)}>
                          {isCurrentPlan ? "Perpanjang paket ini" : "Pilih paket ini"}
                        </Button>
                      )}
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </section>
      </div>

      <Card>
        <CardHeader title="Riwayat transaksi" subtitle={`Pembayaran langganan ${APP_NAME}. Pembayaran yang lunas punya kuitansi.`} />
        <CardContent className="p-0">
          {transactions.length === 0 ? (
            <EmptyState title="Belum ada riwayat" description="Riwayat transaksi akan muncul di sini setelah langganan pertama dibayar." />
          ) : (
            <>
              <CardList
                className="sm:hidden"
                items={transactions}
                keyFor={(tx) => tx.id}
                renderItem={(tx) => (
                  <>
                    <div className="flex items-start justify-between gap-3">
                      <span className="font-semibold text-foreground">{TRANSACTION_TYPE_LABEL[tx.type]}</span>
                      <Badge tone={TRANSACTION_STATUS_TONE[tx.status]}>{TRANSACTION_STATUS_LABEL[tx.status]}</Badge>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <CardListField label={tx.discountAmount > 0 ? "Nominal dibayar" : "Nominal"} value={formatCurrency(tx.amount)} />
                      {tx.discountAmount > 0 && (
                        <CardListField label={`Potongan voucher${tx.voucherCode ? ` ${tx.voucherCode}` : ""}`} value={`−${formatCurrency(tx.discountAmount)}`} />
                      )}
                      <CardListField label="Dibuat" value={formatDate(tx.createdAt)} />
                      <CardListField label="No. referensi" value={<span className="font-mono">{tx.paymentReference}</span>} />
                    </div>
                    {pendingRowActions(tx)}
                    {receiptLink(tx)}
                  </>
                )}
              />
              <div className="hidden sm:block">
                <Table>
                  <THead>
                    <TR>
                      <TH>Jenis</TH>
                      <TH>Metode</TH>
                      <TH>No. referensi</TH>
                      <TH className="text-right">Nominal</TH>
                      <TH>Dibuat</TH>
                      <TH>Status</TH>
                      <TH>Dibayar</TH>
                      <TH>
                        <span className="sr-only">Aksi</span>
                      </TH>
                    </TR>
                  </THead>
                  <TBody>
                    {transactions.map((tx) => (
                      <TR key={tx.id}>
                        <TD className="font-semibold text-foreground">{TRANSACTION_TYPE_LABEL[tx.type]}</TD>
                        <TD>{tx.paymentMethod}</TD>
                        <TD className="font-mono text-[12.5px] whitespace-nowrap">{tx.paymentReference}</TD>
                        <TD className="text-right whitespace-nowrap tabular-nums">
                          {formatCurrency(tx.amount)}
                          {tx.discountAmount > 0 && (
                            <span className="mt-0.5 block text-[12px] text-muted-foreground">
                              Harga awal {formatCurrency(tx.originalAmount)}
                              <br />
                              Voucher{tx.voucherCode ? ` ${tx.voucherCode}` : ""} −{formatCurrency(tx.discountAmount)}
                            </span>
                          )}
                        </TD>
                        <TD className="whitespace-nowrap">{formatDate(tx.createdAt)}</TD>
                        <TD className="whitespace-nowrap">
                          <Badge tone={TRANSACTION_STATUS_TONE[tx.status]}>{TRANSACTION_STATUS_LABEL[tx.status]}</Badge>
                        </TD>
                        <TD className="whitespace-nowrap">{tx.paidAt ? formatDate(tx.paidAt) : "-"}</TD>
                        <TD>{pendingRowActions(tx) ?? receiptLink(tx) ?? "-"}</TD>
                      </TR>
                    ))}
                  </TBody>
                </Table>
              </div>
              <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
            </>
          )}
        </CardContent>
      </Card>

      <Modal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title={selectedPlan && selectedPlan.id === tenant?.planId ? "Perpanjang langganan" : `Berlangganan ${APP_NAME}`}
        description="Setelah ini kode QRIS muncul untuk dibayar lewat aplikasi bank atau e-wallet. Masa aktif baru dimulai setelah masa berjalan Anda selesai."
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmOpen(false)}>
              Batal
            </Button>
            <Button onClick={() => void handleConfirmSubscribe()} disabled={isSubmitting}>
              {isSubmitting ? "Memproses..." : "Bayar sekarang"}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-3 text-[13px]">
          <div className="flex items-center justify-between">
            <span className="text-text-secondary">Paket</span>
            <span className="font-semibold text-text-primary">
              Langganan {APP_NAME} — {selectedPlan?.name ?? "-"}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-text-secondary">Masa aktif</span>
            <span className="font-semibold text-text-primary">{selectedPlan?.durationMonths ?? 0} bulan</span>
          </div>
          {selectedPlan && (
            <div className="flex items-start justify-between gap-4">
              <span className="text-text-secondary">Periode baru</span>
              <span className="text-right font-semibold text-text-primary">
                {formatDate(newPeriod(periodEnd, selectedPlan.durationMonths).start)} sampai {formatDate(newPeriod(periodEnd, selectedPlan.durationMonths).end)}
              </span>
            </div>
          )}

          <div className="flex flex-col gap-1.5 border-t border-border-light pt-3">
            <FieldLabel htmlFor="subscription-voucher-code">Kode voucher (opsional)</FieldLabel>
            <div className="flex gap-2">
              <Input
                id="subscription-voucher-code"
                data-no-autofocus
                value={voucherCode}
                onChange={(e) => resetVoucher(e.target.value.toUpperCase())}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    if (!appliedVoucher) void handleApplyVoucher();
                  }
                }}
                placeholder="cth. HEMAT10"
                autoComplete="off"
                spellCheck={false}
                maxLength={32}
                disabled={isSubmitting}
                aria-invalid={voucherError ? true : undefined}
                aria-describedby={voucherError ? "subscription-voucher-message" : undefined}
                className="font-mono uppercase placeholder:normal-case placeholder:font-sans"
              />
              {appliedVoucher ? (
                <Button variant="secondary" onClick={() => resetVoucher()} disabled={isSubmitting}>
                  Hapus
                </Button>
              ) : (
                <Button variant="secondary" onClick={() => void handleApplyVoucher()} disabled={isCheckingVoucher || isSubmitting}>
                  {isCheckingVoucher ? "Memeriksa..." : "Pakai"}
                </Button>
              )}
            </div>
            {voucherError && (
              <span id="subscription-voucher-message" role="alert" className="flex items-start gap-1 text-xs font-medium text-danger-text">
                <CircleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
                {voucherError}
              </span>
            )}
          </div>

          <div className="flex flex-col gap-2 rounded-md bg-surface-muted px-3.5 py-3" aria-live="polite">
            {appliedVoucher ? (
              <>
                <div className="flex items-center justify-between">
                  <span className="text-text-secondary">Harga awal</span>
                  <span className="tabular-nums text-text-primary">{formatCurrency(appliedVoucher.originalAmount)}</span>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <span className="flex items-center gap-1.5 text-text-secondary">
                    <TicketPercent aria-hidden className="h-3.5 w-3.5 shrink-0" />
                    Potongan <span className="font-mono text-[12px] font-semibold text-text-primary">{appliedVoucher.voucherCode}</span>
                  </span>
                  <span className="tabular-nums font-medium text-success-text">−{formatCurrency(appliedVoucher.discountAmount)}</span>
                </div>
                <div className="flex items-center justify-between border-t border-border-light pt-2">
                  <span className="font-semibold text-text-primary">Harga akhir</span>
                  <span className="text-[15px] font-bold tabular-nums text-text-primary">{formatCurrency(appliedVoucher.finalAmount)}</span>
                </div>
              </>
            ) : (
              <div className="flex items-center justify-between">
                <span className="font-semibold text-text-primary">Biaya</span>
                <span className="text-[15px] font-bold tabular-nums text-text-primary">{formatCurrency(selectedPlan?.price ?? 0)}</span>
              </div>
            )}
          </div>
        </div>
      </Modal>

      <Modal
        open={activeCharge !== null}
        onClose={closeChargeModal}
        title={
          chargeOutcome === "paid"
            ? "Pembayaran berhasil"
            : chargeOutcome === "expired"
              ? "Kode QRIS kedaluwarsa"
              : chargeOutcome === "cancelled"
                ? "Tagihan dibatalkan"
                : "Selesaikan pembayaran"
        }
        description={
          chargeOutcome === "paid"
            ? chargePeriod
              ? `Langganan Anda aktif sampai ${formatDate(chargePeriod.end)}.`
              : "Langganan Anda sudah aktif."
            : chargeOutcome === "expired"
              ? "Kode ini sudah melewati batas waktunya. Tutup jendela ini; jika tagihan masih tertunda di riwayat transaksi, batalkan lalu buat tagihan baru untuk mendapatkan kode QRIS baru."
              : chargeOutcome === "cancelled"
                ? "Tagihan ini dibatalkan dan tidak bisa dibayar lagi. Anda bisa memilih paket dan membuat tagihan baru kapan saja."
                : "Pindai kode QRIS di bawah dengan aplikasi e-wallet atau m-banking Anda."
        }
        size="sm"
        footer={<Button onClick={closeChargeModal}>{chargeOutcome ? "Selesai" : "Tutup"}</Button>}
      >
        {activeCharge && (
          <div className="flex flex-col items-center gap-4 text-center">
            {chargeOutcome === "paid" ? (
              <CheckCircle2 className="h-16 w-16 text-success-text" />
            ) : chargeOutcome === "expired" ? (
              <Clock className="h-16 w-16 text-danger-text" />
            ) : chargeOutcome === "cancelled" ? (
              <XCircle className="h-16 w-16 text-text-secondary" />
            ) : (
              <>
                {activeCharge.qrImageUrl ? (
                  <img src={activeCharge.qrImageUrl} alt="Kode QRIS" className="h-56 w-56 rounded-lg border border-border-light object-contain" />
                ) : (
                  <QrCode className="h-16 w-16 text-text-secondary" />
                )}
                {activeCharge.payCode && (
                  <p className="text-[13px]">
                    Kode bayar: <span className="font-mono font-semibold text-text-primary">{activeCharge.payCode}</span>
                  </p>
                )}
                {activeCharge.checkoutUrl && (
                  <a
                    href={activeCharge.checkoutUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-1.5 text-[13px] font-semibold text-primary hover:underline"
                  >
                    Buka halaman pembayaran <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                )}
                <ChargeCountdown expiresAt={activeCharge.expiresAt} onElapsed={() => setChargeOutcome((o) => o ?? "expired")} />
                {chargePeriod && (
                  <p className="text-[12.5px] text-text-secondary">
                    Setelah dibayar, langganan berlaku {formatDate(chargePeriod.start)} sampai {formatDate(chargePeriod.end)}.
                  </p>
                )}
                <p className="text-[12.5px] text-text-secondary">
                  Menunggu pembayaran. Halaman ini memperbarui statusnya sendiri, tidak perlu dimuat ulang.
                </p>
              </>
            )}
            <div className="w-full border-t border-border-light pt-3 text-[12.5px] text-text-secondary">
              <div className="flex items-center justify-between">
                <span>No. referensi</span>
                <span className="font-mono">{activeCharge.orderRef}</span>
              </div>
              <div className="flex items-center justify-between">
                <span>Nominal</span>
                <span className="font-semibold text-text-primary">{formatCurrency(activeCharge.amount)}</span>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
