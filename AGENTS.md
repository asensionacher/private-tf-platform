# Agent Guidelines for IAC Platform

## Project Structure
- **Backend**: Go (Gin framework) in `/backend` - REST API with PostgreSQL
- **Frontend**: React + TypeScript + Vite + Tailwind in `/frontend` - Web UI
- **Runner**: Go service in `/runner` - Executes Terraform/OpenTofu commands

## Build & Test Commands

### Backend (Go)
```bash
cd backend
go build -o iac-tool main.go              # Build
go run main.go                             # Run
go test ./...                              # Run all tests
go test ./internal/api -v                  # Run specific package tests
go test -run TestFunctionName ./...       # Run single test
```

### Frontend (TypeScript/React)
```bash
cd frontend
pnpm install                               # Install dependencies
pnpm dev                                   # Dev server (port 5173)
pnpm build                                 # Build (tsc + vite build)
pnpm lint                                  # ESLint check
```

### Runner (Go)
```bash
cd runner
go build -o iac-runner main.go            # Build
go run main.go                             # Run
go test ./...                              # Run all tests
```

### Docker Compose
```bash
docker-compose up -d                       # Start all services
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d  # Production
docker-compose logs -f backend             # View logs
./scripts/validate-env.sh                  # Validate environment
./scripts/setup-env.sh                     # Setup production env
```

## Code Style Guidelines

### Go (Backend/Runner)
- **Imports**: Group stdlib, external, then internal (separated by blank lines)
- **Naming**: CamelCase for exports, camelCase for private. Use descriptive names.
- **Error Handling**: Always check errors. Return errors, don't panic in library code. Log with context.
- **Comments**: Use `//` for single-line. Add function doc comments for exported functions.
- **Formatting**: Use `go fmt`. No tabs/spaces mixing.
- **Database**: Use parameterized queries (never string concatenation). Always defer `rows.Close()`.
- **HTTP**: Return proper status codes. Use `gin.H{}` for JSON responses.
- **Structure**: Organize by feature in `/internal` (api, database, models, crypto, git, gpg, registry)

### TypeScript (Frontend)
- **Types**: Use explicit types, avoid `any`. Import types with `import type { ... }`
- **Imports**: External deps first, then internal (`../api`, `../types`, etc.), React hooks last
- **Components**: Functional components with hooks. Export default for pages.
- **Naming**: PascalCase for components/types, camelCase for functions/variables
- **State**: Use React Query for server state, useState for local UI state
- **Error Handling**: Try/catch in mutations, display user-friendly errors
- **Formatting**: 2-space indents, single quotes, semicolons
- **Config**: Strict TypeScript (`strict: true`, `noUnusedLocals: true`, `noUnusedParameters: true`)
- **API**: Centralize API calls in `/src/api/index.ts`, use axios with typed responses

### Environment Variables
- **Required**: `ENCRYPTION_KEY` (32+ chars), `POSTGRES_PASSWORD`
- **Hosts**: `FRONTEND_HOST`, `BACKEND_HOST`, `REGISTRY_HOST` (default: registry.local for REGISTRY_HOST)
- **Ports**: `FRONTEND_PORT` (3000), `BACKEND_PORT` (9080), `RUNNER_PORT` (8080)
- **Security**: Never commit `.env` with real credentials. Use `.env.example` as template.
- **Validation**: Run `./scripts/validate-env.sh` before deployment

### Security Best Practices
- **CORS**: Production must specify exact origins (no `*` wildcard)
- **PostgreSQL**: Never expose externally in production (comment out `POSTGRES_PORT_EXTERNAL`)
- **Secrets**: Use `openssl rand -base64 32` for keys, rotate every 90 days
- **TLS**: Production requires reverse proxy with SSL/TLS termination
- **Logging**: Log auth attempts, API access, errors - never log secrets

### Common Patterns
- **Go API Handlers**: Accept `*gin.Context`, return JSON with proper status codes, handle errors gracefully
- **React Components**: Use hooks (useState, useEffect, useQuery, useMutation), destructure props
- **Database Queries**: Use prepared statements, check `sql.ErrNoRows` separately from other errors
- **Git Operations**: Always decrypt credentials before use, cleanup temp directories
- **Module Structure**: Keep related code together (models with API handlers that use them)

### Documentation
- Update README.md for user-facing changes
- Update docs/SECURITY.md for security-related changes
- Update docs/DEPLOYMENT.md for deployment changes
- Add comments for complex business logic
- Keep CHANGELOG.md updated with notable changes

### Testing (when tests exist)
- Go: Table-driven tests, test error cases, use `t.Run()` for subtests
- React: Test user interactions, not implementation details
- Integration: Test API contracts between services
