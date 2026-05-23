# Frontend

React 18 + TypeScript + Vite web UI for managing the registry.

Dev server port `5173`; production port `3000` (served by Docker). All `/api`, `/v1`, `/.well-known`, `/downloads`, `/shasums` requests are proxied to the backend in dev.

## Run locally

```bash
cd frontend
pnpm install
pnpm dev
```

Use **pnpm only** — not npm or yarn. A stale `package-lock.json` exists in the directory; ignore it.

`frontend/.env` is committed with `VITE_API_BASE_URL=` (empty) — intentional. In dev the Vite proxy handles all API calls.

## Build

```bash
pnpm build    # runs tsc then vite build — both must pass
pnpm lint     # ESLint with --max-warnings 0
```

There is no separate typecheck script — `pnpm build` (or `tsc` directly) is the typecheck step.

## Project layout

```
src/
├── api/index.ts          # All API calls (centralized, typed)
├── types/index.ts        # All TypeScript types
├── App.tsx               # Routes
├── components/
│   ├── Layout.tsx        # Navigation sidebar
│   └── ...
└── pages/
    ├── ApiKeysPage.tsx
    ├── BackendPage.tsx          # CLI setup guide for Terraform/OpenTofu
    ├── DeploymentDetailPage.tsx # Deployment detail + git browser
    ├── DeploymentTFStatePage.tsx
    ├── DeploymentsPage.tsx
    ├── ModuleDetailPage.tsx
    ├── ModulesPage.tsx
    ├── NamespaceDetailPage.tsx
    ├── NamespacesPage.tsx
    ├── ProviderDetailPage.tsx
    ├── ProvidersPage.tsx
    └── TFStateBrowserPage.tsx   # Browse state files as a JSON tree
```

## TypeScript notes

`strict: true`, `noUnusedLocals`, `noUnusedParameters`, `noFallthroughCasesInSwitch` are all enabled. Unused imports and variables are compile errors, not warnings. ESLint is zero-warnings. Both block `pnpm build`.

Path alias `@/` maps to `src/` in both Vite and TypeScript config.
