export type ClientRole = "Bride" | "Groom" | "Family Representative";

export interface Client {
  id: string;
  projectId: string;
  role: ClientRole;
  relationNote: string;
  name: string;
  phone: string;
  email: string;
  isActive: boolean;
  lastCredentialResetAt: string | null;
}
