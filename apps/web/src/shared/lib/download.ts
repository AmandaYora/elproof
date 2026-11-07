import { httpClient } from "@/shared/services/http-client";

/**
 * Downloads a file the API builds on request (CSV exports). It goes through
 * the shared client so the bearer token is attached; a plain link could not.
 */
export async function downloadFile(url: string, params: Record<string, string | undefined>, fallbackName: string): Promise<void> {
  const res = await httpClient.get<Blob>(url, { params, responseType: "blob" });
  const disposition = String(res.headers["content-disposition"] ?? "");
  const match = /filename="?([^";]+)"?/i.exec(disposition);
  const objectUrl = URL.createObjectURL(res.data);
  const link = document.createElement("a");
  link.href = objectUrl;
  link.download = match?.[1] ?? fallbackName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(objectUrl);
}
