import { useEffect, useState } from "react";
import { Copy, Lock, Plus, Users } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { PageHeader } from "@/shared/components/ui/PageHeader";
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

/** Pengaturan → Peran dan akses (PLAN.md Fase 3a, ADR-0026). */
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
      <PageHeader
        className="mb-1"
        title="Peran dan akses"
        description="Atur siapa boleh melakukan apa di usaha Anda. Perubahan langsung berlaku untuk semua orang yang memakai peran itu."
        actions={
          !writeLocked && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => setEditor({})}>
              Buat peran
            </Button>
          )
        }
      />

      {error && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {error}
        </p>
      )}

      {/* ≥ 1024 px: the two kinds of role on the left, the roles of the chosen kind on the right. */}
      <div className="grid gap-5 lg:grid-cols-[260px_minmax(0,1fr)] lg:items-start lg:gap-8">
        <div
          role="tablist"
          aria-label="Jenis peran"
          className="flex gap-1 border-b border-border lg:sticky lg:top-20 lg:flex-col lg:gap-1 lg:border-b-0"
        >
          {TABS.map((t) => {
            const selected = kind === t.kind;
            return (
              <button
                key={t.kind}
                id={`roles-tab-${t.kind}`}
                role="tab"
                aria-selected={selected}
                aria-controls="roles-panel"
                onClick={() => setKind(t.kind)}
                className={cn(
                  "-mb-px border-b-2 px-3.5 py-2.5 text-left text-[13.5px] font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
                  "lg:mb-0 lg:rounded-md lg:border-b-0 lg:px-3 lg:py-2.5",
                  selected
                    ? "border-primary text-foreground lg:bg-accent lg:text-accent-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground lg:hover:bg-muted",
                )}
              >
                <span className="flex items-center justify-between gap-3">
                  {t.label}
                  {loaded[t.kind] && <span className="text-[12px] font-normal tabular-nums text-muted-foreground">{roles[t.kind].length} peran</span>}
                </span>
                <span className="mt-0.5 hidden text-[12.5px] leading-snug font-normal text-muted-foreground lg:block">{t.hint}</span>
              </button>
            );
          })}
        </div>

        <div id="roles-panel" role="tabpanel" aria-labelledby={`roles-tab-${kind}`} className="flex min-w-0 flex-col gap-3">
          <p className="text-[13px] text-muted-foreground lg:hidden">{tab.hint}</p>
          <div className="rounded-xl border border-border bg-card">
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
                        <span className="text-[14.5px] font-semibold text-foreground group-hover:underline">{role.name}</span>
                        {role.isSystem ? (
                          <Badge tone="brand">
                            <Lock className="h-3 w-3" aria-hidden /> Terkunci
                          </Badge>
                        ) : role.templateKey ? (
                          <Badge tone="neutral">Bawaan</Badge>
                        ) : (
                          <Badge tone="info">Kustom</Badge>
                        )}
                      </span>
                      {role.description && <span className="mt-0.5 block text-[13px] leading-relaxed text-muted-foreground">{role.description}</span>}
                      <span className="mt-1.5 flex flex-wrap gap-x-4 gap-y-1 text-[12.5px] text-muted-foreground">
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
        </div>
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
