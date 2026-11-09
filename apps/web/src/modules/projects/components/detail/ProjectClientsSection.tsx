import { useEffect, useRef, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Pencil, Send, UserCog, Plus, Trash2, AlertTriangle } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
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
import { AccessLinkDialog } from "@/shared/components/AccessLinkDialog";
import type { AccessLink, AccessLinkRecipient } from "@/shared/lib/access-link";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { APP_NAME } from "@/shared/constants/brand";
import { ClientLoginStatusBadge, clientLinkActionLabel } from "@/modules/clients/components/ClientLoginStatus";
import { PortalRoleSelect } from "@/modules/clients/components/PortalRoleSelect";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const CONTACT_LABELS: Record<string, string> = { name: "Nama", phone: "Nomor WhatsApp", email: "Email", accessRoleId: "Akses portal" };
const REPLACE_LABELS: Record<string, string> = { ...CONTACT_LABELS, name: "Nama representative baru", relationNote: "Hubungan dengan pengantin" };
const CREATE_LABELS: Record<string, string> = { ...CONTACT_LABELS, relationNote: "Hubungan dengan pengantin", username: "Nama pengguna" };

const ROLE_ORDER: ClientRole[] = ["Bride", "Groom", "Family Representative"];

const ROLE_LABEL: Record<ClientRole, string> = {
  Bride: "Pengantin Wanita",
  Groom: "Pengantin Pria",
  "Family Representative": "Wedding Representative Keluarga",
};

type ModalMode = "contact" | "replace";

interface LinkShare {
  link: AccessLink;
  recipient: AccessLinkRecipient;
  notice?: string;
}

function clientRecipient(c: Pick<Client, "name" | "phone" | "username">): AccessLinkRecipient {
  return { name: c.name, phone: c.phone, username: c.username };
}

interface ModalTarget {
  clientId: string;
  mode: ModalMode;
}

// A stable reference — returning a fresh `[]` from a Zustand selector on
// every call defeats useSyncExternalStore's reference check and causes an
// infinite render loop (React error #185).
const EMPTY_CLIENTS: Client[] = [];

export function ProjectClientsSection({ projectId }: { projectId: string }) {
  // Wedding Planner reads client data for their own project fine (backend
  // already scopes it), but every write action here is Owner/Admin only
  // (see PLAN.md's RBAC section) — hide the buttons for that role instead of
  // letting them click through to a 403.
  const canManage = useCan("client.manage");
  const clients = useClientStore((s) => s.clientsByProject[projectId] ?? EMPTY_CLIENTS);
  const fetchClients = useClientStore((s) => s.fetchClients);
  const createClient = useClientStore((s) => s.createClient);
  const updateContact = useClientStore((s) => s.updateContact);
  const toggleActive = useClientStore((s) => s.toggleActive);
  const deleteClient = useClientStore((s) => s.deleteClient);
  const issueAccessLink = useClientStore((s) => s.issueAccessLink);
  const replaceRepresentative = useClientStore((s) => s.replaceRepresentative);
  const woName = useTenantBrandingStore((s) => s.businessName) ?? APP_NAME;
  const [linkShare, setLinkShare] = useState<LinkShare | null>(null);

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

  // The three submit handlers (contact, replace, create) do not catch: errors
  // propagate to the form, which shows a 422 on its fields and any other failure inside the modal.
  async function handleContactSubmit(values: ClientContactFormValues) {
    if (!modalTarget) return;
    setActionError(null);
    await updateContact(projectId, modalTarget.clientId, values);
    setModalTarget(null);
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
    const { client, link } = await replaceRepresentative(projectId, modalTarget.clientId, values);
    setModalTarget(null);
    setLinkShare({
      link,
      recipient: clientRecipient(client),
      notice: "Perwakilan diganti. Akses perwakilan lama sudah dicabut.",
    });
  }

  async function handleSendLink(client: Client) {
    setActionError(null);
    try {
      const link = await issueAccessLink(projectId, client.id);
      setLinkShare({ link, recipient: clientRecipient(client) });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membuat link"));
    }
  }

  async function handleCreateSubmit(values: ClientCreateFormValues) {
    if (!createRole) return;
    setActionError(null);
    const { client, link } = await createClient(projectId, createRole, values);
    setCreateRole(null);
    setLinkShare({ link, recipient: clientRecipient(client), notice: `${client.name} ditambahkan dengan nama pengguna ${client.username}.` });
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
            <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
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
                  {canManage && (
                    <Button
                      size="sm"
                      variant="secondary"
                      icon={<Plus className="h-3.5 w-3.5" />}
                      onClick={() => {
                        setActionError(null);
                        setCreateRole(role);
                      }}
                    >
                      Tambah {ROLE_LABEL[role]}
                    </Button>
                  )}
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
                      {[client.phone, client.email].filter(Boolean).join(" · ")}
                    </p>
                  </div>
                  <ClientLoginStatusBadge client={client} />
                </div>

                <div className="mt-3 flex flex-wrap items-center gap-2 text-[12.5px] text-text-secondary">
                  <span>Nama pengguna: {client.username}</span>
                  {client.accessRoleName && <span>· Akses portal: {client.accessRoleName}</span>}
                  {client.lastCredentialResetAt && <span>· Link atur ulang terakhir: {formatDate(client.lastCredentialResetAt)}</span>}
                </div>

                {canManage && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Button
                      size="sm"
                      variant="secondary"
                      icon={<Pencil className="h-3.5 w-3.5" />}
                      onClick={() => setModalTarget({ clientId: client.id, mode: "contact" })}
                    >
                      Ubah Kontak
                    </Button>
                    {client.isActive && (
                      <Button
                        size="sm"
                        variant="secondary"
                        icon={<Send className="h-3.5 w-3.5" />}
                        onClick={() => void handleSendLink(client)}
                      >
                        {clientLinkActionLabel(client)}
                      </Button>
                    )}
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
                )}

                {deletingClientId === client.id && (
                  <div className="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
                    <span className="flex items-center gap-2 text-[13px] font-medium text-danger-text">
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
          onSubmit={handleContactSubmit}
        />
      )}

      {activeTarget && modalTarget?.mode === "replace" && (
        <ReplaceRepresentativeModal
          key={activeTarget.id}
          client={activeTarget}
          onClose={() => setModalTarget(null)}
          onSubmit={handleReplaceSubmit}
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

      {createRole && (
        <CreateClientModal
          role={createRole}
          onClose={() => setCreateRole(null)}
          onSubmit={handleCreateSubmit}
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
  onSubmit: (values: ClientContactFormValues) => Promise<void>;
}) {
  const form = useZodForm({
    schema: clientContactSchema,
    initial: { name: client.name, phone: client.phone, email: client.email, accessRoleId: client.accessRoleId ?? 0 },
  });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Kontak belum tersimpan. Periksa koneksi lalu coba lagi." });

  return (
    <Modal
      open
      onClose={onClose}
      title="Ubah Kontak"
      description={`Perbarui informasi kontak untuk ${client.name}.`}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="edit-contact-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan perubahan"}
          </Button>
        </>
      }
    >
      <form id="edit-contact-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={CONTACT_LABELS} root={() => formRef.current} />
        <Field label="Nama" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} />
        </Field>
        <Field label="Nomor WhatsApp" required name="phone" error={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} onBlur={() => blur("phone")} inputMode="tel" />
        </Field>
        <Field label="Email" name="email" error={errors.email} hint="Opsional">
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} />
        </Field>
        <div data-field="accessRoleId">
          <PortalRoleSelect value={values.accessRoleId ?? 0} onChange={(id) => set("accessRoleId", id)} />
        </div>
        <Field label="Username">
          <Input value={client.username} disabled />
        </Field>
      </form>
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
  onSubmit: (values: RepresentativeFormValues) => Promise<void>;
}) {
  const form = useZodForm({
    schema: representativeSchema,
    // No portal role here (the replacement gets the default): accessRoleId stays absent, as before.
    initial: { name: "", phone: "", email: "", relationNote: "" } as Required<RepresentativeFormValues>,
  });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Perwakilan belum diganti. Periksa koneksi lalu coba lagi." });

  return (
    <Modal
      open
      onClose={onClose}
      title="Ganti Wedding Representative"
      description={`${client.name} akan digantikan. Akses login ${client.name} langsung dicabut, lalu Anda mendapat link undangan untuk perwakilan baru.`}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="replace-representative-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Ganti dan buat undangan"}
          </Button>
        </>
      }
    >
      <form id="replace-representative-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={REPLACE_LABELS} root={() => formRef.current} />
        <Field label="Nama Representative Baru" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="Nama lengkap" />
        </Field>
        <Field label="Hubungan dengan Pengantin" required name="relationNote" error={errors.relationNote}>
          <Input
            value={values.relationNote}
            onChange={(e) => set("relationNote", e.target.value)}
            onBlur={() => blur("relationNote")}
            placeholder="cth. Ayah kandung mempelai wanita"
          />
        </Field>
        <Field label="Nomor WhatsApp" required name="phone" error={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} onBlur={() => blur("phone")} inputMode="tel" />
        </Field>
        <Field label="Email" name="email" error={errors.email} hint="Opsional">
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} />
        </Field>
        <Field label="Username">
          <Input value={client.username} disabled />
        </Field>
      </form>
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
  onSubmit: (values: ClientCreateFormValues) => Promise<void>;
}) {
  const form = useZodForm({ schema: clientCreateSchema, initial: { name: "", phone: "", email: "", relationNote: "", username: "", accessRoleId: 0 } });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Client belum tersimpan. Periksa koneksi lalu coba lagi." });

  return (
    <Modal
      open
      onClose={onClose}
      title={`Tambah ${ROLE_LABEL[role]}`}
      description="Client menerima link undangan dan membuat kata sandinya sendiri untuk Client Portal."
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="create-client-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan dan buat undangan"}
          </Button>
        </>
      }
    >
      <form id="create-client-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={CREATE_LABELS} root={() => formRef.current} />
        <Field label="Nama" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="Nama lengkap" />
        </Field>
        {role === "Family Representative" && (
          <Field label="Hubungan dengan Pengantin" name="relationNote" error={errors.relationNote}>
            <Input
              value={values.relationNote}
              onChange={(e) => set("relationNote", e.target.value)}
              onBlur={() => blur("relationNote")}
              placeholder="cth. Ayah kandung mempelai wanita"
            />
          </Field>
        )}
        <Field label="Nomor WhatsApp" required name="phone" error={errors.phone} hint="Link undangan dikirim ke nomor ini">
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} onBlur={() => blur("phone")} placeholder="0812 3456 7890" inputMode="tel" />
        </Field>
        <Field label="Email" name="email" error={errors.email} hint="Opsional. Client juga bisa masuk memakai email ini.">
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} />
        </Field>
        <div data-field="accessRoleId">
          <PortalRoleSelect value={values.accessRoleId ?? 0} onChange={(id) => set("accessRoleId", id)} allowDefault />
        </div>
        <Field label="Nama pengguna" name="username" error={errors.username} hint="Opsional. Kosongkan agar dibuat otomatis dari nama.">
          <Input value={values.username} onChange={(e) => set("username", e.target.value.toLowerCase())} onBlur={() => blur("username")} placeholder="cth. budi.rahman" />
        </Field>
      </form>
    </Modal>
  );
}
