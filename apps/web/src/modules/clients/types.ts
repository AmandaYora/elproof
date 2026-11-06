export type ClientRole = "Bride" | "Groom" | "Family Representative";

export interface Client {
  id: string;
  projectId: string;
  role: ClientRole;
  username: string;
  relationNote: string;
  name: string;
  phone: string;
  email: string;
  isActive: boolean;
  lastCredentialResetAt: string | null;
  /** "invited" until the client set their own password. */
  loginStatus?: "invited" | "active" | "inactive" | "none" | "unknown";
  /** Portal role deciding what this client sees (ADR-0026). */
  accessRoleId: number;
  accessRoleName?: string;
}
