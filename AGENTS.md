# Agent Guidelines for IAC Platform

Self-hosted private Terraform/OpenTofu registry platform with Terraform Registry Protocol v1 and an HTTP state backend at `/api/tfstate/:deploymentId`.

## Services

| Dir | Lang | Port | Purpose |
|---|---|---|---|
| `backend/` | Go 1.24, Gin | 9080 | REST API + registry protocol + TF state backend + PostgreSQL |
| `frontend/` | React 18 + TS 5 + Vite 5 | 5173 (dev) / 3000 (prod) | Web UI |

## Commands

### Backend (`backend/`)
```bash
go build -o iac-tool main.go
go run main.go
go test ./...       # passes vacuously — no tests exist yet
go fmt ./...
go vet ./...
```
`go` is not on the host PATH; use Docker to verify compilation:
```bash
docker compose build backend
```

### Frontend (`frontend/`) — use pnpm, NOT npm or yarn
```bash
pnpm install
pnpm dev            # Vite dev server :5173, proxies /api /v1 /.well-known /downloads /shasums to :9080
pnpm build          # tsc + vite build; this is the ONLY typecheck step
```
- `pnpm lint` is **broken** (eslint not installed) — rely on `pnpm build` for correctness
- `package-lock.json` is a stale artifact; ignore it
- No separate typecheck script — `pnpm build` runs `tsc` then `vite build`

### Docker Compose
```bash
docker compose up -d             # not 'docker-compose'
docker compose up -d --build
docker compose logs -f backend
docker compose down -v           # WARNING: destroys all data volumes
```
`scripts/gen-certs.sh` generates self-signed TLS certs. `docker-compose.prod.yml` does not exist.

## TLS / HTTPS

Three nginx vhosts serve HTTPS on port 443: `registry.lan` (UI), `api.registry.lan` (API), `tf.registry.lan` (state backend). HTTP on port 80 redirects.

`nginx/certs/` is gitignored. Generate with `bash scripts/gen-certs.sh [HOST]` (default `registry.lan`). Nginx templates `${REGISTRY_HOST}` via `envsubst` at startup — the config file is never pre-processed.

**`/shasums` and `/downloads` have no auth**: Terraform/OpenTofu only send `Authorization` to `/v1/*` endpoints. Download URLs in responses are fetched without credentials. Security relies on these URLs being unguessable without first authenticating to `/v1`.

## Terraform/OpenTofu Client Config

`.terraformrc`/`.tofurc` must use `https://` — Terraform refuses to send tokens over plain HTTP:
```hcl
host "api.registry.lan" {
  services = {
    "providers.v1" = "https://api.registry.lan/v1/providers/"
  }
}
credentials "api.registry.lan" {
  token = "<api-key>"
}
```

- **`providers.v1` must point to `api.registry.lan`**, not `tf.registry.lan` — download URLs in responses use the request `Host` header via `getBaseURL()`, causing hostname mismatch for credentials
- State backend uses `https://tf.registry.lan/api/tfstate/<id>` for all of `address`, `lock_address`, `unlock_address`
- `tofu state replace-provider` is needed when existing state references a provider from a different source address — OpenTofu reads all provider addresses from remote state on `init`

## Critical Quirks

**Frontend TypeScript**: `strict: true`, `noUnusedLocals`, `noUnusedParameters`, `noFallthroughCasesInSwitch` — unused imports/vars are compile errors. `@/` maps to `src/` in both Vite and TypeScript.

**Database**: no migration tool. Schema applied via `CREATE TABLE IF NOT EXISTS` in `database.go:createTables()` on every startup. Column additions need manual `ALTER TABLE`. A `default` namespace is auto-seeded (`is_public=true`) on first run.

**Management API has no auth**: `/api/*` is open. Only `/v1/*` (Terraform registry protocol) enforces API key auth via `TerraformAuthMiddleware`.

**Namespace visibility**: `is_public=true` allows unauthenticated `/v1/providers/` and `/v1/modules/` access. Private namespaces require a valid API key on every `/v1/*` request.

**TF state workspace encoding**: OpenTofu/Terraform append `:<workspace>` to the backend address URL, which ends up inside `c.Param("deploymentId")`. Always use `parseDeploymentAndWorkspace()` from `tfstate.go` to split it. **Never** read `c.Param("deploymentId")` directly in tfstate handlers.

**TF state serial conflicts**: The backend must **not** reject POSTs based on serial — the client handles conflict detection. Accept all state writes while the caller holds the lock.

**TF state files are on disk**: state at `/app/data/tfstates/<deploymentId>/<workspace>.tfstate`, locks at `.tfstate.lock`. The UI lists via `GET /api/tfstates` which scans the directory, not the DB.

**TF state UI management endpoints** live under `/api/deployments/:id/tfstates*`, NOT under `/api/tfstate/`. The HTTP protocol endpoints (for `terraform init`) are at `/api/tfstate/:deploymentId`.

**`REGISTRY_HOST` varies by service**:
- Backend env: hostname only (`api.registry.lan`) — for CORS and `getBaseURL()`
- Nginx env: bare hostname without subdomain (`registry.lan`) — for `envsubst` in vhost templates

**`getBaseURL()` in `discovery.go`** builds URLs from the request's `Host` header and `X-Forwarded-Proto`. Nginx sets both when proxying. Never hardcode a base URL.

**Frontend API base URL**: `VITE_API_BASE_URL` is a Docker **build-time arg** (not runtime env). Default in Docker: `https://api.${REGISTRY_HOST}/api`. Falls back to relative `/api` in dev (via Vite proxy). Other build-time args: `VITE_REGISTRY_HOST`, `VITE_REGISTRY_PORT` (443).

**GIN_MODE**: `release` in Docker, debug (verbose Gin logs) on `go run`.

**PostgreSQL**: `postgres:17-alpine` in Compose. `POSTGRES_PORT_EXTERNAL` maps host port 5432 by default — remove the `ports:` stanza to suppress.

**Git auth**: HTTPS only. All git operations set `GIT_TERMINAL_PROMPT=0` to prevent hanging on auth prompts. `isValidGitURL()` in `utils.go` only accepts `https://` URLs.

**No tests, no CI**: `go test ./...` passes vacuously (zero test files). No `.github/workflows/`, no opencode.json, no CLAUDE.md, no .cursorrules.

## Environment Variables

Minimum required in `.env`:
```bash
ENCRYPTION_KEY=<32+ chars>       # openssl rand -base64 32 — AES-256-GCM for credentials
POSTGRES_PASSWORD=<password>
REGISTRY_HOST=registry.lan
```
Note: `.env.example` has insecure defaults – change before deploying.

Optional: `BACKEND_PORT` (9080), `PORT` (9080), `FRONTEND_HOST`, `FRONTEND_PORT`, `VITE_DEV_PORT` — backend CORS allow-list.

## Architecture Notes

- All API calls in `frontend/src/api/index.ts`; all TS types in `frontend/src/types/index.ts`
- `backend/internal/` organized by feature: `api/`, `build/`, `crypto/`, `database/`, `git/`, `gpg/`, `models/`, `registry/`
- Provider binaries at `/app/data/builds`, served at `/downloads/providers/:namespace/*`. Upload uses `multipart/form-data`.
- No migration tool, no codegen, no generated code

## Go Patterns
- Parameterized DB queries only (never string concat)
- Always `defer rows.Close()`
- Check `sql.ErrNoRows` separately from other DB errors
- Decrypt git credentials before use; clean up temp dirs after
- `gin.H{}` for JSON responses with correct HTTP status codes
