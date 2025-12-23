# Private Terraform Registry Platform

Self-hosted Terraform/OpenTofu registry with modules, providers, and deployment management.

## Features

- 🏗️ **Private Module & Provider Registry** - Host your own Terraform resources
- 🚀 **Deployment Management** - Plan, approve, and apply infrastructure changes
- 🔐 **Secure** - Encrypted credentials, GPG signatures, API key auth
- 🌐 **Web UI** - Modern React interface with live log streaming
- 🔄 **Git Integration** - Sync modules/providers from Git repositories

## Quick Start

### 1. Setup

```bash
# Clone and configure
git clone <repository-url>
cd private-tf-platform
cp .env.example .env

# Generate secure keys
openssl rand -base64 32  # Copy to ENCRYPTION_KEY in .env
openssl rand -base64 24  # Copy to POSTGRES_PASSWORD in .env

# Start services
docker compose up -d
```

### 2. Access

- **Web UI**: http://localhost:3000
- **API**: http://localhost:9080

### 3. Configure Terraform CLI

Add to `~/.terraformrc` (Linux/macOS) or `%APPDATA%/terraform.rc` (Windows):

```hcl
host "registry.local:9080" {
  services = {
    "providers.v1" = "http://registry.local:9080/v1/providers/"
    "modules.v1"   = "http://registry.local:9080/v1/modules/"
  }
}

credentials "registry.local:9080" {
  token = "your-api-key-from-ui"
}
```

Add to `/etc/hosts` (Linux/macOS) or `C:\Windows\System32\drivers\etc\hosts` (Windows):

```
127.0.0.1 registry.local
```

## Architecture

| Component | Tech | Port | Description |
|-----------|------|------|-------------|
| **Frontend** | React + TypeScript | 3000 | Web interface |
| **Backend** | Go + Gin | 9080 | REST API & registry protocol |
| **Runner** | Go | 8080 | Executes Terraform/OpenTofu |
| **Database** | PostgreSQL 17 | 5432 | Metadata storage |

## Configuration

Edit `.env` file:

```bash
# Security (required for production)
ENCRYPTION_KEY=<32+ character key>
POSTGRES_PASSWORD=<strong password>

# Domain (used across all services)
REGISTRY_HOST=registry.local

# Ports (optional)
FRONTEND_PORT=3000
BACKEND_PORT=9080
```

## Usage

1. **Create Namespace** → Organize your resources
2. **Add Module/Provider** → Link Git repo, sync versions
3. **Create Deployment** → Select repo directory, configure vars
4. **Run Plan/Apply** → Review changes, approve, deploy

## Common Commands

```bash
# View logs
docker compose logs -f [backend|frontend|runner]

# Stop services
docker compose down

# Reset all data (WARNING: deletes everything)
docker compose down -v

# Update platform
git pull && docker compose up -d --build
```

## Troubleshooting

**Can't access localhost:3000?**
- Check: `docker compose ps` (all containers should be "Up")
- Logs: `docker compose logs frontend`

**Terraform can't find registry?**
- Add `127.0.0.1 registry.local` to `/etc/hosts`
- Verify: `curl http://registry.local:9080/.well-known/terraform.json`

**Deployment fails?**
- Check runner logs: `docker compose logs runner`
- Verify registry URL is accessible from runner container

## Security Notes

- ⚠️ **Change default passwords** in `.env` before production
- 🔒 Use reverse proxy (nginx/traefik) with SSL/TLS
- 🚫 Don't expose PostgreSQL port externally
- 🔑 Generate API keys per namespace for access control

## Development

See component READMEs:
- [Backend](./backend/README.md) - Go API service
- [Frontend](./frontend/README.md) - React UI
- [Runner](./runner/README.md) - Terraform executor

## License

See [LICENSE](./LICENSE) file.

---

_Built with Claude Sonnet 4.5_
