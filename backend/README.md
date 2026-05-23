# Backend

Go + Gin REST API. Implements the Terraform Registry Protocol v1, a Terraform HTTP state backend, and a management API for the web UI.

Port `9080` internally; exposed via nginx on `api.registry.lan:443` and `tf.registry.lan:443`.

## Run locally

```bash
cd backend
go run main.go
```

Required environment variables (see `.env.example` in the repo root):

```bash
ENCRYPTION_KEY=...
POSTGRES_HOST=localhost
POSTGRES_PASSWORD=...
```

## Test

```bash
go test ./...
go test ./internal/api -v
go test -run TestFunctionName ./...
```

## Build

```bash
go build -o iac-tool main.go
```

The Docker image is built automatically by `docker compose up --build`. The `go` binary is typically not on the host PATH — use Docker for verification.

## Project layout

```
backend/
├── main.go                   # Entry point, route registration
└── internal/
    ├── api/
    │   ├── deployments.go    # Deployment CRUD + git browser
    │   ├── discovery.go      # /.well-known/terraform.json + getBaseURL()
    │   ├── modules.go        # Module management + TF registry protocol
    │   ├── namespaces.go     # Namespaces, API keys, TerraformAuthMiddleware
    │   ├── providers.go      # Provider management + TF registry protocol
    │   ├── tfstate.go        # HTTP state backend + UI management endpoints
    │   └── utils.go          # Shared helpers
    ├── crypto/               # AES-256-GCM encryption for stored credentials
    ├── database/             # PostgreSQL connection + schema (CREATE TABLE IF NOT EXISTS)
    ├── git/                  # Clone, list branches/tags, list directory
    ├── gpg/                  # Provider binary signing
    ├── models/               # Struct definitions
    └── registry/             # Registry authentication token
```

## Key endpoints

### Terraform registry protocol (`/v1/*`) — requires API key

```
GET /.well-known/terraform.json
GET /v1/providers/:namespace/:name/versions
GET /v1/providers/:namespace/:name/:version/download/:os/:arch
GET /v1/modules/:namespace/:name/:provider/versions
GET /v1/modules/:namespace/:name/:provider/:version/download
GET /shasums/providers/:namespace/:name/:version
GET /shasums/providers/:namespace/:name/:version/sig
GET /downloads/providers/:namespace/*filepath
```

Authentication is enforced by `TerraformAuthMiddleware` for private namespaces. Public namespaces (`is_public = true`) are accessible without a key.

`/shasums` and `/downloads` are intentionally open — Terraform/OpenTofu never forwards `Authorization` headers to URLs returned inside JSON response bodies, regardless of `.terraformrc` configuration.

### HTTP state backend (`/api/tfstate/*`) — no auth

```
GET    /api/tfstate/:deploymentId
POST   /api/tfstate/:deploymentId
DELETE /api/tfstate/:deploymentId
LOCK   /api/tfstate/:deploymentId
UNLOCK /api/tfstate/:deploymentId
```

OpenTofu/Terraform appends `:<workspace>` to the address URL (bare `:` for the default workspace). All handlers call `parseDeploymentAndWorkspace()` — never read `c.Param("deploymentId")` directly. The backend does not reject POSTs by serial number; serial conflict detection is the client's responsibility.

State files live on disk at `/app/data/tfstates/<deploymentId>/<workspace>.tfstate`, not in PostgreSQL.

### Management API (`/api/*`) — no auth

```
/api/namespaces, /api/api-keys
/api/modules, /api/providers
/api/deployments
/api/tfstates, /api/deployments/:id/tfstates
```

## Database

No migration tool. Schema is applied via `CREATE TABLE IF NOT EXISTS` in `internal/database/database.go` on every startup. Adding columns to existing tables requires a manual `ALTER TABLE`.

A `default` namespace (`id = 'default'`, `is_public = true`) is auto-seeded with `ON CONFLICT DO NOTHING`.
