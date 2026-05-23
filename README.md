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

### 3. Trust the CA certificate

The CA must be trusted on every machine that will access the registry — both the server and any machine running Terraform or OpenTofu. Copy `nginx/certs/ca.crt` to each machine and run the appropriate command:

**Arch / CachyOS**
```bash
sudo trust anchor --store ca.crt && sudo update-ca-trust
```

**Debian / Ubuntu**
```bash
sudo cp ca.crt /usr/local/share/ca-certificates/iac-registry-ca.crt
sudo update-ca-certificates
```

**RHEL / Fedora / CentOS**
```bash
sudo cp ca.crt /etc/pki/ca-trust/source/anchors/iac-registry-ca.crt
sudo update-ca-trust extract
```

**macOS**
```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ca.crt
```

**Windows** (run in an elevated PowerShell)
```powershell
Import-Certificate -FilePath ca.crt -CertStoreLocation Cert:\LocalMachine\Root
```

### 4. Add hostnames to `/etc/hosts`

```
<server-ip>  registry.lan  api.registry.lan  tf.registry.lan
```

### 5. Start

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

The state backend uses **HTTP Basic Auth** — the same username and password you use to log into the web UI. Any user (admin or reader) can access the state backend.

> **Never hardcode credentials in `.tf` files.** Use environment variables instead — Terraform and OpenTofu read `TF_HTTP_USERNAME` and `TF_HTTP_PASSWORD` automatically.

```bash
export TF_HTTP_USERNAME="myuser"
export TF_HTTP_PASSWORD="mypassword"
```

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
