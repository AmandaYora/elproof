import { useMemo, useState } from "react";
import axios from "axios";
import { Lock } from "lucide-react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Textarea } from "@/shared/components/ui/Input";
import { cn } from "@/shared/lib/cn";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { toast } from "@/shared/stores/useToastStore";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { useRolesStore, type Permission, type ProjectScope, type Role, type RoleKind } from "@/modules/access/stores/useRolesStore";

interface RoleEditorDrawerProps {
  open: boolean;
  onClose: () => void;
  kind: RoleKind;
  /** Editing an existing role; undefined = creating one. */
  role?: Role;
  /** When creating: start from this role's permissions. */
  copyFrom?: Role;
}

const SCOPES: { value: ProjectScope; title: string; body: string }[] = [
  { value: "all", title: "Semua project", body: "Izin project berlaku di seluruh project workspace." },
  { value: "assigned", title: "Project yang dipegang", body: "Hanya project di mana orang ini menjadi PIC atau PIC Sales." },
];

/**
 * Creates or edits one workspace role: name, project scope and the
 * permission matrix, grouped like the catalog with a plain-language line per
 * permission. The Owner role is shown read-only.
 */
export function RoleEditorDrawer({ open, onClose, kind, role, copyFrom }: RoleEditorDrawerProps) {
  const catalog = useRolesStore((s) => s.catalog);
  const createRole = useRolesStore((s) => s.createRole);
  const updateRole = useRolesStore((s) => s.updateRole);
  const deleteRole = useRolesStore((s) => s.deleteRole);
  const source = role ?? copyFrom;
  const myGrants = usePermissionStore((s) => s.grants);
  // Nobody but the Owner changes the role they hold themselves (the API
  // refuses it too), so role.manage can't become a way to gain access.
  const ownRole = Boolean(role && myGrants && !myGrants.isOwner && myGrants.roleId === role.id);
  const readOnly = Boolean(role?.isSystem) || ownRole;

  const [name, setName] = useState(role?.name ?? (copyFrom ? `${copyFrom.name} (salinan)` : ""));
  const [description, setDescription] = useState(role?.description ?? copyFrom?.description ?? "");
  const [scope, setScope] = useState<ProjectScope>(source?.projectScope ?? (kind === "staff" ? "assigned" : "all"));
  const [granted, setGranted] = useState<Set<string>>(() => new Set(source?.permissions ?? []));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const groups = useMemo(() => {
    const out = new Map<string, Permission[]>();
    for (const p of catalog.filter((p) => p.kind === kind)) {
      out.set(p.group, [...(out.get(p.group) ?? []), p]);
    }
    return Array.from(out.entries());
  }, [catalog, kind]);

  function toggle(key: string) {
    setGranted((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }

  async function save() {
    setSaving(true);
    setErrors({});
    setFormError(null);
    const input = { kind, name, description, projectScope: scope, permissions: Array.from(granted) };
    try {
      if (role) {
        await updateRole(role.id, input);
        toast.success(
          role.memberCount > 0
            ? `Peran "${name}" disimpan. Berlaku langsung untuk ${role.memberCount} orang.`
            : `Peran "${name}" disimpan`,
        );
      } else {
        await createRole(input);
        toast.success(`Peran "${name}" dibuat`);
      }
      onClose();
    } catch (err) {
      if (axios.isAxiosError(err) && err.response?.status === 422) {
        const f = (err.response.data as { errors?: Record<string, string[]> }).errors ?? {};
        setErrors(Object.fromEntries(Object.entries(f).map(([k, v]) => [k, v[0]])));
      }
      setFormError(getApiErrorMessage(err, "Peran belum tersimpan"));
    } finally {
      setSaving(false);
    }
  }

  async function remove() {
    if (!role) return;
    setSaving(true);
    try {
      await deleteRole(role);
      toast.success(`Peran "${role.name}" dihapus`);
      onClose();
    } catch (err) {
      setFormError(getApiErrorMessage(err, "Peran belum terhapus"));
      setConfirmDelete(false);
    } finally {
      setSaving(false);
    }
  }

  const canDelete = role && !role.isSystem && !role.templateKey;

  return (
    <Drawer
      open={open}
      onClose={onClose}
      size="lg"
      title={role ? (readOnly ? role.name : `Ubah peran ${role.name}`) : "Buat peran"}
      description={
        role
          ? `${role.memberCount} orang memakai peran ini. Perubahan berlaku langsung tanpa perlu masuk ulang.`
          : kind === "staff"
            ? "Peran untuk tim WO. Setelah dibuat, pilih peran ini saat mengundang atau mengubah pengguna."
            : "Peran portal client. Pilih peran ini di data client untuk menentukan apa yang mereka lihat."
      }
      footer={
        readOnly ? (
          <Button variant="secondary" onClick={onClose}>
            Tutup
          </Button>
        ) : (
          <div className="flex w-full flex-wrap items-center justify-between gap-2">
            <div>
              {canDelete &&
                (confirmDelete ? (
                  <span className="flex items-center gap-2 text-[13px] text-danger">
                    Hapus peran ini?
                    <Button variant="danger" size="sm" onClick={() => void remove()} disabled={saving}>
                      Ya, hapus
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => setConfirmDelete(false)}>
                      Batal
                    </Button>
                  </span>
                ) : (
                  <Button variant="ghost" size="sm" className="text-danger" onClick={() => setConfirmDelete(true)}>
                    Hapus peran
                  </Button>
                ))}
            </div>
            <div className="flex gap-2">
              <Button variant="secondary" onClick={onClose} disabled={saving}>
                Batal
              </Button>
              <Button onClick={() => void save()} disabled={saving}>
                {saving ? "Menyimpan..." : role ? "Simpan peran" : "Buat peran"}
              </Button>
            </div>
          </div>
        )
      }
    >
      <div className="flex flex-col gap-6">
        {readOnly && (
          <p className="flex items-start gap-2.5 rounded-md border border-border bg-surface-muted px-3.5 py-3 text-[13px] leading-relaxed text-text-secondary">
            <Lock className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
            {ownRole
              ? "Ini peran yang Anda pakai sendiri. Hanya Owner yang dapat mengubahnya."
              : "Owner selalu memiliki semua akses dan tidak bisa diubah, supaya workspace tidak pernah terkunci."}
          </p>
        )}
        {formError && (
          <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
            {formError}
          </p>
        )}

        {!readOnly && (
          <div className="grid grid-cols-1 gap-4">
            <Field label="Nama peran" htmlFor="role-name" required error={errors.name}>
              <Input id="role-name" value={name} onChange={(e) => setName(e.target.value)} maxLength={60} placeholder="cth. Koordinator Lapangan" />
            </Field>
            <Field label="Deskripsi" htmlFor="role-desc" error={errors.description} hint="Satu kalimat tentang tugas peran ini.">
              <Textarea id="role-desc" rows={2} value={description} onChange={(e) => setDescription(e.target.value)} maxLength={255} />
            </Field>
          </div>
        )}

        {kind === "staff" && (
          <fieldset className="flex flex-col gap-2" disabled={readOnly}>
            <legend className="mb-1 text-[13px] font-semibold text-text-primary">Cakupan project</legend>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {SCOPES.map((s) => (
                <label
                  key={s.value}
                  className={cn(
                    "flex cursor-pointer gap-3 rounded-md border px-3.5 py-3 transition-colors",
                    scope === s.value ? "border-navy-900 bg-navy-900/[0.04]" : "border-border hover:bg-surface-muted",
                    readOnly && "cursor-default",
                  )}
                >
                  <input
                    type="radio"
                    name="scope"
                    value={s.value}
                    checked={scope === s.value}
                    onChange={() => setScope(s.value)}
                    className="mt-1 accent-navy-900"
                  />
                  <span>
                    <span className="block text-[13.5px] font-medium text-text-primary">{s.title}</span>
                    <span className="block text-[12.5px] leading-relaxed text-text-secondary">{s.body}</span>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
        )}

        <div className="flex flex-col gap-5">
          <div className="flex items-baseline justify-between gap-2">
            <h3 className="text-[13px] font-semibold text-text-primary">Izin</h3>
            <span className="text-[12.5px] text-text-secondary">{granted.size} dipilih</span>
          </div>
          {errors.permissions && (
            <p role="alert" className="text-xs font-medium text-danger">
              {errors.permissions}
            </p>
          )}
          {groups.map(([group, perms]) => (
            <fieldset key={group} className="flex flex-col" disabled={readOnly}>
              <legend className="mb-2 text-[12.5px] font-semibold text-text-secondary">{group}</legend>
              <div className="divide-y divide-border-light rounded-md border border-border">
                {perms.map((p) => (
                  <label key={p.key} className={cn("flex gap-3 px-3.5 py-[var(--density-row-y)]", !readOnly && "cursor-pointer hover:bg-surface-muted/60")}>
                    <input
                      type="checkbox"
                      checked={granted.has(p.key)}
                      onChange={() => toggle(p.key)}
                      className="mt-1 h-4 w-4 shrink-0 accent-navy-900"
                    />
                    <span className="min-w-0">
                      <span className="block text-[13.5px] font-medium text-text-primary">{p.label}</span>
                      <span className="block text-[12.5px] leading-relaxed text-text-secondary">{p.description}</span>
                    </span>
                  </label>
                ))}
              </div>
            </fieldset>
          ))}
        </div>
      </div>
    </Drawer>
  );
}
