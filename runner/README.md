# Runner - Terraform/OpenTofu Execution Service

Lightweight Go service that executes Terraform and OpenTofu deployments with live log streaming, approval workflows, and multi-cloud support.

## Overview

The runner service provides:
- **Terraform/OpenTofu Execution** - Run `terraform` or `tofu` commands in isolated environments
- **Live Log Streaming** - Server-Sent Events (SSE) for real-time log output
- **Approval Workflows** - Manual approval gates before `terraform apply`
- **Git Integration** - Clone repositories with HTTPS authentication
- **Multi-Cloud Support** - Pre-installed AWS CLI and Azure CLI
- **Cancellation** - Stop running deployments at any time
- **Private Registry Integration** - Automatic configuration for private module/provider registries

## Architecture

### Technology Stack
- **Framework**: Gin (HTTP web framework)
- **Language**: Go 1.23
- **IaC Tools**: Terraform 1.14.2, OpenTofu 1.11.1
- **Cloud CLIs**: AWS CLI v2, Azure CLI
- **Key Libraries**:
  - `github.com/gin-gonic/gin` - HTTP server
  - `github.com/creack/pty` - PTY for colored Terraform output
  - `github.com/google/uuid` - Deployment ID generation

### Deployment Workflow

```
1. POST /deploy
   ├─> Clone Git repository
   ├─> Run terraform init
   ├─> Run terraform plan -out=tfplan
   ├─> (Optional) Wait for manual approval
   ├─> Run terraform apply tfplan
   └─> Return outputs

2. Live Monitoring
   ├─> GET /deploy/:id/status  (poll status)
   └─> GET /deploy/:id/logs    (stream logs via SSE)

3. Manual Controls
   ├─> POST /deploy/:id/approve (continue with apply)
   ├─> POST /deploy/:id/reject  (cancel deployment)
   └─> POST /deploy/:id/cancel  (stop at any phase)
```

### Phases

1. **Initializing** - Setting up working directory
2. **Cloning** - Cloning Git repository
3. **Init** - Running `terraform init`
4. **Plan** - Running `terraform plan`
5. **Awaiting Approval** - Waiting for user approval (if not auto-approved)
6. **Apply** - Running `terraform apply`
7. **Completed** - Deployment finished (success/failed/cancelled)

## Setup and Installation

### Prerequisites

- **Go 1.23+** - [Install Go](https://golang.org/doc/install)
- **Git** - Required for cloning repositories
- **Terraform** (optional for local testing) - [Install Terraform](https://www.terraform.io/downloads)
- **OpenTofu** (optional for local testing) - [Install OpenTofu](https://opentofu.org/docs/intro/install)

### Manual Setup (without Docker)

1. **Navigate to runner directory**
   ```bash
   cd runner
   ```

2. **Install Go dependencies**
   ```bash
   go mod download
   ```

3. **Install Terraform and OpenTofu**
   ```bash
   # Terraform
   wget https://releases.hashicorp.com/terraform/1.14.2/terraform_1.14.2_linux_amd64.zip
   unzip terraform_1.14.2_linux_amd64.zip
   sudo mv terraform /usr/local/bin/
   terraform version
   
   # OpenTofu
   wget https://github.com/opentofu/opentofu/releases/download/v1.11.1/tofu_1.11.1_linux_amd64.zip
   unzip tofu_1.11.1_linux_amd64.zip
   sudo mv tofu /usr/local/bin/
   tofu version
   ```

4. **Install cloud CLIs (optional)**
   ```bash
   # AWS CLI
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install
   aws --version
   
   # Azure CLI
   curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
   az --version
   ```

5. **Configure environment variables**
   ```bash
   # Required for private registry integration
   export REGISTRY_HOST=http://localhost:9080
   
   # Optional CORS configuration
   export ALLOWED_ORIGINS=*
   ```

6. **Build the runner**
   ```bash
   go build -o iac-runner main.go
   ```

7. **Run the runner**
   ```bash
   ./iac-runner
   ```

   You should see:
   ```
   Runner HTTP server starting on :8080
   ```

### Docker Setup

Using Docker Compose (recommended):

```bash
# From project root
docker-compose up -d runner

# View logs
docker-compose logs -f runner

# Rebuild after changes
docker-compose up -d --build runner
```

Using standalone Docker:

```bash
# Build
docker build -t iac-runner .

# Run
docker run -d \
  -p 8080:8080 \
  -e REGISTRY_HOST=http://host.docker.internal:9080 \
  --name iac-runner \
  iac-runner
```

The Dockerfile uses a multi-stage build:
1. **Stage 1**: Build Go binary
2. **Stage 2**: Install Terraform, OpenTofu, cloud CLIs, and copy binary

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REGISTRY_HOST` | _(none)_ | Private registry URL (e.g., `http://localhost:9080`) |
| `ALLOWED_ORIGINS` | `*` | CORS allowed origins (comma-separated) |

### Cloud Provider Authentication

The runner supports cloud provider authentication via environment variables:

**AWS**:
```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_DEFAULT_REGION=us-east-1
```

**Azure**:
```bash
export ARM_CLIENT_ID=your-client-id
export ARM_CLIENT_SECRET=your-client-secret
export ARM_SUBSCRIPTION_ID=your-subscription-id
export ARM_TENANT_ID=your-tenant-id
```

Alternatively, pass credentials as environment variables in the deployment request (see API Usage below).

### Private Registry Integration

If `REGISTRY_HOST` is set, the runner automatically:
1. Fetches authentication token from backend (`/api/internal/registry-token`)
2. Creates `.terraformrc` in working directory
3. Configures credentials and service discovery
4. Sets `TF_CLI_CONFIG_FILE` environment variable

Example `.terraformrc` generated:
```hcl
credentials "localhost:9080" {
  token = "internal-registry-token"
}

host "localhost:9080" {
  services = {
    "modules.v1"   = "http://localhost:9080/v1/modules/",
    "providers.v1" = "http://localhost:9080/v1/providers/"
  }
}
```

## API Endpoints

### Health Check
```
GET /health
```

Response:
```json
{
  "status": "healthy"
}
```

### Start Deployment
```
POST /deploy
```

Request body:
```json
{
  "tool": "terraform",
  "git_url": "https://github.com/user/repo.git",
  "git_ref": "main",
  "path": "examples/basic",
  "env_vars": {
    "AWS_ACCESS_KEY_ID": "...",
    "AWS_SECRET_ACCESS_KEY": "..."
  },
  "tfvars_files": ["prod.tfvars"],
  "init_flags": "-upgrade",
  "plan_flags": "-compact-warnings",
  "timeout": 60,
  "auto_approve": false,
  "git_auth": {
    "type": "http",
    "username": "token",
    "password": "ghp_xxxxxxxxxxxx"
  }
}
```

Request fields:
- `tool` (required): `"terraform"` or `"tofu"`
- `git_url` (required): Git repository HTTPS URL
- `git_ref` (required): Branch, tag, or commit SHA
- `path` (optional): Working directory within repo (default: `.`)
- `env_vars` (optional): Environment variables for Terraform execution
- `tfvars_files` (optional): Array of `.tfvars` file paths
- `init_flags` (optional): Custom flags for `terraform init`
- `plan_flags` (optional): Custom flags for `terraform plan`
- `timeout` (optional): Timeout in minutes (default: 60)
- `auto_approve` (optional): Skip manual approval (default: false)
- `git_auth` (optional): Git credentials for private repositories

Response (202 Accepted):
```json
{
  "deployment_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "message": "Deployment started"
}
```

### Get Deployment Status
```
GET /deploy/:id/status
```

Response:
```json
{
  "deployment_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "phase": "plan",
  "started_at": "2024-01-15T10:30:00Z",
  "ended_at": null,
  "error": "",
  "init_log": "Initializing...\n...",
  "plan_log": "Planning...\n...",
  "plan_output": "",
  "apply_log": "",
  "apply_output": ""
}
```

Status values:
- `running` - Deployment in progress
- `awaiting_approval` - Waiting for manual approval
- `success` - Deployment completed successfully
- `failed` - Deployment failed
- `cancelled` - Deployment cancelled by user

Phase values:
- `initializing` - Setting up environment
- `cloning` - Cloning Git repository
- `init` - Running `terraform init`
- `plan` - Running `terraform plan`
- `apply` - Running `terraform apply`
- `completed` - Deployment finished

### Stream Deployment Logs
```
GET /deploy/:id/logs
```

Returns Server-Sent Events (SSE) stream:
```
event: log
data: Cloning repository...

event: log
data: Initializing Terraform...

event: log
data: Plan: 5 to add, 0 to change, 0 to destroy.
```

**Frontend usage example:**
```javascript
const eventSource = new EventSource(`http://localhost:8080/deploy/${id}/logs`);
eventSource.addEventListener('log', (event) => {
  console.log(event.data);
});
```

### Approve Deployment
```
POST /deploy/:id/approve
```

Response:
```json
{
  "message": "Deployment approved"
}
```

### Reject Deployment
```
POST /deploy/:id/reject
```

Response:
```json
{
  "message": "Deployment rejected"
}
```

### Cancel Deployment
```
POST /deploy/:id/cancel
```

Response:
```json
{
  "message": "Cancellation requested"
}
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...
```

Currently, the runner does not have unit tests. Consider adding:
- Mock Git cloning
- Mock Terraform command execution
- Test deployment lifecycle
- Test approval workflows

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use descriptive variable names
- Check errors; never ignore `error` return values
- Add comments for exported functions and complex logic
- Use context for cancellation and timeouts

### Adding New Features

1. **Update `DeploymentRequest` struct** if adding new parameters
2. **Modify `executeDeployment()` function** for new workflow steps
3. **Update API handlers** if adding new endpoints
4. **Test with real Terraform/OpenTofu modules**

Example: Adding destroy support:
```go
// Add to DeploymentRequest
Destroy bool `json:"destroy"`

// Add to executeDeployment
if deployment.Request.Destroy {
    deployment.updateStatus("running", "destroy", "")
    deployment.log("Running terraform destroy...")
    destroyLog, err := runTerraformCommand(deployment, deployPath, "destroy", []string{"-auto-approve"})
    // Handle error...
}
```

## Deployment Execution Details

### Working Directory

Each deployment gets an isolated working directory:
```
/tmp/iac-deployments/<deployment-id>/
```

Cleanup:
- Directories are automatically deleted after 24 hours
- Use `docker volume prune` to clean up Docker volumes

### Terraform Configuration

The runner automatically:
1. Clones Git repository to working directory
2. Navigates to specified `path`
3. Creates `.terraformrc` for private registry (if `REGISTRY_HOST` set)
4. Sets `TF_CLI_CONFIG_FILE` environment variable
5. Runs Terraform commands with colored output (via PTY)

### Environment Variables in Terraform

Environment variables passed in `env_vars` are available to Terraform:
```json
{
  "env_vars": {
    "TF_VAR_region": "us-east-1",
    "TF_VAR_instance_type": "t3.micro",
    "AWS_ACCESS_KEY_ID": "...",
    "AWS_SECRET_ACCESS_KEY": "..."
  }
}
```

### Custom Flags

**Init flags** (example):
```json
{
  "init_flags": "-upgrade -backend-config=bucket=my-tfstate"
}
```

**Plan flags** (example):
```json
{
  "plan_flags": "-compact-warnings -parallelism=10"
}
```

The runner uses a shell-aware parser (`parseShellArgs()`) that respects quotes and escapes.

### Approval Workflow

If `auto_approve` is `false`:
1. Deployment runs `terraform plan`
2. Status changes to `awaiting_approval`
3. Frontend polls `/deploy/:id/status` to detect state
4. User clicks "Approve" or "Reject"
5. Frontend sends `POST /deploy/:id/approve` or `POST /deploy/:id/reject`
6. Runner continues with `terraform apply` or cancels

Timeout: 24 hours (deployments auto-cancel if not approved)

## Troubleshooting

### Runner Not Starting

```bash
# Check port 8080 is available
lsof -i :8080

# Check logs
docker-compose logs runner
```

### Terraform/OpenTofu Not Found

```bash
# Verify installation
docker exec -it iac-runner terraform version
docker exec -it iac-runner tofu version
```

### Git Clone Failures

Common issues:
- Invalid credentials (check `git_auth` in request)
- Private repository without authentication
- Invalid `git_ref` (branch/tag doesn't exist)

Check deployment logs:
```bash
curl http://localhost:8080/deploy/<id>/status | jq '.init_log'
```

### Private Registry Connection Issues

```bash
# Verify REGISTRY_HOST is set
docker exec -it iac-runner env | grep REGISTRY_HOST

# Check backend is accessible
docker exec -it iac-runner curl http://host.docker.internal:9080/api/internal/registry-token
```

### Timeout Errors

Default timeout is 60 minutes. For long-running deployments:
```json
{
  "timeout": 120
}
```

### Cloud Provider Authentication

```bash
# AWS - test credentials
docker exec -it iac-runner aws sts get-caller-identity

# Azure - test credentials
docker exec -it iac-runner az account show
```

## Production Deployment

### Security Checklist

- [ ] Set `ALLOWED_ORIGINS` to specific domains (no `*` wildcard)
- [ ] Use HTTPS for `REGISTRY_HOST`
- [ ] Run runner in isolated network (not publicly accessible)
- [ ] Use IAM roles instead of long-lived credentials (AWS)
- [ ] Use managed identities instead of service principals (Azure)
- [ ] Enable audit logging for all deployments
- [ ] Set resource limits (CPU/memory) in Docker
- [ ] Implement API authentication (currently open)
- [ ] Use secrets management (Vault, AWS Secrets Manager, etc.)
- [ ] Enable log retention and monitoring

### Resource Limits

Example `docker-compose.yml`:
```yaml
runner:
  image: iac-runner
  deploy:
    resources:
      limits:
        cpus: '2'
        memory: 2G
      reservations:
        cpus: '1'
        memory: 1G
```

### CORS Configuration

For production, set specific origins:
```bash
export ALLOWED_ORIGINS=https://iac.company.com,https://iac-staging.company.com
```

### Monitoring

Key metrics to monitor:
- Active deployments count
- Deployment success/failure rate
- Average deployment duration
- Resource usage (CPU, memory, disk)
- API response times

Example Prometheus metrics (not yet implemented):
- `iac_runner_deployments_total{status="success|failed|cancelled"}`
- `iac_runner_deployment_duration_seconds`
- `iac_runner_active_deployments`

## Limitations

### Current Limitations
- **No SSH support** for Git (HTTPS only)
- **No state locking** coordination (use Terraform backend locking)
- **No deployment queue** (unlimited concurrent deployments)
- **No API authentication** (open endpoints)
- **No deployment history** beyond active deployments
- **No custom Terraform versions** (pinned to 1.14.2)
- **No OpenTofu version selection** (pinned to 1.11.1)
- **No log persistence** (logs deleted after 24 hours)

### Future Enhancements
- Add SSH key support for Git
- Implement deployment queue with concurrency limits
- Add API key authentication
- Support custom Terraform/OpenTofu versions
- Persist deployment logs to object storage (S3, Azure Blob)
- Add webhooks for deployment events
- Implement policy-as-code validation (OPA, Sentinel)
- Add cost estimation (Infracost integration)

## Related Documentation

- [Root README](../README.md) - Full project overview and quick start
- [Backend README](../backend/README.md) - Backend API documentation
- [Frontend README](../frontend/README.md) - Frontend development guide
- [Terraform Registry Protocol](https://developer.hashicorp.com/terraform/registry/api-docs) - Official protocol documentation
- [OpenTofu Documentation](https://opentofu.org/docs/) - OpenTofu reference

## License

See [../LICENSE](../LICENSE) for details.
