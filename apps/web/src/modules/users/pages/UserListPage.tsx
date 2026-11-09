import { useEffect, useState } from "react";
import { Plus, Pencil, UserCheck, UserX, Send, Trash2, AlertTriangle } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Avatar } from "@/shared/components/ui/Avatar";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
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
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import type { AffectedProject } from "@/shared/types/delete-impact";
import { useWriteLocked } from "@/shared/stores/usePermissionStore";

interface LinkShare {
  link: AccessLink;
  recipient: AccessLinkRecipient;
  notice?: string;
}

function recipientFor(user: Pick<StaffMember, "name" | "phone" | "email" | "role">): AccessLinkRecipient {
  return { name: user.name, phone: user.phone, email: user.email, roleLabel: user.role };
}

/** Status column: invitation state first, then active/inactive. */
function StaffStatusBadge({ user }: { user: StaffMember }) {
  if (!user.isActive) return <Badge tone="neutral">Nonaktif</Badge>;
  if (user.loginStatus === "invited") return <Badge tone="warning">Menunggu undangan</Badge>;
  return <Badge tone="success">Aktif</Badge>;
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
  const toggleStaffActive = useStaffStore((s) => s.toggleStaffActive);
  const fetchStaffDeleteImpact = useStaffStore((s) => s.fetchStaffDeleteImpact);
  const deleteStaff = useStaffStore((s) => s.deleteStaff);

  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query);
  // "Semua" or a role id.
  const [roleFilter, setRoleFilter] = useState<string>("Semua");
  const staffRoles = useRolesStore((s) => s.roles.staff);
  const fetchRoles = useRolesStore((s) => s.fetchRoles);
  useEffect(() => {
    void fetchRoles("staff").catch(() => undefined);
  }, [fetchRoles]);
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<StaffMember | undefined>(undefined);
  const [linkShare, setLinkShare] = useState<LinkShare | null>(null);
  const [creating, setCreating] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<StaffMember | null>(null);
  const [deleteImpact, setDeleteImpact] = useState<AffectedProject[] | null>(null);
  const [impactLoading, setImpactLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    setPage(1);
  }, [debouncedQuery, roleFilter]);

  useEffect(() => {
    void fetchStaffPage(page, debouncedQuery, roleFilter === "Semua" ? "" : roleFilter);
  }, [fetchStaffPage, page, debouncedQuery, roleFilter]);

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

  async function handleCreate(values: UserFormValues) {
    setActionError(null);
    setCreating(true);
    try {
      const result = await createStaff(values);
      closeModal();
      await fetchStaffPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
      setLinkShare({ link: result.invitation, recipient: recipientFor(result.staff), notice: `${result.staff.name} ditambahkan.` });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membuat undangan"));
    } finally {
      setCreating(false);
    }
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
      setActionError(getApiErrorMessage(err, "Gagal membuat link undangan"));
    }
  }

  function reinviteLabel(user: StaffMember) {
    return user.loginStatus === "invited" ? "Kirim ulang undangan" : "Undang ulang (ganti akun Google)";
  }

  async function handleEdit(values: UserFormValues) {
    if (!editingUser) return;
    setActionError(null);
    try {
      await updateStaff(editingUser.id, values);
      closeModal();
      await fetchStaffPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan perubahan pengguna"));
    }
  }

  async function handleToggleActive(user: StaffMember) {
    setActionError(null);
    try {
      await toggleStaffActive(user.id);
      await fetchStaffPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status pengguna"));
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
      await fetchStaffPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus pengguna"));
      closeDeleteConfirm();
    } finally {
      setIsDeleting(false);
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Pengguna</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola akun staff internal yang dapat mengakses {businessName}.</p>
        </div>
        {!writeLocked && (
          <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
            Undang Pengguna
          </Button>
        )}
      </div>

      {actionError && !modalOpen && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama atau email..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Select className="w-48" value={roleFilter} onChange={(e) => setRoleFilter(e.target.value)}>
          <option value="Semua">Semua Peran</option>
          {staffRoles.map((r) => (
            <option key={r.id} value={String(r.id)}>
              {r.name}
            </option>
          ))}
        </Select>
      </div>

      <Card>
        {users.length === 0 ? (
          <EmptyState title="Tidak ada pengguna ditemukan" description="Ubah kata kunci pencarian atau filter role." />
        ) : (
          <>
          <CardList
            className="sm:hidden"
            items={users}
            keyFor={(user) => user.id}
            renderItem={(user) => {
              const isOwner = user.role === "Owner";
              const isSelf = user.id === currentStaffId;
              return (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex min-w-0 items-center gap-2.5">
                      <Avatar name={user.name} />
                      <span className="truncate font-semibold text-text-primary">{user.name}</span>
                    </div>
                    <StaffStatusBadge user={user} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Jabatan" value={user.title} />
                    <CardListField label="Role" value={<UserRoleBadge role={user.role} />} />
                    <CardListField label="Telepon" value={user.phone} />
                    <CardListField label="Email" value={user.email} />
                  </div>
                  {isOwner && !isSelf ? (
                    <p className="pt-1 text-[12px] text-text-secondary">Hanya dapat diubah oleh Owner sendiri</p>
                  ) : (
                    <div className="flex items-center gap-1.5 pt-1">
                      <IconActionButton icon={Pencil} label="Ubah Pengguna" tone="neutral" onClick={() => openEditModal(user)} />
                      {!isOwner && user.isActive && (
                        <IconActionButton icon={Send} label={reinviteLabel(user)} tone="neutral" onClick={() => void handleReinvite(user)} />
                      )}
                      {!isOwner && (user.isActive ? (
                        <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(user)} />
                      ) : (
                        <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(user)} />
                      ))}
                      {!isOwner && (
                        <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => void openDeleteConfirm(user)} />
                      )}
                    </div>
                  )}
                </>
              );
            }}
          />
          <div className="hidden sm:block">
          <Table>
            <THead>
              <TR>
                <TH>Nama</TH>
                <TH>Jabatan</TH>
                <TH>Role</TH>
                <TH>Kontak</TH>
                <TH>Status</TH>
                <TH>Aksi</TH>
              </TR>
            </THead>
            <TBody>
              {users.map((user) => {
                const isOwner = user.role === "Owner";
                const isSelf = user.id === currentStaffId;
                return (
                <TR key={user.id}>
                  <TD>
                    <div className="flex items-center gap-2.5">
                      <Avatar name={user.name} />
                      <span className="font-semibold text-text-primary">{user.name}</span>
                    </div>
                  </TD>
                  <TD>{user.title}</TD>
                  <TD><UserRoleBadge role={user.role} /></TD>
                  <TD>
                    <span className="block">{user.phone}</span>
                    <span className="block text-[12.5px] text-text-secondary">{user.email}</span>
                  </TD>
                  <TD>
                    <StaffStatusBadge user={user} />
                  </TD>
                  <TD>
                    {isOwner && !isSelf ? (
                      <span className="text-[12px] text-text-secondary">Hanya dapat diubah oleh Owner sendiri</span>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <IconActionButton icon={Pencil} label="Ubah Pengguna" tone="neutral" onClick={() => openEditModal(user)} />
                      {!isOwner && user.isActive && (
                        <IconActionButton icon={Send} label={reinviteLabel(user)} tone="neutral" onClick={() => void handleReinvite(user)} />
                      )}
                        {!isOwner && (user.isActive ? (
                          <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(user)} />
                        ) : (
                          <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(user)} />
                        ))}
                        {!isOwner && (
                          <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => void openDeleteConfirm(user)} />
                        )}
                      </div>
                    )}
                  </TD>
                </TR>
                );
              })}
            </TBody>
          </Table>
          </div>
          <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </>
        )}
      </Card>

      <UserFormModal
        key={editingUser?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmitCreate={(values) => void handleCreate(values)}
        onSubmitEdit={(values) => void handleEdit(values)}
        initialUser={editingUser}
        submitting={creating}
        error={modalOpen ? actionError : null}
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
        title={deleteTarget ? `Hapus Pengguna — ${deleteTarget.name}` : "Hapus Pengguna"}
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
                  Pengguna ini masih ditugaskan sebagai PIC pada project berikut:{" "}
                  <strong className="text-text-primary">{deleteImpact.map((p) => p.name).join(", ")}</strong>. Jika dihapus, penugasan
                  PIC pada project tersebut akan kehilangan sumber datanya dan tidak dapat ditelusuri lagi. Apakah Anda yakin ingin
                  menghapus pengguna ini secara permanen?
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
                {isDeleting ? "Menghapus..." : "Ya, Hapus Permanen"}
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
