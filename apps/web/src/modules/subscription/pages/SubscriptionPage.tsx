import { useEffect, useRef, useState } from "react";
import { CheckCircle2, ShieldCheck, Clock, Lock, QrCode, ExternalLink } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { formatCurrency, formatDate, daysBetween } from "@/shared/lib/formatters";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { usePlatformAdminStore, type PaymentCharge } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { APP_NAME } from "@/shared/constants/brand";
import { TRANSACTION_STATUS_TONE, TRANSACTION_STATUS_LABEL, TRANSACTION_TYPE_LABEL } from "@/modules/platform-admin/lib/status";
import { getApiErrorMessage } from "@/shared/lib/api-error";

const POLL_INTERVAL_MS = 4000;

const RENEWAL_WARNING_DAYS = 30;

function today(): string {
  return new Date().toISOString().slice(0, 10);
}

export default function SubscriptionPage() {
  const isOwner = useAuthStore((s) => s.session?.role === "Owner");

  const plans = useSubscriptionPlanStore((s) => s.plans);
  const fetchPlans = useSubscriptionPlanStore((s) => s.fetchPlans);
  const tenant = usePlatformAdminStore((s) => s.myTenant);
  const fetchMyTenant = usePlatformAdminStore((s) => s.fetchMyTenant);
  const transactions = usePlatformAdminStore((s) => s.transactionPage);
  const meta = usePlatformAdminStore((s) => s.transactionPageMeta);
  const fetchTransactionPage = usePlatformAdminStore((s) => s.fetchTransactionPage);
  const paySubscription = usePlatformAdminStore((s) => s.paySubscription);

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [activeCharge, setActiveCharge] = useState<PaymentCharge | null>(null);
  const [chargeOutcome, setChargeOutcome] = useState<"paid" | "expired" | null>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (!isOwner) return;
    void fetchPlans();
    void fetchMyTenant();
  }, [isOwner, fetchPlans, fetchMyTenant]);

  useEffect(() => {
    if (!isOwner) return;
    void fetchTransactionPage(page, "");
  }, [isOwner, fetchTransactionPage, page]);

  const plan = plans.find((p) => p.id === tenant?.planId);
  const isActive = tenant?.subscriptionStatus === "active" || tenant?.subscriptionStatus === "expiring_soon";

  const daysLeft = isActive && tenant?.subscriptionExpiresAt ? daysBetween(today(), tenant.subscriptionExpiresAt) : null;
  const isExpired = tenant?.subscriptionStatus === "expired";
  const isExpiringSoon = tenant?.subscriptionStatus === "expiring_soon";
  const isHealthy = tenant?.subscriptionStatus === "active";

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
        } else {
          setChargeOutcome("expired");
        }
      })();
    }, POLL_INTERVAL_MS);
  }

  async function handleConfirmSubscribe() {
    if (!plan) return;
    setActionError(null);
    setIsSubmitting(true);
    try {
      const charge = await paySubscription(plan.id);
      setConfirmOpen(false);
      setActiveCharge(charge);
      setChargeOutcome(null);
      startPolling(charge.orderRef);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memproses pembayaran"));
    } finally {
      setIsSubmitting(false);
    }
  }

  function closeChargeModal() {
    stopPolling();
    setActiveCharge(null);
    setChargeOutcome(null);
    void fetchTransactionPage(page, "");
  }

  if (!isOwner) {
    return (
      <div className="flex flex-col gap-5">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Langganan</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola langganan aplikasi {APP_NAME} untuk tim WO Console.</p>
        </div>
        <Card className="max-w-md">
          <CardContent className="py-5">
            <EmptyState
              icon={<Lock className="h-8 w-8 text-text-secondary" />}
              title="Akses Dibatasi"
              description="Hanya akun Owner yang dapat mengakses dan mengelola langganan ElProof. Hubungi owner WO Anda untuk informasi lebih lanjut."
            />
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Langganan</h1>
        <p className="mt-1 text-[13px] text-text-secondary">Kelola langganan aplikasi {APP_NAME} untuk tim WO Console.</p>
      </div>

      {actionError && (
        <p className="max-w-md rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
          {actionError}
        </p>
      )}

      <Card className="max-w-md">
        <CardContent className="flex flex-col gap-5 py-5">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-3">
              <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-navy-900 text-[15px] font-bold text-white">
                EP
              </span>
              <div>
                <p className="text-[15px] font-bold text-text-primary">Langganan {APP_NAME}</p>
                <p className="text-[12.5px] text-text-secondary">{plan?.name ?? "Paket tidak ditemukan"}</p>
              </div>
            </div>
            {isHealthy && <Badge tone="success">Aktif</Badge>}
            {isExpiringSoon && <Badge tone="warning">Segera Berakhir</Badge>}
            {isExpired && <Badge tone="danger">Berakhir</Badge>}
            {tenant?.subscriptionStatus === "pending_payment" && <Badge tone="neutral">Belum Aktif</Badge>}
          </div>

          <div className="flex items-end gap-1.5 border-t border-border-light pt-4">
            <span className="text-[28px] font-bold leading-none tabular-nums text-navy-900">
              {formatCurrency(plan?.price ?? 0)}
            </span>
            <span className="pb-0.5 text-[13px] text-text-secondary">/ {plan?.durationMonths ?? 0} bulan</span>
          </div>

          {isActive && tenant?.subscriptionExpiresAt && (
            <p className="flex items-center gap-1.5 text-[12.5px] text-text-secondary">
              <Clock className="h-3.5 w-3.5 shrink-0" />
              {isExpired ? "Berakhir pada" : "Berlaku hingga"} {formatDate(tenant.subscriptionExpiresAt)}
              {isExpiringSoon && daysLeft !== null && daysLeft <= RENEWAL_WARNING_DAYS && (
                <span className="font-semibold text-warning"> · H-{daysLeft}</span>
              )}
            </p>
          )}

          <ul className="flex flex-col gap-2 border-t border-border-light pt-4">
            {(plan?.features ?? []).map((feature) => (
              <li key={feature} className="flex items-start gap-2 text-[13px] text-text-primary">
                <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-success" />
                {feature}
              </li>
            ))}
          </ul>

          <Button className="w-full" onClick={() => setConfirmOpen(true)}>
            {isActive ? "Perpanjang Langganan" : "Berlangganan Sekarang"}
          </Button>

          <p className="flex items-center gap-1.5 text-[11.5px] text-text-secondary">
            <ShieldCheck className="h-3.5 w-3.5 shrink-0" />
            Pembayaran diproses otomatis oleh sistem pembayaran — langganan aktif segera setelah pembayaran diterima.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Riwayat Transaksi" subtitle="Riwayat pembayaran langganan aplikasi ElProof, diproses otomatis oleh sistem pembayaran." />
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
                      <span className="font-semibold text-text-primary">{TRANSACTION_TYPE_LABEL[tx.type]}</span>
                      <Badge tone={TRANSACTION_STATUS_TONE[tx.status]}>{TRANSACTION_STATUS_LABEL[tx.status]}</Badge>
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <CardListField label="Metode" value={tx.paymentMethod} />
                      <CardListField label="No. Referensi" value={<span className="font-mono">{tx.paymentReference}</span>} />
                      <CardListField label="Nominal" value={formatCurrency(tx.amount)} />
                      <CardListField label="Dibuat" value={formatDate(tx.createdAt)} />
                      <CardListField label="Dibayar" value={tx.paidAt ? formatDate(tx.paidAt) : "-"} />
                    </div>
                  </>
                )}
              />
              <div className="hidden sm:block">
              <Table>
                <THead>
                  <TR>
                    <TH>Tipe</TH>
                    <TH>Metode</TH>
                    <TH>No. Referensi</TH>
                    <TH>Nominal</TH>
                    <TH>Dibuat</TH>
                    <TH>Status</TH>
                    <TH>Dibayar</TH>
                  </TR>
                </THead>
                <TBody>
                  {transactions.map((tx) => (
                    <TR key={tx.id}>
                      <TD className="font-semibold text-text-primary">{TRANSACTION_TYPE_LABEL[tx.type]}</TD>
                      <TD>{tx.paymentMethod}</TD>
                      <TD className="font-mono text-[12.5px]">{tx.paymentReference}</TD>
                      <TD className="tabular-nums">{formatCurrency(tx.amount)}</TD>
                      <TD>{formatDate(tx.createdAt)}</TD>
                      <TD>
                        <Badge tone={TRANSACTION_STATUS_TONE[tx.status]}>{TRANSACTION_STATUS_LABEL[tx.status]}</Badge>
                      </TD>
                      <TD>{tx.paidAt ? formatDate(tx.paidAt) : "-"}</TD>
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
        title={isActive ? "Perpanjang Langganan" : "Berlangganan ElProof"}
        description="Anda akan diarahkan ke halaman pembayaran untuk menyelesaikan transaksi."
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirmOpen(false)}>
              Batal
            </Button>
            <Button onClick={() => void handleConfirmSubscribe()} disabled={isSubmitting}>
              {isSubmitting ? "Memproses..." : "Bayar Sekarang"}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-3 text-[13px]">
          <div className="flex items-center justify-between">
            <span className="text-text-secondary">Paket</span>
            <span className="font-semibold text-text-primary">
              Langganan {APP_NAME} — {plan?.name ?? "-"}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-text-secondary">Biaya</span>
            <span className="font-semibold text-text-primary">{formatCurrency(plan?.price ?? 0)}</span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-text-secondary">Masa Aktif</span>
            <span className="font-semibold text-text-primary">{plan?.durationMonths ?? 0} bulan</span>
          </div>
        </div>
      </Modal>

      <Modal
        open={activeCharge !== null}
        onClose={closeChargeModal}
        title={
          chargeOutcome === "paid" ? "Pembayaran Berhasil" : chargeOutcome === "expired" ? "Pembayaran Kedaluwarsa" : "Selesaikan Pembayaran"
        }
        description={
          chargeOutcome === "paid"
            ? "Langganan Anda sudah aktif."
            : chargeOutcome === "expired"
              ? "Tagihan ini sudah kedaluwarsa — silakan buat tagihan baru."
              : "Scan kode QRIS di bawah menggunakan aplikasi e-wallet atau m-banking Anda."
        }
        size="sm"
        footer={<Button onClick={closeChargeModal}>{chargeOutcome ? "Selesai" : "Tutup"}</Button>}
      >
        {activeCharge && (
          <div className="flex flex-col items-center gap-4 text-center">
            {chargeOutcome === "paid" ? (
              <CheckCircle2 className="h-16 w-16 text-success" />
            ) : chargeOutcome === "expired" ? (
              <Clock className="h-16 w-16 text-danger" />
            ) : (
              <>
                {activeCharge.qrImageUrl ? (
                  <img src={activeCharge.qrImageUrl} alt="Kode QRIS" className="h-56 w-56 rounded-lg border border-border-light object-contain" />
                ) : (
                  <QrCode className="h-16 w-16 text-text-secondary" />
                )}
                {activeCharge.payCode && (
                  <p className="text-[13px]">
                    Kode Bayar: <span className="font-mono font-semibold text-text-primary">{activeCharge.payCode}</span>
                  </p>
                )}
                {activeCharge.checkoutUrl && (
                  <a
                    href={activeCharge.checkoutUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-1.5 text-[13px] font-semibold text-navy-900 hover:underline"
                  >
                    Buka Halaman Pembayaran <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                )}
                <p className="text-[12.5px] text-text-secondary">
                  Menunggu konfirmasi pembayaran otomatis — halaman ini akan memperbarui status sendiri.
                </p>
              </>
            )}
            <div className="w-full border-t border-border-light pt-3 text-[12.5px] text-text-secondary">
              <div className="flex items-center justify-between">
                <span>No. Referensi</span>
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
