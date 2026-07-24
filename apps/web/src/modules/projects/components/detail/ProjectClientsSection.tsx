import { useEffect, useState } from "react";
import { Pencil, RefreshCw, UserCog, Plus, Trash2, AlertTriangle } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Field } from "@/shared/components/ui/Input";
import { useClientStore } from "@/modules/clients/stores/useClientStore";
import {
  clientContactSchema,
  clientCreateSchema,
  representativeSchema,
  type ClientContactFormValues,
  type ClientCreateFormValues,
  type RepresentativeFormValues,
} from "@/modules/clients/schemas/client.schema";
import type { Client, ClientRole } from "@/modules/clients/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate } from "@/shared/lib/formatters";

const ROLE_ORDER: ClientRole[] = ["Bride", "Groom", "Family Representative"];

const ROLE_LABEL: Record<ClientRole, string> = {
  Bride: "Pengantin Wanita",
  Groom: "Pengantin Pria",
  "Family Representative": "Wedding Representative Keluarga",
};

type ModalMode = "contact" | "replace" | "reset";

interface ModalTarget {
  clientId: string;
  mode: ModalMode;
}

// A stable reference — returning a fresh `[]` from a Zustand selector on
// every call defeats useSyncExternalStore's reference check and causes an
// infinite render loop (React error #185).
const EMPTY_CLIENTS: Client[] = [];

export function ProjectClientsSection({ projectId }: { projectId: string }) {
  const clients = useClientStore((s) => s.clientsByProject[projectId] ?? EMPTY_CLIENTS);
  const fetchClients = useClientStore((s) => s.fetchClients);
  const createClient = useClientStore((s) => s.createClient);
  const updateContact = useClientStore((s) => s.updateContact);
  const toggleActive = useClientStore((s) => s.toggleActive);
  const deleteClient = useClientStore((s) => s.deleteClient);
  const resetCredential = useClientStore((s) => s.resetCredential);
  const replaceRepresentative = useClientStore((s) => s.replaceRepresentative);

  const [modalTarget, setModalTarget] = useState<ModalTarget | null>(null);
  const [createRole, setCreateRole] = useState<ClientRole | null>(null);
  const [deletingClientId, setDeletingClientId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchClients(projectId);
  }, [projectId, fetchClients]);

  const activeTarget = modalTarget ? clients.find((c) => c.id === modalTarget.clientId) ?? null : null;

  async function handleToggleActive(id: string) {
    setActionError(null);
    try {
      await toggleActive(projectId, id);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status client"));
    }
  }

  async function handleContactSubmit(values: ClientContactFormValues) {
    if (!modalTarget) return;
    setActionError(null);
    try {
      await updateContact(projectId, modalTarget.clientId, values);
      setModalTarget(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui kontak client"));
    }
  }

  async function handleDelete(id: string) {
    setActionError(null);
    try {
      await deleteClient(projectId, id);
      setDeletingClientId(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus client"));
    }
  }

  async function handleReplaceSubmit(values: RepresentativeFormValues) {
    if (!modalTarget) return;
    setActionError(null);
    try {
      await replaceRepresentative(projectId, modalTarget.clientId, values);
      setModalTarget(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengganti wedding representative"));
    }
  }

  async function handleResetSubmit(password: string) {
    if (!modalTarget) return;
    setActionError(null);
    try {
      await resetCredential(projectId, modalTarget.clientId, password);
      setModalTarget(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mereset kredensial client"));
    }
  }

  async function handleCreateSubmit(values: ClientCreateFormValues) {
    if (!createRole) return;
    setActionError(null);
    try {
      await createClient(projectId, createRole, values);
      setCreateRole(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menambahkan client"));
    }
  }

  return (
    <div id="client">
      <Card>
        <CardHeader
          title="Client Project"
          subtitle="Kontak pengantin dan perwakilan keluarga yang terdaftar untuk project ini."
        />
        <CardContent className="flex flex-col gap-3">
          {actionError && (
            <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
          )}
          {ROLE_ORDER.map((role) => {
            const client = clients.find((c) => c.role === role);
            if (!client) {
              return (
                <div
                  key={role}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-dashed border-border px-4 py-3 text-[13px] text-text-secondary"
                >
                  <span>Belum ada data {ROLE_LABEL[role]}.</span>
                  <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setCreateRole(role)}>
                    Tambah {ROLE_LABEL[role]}
                  </Button>
                </div>
              );
            }
            return (
              <div key={client.id} className="rounded-md border border-border p-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="text-[11.5px] font-semibold uppercase tracking-wide text-text-secondary">
                      {ROLE_LABEL[role]}
                    </p>
                    {client.relationNote && <p className="mt-1 text-[13px] text-text-secondary">{client.relationNote}</p>}
                    <p className="mt-1.5 text-sm font-semibold text-text-primary">{client.name}</p>
                    <p className="text-[13px] text-text-secondary">
                      {client.phone} &middot; {client.email}
                    </p>
                  </div>
                  <Badge tone={client.isActive ? "success" : "neutral"}>{client.isActive ? "Aktif" : "Nonaktif"}</Badge>
                </div>

                <div className="mt-3 flex flex-wrap items-center gap-2 text-[12.5px] text-text-secondary">
                  <span>
                    {client.lastCredentialResetAt
                      ? `Direset terakhir: ${formatDate(client.lastCredentialResetAt)}`
                      : "Belum pernah direset"}
                  </span>
                </div>

                <div className="mt-3 flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={<Pencil className="h-3.5 w-3.5" />}
                    onClick={() => setModalTarget({ clientId: client.id, mode: "contact" })}
                  >
                    Ubah Kontak
                  </Button>
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={<RefreshCw className="h-3.5 w-3.5" />}
                    onClick={() => setModalTarget({ clientId: client.id, mode: "reset" })}
                  >
                    Reset Credential
                  </Button>
                  <Button
                    size="sm"
                    variant={client.isActive ? "danger" : "secondary"}
                    onClick={() => void handleToggleActive(client.id)}
                  >
                    {client.isActive ? "Nonaktifkan" : "Aktifkan"}
                  </Button>
                  {role === "Family Representative" && (
                    <Button
                      size="sm"
                      variant="secondary"
                      icon={<UserCog className="h-3.5 w-3.5" />}
                      onClick={() => setModalTarget({ clientId: client.id, mode: "replace" })}
                    >
                      Ganti Wedding Representative
                    </Button>
                  )}
                  <Button
                    size="sm"
                    variant="danger"
                    icon={<Trash2 className="h-3.5 w-3.5" />}
                    onClick={() => setDeletingClientId(client.id)}
                  >
                    Hapus Client
                  </Button>
                </div>

                {deletingClientId === client.id && (
                  <div className="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
                    <span className="flex items-center gap-2 text-[13px] font-medium text-danger">
                      <AlertTriangle className="h-4 w-4 shrink-0" />
                      Yakin ingin menghapus {ROLE_LABEL[role]} ini? Tindakan ini permanen — akun login yang terkait
                      (jika ada) ikut dinonaktifkan, dan slot {ROLE_LABEL[role]} untuk project ini akan kosong kembali.
                    </span>
                    <span className="flex shrink-0 gap-2">
                      <Button variant="secondary" size="sm" onClick={() => setDeletingClientId(null)}>Batal</Button>
                      <Button variant="danger" size="sm" onClick={() => void handleDelete(client.id)}>Ya, Hapus</Button>
                    </span>
                  </div>
                )}
              </div>
            );
          })}
        </CardContent>
      </Card>

      {activeTarget && modalTarget?.mode === "contact" && (
        <EditContactModal
          key={activeTarget.id}
          client={activeTarget}
          onClose={() => setModalTarget(null)}
          onSubmit={(values) => void handleContactSubmit(values)}
        />
      )}

      {activeTarget && modalTarget?.mode === "replace" && (
        <ReplaceRepresentativeModal
          key={activeTarget.id}
          client={activeTarget}
          onClose={() => setModalTarget(null)}
          onSubmit={(values) => void handleReplaceSubmit(values)}
        />
      )}

      {activeTarget && modalTarget?.mode === "reset" && (
        <ResetCredentialModal
          key={activeTarget.id}
          client={activeTarget}
          onClose={() => setModalTarget(null)}
          onSubmit={(password) => void handleResetSubmit(password)}
        />
      )}

      {createRole && (
        <CreateClientModal
          role={createRole}
          onClose={() => setCreateRole(null)}
          onSubmit={(values) => void handleCreateSubmit(values)}
        />
      )}
    </div>
  );
}

function EditContactModal({
  client,
  onClose,
  onSubmit,
}: {
  client: Client;
  onClose: () => void;
  onSubmit: (values: ClientContactFormValues) => void;
}) {
  const [values, setValues] = useState<ClientContactFormValues>({ name: client.name, phone: client.phone, email: client.email });
  const [errors, setErrors] = useState<Partial<Record<keyof ClientContactFormValues, string>>>({});

  function set<K extends keyof ClientContactFormValues>(key: K, value: ClientContactFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = clientContactSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ClientContactFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ClientContactFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="Ubah Kontak"
      description={`Perbarui informasi kontak untuk ${client.name}.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan Perubahan</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} />
        </Field>
        <Field label="No. Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} />
        </Field>
        <Field label="Email" required hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} />
        </Field>
        <Field label="Username">
          <Input value={client.username} disabled />
        </Field>
      </div>
    </Modal>
  );
}

function ReplaceRepresentativeModal({
  client,
  onClose,
  onSubmit,
}: {
  client: Client;
  onClose: () => void;
  onSubmit: (values: RepresentativeFormValues) => void;
}) {
  const [values, setValues] = useState<RepresentativeFormValues>({ name: "", phone: "", email: "", relationNote: "" });
  const [errors, setErrors] = useState<Partial<Record<keyof RepresentativeFormValues, string>>>({});

  function set<K extends keyof RepresentativeFormValues>(key: K, value: RepresentativeFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = representativeSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof RepresentativeFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof RepresentativeFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="Ganti Wedding Representative"
      description={`Data representative keluarga saat ini (${client.name}) akan digantikan oleh data baru. Kredensial login yang sudah ada tetap dipakai (gunakan Reset Credential terpisah bila perlu).`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan Representative Baru</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama Representative Baru" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="Nama lengkap" />
        </Field>
        <Field label="Hubungan dengan Pengantin" required hint={errors.relationNote}>
          <Input
            value={values.relationNote}
            onChange={(e) => set("relationNote", e.target.value)}
            placeholder="cth. Ayah kandung mempelai wanita"
          />
        </Field>
        <Field label="No. Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} />
        </Field>
        <Field label="Email" required hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} />
        </Field>
        <Field label="Username">
          <Input value={client.username} disabled />
        </Field>
      </div>
    </Modal>
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

function CreateClientModal({
  role,
  onClose,
  onSubmit,
}: {
  role: ClientRole;
  onClose: () => void;
  onSubmit: (values: ClientCreateFormValues) => void;
}) {
  const [values, setValues] = useState<ClientCreateFormValues>({ name: "", phone: "", email: "", relationNote: "", username: "", password: "" });
  const [errors, setErrors] = useState<Partial<Record<keyof ClientCreateFormValues, string>>>({});

  function set<K extends keyof ClientCreateFormValues>(key: K, value: ClientCreateFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = clientCreateSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ClientCreateFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ClientCreateFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={`Tambah ${ROLE_LABEL[role]}`}
      description="Membuat data client baru sekaligus akun login untuk Client Portal."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan Client</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="Nama lengkap" />
        </Field>
        {role === "Family Representative" && (
          <Field label="Hubungan dengan Pengantin" hint={errors.relationNote}>
            <Input
              value={values.relationNote}
              onChange={(e) => set("relationNote", e.target.value)}
              placeholder="cth. Ayah kandung mempelai wanita"
            />
          </Field>
        )}
        <Field label="No. Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} />
        </Field>
        <Field label="Email" required hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} />
        </Field>
        <Field label="Username" required hint={errors.username}>
          <Input value={values.username} onChange={(e) => set("username", e.target.value)} placeholder="cth. budi.rahman" />
        </Field>
        <Field label="Password Login" required hint={errors.password}>
          <Input type="password" value={values.password} onChange={(e) => set("password", e.target.value)} placeholder="Minimal 6 karakter" />
        </Field>
      </div>
    </Modal>
  );
}
