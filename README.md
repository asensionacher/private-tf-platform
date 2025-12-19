# Private Terraform Registry Platform

A self-hosted, private Terraform/OpenTofu registry and deployment platform that allows you to manage your own Terraform modules, providers, and Infrastructure as Code (IaC) deployments.

## Features

- **Private Module Registry**: Host and version your own Terraform modules with Git-based synchronization
- **Private Provider Registry**: Mirror and distribute Terraform providers with multi-platform binary support
- **IaC Deployment Management**: Execute and manage Terraform/OpenTofu deployments directly from the UI
- **Namespace Management**: Organize modules and providers with namespaces and API key authentication
- **Git Integration**: Automatic synchronization of modules from Git repositories (public and private)
- **Web UI**: Modern React-based interface with dark mode support
- **Deployment Runs**: Plan, review, approve, and apply infrastructure changes with live log streaming
- **Security**: GPG signature verification, encrypted credentials, and namespace-based access control

## Architecture

The platform consists of four main components:

### 1. **Backend** (`/backend`)
- Go-based REST API server using Gin framework
- Manages modules, providers, namespaces, and deployments
- PostgreSQL database for metadata storage
- Git operations and GPG signature verification
- Handles authentication and encryption

### 2. **Frontend** (`/frontend`)
- React + TypeScript single-page application
- Vite for fast development and optimized builds
- Tailwind CSS for styling
- Real-time polling for deployment status updates

### 3. **Runner** (`/runner`)
- Go-based service that executes Terraform/OpenTofu commands
- Isolated execution environment for deployment runs
- Supports both Terraform and OpenTofu
- Includes AWS CLI and Azure CLI for cloud provider integrations

### 4. **Database**
- PostgreSQL 17 for persistent data storage
- Stores modules, providers, namespaces, API keys, and deployment metadata

## Quick Start with Docker Compose

### Prerequisites

- Docker and Docker Compose installed
- At least 2GB of free disk space
- Ports 3000 (frontend), 9080 (backend), and 8080 (runner) available

### Installation Steps

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd private-tf-platform
   ```

2. **Create environment configuration**
   ```bash
   cp .env.example .env
   ```

3. **Generate secure keys** (Important for production!)
   ```bashlocalhost:3000/deployments/b9f3f535-2aaa-48ff-855c-7b9a1548b058?branch=main
   # Generate encryption key
   openssl rand -base64 32
   
   # Generate database password
   openssl rand -base64 24
   ```
   
   Update `.env` file with these values:
   ```bash
   ENCRYPTION_KEY=<generated-key-here>
   POSTGRES_PASSWORD=<generated-password-here>
   ```

4. **Start the platform**
   ```bash
   docker compose up -d
   ```

5. **Access the platform**
   - Frontend UI: http://localhost:3000
   - Backend API: http://localhost:9080
   - API Documentation: http://localhost:9080/swagger/index.html (if enabled)

6. **Configure your Terraform CLI**
   
   Add to your `~/.terraformrc` (Linux/macOS) or `%APPDATA%/terraform.rc` (Windows):
   ```hcl
   host "registry.local:9080" {
     services = {
       "providers.v1" = "http://registry.local:9080/v1/providers/"
       "modules.v1"   = "http://registry.local:9080/v1/modules/"
     }
   }
   
   credentials "registry.local:9080" {
     token = "your-api-key-here"
   }
   ```
   
   Add to your `/etc/hosts` (Linux/macOS) or `C:\Windows\System32\drivers\etc\hosts` (Windows):
   ```
   127.0.0.1 registry.local
   ```

### Stopping the Platform

```bash
# Stop all services
docker compose down

# Stop and remove all data (WARNING: This deletes all data!)
docker compose down -v
```

## Manual Setup (Development)

Each component can be run independently for development purposes. See individual README files:

- [Backend Manual Setup](./backend/README.md)
- [Frontend Manual Setup](./frontend/README.md)
- [Runner Manual Setup](./runner/README.md)

## Project Structure

```
.
├── backend/           # Go backend API service
│   ├── internal/      # Internal packages
│   │   ├── api/       # HTTP handlers
│   │   ├── build/     # Terraform build logic
│   │   ├── crypto/    # Encryption utilities
│   │   ├── database/  # Database connection
│   │   ├── git/       # Git operations
│   │   ├── gpg/       # GPG verification
│   │   ├── models/    # Data models
│   │   └── registry/  # Registry logic
│   ├── Dockerfile
│   └── main.go
│
├── frontend/          # React TypeScript frontend
│   ├── src/
│   │   ├── api/       # API client
│   │   ├── components/# React components
│   │   ├── context/   # React context
│   │   ├── pages/     # Page components
│   │   └── types/     # TypeScript types
│   ├── Dockerfile
│   └── package.json
│
├── runner/            # Terraform/OpenTofu runner service
│   ├── Dockerfile
│   └── main.go
│
├── scripts/           # Utility scripts
│   ├── setup-env.sh   # Automated environment setup
│   └── validate-env.sh# Environment validation
│
├── docs/              # Documentation
│   └── SECURITY.md    # Security guidelines
│
├── docker-compose.yml # Docker Compose configuration
├── .env.example       # Environment template
└── README.md          # This file
```

## Configuration

### Environment Variables

Key configuration options in `.env`:

- `ENCRYPTION_KEY`: Encryption key for sensitive data (min 32 characters)
- `POSTGRES_PASSWORD`: PostgreSQL password
- `FRONTEND_HOST`: Frontend hostname (default: localhost)
- `BACKEND_HOST`: Backend hostname (default: localhost)
- `REGISTRY_HOST`: Registry hostname for Terraform (default: registry.local)
- `FRONTEND_PORT`: Frontend port (default: 3000)
- `BACKEND_PORT`: Backend port (default: 9080)
- `RUNNER_PORT`: Runner port (default: 8080)

See [.env.example](./.env.example) for full documentation.

### Hostnames

The platform uses `registry.local` as the default registry hostname. This is required because:
1. Terraform requires a hostname with a dot (`.`) for custom registries
2. Using `localhost` doesn't work properly with Terraform's service discovery

**Important**: Add `127.0.0.1 registry.local` to your `/etc/hosts` file.

## Usage

### 1. Create a Namespace

Namespaces organize your modules and providers:
1. Navigate to "Namespaces" in the UI
2. Click "New Namespace"
3. Enter a name (e.g., `mycompany`)
4. Create an API key for the namespace

### 2. Add a Terraform Module

1. Navigate to "Modules"
2. Click "New Module"
3. Select namespace
4. Enter module name and Git URL
5. For private repositories, provide Git credentials
6. Click "Sync Tags" to import versions

### 3. Add a Terraform Provider

1. Navigate to "Providers"
2. Click "New Provider"
3. Select namespace
4. Enter provider name and Git URL
5. Upload platform binaries for each version
6. Enable versions to make them available

### 4. Create a Deployment

1. Navigate to "Deployments"
2. Click "New Deployment"
3. Enter name and Git repository URL
4. Browse the repository and select a directory
5. Click "Deploy" to create a deployment run
6. Configure Terraform/OpenTofu options
7. Review plan output and approve to apply

## Deployment Workflow

1. **Initialize**: Runner clones repository and runs `terraform init`
2. **Plan**: Runner executes `terraform plan` and shows output
3. **Approve**: Review changes and approve in the UI
4. **Apply**: Runner executes `terraform apply` with live log streaming
5. **Complete**: View outputs and final state

## Security Considerations

- **Change default credentials**: Always update `ENCRYPTION_KEY` and `POSTGRES_PASSWORD`
- **Use strong encryption keys**: Generate with `openssl rand -base64 32`
- **Secure Git credentials**: Credentials are encrypted at rest
- **API key management**: Create separate keys for different namespaces
- **Network security**: Use reverse proxy with SSL/TLS in production
- **Database access**: Don't expose PostgreSQL port externally in production

See [docs/SECURITY.md](./docs/SECURITY.md) for detailed security guidelines.

## Troubleshooting

### Common Issues

**1. Cannot access frontend at localhost:3000**
- Check if port is already in use: `docker ps`
- Verify containers are running: `docker compose ps`
- Check logs: `docker compose logs frontend`

**2. Terraform can't connect to registry**
- Verify `/etc/hosts` has `127.0.0.1 registry.local`
- Check `~/.terraformrc` configuration
- Ensure backend is running: `docker compose ps backend`
- Test API: `curl http://registry.local:9080/health`

**3. Git synchronization fails**
- Check Git credentials are correct
- Verify repository URL is accessible
- Check backend logs: `docker compose logs backend`

**4. Deployment runs fail**
- Check runner logs: `docker compose logs runner`
- Verify environment variables are set correctly
- Ensure runner can access the registry

### Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f runner
docker compose logs -f postgres
```

### Resetting Data

```bash
# Stop services and remove all data
docker compose down -v

# Remove only specific volumes
docker volume rm private-tf-platform_postgres-data
docker volume rm private-tf-platform_registry-data
docker volume rm private-tf-platform_deployment-workdir
```

## Updating

```bash
# Pull latest changes
git pull

# Rebuild and restart services
docker compose down
docker compose up -d --build
```

## Development

For development setup and contribution guidelines, see individual component READMEs:
- [Backend Development](./backend/README.md)
- [Frontend Development](./frontend/README.md)
- [Runner Development](./runner/README.md)

## Support

For issues, questions, or contributions:
- Check existing issues in the repository
- Check component-specific README files

## License

See [LICENSE](./LICENSE) file for details.


_This project was fully deployed using Cloude Sonnet 4.5_