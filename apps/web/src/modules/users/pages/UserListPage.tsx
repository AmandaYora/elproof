import { useCallback, useEffect, useState } from "react";
import { Plus, Pencil, UserCheck, UserX, Send, Trash2, AlertTriangle } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { Select } from "@/shared/components/ui/Input";
import { Avatar } from "@/shared/components/ui/Avatar";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { UserRoleBadge } from "@/modules/users/components/UserRoleBadge";
import { UserFormModal } from "@/modules/users/components/UserFormModal";
import type { UserFormValues } from "@/modules/users/schemas/user.schema";
import { useRolesStore } from "@/modules/access/stores/useRolesStore";
import { AccessLinkDialog } from "@/shared/components/AccessLinkDialog";
import type { AccessLink, AccessLinkRecipient } from "@/shared/lib/access-link";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { APP_NAME } from "@/shared/constants/brand";
import type { StaffMember } from "@/modules/users/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { AffectedProject } from "@/shared/types/delete-impact";
import { coversRole, usePermissionStore, useWriteLocked } from "@/shared/stores/usePermissionStore";

interface LinkShare {
  link: AccessLink;
  recipient: AccessLinkRecipient;
  notice?: string;
}

function recipientFor(user: Pick<StaffMember, "name" | "phone" | "email" | "role">): AccessLinkRecipient {
  return { name: user.name, phone: user.phone, email: user.email, roleLabel: user.role };
}

/** "Status masuk" column: has the person accepted their invitation yet? */
function StaffLoginStatus({ user }: { user: StaffMember }) {
  if (user.loginStatus === "invited") return <Badge tone="warning">Menunggu undangan</Badge>;
  if (user.loginStatus === "active") return <Badge tone="success">Sudah masuk</Badge>;
  return <span className="text-muted-foreground">-</span>;
}

/** "Status" column: whether the account is switched on. */
function StaffActiveBadge({ isActive }: { isActive: boolean }) {
  return isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>;
}

export default function UserListPage() {
  // Read-only workspace (subscription): no new records until it is renewed.
  const writeLocked = useWriteLocked();
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const businessName = useTenantBrandingStore((s) => s.businessName) ?? APP_NAME;
  const users = useStaffStore((s) => s.staffPage);
  const meta = useStaffStore((s) => s.staffPageMeta);
  const fetchStaffPage = useStaffStore((s) => s.fetchStaffPage);
  const createStaff = useStaffStore((s) => s.createStaff);
  const reinviteStaff = useStaffStore((s) => s.reinviteStaff);
  const updateStaff = useStaffStore((s) => s.updateStaff);
  const setStaffActive = useStaffStore((s) => s.setStaffActive);
  const fetchStaffDeleteImpact = useStaffStore((s) => s.fetchStaffDeleteImpact);
  const deleteStaff = useStaffStore((s) => s.deleteStaff);

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState({ filters: { roleId: "", isActive: "" } });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("staff");
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const staffRoles = useRolesStore((s) => s.roles.staff);
  const myGrants = usePermissionStore((s) => s.grants);
  // Someone whose role allows more than one's own is the Owner's to change (ADR-0026; the API refuses it too).
  const manageable = (user: StaffMember) => {
    if (myGrants?.isOwner) return true;
    const role = staffRoles.find((r) => r.id === user.roleId);
    // Roles not loaded yet: keep the actions, the API still decides.
    return role === undefined || coversRole(myGrants, role);
  };
  const fetchRoles = useRolesStore((s) => s.fetchRoles);
  useEffect(() => {
    void fetchRoles("staff").catch(() => undefined);
  }, [fetchRoles]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<StaffMember | undefined>(undefined);
  const [linkShare, setLinkShare] = useState<LinkShare | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<StaffMember | null>(null);
  const [deleteImpact, setDeleteImpact] = useState<AffectedProject[] | null>(null);
  const [impactLoading, setImpactLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchStaffPage(list.params)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchStaffPage, list.paramsKey, attempt]);

  const reload = useCallback(() => fetchStaffPage(list.params), [fetchStaffPage, list.params]);

  function openCreateModal() {
    setActionError(null);
    setEditingUser(undefined);
    setModalOpen(true);
  }

  function openEditModal(user: StaffMember) {
    setActionError(null);
    setEditingUser(user);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingUser(undefined);
  }

  // handleCreate and handleEdit do not catch: errors propagate to the form, which shows a 422 on its fields.
  async function handleCreate(values: UserFormValues) {
    setActionError(null);
    const result = await createStaff(values);
    closeModal();
    await reload();
    setLinkShare({ link: result.invitation, recipient: recipientFor(result.staff), notice: `${result.staff.name} ditambahkan.` });
  }

  async function handleReinvite(user: StaffMember) {
    setActionError(null);
    try {
      const link = await reinviteStaff(user.id);
      setLinkShare({
        link,
        recipient: recipientFor(user),
        notice: user.loginStatus === "invited" ? "Link baru dibuat. Link sebelumnya tidak berlaku lagi." : "Saat link ini diterima, akun Google lama tidak bisa dipakai masuk lagi.",
      });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Link undangan belum bisa dibuat. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  function reinviteLabel(user: StaffMember) {
    return user.loginStatus === "invited" ? "Kirim ulang undangan" : "Undang ulang (ganti akun Google)";
  }

  async function handleEdit(values: UserFormValues) {
    if (!editingUser) return;
    setActionError(null);
    await updateStaff(editingUser.id, values);
    closeModal();
    await reload();
  }

  async function handleToggleActive(user: StaffMember) {
    setActionError(null);
    try {
      await setStaffActive(user.id, !user.isActive);
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status anggota tim belum bisa diubah. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  // Hard delete (PLAN.md) — informed-consent, not a blocking precondition.
  // The Owner row never reaches this (no "Hapus Permanen" button is ever
  // rendered for it, matching the backend's absolute, non-skippable rule).
  async function openDeleteConfirm(user: StaffMember) {
    setDeleteTarget(user);
    setDeleteImpact(null);
    setImpactLoading(true);
    try {
      setDeleteImpact(await fetchStaffDeleteImpact(user.id));
    } catch {
      setDeleteImpact([]);
    } finally {
      setImpactLoading(false);
    }
  }

  function closeDeleteConfirm() {
    setDeleteTarget(null);
    setDeleteImpact(null);
  }

  async function handleConfirmDelete() {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await deleteStaff(deleteTarget.id);
      closeDeleteConfirm();
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Anggota tim belum bisa dihapus. Periksa koneksi Anda, lalu coba lagi."));
      closeDeleteConfirm();
    } finally {
      setIsDeleting(false);
    }
  }

  const columns: Column<StaffMember>[] = [
    {
      id: "name",
      header: "Nama",
      sortKey: "name",
      primary: true,
      cell: (u) => (
        <div className="flex items-center gap-2.5">
          <Avatar name={u.name} />
          <div className="min-w-0">
            <span className="block truncate font-semibold">{u.name}</span>
            <span className="block truncate text-[12.5px] text-muted-foreground">{u.email}</span>
          </div>
        </div>
      ),
    },
    { id: "title", header: "Jabatan", sortKey: "title", cell: (u) => u.title },
    { id: "role", header: "Peran", sortKey: "roleName", cell: (u) => <UserRoleBadge role={u.roleName ?? u.role} /> },
    { id: "phone", header: "Telepon", cell: (u) => u.phone },
    { id: "login", header: "Status masuk", cell: (u) => <StaffLoginStatus user={u} /> },
    { id: "status", header: "Status", sortKey: "isActive", cell: (u) => <StaffActiveBadge isActive={u.isActive} /> },
  ];

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Tim</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola anggota tim yang dapat mengakses {businessName}.</p>
        </div>
        {!writeLocked && (
          <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
            Undang anggota tim
          </Button>
        )}
      </div>

      {actionError && !modalOpen && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama atau email..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Select className="w-48" aria-label="Peran" value={list.filters.roleId || "Semua"} onChange={(e) => list.setFilter("roleId", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua peran</option>
          {staffRoles.map((r) => (
            <option key={r.id} value={String(r.id)}>
              {r.name}
            </option>
          ))}
        </Select>
        <Select className="w-40" aria-label="Status" value={list.filters.isActive || "Semua"} onChange={(e) => list.setFilter("isActive", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua status</option>
          <option value="true">Aktif</option>
          <option value="false">Nonaktif</option>
        </Select>
      </ListToolbar>

      <DataTable
        label="Daftar anggota tim"
        columns={columns}
        rows={users}
        rowId={(u) => u.id}
        rowLabel={(u) => u.name}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada anggota tim",
          description: "Undang anggota tim agar bisa masuk dan membantu mengelola proyek. Peran menentukan apa yang boleh mereka lakukan.",
          action: !writeLocked ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>Undang anggota tim</Button> : undefined,
        }}
        density={density}
        actions={(user) => {
          const isOwner = user.role === "Owner";
          const isSelf = user.id === currentStaffId;
          if (isOwner && !isSelf) return <span className="text-[12px] text-muted-foreground">Hanya dapat diubah oleh Owner sendiri</span>;
          if (!isOwner && !manageable(user)) return <span className="text-[12px] text-muted-foreground">Hanya dapat diubah oleh Owner</span>;
          return (
            <div className="flex items-center gap-1.5">
              <IconActionButton icon={Pencil} label="Ubah anggota tim" tone="neutral" onClick={() => openEditModal(user)} />
              {!isOwner && user.isActive && (
                <IconActionButton icon={Send} label={reinviteLabel(user)} tone="neutral" onClick={() => void handleReinvite(user)} />
              )}
              {!isOwner && (user.isActive ? (
                <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(user)} />
              ) : (
                <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(user)} />
              ))}
              {!isOwner && (
                <IconActionButton icon={Trash2} label="Hapus permanen" tone="danger" onClick={() => void openDeleteConfirm(user)} />
              )}
            </div>
          );
        }}
      />

      <UserFormModal
        key={editingUser?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmitCreate={handleCreate}
        onSubmitEdit={handleEdit}
        initialUser={editingUser}
      />

      <AccessLinkDialog
        open={linkShare !== null}
        onClose={() => setLinkShare(null)}
        link={linkShare?.link ?? null}
        recipient={linkShare?.recipient ?? { name: "" }}
        woName={businessName}
        notice={linkShare?.notice}
      />

      <Modal
        open={deleteTarget !== null}
        onClose={closeDeleteConfirm}
        title={deleteTarget ? `Hapus anggota tim — ${deleteTarget.name}` : "Hapus anggota tim"}
        size="sm"
      >
        {impactLoading ? (
          <p className="text-[13px] text-text-secondary">Memeriksa penugasan...</p>
        ) : (
          <div className="flex flex-col gap-4">
            <p className="flex items-start gap-2 text-[13px] text-text-secondary">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
              {deleteImpact && deleteImpact.length > 0 ? (
                <span>
                  Anggota tim ini masih ditugaskan sebagai PIC pada proyek berikut:{" "}
                  <strong className="text-text-primary">{deleteImpact.map((p) => p.name).join(", ")}</strong>. Jika dihapus, penugasan
                  PIC pada proyek tersebut akan kehilangan sumber datanya dan tidak dapat ditelusuri lagi. Apakah Anda yakin ingin
                  menghapus anggota tim ini secara permanen?
                </span>
              ) : (
                <span>Data ini akan dihapus secara permanen dan tidak dapat dikembalikan. Apakah Anda yakin?</span>
              )}
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={closeDeleteConfirm} disabled={isDeleting}>
                Batal
              </Button>
              <Button variant="danger" onClick={() => void handleConfirmDelete()} disabled={isDeleting}>
                {isDeleting ? "Menghapus..." : "Hapus anggota tim"}
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
