import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Pencil, KeyRound, UserCheck, UserX, Repeat } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { Avatar } from "@/shared/components/ui/Avatar";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select, Input, Field } from "@/shared/components/ui/Input";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { ClientRoleBadge } from "@/modules/clients/components/ClientRoleBadge";
import { ClientContactFormModal } from "@/modules/clients/components/ClientContactFormModal";
import type { ClientContactFormValues } from "@/modules/clients/schemas/client.schema";
import { useClientStore } from "@/modules/clients/stores/useClientStore";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { Client, ClientRole } from "@/modules/clients/types";
import type { Project } from "@/modules/projects/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const ROLE_OPTIONS: ClientRole[] = ["Bride", "Groom", "Family Representative"];

const ROLE_FILTER_LABEL: Record<ClientRole, string> = {
  Bride: "Pengantin Wanita",
  Groom: "Pengantin Pria",
  "Family Representative": "Wedding Representative",
};

type ModalMode = "edit" | "replace" | "reset";

interface ModalTarget {
  clientId: string;
  projectId: string;
  mode: ModalMode;
}

export default function ClientListPage() {
  const projects = useProjectStore((s) => s.projects);
  const fetchProjects = useProjectStore((s) => s.fetchProjects);
  const clientsByProject = useClientStore((s) => s.clientsByProject);
  const fetchClients = useClientStore((s) => s.fetchClients);
  const updateContact = useClientStore((s) => s.updateContact);
  const toggleActive = useClientStore((s) => s.toggleActive);
  const resetCredential = useClientStore((s) => s.resetCredential);
  const replaceRepresentative = useClientStore((s) => s.replaceRepresentative);

  const [query, setQuery] = useState("");
  const [roleFilter, setRoleFilter] = useState<"Semua" | ClientRole>("Semua");
  const [modalTarget, setModalTarget] = useState<ModalTarget | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchProjects();
  }, [fetchProjects]);

  useEffect(() => {
    projects.forEach((p) => void fetchClients(p.id));
  }, [projects, fetchClients]);

  const allClients = useMemo(
    () => Object.values(clientsByProject).flat(),
    [clientsByProject]
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return allClients.filter((c) => {
      const matchesQuery = q.length === 0 || c.name.toLowerCase().includes(q);
      const matchesRole = roleFilter === "Semua" || c.role === roleFilter;
      return matchesQuery && matchesRole;
    });
  }, [allClients, query, roleFilter]);

  const groups = useMemo(() => {
    return projects
      .map((project) => ({ project, clients: filtered.filter((c) => c.projectId === project.id) }))
      .filter((group) => group.clients.length > 0);
  }, [projects, filtered]);

  const activeTarget = modalTarget ? clientsByProject[modalTarget.projectId]?.find((c) => c.id === modalTarget.clientId) ?? null : null;
  const { page, setPage, totalPages, totalItems, pageSize, pageItems: pageGroups } = usePagination(groups, 5);

  async function handleToggleActive(projectId: string, clientId: string) {
    setActionError(null);
    try {
      await toggleActive(projectId, clientId);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status client"));
    }
  }

  async function handleContactSubmit(values: ClientContactFormValues) {
    if (!modalTarget || !activeTarget) return;
    setActionError(null);
    try {
      if (modalTarget.mode === "replace") {
        await replaceRepresentative(modalTarget.projectId, modalTarget.clientId, { ...values, relationNote: activeTarget.relationNote });
      } else {
        await updateContact(modalTarget.projectId, modalTarget.clientId, values);
      }
      setModalTarget(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan perubahan client"));
    }
  }

  async function handleResetSubmit(password: string) {
    if (!modalTarget) return;
    setActionError(null);
    try {
      await resetCredential(modalTarget.projectId, modalTarget.clientId, password);
      setModalTarget(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mereset kredensial client"));
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Client</h1>
          <p className="mt-1 text-[13px] text-text-secondary">
            Kelola seluruh akun client, dikelompokkan berdasarkan project — kontak, status akun, dan wedding representative.
          </p>
        </div>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama client..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Select className="w-56" value={roleFilter} onChange={(e) => setRoleFilter(e.target.value as "Semua" | ClientRole)}>
          <option value="Semua">Semua Role</option>
          {ROLE_OPTIONS.map((r) => (
            <option key={r} value={r}>{ROLE_FILTER_LABEL[r]}</option>
          ))}
        </Select>
      </div>

      {groups.length === 0 ? (
        <div className="rounded-xl border border-border bg-surface">
          <EmptyState title="Tidak ada client ditemukan" description="Ubah kata kunci pencarian atau filter role." />
        </div>
      ) : (
        <>
          <div className="flex flex-col gap-4">
            {pageGroups.map(({ project, clients: projectClients }) => (
              <ProjectClientGroup
                key={project.id}
                project={project}
                clients={projectClients}
                onEdit={(id) => setModalTarget({ clientId: id, projectId: project.id, mode: "edit" })}
                onReplace={(id) => setModalTarget({ clientId: id, projectId: project.id, mode: "replace" })}
                onReset={(id) => setModalTarget({ clientId: id, projectId: project.id, mode: "reset" })}
                onToggleActive={(id) => void handleToggleActive(project.id, id)}
              />
            ))}
          </div>
          <div className="rounded-xl border border-border bg-surface">
            <Pagination page={page} totalPages={totalPages} totalItems={totalItems} pageSize={pageSize} onPageChange={setPage} />
          </div>
        </>
      )}

      {modalTarget && activeTarget && (modalTarget.mode === "edit" || modalTarget.mode === "replace") && (
        <ClientContactFormModal
          key={activeTarget.id}
          open
          onClose={() => setModalTarget(null)}
          onSubmit={(values) => void handleContactSubmit(values)}
          initialValues={modalTarget.mode === "replace" ? { name: "", phone: "", email: "" } : { name: activeTarget.name, phone: activeTarget.phone, email: activeTarget.email }}
          title={modalTarget.mode === "replace" ? "Ganti Wedding Representative" : "Ubah Kontak Client"}
          description={
            modalTarget.mode === "replace"
              ? "Perbarui data wedding representative untuk project ini. Kredensial login yang sudah ada tetap dipakai."
              : "Perbarui nama dan informasi kontak client."
          }
        />
      )}

      {modalTarget && activeTarget && modalTarget.mode === "reset" && (
        <ResetCredentialModal client={activeTarget} onClose={() => setModalTarget(null)} onSubmit={(password) => void handleResetSubmit(password)} />
      )}
    </div>
  );
}

function ResetCredentialModal({
  client,
  onClose,
  onSubmit,
}: {
  client: Client;
  onClose: () => void;
  onSubmit: (password: string) => void;
}) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  function handleSubmit() {
    if (password.length < 6) {
      setError("Password minimal 6 karakter");
      return;
    }
    onSubmit(password);
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="Reset Credential"
      description={`Atur password login baru untuk ${client.name}.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Reset Password</Button>
        </>
      }
    >
      <Field label="Password Baru" required hint={error ?? undefined}>
        <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Minimal 6 karakter" />
      </Field>
    </Modal>
  );
}

function ProjectClientGroup({
  project,
  clients,
  onEdit,
  onReplace,
  onReset,
  onToggleActive,
}: {
  project: Project;
  clients: Client[];
  onEdit: (id: string) => void;
  onReplace: (id: string) => void;
  onReset: (id: string) => void;
  onToggleActive: (id: string) => void;
}) {
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-surface">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border-light bg-surface-muted/50 px-5 py-3.5">
        <div>
          <Link to={ROUTE_PATHS.projectDetail(project.id)} className="font-semibold text-text-primary hover:text-navy-900 hover:underline">
            {project.name}
          </Link>
          <p className="text-[12.5px] text-text-secondary">{formatDate(project.eventDate)} · {project.venue}</p>
        </div>
        <ProjectStatusBadge status={project.status} />
      </div>
      <div className="divide-y divide-border-light">
        {clients.map((c) => (
          <div key={c.id} className="flex flex-wrap items-center justify-between gap-3 px-5 py-3.5">
            <div className="flex min-w-0 items-center gap-3">
              <Avatar name={c.name} />
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-semibold text-text-primary">{c.name}</span>
                  <ClientRoleBadge role={c.role} />
                </div>
                <p className="truncate text-[12.5px] text-text-secondary">{c.phone} · {c.email}</p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <div className="text-right">
                {c.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
                <p className="mt-1 text-[11.5px] text-text-secondary">
                  {c.lastCredentialResetAt ? `Reset: ${formatDate(c.lastCredentialResetAt)}` : "Belum pernah direset"}
                </p>
              </div>
              <div className="flex items-center gap-1.5">
                <IconActionButton icon={Pencil} label="Ubah Kontak" tone="neutral" onClick={() => onEdit(c.id)} />
                <IconActionButton icon={KeyRound} label="Reset Credential" tone="info" onClick={() => onReset(c.id)} />
                {c.isActive ? (
                  <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => onToggleActive(c.id)} />
                ) : (
                  <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => onToggleActive(c.id)} />
                )}
                {c.role === "Family Representative" && (
                  <IconActionButton icon={Repeat} label="Ganti Representative" tone="navy" onClick={() => onReplace(c.id)} />
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
