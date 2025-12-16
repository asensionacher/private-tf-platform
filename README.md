# Terraform Private Registry

A self-hosted private registry for Terraform that supports both modules and providers. Host your own Terraform registry compatible with the official protocol, featuring a web interface for management and automatic synchronization from Git repositories.

## What does this application do?

This platform allows you to:

- **Host private Terraform modules**: Publish and version your internal Terraform modules
- **Host private Terraform providers**: Distribute custom providers with automatic GPG signing
- **Git synchronization**: Automatically import versions from Git tags
- **Visual management**: Modern web interface to manage modules, providers, and versions
- **Terraform CLI compatible**: Works directly with `terraform init` without modifications
- **Multi-namespace**: Organize resources by teams or organizations
- **Version control**: Enable/disable specific versions as needed

## Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐      ┌─────────────┐
│  Terraform  │─────▶│   Frontend   │─────▶│   Backend   │─────▶│   Runner    │
│     CLI     │      │ (React + TS) │      │  (Go + DB)  │      │(TF/OpenTofu)│
└─────────────┘      └──────────────┘      └─────────────┘      └─────────────┘
                            │                      │                     │
                            │                      ▼                     ▼
                            │              ┌──────────────┐      ┌──────────────┐
                            └─────────────▶│ Git Repos    │      │ Deployments  │
                                           └──────────────┘      └──────────────┘
```

- **Frontend**: React + TypeScript + Vite + Tailwind CSS
- **Backend**: REST API in Go with SQLite database
- **Runner**: Isolated executable that runs Terraform/OpenTofu commands
- **Synchronization**: Automatic Git repo cloning to extract tags/versions
- **GPG Signing**: Automatic generation and signing of provider binaries

### Runner Architecture

The platform uses a separated runner architecture for executing Terraform/OpenTofu deployments:

- **Isolation**: The runner is a separate executable that handles all IaC tool execution
- **Security**: Backend and runner communicate via stdin/stdout JSON protocol
- **Flexibility**: Easy to scale runners independently or run them in separate containers
- **Tool Support**: Supports both Terraform and OpenTofu seamlessly

See [runner/README.md](runner/README.md) for more details on the runner component.

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Git repositories with your Terraform modules/providers

### Installation

1. Clone the repository:
```bash
git clone https://github.com/asensionacher/private-tf-platform.git
cd private-tf-platform
```

2. Start the services:
```bash
docker compose up -d
```

3. Access the web interface at `http://localhost:3000`

The backend API will be available at `http://localhost:9080`

## Configuration

### Environment Variables

#### Backend (API Server)
- `PORT`: API server port (default: 9080)
- `BASE_URL`: Base URL for the registry (default: http://localhost:9080)
- `DB_PATH`: SQLite database path (default: /app/data/registry.db)
- `GPG_HOME`: GPG home directory for signing keys (default: /app/data/gpg)
- `ENCRYPTION_KEY`: **Required for production** - 32-byte encryption key for securing credentials (SSH keys, passwords, tokens)

#### Frontend
- `VITE_API_URL`: Backend API URL (default: http://localhost:9080)

### Docker Compose

Edit `docker-compose.yml` to customize ports and volumes:

```yaml
services:
  backend:
    ports:
      - "9080:9080"  # Change host port as needed
    volumes:
      - ./data:/app/data  # Persistent storage
    environment:
      - BASE_URL=https://registry.example.com  # Your domain
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}  # Set in .env file
```

### Security Configuration

**Important**: For production environments, you must set a secure encryption key:

1. Generate a strong encryption key:
```bash
openssl rand -base64 32
```

2. Create a `.env` file in the project root:
```bash
cp .env.example .env
# Edit .env and set ENCRYPTION_KEY to your generated key
```

3. Restart the containers:
```bash
docker compose down && docker compose up -d
```

The encryption key is used to protect sensitive authentication data (SSH private keys, HTTPS passwords/tokens) stored in the database using AES-256-GCM encryption.

## Usage

### Using Terraform CLI

#### Configure Terraform to use your registry

Create or edit `~/.terraformrc`:

```hcl
host "localhost.localdomain:3000" {
  services = {
    "modules.v1"   = "http://localhost.localdomain:3000/v1/modules/",
    "providers.v1" = "http://localhost.localdomain:3000/v1/providers/"
  }
}
```

For production, replace `localhost.localdomain:3000` with your domain.

#### Using Modules

```hcl
module "example" {
  source  = "localhost.localdomain:3000/namespace/name/provider"
  version = "1.0.0"
}
```

#### Using Providers

```hcl
terraform {
  required_providers {
    azurerm = {
      source  = "localhost.localdomain:3000/default/azurerm"
      version = "4.55.0"
    }
  }
}
```

### Managing Modules

1. **Add a Module**: 
   - Navigate to Modules page
   - Click "Add Module"
   - Provide namespace, name, provider, and Git source URL
   
2. **Sync Versions**:
   - Open module details
   - Click "Sync Tags" to fetch versions from Git repository
   - Enable the versions you want to publish

3. **API Key**: Required for creating/updating modules
   - Navigate to Namespaces → Select namespace → Generate API key
   - Use the key in `X-API-Key` header for API requests

### Managing Providers

1. **Add a Provider**:
   - Navigate to Providers page
   - Click "Add Provider"
   - Provide namespace, name, and Git source URL

2. **Sync Versions**:
   - Open provider details
   - Click "Sync Tags" to fetch versions from Git repository
   - Add platform binaries for each version

3. **Platform Binaries**:
   - Each version needs binaries for target platforms (linux_amd64, darwin_arm64, etc.)
   - Binaries are automatically signed with GPG

## API Endpoints

### Service Discovery
- `GET /.well-known/terraform.json` - Service discovery endpoint

### Modules
- `GET /v1/modules/:namespace/:name/:provider/versions` - List module versions
- `GET /v1/modules/:namespace/:name/:provider/:version/download` - Download module

### Providers  
- `GET /v1/providers/:namespace/:name/versions` - List provider versions
- `GET /v1/providers/:namespace/:name/:version/download/:os/:arch` - Download provider

### Management API (requires API key)
- `GET /api/modules` - List all modules
- `POST /api/modules` - Create module
- `POST /api/modules/:id/sync-tags` - Sync versions from Git
- `GET /api/providers` - List all providers
- `POST /api/providers` - Create provider
- `POST /api/providers/:id/sync-tags` - Sync versions from Git

## Architecture

- **Backend**: Go-based API server with SQLite database
- **Frontend**: React + TypeScript with Vite
- **Web Server**: Nginx serving the frontend
- **Database**: SQLite for module/provider metadata
- **Storage**: File-based storage for GPG keys and data

## Development

### Backend Development
```bash
cd backend
go run .
```

### Frontend Development
```bash
cd frontend
pnpm install
pnpm dev
```

### Building
```bash
# Build both services
docker compose build

# Build specific service
docker compose build backend
docker compose build frontend
```

## Data Persistence

All data is stored in the `./data` directory (mounted as volume):
- `registry.db` - SQLite database
- `gpg/` - GPG keys for signing

Make sure to backup this directory regularly.

## Security Considerations

1. **API Keys**: Store API keys securely and rotate them regularly
2. **GPG Keys**: Automatically generated and stored in the GPG home directory
3. **HTTPS**: Use HTTPS in production with a reverse proxy (nginx, traefik, etc.)
4. **Network**: Consider restricting network access to the registry
5. **Authentication**: API keys are required for write operations

## Production Deployment

For production deployments:

1. Use a reverse proxy (nginx, traefik) with HTTPS
2. Configure proper BASE_URL in environment variables
3. Set up regular backups of the data directory
4. Configure resource limits in docker-compose.yml
5. Monitor logs and disk usage
6. Consider using external database for high availability

Example nginx configuration:
```nginx
server {
    listen 443 ssl http2;
    server_name registry.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/ {
        proxy_pass http://localhost:9080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Troubleshooting

### Module/Provider not found
- Verify the source URL is correct
- Check that versions are synced and enabled
- Ensure API key is set correctly for private namespaces

### GPG signature verification failed
- Delete the module/provider and re-sync to regenerate signatures
- Check backend logs for GPG errors

### Git clone failures
- Ensure Git repositories are accessible
- Check SSH keys if using SSH URLs
- Verify network connectivity from container

### Version ordering issues
- Re-sync tags to update tag dates
- Versions are sorted by tag date (newest first)

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing issues for solutions
- Review the troubleshooting section

## Acknowledgments

Built with:
- Go
- React + TypeScript
- SQLite
- Docker
- Terraform Protocol
