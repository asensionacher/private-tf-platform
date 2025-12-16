# IAC Runner

El runner es un ejecutable separado que se encarga de ejecutar comandos de Terraform y OpenTofu. El backend se comunica con el runner a través de stdin/stdout usando JSON.

## Arquitectura

El backend ya no ejecuta Terraform/OpenTofu directamente. En su lugar:
1. El backend crea una solicitud JSON con los detalles del comando a ejecutar
2. Llama al ejecutable `iac-runner` pasando la solicitud por stdin
3. El runner ejecuta el comando de Terraform/OpenTofu
4. El runner devuelve la respuesta en formato JSON por stdout

## Ventajas

- **Aislamiento**: El runner puede ejecutarse con diferentes permisos o en un contenedor separado
- **Seguridad**: Mejor aislamiento entre la API del backend y la ejecución de comandos
- **Escalabilidad**: Los runners pueden distribuirse y escalarse independientemente
- **Flexibilidad**: Fácil agregar nuevas herramientas IaC sin modificar el backend

## Protocolo de Comunicación

### Request (stdin)
```json
{
  "tool": "terraform",
  "work_dir": "/tmp/iac-deployments/run-123",
  "command": "plan",
  "args": ["-out=tfplan"],
  "env_vars": {
    "TF_VAR_example": "value"
  },
  "timeout": 30
}
```

### Response (stdout)
```json
{
  "success": true,
  "output": "...(output con colores ANSI)...",
  "exit_code": 0,
  "error_msg": "",
  "started_at": "2025-12-16T10:00:00Z",
  "ended_at": "2025-12-16T10:05:00Z"
}
```

## Compilación

### Local
```bash
cd runner
go build -o iac-runner main.go
```

### Docker
El runner se compila automáticamente como parte del build del backend usando el `Dockerfile.combined` desde la raíz del proyecto.

## Uso

### Standalone
```bash
echo '{"tool":"terraform","work_dir":"/path","command":"version","args":[],"env_vars":{},"timeout":5}' | ./iac-runner
```

### Desde el Backend
El backend automáticamente llama al runner cuando se ejecuta un deployment. El runner debe estar disponible en `/usr/local/bin/iac-runner`.

## Desarrollo

Para probar el runner localmente:

```bash
# Compilar
cd runner
go build -o iac-runner main.go

# Probar
echo '{
  "tool": "terraform",
  "work_dir": "/tmp",
  "command": "version",
  "args": [],
  "env_vars": {},
  "timeout": 5
}' | ./iac-runner
```
