# Arquitectura del Sistema con Runner Separado

## Flujo de Deployment

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Usuario / Frontend                          │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
                                 │ HTTP REST API
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                           Backend (Go)                               │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  API Layer (gin-gonic)                                        │  │
│  │  - Deployments, Modules, Providers, Namespaces               │  │
│  └────────────────────────┬─────────────────────────────────────┘  │
│                           │                                          │
│  ┌────────────────────────▼─────────────────────────────────────┐  │
│  │  Business Logic                                               │  │
│  │  - Git Sync, GPG Signing, Build Management                   │  │
│  └────────────────────────┬─────────────────────────────────────┘  │
│                           │                                          │
│  ┌────────────────────────▼─────────────────────────────────────┐  │
│  │  Database (SQLite)                                            │  │
│  │  - Modules, Providers, Deployments, API Keys                 │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  Deployment Execution (terraform.go)                          │  │
│  │  1. Clone Git repo                                            │  │
│  │  2. Prepare working directory                                 │  │
│  │  3. Create RunnerRequest (JSON)                               │  │
│  │  4. Execute runner via stdin/stdout                           │  │
│  │  5. Parse RunnerResponse                                      │  │
│  │  6. Update deployment status                                  │  │
│  └────────────────────────┬─────────────────────────────────────┘  │
└─────────────────────────────┼────────────────────────────────────────┘
                              │
                              │ JSON via stdin/stdout
                              │ RunnerRequest {
                              │   tool, work_dir, command, args,
                              │   env_vars, timeout
                              │ }
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Runner (iac-runner)                           │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  1. Parse JSON request from stdin                             │  │
│  └────────────────────────┬─────────────────────────────────────┘  │
│                           │                                          │
│  ┌────────────────────────▼─────────────────────────────────────┐  │
│  │  2. Validate request                                          │  │
│  │     - Check tool (terraform/tofu)                             │  │
│  │     - Validate work_dir exists                                │  │
│  │     - Set timeout defaults                                    │  │
│  └────────────────────────┬─────────────────────────────────────┘  │
│                           │                                          │
│  ┌────────────────────────▼─────────────────────────────────────┐  │
│  │  3. Execute command                                           │  │
│  │     - Create exec.Command with context                        │  │
│  │     - Set working directory                                   │  │
│  │     - Apply environment variables                             │  │
│  │     - Use PTY for colored output                              │  │
│  └────────────────────────┬─────────────────────────────────────┘  │
│                           │                                          │
│  ┌────────────────────────▼─────────────────────────────────────┐  │
│  │  4. Capture output and exit code                              │  │
│  └────────────────────────┬─────────────────────────────────────┘  │
│                           │                                          │
│  ┌────────────────────────▼─────────────────────────────────────┐  │
│  │  5. Return JSON response to stdout                            │  │
│  │     RunnerResponse {                                          │  │
│  │       success, output, exit_code,                             │  │
│  │       error_msg, started_at, ended_at                         │  │
│  │     }                                                          │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────┬────────────────────────────────────────┘
                              │
                              │ Execute actual IaC commands
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Terraform / OpenTofu                              │
│  - init                                                              │
│  - plan                                                              │
│  - apply                                                             │
│  - destroy                                                           │
└──────────────────────────────────────────────────────────────────────┘
```

## Ventajas de la Separación

### 1. Aislamiento y Seguridad
- El backend no ejecuta comandos del sistema directamente
- El runner puede ejecutarse con permisos limitados
- Fácil auditoría de comandos ejecutados

### 2. Escalabilidad
- Múltiples runners pueden procesar despliegues en paralelo
- Runners pueden ejecutarse en máquinas diferentes
- Pool de workers para alta disponibilidad

### 3. Flexibilidad
- Fácil soportar diferentes versiones de Terraform/OpenTofu
- Runners especializados por entorno (dev, staging, prod)
- Posibilidad de runners en diferentes clouds

### 4. Mantenibilidad
- Código modular y fácil de testear
- Actualizar Terraform/OpenTofu sin tocar el backend
- Fácil debugging y logging

## Protocolo de Comunicación

### Request Format
```json
{
  "tool": "terraform | tofu",
  "work_dir": "/path/to/working/directory",
  "command": "init | plan | apply | destroy",
  "args": ["additional", "arguments"],
  "env_vars": {
    "TF_VAR_name": "value",
    "AWS_REGION": "us-east-1"
  },
  "timeout": 30
}
```

### Response Format
```json
{
  "success": true,
  "output": "Command output with ANSI colors preserved",
  "exit_code": 0,
  "error_msg": "Optional error message",
  "started_at": "2025-12-16T10:00:00Z",
  "ended_at": "2025-12-16T10:05:00Z"
}
```

## Ejemplo de Flujo Completo

1. **Usuario crea deployment** → Frontend envía POST a backend
2. **Backend recibe deployment** → Valida y guarda en DB
3. **Backend ejecuta deployment run**:
   - Clona repositorio Git
   - Crea directorio de trabajo temporal
   - Marca status = 'initializing'
4. **Backend llama runner para init**:
   - Crea RunnerRequest con command="init"
   - Ejecuta `/usr/local/bin/iac-runner` con JSON en stdin
   - Captura RunnerResponse de stdout
   - Guarda init_log en DB
5. **Backend llama runner para plan**:
   - Marca status = 'planning'
   - Similar a init pero command="plan"
   - Guarda plan_log en DB
6. **Espera aprobación** → status = 'awaiting_approval'
7. **Usuario aprueba** → Frontend actualiza deployment run
8. **Backend llama runner para apply**:
   - Marca status = 'applying'
   - Ejecuta runner con command="apply"
   - Guarda apply_log en DB
9. **Deployment completado** → status = 'success'
10. **Limpieza programada** → Elimina directorio de trabajo después de 24h
