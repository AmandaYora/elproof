// Real backend types for the `staff` module — see docs/API_CONTRACT.md.

export type StaffRole = "Owner" | "Admin" | "Staff";

// ROLE_LABELS is a display-only relabeling — "Staff" is shown to users as
// "Wedding Planner" (the role's real-world business name), but the value
// stored in the database, sent to the API, and carried in the JWT stays the
// literal string "Staff" everywhere. Never compare against these labels;
// always compare against the raw StaffRole value.
export const ROLE_LABELS: Record<StaffRole, string> = {
  Owner: "Owner",
  Admin: "Admin",
  Staff: "Wedding Planner",
};

export interface StaffMember {
  id: string;
  name: string;
  title: string;
  initials: string;
  role: StaffRole;
  username: string;
  email: string;
  phone: string;
  isActive: boolean;
}

// Public-safe subset (any staff role, unlike StaffMember's Owner-only
// endpoint) — {id, name, title} only, powers PIC pickers/labels throughout
// the `projects` module.
export interface StaffSummary {
  id: string;
  name: string;
  title: string;
}
