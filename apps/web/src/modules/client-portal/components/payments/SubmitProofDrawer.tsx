import { useEffect, useRef } from "react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Textarea } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { useZodForm, type FormValues } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { newIdempotencyKey } from "@/shared/lib/direct-upload";
import { cn } from "@/shared/lib/cn";
import { todayISO } from "@/shared/lib/formatters";
import { SUBMIT_METHOD_OPTIONS, submitProofSchema } from "@/modules/client-portal/schemas/submit-proof.schema";
import { submitTransferProof } from "@/modules/client-portal/lib/submit-proof";

type Values = FormValues<typeof submitProofSchema>;

const LABELS: Record<string, string> = {
  amount: "Nominal", paymentDate: "Tanggal transfer", method: "Cara bayar", referenceNumber: "No. referensi", notes: "Catatan", proofFile: "Bukti transfer",
};

const FILE_CLASS =
  "block w-full text-[14px] text-muted-foreground file:mr-3 file:h-11 file:rounded-md file:border-0 file:bg-primary file:px-4 file:text-[14px] file:font-semibold file:text-primary-foreground";

function initial(amount: number): Values {
  return { amount, paymentDate: todayISO(), method: "Transfer Bank", referenceNumber: "", notes: "", proofFile: undefined };
}

interface Props {
  open: boolean;
  onClose: () => void;
  projectId: string;
  /** What the next term still asks for: filled in, the couple corrects it when they paid a different amount. */
  suggestedAmount: number;
  onSent: () => void;
}

/**
 * "Kirim bukti transfer": the couple reports a transfer with its receipt. It lands in the WO's "Bukti menunggu
 * verifikasi" queue and shows in the history as waiting; it counts toward the contract only once the WO verifies it.
 * The file goes up first (signed upload), then the payment, with one Idempotency-Key per attempt so a retry never
 * records it twice. A 422 lands under its field; typed values are never cleared.
 */
export default function SubmitProofDrawer({ open, onClose, projectId, suggestedAmount, onSent }: Props) {
  const form = useZodForm({ schema: submitProofSchema, initial: initial(suggestedAmount) });
  const { values, errors, blur } = form;
  const keyRef = useRef(newIdempotencyKey());
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (open) {
      form.reset(initial(suggestedAmount));
      keyRef.current = newIdempotencyKey();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  // A changed form is a different request: it must not reuse the earlier key.
  function set<K extends keyof Values & string>(key: K, value: Values[K]) {
    keyRef.current = newIdempotencyKey();
    form.set(key, value);
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      await submitTransferProof(projectId, parsed, keyRef.current);
      keyRef.current = newIdempotencyKey();
      onSent();
    },
    { root: () => formRef.current, fallback: "Bukti transfer belum terkirim. Periksa koneksi, lalu coba lagi.", known: (key) => key in LABELS },
  );

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="Kirim bukti transfer"
      description="Wedding organizer Anda memeriksa transfer ini dulu. Setelah diverifikasi, pembayaran dihitung dan diberi tanda terverifikasi."
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="submit-proof-form" disabled={form.submitting}>
            {form.submitting ? "Mengirim..." : "Kirim bukti transfer"}
          </Button>
        </>
      }
    >
      <form id="submit-proof-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={LABELS} root={() => formRef.current} />

        <Field label="Nominal yang ditransfer (Rp)" required name="amount" error={errors.amount}>
          <CurrencyInput value={values.amount} onChange={(n) => set("amount", n)} onBlur={() => blur("amount")} aria-invalid={errors.amount ? true : undefined} />
        </Field>
        <Field label="Tanggal transfer" required name="paymentDate" error={errors.paymentDate}>
          <Input type="date" max={todayISO()} value={values.paymentDate} onChange={(e) => set("paymentDate", e.target.value)} onBlur={() => blur("paymentDate")} />
        </Field>

        <fieldset data-field="method" className="flex flex-col gap-1.5">
          <legend className="mb-1.5 text-[13px] font-medium text-foreground">Cara bayar</legend>
          <div className="grid grid-cols-3 gap-2">
            {SUBMIT_METHOD_OPTIONS.map((m) => (
              <label
                key={m}
                className={cn(
                  "flex min-h-11 cursor-pointer items-center justify-center rounded-md border px-2 text-center text-[14px] font-medium has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-solid has-focus-visible:outline-ring",
                  values.method === m ? "border-primary bg-accent text-primary" : "border-input bg-card text-foreground hover:bg-muted",
                )}
              >
                <input type="radio" name="method" value={m} checked={values.method === m} onChange={() => set("method", m)} className="sr-only" />
                {m === "Transfer Bank" ? "Transfer bank" : m}
              </label>
            ))}
          </div>
          {errors.method && <span role="alert" className="text-xs font-medium text-danger-text">{errors.method}</span>}
        </fieldset>

        <Field label="Bukti transfer" required name="proofFile" error={errors.proofFile} hint="Foto struk atau tangkapan layar (JPG, PNG) atau PDF.">
          <input
            type="file"
            accept="image/jpeg,image/png,image/webp,application/pdf"
            onChange={(e) => set("proofFile", e.target.files?.[0] ?? undefined)}
            aria-invalid={errors.proofFile ? true : undefined}
            className={FILE_CLASS}
          />
        </Field>
        <Field label="No. referensi" name="referenceNumber" error={errors.referenceNumber} hint="Opsional. Tertera di struk atau riwayat m-banking.">
          <Input value={values.referenceNumber} onChange={(e) => set("referenceNumber", e.target.value)} onBlur={() => blur("referenceNumber")} />
        </Field>
        <Field label="Catatan" name="notes" error={errors.notes} hint="Opsional, mis. untuk termin yang mana.">
          <Textarea rows={2} value={values.notes} onChange={(e) => set("notes", e.target.value)} onBlur={() => blur("notes")} />
        </Field>
      </form>
    </Drawer>
  );
}
