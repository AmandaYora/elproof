import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { compressFileToBlob } from "@/shared/lib/image-compression";

/** What POST /projects/{id}/uploads answers (PLAN.md E3, ADR-0033). */
interface UploadSlot {
  uploadToken: string;
  method: "PUT";
  url: string;
  headers: Record<string, string>;
  expiresAt: string;
  /** true: a presigned URL to object storage; false: an API path that takes the same PUT. */
  direct: boolean;
}

/**
 * Sends one file to storage and returns the token a save request attaches
 * (`evidence: [{ uploadToken, type }]`). The file only becomes a proof when that
 * request commits, so a failure here leaves nothing behind.
 */
export async function uploadFile(projectId: string, file: File): Promise<string> {
  const { blob, fileName, mimeType } = await compressFileToBlob(file);
  const slot = (
    await httpClient.post(API.projects.uploads(projectId), { fileName, mimeType, size: blob.size })
  ).data.data as UploadSlot;

  if (slot.direct) {
    // A bare fetch: the presigned URL is signed for storage and must not carry our bearer token.
    const res = await fetch(slot.url, { method: slot.method, headers: slot.headers, body: blob });
    if (!res.ok) throw new Error(`Unggahan ditolak penyimpanan (${res.status})`);
  } else {
    // Through the API: same PUT, with the app's auth and base URL.
    await httpClient.put(slot.url, blob, { headers: slot.headers });
  }
  return slot.uploadToken;
}

/** One Idempotency-Key per save attempt: reused when the same request is retried, new once the form changed or succeeded. */
export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}
