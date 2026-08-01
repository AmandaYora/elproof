import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { Building2, ExternalLink } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Select, Field, Input } from "@/shared/components/ui/Input";
import { Modal } from "@/shared/components/ui/Modal";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";
import type { Venue } from "@/modules/venues/types";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";

// Staff-facing (WO Console) — unlike Client Portal's Venue tab, this shows
// the full record (rental price, charge, PIC contact), fetched directly
// from GET /venues/{id} once currentProject.venueId is known, rather than
// the public-safe GET /projects/{id}/venue summary (see ADR-0016).
export default function ProjectVenueTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const currentProject = useProjectStore((s) => s.currentProject);
  const setProjectVenue = useProjectStore((s) => s.setProjectVenue);
  const venues = useVenueStore((s) => s.venues);
  const fetchVenues = useVenueStore((s) => s.fetchVenues);
  const fetchVenue = useVenueStore((s) => s.fetchVenue);

  const [venue, setVenue] = useState<Venue | null>(null);
  const [loading, setLoading] = useState(true);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [selectedVenueId, setSelectedVenueId] = useState("");
  // Per-project cost snapshot being edited in the picker — prefilled from
  // the project's own existing snapshot when reopening for the
  // already-attached venue, or from the newly-picked venue's current
  // master price when switching to a different one (see
  // handleVenueSelect) — never silently overwritten by re-fetching master
  // data just because the modal was opened. See PLAN.md "Financial
  // Calculation Correctness".
  const [rentalPrice, setRentalPrice] = useState(0);
  const [charge, setCharge] = useState(0);
  const [actionError, setActionError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    void fetchVenues();
  }, [fetchVenues]);

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
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [currentProject?.venueId, fetchVenue]);

  function openPicker() {
    setSelectedVenueId(currentProject?.venueId ?? "");
    setRentalPrice(currentProject?.venueRentalPrice ?? 0);
    setCharge(currentProject?.venueCharge ?? 0);
    setActionError(null);
    setPickerOpen(true);
  }

  // Switching to a different venue than the one already attached suggests
  // that venue's own current master price as a starting point; re-selecting
  // the SAME venue that's already attached keeps showing the project's
  // existing snapshot instead (so merely opening the picker and clicking
  // around never silently discards an already-agreed price).
  function handleVenueSelect(newVenueId: string) {
    setSelectedVenueId(newVenueId);
    if (newVenueId === (currentProject?.venueId ?? "")) {
      setRentalPrice(currentProject?.venueRentalPrice ?? 0);
      setCharge(currentProject?.venueCharge ?? 0);
      return;
    }
    const picked = venues.find((v) => v.id === newVenueId);
    setRentalPrice(picked?.rentalPrice ?? 0);
    setCharge(picked?.charge ?? 0);
  }

  async function handleSave() {
    if (!selectedVenueId) return;
    setSaving(true);
    setActionError(null);
    try {
      await setProjectVenue(projectId, selectedVenueId, rentalPrice, charge);
      setPickerOpen(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan venue"));
    } finally {
      setSaving(false);
    }
  }

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
    return <p className="py-10 text-center text-[13px] text-text-secondary">Memuat...</p>;
  }

  return (
    <div className="flex flex-col gap-4">
      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      {!venue ? (
        <Card>
          <EmptyState
            title="Belum ada venue terpasang"
            description="Pilih venue dari direktori untuk project ini."
            action={<Button onClick={openPicker}>Pilih Venue</Button>}
          />
        </Card>
      ) : (
        <Card className="flex flex-col gap-4 p-5">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-lg bg-navy-900 text-white">
                <Building2 className="h-5 w-5" />
              </div>
              <div>
                <p className="text-[15px] font-bold text-text-primary">{venue.name}</p>
                <p className="text-[12.5px] text-text-secondary">
                  {venue.address ?? "-"}
                  {venue.city ? ` · ${venue.city}` : ""}
                </p>
              </div>
            </div>
            <div className="flex gap-2">
              <Button variant="secondary" size="sm" onClick={openPicker}>
                Ubah Venue
              </Button>
              <Button variant="secondary" size="sm" onClick={() => void handleDetach()}>
                Lepas
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-x-4 gap-y-3 border-t border-border-light pt-4 sm:grid-cols-3">
            <div>
              <p className="text-[11.5px] text-text-secondary">Harga Sewa</p>
              <p className="text-[13.5px] font-semibold text-text-primary">
                {currentProject?.venueRentalPrice ? formatCurrency(currentProject.venueRentalPrice) : "Belum diisi"}
              </p>
            </div>
            <div>
              <p className="text-[11.5px] text-text-secondary">Charge</p>
              <p className="text-[13.5px] font-semibold text-text-primary">
                {currentProject?.venueCharge ? formatCurrency(currentProject.venueCharge) : "-"}
              </p>
            </div>
            <div>
              <p className="text-[11.5px] text-text-secondary">Kapasitas</p>
              <p className="text-[13.5px] font-semibold text-text-primary">{venue.capacity ? `${venue.capacity} orang` : "-"}</p>
            </div>
            <div>
              <p className="text-[11.5px] text-text-secondary">PIC</p>
              <p className="text-[13.5px] font-semibold text-text-primary">{venue.picName}</p>
            </div>
            <div>
              <p className="text-[11.5px] text-text-secondary">No Tlp PIC</p>
              <p className="text-[13.5px] font-semibold text-text-primary">{venue.phonePic}</p>
            </div>
            <div>
              <p className="text-[11.5px] text-text-secondary">No Tlp Venue</p>
              <p className="text-[13.5px] font-semibold text-text-primary">{venue.phoneVenue ?? "-"}</p>
            </div>
            <div>
              <p className="text-[11.5px] text-text-secondary">Email</p>
              <p className="text-[13.5px] font-semibold text-text-primary">{venue.email ?? "-"}</p>
            </div>
          </div>

          {venue.facilities && (
            <div className="border-t border-border-light pt-4">
              <p className="mb-1.5 text-[11.5px] text-text-secondary">Fasilitas</p>
              <p className="whitespace-pre-line text-[13px] text-text-primary">{venue.facilities}</p>
            </div>
          )}

          {venue.socialMedia && (
            <div className="border-t border-border-light pt-4">
              <p className="mb-1.5 text-[11.5px] text-text-secondary">Sosial Media</p>
              <p className="whitespace-pre-line text-[13px] text-text-primary">{venue.socialMedia}</p>
            </div>
          )}

          {venue.hasAttachment && (
            <div className="border-t border-border-light pt-4">
              <Button variant="secondary" size="sm" icon={<ExternalLink className="h-3.5 w-3.5" />} onClick={() => void handleViewAttachment()}>
                {venue.attachmentIsImage ? "Lihat Foto" : "Lihat Dokumen"}
              </Button>
            </div>
          )}
        </Card>
      )}

      <Modal
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        title="Pilih Venue"
        description="Pilih venue dari direktori untuk project ini."
        size="sm"
        footer={
          <>
            <Button variant="secondary" onClick={() => setPickerOpen(false)}>
              Batal
            </Button>
            <Button onClick={() => void handleSave()} disabled={!selectedVenueId || saving}>
              {saving ? "Menyimpan..." : "Simpan"}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-4">
          <Field label="Venue">
            <Select value={selectedVenueId} onChange={(e) => handleVenueSelect(e.target.value)}>
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
          <Field label="Harga Sewa (untuk project ini)" hint="Terisi dari harga master venue, bisa diubah sesuai kesepakatan project ini.">
            <Input type="number" min={0} value={rentalPrice} onChange={(e) => setRentalPrice(Number(e.target.value))} />
          </Field>
          <Field label="Charge (untuk project ini)">
            <Input type="number" min={0} value={charge} onChange={(e) => setCharge(Number(e.target.value))} />
          </Field>
        </div>
      </Modal>
    </div>
  );
}
