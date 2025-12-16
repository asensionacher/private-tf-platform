# Resumen de Cambios: Arquitectura con Runner Separado

## 📋 Cambios Implementados

### ✅ Nuevos Archivos Creados

#### Runner (nuevo componente)
- [runner/main.go](runner/main.go) - Ejecutable principal del runner
- [runner/go.mod](runner/go.mod) - Dependencias del módulo Go
- [runner/Dockerfile](runner/Dockerfile) - Dockerfile standalone para el runner
- [runner/README.md](runner/README.md) - Documentación del runner
- [runner/build.sh](runner/build.sh) - Script de compilación local
- [runner/test.sh](runner/test.sh) - Script de pruebas
- [runner/.gitignore](runner/.gitignore) - Ignora binarios compilados
- [runner/config.example](runner/config.example) - Ejemplo de configuración

#### Documentación
- [MIGRATION.md](MIGRATION.md) - Guía de migración
- [ARCHITECTURE.md](ARCHITECTURE.md) - Documentación de arquitectura detallada

#### Build
- [Dockerfile.combined](Dockerfile.combined) - Multi-stage build para backend + runner

### 🔧 Archivos Modificados

1. **backend/internal/build/terraform.go**
   - ❌ Eliminada ejecución directa de terraform/tofu
   - ❌ Removidas dependencias de `pty`, `context`, `bufio`
   - ✅ Agregado protocolo JSON para comunicación con runner
   - ✅ Nueva función `executeCommand()` que llama al runner
   - ✅ Agregados tipos `RunnerRequest` y `RunnerResponse`

2. **docker-compose.yml**
   - Cambiado `context` de `./backend` a `.` (raíz)
   - Cambiado `dockerfile` a `Dockerfile.combined`

3. **README.md**
   - Actualizado diagrama de arquitectura
   - Agregada sección sobre el runner
   - Documentadas ventajas de la nueva arquitectura

## 🎯 Funcionalidades

### El Runner Proporciona
- ✅ Ejecución aislada de comandos Terraform/OpenTofu
- ✅ Comunicación via JSON (stdin/stdout)
- ✅ Manejo de timeouts configurables
- ✅ Preservación de colores ANSI en output
- ✅ Manejo de variables de entorno
- ✅ Códigos de salida y mensajes de error detallados

### El Backend Ahora
- ✅ Delega ejecución al runner
- ✅ Se enfoca en lógica de negocio
- ✅ Mantiene la misma API REST
- ✅ Compatible con código frontend existente
- ✅ Mejor separación de responsabilidades

## 🚀 Cómo Usar

### Desarrollo Local

```bash
# Compilar el runner
cd runner
./build.sh

# Probar el runner
./test.sh

# Compilar el backend
cd ../backend
go build -o iac-tool main.go

# Ejecutar (el runner debe estar en PATH o en ./runner/)
./iac-tool
```

### Docker

```bash
# Construir ambos componentes
docker compose build

# Iniciar servicios
docker compose up -d

# Ver logs
docker compose logs -f backend
```

## 📊 Comparación Antes/Después

### Antes
```go
// backend ejecutaba directamente:
cmd := exec.Command("terraform", "plan")
cmd.Dir = workDir
// ... configuración ...
output, err := cmd.CombinedOutput()
```

### Después
```go
// backend crea request:
request := RunnerRequest{
    Tool: "terraform",
    WorkDir: workDir,
    Command: "plan",
    Args: []string{"-out=tfplan"},
}

// ejecuta runner:
cmd := exec.Command("/usr/local/bin/iac-runner")
cmd.Stdin = requestJSON
response := parseResponse(cmd.Output())
```

## 🔐 Seguridad

### Ventajas de Seguridad
- ✅ Backend no ejecuta comandos arbitrarios del sistema
- ✅ Runner puede ejecutarse con usuario no-root
- ✅ Fácil implementar sandbox o contenedor separado
- ✅ Mejor auditoría y logging de comandos ejecutados
- ✅ Aislamiento de credenciales y secretos

## 📈 Escalabilidad Futura

### Posibles Mejoras
- [ ] Pool de runners distribuido
- [ ] Queue de jobs con Redis/RabbitMQ
- [ ] Runners remotos via gRPC o HTTP
- [ ] Auto-scaling de runners
- [ ] Métricas y monitoreo (Prometheus)
- [ ] Cancelación de jobs en progreso
- [ ] Streaming de logs en tiempo real

## 🧪 Testing

### Tests Automatizados Posibles
```bash
# Test unitario del runner
cd runner
go test ./...

# Test de integración
./test.sh

# Test del backend
cd ../backend
go test ./internal/build/...
```

## 📦 Despliegue

### Producción
1. Usar `Dockerfile.combined` para build completo
2. Asegurar que Terraform/OpenTofu están instalados en el contenedor
3. Configurar límites de recursos (CPU, memoria)
4. Configurar timeouts apropiados
5. Monitorear ejecuciones de runners

### Kubernetes (ejemplo)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: iac-backend
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: backend
        image: iac-platform:latest
        resources:
          limits:
            cpu: "2"
            memory: "4Gi"
```

## 🐛 Troubleshooting

### Runner no encontrado
```bash
# Verificar ubicación
which iac-runner

# O copiar manualmente
sudo cp runner/iac-runner /usr/local/bin/
```

### Timeouts en deployments
```go
// Ajustar timeout en terraform.go:
request.Timeout = 60 // 60 minutos
```

### Logs del runner
Los logs del runner aparecen en los deployment_runs:
- `init_log`
- `plan_log`
- `apply_log`

## ✨ Conclusión

La nueva arquitectura proporciona:
- 🎯 Mejor separación de responsabilidades
- 🔒 Mayor seguridad y aislamiento
- 📈 Mejor escalabilidad
- 🛠️ Más fácil de mantener y extender
- 🚀 Base sólida para futuras mejoras

Todos los cambios son retrocompatibles con el frontend existente y no requieren cambios en la API REST.
