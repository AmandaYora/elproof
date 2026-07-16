import { useEffect, useState } from "react";
import { Plus, Pencil, UserCheck, UserX } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Avatar } from "@/shared/components/ui/Avatar";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { UserRoleBadge } from "@/modules/users/components/UserRoleBadge";
import { UserFormModal } from "@/modules/users/components/UserFormModal";
import { STAFF_ROLE_OPTIONS, type UserFormValues } from "@/modules/users/schemas/user.schema";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import type { StaffMember, StaffRole } from "@/modules/users/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export default function UserListPage() {
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

  async function handleSubmit(values: UserFormValues) {
    setActionError(null);
    try {
      if (editingUser) {
        await updateStaff(editingUser.id, values);
      } else {
        await createStaff(values);
      }
      closeModal();
      await fetchStaffPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan pengguna"));
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
              {users.map((user) => (
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
                    {user.role === "Owner" ? (
                      <span className="text-[12px] text-text-secondary">Dikelola via Platform Console</span>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <IconActionButton icon={Pencil} label="Ubah Pengguna" tone="neutral" onClick={() => openEditModal(user)} />
                        {user.isActive ? (
                          <IconActionButton icon={UserX} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(user)} />
                        ) : (
                          <IconActionButton icon={UserCheck} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(user)} />
                        )}
                      </div>
                    )}
                  </TD>
                </TR>
              ))}
            </TBody>
          </Table>
          <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </>
        )}
      </Card>

      <UserFormModal
        key={editingUser?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmit={(values) => void handleSubmit(values)}
        initialUser={editingUser}
      />
    </div>
  );
}
