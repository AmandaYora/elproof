import { useEffect, useState } from "react";
import { httpClient } from "@/shared/services/http-client";

/**
 * The hero's cover photo as something an <img> can show. An API path ("/api/v1/…") needs the bearer token, which a
 * plain <img> cannot send, so it is fetched as a blob through the HTTP client (the same way the venue photo is); a full
 * URL (a signed storage link) is used as is. A photo that fails to load simply leaves the plain hero.
 */
export function useCoverImage(url: string | null | undefined): string | null {
  const needsAuth = Boolean(url && url.startsWith("/"));
  const [blobUrl, setBlobUrl] = useState<string | null>(null);

  useEffect(() => {
    setBlobUrl(null);
    if (!url || !needsAuth) return;
    let cancelled = false;
    let created: string | null = null;
    httpClient
      .get(url, { responseType: "blob" })
      .then((res) => {
        if (cancelled) return;
        created = URL.createObjectURL(res.data as Blob);
        setBlobUrl(created);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
      if (created) URL.revokeObjectURL(created);
    };
  }, [url, needsAuth]);

  if (!url) return null;
  return needsAuth ? blobUrl : url;
}
