# ADR-0007: Server time as the single source of "now"

## Status
Accepted

## Context
The frontend mock hardcodes `TODAY = "2026-07-12"` (`mock/seed.ts`) and reuses it everywhere "now"
matters: D-day countdowns, dashboard month/year trend toggles
(`modules/platform-admin/lib/trend.ts`), overdue-milestone checks. This was a deliberate mock
shortcut, not a design to preserve.

## Decision
- The backend is the single source of truth for "now". All date-dependent computations (dashboard
  trends, D-day countdowns, expiry checks, `granted`/`paid` transaction timestamps) use the Go
  server's clock (`time.Now()`), explicitly in the `Asia/Jakarta` timezone, never a client-supplied
  "current date".
- Endpoints that return "days until X" or "this month"/"this year" bucketed data compute the bucket
  boundaries server-side and return already-labeled points (mirroring the shape
  `modules/platform-admin/lib/trend.ts`'s `TrendPoint[]` already expects), so the frontend keeps the
  same rendering components (`TrendBarChart`, `ProjectTrendChart`, `RevenueTrendChart`) and only
  swaps the data source.
- No client-supplied "as of" date parameter is accepted on these endpoints in this phase — if a
  future need arises (e.g. historical reporting), it will be an explicit, separately-reviewed
  addition.

## Consequences
- All the `TODAY`-based helper functions in `mock/selectors.ts` and
  `modules/platform-admin/lib/trend.ts` become dead code once their corresponding backend endpoint
  ships (tracked per-module in PLAN.md Fase 4/5), not something to keep "just in case".
- Server and any local dev environment must have correct system clock/timezone; no reliance on
  browser time.
