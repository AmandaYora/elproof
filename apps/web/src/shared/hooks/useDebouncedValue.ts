import { useEffect, useState } from "react";

// Delays reflecting `value` until it stops changing for `delayMs` — use to
// avoid firing a fetch on every keystroke in a search input.
export function useDebouncedValue<T>(value: T, delayMs = 150): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);

  return debounced;
}
