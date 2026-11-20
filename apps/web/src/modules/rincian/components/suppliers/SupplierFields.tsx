import { Lock } from "lucide-react";
import { ENGAGEMENT_STATUS_LABELS, SUPPLIER_PAYER_LABELS } from "@/shared/lib/labels";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import type { FormErrors } from "@/shared/forms/useZodForm";
import { ENGAGEMENT_STATUS_OPTIONS, SUPPLIER_PAYER_OPTIONS } from "@/modules/rincian/schemas/supplier.schema";
import type { SupplierPayer, EngagementStatus } from "@/modules/rincian/types";

/** The supplier form's values as the inputs hold them (agreedCost: "" = not filled in yet). */
export interface SupplierFieldValues {
  /** The directory vendor: chosen in "Tambah vendor" only, never changed afterwards. */
  vendorId: string;
  paidBy: SupplierPayer;
  showName: boolean;
  agreedCost: "" | number;
  engagementStatus: EngagementStatus;
  bookingDate: string;
  dueDate: string;
  picStaffId: string;
  scope: string;
  notes: string;
}

export const SUPPLIER_FIELD_LABELS: Record<string, string> = {
  vendorId: "Vendor", paidBy: "Dibayar oleh", showName: "Tampilkan nama ke klien", agreedCost: "Biaya rekanan",
  engagementStatus: "Status", bookingDate: "Tanggal booking", dueDate: "Jatuh tempo berikutnya", picStaffId: "PIC internal WO",
  scope: "Lingkup pekerjaan", notes: "Catatan",
};

interface Props {
  values: SupplierFieldValues;
  errors: FormErrors;
  set: <K extends keyof SupplierFieldValues & string>(key: K, value: SupplierFieldValues[K]) => void;
  blur: (key: string) => void;
  /** finance.margin.view: "Biaya rekanan" is shown (for a vendor or venue the WO pays). */
  canMargin: boolean;
  /** The server's reason "Dibayar oleh" can no longer change, shown as it is; "" when it can. */
  paidByLocked?: string;
  /** A venue is always named to the client: no "Tampilkan nama ke klien" switch. */
  isVenue?: boolean;
  staff: { id: string; name: string; title: string }[];
  disabled?: boolean;
}

/**
 * The settings of one vendor or venue of the project (DESAIN §5.2), shared by "Tambah vendor" and the drawer: who pays
 * it, whether the client sees its name, the "Biaya rekanan", status, PIC, dates, scope and notes.
 */
export function SupplierFields({ values, errors, set, blur, canMargin, paidByLocked = "", isVenue = false, staff, disabled = false }: Props) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <fieldset className="flex flex-col gap-2 sm:col-span-2" data-field="paidBy" disabled={disabled}>
        <legend className="mb-1.5 text-[13px] font-medium text-foreground">Dibayar oleh</legend>
        {paidByLocked ? (
          <p className="flex items-start gap-2 rounded-md bg-muted px-3.5 py-2.5 text-[13px] text-foreground">
            <Lock aria-hidden className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
            <span>{paidByLocked}</span>
          </p>
        ) : (
          <div className="flex flex-wrap gap-x-6 gap-y-2">
            {SUPPLIER_PAYER_OPTIONS.map((payer) => (
              <label key={payer} className="flex items-center gap-2 text-[13px] text-foreground">
                <input
                  type="radio"
                  name="paidBy"
                  value={payer}
                  checked={values.paidBy === payer}
                  onChange={() => set("paidBy", payer)}
                  className="size-4 accent-primary"
                />
                {SUPPLIER_PAYER_LABELS[payer]}
              </label>
            ))}
          </div>
        )}
        {errors.paidBy && <span role="alert" className="text-xs font-medium text-danger-text">{errors.paidBy}</span>}
      </fieldset>

      <div className="sm:col-span-2">
        {values.paidBy === "WO" && !isVenue ? (
          <label className="flex items-start justify-between gap-4 rounded-md border border-border px-3.5 py-3 text-[13px]" data-field="showName">
            <span>
              <span className="block font-medium text-foreground">Tampilkan nama ke klien</span>
              <span className="block text-[12.5px] text-muted-foreground">Mati: klien hanya melihat layanannya, tanpa nama vendor atau venue.</span>
            </span>
            <input
              type="checkbox"
              role="switch"
              checked={values.showName}
              disabled={disabled}
              onChange={(e) => set("showName", e.target.checked)}
              className="mt-0.5 size-4 shrink-0 accent-primary"
            />
          </label>
        ) : (
          <p className="rounded-md border border-border px-3.5 py-3 text-[13px] text-muted-foreground">
            {isVenue ? "Selalu tampil ke klien: venue tempat acaranya." : "Selalu tampil ke klien: klien yang membayarnya langsung."}
          </p>
        )}
      </div>

      {canMargin && values.paidBy === "WO" && (
        <div className="sm:col-span-2">
          <Field label="Biaya rekanan (Rp)" name="agreedCost" error={errors.agreedCost} hint="Yang dibayar WO ke vendor atau venue ini. Kosongkan bila belum disepakati.">
            <CurrencyInput
              optional
              value={values.agreedCost === "" ? 0 : Number(values.agreedCost) || 0}
              onChange={(n) => set("agreedCost", n === 0 ? "" : n)}
              onBlur={() => blur("agreedCost")}
              disabled={disabled}
            />
          </Field>
        </div>
      )}

      <Field label="Status" required name="engagementStatus" error={errors.engagementStatus}>
        <Select aria-label="Status" value={values.engagementStatus} disabled={disabled} onChange={(e) => set("engagementStatus", e.target.value as EngagementStatus)}>
          {ENGAGEMENT_STATUS_OPTIONS.map((s) => (
            <option key={s} value={s}>{ENGAGEMENT_STATUS_LABELS.staff[s] ?? s}</option>
          ))}
        </Select>
      </Field>
      <Field label="PIC internal WO" required name="picStaffId" error={errors.picStaffId}>
        <Select aria-label="PIC internal WO" value={values.picStaffId} disabled={disabled} onChange={(e) => set("picStaffId", e.target.value)}>
          {staff.map((s) => (
            <option key={s.id} value={s.id}>{s.title ? `${s.name}, ${s.title}` : s.name}</option>
          ))}
        </Select>
      </Field>
      <Field label="Tanggal booking" name="bookingDate" error={errors.bookingDate}>
        <Input type="date" value={values.bookingDate} disabled={disabled} onChange={(e) => set("bookingDate", e.target.value)} onBlur={() => blur("bookingDate")} />
      </Field>
      <Field label="Jatuh tempo berikutnya" name="dueDate" error={errors.dueDate}>
        <Input type="date" value={values.dueDate} disabled={disabled} onChange={(e) => set("dueDate", e.target.value)} onBlur={() => blur("dueDate")} />
      </Field>
      <div className="sm:col-span-2">
        <Field label="Lingkup pekerjaan" name="scope" error={errors.scope}>
          <Input value={values.scope} disabled={disabled} onChange={(e) => set("scope", e.target.value)} onBlur={() => blur("scope")} placeholder="cth. Dekorasi pelaminan dan meja tamu" />
        </Field>
      </div>
      <div className="sm:col-span-2">
        <Field label="Catatan" name="notes" error={errors.notes}>
          <Textarea rows={2} value={values.notes} disabled={disabled} onChange={(e) => set("notes", e.target.value)} onBlur={() => blur("notes")} />
        </Field>
      </div>
    </div>
  );
}
