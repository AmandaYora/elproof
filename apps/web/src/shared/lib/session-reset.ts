// Stores that hold one person's data register here so a logout (or ending an
// impersonation) clears them without `shared/` having to import any module.
const resets = new Set<() => void>();

/** Registers a function that clears a store when the session ends. Call once, at module load. */
export function onSessionReset(reset: () => void): void {
  resets.add(reset);
}

/** Clears every registered store. */
export function runSessionResets(): void {
  resets.forEach((reset) => reset());
}
