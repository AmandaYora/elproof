// Real backend types for the `staff` module — see docs/API_CONTRACT.md.

// Staff roles are per workspace (Pengaturan → Peran & Akses, ADR-0026):
// `role` is the role's current name, `roleId` its id. The Owner role is
// always named "Owner".
export const OWNER_ROLE_NAME = "Owner";

export interface StaffMember {
  id: string;
  name: string;
  title: string;
  initials: string;
  role: string;
  /** Same as `role`; the name the list's sort field `roleName` refers to. */
  roleName?: string;
  roleId: number;
  username: string;
  email: string;
  phone: string;
  isActive: boolean;
  /** "invited" until the person accepted their invitation with Google. */
  loginStatus?: "invited" | "active" | "inactive" | "none" | "unknown";
}

// Public-safe subset (any staff role, unlike StaffMember's Owner-only
// endpoint) — {id, name, title} only, powers PIC pickers/labels throughout
// the `projects` module.
export interface StaffSummary {
  id: string;
  name: string;
  title: string;
}
