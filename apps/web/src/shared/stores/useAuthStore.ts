import { create } from "zustand";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export type PrincipalType = "staff" | "client" | "platform_admin";

/**
 * Set while a Platform Console admin looks at a workspace as its Owner
 * (read-only support session, FLOW-IDEAL 1.5). `returnTo` is the admin's own
 * session, restored when the session ends.
 */
export interface ImpersonationInfo {
  sessionId: number;
  tenantName: string;
  expiresAt: string;
  returnTo: AuthSession;
}

/**
 * The signed-in principal. The access token lives only in memory: after a
 * page load it is "" until the HTTP client gets a new one from
 * `POST /auth/refresh`, which reads the refresh token from an HttpOnly cookie
 * that no script can read (ADR-0044).
 */
export interface AuthSession {
  accessToken: string;
  principalType: PrincipalType;
  principalId: string;
  tenantId: number | null;
  role: string;
  displayName: string;
  /** Session established with a second factor (Platform Console admins). */
  mfa?: boolean;
  /** A Platform Console admin must set up two-factor before anything else. */
  mfaSetupRequired?: boolean;
  impersonation?: ImpersonationInfo;
}

/**
 * Who is signed in, without any token, so the first paint after a reload
 * knows which console to show. Shared by the browser's tabs.
 */
export const PROFILE_KEY = "elproof.profile";
/** Before ADR-0044 the whole session, tokens included, was stored here. */
const LEGACY_SESSION_KEY = "elproof.session";
/** A support session crossing the full page load that starts it (read once, then deleted). */
const SUPPORT_HANDOFF_KEY = "elproof.support-handoff";
/** Marks this tab as being in a support session, so a reload can end it properly. */
const SUPPORT_ACTIVE_KEY = "elproof.support-active";

const PRINCIPAL_TYPES: readonly PrincipalType[] = ["staff", "client", "platform_admin"];

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

/** A support session interrupted by a reload of its tab: the app ends it and returns the admin (impersonation.ts). */
export interface InterruptedSupportSession {
  sessionId: number;
  tenantId: number | null;
}

function storage(kind: "local" | "session"): Storage | null {
  try {
    return kind === "local" ? window.localStorage : window.sessionStorage;
  } catch {
    return null; // storage blocked (private mode, sandbox)
  }
}

function read(kind: "local" | "session", key: string): string | null {
  try {
    return storage(kind)?.getItem(key) ?? null;
  } catch {
    return null;
  }
}

function write(kind: "local" | "session", key: string, value: string | null) {
  try {
    if (value === null) storage(kind)?.removeItem(key);
    else storage(kind)?.setItem(key, value);
  } catch {
    // Storage unavailable: the session just won't survive a reload.
  }
}

/** The non-secret part of a session; the only part ever written to storage. */
function profileOf(session: AuthSession) {
  return {
    principalType: session.principalType,
    principalId: session.principalId,
    tenantId: session.tenantId,
    role: session.role,
    displayName: session.displayName,
    mfa: session.mfa,
    mfaSetupRequired: session.mfaSetupRequired,
  };
}

function parseProfile(raw: string | null): AuthSession | null {
  if (!raw) return null;
  try {
    const p = JSON.parse(raw) as Partial<AuthSession>;
    if (!p || !PRINCIPAL_TYPES.includes(p.principalType as PrincipalType) || typeof p.principalId !== "string") return null;
    return {
      accessToken: "",
      principalType: p.principalType as PrincipalType,
      principalId: p.principalId,
      tenantId: typeof p.tenantId === "number" ? p.tenantId : null,
      role: typeof p.role === "string" ? p.role : "",
      displayName: typeof p.displayName === "string" ? p.displayName : "",
      mfa: p.mfa === true,
      mfaSetupRequired: p.mfaSetupRequired === true,
    };
  } catch {
    return null;
  }
}

/** Identity of a stored profile, to tell a sign-in as someone else in another tab apart from a token refresh. */
export function profileIdentity(raw: string | null): string {
  const p = parseProfile(raw);
  return p ? `${p.principalType}:${p.principalId}` : "";
}

function persistProfile(session: AuthSession | null) {
  // A support session is never stored: other tabs keep the admin's own
  // profile, and a reload of this tab returns to it.
  if (session?.impersonation) return;
  write("local", PROFILE_KEY, session ? JSON.stringify(profileOf(session)) : null);
}

/** Starts a support session across the full page load that follows (impersonation.ts). */
export function handOffSupportSession(session: AuthSession) {
  write("session", SUPPORT_HANDOFF_KEY, JSON.stringify(session));
}

/** Forgets that this tab is in a support session (ended or abandoned). */
export function clearSupportMarker() {
  write("session", SUPPORT_ACTIVE_KEY, null);
}

let interrupted: InterruptedSupportSession | null = null;

/** Whether a reload interrupted a support session in this tab (not consumed). */
export function hasInterruptedSupportSession(): boolean {
  return interrupted !== null;
}

/** The support session a reload interrupted, once; null otherwise. */
export function takeInterruptedSupportSession(): InterruptedSupportSession | null {
  const value = interrupted;
  interrupted = null;
  return value;
}

function loadInitialSession(): AuthSession | null {
  // Sessions stored by earlier versions carried both tokens: drop them.
  write("local", LEGACY_SESSION_KEY, null);
  const profile = parseProfile(read("local", PROFILE_KEY));

  const handoff = read("session", SUPPORT_HANDOFF_KEY);
  write("session", SUPPORT_HANDOFF_KEY, null);
  if (handoff) {
    try {
      const support = JSON.parse(handoff) as AuthSession;
      if (support.impersonation && support.accessToken && new Date(support.impersonation.expiresAt).getTime() > Date.now()) {
        write("session", SUPPORT_ACTIVE_KEY, JSON.stringify({ sessionId: support.impersonation.sessionId, tenantId: support.tenantId }));
        return support;
      }
    } catch {
      // unreadable: fall back to the stored profile
    }
  }

  const marker = read("session", SUPPORT_ACTIVE_KEY);
  if (marker) {
    clearSupportMarker();
    try {
      interrupted = JSON.parse(marker) as InterruptedSupportSession;
    } catch {
      interrupted = null;
    }
    if (interrupted && profile?.principalType === "platform_admin") {
      // Back to the workspace's page in the Platform Console before the
      // router reads the address, so the WO Console never flashes up.
      const target = interrupted.tenantId ? ROUTE_PATHS.platformTenantDetail(String(interrupted.tenantId)) : ROUTE_PATHS.platformTenants;
      window.history.replaceState(null, "", target);
    }
  }
  return profile;
}

function deriveLegacyIds(session: AuthSession | null) {
  return {
    currentStaffId: session?.principalType === "staff" ? session.principalId : FALLBACK_STAFF_ID,
    currentClientId: session?.principalType === "client" ? session.principalId : FALLBACK_CLIENT_ID,
    currentPlatformAdminId:
      session?.principalType === "platform_admin" ? session.principalId : FALLBACK_PLATFORM_ADMIN_ID,
  };
}

const initialSession = loadInitialSession();

export const useAuthStore = create<AuthState>((set) => ({
  session: initialSession,
  isAuthenticated: initialSession !== null,
  ...deriveLegacyIds(initialSession),
  login: (session) => {
    persistProfile(session);
    if (!session.impersonation) clearSupportMarker();
    set({ session, isAuthenticated: true, ...deriveLegacyIds(session) });
  },
  logout: () => {
    persistProfile(null);
    clearSupportMarker();
    set({ session: null, isAuthenticated: false, ...deriveLegacyIds(null) });
  },
}));

// Another tab signed out or signed in as someone else: this tab's in-memory
// state belongs to a session that no longer exists, so start over.
if (typeof window !== "undefined") {
  window.addEventListener("storage", (event) => {
    if (event.key !== PROFILE_KEY && event.key !== null) return;
    const current = useAuthStore.getState().session;
    const own = current?.impersonation ? current.impersonation.returnTo : current;
    const ownIdentity = own ? `${own.principalType}:${own.principalId}` : "";
    if (profileIdentity(event.key === null ? null : event.newValue) !== ownIdentity) window.location.reload();
  });
}
