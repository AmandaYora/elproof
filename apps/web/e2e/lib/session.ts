import type { Page } from "@playwright/test";
import { apiURL, type Session } from "./config";
import type { Reply } from "./api";

/**
 * The browser session lives in two places since ADR-0042: the refresh token in an HttpOnly cookie scoped to
 * /api/v1/auth (APP_ENV=development: no `__Secure-` prefix, not Secure), and a token-free profile in localStorage
 * that tells the app who is signed in before its first refresh. The access token is never stored.
 */
export const REFRESH_COOKIE = "elproof_refresh";
export const PROFILE_KEY = "elproof.profile";

/** The refresh token the API set on a sign-in reply (its Set-Cookie header). */
export function refreshCookieOf(reply: Reply): string {
  for (const line of reply.cookies) {
    const [pair] = line.split(";");
    const eq = pair.indexOf("=");
    if (pair.slice(0, eq).trim() === REFRESH_COOKIE) return pair.slice(eq + 1).trim();
  }
  throw new Error(`no ${REFRESH_COOKIE} cookie in the sign-in reply: ${reply.cookies.join(" | ") || "(none)"}`);
}

/** A sign-in reply as a Session: the JSON body plus the refresh token from the cookie. */
export function sessionOf(reply: Reply): Session {
  return { ...(reply.json.data as Omit<Session, "refreshCookie">), refreshCookie: refreshCookieOf(reply) };
}

/**
 * A Playwright storage state that is signed in as `session`. Several browser contexts may start from one state: the
 * e2e API accepts a just-rotated refresh token for REFRESH_REUSE_GRACE (set long in stack.ts), so they do not sign each
 * other out.
 */
export function storageStateFor(webURL: string, session: Session | null) {
  if (!session) return { cookies: [], origins: [] };
  const profile = {
    principalType: session.principalType,
    principalId: session.principalId,
    tenantId: session.tenantId,
    role: session.role,
    displayName: session.displayName,
    mfa: session.mfa,
    mfaSetupRequired: session.mfaSetupRequired,
  };
  return {
    cookies: [
      {
        name: REFRESH_COOKIE,
        value: session.refreshCookie,
        domain: new URL(apiURL()).hostname,
        path: "/api/v1/auth",
        expires: -1,
        httpOnly: true,
        secure: false,
        sameSite: "Strict" as const,
      },
    ],
    origins: [{ origin: webURL, localStorage: [{ name: PROFILE_KEY, value: JSON.stringify(profile) }] }],
  };
}

/**
 * The signed-in session of the page's browser context, with a fresh access token: what the app itself does after a
 * page load (POST /auth/refresh with the cookie). The page must be on the web origin.
 */
export async function browserSession(page: Page): Promise<{ accessToken: string; principalId: string; principalType: string; tenantId: number | null }> {
  return page.evaluate(async (api) => {
    const res = await fetch(`${api}/api/v1/auth/refresh`, { method: "POST", credentials: "include", headers: { "X-Requested-With": "XMLHttpRequest" } });
    const body = (await res.json()) as { data?: { accessToken: string; principalId: string; principalType: string; tenantId: number | null } };
    if (!res.ok || !body.data) throw new Error(`refresh failed: ${res.status}`);
    return body.data;
  }, apiURL());
}
