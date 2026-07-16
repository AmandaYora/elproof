import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import { userSchema, STAFF_ROLE_OPTIONS, type UserFormValues } from "@/modules/users/schemas/user.schema";
import type { StaffMember } from "@/modules/users/types";

interface UserFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: UserFormValues) => void;
  initialUser?: StaffMember;
}

function toFormValues(user?: StaffMember): UserFormValues {
  if (!user) {
    return { name: "", title: "", role: "Staff", email: "", phone: "" };
  }
  // Owner accounts are provisioned via Platform Console and never open this form (see UserListPage),
  // but the role field still needs a value assignable to the editable Admin/Staff set.
  const role = user.role === "Owner" ? "Staff" : user.role;
  return { name: user.name, title: user.title, role, email: user.email, phone: user.phone };
}

export function UserFormModal({ open, onClose, onSubmit, initialUser }: UserFormModalProps) {
  const [values, setValues] = useState<UserFormValues>(() => toFormValues(initialUser));
  const [errors, setErrors] = useState<Partial<Record<keyof UserFormValues, string>>>({});

  function set<K extends keyof UserFormValues>(key: K, value: UserFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = userSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof UserFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof UserFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setErrors({});
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initialUser ? "Ubah Pengguna" : "Tambah Pengguna Baru"}
      description="Kelola akun staff internal WO yang dapat mengakses WO Console."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>{initialUser ? "Simpan Perubahan" : "Simpan Pengguna"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Lengkap" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Anisa Putri" />
        </Field>
        <Field label="Role" required>
          <Select value={values.role} onChange={(e) => set("role", e.target.value as UserFormValues["role"])}>
            {STAFF_ROLE_OPTIONS.map((r) => (
              <option key={r} value={r}>{r}</option>
            ))}
          </Select>
        </Field>
        <Field label="Jabatan" required hint={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} placeholder="cth. Lead Planner" />
        </Field>
        <Field label="Nomor Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="0812-xxxx-xxxx" />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Email" required hint={errors.email}>
            <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} placeholder="nama@elproof.id" />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
