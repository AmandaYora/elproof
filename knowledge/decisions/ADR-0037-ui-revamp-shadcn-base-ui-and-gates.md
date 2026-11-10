# ADR-0037: UI revamp — shadcn/ui on Base UI, neutral tokens, one language, and the gates that keep it that way

## Status
Accepted — implemented 2026-11-08 … 2026-11-10 (docs/plan/revamp-ui-shadcn/PLAN.md, Tahap 0–9). The API half is in ADR-0032 … ADR-0036; this ADR is the
front end and the process around it. The visual contract is `knowledge/DESIGN.md`.

## Context
The web app had grown to ~190 `.tsx` files with its own hand-written components. Four things were wrong, and measured in Tahap 0:

1. **Look and accessibility.** Status colours failed 4.5:1 on their badge backgrounds (~173 uses), the sidebar was a heavy brand-coloured block, control borders were
   below 3:1, focus and overlays behaved differently per screen, and there was no automated accessibility check.
2. **Words.** "Project"/"Proyek", "Client"/"Klien", "Timeline"/"Jadwal", "Evidence"/"Dokumen" lived side by side.
3. **Cost of a screen.** The project page made ~9 calls, a portal home ~9, and a payment with its proof was 2–3 non-atomic calls; margins and remaining amounts were
   recomputed in the browser. Several lists loaded every row and filtered in the browser.
4. **Safety of edits.** Two people could silently overwrite each other; toggles flipped whatever state was current.

## Decision

### Components
* **shadcn/ui copies on Base UI** (not Radix; D1) live in `src/shared/components/kit/` and are the only place primitives come from. Only components in use are kept
  (nothing is installed "just in case"; unused ones were removed in Tahap 9).
* **The old PascalCase components in `src/shared/components/ui/` stay as the app-facing API** (D2): `Button`, `Input`, `Modal`, `Drawer`, `Badge`, `Card`, `Table`, … keep
  their props and wrap the kit, so a screen is migrated by the kit changing underneath it, not by editing hundreds of call sites. New code uses them too. A component
  there is domain-agnostic; domain pieces live in their module.
* **Forms:** `shared/forms` (`useZodForm`, `Field name=`, `FormErrorSummary`): Zod schema per module, `z.infer` types, errors per field from the API's 422 keys, focus on the first error.
  No react-hook-form (D3). Dates use the native picker (D7).
* **Lists:** `shared/data-table` (`DataTable`, `useListState`, saved views): state in the URL, server-side sort/filter/paging (ADR-0035), two densities, cards on a phone, bulk bar.
* **Layout and navigation:** one `AppSidebar` for the WO and Platform consoles (icon rail, sheet on a phone, `Ctrl/Cmd+B`), `Breadcrumbs`, ⌘K palette as a shortcut
  over the server search, never the main navigation (D9). Anything only needed after a user action (sheet, palette dialog, rail tooltip, conflict dialog, toaster,
  evidence viewer) is loaded on demand so the entry chunk stays ≤ 165 KB gzip.
* **Feedback:** Sonner behind `toast.*` with "Urungkan" for soft deletes (D4, D18); one global "Data diubah orang lain" dialog for 409 (ADR-0036).
* **Charts:** one hand-made `TrendBarChart` with a data table for screen readers (D11); the Gantt library only for the project schedule, themed from tokens.
* **No** dark mode (D6), Storybook (D10), Recharts, glass effects or tours.

### Tokens and look
Semantic tokens in `src/theme/theme.css` (`--background`, `--primary`, `--muted-foreground`, `--color-*-text` for text in a status colour, `--input` at 3:1, one z-index scale,
two densities). Tenant branding overrides only `--brand-navy-*` at runtime. Warm neutral surface, neutral sidebar, the brand only on the logo, the primary button, focus and
the active item (D13). Plus Jakarta Sans for UI, Fraunces only for marketing and the portal home (the couple's names, the countdown figure and the next amount due) (D8, D14). Money is "Rp1.500.000" in tabular figures (D5). The single
expressive element is `ProofStamp`, which respects `prefers-reduced-motion`.

### Language
One term per concept, Bahasa Indonesia, per the glossary in `DESIGN.md` §7 (Proyek, Klien, Tagihan, Dokumen/bukti, Kendala, Jadwal, Tahapan, Tim). Enforced, not hoped for:
`apps/web/scripts/check-ui-terms.mjs` parses the TSX and fails on a disallowed term in anything a person reads (JSX text, labels, placeholders, `toast.*`, `set…Error(...)`).
It runs in `npm run check:terms`, as an e2e test and in CI. Enum values sent to the API stay English; `shared/lib/labels.ts` is the only place they get a Indonesian label.

### API compatibility policy (D16)
The API may change when a screen's flow would otherwise be worse, but **additive first**: the old shape lives on until the web no longer calls it, then it is removed **in the same Tahap**
with the replacement named in the ADR (the toggle routes: added in ADR-0036's first commit, removed in its second). `If-Match` was optional for one Tahap and is required for
field edits from Tahap 9 (ADR-0036). External integrators are not affected: the only external surface is the payment gateway (`/auth/app/token`, `/external/*`), which was not changed.

### Gates (what "done" means, enforced mechanically)
| Gate | Where |
|---|---|
| Go build, vet, unit tests, tenant-isolation scenarios against real MySQL and object storage | `.github/workflows/ci.yml` job `api` |
| TypeScript, wording checker, bundle budget (`perf-budget.json`) | job `web` |
| axe on 68 pages × 2 viewports, **strict** (any serious/critical fails) | job `e2e` |
| Behaviour tests (menu, overlays, forms, lists, conflicts) and write tests | job `e2e` |
| Per-screen API-call ceiling and first-load JS | job `e2e`, `npm run budget:routes` |
| Pixel snapshots (macOS renders) | local only: `npm run e2e` before a visual change is committed; not in CI because Linux fonts differ |

LCP and INP are measured (`e2e/reports/BASELINE.md`, `npm run e2e:perf`) but not gated: they move with the machine. What is not and cannot be automated is listed in
`docs/plan/revamp-ui-shadcn/UJI-MANUAL.md` (screen reader, real iOS and Android devices, usability rounds).

## Decisions of PLAN §9, as implemented
| # | Decision | State |
|---|---|---|
| D1, D4, D6, D7, D9, D10, D11, D16 | Base UI; Sonner; no dark mode; native date; ⌘K as shortcut; no Storybook; own charts; API policy above | Done |
| D2, D3, D17, D18 | Old wrappers kept over the kit; `Field` + Zod; signed uploads; soft delete + restore | Done (ADR-0033, ADR-0034) |
| D5 | "Rp1.500.000" | Done; owner confirmation of the visible format still pending |
| D8, D13, D14, D15 | Fraunces on marketing and the portal home only; warm neutral + neutral sidebar; Plus Jakarta Sans; "Jadwal"/"tahapan" | **Implemented; owner sign-off pending** (they were open decisions in PLAN §9: change them in `theme.css` / `labels.ts`, nothing else depends on them) |
| D12 | Neutral tone for overdue amounts on the portal | **A hypothesis**: judged only by the usability round with real clients |

## Consequences
* A screen change is a change to a small set of shared pieces, and a regression in contrast, wording, call count or bundle size fails a build instead of being found by a user.
* The `ui/` wrappers add one layer between a screen and the primitive. That was the price of migrating without a big bang; collapsing them into the kit is possible later and mechanical.
* Pixel snapshots are macOS-only, so visual regressions are caught on a developer's machine, not in CI.
* Human verification remains: a person with a screen reader, real phones, and the usability rounds. Passing every gate here means "no known mechanical regression", not "usable".
