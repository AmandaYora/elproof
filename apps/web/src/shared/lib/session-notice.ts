// One-shot message shown on the login page after the app ends a session on
// its own (e.g. the tenant was frozen), so the person knows why. Read with
// peekLoginNotice during render and clear it in an effect: reading is
// side-effect free, so React StrictMode's double render can't lose it.
const KEY = "elproof.loginNotice";

export function setLoginNotice(message: string) {
  try {
    sessionStorage.setItem(KEY, message);
  } catch {
    // storage unavailable: the login page simply shows no notice
  }
}

export function peekLoginNotice(): string | null {
  try {
    return sessionStorage.getItem(KEY);
  } catch {
    return null;
  }
}

export function clearLoginNotice() {
  try {
    sessionStorage.removeItem(KEY);
  } catch {
    // ignore
  }
}
