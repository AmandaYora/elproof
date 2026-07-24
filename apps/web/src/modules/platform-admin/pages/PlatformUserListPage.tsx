import { useEffect, useState } from "react";
import { Plus, Pencil, KeyRound, UserX, UserCheck } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Badge } from "@/shared/components/ui/Badge";
import { Avatar } from "@/shared/components/ui/Avatar";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { PlatformAdminRoleBadge } from "@/modules/platform-admin/components/PlatformAdminRoleBadge";
import { PlatformAdminFormModal } from "@/modules/platform-admin/components/PlatformAdminFormModal";
import { PlatformAdminResetPasswordModal } from "@/modules/platform-admin/components/PlatformAdminResetPasswordModal";
import type { PlatformAdmin } from "@/modules/platform-admin/data/types";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import type {
  PlatformAdminCreateFormValues,
  PlatformAdminFormValues,
  ResetPlatformAdminPasswordFormValues,
} from "@/modules/platform-admin/schemas/platform-admin.schema";
import { PLATFORM_ADMIN_ROLE_OPTIONS } from "@/modules/platform-admin/schemas/platform-admin.schema";
import { getApiErrorMessage } from "@/shared/lib/api-error";

type CredentialReveal =
  | { kind: "create"; admin: PlatformAdmin; username: string; password: string }
  | { kind: "reset"; admin: PlatformAdmin; username: string; password: string };

export default function PlatformUserListPage() {
  const currentPlatformAdminId = useAuthStore((s) => s.currentPlatformAdminId);
  const admins = usePlatformAdminStore((s) => s.platformAdminPage);
  const meta = usePlatformAdminStore((s) => s.platformAdminPageMeta);
  const fetchPlatformAdminPage = usePlatformAdminStore((s) => s.fetchPlatformAdminPage);
  const registerPlatformAdmin = usePlatformAdminStore((s) => s.registerPlatformAdmin);
  const updatePlatformAdmin = usePlatformAdminStore((s) => s.updatePlatformAdmin);
  const togglePlatformAdminActive = usePlatformAdminStore((s) => s.togglePlatformAdminActive);
  const resetPlatformAdminPassword = usePlatformAdminStore((s) => s.resetPlatformAdminPassword);

  const [query, setQuery] = useState("");
  const [roleFilter, setRoleFilter] = useState<"Semua" | PlatformAdmin["role"]>("Semua");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingAdmin, setEditingAdmin] = useState<PlatformAdmin | undefined>(undefined);
  const [resetPasswordAdmin, setResetPasswordAdmin] = useState<PlatformAdmin | undefined>(undefined);
  const [credentialReveal, setCredentialReveal] = useState<CredentialReveal | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    setPage(1);
  }, [query, roleFilter]);

  useEffect(() => {
    void fetchPlatformAdminPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
  }, [fetchPlatformAdminPage, page, query, roleFilter]);

  function refetch() {
    return fetchPlatformAdminPage(page, query, roleFilter === "Semua" ? "" : roleFilter);
  }

  function openCreateModal() {
    setEditingAdmin(undefined);
    setModalOpen(true);
  }

  function openEditModal(admin: PlatformAdmin) {
    setEditingAdmin(admin);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingAdmin(undefined);
  }

  async function handleRegister(values: PlatformAdminCreateFormValues) {
    setActionError(null);
    try {
      const result = await registerPlatformAdmin(values);
      closeModal();
      await refetch();
      setCredentialReveal({ kind: "create", admin: result.admin, username: result.username, password: result.password });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menambahkan pengguna"));
    }
  }

  async function handleEditSubmit(values: PlatformAdminFormValues) {
    if (!editingAdmin) return;
    setActionError(null);
    try {
      await updatePlatformAdmin(editingAdmin.id, values);
      closeModal();
      await refetch();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui pengguna"));
    }
  }

  async function handleResetPasswordSubmit(values: ResetPlatformAdminPasswordFormValues) {
    if (!resetPasswordAdmin) return;
    setActionError(null);
    try {
      const result = await resetPlatformAdminPassword(resetPasswordAdmin.id, values.password);
      setCredentialReveal({ kind: "reset", admin: resetPasswordAdmin, username: result.username, password: result.password });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mereset password"));
    }
    setResetPasswordAdmin(undefined);
  }

  async function handleToggleActive(adminId: string) {
    setActionError(null);
    try {
      await togglePlatformAdminActive(adminId);
      await refetch();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status pengguna"));
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Pengguna</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola akun tim internal ElProof yang dapat mengakses Platform Console.</p>
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
        <Select className="w-44" value={roleFilter} onChange={(e) => setRoleFilter(e.target.value as "Semua" | PlatformAdmin["role"])}>
          <option value="Semua">Semua Role</option>
          {PLATFORM_ADMIN_ROLE_OPTIONS.map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </Select>
      </div>

      <Card>
        {admins.length === 0 ? (
          <EmptyState title="Tidak ada pengguna ditemukan" description="Ubah kata kunci pencarian atau filter role." />
        ) : (
          <>
            <CardList
              className="sm:hidden"
              items={admins}
              keyFor={(admin) => admin.id}
              renderItem={(admin) => {
                const isSelf = admin.id === currentPlatformAdminId;
                return (
                  <>
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex min-w-0 items-center gap-2.5">
                        <Avatar name={admin.name} />
                        <span className="truncate font-semibold text-text-primary">{admin.name}</span>
                      </div>
                      {admin.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <CardListField label="Jabatan" value={admin.title} />
                      <CardListField label="Role" value={<PlatformAdminRoleBadge role={admin.role} />} />
                      <CardListField label="Telepon" value={admin.phone} />
                      <CardListField label="Email" value={admin.email} />
                    </div>
                    <div className="flex items-center gap-1.5 pt-1">
                      <IconActionButton icon={Pencil} label="Ubah Pengguna" tone="neutral" onClick={() => openEditModal(admin)} />
                      <IconActionButton
                        icon={KeyRound}
                        label="Reset Password"
                        tone="info"
                        onClick={() => setResetPasswordAdmin(admin)}
                      />
                      {admin.isActive ? (
                        <IconActionButton
                          icon={UserX}
                          label={isSelf ? "Tidak dapat menonaktifkan akun sendiri" : "Nonaktifkan"}
                          tone="danger"
                          disabled={isSelf}
                          onClick={() => void handleToggleActive(admin.id)}
                        />
                      ) : (
                        <IconActionButton
                          icon={UserCheck}
                          label="Aktifkan"
                          tone="success"
                          onClick={() => void handleToggleActive(admin.id)}
                        />
                      )}
                    </div>
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
                {admins.map((admin) => {
                  const isSelf = admin.id === currentPlatformAdminId;
                  return (
                    <TR key={admin.id}>
                      <TD>
                        <div className="flex items-center gap-2.5">
                          <Avatar name={admin.name} />
                          <span className="font-semibold text-text-primary">{admin.name}</span>
                        </div>
                      </TD>
                      <TD>{admin.title}</TD>
                      <TD>
                        <PlatformAdminRoleBadge role={admin.role} />
                      </TD>
                      <TD>
                        <span className="block">{admin.phone}</span>
                        <span className="block text-[12.5px] text-text-secondary">{admin.email}</span>
                      </TD>
                      <TD>{admin.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}</TD>
                      <TD>
                        <div className="flex items-center gap-1.5">
                          <IconActionButton icon={Pencil} label="Ubah Pengguna" tone="neutral" onClick={() => openEditModal(admin)} />
                          <IconActionButton
                            icon={KeyRound}
                            label="Reset Password"
                            tone="info"
                            onClick={() => setResetPasswordAdmin(admin)}
                          />
                          {admin.isActive ? (
                            <IconActionButton
                              icon={UserX}
                              label={isSelf ? "Tidak dapat menonaktifkan akun sendiri" : "Nonaktifkan"}
                              tone="danger"
                              disabled={isSelf}
                              onClick={() => void handleToggleActive(admin.id)}
                            />
                          ) : (
                            <IconActionButton
                              icon={UserCheck}
                              label="Aktifkan"
                              tone="success"
                              onClick={() => void handleToggleActive(admin.id)}
                            />
                          )}
                        </div>
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

      <PlatformAdminFormModal
        key={editingAdmin?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        initialAdmin={editingAdmin}
        onSubmitCreate={handleRegister}
        onSubmitEdit={handleEditSubmit}
      />

      <PlatformAdminResetPasswordModal
        key={resetPasswordAdmin?.id ?? "reset-none"}
        open={resetPasswordAdmin !== undefined}
        onClose={() => setResetPasswordAdmin(undefined)}
        adminName={resetPasswordAdmin?.name}
        onSubmit={handleResetPasswordSubmit}
      />

      <Modal
        open={credentialReveal !== null}
        onClose={() => setCredentialReveal(null)}
        title={credentialReveal?.kind === "create" ? "Pengguna Berhasil Ditambahkan" : "Password Berhasil Direset"}
        description={
          credentialReveal
            ? credentialReveal.kind === "create"
              ? `${credentialReveal.admin.name} kini dapat masuk ke Platform Console dengan kredensial berikut.`
              : `Kredensial login ${credentialReveal.admin.name} telah diperbarui.`
            : undefined
        }
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
