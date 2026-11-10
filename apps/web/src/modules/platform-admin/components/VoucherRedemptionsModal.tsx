import { useEffect, useRef, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { formatCurrency, formatDateTime } from "@/shared/lib/formatters";
import { VOUCHER_REDEMPTION_STATUS_LABELS } from "@/shared/lib/labels";
import { VOUCHER_REDEMPTION_STATUS_TONE } from "@/modules/platform-admin/lib/status";
import { appLabel } from "@/shared/data/subscriptionPlans";
import { EMPTY_PAGINATION_META, type PaginationMeta } from "@/shared/types/pagination";
import { useVoucherStore, type Voucher, type VoucherRedemption } from "@/modules/platform-admin/stores/useVoucherStore";
import type { PaymentApp } from "@/modules/platform-admin/stores/useAppsStore";

interface VoucherRedemptionsModalProps {
  /** The voucher whose uses are shown; null = closed. */
  voucher: Voucher | null;
  onClose: () => void;
  /** Every registered App, to name an App customer (the API leaves its name empty). */
  apps: PaymentApp[];
}

type Load = { status: "loading" | "error" | "ready"; rows: VoucherRedemption[]; meta: PaginationMeta };

/** Who used a voucher, what they paid, and whether that payment went through. One page at a time, from the server. */
export function VoucherRedemptionsModal({ voucher, onClose, apps }: VoucherRedemptionsModalProps) {
  const fetchRedemptions = useVoucherStore((s) => s.fetchRedemptions);
  // Kept while the dialog animates closed, so its title does not blank out.
  const [shown, setShown] = useState<Voucher | null>(voucher);
  const [page, setPage] = useState(1);
  const [attempt, setAttempt] = useState(0);
  const [load, setLoad] = useState<Load>({ status: "loading", rows: [], meta: EMPTY_PAGINATION_META });
  // The request the dialog is waiting for: an answer for a voucher (or page) opened earlier is dropped.
  const latest = useRef("");

  useEffect(() => {
    if (voucher) {
      setShown(voucher);
      setPage(1);
    }
  }, [voucher?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const id = voucher?.id ?? null;
  useEffect(() => {
    if (id === null) return;
    const key = `${id}:${page}:${attempt}`;
    latest.current = key;
    setLoad((prev) => ({ status: "loading", rows: prev.rows, meta: prev.meta }));
    fetchRedemptions(id, page)
      .then(({ rows, meta }) => {
        if (latest.current === key) setLoad({ status: "ready", rows, meta });
      })
      .catch(() => {
        if (latest.current === key) setLoad({ status: "error", rows: [], meta: EMPTY_PAGINATION_META });
      });
  }, [id, page, attempt, fetchRedemptions]);

  // A dialog opened for another voucher never shows the previous one's rows, even for a moment.
  useEffect(() => {
    if (id !== null) setLoad({ status: "loading", rows: [], meta: EMPTY_PAGINATION_META });
  }, [id]);

  function customer(r: VoucherRedemption) {
    if (r.customerType === "app") return appLabel(apps.find((a) => a.appId === r.customerRef), r.customerRef);
    return r.customerName || `Tenant ${r.customerRef}`;
  }

  const columns: Column<VoucherRedemption>[] = [
    { id: "customer", header: "Pelanggan", primary: true, cell: (r) => <span className="font-semibold">{customer(r)}</span> },
    { id: "orderRef", header: "No. Referensi", cell: (r) => <span className="font-mono text-[12.5px]">{r.orderRef}</span> },
    { id: "original", header: "Harga awal", align: "right", cell: (r) => formatCurrency(r.originalAmount) },
    { id: "discount", header: "Potongan", align: "right", cell: (r) => `−${formatCurrency(r.discountAmount)}` },
    { id: "final", header: "Harga akhir", align: "right", cell: (r) => <span className="font-semibold">{formatCurrency(r.finalAmount)}</span> },
    {
      id: "status",
      header: "Status",
      cell: (r) => <Badge tone={VOUCHER_REDEMPTION_STATUS_TONE[r.status]}>{VOUCHER_REDEMPTION_STATUS_LABELS[r.status]}</Badge>,
    },
    { id: "createdAt", header: "Waktu", cell: (r) => formatDateTime(r.createdAt) },
  ];

  return (
    <Modal
      open={voucher !== null}
      onClose={onClose}
      size="xl"
      title={`Pemakaian voucher ${shown?.code ?? ""}`}
      description="Setiap pembayaran yang memakai kode ini. Yang masih menunggu pembayaran ikut menghitung kuota; yang dibatalkan tidak."
      footer={<Button onClick={onClose}>Tutup</Button>}
    >
      <DataTable
        label="Daftar pemakaian voucher"
        columns={columns}
        rows={load.rows}
        rowId={(r) => String(r.id)}
        status={load.status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={load.meta}
        onPageChange={setPage}
        sort={null}
        onSortChange={() => undefined}
        filtered={false}
        onResetFilters={() => undefined}
        empty={{ title: "Belum dipakai", description: "Pemakaian muncul di sini begitu pelanggan membayar dengan kode ini." }}
      />
    </Modal>
  );
}
