import { useEffect } from "react";
import { Field, Select } from "@/shared/components/ui/Input";
import { useRolesStore } from "@/modules/access/stores/useRolesStore";

/**
 * Picks the client's portal role (what they see: Pengantin, Keluarga or a
 * role the WO made, ADR-0026). 0 = the default for the client's relation.
 */
export function PortalRoleSelect({
  value,
  onChange,
  allowDefault = false,
}: {
  value: number;
  onChange: (roleId: number) => void;
  allowDefault?: boolean;
}) {
  const roles = useRolesStore((s) => s.roles.client);
  const fetchRoles = useRolesStore((s) => s.fetchRoles);
  useEffect(() => {
    void fetchRoles("client").catch(() => undefined);
  }, [fetchRoles]);
  return (
    <Field label="Akses portal" hint="Menentukan apa yang dilihat client. Atur isinya di Pengaturan → Peran & Akses.">
      <Select value={String(value)} onChange={(e) => onChange(Number(e.target.value))}>
        {allowDefault && <option value="0">Bawaan (Pengantin / Keluarga sesuai peran)</option>}
        {roles.map((r) => (
          <option key={r.id} value={r.id}>
            {r.name}
          </option>
        ))}
      </Select>
    </Field>
  );
}
