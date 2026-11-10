import { SupportAccessCard } from "@/modules/business-profile/components/SupportAccessCard";
import { useCallback, useEffect, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { useWriteLocked } from "@/shared/stores/usePermissionStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { toast } from "@/shared/stores/useToastStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { CompleteProfileModal } from "@/modules/onboarding/components/CompleteProfileModal";
import { BusinessDetailsDrawer, type BusinessDetails } from "@/modules/business-profile/components/BusinessDetailsDrawer";
import { cn } from "@/shared/lib/cn";

interface Tenant {
  businessName: string;
  ownerName: string;
  email: string;
  phone: string;
  city: string;
  hasLogo: boolean;
}

/** 62812… shown the way people write it: 0812… */
function displayPhone(stored: string) {
  return stored.startsWith("62") ? "0" + stored.slice(2) : stored;
}

/** Pengaturan → Profil Usaha: the Owner's own business details and logo. */
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

  useEffect(() => {
    void load();
    loadDetails();
  }, [load, loadDetails]);

  if (loadError) {
    return <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{loadError}</p>;
  }
  if (!tenant) return <div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>;

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Profil Usaha</h1>
        <p className="mt-1 text-[13px] text-text-secondary">Nama, kontak, dan logo WO Anda. Klien melihat ini di portal mereka.</p>
      </div>

      <Card>
        <CardHeader
          title="Data usaha"
          action={!writeLocked && <Button size="sm" variant="secondary" onClick={() => setEditing(true)}>Ubah profil</Button>}
        />
        <CardContent>
          <div className="flex flex-wrap items-start gap-5">
            <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-border bg-surface-muted">
              {logoUrl ? (
                <img src={logoUrl} alt={`Logo ${tenant.businessName}`} className="h-full w-full object-contain" />
              ) : (
                <span className="px-2 text-center text-[11.5px] text-text-secondary">Belum ada logo</span>
              )}
            </div>
            <dl className="grid min-w-0 flex-1 grid-cols-1 gap-x-8 gap-y-3 sm:grid-cols-2">
              <Item label="Nama usaha" value={tenant.businessName} />
              <Item label="Kota" value={tenant.city} />
              <Item label="WhatsApp" value={displayPhone(tenant.phone)} />
              <Item label="Email pemilik" value={tenant.email} />
            </dl>
          </div>
        </CardContent>
      </Card>

      <Card>
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
            <dl className="grid grid-cols-1 gap-x-8 gap-y-3 sm:grid-cols-2">
              <Item label="Alamat usaha" value={details.address} className="sm:col-span-2" />
              <Item label="Bank" value={details.bankName} />
              <Item label="Nomor rekening" value={details.bankAccountNumber} />
              <Item label="Atas nama" value={details.bankAccountHolder} />
            </dl>
          )}
          {details && !details.bankAccountNumber && (
            <p className="mt-3 text-[13px] text-text-secondary">
              Belum ada rekening. Tanpa rekening, klien melihat tagihannya tetapi tidak tahu ke mana harus membayar.
            </p>
          )}
        </CardContent>
      </Card>

      <SupportAccessCard />

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
      <dt className="text-[12px] font-medium text-text-secondary">{label}</dt>
      <dd className="mt-0.5 break-words text-[13.5px] font-medium text-text-primary">{value || "—"}</dd>
    </div>
  );
}
