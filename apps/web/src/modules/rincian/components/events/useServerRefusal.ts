import { useCallback, useState } from "react";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";
import { bracketKey } from "@/modules/rincian/components/events/event-display";

interface Refusal {
  /** Messages by form key ("sessions[1].guests"). */
  fields: Record<string, string>;
  message: string;
  /** The values the refused request was built from. */
  at: string;
}

/**
 * What the server refused about the values as they were sent (a 422 with field keys). The messages describe that
 * exact submission, so they are shown only while the form still holds it: the first change hides them, and the
 * capacity left per session (worked out on screen) takes over until the next save.
 */
export function useServerRefusal(values: unknown) {
  const [refusal, setRefusal] = useState<Refusal | null>(null);
  const now = JSON.stringify(values);
  const active = refusal && refusal.at === now ? refusal : null;

  /** Records a refusal that names fields and answers true; anything else is left to the caller (false). */
  const record = useCallback(
    (err: unknown, sent: unknown, fallback: string): boolean => {
      const raw = getApiFieldErrors(err);
      const keys = Object.keys(raw);
      if (keys.length === 0) return false;
      const fields: Record<string, string> = {};
      for (const k of keys) fields[bracketKey(k)] ??= raw[k];
      setRefusal({ fields, message: getApiErrorMessage(err, fallback), at: JSON.stringify(sent) });
      return true;
    },
    [],
  );

  const clear = useCallback(() => setRefusal(null), []);

  return { active, record, clear };
}
