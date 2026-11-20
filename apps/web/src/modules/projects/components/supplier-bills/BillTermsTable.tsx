import { lazy, Suspense, useMemo, useState, type ReactNode } from "react";
import { Pencil, Trash2 } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { RowActions } from "@/shared/components/ui/RowActions";
import type { Column } from "@/shared/data-table/DataTable";
import { usePagination } from "@/shared/hooks/usePagination";
import { useCan, useWriteLocked } from "@/shared/stores/usePermissionStore";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { LedgerTable } from "@/modules/projects/components/detail/finance-ui";
import { TERM_STATUS_TONE } from "@/modules/projects/lib/status";
import { useSupplierBillStore } from "@/modules/projects/stores/useSupplierBillStore";
import type { SupplierBill, SupplierBillTerm } from "@/modules/projects/types";

// "Tambah tagihan" / "Ubah tagihan" open on a click: the form loads then.
const BillTermFormModal = lazy(() => import("@/modules/projects/components/supplier-bills/BillTermFormModal"));

// Overdue first, then what is coming up by date, paid last (the same order as the client's Jadwal termin).
const ORDER: Record<SupplierBillTerm["status"], number> = { overdue: 0, partial: 1, due_soon: 1, upcoming: 1, paid: 2 };

/** Whether the caller may change the bills of vendors the client pays directly (finance.record, writable workspace). */
export function useCanEditBills(): boolean {
  const writeLocked = useWriteLocked();
  const record = useCan("finance.record");
  return record && !writeLocked;
}

/**
 * The "Tambah tagihan" / "Ubah tagihan" dialog for a list of vendors the client pays directly. `openAdd(supplierId?)`
 * and `openEdit(term)` open it; render `element` once.
 */
export function useBillTermEditor(projectId: string, bills: SupplierBill[]) {
  const [state, setState] = useState<{ term?: SupplierBillTerm; supplierId?: string } | null>(null);
  const options = useMemo(() => bills.map((b) => ({ value: b.projectSupplierId, label: b.name })), [bills]);
  const element: ReactNode = state ? (
    <Suspense fallback={null}>
      <BillTermFormModal
        projectId={projectId}
        supplierOptions={options}
        initialSupplierId={state.supplierId}
        term={state.term}
        onClose={() => setState(null)}
      />
    </Suspense>
  ) : null;
  return {
    openAdd: (supplierId?: string) => setState({ supplierId }),
    openEdit: (term: SupplierBillTerm) => setState({ term }),
    element,
  };
}

interface Row {
  term: SupplierBillTerm;
  supplierName: string;
}

interface Props {
  projectId: string;
  bills: SupplierBill[];
  /** Keuangan lists every vendor's bills; a vendor's own drawer leaves the name column out. */
  showSupplier: boolean;
  onEdit: (term: SupplierBillTerm) => void;
  empty: { title: string; description?: string; action?: ReactNode };
  status?: "loading" | "error" | "ready";
  onRetry?: () => void;
}

/**
 * The bills of vendors and venues the client pays directly ("Tagihan vendor", Paket WO Tahap 4): what each asks for,
 * by when, how much the counted payments cover and what the couple sent that is not checked yet. "Ubah tagihan" and
 * "Hapus tagihan" (undo through the toast) per row.
 */
export function BillTermsTable({ projectId, bills, showSupplier, onEdit, empty, status, onRetry }: Props) {
  const deleteTerm = useSupplierBillStore((s) => s.deleteTerm);
  const canEdit = useCanEditBills();
  const rows: Row[] = useMemo(
    () =>
      bills
        .flatMap((b) => b.terms.map((term) => ({ term, supplierName: b.name })))
        .sort((a, b) => ORDER[a.term.status] - ORDER[b.term.status] || a.term.dueDate.localeCompare(b.term.dueDate)),
    [bills],
  );
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(rows);

  const columns: Column<Row>[] = [
    ...(showSupplier
      ? [
          {
            id: "supplier",
            header: "Vendor atau venue",
            primary: true,
            truncate: true,
            className: "w-[26%] min-w-40",
            cell: (r: Row) => <span className="font-medium" title={r.supplierName}>{r.supplierName}</span>,
          } satisfies Column<Row>,
        ]
      : []),
    { id: "label", header: "Tagihan", primary: !showSupplier, card: true, truncate: true, cell: (r) => <span title={r.term.label}>{r.term.label}</span> },
    { id: "due", header: "Jatuh tempo", card: true, className: "w-32 whitespace-nowrap", cell: (r) => formatDate(r.term.dueDate) },
    { id: "amount", header: "Nominal", align: "right", card: true, cell: (r) => <span className="tabular-nums">{formatCurrency(r.term.amount)}</span> },
    {
      id: "left",
      header: "Sisa",
      align: "right",
      card: true,
      cell: ({ term }) => (
        <span className="flex flex-col items-end">
          <span className="tabular-nums">{formatCurrency(term.remaining)}</span>
          {term.pending > 0 && <span className="text-[12px] whitespace-nowrap text-text-secondary">{formatCurrency(term.pending)} menunggu diperiksa</span>}
        </span>
      ),
    },
    {
      id: "status",
      header: "Status",
      card: true,
      className: "whitespace-nowrap",
      cell: ({ term }) => <Badge tone={TERM_STATUS_TONE[term.status]}>{TERM_STATUS_LABELS.staff[term.status]}</Badge>,
    },
  ];

  return (
    <LedgerTable
      label="Tagihan vendor"
      columns={columns}
      rows={pageItems}
      rowId={(r) => r.term.id}
      page={page}
      pageSize={pageSize}
      totalPages={totalPages}
      totalItems={totalItems}
      onPageChange={setPage}
      empty={empty}
      status={status}
      onRetry={onRetry}
      actions={
        canEdit
          ? (r) => (
              <RowActions
                rowLabel={`${r.supplierName} ${r.term.label}`}
                primary={{ label: "Ubah tagihan", icon: <Pencil />, onClick: () => onEdit(r.term) }}
                items={[
                  {
                    label: "Hapus tagihan",
                    icon: <Trash2 className="h-4 w-4" />,
                    tone: "danger",
                    onSelect: () => void deleteWithUndo("Tagihan vendor", () => deleteTerm(projectId, r.term.id)),
                  },
                ]}
              />
            )
          : undefined
      }
    />
  );
}
