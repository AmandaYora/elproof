# Rule: Frontend (React)

Applies to: `apps/web/**`.

- React 19 + Tailwind 4, react-router-dom with lazy-loaded routes/pages.
- State: Zustand (global in `shared/stores`, module state in `modules/<m>/stores`).
- Validation: Zod schemas in `modules/<m>/schemas`. Prefer `z.infer` over manual form types.
- HTTP: all requests go through `shared/services/http-client.ts` (one Axios instance).
- Use the `@/*` import alias. Centralize theme color in `src/theme/`.
- Shared UI components must be domain-agnostic. Use the `frontend-design` skill for UI; if it isn't installed, install it first from https://github.com/anthropics/skills/tree/main/skills/frontend-design (do not skip it).
- Primitives come from `shared/components/kit` (shadcn on Base UI); screens use `shared/components/ui` (`Button`, `Modal`, `Drawer`, `Field`, …). No hand-made dialogs, no hex or palette colours in components: semantic tokens only (`knowledge/FRONTEND_GUIDE.md`, ADR-0037).
- Forms: `shared/forms` (`useZodForm`, `Field name=`). Lists: `shared/data-table` with server-side sort/filter/paging; never load every row to filter in the browser.
- Text a person reads is Indonesian and uses one glossary term per concept (`knowledge/DESIGN.md` §7); API enum values go through `shared/lib/labels.ts`. `npm run check:terms` must stay green.
- Saving something read earlier sends its `updatedAt` as `If-Match` (ADR-0036); state changes are explicit (`setXActive(id, bool)`), never a flip.
- Anything only needed after a click (sheet, dialog, tooltip, toaster) loads on demand; `npm run budget` guards the entry chunk. Before a commit that touches a screen: `npx tsc -b && npm run check:terms && npm run build && npm run budget`, and `npm run e2e` when the look or behaviour changed.
