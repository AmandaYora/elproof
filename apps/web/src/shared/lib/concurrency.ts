import type { AxiosError } from "axios";

/**
 * `If-Match` header for a save of something read earlier (ADR-0036). `updatedAt` is the version the API gave with the item;
 * without one (an item from an older response) the save goes without the check, as it did before.
 */
export function ifMatch(updatedAt: string | undefined | null): Record<string, string> | undefined {
  return updatedAt ? { "If-Match": `"${updatedAt}"` } : undefined;
}

/** True when the API refused a save because the item changed after it was read. */
export function isStaleConflict(error: unknown): boolean {
  const e = error as AxiosError<{ errors?: { _conflict?: unknown } }> | undefined;
  return e?.response?.status === 409 && Boolean(e.response.data?.errors && "_conflict" in e.response.data.errors);
}
