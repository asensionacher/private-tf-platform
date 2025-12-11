# Frontend - Terraform Private Registry UI

Interfaz web moderna para gestionar módulos y providers de Terraform. Desarrollada con React, TypeScript y Tailwind CSS.

## Tecnologías

- **Framework**: React 18 + TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **Routing**: React Router v6
- **State Management**: TanStack Query (React Query)
- **HTTP Client**: Axios
- **Icons**: Lucide React
- **Markdown**: React Markdown con GitHub Flavored Markdown

## Estructura del Proyecto

```
frontend/
├── src/
│   ├── main.tsx              # Punto de entrada
│   ├── App.tsx               # Componente raíz con routing
│   ├── index.css             # Estilos globales
│   ├── api/
│   │   └── index.ts          # Cliente API (axios)
│   ├── components/
│   │   └── Layout.tsx        # Layout principal con navegación
│   ├── context/
│   │   └── ThemeContext.tsx  # Contexto de tema (dark/light)
│   ├── pages/
│   │   ├── ModulesPage.tsx           # Listado de módulos
│   │   ├── ModuleDetailPage.tsx      # Detalle de módulo
│   │   ├── ProvidersPage.tsx         # Listado de providers
│   │   ├── ProviderDetailPage.tsx    # Detalle de provider
│   │   ├── NamespacesPage.tsx        # Gestión de namespaces
│   │   ├── NamespaceDetailPage.tsx   # Detalle de namespace
│   │   └── SettingsPage.tsx          # Configuración
│   └── types/
│       └── index.ts          # TypeScript types/interfaces
├── public/
├── index.html
├── vite.config.ts
├── tailwind.config.js
├── package.json
└── Dockerfile
```

## Características

### 🎨 UI/UX

- **Diseño responsive**: Funciona en desktop, tablet y móvil
- **Dark mode**: Cambio automático según preferencias del sistema
- **Navegación intuitiva**: Sidebar con secciones principales
- **Estados de carga**: Spinners y esqueletos durante peticiones
- **Feedback visual**: Mensajes de éxito/error, confirmaciones

### 📦 Gestión de Módulos

- Listar módulos agrupados por namespace
- Crear módulos desde repositorios Git
- Ver detalles: versiones, README, metadata
- Sincronizar versiones automáticamente desde Git tags
- Habilitar/deshabilitar versiones específicas
- **Manejo de errores de sincronización**:
  - Mostrar errores detallados
  - Botón "Retry Sync" para reintentar
  - Botón "Delete" para eliminar módulos problemáticos
- Auto-refresh mientras sincroniza (polling cada 3s)

### 🔌 Gestión de Providers

- Listar providers agrupados por namespace
- Crear providers desde repositorios Git
- Ver versiones y plataformas (OS/arch)
- Subir binarios por plataforma
- Gestión de signing keys (GPG)
- Similar manejo de errores que módulos

### 🏢 Gestión de Namespaces

- Crear y editar namespaces
- Ver estadísticas (número de módulos/providers)
- Generar API keys por namespace
- Configurar permisos (read/write/admin)

### 📖 Visualización de README

- Renderizado de Markdown con sintaxis GitHub
- Soporte para imágenes, tablas, código
- Sanitización de HTML por seguridad
- Carga dinámica desde Git repository

## Instalación y Desarrollo

### Requisitos

- Node.js 18+ o pnpm

### Instalación

```bash
cd frontend

# Instalar dependencias
pnpm install
# o
npm install
```

### Desarrollo

```bash
# Servidor de desarrollo con hot-reload
pnpm dev

# Disponible en http://localhost:5173
```

### Build para Producción

```bash
# Compilar y optimizar
pnpm build

# Los archivos estáticos se generan en dist/
```

### Preview del Build

```bash
pnpm preview
```

## Configuración

### Variables de Entorno

Crear archivo `.env` o `.env.local`:

```env
# URL del backend API
VITE_API_URL=http://localhost:9080
```

### Configuración de API

En `src/api/index.ts`:

```typescript
const api = axios.create({
  baseURL: '/api',  // Proxy configurado en vite.config.ts
});
```

### Proxy de Desarrollo (Vite)

En `vite.config.ts`:

```typescript
export default defineConfig({
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      },
      '/.well-known': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      },
      '/v1': {
        target: 'http://localhost:9080',
        changeOrigin: true,
      }
    }
  }
});
```

## Componentes Principales

### Layout

```tsx
// Layout con sidebar, header y contenido
<Layout>
  <Outlet /> {/* React Router */}
</Layout>
```

### ThemeProvider

```tsx
// Manejo de tema claro/oscuro
const { theme, toggleTheme } = useTheme();
```

### React Query

```tsx
// Cache y sincronización de datos
const { data, isLoading } = useQuery({
  queryKey: ['modules'],
  queryFn: () => modulesApi.getAll(),
});

const mutation = useMutation({
  mutationFn: modulesApi.create,
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['modules'] });
  },
});
```

## Funcionalidades Destacadas

### Auto-refresh durante Sincronización

```tsx
// En ModulesPage.tsx
const hasSyncingModules = modules.some(m => !m.synced);

useEffect(() => {
  if (hasSyncingModules) {
    const interval = setInterval(() => {
      queryClient.invalidateQueries({ queryKey: ['modules'] });
    }, 3000);
    return () => clearInterval(interval);
  }
}, [hasSyncingModules, queryClient]);
```

### Manejo de Errores de Sincronización

Los módulos pueden tener tres estados:
- **Sincronizando**: Spinner amarillo, no clickeable
- **Error**: Badge rojo con mensaje, botones "Retry" y "Delete"
- **Sincronizado**: Verde, clickeable para ver detalles

### Confirmación de Acciones Destructivas

```tsx
const handleDelete = (e: React.MouseEvent, moduleId: string) => {
  e.stopPropagation();
  if (confirm('Are you sure you want to delete this module?')) {
    deleteMutation.mutate(moduleId);
  }
};
```

## Estilos y Temas

### Tailwind CSS

Clases principales:
- `dark:` prefix para modo oscuro
- Colores principales: `indigo` (primary), `gray` (neutral)
- Estados: `hover:`, `focus:`, `disabled:`

### Dark Mode

```tsx
// En tailwind.config.js
module.exports = {
  darkMode: 'class', // Controlado por clase .dark en <html>
  // ...
}

// En ThemeContext
document.documentElement.classList.toggle('dark', theme === 'dark');
```

## TypeScript Types

### Principales Interfaces

```typescript
// Módulo
interface Module {
  id: string;
  namespace_id: string;
  namespace: string;
  name: string;
  provider: string;
  description?: string;
  source_url?: string;
  synced: boolean;
  sync_error?: string;  // Nuevo: errores de sincronización
  created_at: string;
  updated_at: string;
}

// Versión de módulo
interface ModuleVersion {
  id: string;
  module_id: string;
  version: string;
  download_url: string;
  enabled: boolean;
  tag_date?: string;
  created_at: string;
}

// Namespace
interface Namespace {
  id: string;
  name: string;
  description?: string;
  is_public: boolean;
  module_count?: number;
  provider_count?: number;
}
```

## Build para Docker

### Dockerfile Multi-stage

```dockerfile
# Build stage
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install
COPY . .
RUN pnpm build

# Production stage
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Configuración Nginx

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    # SPA routing
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API requests
    location /api/ {
        proxy_pass http://backend:9080;
    }
    
    location /v1/ {
        proxy_pass http://backend:9080;
    }
}
```

## Mejoras de Performance

- **Code splitting**: Rutas lazy-loaded
- **Tree shaking**: Vite elimina código no usado
- **Minificación**: CSS y JS minificados
- **Caching**: React Query cachea respuestas API
- **Optimistic updates**: UI actualiza antes de confirmar

## Accesibilidad

- Navegación por teclado
- Labels en formularios
- Contraste de colores (WCAG AA)
- Focus indicators visibles
- ARIA attributes en componentes interactivos

## Testing (Recomendado)

```bash
# Instalar dependencias de testing
pnpm add -D vitest @testing-library/react @testing-library/jest-dom

# Ejecutar tests
pnpm test
```

## Troubleshooting

### Error: "Cannot connect to API"
- Verificar que backend esté ejecutándose
- Revisar proxy en `vite.config.ts`
- Comprobar CORS en backend

### Error: "Module not found"
- Limpiar node_modules: `rm -rf node_modules && pnpm install`
- Verificar imports relativos
- Revisar alias en `tsconfig.json`

### Build falla
- Revisar errores TypeScript: `pnpm tsc --noEmit`
- Verificar versiones de Node/pnpm
- Limpiar cache: `pnpm store prune`

### Hot reload no funciona
- Reiniciar servidor dev
- Verificar puertos no estén ocupados
- Comprobar file watchers (Linux: `fs.inotify.max_user_watches`)

## Contribuir

Para contribuir al frontend:
1. Fork del repositorio
2. Crear branch feature
3. Seguir guías de estilo TypeScript/React
4. Usar Prettier para formateo
5. Asegurar que build funciona: `pnpm build`

## Licencia

MIT
