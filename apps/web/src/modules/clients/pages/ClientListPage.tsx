import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Pencil, Send, UserCheck, UserX, Repeat, Trash2, AlertTriangle } from "lucide-react";
import { Avatar } from "@/shared/components/ui/Avatar";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
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
import { AccessLinkDialog } from "@/shared/components/AccessLinkDialog";
import type { AccessLink, AccessLinkRecipient } from "@/shared/lib/access-link";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { APP_NAME } from "@/shared/constants/brand";
import { ClientLoginStatusBadge, clientLinkActionLabel } from "@/modules/clients/components/ClientLoginStatus";

interface LinkShare {
  link: AccessLink;
  recipient: AccessLinkRecipient;
  notice?: string;
}

function clientRecipient(c: Pick<Client, "name" | "phone" | "username">): AccessLinkRecipient {
  return { name: c.name, phone: c.phone, username: c.username };
}

const ROLE_OPTIONS: ClientRole[] = ["Bride", "Groom", "Family Representative"];

const ROLE_FILTER_LABEL: Record<ClientRole, string> = {
  Bride: "Pengantin Wanita",
  Groom: "Pengantin Pria",
  "Family Representative": "Wedding Representative",
};

type ModalMode = "edit" | "replace";

interface ModalTarget {
  clientId: string;
  projectId: string;
  mode: ModalMode;
}

export default function ClientListPage() {
  const projects = useProjectStore((s) => s.projects);
  const fetchProjects = useProjectStore((s) => s.fetchProjects);
  const allClients = useClientStore((s) => s.allClients);
  const fetchAllClients = useClientStore((s) => s.fetchAllClients);
  const updateContact = useClientStore((s) => s.updateContact);
  const toggleActive = useClientStore((s) => s.toggleActive);
  const deleteClient = useClientStore((s) => s.deleteClient);
  const issueAccessLink = useClientStore((s) => s.issueAccessLink);
  const replaceRepresentative = useClientStore((s) => s.replaceRepresentative);
  const woName = useTenantBrandingStore((s) => s.businessName) ?? APP_NAME;
  const [linkShare, setLinkShare] = useState<LinkShare | null>(null);

  const [query, setQuery] = useState("");
  const [roleFilter, setRoleFilter] = useState<"Semua" | ClientRole>("Semua");
  const [modalTarget, setModalTarget] = useState<ModalTarget | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchProjects();
    void fetchAllClients();
  }, [fetchProjects, fetchAllClients]);

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

  const activeTarget = modalTarget ? allClients.find((c) => c.id === modalTarget.clientId) ?? null : null;
  const { page, setPage, totalPages, totalItems, pageSize, pageItems: pageGroups } = usePagination(groups, 5);

  async function handleToggleActive(projectId: string, clientId: string) {
    setActionError(null);
    try {
      await toggleActive(projectId, clientId);
      await fetchAllClients();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status client"));
    }
  }

  async function handleDelete(projectId: string, clientId: string) {
    setActionError(null);
    try {
      await deleteClient(projectId, clientId);
      await fetchAllClients();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus client"));
    }
  }

  async function handleContactSubmit(values: ClientContactFormValues) {
    if (!modalTarget || !activeTarget) return;
    setActionError(null);
    try {
      if (modalTarget.mode === "replace") {
        const { client, link } = await replaceRepresentative(modalTarget.projectId, modalTarget.clientId, {
          ...values,
          relationNote: activeTarget.relationNote,
        });
        setLinkShare({ link, recipient: clientRecipient(client), notice: "Perwakilan diganti. Akses perwakilan lama sudah dicabut." });
      } else {
        await updateContact(modalTarget.projectId, modalTarget.clientId, values);
      }
      await fetchAllClients();
      setModalTarget(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan perubahan client"));
    }
  }

  async function handleSendLink(projectId: string, client: Client) {
    setActionError(null);
    try {
      const link = await issueAccessLink(projectId, client.id);
      await fetchAllClients();
      setLinkShare({ link, recipient: clientRecipient(client) });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membuat link"));
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
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
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
                onSendLink={(c) => void handleSendLink(project.id, c)}
                onToggleActive={(id) => void handleToggleActive(project.id, id)}
                onDelete={(id) => void handleDelete(project.id, id)}
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
          initialValues={
            modalTarget.mode === "replace"
              ? { name: "", phone: "", email: "" }
              : { name: activeTarget.name, phone: activeTarget.phone, email: activeTarget.email, accessRoleId: activeTarget.accessRoleId }
          }
          username={activeTarget.username}
          title={modalTarget.mode === "replace" ? "Ganti Wedding Representative" : "Ubah Kontak Client"}
          description={
            modalTarget.mode === "replace"
              ? "Akses login perwakilan lama langsung dicabut, lalu Anda mendapat link undangan untuk perwakilan baru."
              : "Perbarui nama dan informasi kontak client."
          }
        />
      )}

      <AccessLinkDialog
        open={linkShare !== null}
        onClose={() => setLinkShare(null)}
        link={linkShare?.link ?? null}
        recipient={linkShare?.recipient ?? { name: "" }}
        woName={woName}
        notice={linkShare?.notice}
      />
    </div>
  );
}

function ProjectClientGroup({
  project,
  clients,
  onEdit,
  onReplace,
  onSendLink,
  onToggleActive,
  onDelete,
}: {
  project: Project;
  clients: Client[];
  onEdit: (id: string) => void;
  onReplace: (id: string) => void;
  onSendLink: (client: Client) => void;
  onToggleActive: (id: string) => void;
  onDelete: (id: string) => void;
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
          <ClientRow
            key={c.id}
            client={c}
            onEdit={() => onEdit(c.id)}
            onReplace={() => onReplace(c.id)}
            onSendLink={() => onSendLink(c)}
            onToggleActive={() => onToggleActive(c.id)}
            onDelete={() => onDelete(c.id)}
          />
        ))}
      </div>
    </div>
  );
}

function ClientRow({
  client: c,
  onEdit,
  onReplace,
  onSendLink,
  onToggleActive,
  onDelete,
}: {
  client: Client;
  onEdit: () => void;
  onReplace: () => void;
  onSendLink: () => void;
  onToggleActive: () => void;
  onDelete: () => void;
}) {
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  return (
    <div className="px-5 py-3.5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <Avatar name={c.name} />
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-semibold text-text-primary">{c.name}</span>
              <ClientRoleBadge role={c.role} />
            </div>
            <p className="truncate text-[12.5px] text-text-secondary">
              {[c.username, c.phone, c.email, c.accessRoleName && `Portal: ${c.accessRoleName}`].filter(Boolean).join(" · ")}
            </p>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <div className="text-right">
            <ClientLoginStatusBadge client={c} />
            {c.lastCredentialResetAt && (
              <p className="mt-1 text-[11.5px] text-text-secondary">Link atur ulang: {formatDate(c.lastCredentialResetAt)}</p>
            )}
          </div>
          <div className="flex items-center gap-1.5">
            <IconActionButton icon={Pencil} label="Ubah Kontak" tone="neutral" onClick={onEdit} />
            {c.isActive && <IconActionButton icon={Send} label={clientLinkActionLabel(c)} tone="info" onClick={onSendLink} />}
            {c.isActive ? (
              <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={onToggleActive} />
            ) : (
              <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={onToggleActive} />
            )}
            {c.role === "Family Representative" && (
              <IconActionButton icon={Repeat} label="Ganti Representative" tone="navy" onClick={onReplace} />
            )}
            <IconActionButton icon={Trash2} label="Hapus Client" tone="danger" onClick={() => setConfirmingDelete(true)} />
          </div>
        </div>
      </div>

      {confirmingDelete && (
        <div className="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
          <span className="flex items-center gap-2 text-[13px] font-medium text-danger-text">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            Yakin ingin menghapus {c.name}? Tindakan ini permanen — akun login yang terkait (jika ada) ikut dinonaktifkan.
          </span>
          <span className="flex shrink-0 gap-2">
            <Button variant="secondary" size="sm" onClick={() => setConfirmingDelete(false)}>Batal</Button>
            <Button
              variant="danger"
              size="sm"
              onClick={() => {
                onDelete();
                setConfirmingDelete(false);
              }}
            >
              Ya, Hapus
            </Button>
          </span>
        </div>
      )}
    </div>
  );
}
