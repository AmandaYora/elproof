import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import { userSchema, type UserFormValues } from "@/modules/users/schemas/user.schema";
import { OWNER_ROLE_NAME, type StaffMember } from "@/modules/users/types";
import { useRolesStore } from "@/modules/access/stores/useRolesStore";
import { useEffect, useRef } from "react";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { APP_NAME } from "@/shared/constants/brand";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { name: "Nama lengkap", roleId: "Peran", title: "Jabatan", phone: "Nomor WhatsApp", email: "Email akun Google" };

function toFormState(user?: StaffMember): UserFormValues {
  if (!user) return { name: "", title: "", roleId: 0, email: "", phone: "" };
  return { name: user.name, title: user.title, roleId: user.roleId, email: user.email, phone: user.phone };
}

interface UserFormModalProps {
  open: boolean;
  onClose: () => void;
  initialUser?: StaffMember;
  /** Inviting: the parent shows the invitation link afterwards. */
  /** Saves, then closes the modal. Throw to keep it open: a 422 (or a refusal such as the trial limit) is shown inside it. */
  onSubmitCreate: (values: UserFormValues) => Promise<void>;
  onSubmitEdit: (values: UserFormValues) => Promise<void>;
}

export function UserFormModal({ open, onClose, initialUser, onSubmitCreate, onSubmitEdit }: UserFormModalProps) {
  const businessName = useTenantBrandingStore((s) => s.businessName) ?? APP_NAME;
  const isEditing = Boolean(initialUser);
  const pendingInvite = initialUser?.loginStatus === "invited";
  const form = useZodForm({ schema: userSchema, initial: toFormState(initialUser) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const roles = useRolesStore((s) => s.roles.staff);
  const fetchRoles = useRolesStore((s) => s.fetchRoles);
  const assignable = roles.filter((r) => !r.isSystem);

  useEffect(() => {
    void fetchRoles("staff").catch(() => undefined);
  }, [fetchRoles]);

  // Mounted once and kept alive while closed: every opening starts from the person as they are now.
  useEffect(() => {
    form.reset(toFormState(initialUser));
  }, [initialUser, open]);

  const submit = form.handleSubmit(
    async (parsed) => {
      if (isEditing) await onSubmitEdit(parsed);
      else await onSubmitCreate(parsed);
    },
    { root: () => formRef.current, fallback: isEditing ? "Perubahan belum tersimpan. Periksa koneksi lalu coba lagi." : "Undangan belum dibuat. Periksa koneksi lalu coba lagi." },
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEditing ? "Ubah Pengguna" : "Undang Pengguna"}
      description={
        isEditing
          ? `Data staf ${businessName}.`
          : `Orang ini menerima link undangan dan masuk ke ${businessName} dengan akun Google-nya.`
      }
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="user-form" disabled={form.submitting}>
            {isEditing ? (form.submitting ? "Menyimpan..." : "Simpan perubahan") : form.submitting ? "Membuat undangan..." : "Buat undangan"}
          </Button>
        </>
      }
    >
      <form id="user-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
      <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Lengkap" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Anisa Putri" />
        </Field>
        <Field label="Peran" required name="roleId" error={errors.roleId} hint="Izin tiap peran diatur di Pengaturan → Peran & Akses.">
          {initialUser?.role === OWNER_ROLE_NAME ? (
            <Input value={OWNER_ROLE_NAME} disabled />
          ) : (
            <Select value={values.roleId ? String(values.roleId) : ""} onChange={(e) => set("roleId", Number(e.target.value))}>
              <option value="" disabled>
                Pilih peran
              </option>
              {assignable.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name}
                </option>
              ))}
            </Select>
          )}
        </Field>
        <Field label="Jabatan" required name="title" error={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} onBlur={() => blur("title")} placeholder="cth. Lead Planner" />
        </Field>
        <Field label="Nomor WhatsApp" name="phone" error={errors.phone} hint="Untuk mengirim link undangan lewat WhatsApp">
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} onBlur={() => blur("phone")} placeholder="0812 3456 7890" inputMode="tel" />
        </Field>
        <div className="sm:col-span-2">
          <Field
            label="Email akun Google"
            required
            name="email"
            error={errors.email}
            hint={
              isEditing && !pendingInvite
                ? "Akun ini sudah terhubung ke Google. Mengubah email tidak mengganti akun Google yang dipakai masuk."
                : "Undangan hanya bisa diterima dengan akun Google yang memakai email ini."
            }
          >
            <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} placeholder="nama@gmail.com" autoComplete="off" />
          </Field>
        </div>
      </div>
      </form>
    </Modal>
  );
}
