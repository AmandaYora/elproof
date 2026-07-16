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
