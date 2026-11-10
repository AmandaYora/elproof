import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Pencil, Send, UserCheck, UserX, Repeat, Trash2 } from "lucide-react";
import { Avatar } from "@/shared/components/ui/Avatar";
import { Badge } from "@/shared/components/ui/Badge";
import { Select } from "@/shared/components/ui/Input";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { ClientRoleBadge } from "@/modules/clients/components/ClientRoleBadge";
import { ClientContactFormModal } from "@/modules/clients/components/ClientContactFormModal";
import type { ClientContactFormValues } from "@/modules/clients/schemas/client.schema";
import { useClientStore } from "@/modules/clients/stores/useClientStore";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { Client, ClientRole } from "@/modules/clients/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
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
  Bride: "Pengantin wanita",
  Groom: "Pengantin pria",
  "Family Representative": "Perwakilan keluarga",
};

// The values the API's loginStatus filter takes, in the words the list's badge uses.
const LOGIN_STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: "active", label: "Aktif" },
  { value: "invited", label: "Menunggu undangan" },
  { value: "inactive", label: "Nonaktif" },
  { value: "none", label: "Tanpa akses masuk" },
];

type ModalMode = "edit" | "replace";

interface ModalTarget {
  /** The row that was clicked: it stays valid even if a reload changes the page behind the form. */
  client: Client;
  mode: ModalMode;
}

export default function ClientListPage() {
  const clients = useClientStore((s) => s.clientPage);
  const meta = useClientStore((s) => s.clientPageMeta);
  const fetchClientPage = useClientStore((s) => s.fetchClientPage);
  const updateContact = useClientStore((s) => s.updateContact);
  const toggleActive = useClientStore((s) => s.toggleActive);
  const deleteClient = useClientStore((s) => s.deleteClient);
  const issueAccessLink = useClientStore((s) => s.issueAccessLink);
  const replaceRepresentative = useClientStore((s) => s.replaceRepresentative);
  const fetchProjectOptions = useProjectStore((s) => s.fetchProjectOptions);
  const woName = useTenantBrandingStore((s) => s.businessName) ?? APP_NAME;
  const [linkShare, setLinkShare] = useState<LinkShare | null>(null);

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState({ filters: { role: "", loginStatus: "", projectId: "" }, searchParam: "q" });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("clients");
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [projectOptions, setProjectOptions] = useState<{ id: string; name: string }[]>([]);
  const [modalTarget, setModalTarget] = useState<ModalTarget | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Client | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  // The project filter's picker: the first 100 projects by name, loaded once (not the whole roster).
  useEffect(() => {
    void fetchProjectOptions().then(setProjectOptions).catch(() => undefined);
  }, [fetchProjectOptions]);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchClientPage(list.params)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchClientPage, list.paramsKey, attempt]);

  const reload = useCallback(() => fetchClientPage(list.params), [fetchClientPage, list.params]);
  // The undo toast outlives this render: it must reload with the query as it is then, not as it was.
  const reloadRef = useRef(reload);
  reloadRef.current = reload;

  async function handleToggleActive(client: Client) {
    setActionError(null);
    try {
      await toggleActive(client.projectId, client.id);
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status klien belum bisa diubah. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  async function handleConfirmDelete() {
    if (!deleteTarget) return;
    setActionError(null);
    setIsDeleting(true);
    // Confirmed first (it switches the family's portal login off), then "Urungkan" stays available for a few seconds.
    const done = await deleteWithUndo("Klien", async () => {
      const restore = await deleteClient(deleteTarget.projectId, deleteTarget.id);
      await reloadRef.current().catch(() => undefined);
      return async () => {
        await restore();
        await reloadRef.current().catch(() => undefined);
      };
    });
    setIsDeleting(false);
    if (done) setDeleteTarget(null);
  }

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleContactSubmit(values: ClientContactFormValues) {
    if (!modalTarget) return;
    const { client: target, mode } = modalTarget;
    setActionError(null);
    if (mode === "replace") {
      const { client, link } = await replaceRepresentative(target.projectId, target.id, {
        ...values,
        relationNote: target.relationNote,
      });
      setLinkShare({ link, recipient: clientRecipient(client), notice: "Perwakilan diganti. Akses perwakilan lama sudah dicabut." });
    } else {
      await updateContact(target.projectId, target.id, values);
    }
    await reload();
    setModalTarget(null);
  }

  async function handleSendLink(client: Client) {
    setActionError(null);
    try {
      const link = await issueAccessLink(client.projectId, client.id);
      await reload();
      setLinkShare({ link, recipient: clientRecipient(client) });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membuat link"));
    }
  }

  const activeTarget = modalTarget?.client ?? null;
  // A project chosen through the URL may be beyond the picker's first 100: keep it selectable by its id.
  const pickedProject = list.filters.projectId;
  const projectChoices =
    pickedProject && !projectOptions.some((p) => p.id === pickedProject) ? [...projectOptions, { id: pickedProject, name: `Proyek ${pickedProject}` }] : projectOptions;

  const columns: Column<Client>[] = [
    {
      id: "name",
      header: "Klien",
      sortKey: "name",
      primary: true,
      cell: (c) => (
        <span className="flex items-center gap-3">
          <Avatar name={c.name} />
          <span className="min-w-0">
            <span className="block font-semibold">{c.name}</span>
            <span className="block truncate text-[12.5px] font-normal text-muted-foreground">
              {[c.username, c.phone, c.email, c.accessRoleName && `Portal: ${c.accessRoleName}`].filter(Boolean).join(" · ")}
            </span>
          </span>
        </span>
      ),
    },
    { id: "role", header: "Peran", sortKey: "role", cell: (c) => <ClientRoleBadge role={c.role} /> },
    {
      id: "project",
      header: "Proyek",
      sortKey: "projectName",
      cell: (c) => (
        <Link to={ROUTE_PATHS.projectDetail(c.projectId, "client")} className="font-medium underline-offset-2 hover:underline">
          {c.projectName || `Proyek ${c.projectId}`}
        </Link>
      ),
    },
    { id: "eventDate", header: "Tanggal acara", sortKey: "projectEventDate", cell: (c) => formatDate(c.projectEventDate ?? null) },
    {
      id: "loginStatus",
      header: "Status masuk",
      sortKey: "loginStatus",
      cell: (c) => (
        <>
          {c.loginStatus === "none" ? <Badge tone="neutral">Tanpa akses masuk</Badge> : <ClientLoginStatusBadge client={c} />}
          {c.lastCredentialResetAt && (
            <span className="mt-1 block text-[11.5px] font-normal text-muted-foreground">Link atur ulang: {formatDate(c.lastCredentialResetAt)}</span>
          )}
        </>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Klien</h1>
          <p className="mt-1 text-[13px] text-text-secondary">
            Kelola seluruh akun klien di semua proyek — kontak, status akun, dan perwakilan keluarga.
          </p>
        </div>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama atau email klien..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Select className="w-56" aria-label="Peran" value={list.filters.role || "Semua"} onChange={(e) => list.setFilter("role", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua peran</option>
          {ROLE_OPTIONS.map((r) => (
            <option key={r} value={r}>
              {ROLE_FILTER_LABEL[r]}
            </option>
          ))}
        </Select>
        <Select className="w-52" aria-label="Status masuk" value={list.filters.loginStatus || "Semua"} onChange={(e) => list.setFilter("loginStatus", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua status masuk</option>
          {LOGIN_STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
        <Select className="w-56" aria-label="Proyek" placeholder="Cari proyek..." value={pickedProject || "Semua"} onChange={(e) => list.setFilter("projectId", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua proyek</option>
          {projectChoices.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </Select>
      </ListToolbar>

      <DataTable
        label="Daftar klien"
        columns={columns}
        rows={clients}
        rowId={(c) => c.id}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada klien",
          description: "Klien adalah pengantin dan perwakilan keluarga yang diundang ke portal sebuah proyek. Tambahkan dari tab Klien di dalam proyek.",
          action: (
            <Link to={ROUTE_PATHS.projects}>
              <Button variant="secondary">Buka daftar proyek</Button>
            </Link>
          ),
        }}
        density={density}
        actions={(c) => (
          <div className="flex items-center gap-1.5">
            <IconActionButton icon={Pencil} label="Ubah kontak" tone="neutral" onClick={() => setModalTarget({ client: c, mode: "edit" })} />
            {c.isActive && <IconActionButton icon={Send} label={clientLinkActionLabel(c)} tone="info" onClick={() => void handleSendLink(c)} />}
            {c.isActive ? (
              <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(c)} />
            ) : (
              <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(c)} />
            )}
            {c.role === "Family Representative" && (
              <IconActionButton icon={Repeat} label="Ganti perwakilan" tone="navy" onClick={() => setModalTarget({ client: c, mode: "replace" })} />
            )}
            <IconActionButton icon={Trash2} label="Hapus klien" tone="danger" onClick={() => setDeleteTarget(c)} />
          </div>
        )}
      />

      {modalTarget && activeTarget && (
        <ClientContactFormModal
          key={activeTarget.id}
          open
          onClose={() => setModalTarget(null)}
          onSubmit={handleContactSubmit}
          initialValues={
            modalTarget.mode === "replace"
              ? { name: "", phone: "", email: "" }
              : { name: activeTarget.name, phone: activeTarget.phone, email: activeTarget.email, accessRoleId: activeTarget.accessRoleId }
          }
          username={activeTarget.username}
          title={modalTarget.mode === "replace" ? "Ganti perwakilan keluarga" : "Ubah kontak klien"}
          submitLabel={modalTarget.mode === "replace" ? "Ganti dan buat undangan" : undefined}
          description={
            modalTarget.mode === "replace"
              ? "Akses login perwakilan lama langsung dicabut, lalu Anda mendapat link undangan untuk perwakilan baru."
              : "Perbarui nama dan informasi kontak klien."
          }
        />
      )}

      <Modal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={deleteTarget ? `Hapus ${deleteTarget.name}?` : "Hapus klien"}
        description="Akun login yang terkait (jika ada) ikut dinonaktifkan. Anda bisa mengurungkannya beberapa detik setelah dihapus."
        size="sm"
        busy={isDeleting}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleteTarget(null)} disabled={isDeleting}>
              Batal
            </Button>
            <Button variant="danger" onClick={() => void handleConfirmDelete()} disabled={isDeleting}>
              {isDeleting ? "Menghapus..." : "Hapus klien"}
            </Button>
          </>
        }
      >
        <p className="text-[13px] text-text-secondary">
          {deleteTarget ? `${deleteTarget.name} akan hilang dari ${deleteTarget.projectName || "proyeknya"}.` : ""}
        </p>
      </Modal>

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
