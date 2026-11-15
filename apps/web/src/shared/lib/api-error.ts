import axios from "axios";

// Every backend error response body is { success: false, message, errors? }
// (.claude/rules/api-standard.md) — this extracts the human-readable message
// consistently wherever a page/store action needs to surface it.
export function getApiErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err)) {
    const message = (err.response?.data as { message?: string } | undefined)?.message;
    if (message) return message;
  }
  return fallback;
}

/**
 * Renames the field keys of a 422 in place, so a form can show a server
 * message under its own input (e.g. `evidence[0].uploadToken` → `proofFile`).
 * `rename` returns the new key, or undefined to keep the key as it is.
 * Anything that is not an API error passes through untouched.
 */
export function renameApiFieldErrors(err: unknown, rename: (key: string) => string | undefined): unknown {
  if (!axios.isAxiosError(err)) return err;
  const data = err.response?.data as { errors?: Record<string, unknown> } | undefined;
  if (!data?.errors || typeof data.errors !== "object") return err;
  const next: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(data.errors)) {
    const to = rename(key) ?? key;
    // Two server keys can land on one field (`evidence[0]` and `evidence[0].uploadToken` say the same): keep the first.
    if (!(to in next)) next[to] = value;
  }
  data.errors = next;
  return err;
}

/**
 * The per-field messages of a 422 (`errors: { field: [message] }`), first
 * message of each, so a form can show each one under its own input.
 */
export function getApiFieldErrors(err: unknown): Record<string, string> {
  const out: Record<string, string> = {};
  if (!axios.isAxiosError(err)) return out;
  const errors = (err.response?.data as { errors?: unknown } | undefined)?.errors;
  if (errors && typeof errors === "object") {
    for (const [field, value] of Object.entries(errors as Record<string, unknown>)) {
      if (Array.isArray(value) && typeof value[0] === "string") out[field] = value[0];
    }
  }
  return out;
}
