# Migración a Arquitectura con Runner Separado

## Resumen de Cambios

Se ha refactorizado la plataforma para utilizar un ejecutable runner separado que maneja la ejecución de comandos Terraform/OpenTofu, en lugar de ejecutarlos directamente desde el backend.

## ¿Qué ha cambiado?

### Antes
```
Backend ──> Ejecuta terraform/tofu directamente
```

### Ahora
```
Backend ──> Runner ──> Ejecuta terraform/tofu
         (JSON)     (stdin/stdout)
```

## Nuevos Componentes

### Runner (`/runner`)
- **Ubicación**: Directorio `runner/` en la raíz del proyecto
- **Propósito**: Ejecutable standalone que recibe comandos por stdin y devuelve resultados por stdout
- **Lenguaje**: Go
- **Compilación**: Se compila junto con el backend en el Dockerfile

### Archivos Modificados

1. **backend/internal/build/terraform.go**
   - Eliminadas las dependencias de `pty` y ejecución directa
   - Agregado protocolo de comunicación con el runner
   - Nueva función `executeCommand()` que llama al runner via stdin/stdout

2. **docker-compose.yml**
   - Actualizado para usar `Dockerfile.combined`
   - Compila backend y runner en una sola imagen

3. **Dockerfile.combined** (nuevo)
   - Multi-stage build que compila backend y runner
   - Instala Terraform y OpenTofu
   - Incluye ambos binarios en la imagen final

## Ventajas

### Seguridad
- Mejor aislamiento entre la API y la ejecución de comandos
- El runner puede ejecutarse con permisos restringidos
- Posibilidad de ejecutar el runner en un sandbox o contenedor separado

### Escalabilidad
- Los runners pueden distribuirse en múltiples máquinas
- Fácil implementar un pool de runners
- Backend puede delegar trabajo sin bloquear

### Mantenibilidad
- Código más modular y fácil de probar
- Cambios en la lógica de ejecución no afectan al backend
- Fácil agregar soporte para nuevas herramientas IaC

### Flexibilidad
- Posibilidad de tener diferentes versiones de Terraform/OpenTofu
- Runners especializados por tipo de tarea
- Configuración independiente del runtime

## Migración

### Para Desarrollo Local

1. **Compilar el runner**:
   ```bash
   cd runner
   ./build.sh
   ```

2. **Opción 1: Copiar el runner al PATH del sistema**:
   ```bash
   sudo cp iac-runner /usr/local/bin/
   ```

3. **Opción 2: El backend buscará automáticamente en**:
   - `/usr/local/bin/iac-runner` (producción)
   - `./runner/iac-runner` (desarrollo)

### Para Docker

Simplemente reconstruir la imagen:
```bash
docker compose build
docker compose up -d
```

El `Dockerfile.combined` compilará automáticamente ambos componentes.

### Para Producción

1. **Generar encryption key** (si no lo has hecho):
   ```bash
   ./generate-encryption-key.sh
   ```

2. **Actualizar docker-compose.yml** con tu configuración

3. **Rebuild y reiniciar**:
   ```bash
   docker compose down
   docker compose build
   docker compose up -d
   ```

## Protocolo de Comunicación

### Request (stdin → runner)
```json
{
  "tool": "terraform",
  "work_dir": "/tmp/iac-deployments/run-123",
  "command": "plan",
  "args": ["-out=tfplan"],
  "env_vars": {
    "TF_VAR_region": "us-east-1"
  },
  "timeout": 30
}
```

### Response (runner → stdout)
```json
{
  "success": true,
  "output": "Terraform will perform the following actions...",
  "exit_code": 0,
  "error_msg": "",
  "started_at": "2025-12-16T10:00:00Z",
  "ended_at": "2025-12-16T10:05:00Z"
}
```

## Testing

### Probar el runner standalone
```bash
cd runner
echo '{
  "tool": "terraform",
  "work_dir": "/tmp",
  "command": "version",
  "args": [],
  "env_vars": {},
  "timeout": 5
}' | ./iac-runner
```

### Resultado esperado
```json
{
  "success": true,
  "output": "Terraform v1.14.2...",
  "exit_code": 0,
  "started_at": "2025-12-16T...",
  "ended_at": "2025-12-16T..."
}
```

## Troubleshooting

### Error: "runner execution failed"
- **Causa**: El binario del runner no se encuentra
- **Solución**: Verificar que `/usr/local/bin/iac-runner` existe o copiar el binario

### Error: "Failed to parse runner response"
- **Causa**: El runner está devolviendo un formato incorrecto
- **Solución**: Verificar la versión del runner y recompilar

### Deployments no se ejecutan
- **Causa**: El runner no tiene permisos para ejecutar terraform/tofu
- **Solución**: Verificar que terraform/tofu están instalados y en el PATH

## Mejoras Futuras

- [ ] Pool de runners distribuido
- [ ] Métricas y monitoreo de runners
- [ ] Queue de jobs con sistema de prioridades
- [ ] Runners especializados por proveedor (AWS, Azure, GCP)
- [ ] Soporte para runners remotos vía API
- [ ] Logs streaming en tiempo real
- [ ] Cancelación de ejecuciones en progreso

## Soporte

Para más información sobre el runner:
- Ver [runner/README.md](runner/README.md)
- Revisar el código en [runner/main.go](runner/main.go)
- Contactar al equipo de desarrollo
