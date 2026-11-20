import { useEffect, useRef, useState } from "react";
import { FilePlus2, Info, Paperclip } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Select } from "@/shared/components/ui/Input";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useZodForm } from "@/shared/forms/useZodForm";
import { formatCurrency, formatDate, todayISO } from "@/shared/lib/formatters";
import { ACCEPT_CHANNEL_LABELS } from "@/shared/lib/labels";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { toast } from "@/shared/stores/useToastStore";
import { ACCEPT_CHANNEL_OPTIONS, acceptSchema } from "@/modules/offers/schemas/quotation.schema";
import { apiErrorCode, useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { daysFromToday } from "@/modules/offers/lib/display";
import type { QuotationDetail, QuotationRevision } from "@/modules/offers/types";

const FIELD_LABELS: Record<string, string> = {
  revisionN: "Revisi yang diterima",
  acceptedOn: "Tanggal diterima",
  channel: "Lewat",
  extendUntil: "Perpanjang sampai",
};
const ACCEPTED_FILES = "image/jpeg,image/png,image/webp,application/pdf";
const MAX_BYTES = 10 * 1024 * 1024;

interface Props {
  detail: QuotationDetail;
  revisions: QuotationRevision[];
  onClose: () => void;
  /** "Buat revisi untuk klien" from inside the dialog when the working copy has changes no revision holds. */
  onCreateRevision: () => Promise<void>;
}

/**
 * "Tandai diterima" (DESAIN §6.5): a summary before the commitment, which revision, when and how. A quotation past
 * its validity is extended in the same step; alternatives that are still open are named because they become
 * Digantikan. Undone with "Batalkan terima" (WCAG 3.3.4).
 */
export default function AcceptDialog({ detail, revisions, onClose, onCreateRevision }: Props) {
  const setStatus = useQuotationStore((s) => s.setStatus);
  const uploadAcceptance = useQuotationStore((s) => s.uploadAcceptance);
  const q = detail.quotation;
  const expired = q.displayStatus === "Expired";
  const needsRevision = detail.hasUnsentChanges || revisions.length === 0;
  const latest = revisions[0]?.n ?? detail.latestRevisionN ?? 0;
  const formRef = useRef<HTMLFormElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [needsRevisionMessage, setNeedsRevisionMessage] = useState<string | null>(null);
  const [making, setMaking] = useState(false);

  const form = useZodForm({
    schema: acceptSchema,
    initial: {
      revisionN: String(detail.matchingRevisionN ?? latest),
      acceptedOn: todayISO(),
      channel: "WhatsApp",
      expired,
      extendUntil: expired ? daysFromToday(7) : "",
    },
  });
  const { values, errors, set, blur } = form;
  // A revision made from inside the dialog becomes the one accepted.
  const preferred = String(detail.matchingRevisionN ?? latest);
  useEffect(() => {
    set("revisionN", preferred);
  }, [preferred, set]);
  const chosen = revisions.find((r) => String(r.n) === values.revisionN);
  const openAlternatives = detail.group.filter((m) => !m.isCurrent && (m.displayStatus === "Draft" || m.displayStatus === "Expired"));

  function pickFile(f: File | null) {
    setFileError(null);
    if (f && !ACCEPTED_FILES.split(",").includes(f.type)) {
      setFileError("Pilih foto (JPG, PNG, WEBP) atau PDF");
      return;
    }
    if (f && f.size > MAX_BYTES) {
      setFileError("Ukuran berkas paling besar 10 MB");
      return;
    }
    setFile(f);
  }

  const submit = form.handleSubmit(
    async (v) => {
      try {
        const after = await setStatus(q.id, {
          status: "Accepted",
          revisionN: Number(v.revisionN),
          acceptedOn: v.acceptedOn,
          channel: v.channel,
          extendUntil: v.expired ? v.extendUntil : "",
        });
        toast.success("Penawaran ditandai diterima");
        if (file) {
          try {
            await uploadAcceptance(q.id, file, after.quotation.updatedAt);
          } catch (err) {
            toast.error(getApiErrorMessage(err, "Lampiran belum terunggah. Unggah lagi dari halaman penawaran."));
          }
        }
        onClose();
      } catch (err) {
        if (apiErrorCode(err) === "quotation_needs_revision") {
          setNeedsRevisionMessage(getApiErrorMessage(err, "Ada perubahan yang belum ada di revisi mana pun."));
          return;
        }
        throw err;
      }
    },
    { root: () => formRef.current, fallback: "Penawaran belum ditandai diterima. Periksa koneksi lalu coba lagi.", known: (key) => key in values },
  );

  async function makeRevision() {
    setMaking(true);
    try {
      await onCreateRevision();
      setNeedsRevisionMessage(null);
    } catch {
      // The page already said why (a toast); the dialog stays as it is.
    } finally {
      setMaking(false);
    }
  }

  // The working copy holds changes that no revision has: the client never saw them, so there is nothing to accept yet.
  if (needsRevision || needsRevisionMessage) {
    return (
      <Modal
        open
        onClose={onClose}
        size="sm"
        title="Buat revisi dulu"
        busy={making}
        footer={
          <>
            <Button variant="secondary" onClick={onClose} disabled={making}>
              Batal
            </Button>
            <Button icon={<FilePlus2 className="h-4 w-4" />} onClick={() => void makeRevision()} disabled={making}>
              {making ? "Membuat revisi..." : "Buat revisi untuk klien"}
            </Button>
          </>
        }
      >
        <p className="text-[13px] text-text-secondary">
          {needsRevisionMessage ??
            (revisions.length === 0
              ? "Penawaran ini belum punya revisi. Klien menerima isi sebuah revisi, jadi buat revisi untuk klien lebih dulu."
              : `Ada perubahan yang belum dikirim ke klien (sejak Revisi ${latest}). Buat revisi baru agar yang diterima sama dengan yang klien lihat.`)}
        </p>
      </Modal>
    );
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="Tandai penawaran diterima"
      description={
        chosen
          ? `Revisi ${chosen.n} · Nilai kontrak ${formatCurrency(chosen.contractTotal)} · ${detail.terms.length} termin`
          : undefined
      }
      busy={form.submitting || making}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="quotation-accept-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Tandai diterima"}
          </Button>
        </>
      }
    >
      <form id="quotation-accept-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Revisi yang diterima" required name="revisionN" error={errors.revisionN}>
          <Select aria-label="Revisi yang diterima" value={values.revisionN} onChange={(e) => set("revisionN", e.target.value)}>
            {revisions.map((r) => (
              <option key={r.n} value={String(r.n)}>
                {`Revisi ${r.n}${r.n === latest ? " (terbaru)" : ""} · ${formatCurrency(r.contractTotal)}`}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Tanggal diterima" required name="acceptedOn" error={errors.acceptedOn}>
          <Input type="date" max={todayISO()} value={values.acceptedOn} onChange={(e) => set("acceptedOn", e.target.value)} onBlur={() => blur("acceptedOn")} />
        </Field>
        <fieldset className="flex flex-col gap-2" data-field="channel">
          <legend className="mb-1.5 text-[13px] font-medium text-foreground">Lewat</legend>
          <div className="flex flex-wrap gap-x-5 gap-y-2">
            {ACCEPT_CHANNEL_OPTIONS.map((c) => (
              <label key={c} className="flex items-center gap-2 text-[13px] text-foreground">
                <input type="radio" name="channel" value={c} checked={values.channel === c} onChange={() => set("channel", c)} className="size-4 accent-primary" />
                {ACCEPT_CHANNEL_LABELS[c]}
              </label>
            ))}
          </div>
          {errors.channel && (
            <span role="alert" className="text-xs font-medium text-danger-text">
              {errors.channel}
            </span>
          )}
        </fieldset>
        {expired && (
          <Field
            label="Perpanjang sampai"
            required
            name="extendUntil"
            error={errors.extendUntil}
            hint={`Masa berlaku lewat sejak ${formatDate(q.validUntil)}. Perpanjangan dan penerimaan tercatat bersama.`}
          >
            <Input type="date" min={todayISO()} value={values.extendUntil} onChange={(e) => set("extendUntil", e.target.value)} onBlur={() => blur("extendUntil")} />
          </Field>
        )}
        <Field label="Lampiran (opsional)" name="attachment" error={fileError ?? undefined} hint="Foto atau PDF penawaran bertanda tangan, paling besar 10 MB.">
          <Input type="file" accept={ACCEPTED_FILES} onChange={(e) => pickFile(e.target.files?.[0] ?? null)} />
        </Field>
        {file && (
          <p className="flex items-center gap-2 text-[13px] text-text-secondary">
            <Paperclip aria-hidden className="h-3.5 w-3.5" />
            {file.name}
          </p>
        )}
        {openAlternatives.length > 0 && (
          <p className="flex items-start gap-2 rounded-md bg-info-soft px-3.5 py-2.5 text-[13px] text-info-text">
            <Info aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
            <span>
              {openAlternatives.map((m) => m.number).join(", ")} (alternatif) akan menjadi Digantikan.
            </span>
          </p>
        )}
      </form>
    </Modal>
  );
}
