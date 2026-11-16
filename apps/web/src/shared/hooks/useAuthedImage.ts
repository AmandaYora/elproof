import { useEffect, useState } from "react";
import { httpClient } from "@/shared/services/http-client";

export type AuthedImageState = { status: "idle" | "loading" | "error"; src: null } | { status: "ready"; src: string };

/**
 * An image the API only gives to a signed-in session (an `<img src>` sends no Authorization header): fetched through
 * the one Axios instance as a blob and shown from an object URL, revoked when the URL changes or the component goes.
 * `url` is an API path such as a project's `coverUrl`; null shows nothing.
 */
export function useAuthedImage(url: string | null): AuthedImageState {
  const [state, setState] = useState<AuthedImageState>({ status: "idle", src: null });

  useEffect(() => {
    if (!url) {
      setState({ status: "idle", src: null });
      return;
    }
    let objectUrl: string | null = null;
    let cancelled = false;
    setState({ status: "loading", src: null });
    httpClient
      .get(url, { responseType: "blob" })
      .then((res) => {
        if (cancelled) return;
        objectUrl = URL.createObjectURL(res.data as Blob);
        setState({ status: "ready", src: objectUrl });
      })
      .catch(() => {
        if (!cancelled) setState({ status: "error", src: null });
      });
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [url]);

  return state;
}
