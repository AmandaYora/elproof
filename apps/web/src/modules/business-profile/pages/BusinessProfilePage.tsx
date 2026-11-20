import { SupportAccessCard } from "@/modules/business-profile/components/SupportAccessCard";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { useWriteLocked } from "@/shared/stores/usePermissionStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { toast } from "@/shared/stores/useToastStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { CompleteProfileModal } from "@/modules/onboarding/components/CompleteProfileModal";
import { BusinessDetailsDrawer, type BusinessDetails } from "@/modules/business-profile/components/BusinessDetailsDrawer";
import { exampleQuoteNumber, type QuoteSettings } from "@/modules/business-profile/components/quote-settings";
import { cn } from "@/shared/lib/cn";

interface Tenant {
  businessName: string;
  ownerName: string;
  email: string;
  phone: string;
  city: string;
  hasLogo: boolean;
}

const SECTIONS = [
  { id: "data-usaha", label: "Data usaha" },
  { id: "alamat-rekening", label: "Alamat dan rekening" },
  { id: "penawaran", label: "Penawaran" },
  { id: "akses-dukungan", label: "Akses dukungan" },
];

// Only needed after "Ubah" is pressed: fetched then, not with the page.
const QuoteSettingsDrawer = lazy(() => import("@/modules/business-profile/components/QuoteSettingsDrawer").then((m) => ({ default: m.QuoteSettingsDrawer })));

/** 62812… shown the way people write it: 0812… */
function displayPhone(stored: string) {
  return stored.startsWith("62") ? "0" + stored.slice(2) : stored;
}

/** Pengaturan → Profil usaha: the Owner's own business details and logo. */
export default function BusinessProfilePage() {
  const writeLocked = useWriteLocked();
  const hydrateBranding = useTenantBrandingStore((s) => s.hydrate);
  const logoUrl = useTenantBrandingStore((s) => s.logoUrl);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [details, setDetails] = useState<BusinessDetails | null>(null);
  const [editingDetails, setEditingDetails] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await httpClient.get(API.platform.tenantMe);
      setTenant(res.data.data as Tenant);
      setLoadError(null);
    } catch (err) {
      setLoadError(getApiErrorMessage(err, "Profil usaha belum bisa dimuat. Muat ulang halaman."));
    }
  }, []);

  // A failed load must never look like "no bank account": the editor sends all
  // four fields, so editing from a blank form would wipe the real account.
  const [detailsFailed, setDetailsFailed] = useState(false);
  const loadDetails = useCallback(() => {
    setDetailsFailed(false);
    httpClient
      .get(API.platform.tenantMeBusinessDetails)
      .then((res) => setDetails(res.data.data as BusinessDetails))
      .catch(() => setDetailsFailed(true));
  }, []);

  const [quote, setQuote] = useState<QuoteSettings | null>(null);
  const [quoteFailed, setQuoteFailed] = useState(false);
  const [editingQuote, setEditingQuote] = useState(false);
  const loadQuote = useCallback(() => {
    setQuoteFailed(false);
    httpClient
      .get(API.platform.tenantMeQuoteSettings)
      .then((res) => setQuote(res.data.data as QuoteSettings))
      .catch(() => setQuoteFailed(true));
  }, []);

  useEffect(() => {
    void load();
    loadDetails();
    loadQuote();
  }, [load, loadDetails, loadQuote]);

  if (loadError) {
    return <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{loadError}</p>;
  }
  if (!tenant) {
    return (
      <div className="flex flex-col gap-5">
        <PageHeader className="mb-1" title="Profil usaha" />
        <div role="status" className="py-16 text-center text-sm text-muted-foreground">Memuat...</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <PageHeader className="mb-1" title="Profil usaha" description="Nama, kontak, logo, dan rekening WO Anda. Klien melihatnya di portal mereka." />

      {/* ≥ 1024 px: who you are on the left (what clients see first), the details to keep up to date on the right. */}
      <div className="grid gap-5 lg:grid-cols-[300px_minmax(0,1fr)] lg:items-start lg:gap-8">
        <aside className="flex flex-col gap-5 lg:sticky lg:top-20">
          <Card>
            <CardContent className="flex items-center gap-4 py-5 lg:flex-col lg:items-start">
              <div className="flex size-20 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-border bg-muted">
                {logoUrl ? (
                  <img src={logoUrl} alt={`Logo ${tenant.businessName}`} className="h-full w-full object-contain" />
                ) : (
                  <span className="px-2 text-center text-[12px] text-muted-foreground">Belum ada logo</span>
                )}
              </div>
              <div className="flex min-w-0 flex-col gap-1">
                <p className="text-[16px] leading-snug font-semibold break-words text-foreground">{tenant.businessName}</p>
                <p className="text-[13px] text-muted-foreground">{tenant.city || "Kota belum diisi"}</p>
                {!writeLocked && (
                  <Button size="sm" variant="secondary" className="mt-2 self-start" onClick={() => setEditing(true)}>
                    Ubah profil
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
          <nav aria-label="Bagian halaman ini" className="hidden flex-col gap-0.5 lg:flex">
            {SECTIONS.map((s) => (
              <a
                key={s.id}
                href={`#${s.id}`}
                className="rounded-md px-3 py-2 text-[13.5px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
              >
                {s.label}
              </a>
            ))}
          </nav>
        </aside>

        <div className="flex min-w-0 flex-col gap-5">
          <Card id="data-usaha" className="scroll-mt-24">
            <CardHeader title="Data usaha" />
            <CardContent>
              <dl className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2">
                <Item label="Nama usaha" value={tenant.businessName} />
                <Item label="Kota" value={tenant.city} />
                <Item label="WhatsApp" value={displayPhone(tenant.phone)} />
                <Item label="Email pemilik" value={tenant.email} />
              </dl>
            </CardContent>
          </Card>

          <Card id="alamat-rekening" className="scroll-mt-24">
            <CardHeader
              title="Alamat dan rekening pembayaran"
              subtitle="Rekening ini tampil di portal klien yang punya tagihan."
              action={!writeLocked && details && <Button size="sm" variant="secondary" onClick={() => setEditingDetails(true)}>Ubah</Button>}
            />
            <CardContent>
              {detailsFailed && !details && (
                <LoadError message="Alamat dan rekening belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={loadDetails} />
              )}
              {details && (
                <dl className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-3">
                  <Item label="Alamat usaha" value={details.address} className="sm:col-span-3" />
                  <Item label="Bank" value={details.bankName} />
                  <Item label="Nomor rekening" value={details.bankAccountNumber} />
                  <Item label="Atas nama" value={details.bankAccountHolder} />
                </dl>
              )}
              {details && !details.bankAccountNumber && (
                <p className="mt-3 text-[13px] text-muted-foreground">
                  Belum ada rekening. Tanpa rekening, klien melihat tagihannya tetapi tidak tahu ke mana harus membayar.
                </p>
              )}
            </CardContent>
          </Card>

          <Card id="penawaran" className="scroll-mt-24">
            <CardHeader
              title="Penawaran"
              subtitle="Dipakai setiap kali tim menyusun penawaran baru."
              action={!writeLocked && quote && <Button size="sm" variant="secondary" onClick={() => setEditingQuote(true)}>Ubah</Button>}
            />
            <CardContent>
              {quoteFailed && !quote && <LoadError message="Pengaturan penawaran belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={loadQuote} />}
              {!quote && !quoteFailed && <p role="status" className="text-[13px] text-muted-foreground">Memuat...</p>}
              {quote && (
                <dl className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-3">
                  <div className="min-w-0">
                    <dt className="text-[12.5px] text-muted-foreground">Kode usaha</dt>
                    <dd className="mt-0.5 text-[14px] font-medium text-foreground">{quote.documentCode}</dd>
                    <dd className="text-[12.5px] text-muted-foreground tabular-nums">Contoh nomor: {exampleQuoteNumber(quote.documentCode)}</dd>
                  </div>
                  <Item label="Status pajak" value={quote.isPkp ? "PKP: PPN ditambahkan otomatis" : "Bukan PKP: tanpa PPN"} />
                  <Item label="Masa berlaku bawaan" value={`${quote.defaultValidityDays} hari`} />
                  <div className="min-w-0 sm:col-span-3">
                    <dt className="text-[12.5px] text-muted-foreground">Syarat & ketentuan</dt>
                    <dd className="mt-1 line-clamp-4 whitespace-pre-line break-words text-[13.5px] text-foreground">{quote.defaultTermsText || "—"}</dd>
                  </div>
                </dl>
              )}
            </CardContent>
          </Card>

          <section id="akses-dukungan" aria-label="Akses dukungan ElProof" className="scroll-mt-24">
            <SupportAccessCard />
          </section>
        </div>
      </div>

      {editingDetails && details && (
        <BusinessDetailsDrawer
          initial={details}
          onClose={() => setEditingDetails(false)}
          onSaved={(saved) => {
            setDetails(saved);
            setEditingDetails(false);
            toast.success("Alamat dan rekening disimpan");
          }}
        />
      )}

      {editingQuote && quote && (
        <Suspense fallback={null}>
          <QuoteSettingsDrawer
            initial={quote}
            onClose={() => setEditingQuote(false)}
            onSaved={(saved) => {
              setQuote(saved);
              setEditingQuote(false);
              toast.success("Pengaturan penawaran disimpan");
            }}
          />
        </Suspense>
      )}

      <CompleteProfileModal
        open={editing}
        onClose={() => setEditing(false)}
        onSaved={() => {
          setEditing(false);
          void hydrateBranding("WO Console");
          void load();
          toast.success("Profil usaha disimpan");
        }}
      />
    </div>
  );
}

function Item({ label, value, className }: { label: string; value: string; className?: string }) {
  return (
    <div className={cn("min-w-0", className)}>
      <dt className="text-[12.5px] text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 break-words text-[14px] font-medium text-foreground">{value || "—"}</dd>
    </div>
  );
}
