import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

// Google Identity Services (GIS) — "Sign in with Google" for WO Owners and
// staff (PLAN.md Fase 2). The browser only ever receives an ID token
// ("credential"); the backend verifies it (signature, audience, expiry,
// verified email). No client secret exists in this flow.

interface CredentialResponse {
  credential: string;
}

interface GsiButtonOptions {
  type?: "standard" | "icon";
  theme?: "outline" | "filled_blue" | "filled_black";
  size?: "large" | "medium" | "small";
  text?: "signin_with" | "signup_with" | "continue_with" | "signin";
  shape?: "rectangular" | "pill" | "circle" | "square";
  logo_alignment?: "left" | "center";
  width?: number;
  locale?: string;
}

interface GoogleAccountsId {
  initialize(config: {
    client_id: string;
    callback: (response: CredentialResponse) => void;
    ux_mode?: "popup" | "redirect";
    auto_select?: boolean;
    cancel_on_tap_outside?: boolean;
    context?: "signin" | "signup" | "use";
    itp_support?: boolean;
  }): void;
  renderButton(parent: HTMLElement, options: GsiButtonOptions): void;
  disableAutoSelect(): void;
}

declare global {
  interface Window {
    google?: { accounts: { id: GoogleAccountsId } };
  }
}

const GSI_SRC = "https://accounts.google.com/gsi/client";

let scriptPromise: Promise<void> | null = null;
let clientIdPromise: Promise<string> | null = null;

function loadScript(): Promise<void> {
  if (window.google?.accounts?.id) return Promise.resolve();
  if (!scriptPromise) {
    scriptPromise = new Promise<void>((resolve, reject) => {
      const script = document.createElement("script");
      script.src = GSI_SRC;
      script.async = true;
      script.defer = true;
      script.onload = () => resolve();
      script.onerror = () => {
        scriptPromise = null;
        script.remove();
        reject(new Error("Google Identity Services gagal dimuat"));
      };
      document.head.appendChild(script);
    });
  }
  return scriptPromise;
}

function loadClientId(): Promise<string> {
  if (!clientIdPromise) {
    clientIdPromise = httpClient
      .get(API.public.config)
      .then((res) => {
        const id = (res.data.data as { googleClientId?: string }).googleClientId;
        if (!id) throw new Error("Login Google belum dikonfigurasi");
        return id;
      })
      .catch((err) => {
        clientIdPromise = null;
        throw err;
      });
  }
  return clientIdPromise;
}

export type GoogleButtonText = NonNullable<GsiButtonOptions["text"]>;

/**
 * Renders Google's own button into `parent` (Google requires its branded
 * button; it cannot be restyled). `onCredential` receives the ID token.
 */
export async function renderGoogleButton(
  parent: HTMLElement,
  opts: {
    text: GoogleButtonText;
    context: "signin" | "signup";
    onCredential: (credential: string) => void;
    /** Checked after loading: a superseded render must not draw a second button. */
    isCancelled?: () => boolean;
  },
) {
  const [clientId] = await Promise.all([loadClientId(), loadScript()]);
  if (opts.isCancelled?.()) return;
  const gis = window.google?.accounts?.id;
  if (!gis) throw new Error("Google Identity Services tidak tersedia");
  gis.initialize({
    client_id: clientId,
    callback: (response) => opts.onCredential(response.credential),
    ux_mode: "popup",
    auto_select: false,
    cancel_on_tap_outside: true,
    context: opts.context,
    itp_support: true,
  });
  parent.innerHTML = "";
  // GIS accepts 200–400px.
  const width = Math.max(200, Math.min(400, Math.floor(parent.getBoundingClientRect().width) || 320));
  gis.renderButton(parent, {
    type: "standard",
    theme: "outline",
    size: "large",
    text: opts.text,
    shape: "rectangular",
    logo_alignment: "center",
    width,
    locale: "id",
  });
}

/** Stops Google from silently re-selecting the last account after logout. */
export function disableGoogleAutoSelect() {
  window.google?.accounts?.id?.disableAutoSelect();
}
