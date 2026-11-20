import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import axios from "axios";
import { ArrowLeft, CircleAlert, RefreshCw } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Card, CardContent, CardHeader } from "@/shared/components/ui/Card";
import { Field, Input } from "@/shared/components/ui/Input";
import { Combobox } from "@/shared/components/ui/Combobox";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { Skeleton } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useZodForm } from "@/shared/forms/useZodForm";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { cn } from "@/shared/lib/cn";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useEventKindStore } from "@/modules/event-kinds/stores/useEventKindStore";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { venueCapacityText } from "@/modules/venues/lib/venue-display";
import { usePackageStore } from "@/modules/packages/stores/usePackageStore";
import { packageSchema, type PackageFormValues } from "@/modules/packages/schemas/package.schema";
import { defaultPreviewGuests, examplePackage, newRowKey } from "@/modules/packages/lib/package-display";
import { PackagePreviewCard } from "@/modules/packages/components/PackagePreviewCard";
import { PackageComponentRows } from "@/modules/packages/components/PackageComponentRows";
import type { Package } from "@/modules/packages/types";

const EMPTY: PackageFormValues = {
  name: "",
  price: 0,
  eventKindIds: [],
  includedGuests: "",
  extraGuestStep: "",
  extraGuestStepPrice: 0,
  defaultVenueId: "",
  components: [],
};

const LABELS: Record<string, string> = {
  name: "Nama paket",
  price: "Harga",
  eventKindIds: "Acara yang dicakup",
  includedGuests: "Tamu termasuk",
  extraGuestStep: "Tambahan per … tamu",
  extraGuestStepPrice: "Harga tambahan",
  defaultVenueId: "Venue bawaan",
  components: "Isi paket",
};

function toFormValues(p: Package): PackageFormValues {
  return {
    name: p.name,
    price: p.price,
    eventKindIds: p.eventKinds.map((k) => k.id),
    includedGuests: p.includedGuests === null ? "" : String(p.includedGuests),
    extraGuestStep: p.extraGuestStep === null ? "" : String(p.extraGuestStep),
    extraGuestStepPrice: p.extraGuestStepPrice ?? 0,
    defaultVenueId: p.defaultVenue?.id ?? "",
    components: p.components.map((c) => ({
      key: newRowKey(),
      id: c.id,
      description: c.description,
      categoryId: c.categoryId ?? "",
      vendorId: c.vendorId ?? "",
      estimatedCost: c.estimatedCost ?? 0,
    })),
  };
}

const digits = (text: string) => text.replace(/\D/g, "").slice(0, 6);

/** Katalog › Paket › one package (DESAIN.md §6.6): a full page, the form on the left and the worked example on the right at ≥ 1280 px. */
export default function PackageFormPage() {
  const { packageId } = useParams();
  const isNew = packageId === undefined;
  const [search] = useSearchParams();
  const navigate = useNavigate();
  const canManage = useCan("package.manage");
  const canSeeCost = useCan("finance.margin.view");

  const fetchPackage = usePackageStore((s) => s.fetchPackage);
  const createPackage = usePackageStore((s) => s.createPackage);
  const updatePackage = usePackageStore((s) => s.updatePackage);
  const kinds = useEventKindStore((s) => s.kinds);
  const fetchKinds = useEventKindStore((s) => s.fetchKinds);
  const venues = useVenueStore((s) => s.venues);
  const fetchVenues = useVenueStore((s) => s.fetchVenues);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);

  const [pkg, setPkg] = useState<Package | null>(null);
  const [status, setStatus] = useState<"loading" | "error" | "missing" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [stale, setStale] = useState(false);
  const form = useZodForm({ schema: packageSchema, initial: EMPTY });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // The pickers' lists: a failure leaves a picker empty, never the page (the package itself is what must load).
  useEffect(() => {
    void fetchVenues().catch(() => undefined);
    void fetchVendors().catch(() => undefined);
    void fetchCategories().catch(() => undefined);
  }, [fetchVenues, fetchVendors, fetchCategories]);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    const load = async () => {
      const loadedKinds = await fetchKinds();
      if (isNew) {
        form.reset(search.get("contoh") === "1" ? examplePackage(loadedKinds) : EMPTY);
        return null;
      }
      return fetchPackage(packageId);
    };
    load()
      .then((p) => {
        if (cancelled) return;
        if (p) {
          setPkg(p);
          form.reset(toFormValues(p));
        }
        setStatus("ready");
      })
      .catch((err) => {
        if (cancelled) return;
        setStatus(axios.isAxiosError(err) && err.response?.status === 404 ? "missing" : "error");
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [packageId, attempt]);

  const included = values.includedGuests === "" ? null : Number(values.includedGuests);
  const step = values.extraGuestStep === "" ? null : Number(values.extraGuestStep);
  const venue = venues.find((v) => v.id === values.defaultVenueId);
  const venueMax = venue ? Math.max(venue.capacityStanding ?? 0, venue.capacitySeated ?? 0) : 0;

  // The rule the preview prices; null while it is incomplete or contradicts itself (the form says why).
  const previewRule = useMemo(() => {
    const hasStepPrice = values.extraGuestStepPrice > 0;
    if (included !== null && included < 1) return null;
    if ((step === null) !== !hasStepPrice) return null;
    if (step !== null && step < 1) return null;
    if (step !== null && included === null) return null;
    return { price: values.price, includedGuests: included, extraGuestStep: step, extraGuestStepPrice: hasStepPrice ? values.extraGuestStepPrice : null };
  }, [values.price, included, step, values.extraGuestStepPrice]);

  const selectableKinds = kinds.filter((k) => k.isActive || values.eventKindIds.includes(k.id));
  const selectableVenues = venues.filter((v) => v.isActive || v.id === values.defaultVenueId);

  function toggleKind(id: string, on: boolean) {
    const next = on ? [...values.eventKindIds, id] : values.eventKindIds.filter((k) => k !== id);
    // Keep the workspace's own order (Lamaran before Akad before Resepsi…), whatever order they were ticked in.
    set("eventKindIds", kinds.map((k) => k.id).filter((k) => next.includes(k)));
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      setStale(false);
      try {
        if (pkg) {
          const saved = await updatePackage(pkg, parsed, canSeeCost);
          setPkg(saved);
          form.reset(toFormValues(saved));
          toast.success("Paket disimpan");
        } else {
          const saved = await createPackage(parsed, canSeeCost);
          toast.success("Paket disimpan");
          navigate(ROUTE_PATHS.packageDetail(saved.id), { replace: true });
        }
      } catch (err) {
        if (isStaleConflict(err)) {
          setStale(true);
          return;
        }
        throw err;
      }
    },
    { root: () => formRef.current, fallback: "Paket belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  const title = isNew ? "Paket baru" : pkg?.name ?? "Paket";
  const actions = (
    <>
      <Button variant="secondary" onClick={() => navigate(ROUTE_PATHS.packages)} disabled={form.submitting}>
        Batal
      </Button>
      <Button type="submit" disabled={form.submitting}>
        {form.submitting ? "Menyimpan..." : "Simpan paket"}
      </Button>
    </>
  );

  return (
    <div className="flex flex-col gap-5">
      <div>
        <Link to={ROUTE_PATHS.packages} className="inline-flex items-center gap-1.5 text-[13px] font-medium text-text-secondary hover:text-text-primary">
          <ArrowLeft aria-hidden className="h-3.5 w-3.5" /> Kembali ke daftar paket
        </Link>
        <div className="mt-2 flex flex-wrap items-center gap-2.5">
          <h1 className="min-w-0 break-words text-xl font-bold text-text-primary">{title}</h1>
          {pkg && !pkg.isActive && <Badge tone="neutral">Nonaktif</Badge>}
        </div>
        {isNew && search.get("contoh") === "1" && status === "ready" && (
          <p className="mt-1 text-[13px] text-muted-foreground">Terisi dengan contoh. Ubah sesuai paket Anda; belum ada yang tersimpan sampai Anda menekan Simpan paket.</p>
        )}
      </div>

      {status === "loading" && (
        <div role="status" aria-label="Memuat paket" className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <div className="flex flex-col gap-5">
            <Skeleton className="h-56 w-full rounded-xl" />
            <Skeleton className="h-40 w-full rounded-xl" />
            <Skeleton className="h-64 w-full rounded-xl" />
          </div>
          <Skeleton className="h-72 w-full rounded-xl max-xl:hidden" />
        </div>
      )}
      {status === "error" && <LoadError message="Paket belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />}
      {status === "missing" && (
        <div className="rounded-xl border border-border bg-card px-5 py-10 text-center">
          <p className="font-semibold text-foreground">Paket tidak ditemukan</p>
          <p className="mt-1 text-[13px] text-muted-foreground">Mungkin sudah dihapus. Kembali ke daftar untuk memilih paket lain.</p>
          <Button className="mt-4" variant="secondary" onClick={() => navigate(ROUTE_PATHS.packages)}>
            Ke daftar paket
          </Button>
        </div>
      )}

      {status === "ready" && (
        <form ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px] xl:items-start">
          <fieldset disabled={!canManage} className="flex min-w-0 flex-col gap-5">
            {!canManage && (
              <p className="rounded-md border border-border bg-muted px-3.5 py-2.5 text-[13px] text-muted-foreground">Anda hanya bisa melihat paket ini.</p>
            )}
            {stale && (
              <div role="alert" className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-warning/40 bg-warning-soft px-3.5 py-2.5 text-[13px] text-warning-text">
                <span className="font-medium">Paket ini baru saja diubah orang lain. Perubahan Anda belum tersimpan.</span>
                <Button size="sm" variant="secondary" icon={<RefreshCw className="h-4 w-4" />} onClick={() => setAttempt((n) => n + 1)}>
                  Muat versi terbaru
                </Button>
              </div>
            )}
            <FormErrorSummary errors={form.errorList} formError={form.formError} labels={LABELS} root={() => formRef.current} />

            <Card>
              <CardHeader title="Paket" />
              <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="sm:col-span-2">
                  <Field label="Nama paket" required name="name" error={errors.name}>
                    <Input value={values.name} maxLength={150} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Paket Resepsi 500 tamu" autoFocus={isNew} />
                  </Field>
                </div>
                <Field label="Harga" required name="price" error={errors.price}>
                  <CurrencyInput value={values.price} onChange={(n) => set("price", n)} onBlur={() => blur("price")} placeholder="Rp0" />
                </Field>
                <fieldset data-field="eventKindIds" className="flex flex-col gap-1.5 sm:col-span-2" aria-describedby={errors.eventKindIds ? "event-kinds-error" : undefined}>
                  <legend className="mb-1.5 text-[13px] font-medium text-foreground">
                    Acara yang dicakup
                    <span aria-hidden className="text-danger-text"> *</span>
                    <span className="sr-only"> (wajib diisi)</span>
                  </legend>
                  <div className="flex flex-wrap gap-2">
                    {selectableKinds.map((k) => {
                      const checked = values.eventKindIds.includes(k.id);
                      return (
                        <label
                          key={k.id}
                          className={cn(
                            "inline-flex h-9 cursor-pointer items-center gap-2 rounded-full border px-3.5 text-[13px] font-medium transition-colors pointer-coarse:h-11",
                            "has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-solid has-[:focus-visible]:outline-ring",
                            checked ? "border-primary bg-primary/10 text-foreground" : "border-border bg-card text-muted-foreground hover:text-foreground",
                          )}
                        >
                          <input type="checkbox" className="size-4 accent-primary" checked={checked} onChange={(e) => toggleKind(k.id, e.target.checked)} />
                          {k.name}
                          {!k.isActive && <span className="text-[12px] font-normal">(nonaktif)</span>}
                        </label>
                      );
                    })}
                  </div>
                  {errors.eventKindIds ? (
                    <p id="event-kinds-error" role="alert" className="flex items-start gap-1 text-xs font-medium text-danger-text">
                      <CircleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
                      {errors.eventKindIds}
                    </p>
                  ) : (
                    <p className="text-xs text-muted-foreground">
                      Tambah atau nonaktifkan pilihan di{" "}
                      <Link to={ROUTE_PATHS.eventKinds} className="font-medium text-foreground underline underline-offset-2">
                        Jenis acara
                      </Link>
                      .
                    </p>
                  )}
                </fieldset>
              </CardContent>
            </Card>

            <Card>
              <CardHeader title="Tamu" subtitle="Kosongkan bila harga paket tidak bergantung jumlah tamu." />
              <CardContent className="flex flex-col gap-4">
                <div className="max-w-56">
                  <Field label="Tamu termasuk" name="includedGuests" error={errors.includedGuests}>
                    <Input
                      inputMode="numeric"
                      className="tabular-nums"
                      value={values.includedGuests}
                      onChange={(e) => set("includedGuests", digits(e.target.value))}
                      onBlur={() => blur("includedGuests")}
                      placeholder="cth. 500"
                    />
                  </Field>
                </div>
                {/* One sentence to read, not a formula to decode (DESAIN.md §10 A3: "kelipatan" confused people). */}
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <Field label="Tambahan per … tamu" name="extraGuestStep" error={errors.extraGuestStep} hint="Tamu di atas yang termasuk dihitung per kelompok ini.">
                    <Input
                      inputMode="numeric"
                      className="tabular-nums"
                      value={values.extraGuestStep}
                      onChange={(e) => set("extraGuestStep", digits(e.target.value))}
                      onBlur={() => blur("extraGuestStep")}
                      placeholder="cth. 50"
                    />
                  </Field>
                  <Field label="Seharga" name="extraGuestStepPrice" error={errors.extraGuestStepPrice} hint="Harga untuk setiap kelompok tamu tambahan.">
                    <CurrencyInput optional value={values.extraGuestStepPrice} onChange={(n) => set("extraGuestStepPrice", n)} onBlur={() => blur("extraGuestStepPrice")} placeholder="Rp0" />
                  </Field>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader title="Venue bawaan" subtitle="Opsional. Dipilihkan otomatis saat paket ini dipakai di penawaran." />
              <CardContent className="flex flex-col gap-2">
                <div className="max-w-md">
                  <Field label="Venue" name="defaultVenueId" error={errors.defaultVenueId}>
                    <Combobox aria-label="Venue bawaan" disabled={!canManage} value={values.defaultVenueId} onChange={(e) => set("defaultVenueId", e.target.value)} placeholder="Tanpa venue bawaan">
                      <option value="">Tanpa venue bawaan</option>
                      {selectableVenues.map((v) => (
                        <option key={v.id} value={v.id}>
                          {v.isActive ? v.name : `${v.name} (nonaktif)`}
                        </option>
                      ))}
                    </Combobox>
                  </Field>
                </div>
                {venue && (
                  <p className="text-[13px] text-muted-foreground">
                    Kapasitas {venueCapacityText(venue) ?? "belum diisi"}
                    {included !== null && venueMax > 0 && included > venueMax && (
                      <span className="mt-1 block font-medium text-warning-text">
                        {venue.name} muat paling banyak {venueMax.toLocaleString("id-ID")} orang, tamu termasuk {included.toLocaleString("id-ID")}.
                      </span>
                    )}
                  </p>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader title="Isi paket" subtitle="Yang termasuk paket, sesuai urutan yang dibaca klien." />
              <CardContent>
                <PackageComponentRows
                  rows={values.components}
                  onChange={(rows) => set("components", rows)}
                  errors={errors}
                  vendors={vendors.map((v) => ({ id: v.id, name: v.name, isActive: v.isActive, categoryId: v.categoryId }))}
                  categories={categories.map((c) => ({ id: c.id, name: c.name, isActive: c.isActive }))}
                  showCosts={canSeeCost}
                  price={values.price}
                  disabled={!canManage}
                />
              </CardContent>
            </Card>
          </fieldset>

          <aside className="flex flex-col gap-4 xl:sticky xl:top-20">
            <PackagePreviewCard
              name={values.name}
              rule={previewRule}
              items={values.components.map((c) => c.description.trim()).filter(Boolean)}
              initialGuests={defaultPreviewGuests(included, step)}
            />
            {canManage && <div className="hidden justify-end gap-2 xl:flex">{actions}</div>}
          </aside>
          {/* Below xl the preview sits under the form, so the buttons stay pinned to the bottom of the whole form. */}
          {canManage && (
            <div className="sticky bottom-0 z-10 -mx-4 flex justify-end gap-2 border-t border-border bg-background px-4 py-3 sm:-mx-6 sm:px-6 xl:hidden">
              {actions}
            </div>
          )}
        </form>
      )}
    </div>
  );
}
