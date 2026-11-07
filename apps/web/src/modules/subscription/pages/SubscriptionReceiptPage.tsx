import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Printer } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { APP_NAME } from "@/shared/constants/brand";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { formatCurrency, formatDateTime } from "@/shared/lib/formatters";
import { usePlatformAdminStore, toTransaction, type RawTransaction } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { TRANSACTION_TYPE_LABEL } from "@/modules/platform-admin/lib/status";
import type { SubscriptionTransaction } from "@/modules/platform-admin/data/types";

type LoadState = { kind: "loading" } | { kind: "failed" } | { kind: "missing" } | { kind: "ready"; tx: SubscriptionTransaction };

/**
 * Receipt for one paid subscription payment. "Simpan sebagai PDF" opens the
 * browser's print dialog (choose "Save as PDF" there), so no PDF library is
 * shipped; the rest of the console is hidden while printing.
 */
export default function SubscriptionReceiptPage() {
  const { id = "" } = useParams();
  const tenant = usePlatformAdminStore((s) => s.myTenant);
  const fetchMyTenant = usePlatformAdminStore((s) => s.fetchMyTenant);
  const [state, setState] = useState<LoadState>({ kind: "loading" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setState({ kind: "loading" });
    Promise.all([httpClient.get(API.billing.transactions, { params: { all: true } }), fetchMyTenant()])
      .then(([res]) => {
        if (cancelled) return;
        const tx = (res.data.data as RawTransaction[]).map(toTransaction).find((t) => t.id === id);
        setState(tx && tx.status === "paid" ? { kind: "ready", tx } : { kind: "missing" });
      })
      .catch(() => {
        if (!cancelled) setState({ kind: "failed" });
      });
    return () => {
      cancelled = true;
    };
  }, [id, attempt, fetchMyTenant]);

  if (state.kind === "failed") return <LoadError onRetry={() => setAttempt((n) => n + 1)} />;
  if (state.kind === "loading") return <div className="py-20 text-center text-sm text-text-secondary">Memuat kuitansi...</div>;
  if (state.kind === "missing") {
    return (
      <EmptyState
        title="Kuitansi tidak tersedia"
        description="Kuitansi hanya ada untuk pembayaran yang sudah berhasil. Pastikan tautannya benar."
        action={
          <Link to={ROUTE_PATHS.subscription} className="text-[13px] font-semibold text-navy-900 hover:underline">
            Kembali ke Langganan
          </Link>
        }
      />
    );
  }

  const { tx } = state;
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3 print:hidden">
        <Link to={ROUTE_PATHS.subscription} className="inline-flex items-center gap-1.5 text-[13px] font-medium text-text-secondary hover:text-text-primary">
          <ArrowLeft className="h-4 w-4" /> Kembali ke Langganan
        </Link>
        <Button icon={<Printer className="h-4 w-4" />} onClick={() => window.print()}>
          Simpan sebagai PDF
        </Button>
      </div>

      <article className="rounded-lg border border-border bg-white p-8 text-text-primary print:rounded-none print:border-0 print:p-0">
        <header className="flex items-start justify-between gap-4 border-b border-border pb-5">
          <div>
            <p className="text-lg font-bold">{APP_NAME}</p>
            <p className="text-[12.5px] text-text-secondary">Aplikasi manajemen wedding organizer</p>
          </div>
          <div className="text-right">
            <p className="text-lg font-bold">Kuitansi</p>
            <p className="font-mono text-[12.5px] text-text-secondary">{tx.paymentReference}</p>
          </div>
        </header>

        <dl className="mt-6 grid grid-cols-1 gap-x-8 gap-y-4 text-[13.5px] sm:grid-cols-2">
          <div>
            <dt className="text-[12.5px] text-text-secondary">Diterima dari</dt>
            <dd className="mt-0.5 font-semibold">{tenant?.businessName ?? "-"}</dd>
            {tenant && <dd className="text-[12.5px] text-text-secondary">{tenant.ownerName}, {tenant.email}</dd>}
          </div>
          <div>
            <dt className="text-[12.5px] text-text-secondary">Tanggal pembayaran</dt>
            <dd className="mt-0.5 font-semibold">{formatDateTime(tx.paidAt)} WIB</dd>
          </div>
          <div>
            <dt className="text-[12.5px] text-text-secondary">Untuk pembayaran</dt>
            <dd className="mt-0.5 font-semibold">Langganan {APP_NAME}, {TRANSACTION_TYPE_LABEL[tx.type].toLowerCase()}</dd>
          </div>
          <div>
            <dt className="text-[12.5px] text-text-secondary">Metode pembayaran</dt>
            <dd className="mt-0.5 font-semibold">{tx.paymentMethod}</dd>
          </div>
        </dl>

        <div className="mt-8 flex items-center justify-between rounded-md bg-surface-muted px-5 py-4 print:border print:border-border">
          <span className="text-[13.5px] font-semibold">Total dibayar</span>
          <span className="text-[22px] font-bold tabular-nums">{formatCurrency(tx.amount)}</span>
        </div>

        <p className="mt-6 text-[12.5px] leading-relaxed text-text-secondary">
          Pembayaran ini sudah diterima dan langganan telah diperpanjang. Kuitansi dibuat otomatis oleh sistem {APP_NAME} dari data pembayaran yang tercatat.
        </p>
      </article>
    </div>
  );
}
