import { Users } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import type { TenantMember } from "@/modules/platform-admin/data/console.types";

export function TenantMembersTab({ members }: { members: TenantMember[] }) {
  if (members.length === 0) {
    return (
      <Card>
        <EmptyState icon={<Users className="h-8 w-8" />} title="Belum ada pengguna" description="Owner belum menerima undangan atau belum ada anggota tim." />
      </Card>
    );
  }
  // The same definition as the Pemakaian tab and the trial cap: active members other than the Owner (EP-12).
  const seats = members.filter((m) => m.countsTowardSeats).length;
  const active = members.filter((m) => m.isActive).length;
  return (
    <Card>
      <p className="border-b border-border-light px-5 py-3 text-[13px] text-text-secondary">
        <span className="font-semibold text-text-primary">{active} aktif</span>, termasuk Owner. Yang dihitung untuk batas anggota tim:{" "}
        <span className="font-semibold text-text-primary">{seats} anggota tim aktif selain Owner</span>.
      </p>
      <Table>
        <THead>
          <TR>
            <TH>Nama</TH>
            <TH>Email</TH>
            <TH>Peran</TH>
            <TH>Status</TH>
          </TR>
        </THead>
        <TBody>
          {members.map((m) => (
            <TR key={m.id}>
              <TD className="font-semibold text-text-primary">{m.name}</TD>
              <TD>{m.email}</TD>
              <TD>
                {m.role}
                {m.isActive && !m.countsTowardSeats && <span className="block text-[12px] text-muted-foreground">Tidak dihitung untuk batas</span>}
              </TD>
              <TD>
                <Badge tone={m.isActive ? "success" : "neutral"}>{m.isActive ? "Aktif" : "Nonaktif"}</Badge>
              </TD>
            </TR>
          ))}
        </TBody>
      </Table>
    </Card>
  );
}
