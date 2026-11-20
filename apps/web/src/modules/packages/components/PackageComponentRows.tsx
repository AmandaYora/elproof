import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { Combobox } from "@/shared/components/ui/Combobox";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { formatCurrency } from "@/shared/lib/formatters";
import type { FormErrors } from "@/shared/forms/useZodForm";
import type { PackageComponentFormValues } from "@/modules/packages/schemas/package.schema";
import { newRowKey } from "@/modules/packages/lib/package-display";

export interface PickerOption {
  id: string;
  name: string;
  isActive: boolean;
  /** A vendor's category: copied into the row when the row has none yet. */
  categoryId?: string;
}

interface PackageComponentRowsProps {
  rows: PackageComponentFormValues[];
  onChange: (rows: PackageComponentFormValues[]) => void;
  errors: FormErrors;
  vendors: PickerOption[];
  categories: PickerOption[];
  /** finance.margin.view: the cost column and the cost total. */
  showCosts: boolean;
  price: number;
  disabled: boolean;
}

const MAX_ROWS = 50;

/** Options for a picker: the active ones, plus the one the row already points at even if it was deactivated since. */
function options(all: PickerOption[], current: string) {
  return all.filter((o) => o.isActive || o.id === current);
}

/**
 * "Isi paket" (DESAIN.md §6.6): what the package contains, in the order the client reads it. Each row may name the
 * vendor that usually does it (picking one fills in its category) and, for a role that sees margins, an estimated cost.
 */
export function PackageComponentRows({ rows, onChange, errors, vendors, categories, showCosts, price, disabled }: PackageComponentRowsProps) {
  const update = (index: number, patch: Partial<PackageComponentFormValues>) => onChange(rows.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  const move = (index: number, by: -1 | 1) => {
    const next = [...rows];
    const [row] = next.splice(index, 1);
    next.splice(index + by, 0, row);
    onChange(next);
  };
  const add = () => onChange([...rows, { key: newRowKey(), id: "", description: "", categoryId: "", vendorId: "", estimatedCost: 0 }]);

  const costTotal = rows.reduce((sum, r) => sum + (r.estimatedCost || 0), 0);
  const err = (i: number, field: string) => errors[`components[${i}].${field}`];

  return (
    <div className="flex flex-col gap-3">
      {rows.length === 0 && (
        <p className="rounded-lg border border-dashed border-border px-4 py-5 text-center text-[13px] text-muted-foreground">
          Belum ada isi. Tambahkan apa saja yang termasuk paket, misalnya katering, dekorasi, atau dokumentasi.
        </p>
      )}

      <ol className="flex flex-col gap-3">
        {rows.map((row, i) => (
          <li key={row.key} className="flex gap-2 rounded-lg border border-border p-3 sm:gap-3">
            {/* The row's place in the list: the client reads the rows in this order. */}
            <span aria-hidden className="mt-7 hidden w-5 shrink-0 text-right sm:block text-[13px] font-semibold tabular-nums text-muted-foreground">
              {i + 1}
            </span>
            <div className="grid min-w-0 flex-1 grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,0.8fr)]">
              <div className="sm:col-span-2 xl:col-span-3">
                <Field label={`Isi paket ${i + 1}`} required name={`components[${i}].description`} error={err(i, "description")}>
                  <Input
                    value={row.description}
                    maxLength={255}
                    onChange={(e) => update(i, { description: e.target.value })}
                    placeholder="cth. Katering buffet 450 porsi + 6 gubukan"
                  />
                </Field>
              </div>
              <Field label="Vendor" name={`components[${i}].vendorId`} error={err(i, "vendorId")}>
                <Combobox
                  aria-label={`Vendor isi paket ${i + 1}`}
                  disabled={disabled}
                  value={row.vendorId}
                  onChange={(e) => {
                    const vendor = vendors.find((v) => v.id === e.target.value);
                    update(i, { vendorId: e.target.value, ...(vendor?.categoryId && !row.categoryId ? { categoryId: vendor.categoryId } : {}) });
                  }}
                  placeholder="Tanpa vendor"
                >
                  <option value="">Tanpa vendor</option>
                  {options(vendors, row.vendorId).map((v) => (
                    <option key={v.id} value={v.id}>
                      {v.isActive ? v.name : `${v.name} (nonaktif)`}
                    </option>
                  ))}
                </Combobox>
              </Field>
              <Field label="Kategori" name={`components[${i}].categoryId`} error={err(i, "categoryId")}>
                <Combobox aria-label={`Kategori isi paket ${i + 1}`} disabled={disabled} value={row.categoryId} onChange={(e) => update(i, { categoryId: e.target.value })} placeholder="Tanpa kategori">
                  <option value="">Tanpa kategori</option>
                  {options(categories, row.categoryId).map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.isActive ? c.name : `${c.name} (nonaktif)`}
                    </option>
                  ))}
                </Combobox>
              </Field>
              {showCosts && (
                <Field label="Perkiraan biaya" name={`components[${i}].estimatedCost`} error={err(i, "estimatedCost")} hint="Hanya terlihat oleh tim">
                  <CurrencyInput optional value={row.estimatedCost} onChange={(n) => update(i, { estimatedCost: n })} placeholder="Rp0" />
                </Field>
              )}
            </div>
            <div className="mt-6 shrink-0">
              <ActionMenu
                variant="ghost"
                label={`Aksi isi paket ${i + 1}`}
                disabled={disabled}
                items={[
                  { label: "Pindah ke atas", icon: <ArrowUp />, onSelect: () => move(i, -1), hidden: i === 0 },
                  { label: "Pindah ke bawah", icon: <ArrowDown />, onSelect: () => move(i, 1), hidden: i === rows.length - 1 },
                  { label: "Hapus isi", icon: <Trash2 />, tone: "danger", onSelect: () => onChange(rows.filter((_, j) => j !== i)) },
                ]}
              />
            </div>
          </li>
        ))}
      </ol>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <Button variant="secondary" size="sm" icon={<Plus className="h-4 w-4" />} onClick={add} disabled={disabled || rows.length >= MAX_ROWS}>
          Tambah isi
        </Button>
        {showCosts && rows.length > 0 && (
          <p className="text-[13px] text-muted-foreground tabular-nums">
            Perkiraan biaya <strong className="font-semibold text-foreground">{formatCurrency(costTotal)}</strong>
            {costTotal > 0 && (
              <>
                {" "}
                · selisih dengan harga <strong className="font-semibold text-foreground">{formatCurrency(price - costTotal)}</strong>
              </>
            )}
          </p>
        )}
      </div>
      {errors.components && <p role="alert" className="text-xs font-medium text-danger-text">{errors.components}</p>}
    </div>
  );
}
