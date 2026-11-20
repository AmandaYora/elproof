import { useState } from "react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Field, Select, Textarea } from "@/shared/components/ui/Input";
import { LINE_KIND_LABELS } from "@/shared/lib/labels";
import type { EditorRow } from "@/modules/rincian/lib/editor";
import type { NamedChoice, SupplierChoice } from "@/modules/rincian/types";

export interface LineDetailDrawerProps {
  row: EditorRow;
  events: NamedChoice[];
  /** Vendors and venues the WO pays (a line only points at one of those). */
  suppliers: SupplierChoice[];
  categories: NamedChoice[];
  onApply: (changes: Pick<EditorRow, "description" | "eventId" | "supplierRef" | "categoryId">) => void;
  onClose: () => void;
}

/**
 * "Detail baris" (DESAIN §5.3): what does not fit in the row: the full description, the acara it is for, the vendor or
 * venue (paid by the WO) that supplies it, and the category. Applied to the rincian being edited; "Simpan rincian"
 * saves it.
 */
export default function LineDetailDrawer({ row, events, suppliers, categories, onApply, onClose }: LineDetailDrawerProps) {
  const [description, setDescription] = useState(row.description);
  const [eventId, setEventId] = useState(row.eventId);
  const [supplierRef, setSupplierRef] = useState(row.supplierRef);
  const [categoryId, setCategoryId] = useState(row.categoryId);
  const [error, setError] = useState<string | null>(null);

  function apply() {
    if (!description.trim()) {
      setError("Isi uraian baris");
      return;
    }
    onApply({ description: description.trim(), eventId, supplierRef, categoryId });
  }

  return (
    <Drawer
      open
      onClose={onClose}
      title="Detail baris"
      description={`${LINE_KIND_LABELS[row.kind]}${row.isBonus ? " (bonus)" : ""}. Perubahan ikut tersimpan saat Anda menyimpan rincian.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={apply}>Terapkan</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Uraian" name="description" error={error ?? undefined} required>
          <Textarea
            value={description}
            rows={3}
            onChange={(e) => {
              setDescription(e.target.value);
              setError(null);
            }}
          />
        </Field>
        <Field label="Acara" name="eventId" hint="Acara yang dilayani baris ini.">
          <Select value={eventId} onChange={(e) => setEventId(e.target.value)} aria-label="Acara">
            <option value="">Semua acara</option>
            {events.map((e) => (
              <option key={e.id} value={e.id}>
                {e.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Vendor atau venue" name="supplierRef" hint="Hanya vendor dan venue yang dibayar WO.">
          <Select value={supplierRef} onChange={(e) => setSupplierRef(e.target.value)} aria-label="Vendor atau venue">
            <option value="">Belum ditentukan</option>
            {suppliers.map((s) => (
              <option key={s.id} value={s.id}>
                {`${s.name} (${s.kindLabel})`}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Kategori" name="categoryId">
          <Select value={categoryId} onChange={(e) => setCategoryId(e.target.value)} aria-label="Kategori">
            <option value="">Tanpa kategori</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        </Field>
      </div>
    </Drawer>
  );
}
