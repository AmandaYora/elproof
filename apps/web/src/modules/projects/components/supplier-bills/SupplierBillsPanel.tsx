import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { formatCurrency } from "@/shared/lib/formatters";
import type { RecordPaymentValues } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import { SupplierPaymentLedger, useSupplierPaymentPerms } from "@/modules/projects/components/suppliers/SupplierPaymentLedger";
import { BillTermsTable, useBillTermEditor, useCanEditBills } from "@/modules/projects/components/supplier-bills/BillTermsTable";
import { useProjectStore, type SaveOptions } from "@/modules/projects/stores/useProjectStore";
import { useSupplierBillStore } from "@/modules/projects/stores/useSupplierBillStore";
import type { ProjectSupplier, SupplierBill } from "@/modules/projects/types";

const RecordPaymentDrawer = lazy(() => import("@/modules/projects/components/detail/RecordPaymentDrawer").then((m) => ({ default: m.RecordPaymentDrawer })));

/**
 * The part of a vendor's drawer for a vendor or venue the client pays directly (Paket WO Tahap 4): its bills
 * ("Tagihan vendor", what the couple sees on the portal), what the client paid it, and every payment to it with the
 * proofs the couple sent waiting for "Periksa bukti". Nothing here is WO cash.
 */
export function SupplierBillsPanel({ projectId, supplier }: { projectId: string; supplier: ProjectSupplier }) {
  const supplierPayments = useProjectStore((s) => s.supplierPayments);
  const createSupplierPayment = useProjectStore((s) => s.createSupplierPayment);
  const billsProject = useSupplierBillStore((s) => s.projectId);
  const bills = useSupplierBillStore((s) => s.bills);
  const fetchBills = useSupplierBillStore((s) => s.fetchBills);
  const perms = useSupplierPaymentPerms();
  const canRecord = perms.canRecord("Client");
  const canEditBills = useCanEditBills();
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [recording, setRecording] = useState(false);

  // Loaded again when the vendor changes to "Klien langsung": the list holds client-paid vendors only.
  const load = useCallback(() => {
    setStatus("loading");
    fetchBills(projectId).then(
      () => setStatus("ready"),
      () => setStatus("error"),
    );
  }, [projectId, fetchBills]);
  useEffect(load, [load, supplier.paidBy]);

  const ready = status === "ready" && billsProject === projectId;
  // A vendor with no bills yet still gets an (empty) bill of its own, so "Tambah tagihan" knows who it is for.
  const bill: SupplierBill = bills.find((b) => b.projectSupplierId === supplier.id) ?? {
    projectSupplierId: supplier.id,
    name: supplier.name,
    kind: supplier.kind,
    total: 0,
    paid: 0,
    remaining: 0,
    pending: 0,
    terms: [],
    payments: [],
  };
  const editor = useBillTermEditor(projectId, [bill]);
  const own = supplierPayments.filter((p) => p.projectSupplierId === supplier.id && p.payer === "Client");

  async function record(values: RecordPaymentValues, options: SaveOptions) {
    await createSupplierPayment(projectId, values, options);
    setRecording(false);
    void fetchBills(projectId).catch(() => undefined);
  }

  return (
    <>
      <section aria-labelledby="supplier-bills" className="flex flex-col gap-3 border-t border-border pt-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h3 id="supplier-bills" className="text-[14px] font-semibold text-foreground">Tagihan vendor</h3>
          {canEditBills && ready && (
            <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => editor.openAdd(supplier.id)}>
              Tambah tagihan
            </Button>
          )}
        </div>
        <p className="text-[12.5px] text-muted-foreground">
          Dibayar klien langsung, di luar nilai kontrak dan kas WO. Pasangan melihat tagihan ini di portal dan mengirim bukti bayarnya.
        </p>
        {status === "error" ? (
          <LoadError message="Tagihan vendor belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={load} />
        ) : !ready ? (
          <SkeletonRows rows={2} label="Memuat tagihan vendor" />
        ) : (
          <>
            <dl className="grid grid-cols-3 gap-3 rounded-md border border-border px-3.5 py-3 text-[13px]">
              <div>
                <dt className="text-muted-foreground">Total tagihan</dt>
                <dd className="mt-0.5 font-semibold tabular-nums text-foreground">{formatCurrency(bill.total)}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Dibayar klien</dt>
                <dd className="mt-0.5 font-semibold tabular-nums text-foreground">{formatCurrency(bill.paid)}</dd>
                {bill.pending > 0 && <dd className="text-[12px] text-muted-foreground">{formatCurrency(bill.pending)} menunggu diperiksa</dd>}
              </div>
              <div>
                <dt className="text-muted-foreground">Sisa</dt>
                <dd className="mt-0.5 font-semibold tabular-nums text-foreground">{formatCurrency(bill.remaining)}</dd>
              </div>
            </dl>
            <BillTermsTable
              projectId={projectId}
              bills={[bill]}
              showSupplier={false}
              onEdit={editor.openEdit}
              empty={{
                title: "Belum ada tagihan",
                description: "Catat tagihan dari vendor ini (DP, pelunasan) agar pasangan tahu berapa dan kapan membayarnya.",
              }}
            />
          </>
        )}
      </section>

      <section aria-labelledby="supplier-payments" className="flex flex-col gap-3 border-t border-border pt-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h3 id="supplier-payments" className="text-[14px] font-semibold text-foreground">Dibayar klien langsung</h3>
          {canRecord && (
            <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setRecording(true)}>
              Catat pembayaran
            </Button>
          )}
        </div>
        <SupplierPaymentLedger
          projectId={projectId}
          label={`Pembayaran ${supplier.name}`}
          payments={own}
          suppliers={[supplier]}
          showSupplier={false}
          empty={{
            title: "Belum ada pembayaran",
            description: "Bukti bayar yang dikirim pasangan muncul di sini untuk diperiksa. Anda juga bisa mencatatnya sendiri.",
          }}
        />
      </section>

      {editor.element}

      {recording && (
        <Suspense fallback={null}>
          <RecordPaymentDrawer
            open
            onClose={() => setRecording(false)}
            title="Catat pembayaran klien"
            description={`Yang dibayar klien langsung ke ${supplier.name}. Tidak masuk kas WO.`}
            supplierOptions={[{ value: supplier.id, label: supplier.name }]}
            initialSupplierId={supplier.id}
            slots={{ invoice: true }}
            owed={() => ({ contractValue: 0, existing: [], owedLabel: "" })}
            onSubmit={record}
          />
        </Suspense>
      )}
    </>
  );
}
