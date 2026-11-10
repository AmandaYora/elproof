# ElProof — Claude Code Gateway

Project: ElProof. Backend: go (modular monolith). Frontend: React 19 + Tailwind 4.

## Commands (run from root)

```bash
npm run dev:web   # start frontend
npm run dev:api   # start backend
```

Checks (from `apps/web`): `npx tsc -b && npm run check:terms && npm run build && npm run budget`; `npm run e2e` for the browser tests. CI (`.github/workflows/ci.yml`) runs the same gates.

## Critical architecture rules

- Backend is a **modular monolith**. A module exposes only `contracts/` to other modules.
  No cross-module service/repository/domain imports, no cross-module DB joins or foreign keys.
- Cross-module relations are stored as primitive IDs and resolved via module clients.
- Frontend uses the `@/*` alias, lazy routes, Zustand, Zod, Axios. Theme color is centralized.
- Docker = one app container; the database runs on the host.

## Before changing code

Read the relevant file in `knowledge/` first (start at `knowledge/INDEX.md`), and follow
the path-scoped rules in `.claude/rules/`. Do not duplicate that knowledge here — this file
is a gateway, not a documentation dump.
