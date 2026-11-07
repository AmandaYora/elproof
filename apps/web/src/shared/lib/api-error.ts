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
