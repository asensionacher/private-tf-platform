# Backend - Terraform Registry API

Go-based REST API service that implements the Terraform Registry Protocol and provides a management API for the private Terraform registry platform.

## Overview

The backend service provides:
- **Terraform Registry Protocol v1** - Full compatibility with `terraform init` and Terraform CLI
- **Management API** - REST endpoints for managing modules, providers, deployments, and namespaces
- **Service Discovery** - Standard Terraform service discovery endpoint
- **Authentication** - API key-based authentication for Terraform CLI access
- **Encryption** - Secure storage of Git credentials and sensitive data
- **GPG Signing** - Optional provider binary signing for verification

## Architecture

### Technology Stack
- **Framework**: Gin (HTTP web framework)
- **Database**: PostgreSQL 17
- **Language**: Go 1.24.0
- **Key Libraries**:
  - `github.com/gin-gonic/gin` - HTTP routing and middleware
  - `github.com/lib/pq` - PostgreSQL driver
  - `github.com/gin-contrib/cors` - CORS middleware
  - `github.com/google/uuid` - UUID generation

### Project Structure

```
backend/
├── internal/
│   ├── api/              # HTTP handlers and middleware
│   │   ├── deployments.go    # Deployment management endpoints
│   │   ├── discovery.go      # Terraform service discovery
│   │   ├── modules.go        # Module management endpoints
│   │   ├── namespaces.go     # Namespace management endpoints
│   │   ├── providers.go      # Provider management endpoints
│   │   ├── registry.go       # Registry token management
│   │   └── utils.go          # Common API utilities
│   ├── build/            # Terraform build and execution
│   │   ├── build.go          # Build orchestration
│   │   └── terraform.go      # Terraform CLI wrapper
│   ├── crypto/           # Encryption and security
│   │   └── crypto.go         # AES encryption for credentials
│   ├── database/         # Database layer
│   │   └── database.go       # Connection, migrations, schema
│   ├── git/              # Git operations
│   │   ├── git.go            # Clone, checkout, tag operations
│   │   └── git_additions.go  # Additional Git utilities
│   ├── gpg/              # GPG signing
│   │   └── gpg.go            # Provider binary signing
│   ├── models/           # Database models
│   │   ├── deployment.go     # Deployment and run models
│   │   ├── module.go         # Module and version models
│   │   ├── namespace.go      # Namespace model
│   │   └── provider.go       # Provider and platform models
│   └── registry/         # Registry-specific logic
│       └── token.go          # Registry token generation
├── Dockerfile            # Multi-stage Docker build
├── go.mod                # Go module dependencies
├── go.sum                # Dependency checksums
└── main.go               # Application entry point
```

## Database Schema

The backend uses PostgreSQL with the following tables:

### Core Tables
- **namespaces** - Organizations/authorities (e.g., `hashicorp`, `private`)
- **api_keys** - Global API keys for Terraform CLI authentication
- **modules** - Terraform modules with Git source information
- **module_versions** - Specific versions of modules
- **providers** - Terraform providers with Git source information
- **provider_versions** - Specific versions of providers
- **provider_platforms** - Platform-specific binaries (OS/arch combinations)
- **deployments** - IaC deployment configurations
- **deployment_runs** - Individual plan/apply execution runs

### Key Relationships
- Modules and Providers belong to Namespaces (one-to-many)
- Versions belong to Modules/Providers (one-to-many)
- Platforms belong to Provider Versions (one-to-many)
- Deployment Runs belong to Deployments (one-to-many)

See `internal/database/database.go` lines 57-228 for the complete schema.

## API Endpoints

### Terraform Registry Protocol (requires API key)

#### Service Discovery
```
GET /.well-known/terraform.json
```

#### Module Registry
```
GET /v1/modules/:namespace/:name/:provider/versions
GET /v1/modules/:namespace/:name/:provider/:version/download
```

#### Provider Registry
```
GET /v1/providers/:namespace/:name/versions
GET /v1/providers/:namespace/:name/:version/download/:os/:arch
```

#### Provider Verification
```
GET /shasums/providers/:namespace/:name/:version
GET /shasums/providers/:namespace/:name/:version/sig
```

### Management API (no authentication)

#### Modules
```
GET    /api/modules                          # List all modules
GET    /api/modules/:id                      # Get module details
GET    /api/modules/:id/versions             # List module versions
GET    /api/modules/:id/git-tags             # Get available Git tags
GET    /api/modules/:id/readme               # Get module README
POST   /api/modules                          # Create module from Git
PUT    /api/modules/:id                      # Update module
DELETE /api/modules/:id                      # Delete module
POST   /api/modules/:id/sync-tags            # Sync Git tags
POST   /api/modules/:id/versions             # Add version
PATCH  /api/modules/:id/versions/:versionId  # Toggle version enabled/disabled
```

#### Providers
```
GET    /api/providers                                            # List all providers
GET    /api/providers/:id                                        # Get provider details
GET    /api/providers/:id/versions                               # List provider versions
GET    /api/providers/:id/git-tags                               # Get available Git tags
GET    /api/providers/:id/readme                                 # Get provider README
POST   /api/providers                                            # Create provider from Git
DELETE /api/providers/:id                                        # Delete provider
POST   /api/providers/:id/sync-tags                              # Sync Git tags
POST   /api/providers/:id/versions                               # Add version
PATCH  /api/providers/:id/versions/:versionId                    # Toggle version enabled/disabled
GET    /api/providers/:id/versions/:versionId/platforms          # List platform binaries
POST   /api/providers/:id/versions/:versionId/platforms          # Add platform binary
POST   /api/providers/:id/versions/:versionId/platforms/upload   # Upload platform binary
```

#### Namespaces
```
GET    /api/namespaces        # List all namespaces
GET    /api/namespaces/:id    # Get namespace details
POST   /api/namespaces        # Create namespace
PATCH  /api/namespaces/:id    # Update namespace
DELETE /api/namespaces/:id    # Delete namespace
```

#### API Keys
```
GET    /api/api-keys           # List all API keys
POST   /api/api-keys           # Create API key
DELETE /api/api-keys/:keyId    # Delete API key
```

#### Deployments
```
GET    /api/deployments                                  # List all deployments
GET    /api/deployments/:id                              # Get deployment details
POST   /api/deployments                                  # Create deployment
DELETE /api/deployments/:id                              # Delete deployment
GET    /api/deployments/:id/references                   # Get module/provider references
GET    /api/deployments/:id/browse                       # Browse Git repository
GET    /api/deployments/:id/tfvars                       # Get .tfvars files
GET    /api/deployments/:id/status                       # Get directory status
POST   /api/deployments/:id/runs                         # Create deployment run
GET    /api/deployments/:id/runs                         # List deployment runs
GET    /api/deployments/:id/runs/:runId                  # Get run details
GET    /api/deployments/:id/runs/:runId/stream           # Stream run logs
POST   /api/deployments/:id/runs/:runId/approve          # Approve pending run
POST   /api/deployments/:id/runs/:runId/cancel           # Cancel running run
DELETE /api/deployments/:id/runs/:runId                  # Delete run
```

## Setup and Installation

### Prerequisites

- **Go 1.24+** - [Install Go](https://golang.org/doc/install)
- **PostgreSQL 17** - [Install PostgreSQL](https://www.postgresql.org/download/)
- **Git** - Required for Git operations
- **Terraform or OpenTofu** (optional) - For local testing

### Manual Setup (without Docker)

1. **Clone the repository**
   ```bash
   cd backend
   ```

2. **Install Go dependencies**
   ```bash
   go mod download
   ```

3. **Set up PostgreSQL database**
   ```bash
   # Connect to PostgreSQL
   psql -U postgres
   
   # Create database and user
   CREATE DATABASE registry;
   CREATE USER registry WITH ENCRYPTED PASSWORD 'your-secure-password';
   GRANT ALL PRIVILEGES ON DATABASE registry TO registry;
   ```

4. **Configure environment variables**
   ```bash
   cp ../.env.example .env
   # Edit .env with your settings
   
   # Required variables:
   export POSTGRES_HOST=localhost
   export POSTGRES_PORT=5432
   export POSTGRES_USER=registry
   export POSTGRES_PASSWORD=your-secure-password
   export POSTGRES_DB=registry
   export ENCRYPTION_KEY=$(openssl rand -base64 32)
   export PORT=9080
   export REGISTRY_HOST=localhost
   export FRONTEND_HOST=localhost
   export FRONTEND_PORT=3000
   ```

5. **Build the application**
   ```bash
   go build -o iac-tool main.go
   ```

6. **Run the application**
   ```bash
   ./iac-tool
   ```

   You should see:
   ```
   ✓ Registry authentication token initialized
   Database initialized successfully (PostgreSQL at localhost:5432)
   Terraform Private Registry starting on :9080
   Service discovery: http://localhost:9080/.well-known/terraform.json
   Module registry:   http://localhost:9080/v1/modules/
   Provider registry: http://localhost:9080/v1/providers/
   Management API:    http://localhost:9080/api/
   ```

### Docker Setup

Using Docker Compose (recommended):

```bash
# From project root
docker-compose up -d backend

# View logs
docker-compose logs -f backend

# Rebuild after changes
docker-compose up -d --build backend
```

Using standalone Docker:

```bash
# Build
docker build -t iac-backend .

# Run
docker run -d \
  -p 9080:9080 \
  -e POSTGRES_HOST=host.docker.internal \
  -e POSTGRES_PASSWORD=your-password \
  -e ENCRYPTION_KEY=$(openssl rand -base64 32) \
  --name iac-backend \
  iac-backend
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9080` | HTTP server port |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `registry` | PostgreSQL username |
| `POSTGRES_PASSWORD` | `registry` | PostgreSQL password |
| `POSTGRES_DB` | `registry` | PostgreSQL database name |
| `ENCRYPTION_KEY` | **required** | 32+ character encryption key for credentials |
| `REGISTRY_HOST` | `localhost` | Registry hostname (for logs) |
| `FRONTEND_HOST` | `localhost` | Frontend hostname (for CORS) |
| `FRONTEND_PORT` | `3000` | Frontend port (for CORS) |
| `VITE_DEV_PORT` | `5173` | Vite dev server port (for CORS) |
| `BUILD_DIR` | `/app/data/builds` | Directory for provider binaries |
| `GPG_KEY_ID` | _(optional)_ | GPG key ID for provider signing |
| `GPG_PRIVATE_KEY` | _(optional)_ | GPG private key (base64 encoded) |
| `GPG_PASSPHRASE` | _(optional)_ | GPG key passphrase |

### Security Configuration

**CORS**: In production, update `main.go` lines 66-70 to specify exact origins:
```go
config.AllowOrigins = []string{
    "https://your-frontend-domain.com",
}
// Remove the "*" wildcard!
```

**API Authentication**: Terraform CLI endpoints (`/v1/*`) require API key authentication via `Authorization: Bearer <token>` header.

**Encryption**: Git credentials and sensitive data are encrypted using AES-256-GCM with the `ENCRYPTION_KEY`.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/api -v

# Run specific test
go test -run TestFunctionName ./...
```

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Import order: stdlib, external packages, internal packages (separated by blank lines)
- Always check errors; never ignore `error` return values
- Use descriptive variable names; avoid abbreviations
- Add comments for exported functions and complex logic
- Use parameterized queries for all database operations

### Adding New Endpoints

1. Create handler function in appropriate file under `internal/api/`
2. Register route in `main.go` (lines 118-182)
3. Add model if needed in `internal/models/`
4. Update database schema if needed in `internal/database/database.go`
5. Test endpoint manually or add tests

Example handler:
```go
func GetExample(c *gin.Context) {
    id := c.Param("id")
    
    // Query database
    var result SomeModel
    err := database.DB.QueryRow("SELECT * FROM table WHERE id = $1", id).Scan(&result.Field)
    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, result)
}
```

### Database Migrations

Currently uses simple `CREATE TABLE IF NOT EXISTS` statements in `internal/database/database.go`. For production, consider using a migration tool like:
- [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- [pressly/goose](https://github.com/pressly/goose)

## Troubleshooting

### Database Connection Issues

```bash
# Check PostgreSQL is running
psql -U registry -d registry -h localhost

# Check environment variables
env | grep POSTGRES

# Check database logs
docker-compose logs postgres
```

### Encryption Key Errors

```bash
# Generate new encryption key
openssl rand -base64 32

# Set in environment
export ENCRYPTION_KEY="<generated-key>"
```

### CORS Errors

Check that `FRONTEND_HOST` and `FRONTEND_PORT` match your frontend configuration.

### Port Already in Use

```bash
# Find process using port 9080
lsof -i :9080

# Kill process
kill -9 <PID>
```

## Production Deployment

### Checklist

- [ ] Set strong `ENCRYPTION_KEY` (32+ characters)
- [ ] Set secure `POSTGRES_PASSWORD`
- [ ] Update CORS origins (remove `*` wildcard)
- [ ] Enable PostgreSQL SSL/TLS
- [ ] Do NOT expose PostgreSQL port externally
- [ ] Use reverse proxy (nginx/traefik) with SSL/TLS termination
- [ ] Set up database backups
- [ ] Configure log aggregation
- [ ] Set up monitoring and alerts
- [ ] Rotate API keys and credentials every 90 days

### Reverse Proxy Example (nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name registry.yourdomain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:9080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Related Documentation

- [Root README](../README.md) - Full project overview and quick start
- [Frontend README](../frontend/README.md) - Frontend development guide
- [Runner README](../runner/README.md) - Runner service documentation
- [Terraform Registry Protocol](https://developer.hashicorp.com/terraform/registry/api-docs) - Official protocol documentation

## License

See [../LICENSE](../LICENSE) for details.

_This project was fully deployed using Cloude Sonnet 4.5_
