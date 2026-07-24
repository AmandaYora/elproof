import { useEffect, useState } from "react";
import { Card } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Select } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import type { SubscriptionTransactionStatus } from "@/modules/platform-admin/data";
import { TRANSACTION_STATUS_TONE, TRANSACTION_STATUS_LABEL, TRANSACTION_TYPE_LABEL } from "@/modules/platform-admin/lib/status";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

const STATUS_FILTERS: SubscriptionTransactionStatus[] = ["unpaid", "pending", "paid", "expired", "granted"];

export default function PlatformTransactionsPage() {
  const tenants = usePlatformAdminStore((s) => s.tenants);
  const transactions = usePlatformAdminStore((s) => s.transactionPage);
  const meta = usePlatformAdminStore((s) => s.transactionPageMeta);
  const fetchTenants = usePlatformAdminStore((s) => s.fetchTenants);
  const fetchTransactionPage = usePlatformAdminStore((s) => s.fetchTransactionPage);
  const [statusFilter, setStatusFilter] = useState<string>("Semua");
  const [page, setPage] = useState(1);

  useEffect(() => {
    void fetchTenants();
  }, [fetchTenants]);

  useEffect(() => {
    setPage(1);
  }, [statusFilter]);

  useEffect(() => {
    void fetchTransactionPage(page, statusFilter === "Semua" ? "" : statusFilter);
  }, [fetchTransactionPage, page, statusFilter]);

  function tenantName(tenantId: string) {
    return tenants.find((t) => t.id === tenantId)?.businessName ?? "-";
  }

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Transaksi</h1>
        <p className="mt-1 text-[13px] text-text-secondary">
          Riwayat transaksi langganan tenant, diproses otomatis oleh sistem pembayaran.
        </p>
      </div>

      <div className="flex flex-wrap gap-3">
        <Select className="w-56" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="Semua">Semua Status</option>
          {STATUS_FILTERS.map((status) => (
            <option key={status} value={status}>
              {TRANSACTION_STATUS_LABEL[status]}
            </option>
          ))}
        </Select>
      </div>

      <Card>
        {transactions.length === 0 ? (
          <EmptyState title="Tidak ada transaksi ditemukan" description="Ubah filter status untuk melihat transaksi lainnya." />
        ) : (
          <>
            <CardList
              className="sm:hidden"
              items={transactions}
              keyFor={(tx) => tx.id}
              renderItem={(tx) => (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className="font-semibold text-text-primary">{tenantName(tx.tenantId)}</span>
                    <Badge tone={TRANSACTION_STATUS_TONE[tx.status]}>{TRANSACTION_STATUS_LABEL[tx.status]}</Badge>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Tipe" value={TRANSACTION_TYPE_LABEL[tx.type]} />
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
                  <TH>Tenant</TH>
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
                    <TD className="font-semibold text-text-primary">{tenantName(tx.tenantId)}</TD>
                    <TD>{TRANSACTION_TYPE_LABEL[tx.type]}</TD>
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
      </Card>
    </div>
  );
}
