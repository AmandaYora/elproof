import { Pencil, CheckCircle2, Ban } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { formatCurrency } from "@/shared/lib/formatters";

interface PlanTableProps {
  plans: SubscriptionPlan[];
  onEdit: (plan: SubscriptionPlan) => void;
  onToggleActive: (planId: string) => void;
}

// One table per App section (ADR-0040): every plan belongs to exactly one App.
export function PlanTable({ plans, onEdit, onToggleActive }: PlanTableProps) {
  return (
    <>
      <CardList
        className="sm:hidden"
        items={plans}
        keyFor={(plan) => plan.id}
        renderItem={(plan) => (
          <>
            <div className="flex items-start justify-between gap-3">
              <span className="font-semibold text-text-primary">{plan.name}</span>
              <div className="flex items-center gap-1.5">
                {plan.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
              </div>
            </div>
            <div className="flex flex-col gap-1.5">
              <CardListField label="Durasi" value={`${plan.durationMonths} bulan`} />
              <CardListField label="Harga" value={formatCurrency(plan.price)} />
              <CardListField label="Fitur" value={`${plan.features.length} fitur`} />
            </div>
            <div className="flex items-center gap-1.5 pt-1">
              <IconActionButton icon={Pencil} label="Ubah Paket" tone="neutral" onClick={() => onEdit(plan)} />
              {plan.isActive ? (
                <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => onToggleActive(plan.id)} />
              ) : (
                <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => onToggleActive(plan.id)} />
              )}
            </div>
          </>
        )}
      />
      <div className="hidden sm:block">
        <Table>
          <THead>
            <TR>
              <TH>Nama Paket</TH>
              <TH>Durasi</TH>
              <TH>Harga</TH>
              <TH>Fitur</TH>
              <TH>Status</TH>
              <TH>Aksi</TH>
            </TR>
          </THead>
          <TBody>
            {plans.map((plan) => (
              <TR key={plan.id}>
                <TD className="font-semibold text-text-primary">
                  {plan.name}
                </TD>
                <TD>{plan.durationMonths} bulan</TD>
                <TD className="tabular-nums">{formatCurrency(plan.price)}</TD>
                <TD>
                  <span className="text-[12.5px] text-text-secondary" title={plan.features.join("\n")}>
                    {plan.features.length} fitur
                  </span>
                </TD>
                <TD>{plan.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}</TD>
                <TD>
                  <div className="flex items-center gap-1.5">
                    <IconActionButton icon={Pencil} label="Ubah Paket" tone="neutral" onClick={() => onEdit(plan)} />
                    {plan.isActive ? (
                      <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => onToggleActive(plan.id)} />
                    ) : (
                      <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => onToggleActive(plan.id)} />
                    )}
                  </div>
                </TD>
              </TR>
            ))}
          </TBody>
        </Table>
      </div>
    </>
  );
}
