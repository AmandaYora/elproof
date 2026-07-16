// Real backend types for the `staff` module — see docs/API_CONTRACT.md.

export type StaffRole = "Owner" | "Admin" | "Staff";

export interface StaffMember {
  id: string;
  name: string;
  title: string;
  initials: string;
  role: StaffRole;
  email: string;
  phone: string;
  isActive: boolean;
}
