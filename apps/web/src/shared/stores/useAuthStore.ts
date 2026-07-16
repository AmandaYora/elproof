import { create } from "zustand";

export type PrincipalType = "staff" | "client" | "platform_admin";

export interface AuthSession {
  accessToken: string;
  refreshToken: string;
  principalType: PrincipalType;
  principalId: string;
  tenantId: number | null;
  role: string;
  displayName: string;
}

const STORAGE_KEY = "elproof.session";

// Pre-integration default identities — kept as the fallback when no real
// session exists yet, so the many pages built before real auth existed
// (Sidebar, ClientPortalLayout, PlatformLayout, project detail sections, etc.)
// keep rendering unchanged until Fase 7 adds route guards and a true
// logged-out state. Once a real login happens, the real principal takes over.
const FALLBACK_STAFF_ID = "1";
const FALLBACK_CLIENT_ID = "cl-p1-bride";
const FALLBACK_PLATFORM_ADMIN_ID = "pa1";

interface AuthState {
  session: AuthSession | null;
  isAuthenticated: boolean;
  currentStaffId: string;
  currentClientId: string;
  currentPlatformAdminId: string;
  login: (session: AuthSession) => void;
  logout: () => void;
}

function loadPersistedSession(): AuthSession | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as AuthSession) : null;
  } catch {
    return null;
  }
}

function persistSession(session: AuthSession | null) {
  try {
    if (session) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // localStorage unavailable (e.g. private mode) — session just won't survive reload.
  }
}

function deriveLegacyIds(session: AuthSession | null) {
  return {
    currentStaffId: session?.principalType === "staff" ? session.principalId : FALLBACK_STAFF_ID,
    currentClientId: session?.principalType === "client" ? session.principalId : FALLBACK_CLIENT_ID,
    currentPlatformAdminId:
      session?.principalType === "platform_admin" ? session.principalId : FALLBACK_PLATFORM_ADMIN_ID,
  };
}

const initialSession = loadPersistedSession();

export const useAuthStore = create<AuthState>((set) => ({
  session: initialSession,
  isAuthenticated: initialSession !== null,
  ...deriveLegacyIds(initialSession),
  login: (session) => {
    persistSession(session);
    set({ session, isAuthenticated: true, ...deriveLegacyIds(session) });
  },
  logout: () => {
    persistSession(null);
    set({ session: null, isAuthenticated: false, ...deriveLegacyIds(null) });
  },
}));
