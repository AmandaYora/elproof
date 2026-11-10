import { useEffect, useState } from "react";
import { Copy, Lock, Plus, Users } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { cn } from "@/shared/lib/cn";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useRolesStore, type Role, type RoleKind } from "@/modules/access/stores/useRolesStore";
import { RoleEditorDrawer } from "@/modules/access/components/RoleEditorDrawer";
import { useWriteLocked } from "@/shared/stores/usePermissionStore";

const TABS: { kind: RoleKind; label: string; hint: string }[] = [
  { kind: "staff", label: "Tim WO", hint: "Apa yang boleh dilakukan setiap anggota tim di WO Console." },
  { kind: "client", label: "Klien", hint: "Apa yang dilihat pengantin dan keluarga di portal klien." },
];

interface EditorState {
  role?: Role;
  copyFrom?: Role;
}

/** Pengaturan → Peran & Akses (PLAN.md Fase 3a, ADR-0026). */
export default function RolesPage() {
  // Read-only workspace (subscription): no new records until it is renewed.
  const writeLocked = useWriteLocked();
  const catalog = useRolesStore((s) => s.catalog);
  const roles = useRolesStore((s) => s.roles);
  const loaded = useRolesStore((s) => s.loaded);
  const fetchCatalog = useRolesStore((s) => s.fetchCatalog);
  const fetchRoles = useRolesStore((s) => s.fetchRoles);
  const [kind, setKind] = useState<RoleKind>("staff");
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([fetchCatalog(), fetchRoles("staff"), fetchRoles("client")]).catch((err) =>
      setError(getApiErrorMessage(err, "Peran gagal dimuat")),
    );
  }, [fetchCatalog, fetchRoles]);

  const list = roles[kind];
  const total = catalog.filter((p) => p.kind === kind).length;
  const tab = TABS.find((t) => t.kind === kind)!;

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Peran & Akses</h1>
          <p className="mt-1 max-w-2xl text-[13px] text-text-secondary">
            Atur siapa boleh melakukan apa di workspace Anda. Perubahan langsung berlaku untuk semua orang yang memakai peran itu.
          </p>
        </div>
        {!writeLocked && (
          <Button icon={<Plus className="h-4 w-4" />} onClick={() => setEditor({})}>
            Buat peran
          </Button>
        )}
      </div>

      <div role="tablist" aria-label="Jenis peran" className="flex gap-1 border-b border-border">
        {TABS.map((t) => (
          <button
            key={t.kind}
            role="tab"
            aria-selected={kind === t.kind}
            onClick={() => setKind(t.kind)}
            className={cn(
              "-mb-px border-b-2 px-3.5 py-2.5 text-[13.5px] font-medium transition-colors",
              kind === t.kind ? "border-navy-900 text-text-primary" : "border-transparent text-text-secondary hover:text-text-primary",
            )}
          >
            {t.label}
          </button>
        ))}
      </div>
      <p className="-mt-2 text-[13px] text-text-secondary">{tab.hint}</p>

      {error && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {error}
        </p>
      )}

      <div className="rounded-xl border border-border bg-surface">
        {!loaded[kind] ? (
          <div className="p-5">
            <SkeletonRows rows={4} label="Memuat peran" />
          </div>
        ) : (
          <ul className="divide-y divide-border-light">
            {list.map((role) => (
              <li key={role.id} className="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:gap-5">
                <button type="button" onClick={() => setEditor({ role })} className="group min-w-0 flex-1 text-left">
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="text-[14.5px] font-semibold text-text-primary group-hover:underline">{role.name}</span>
                    {role.isSystem ? (
                      <Badge tone="navy">
                        <Lock className="h-3 w-3" aria-hidden /> Terkunci
                      </Badge>
                    ) : role.templateKey ? (
                      <Badge tone="neutral">Bawaan</Badge>
                    ) : (
                      <Badge tone="info">Kustom</Badge>
                    )}
                  </span>
                  {role.description && <span className="mt-0.5 block text-[13px] leading-relaxed text-text-secondary">{role.description}</span>}
                  <span className="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-[12.5px] text-text-secondary">
                    <span>
                      {role.permissions.length} dari {total} izin
                    </span>
                    {kind === "staff" && <span>{role.projectScope === "all" ? "Semua proyek" : "Proyek yang dipegang"}</span>}
                    <span className="inline-flex items-center gap-1">
                      <Users className="h-3.5 w-3.5" aria-hidden /> {role.memberCount} orang
                    </span>
                  </span>
                </button>
                <div className="flex shrink-0 gap-2">
                  <Button variant="secondary" size="sm" onClick={() => setEditor({ role })}>
                    {role.isSystem ? "Lihat" : "Ubah"}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    icon={<Copy className="h-3.5 w-3.5" />}
                    onClick={() => setEditor({ copyFrom: role })}
                    aria-label={`Duplikat peran ${role.name}`}
                  >
                    Duplikat
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {editor && (
        <RoleEditorDrawer
          key={editor.role?.id ?? `new-${editor.copyFrom?.id ?? "blank"}-${kind}`}
          open
          onClose={() => setEditor(null)}
          kind={editor.role?.kind ?? editor.copyFrom?.kind ?? kind}
          role={editor.role}
          copyFrom={editor.copyFrom}
        />
      )}
    </div>
  );
}
