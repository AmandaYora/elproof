import { useEffect, useRef, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Building2, ExternalLink, Unlink } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/shared/components/ui/Card";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { Button } from "@/shared/components/ui/Button";
import { Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { Modal } from "@/shared/components/ui/Modal";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import type { Venue } from "@/modules/venues/types";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import { projectVenueSchema } from "@/modules/projects/schemas/project-venue.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { venueId: "Venue", rentalPrice: "Harga sewa", charge: "Charge" };

// Staff-facing (WO Console) — unlike Client Portal's Venue tab, this shows
// the full record (rental price, charge, PIC contact), fetched directly
// from GET /venues/{id} once currentProject.venueId is known, rather than
// the public-safe GET /projects/{id}/venue summary (see ADR-0016).
export function ProjectVenueSection({ projectId }: { projectId: string }) {
  // Attaching or changing the venue is a project change (ADR-0026).
  const canEdit = useCan("project.edit");
  const currentProject = useProjectStore((s) => s.currentProject);
  const setProjectVenue = useProjectStore((s) => s.setProjectVenue);
  const venues = useVenueStore((s) => s.venues);
  const fetchVenues = useVenueStore((s) => s.fetchVenues);
  const fetchVenue = useVenueStore((s) => s.fetchVenue);

  const [venue, setVenue] = useState<Venue | null>(null);
  const [loading, setLoading] = useState(true);
  const [pickerOpen, setPickerOpen] = useState(false);
  // Per-project cost snapshot being edited in the picker — prefilled from
  // the project's own existing snapshot when reopening for the
  // already-attached venue, or from the newly-picked venue's current
  // master price when switching to a different one (see
  // handleVenueSelect) — never silently overwritten by re-fetching master
  // data just because the modal was opened. See PLAN.md "Financial
  // Calculation Correctness".
  const form = useZodForm({ schema: projectVenueSchema, initial: { venueId: "", rentalPrice: 0, charge: 0 } });
  const { values, errors, set } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    if (!currentProject?.venueId) {
      setVenue(null);
      setLoading(false);
      return;
    }
    void fetchVenue(currentProject.venueId)
      .then((v) => {
        if (!cancelled) setVenue(v);
      })
      .catch(() => {
        // The attached venue may have since been hard-deleted (PLAN.md) --
        // project.venueId is a stale primitive reference with no cleanup, so
        // a 404 here just means "no longer resolvable," same as no venue
        // ever having been attached. Falls back to the same "no venue" line below
        // rather than leaving an unhandled rejection.
        if (!cancelled) setVenue(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [currentProject?.venueId, fetchVenue]);

  function openPicker() {
    // The directory is only needed to pick from: it loads when the picker opens, not with the tab.
    void fetchVenues().catch(() => undefined);
    form.reset({
      venueId: currentProject?.venueId ?? "",
      rentalPrice: currentProject?.venueRentalPrice ?? 0,
      charge: currentProject?.venueCharge ?? 0,
    });
    setActionError(null);
    setPickerOpen(true);
  }

  // Switching to a different venue than the one already attached suggests
  // that venue's own current master price as a starting point; re-selecting
  // the SAME venue that's already attached keeps showing the project's
  // existing snapshot instead (so merely opening the picker and clicking
  // around never silently discards an already-agreed price).
  function handleVenueSelect(newVenueId: string) {
    set("venueId", newVenueId);
    if (newVenueId === (currentProject?.venueId ?? "")) {
      set("rentalPrice", currentProject?.venueRentalPrice ?? 0);
      set("charge", currentProject?.venueCharge ?? 0);
      return;
    }
    const picked = venues.find((v) => v.id === newVenueId);
    set("rentalPrice", picked?.rentalPrice ?? 0);
    set("charge", picked?.charge ?? 0);
  }

  // Errors propagate to the form: a 422 shows on its fields, anything else inside the modal.
  const submit = form.handleSubmit(
    async (parsed) => {
      setActionError(null);
      await setProjectVenue(projectId, parsed.venueId, parsed.rentalPrice, parsed.charge);
      setPickerOpen(false);
    },
    { root: () => formRef.current, fallback: "Venue belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  async function handleDetach() {
    setActionError(null);
    try {
      await setProjectVenue(projectId, null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal melepas venue"));
    }
  }

  async function handleViewAttachment() {
    if (!venue) return;
    try {
      const res = await httpClient.get(API.venues.attachment(venue.id), { responseType: "blob" });
      window.open(URL.createObjectURL(res.data as Blob), "_blank");
    } catch {
      setActionError("Gagal membuka lampiran");
    }
  }

  if (loading) {
    return (
      <Card>
        <CardHeader title="Venue" />
        <CardContent>
          <SkeletonRows rows={2} label="Memuat venue" />
        </CardContent>
      </Card>
    );
  }

  return (
    <section aria-labelledby="venue-heading">
      <Card>
        <CardHeader
          title={<span id="venue-heading">Venue</span>}
          action={
            venue && canEdit ? (
              <div className="flex items-center gap-1">
                <Button variant="secondary" size="sm" onClick={openPicker}>
                  Ubah venue
                </Button>
                <ActionMenu items={[{ label: "Lepas venue dari proyek", icon: <Unlink className="h-4 w-4" />, tone: "danger", onSelect: () => void handleDetach() }]} label="Aksi lain untuk venue" />
              </div>
            ) : undefined
          }
        />
        {actionError && (
          <p className="mx-5 mt-4 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
        )}
        {!venue ? (
          // One line, not a big empty state: the vendors below are what this tab is mostly about.
          <CardContent className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <span aria-hidden className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
                <Building2 className="h-4 w-4" />
              </span>
              <div className="min-w-0">
                <p className="text-[13.5px] font-medium text-text-primary">Belum ada venue dari direktori</p>
                <p className="text-[12.5px] text-text-secondary">
                  {currentProject?.venue ? `Tertulis di proyek: ${currentProject.venue}. ` : ""}
                  {canEdit ? "Pilih dari direktori untuk mencatat harga sewa dan kontak PIC-nya." : "Venue proyek ini belum dipilih."}
                </p>
              </div>
            </div>
            {canEdit && (
              <Button size="sm" variant="secondary" onClick={openPicker}>
                Pilih venue
              </Button>
            )}
          </CardContent>
        ) : (
          <CardContent className="flex flex-col gap-4">
            <div className="flex min-w-0 items-start gap-3">
              <span aria-hidden className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-accent text-accent-foreground">
                <Building2 className="h-4 w-4" />
              </span>
              <div className="min-w-0">
                <p className="truncate text-[15px] font-semibold text-text-primary">{venue.name}</p>
                <p className="text-[13px] text-text-secondary">{[venue.address, venue.city].filter(Boolean).join(", ") || "Alamat belum diisi"}</p>
              </div>
            </div>

            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3 xl:grid-cols-4">
              <VenueFact label="Harga sewa" value={currentProject?.venueRentalPrice ? formatCurrency(currentProject.venueRentalPrice) : "Belum diisi"} numeric />
              <VenueFact label="Charge" value={currentProject?.venueCharge ? formatCurrency(currentProject.venueCharge) : "-"} numeric />
              <VenueFact label="Kapasitas" value={venue.capacity ? `${venue.capacity} orang` : "-"} />
              <VenueFact label="PIC" value={venue.picName || "-"} />
              <VenueFact label="Telepon PIC" value={venue.phonePic || "-"} numeric />
              <VenueFact label="Telepon venue" value={venue.phoneVenue ?? "-"} numeric />
              <VenueFact label="Email" value={venue.email ?? "-"} />
            </dl>

            {(venue.facilities || venue.socialMedia || venue.hasAttachment) && (
              <div className="flex flex-col gap-3 border-t border-border-light pt-4">
                {venue.facilities && (
                  <div>
                    <p className="mb-1 text-[12.5px] text-text-secondary">Fasilitas</p>
                    <p className="whitespace-pre-line text-[13px] text-text-primary">{venue.facilities}</p>
                  </div>
                )}
                {venue.socialMedia && (
                  <div>
                    <p className="mb-1 text-[12.5px] text-text-secondary">Media sosial</p>
                    <p className="whitespace-pre-line text-[13px] text-text-primary">{venue.socialMedia}</p>
                  </div>
                )}
                {venue.hasAttachment && (
                  <div>
                    <Button variant="secondary" size="sm" icon={<ExternalLink className="h-3.5 w-3.5" />} onClick={() => void handleViewAttachment()}>
                      {venue.attachmentIsImage ? "Lihat foto" : "Lihat dokumen"}
                    </Button>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        )}
      </Card>


      <Modal
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        title="Pilih venue"
        description="Pilih venue dari direktori untuk proyek ini."
        size="sm"
        busy={form.submitting}
        footer={
          <>
            <Button variant="secondary" onClick={() => setPickerOpen(false)} disabled={form.submitting}>
              Batal
            </Button>
            <Button type="submit" form="project-venue-form" disabled={form.submitting}>
              {form.submitting ? "Menyimpan..." : "Simpan venue"}
            </Button>
          </>
        }
      >
        <form id="project-venue-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
          <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
          <Field label="Venue" required name="venueId" error={errors.venueId}>
            <Select value={values.venueId} onChange={(e) => handleVenueSelect(e.target.value)}>
              <option value="">Pilih venue...</option>
              {venues
                .filter((v) => v.isActive)
                .map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.city ? `${v.name} — ${v.city}` : v.name}
                  </option>
                ))}
            </Select>
          </Field>
          <Field label="Harga sewa (untuk proyek ini)" name="rentalPrice" error={errors.rentalPrice} hint="Terisi dari harga master venue, bisa diubah sesuai kesepakatan proyek ini.">
            <CurrencyInput value={values.rentalPrice} onChange={(n) => set("rentalPrice", n)} aria-invalid={errors.rentalPrice ? true : undefined} />
          </Field>
          <Field label="Charge (untuk proyek ini)" name="charge" error={errors.charge}>
            <CurrencyInput value={values.charge} onChange={(n) => set("charge", n)} aria-invalid={errors.charge ? true : undefined} />
          </Field>
        </form>
      </Modal>
    </section>
  );
}

function VenueFact({ label, value, numeric }: { label: string; value: string; numeric?: boolean }) {
  return (
    <div className="min-w-0">
      <dt className="text-[12.5px] text-text-secondary">{label}</dt>
      <dd className={`mt-0.5 truncate text-[13.5px] font-medium text-text-primary ${numeric ? "tabular-nums" : ""}`}>{value}</dd>
    </div>
  );
}
