import type { ReactNode } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { formatDate, formatDateTime } from "@/shared/lib/formatters";
import type { Tenant } from "@/modules/platform-admin/data/types";
import type { SignupConsent } from "@/modules/platform-admin/data/console.types";

export function DefinitionList({ rows }: { rows: { label: string; value: ReactNode }[] }) {
  return (
    <dl className="grid grid-cols-1 gap-x-8 gap-y-3 sm:grid-cols-2">
      {rows.map((r) => (
        <div key={r.label}>
          <dt className="text-[12.5px] text-text-secondary">{r.label}</dt>
          <dd className="mt-0.5 break-words text-[13.5px] font-medium text-text-primary">{r.value || "-"}</dd>
        </div>
      ))}
    </dl>
  );
}

export function TenantProfileTab({ tenant, consent }: { tenant: Tenant; consent: SignupConsent | null }) {
  return (
    <div className="flex flex-col gap-5">
      <Card>
        <CardHeader title="Profil WO" />
        <CardContent>
          <DefinitionList
            rows={[
              { label: "Nama WO", value: tenant.businessName },
              { label: "Kota", value: tenant.city },
              { label: "Owner", value: tenant.ownerName },
              { label: "Username Owner", value: `@${tenant.username}` },
              { label: "Email (akun Google Owner)", value: tenant.email },
              { label: "Telepon", value: tenant.phone },
              { label: "Bergabung", value: formatDate(tenant.joinedAt) },
              { label: "Sumber pendaftaran", value: tenant.signupSource === "self_service" ? "Daftar mandiri di situs" : "Dibuat tim ElProof" },
              { label: "Logo", value: tenant.hasLogo ? "Sudah diunggah" : "Belum ada" },
              { label: "Kredensial Owner terakhir diatur ulang", value: tenant.lastCredentialResetAt ? formatDateTime(tenant.lastCredentialResetAt) : "Belum pernah" },
              { label: "Status akun", value: tenant.isSuspended ? <Badge tone="danger">Dibekukan</Badge> : <Badge tone="success">Normal</Badge> },
            ]}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Persetujuan saat mendaftar" subtitle="Bukti Owner menyetujui syarat layanan dan kebijakan privasi." />
        <CardContent>
          {consent ? (
            <DefinitionList
              rows={[
                { label: "Versi syarat yang disetujui", value: consent.termsVersion },
                { label: "Waktu persetujuan", value: formatDateTime(consent.termsAcceptedAt) },
                { label: "Alamat IP", value: <span className="font-mono">{consent.ipAddress}</span> },
                { label: "Setuju menerima kabar produk", value: consent.marketingOptIn ? "Ya" : "Tidak" },
              ]}
            />
          ) : (
            <p className="text-[13px] text-text-secondary">Tenant ini dibuat oleh tim ElProof, jadi tidak ada catatan persetujuan pendaftaran mandiri.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
