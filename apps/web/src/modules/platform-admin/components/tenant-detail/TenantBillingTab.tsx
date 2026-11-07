import { Receipt } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import type { SubscriptionTransaction } from "@/modules/platform-admin/data/types";
import { TRANSACTION_STATUS_LABEL, TRANSACTION_STATUS_TONE, TRANSACTION_TYPE_LABEL } from "@/modules/platform-admin/lib/status";

export function TenantBillingTab({ transactions }: { transactions: SubscriptionTransaction[] }) {
  if (transactions.length === 0) {
    return (
      <Card>
        <EmptyState icon={<Receipt className="h-8 w-8" />} title="Belum ada tagihan" description="Tagihan muncul saat tenant memilih paket atau tim mengaktifkan langganan." />
      </Card>
    );
  }
  return (
    <Card>
      <Table>
        <THead>
          <TR>
            <TH>Dibuat</TH>
            <TH>Tipe</TH>
            <TH>Metode</TH>
            <TH>No. referensi</TH>
            <TH>Nominal</TH>
            <TH>Status</TH>
            <TH>Dibayar</TH>
          </TR>
        </THead>
        <TBody>
          {transactions.map((tx) => (
            <TR key={tx.id}>
              <TD>{formatDate(tx.createdAt)}</TD>
              <TD>{TRANSACTION_TYPE_LABEL[tx.type]}</TD>
              <TD>{tx.paymentMethod}</TD>
              <TD className="font-mono text-[12.5px]">{tx.paymentReference}</TD>
              <TD className="tabular-nums">{formatCurrency(tx.amount)}</TD>
              <TD>
                <Badge tone={TRANSACTION_STATUS_TONE[tx.status]}>{TRANSACTION_STATUS_LABEL[tx.status]}</Badge>
              </TD>
              <TD>{tx.paidAt ? formatDate(tx.paidAt) : "-"}</TD>
            </TR>
          ))}
        </TBody>
      </Table>
    </Card>
  );
}
