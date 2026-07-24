import { useEffect, useState } from "react";
import { Plus, Pencil, UserCheck, UserX, KeyRound } from "lucide-react";
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
import { STAFF_ROLE_OPTIONS, type UserFormValues, type UserCreateFormValues } from "@/modules/users/schemas/user.schema";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import type { StaffMember, StaffRole } from "@/modules/users/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";

interface CredentialReveal {
  name: string;
  username: string;
  password: string;
}

export default function UserListPage() {
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const users = useStaffStore((s) => s.staffPage);
  const meta = useStaffStore((s) => s.staffPageMeta);
  const fetchStaffPage = useStaffStore((s) => s.fetchStaffPage);
  const createStaff = useStaffStore((s) => s.createStaff);
  const updateStaff = useStaffStore((s) => s.updateStaff);
  const toggleStaffActive = useStaffStore((s) => s.toggleStaffActive);

  const [query, setQuery] = useState("");
  const [roleFilter, setRoleFilter] = useState<"Semua" | StaffRole>("Semua");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<StaffMember | undefined>(undefined);
  const [credentialReveal, setCredentialReveal] = useState<CredentialReveal | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    setPage(1);
  }, [query, roleFilter]);

  useEffect(() => {
    void fetchStaffPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
  }, [fetchStaffPage, page, query, roleFilter]);

  function openCreateModal() {
    setEditingUser(undefined);
    setModalOpen(true);
  }

  function openEditModal(user: StaffMember) {
    setEditingUser(user);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingUser(undefined);
  }

  async function handleCreate(values: UserCreateFormValues) {
    setActionError(null);
    try {
      const result = await createStaff(values);
      closeModal();
      await fetchStaffPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
      setCredentialReveal({ name: result.staff.name, username: result.username, password: result.password });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menambahkan pengguna"));
    }
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

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Pengguna</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola akun staff internal yang dapat mengakses WO Console.</p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
          Tambah Pengguna
        </Button>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama atau email..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Select className="w-44" value={roleFilter} onChange={(e) => setRoleFilter(e.target.value as "Semua" | StaffRole)}>
          <option value="Semua">Semua Role</option>
          {STAFF_ROLE_OPTIONS.map((r) => (
            <option key={r} value={r}>{r}</option>
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
                    {user.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
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
                      {!isOwner && (user.isActive ? (
                        <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(user)} />
                      ) : (
                        <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(user)} />
                      ))}
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
                    {user.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
                  </TD>
                  <TD>
                    {isOwner && !isSelf ? (
                      <span className="text-[12px] text-text-secondary">Hanya dapat diubah oleh Owner sendiri</span>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <IconActionButton icon={Pencil} label="Ubah Pengguna" tone="neutral" onClick={() => openEditModal(user)} />
                        {!isOwner && (user.isActive ? (
                          <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(user)} />
                        ) : (
                          <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(user)} />
                        ))}
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
      />

      <Modal
        open={credentialReveal !== null}
        onClose={() => setCredentialReveal(null)}
        title="Pengguna Berhasil Ditambahkan"
        description={credentialReveal ? `${credentialReveal.name} kini dapat masuk ke WO Console dengan kredensial berikut.` : undefined}
        size="sm"
        footer={<Button onClick={() => setCredentialReveal(null)}>Selesai</Button>}
      >
        {credentialReveal && (
          <div className="flex flex-col gap-3 text-[13px]">
            <div className="flex items-center gap-2 text-text-secondary">
              <KeyRound className="h-3.5 w-3.5 shrink-0" />
              Kredensial akun:
            </div>
            <div className="flex items-center justify-between rounded-md bg-surface-muted px-3.5 py-2.5">
              <span className="text-text-secondary">Username</span>
              <span className="font-semibold text-text-primary">{credentialReveal.username}</span>
            </div>
            <div className="flex items-center justify-between rounded-md bg-surface-muted px-3.5 py-2.5">
              <span className="text-text-secondary">Password</span>
              <span className="font-mono font-semibold text-text-primary">{credentialReveal.password}</span>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
