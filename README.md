# ElProof

Monorepo generated to Dimas' standard (modular-monolith backend + React 19 frontend).

## Development

Run frontend and backend separately from the project root:

```bash
npm run dev:web
npm run dev:api
```

## Structure

- `apps/web` — React 19 + Tailwind 4 frontend
- `apps/api` — go modular-monolith backend
- `packages/` — shared code and API contract
- `knowledge/` — project knowledge base for Claude Code (read before editing)
- `.claude/rules/` — path-scoped technical rules
- `docs/` — PRD, system design, API contract, DB schema, deployment
- `infra/` — Docker and nginx

## Deployment

One Docker app container serving both the static frontend and the API on port 8080.
The database runs on the host (see `docs/DEPLOYMENT.md`).
