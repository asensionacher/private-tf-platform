# Private Terraform/OpenTofu Registry

Self-hosted private registry for Terraform and OpenTofu modules and providers, with an HTTP state backend and a web UI.

## Services

Three subdomains are derived from `REGISTRY_HOST` in `.env` (default: `registry.lan`):

| Subdomain | Purpose |
|---|---|
| `registry.lan` | Web UI |
| `api.registry.lan` | REST API + Terraform/OpenTofu registry protocol |
| `tf.registry.lan` | Terraform/OpenTofu HTTP state backend |

All three are served over **HTTPS on port 443** via nginx.

## Quick start

### 1. Configure

```bash
cp .env.example .env
```

Edit `.env` and set at minimum:

```bash
ENCRYPTION_KEY=<openssl rand -base64 32>
POSTGRES_PASSWORD=<openssl rand -base64 24>
REGISTRY_HOST=registry.lan
```

### 2. Generate TLS certificates

```bash
bash scripts/gen-certs.sh
```

This creates a self-signed CA and a server certificate covering all three subdomains, saved to `nginx/certs/`. The certs are gitignored — regenerate them on each new clone.

Trust the CA so browsers and CLI tools accept it:

```bash
# Arch / CachyOS
sudo trust anchor --store nginx/certs/ca.crt && sudo update-ca-trust

# Debian / Ubuntu
sudo cp nginx/certs/ca.crt /usr/local/share/ca-certificates/iac-registry-ca.crt
sudo update-ca-certificates
```

### 3. Add hostnames to `/etc/hosts`

```
<server-ip>  registry.lan  api.registry.lan  tf.registry.lan
```

### 4. Start

```bash
docker compose up -d --build
```

Open `https://registry.lan`.

For a complete walkthrough — including setting up a private provider, a private module, and remote state — see **[HOWTO.md](./HOWTO.md)**.

---

## Terraform / OpenTofu CLI setup

### Why HTTPS is required for private namespaces

Terraform and OpenTofu only send `Authorization` headers over HTTPS. If the registry endpoint is plain HTTP, the CLI silently omits credentials and every request to a private namespace fails with a 401.

Public namespaces (`is_public = true`) work over HTTP. Private namespaces require HTTPS — that is the only reason TLS is set up here.

### Configure the CLI

Add to `~/.terraformrc` (Terraform) or `~/.tofurc` (OpenTofu):

```hcl
host "api.registry.lan" {
  services = {
    "providers.v1" = "https://api.registry.lan/v1/providers/"
    "modules.v1"   = "https://api.registry.lan/v1/modules/"
  }
}

credentials "api.registry.lan" {
  token = "<api-key-from-the-web-ui>"
}
```

Generate an API key in the Web UI under **API Keys**.

### Use a private provider in a module

```hcl
terraform {
  required_providers {
    myprovider = {
      source  = "api.registry.lan/my-org/myprovider"
      version = "~> 1.0"
    }
  }
}
```

---

## HTTP state backend

Store remote Terraform/OpenTofu state without S3 or Consul.

```hcl
terraform {
  backend "http" {
    address        = "https://tf.registry.lan/api/tfstate/<deployment-id>"
    lock_address   = "https://tf.registry.lan/api/tfstate/<deployment-id>"
    unlock_address = "https://tf.registry.lan/api/tfstate/<deployment-id>"
    lock_method    = "LOCK"
    unlock_method  = "UNLOCK"
  }
}
```

Use the same URL for all three fields. OpenTofu/Terraform appends `:<workspace>` automatically for named workspaces.

State files live in the `registry-data` Docker volume at `/app/data/tfstates/<deployment-id>/`. The Web UI's **TF State** page lets you browse, inspect, and manage them.

---

## Common commands

```bash
# Start
docker compose up -d --build

# Logs
docker compose logs -f backend

# Stop (keeps data)
docker compose down

# Stop and delete all data
docker compose down -v
```

---

## Configuration reference

| Variable | Required | Description |
|---|---|---|
| `ENCRYPTION_KEY` | Yes | AES-256-GCM key for stored credentials. Generate with `openssl rand -base64 32`. |
| `POSTGRES_PASSWORD` | Yes | PostgreSQL password. |
| `REGISTRY_HOST` | No (default `registry.lan`) | Base domain for all three subdomains. |

See `.env.example` for all variables.

---

## Backup

Back up the `registry-data` Docker volume — it holds all provider binaries and Terraform state files.
