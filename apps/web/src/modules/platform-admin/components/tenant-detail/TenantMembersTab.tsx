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
  return (
    <Card>
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
              <TD>{m.role}</TD>
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
