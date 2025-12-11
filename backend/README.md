# Backend - Terraform Private Registry API

API REST desarrollada en Go que implementa el protocolo oficial de Terraform Registry para módulos y providers.

## Tecnologías

- **Lenguaje**: Go 1.21+
- **Framework Web**: Gin
- **Base de datos**: SQLite (modernc.org/sqlite)
- **Firma GPG**: Automática para providers
- **Git**: Sincronización de versiones desde repositorios

## Estructura del Proyecto

```
backend/
├── main.go                    # Punto de entrada
├── internal/
│   ├── api/                   # Handlers HTTP
│   │   ├── discovery.go       # Service discovery de Terraform
│   │   ├── modules.go         # Endpoints de módulos
│   │   ├── providers.go       # Endpoints de providers
│   │   ├── namespaces.go      # Gestión de namespaces
│   │   └── utils.go           # Utilidades comunes
│   ├── database/
│   │   └── database.go        # Inicialización y migraciones SQLite
│   ├── models/
│   │   ├── module.go          # Modelos de módulos
│   │   ├── provider.go        # Modelos de providers
│   │   └── namespace.go       # Modelos de namespaces
│   ├── git/
│   │   └── git.go             # Operaciones Git (clone, tags)
│   ├── gpg/
│   │   └── gpg.go             # Firma GPG de binarios
│   └── build/
│       └── build.go           # Generación de archivos de descarga
├── Dockerfile
└── go.mod
```

## Endpoints Principales

### Service Discovery
- `GET /.well-known/terraform.json` - Descubrimiento de servicios Terraform

### Terraform Protocol (Módulos)
- `GET /v1/modules/:namespace/:name/:provider/versions` - Listar versiones
- `GET /v1/modules/:namespace/:name/:provider/:version/download` - Descargar módulo

### Terraform Protocol (Providers)
- `GET /v1/providers/:namespace/:name/versions` - Listar versiones
- `GET /v1/providers/:namespace/:name/:version/download/:os/:arch` - Descargar provider

### Management API
- `GET /api/modules` - Listar módulos
- `POST /api/modules` - Crear módulo desde Git
- `POST /api/modules/:id/sync-tags` - Sincronizar versiones desde Git
- `DELETE /api/modules/:id` - Eliminar módulo
- `GET /api/modules/:id/versions` - Listar versiones de módulo
- `PATCH /api/modules/:id/versions/:versionId` - Habilitar/deshabilitar versión

Similar para providers (`/api/providers/*`)

### Namespaces
- `GET /api/namespaces` - Listar namespaces
- `POST /api/namespaces` - Crear namespace
- `POST /api/namespaces/:id/api-keys` - Generar API key

## Variables de Entorno

| Variable | Descripción | Valor por defecto |
|----------|-------------|-------------------|
| `PORT` | Puerto HTTP | `9080` |
| `BASE_URL` | URL base del registry | `http://localhost:9080` |
| `DB_PATH` | Ruta de la base de datos SQLite | `./registry.db` |
| `GPG_HOME` | Directorio GPG para firmas | `/app/data/gpg` |
| `BUILD_DIR` | Directorio para binarios de providers | `/app/data/builds` |

## Desarrollo Local

### Requisitos

- Go 1.21 o superior
- Git instalado
- (Opcional) GPG para firma de providers

### Instalación

```bash
cd backend

# Descargar dependencias
go mod download

# Ejecutar en modo desarrollo
go run .
```

El servidor estará disponible en `http://localhost:9080`

### Compilación

```bash
# Compilar binario
go build -o registry-api

# Ejecutar
./registry-api
```

### Tests

```bash
# Ejecutar tests
go test ./...

# Con cobertura
go test -cover ./...

# Tests específicos
go test ./internal/git/...
```

## Base de Datos

SQLite con las siguientes tablas principales:

- **namespaces**: Organizaciones/equipos
- **api_keys**: Claves de autenticación por namespace
- **modules**: Módulos Terraform
- **module_versions**: Versiones de módulos
- **providers**: Providers Terraform
- **provider_versions**: Versiones de providers
- **provider_platforms**: Binarios por plataforma (OS/arch)

### Migraciones

Las migraciones se ejecutan automáticamente al iniciar:
- `ALTER TABLE` para agregar nuevas columnas
- Datos por defecto (namespace "default")

## Sincronización Git

### Flujo de sincronización de módulos

1. Usuario crea módulo con URL de Git
2. Backend ejecuta `syncModuleTagsBackground` en goroutine
3. Clona repositorio con `git clone --bare --filter=blob:none`
4. Extrae tags con `git for-each-ref refs/tags`
5. Filtra tags que parecen versiones (regex: `v?\d+\.\d+(\.\d+)?`)
6. Crea `module_versions` por cada tag
7. Genera URL de descarga: `git::https://...?ref=<tag>`

### Manejo de errores

- Si falla el clone → `sync_error` con mensaje de error
- Si no hay tags → `sync_error = "No valid version tags found"`
- Si hay panic → Recovery y `sync_error` con stack trace
- Siempre marca `synced = TRUE` para evitar reintentos infinitos

## Firma GPG (Providers)

### Inicialización

Al arrancar, el backend:
1. Verifica si existe clave GPG en `$GPG_HOME`
2. Si no existe, genera una clave RSA 4096 bits
3. Exporta clave pública para distribución

### Proceso de firma

1. Provider binario se sube o genera
2. Backend calcula SHA256 del binario
3. Crea archivo `SHASUMS` con todos los binarios de la versión
4. Firma archivo con GPG: `SHASUMS.sig`
5. Terraform verifica firma al descargar

## API Keys y Autenticación

- Operaciones de **lectura** son públicas
- Operaciones de **escritura** requieren API key
- Header: `X-API-Key: <key>`
- Keys vinculadas a namespaces
- Permisos: `read`, `write`, `admin`

### Middleware de autenticación

```go
// En modules.go, providers.go
func RequireAPIKey() gin.HandlerFunc {
    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        // Validar y verificar permisos
        // Establecer namespace en contexto
    }
}
```

## Construcción de URLs de Descarga

### Módulos

```
git::<git-url>?ref=<tag>[//<subdir>]
```

Terraform soporta nativamente clonado de Git con esta sintaxis.

### Providers

```
http://<base_url>/downloads/<namespace>/<name>/<version>/<os>_<arch>/<filename>
```

Binarios almacenados físicamente en `$BUILD_DIR`.

## Logging

```go
log.Printf("Starting background tag sync for module %s", moduleID)
log.Printf("Failed to fetch tags for module %s: %v", moduleID, err)
```

- Sincronizaciones en background logean inicio y fin
- Errores se logean con stack trace
- Operaciones HTTP logean requests (Gin default)

## Troubleshooting

### Error: "git clone failed"
- Verificar conectividad a repositorio Git
- Revisar SSH keys si usa git@
- Comprobar permisos de red del contenedor

### Error: "GPG initialization failed"
- Verificar permisos en `$GPG_HOME`
- Revisar que gpg esté instalado
- Providers funcionarán sin firma (warning)

### Error: "database locked"
- SQLite no soporta alta concurrencia
- Considerar PostgreSQL para producción
- Verificar que no haya múltiples procesos accediendo

### Módulos no sincronizan
- Revisar logs del backend
- Verificar campo `sync_error` en base de datos
- Usar endpoint `POST /api/modules/:id/sync-tags` para reintentar

## Producción

### Recomendaciones

1. **Usar volúmenes persistentes** para `/app/data`
2. **Configurar BASE_URL** correctamente para URLs públicas
3. **HTTPS** mediante reverse proxy (nginx, traefiger)
4. **Backups** regulares de SQLite y GPG keys
5. **Monitoreo** de logs y métricas

### Ejemplo docker-compose

```yaml
services:
  backend:
    build: ./backend
    ports:
      - "9080:9080"
    volumes:
      - ./data:/app/data
    environment:
      - BASE_URL=https://registry.example.com
      - DB_PATH=/app/data/registry.db
      - GPG_HOME=/app/data/gpg
    restart: unless-stopped
```

## Performance

- **SQLite** es suficiente para <1000 módulos/providers
- **Git clones** son eficientes con `--filter=blob:none`
- **Sincronización** en background no bloquea requests
- **Caché** de Git tags en base de datos

## Contribuir

Ver [CONTRIBUTING.md](../CONTRIBUTING.md) en el repositorio principal.

## Licencia

MIT
