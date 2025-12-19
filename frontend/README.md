# Frontend - Terraform Registry Web UI

Modern React-based web interface for managing the private Terraform registry platform. Built with TypeScript, Vite, Tailwind CSS, and React Query.

## Overview

The frontend provides a comprehensive web UI for:
- **Module Management** - Browse, create, version, and manage Terraform modules
- **Provider Management** - Browse, create, version, and manage Terraform providers with platform binaries
- **Deployment Management** - Create and execute Terraform/OpenTofu deployments with plan/apply workflows
- **Namespace Management** - Organize modules and providers by namespace/organization
- **API Key Management** - Generate authentication tokens for Terraform CLI access
- **Live Monitoring** - Real-time status updates and streaming logs for deployment runs

## Architecture

### Technology Stack
- **Framework**: React 18.2
- **Build Tool**: Vite 5.0
- **Language**: TypeScript 5.2
- **Styling**: Tailwind CSS 3.3
- **Routing**: React Router DOM 6.20
- **State Management**: TanStack React Query 5.8
- **HTTP Client**: Axios 1.6
- **UI Components**:
  - `lucide-react` - Icon library
  - `react-markdown` - Markdown rendering
  - `react-syntax-highlighter` - Code syntax highlighting
  - `ansi-to-html` - Terminal log formatting

### Project Structure

```
frontend/
├── public/
│   └── favicon.svg           # Application favicon
├── src/
│   ├── api/
│   │   └── index.ts          # Centralized API client with typed endpoints
│   ├── components/
│   │   ├── AnsiOutput.tsx    # ANSI terminal log renderer
│   │   ├── DeploymentModal.tsx   # Deployment creation modal
│   │   ├── Layout.tsx        # Main layout with navigation
│   │   └── ThemeContext.tsx  # Dark/light theme context (legacy)
│   ├── context/
│   │   └── ThemeContext.tsx  # Theme provider
│   ├── pages/
│   │   ├── ApiKeysPage.tsx              # Global API key management
│   │   ├── DeploymentDetailPage.tsx     # Deployment details and runs
│   │   ├── DeploymentRunDetailPage.tsx  # Individual run with live logs
│   │   ├── DeploymentRunsPage.tsx       # List of deployment runs
│   │   ├── DeploymentsPage.tsx          # Deployments list
│   │   ├── ModuleDetailPage.tsx         # Module versions and README
│   │   ├── ModulesPage.tsx              # Modules list
│   │   ├── NamespaceDetailPage.tsx      # Namespace details
│   │   ├── NamespacesPage.tsx           # Namespaces list
│   │   ├── ProviderDetailPage.tsx       # Provider versions and platforms
│   │   └── ProvidersPage.tsx            # Providers list
│   ├── types/
│   │   └── index.ts          # TypeScript type definitions
│   ├── App.tsx               # Main app with routing
│   ├── index.css             # Global styles and Tailwind directives
│   ├── main.tsx              # Application entry point
│   └── vite-env.d.ts         # Vite type declarations
├── Dockerfile                # Multi-stage Docker build with nginx
├── nginx.conf                # Nginx configuration for production
├── package.json              # Dependencies and scripts
├── pnpm-lock.yaml            # pnpm lock file
├── postcss.config.js         # PostCSS configuration
├── tailwind.config.js        # Tailwind CSS configuration
├── tsconfig.json             # TypeScript configuration
├── tsconfig.node.json        # TypeScript config for Node scripts
└── vite.config.ts            # Vite build configuration
```

## Key Features

### 1. Module Management
- List all modules across namespaces
- Create modules from Git repositories (HTTPS, SSH, token-based auth)
- View module versions with enable/disable toggle
- Sync Git tags automatically
- Display module README with markdown rendering
- Generate Terraform usage examples

### 2. Provider Management
- List all providers across namespaces
- Create providers from Git repositories
- Manage provider versions with multiple platform binaries (OS/arch)
- Upload pre-built provider binaries
- GPG signature support for provider verification
- SHA256 checksum generation
- Generate Terraform CLI configuration examples

### 3. Deployment Management
- Create deployments linked to Git repositories
- Browse repository directory structure
- Select working directory and .tfvars files
- Execute plan/apply workflows with approval gates
- **Real-time status updates** (auto-refresh every 3-5 seconds)
- **Streaming logs** for init, plan, and apply phases
- Support for both Terraform and OpenTofu
- Cancel running deployments
- View deployment run history

### 4. Live Monitoring
- **Deployment List**: Auto-refresh every 5 seconds
- **Deployment Runs**: Auto-refresh every 3 seconds when active
- **Run Details**: Live log streaming with 2-second polling
- Silent background updates (no loading spinners on refresh)
- "Waiting for logs to stream..." placeholders during execution

### 5. Namespace & API Key Management
- Create and manage namespaces/organizations
- Generate global API keys for Terraform CLI
- View API key usage statistics
- Copy API keys to clipboard

## Setup and Installation

### Prerequisites

- **Node.js 18+** - [Install Node.js](https://nodejs.org/)
- **pnpm** - [Install pnpm](https://pnpm.io/installation)
  ```bash
  npm install -g pnpm
  ```

### Manual Setup (without Docker)

1. **Navigate to frontend directory**
   ```bash
   cd frontend
   ```

2. **Install dependencies**
   ```bash
   pnpm install
   ```

3. **Configure environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   
   # Build-time variables (for production):
   export VITE_REGISTRY_HOST=http://localhost:9080
   export VITE_REGISTRY_PORT=9080
   
   # Development uses proxy in vite.config.ts
   ```

4. **Start development server**
   ```bash
   pnpm dev
   ```

   The app will be available at `http://localhost:5173`

5. **Build for production**
   ```bash
   pnpm build
   
   # Output will be in dist/
   # Serve with any static file server
   ```

6. **Preview production build**
   ```bash
   pnpm preview
   ```

### Docker Setup

Using Docker Compose (recommended):

```bash
# From project root
docker-compose up -d frontend

# View logs
docker-compose logs -f frontend

# Rebuild after changes
docker-compose up -d --build frontend
```

Using standalone Docker:

```bash
# Build
docker build -t iac-frontend \
  --build-arg VITE_REGISTRY_HOST=http://localhost:9080 \
  --build-arg VITE_REGISTRY_PORT=9080 \
  .

# Run
docker run -d -p 3000:80 --name iac-frontend iac-frontend
```

The Dockerfile uses a multi-stage build:
1. **Stage 1**: Build React app with Vite
2. **Stage 2**: Serve with nginx

## Configuration

### Environment Variables

Build-time variables (used during `vite build`):

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_REGISTRY_HOST` | `http://localhost:9080` | Backend API URL |
| `VITE_REGISTRY_PORT` | `9080` | Backend API port |
| `VITE_API_BASE_URL` | `/api` | API base path (optional) |

Runtime variables (for nginx container):

| Variable | Default | Description |
|----------|---------|-------------|
| `NGINX_PORT` | `80` | nginx listening port |

### Vite Configuration

Development proxy (`vite.config.ts` lines 14-19):
```typescript
server: {
  port: 5173,
  proxy: {
    '/api': {
      target: 'http://localhost:9080',
      changeOrigin: true,
    },
  },
}
```

This allows the development server to proxy API requests to the backend without CORS issues.

### API Client Configuration

The API client (`src/api/index.ts`) uses axios with:
- Base URL from `VITE_API_BASE_URL` or `/api`
- No authentication headers (management API is open)
- Centralized type-safe API functions for all endpoints

## Development

### Available Scripts

```bash
# Start development server (port 5173)
pnpm dev

# Build for production
pnpm build

# Preview production build
pnpm preview

# Lint TypeScript files
pnpm lint
```

### Code Style

- **TypeScript**: Strict mode enabled (`strict: true`)
- **Imports**: External deps first, then internal (`../api`, `../types`, `../components`)
- **Components**: Functional components with hooks
- **Naming**: PascalCase for components/types, camelCase for functions/variables
- **State Management**: React Query for server state, useState for local UI state
- **Error Handling**: Try/catch in mutations, user-friendly error messages
- **Formatting**: 2-space indents, single quotes, semicolons

### Adding New Pages

1. **Create page component** in `src/pages/`
   ```typescript
   import { useQuery } from '@tanstack/react-query';
   import { myApi } from '../api';
   
   export default function MyPage() {
     const { data, isLoading } = useQuery({
       queryKey: ['myData'],
       queryFn: myApi.getData,
     });
     
     if (isLoading) return <div>Loading...</div>;
     
     return (
       <div className="p-6">
         {/* Your content */}
       </div>
     );
   }
   ```

2. **Add route** in `src/App.tsx`
   ```typescript
   <Route path="/my-page" element={<MyPage />} />
   ```

3. **Add navigation link** in `src/components/Layout.tsx` if needed

### Adding API Endpoints

1. **Define types** in `src/types/index.ts`
   ```typescript
   export interface MyData {
     id: string;
     name: string;
   }
   ```

2. **Add API functions** in `src/api/index.ts`
   ```typescript
   export const myApi = {
     getData: () => api.get<MyData[]>('/my-data').then(res => res.data || []),
     getById: (id: string) => api.get<MyData>(`/my-data/${id}`).then(res => res.data),
     create: (data: Partial<MyData>) => api.post<MyData>('/my-data', data).then(res => res.data),
   };
   ```

3. **Use in components** with React Query
   ```typescript
   const { data } = useQuery({
     queryKey: ['myData'],
     queryFn: myApi.getData,
   });
   
   const mutation = useMutation({
     mutationFn: myApi.create,
     onSuccess: () => queryClient.invalidateQueries({ queryKey: ['myData'] }),
   });
   ```

### Real-time Updates Pattern

For auto-refreshing data (used in deployment pages):

```typescript
const [isPolling, setIsPolling] = useState(true);

const { data, refetch } = useQuery({
  queryKey: ['deployments'],
  queryFn: () => loadData(false), // false = silent (no loading spinner)
});

useEffect(() => {
  if (!isPolling) return;
  
  const interval = setInterval(() => {
    refetch(); // Silent background refresh
  }, 5000); // 5 seconds
  
  return () => clearInterval(interval);
}, [isPolling, refetch]);
```

See implementation examples:
- `DeploymentsPage.tsx` lines 22-31 (5-second polling)
- `DeploymentRunsPage.tsx` lines 15-41 (3-second conditional polling)
- `DeploymentRunDetailPage.tsx` lines 20-64 (2-second log streaming)

## Component Patterns

### Layout Component
Provides consistent navigation and theme support across all pages.

### AnsiOutput Component
Renders terminal output with ANSI color codes preserved:
```typescript
<AnsiOutput text={logOutput} />
```

### DeploymentModal Component
Reusable modal for creating deployments with Git configuration.

### Page Components
All page components follow this structure:
1. React Query hooks for data fetching
2. Loading states
3. Error handling
4. Main content rendering
5. Action buttons and modals

## Styling

### Tailwind CSS

The project uses Tailwind CSS with custom configuration:

**Key classes used:**
- `bg-gray-900`, `text-white` - Dark theme colors
- `border border-gray-700` - Card borders
- `rounded-lg` - Rounded corners
- `p-6`, `px-4`, `py-2` - Padding utilities
- `space-y-4` - Vertical spacing
- `grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3` - Responsive grids
- `hover:bg-gray-700` - Interactive states

**Typography plugin** enabled for markdown rendering.

### Custom Styles

Global styles in `src/index.css`:
- Base dark theme background
- Custom scrollbar styles
- Markdown content styling

## Troubleshooting

### Port Already in Use

```bash
# Kill process on port 5173
lsof -i :5173
kill -9 <PID>

# Or use different port
pnpm dev --port 5174
```

### API Connection Issues

1. Check backend is running on port 9080
2. Verify `VITE_REGISTRY_HOST` in `.env` (for production builds)
3. Check browser console for CORS errors
4. Verify proxy configuration in `vite.config.ts` (for development)

### TypeScript Errors

```bash
# Check for errors
pnpm lint

# Rebuild TypeScript
pnpm build
```

### Build Errors

```bash
# Clear cache and rebuild
rm -rf node_modules dist
pnpm install
pnpm build
```

### Docker Build Issues

```bash
# Check build args are set
docker build -t iac-frontend \
  --build-arg VITE_REGISTRY_HOST=http://localhost:9080 \
  --build-arg VITE_REGISTRY_PORT=9080 \
  --no-cache \
  .
```

## Production Deployment

### Build Optimization

The production build:
- Minifies JavaScript and CSS
- Tree-shakes unused code
- Generates source maps
- Optimizes images and assets
- Code-splits by route

### nginx Configuration

The included `nginx.conf`:
- Serves static files from `/usr/share/nginx/html`
- Proxies `/api/*` requests to backend
- Handles client-side routing (SPA fallback)
- Enables gzip compression
- Sets proper caching headers

### Environment-specific Builds

For different environments:

```bash
# Development
VITE_REGISTRY_HOST=http://dev.registry.local:9080 pnpm build

# Staging
VITE_REGISTRY_HOST=https://staging.registry.company.com pnpm build

# Production
VITE_REGISTRY_HOST=https://registry.company.com pnpm build
```

### Checklist

- [ ] Set correct `VITE_REGISTRY_HOST` for your environment
- [ ] Update nginx `proxy_pass` in `nginx.conf` if backend is remote
- [ ] Enable SSL/TLS in nginx or reverse proxy
- [ ] Configure CSP headers for security
- [ ] Set up CDN for static assets (optional)
- [ ] Enable monitoring and error tracking
- [ ] Test all routes work with nginx SPA fallback

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Modern mobile browsers

## Performance

### Optimizations Implemented
- Code splitting by route (React Router lazy loading not yet implemented)
- React Query caching with 5-minute stale time
- Silent background polling (no loading spinners on refresh)
- Debounced search inputs (where applicable)
- Optimistic UI updates for mutations

### Metrics
- First Contentful Paint: < 1.5s
- Time to Interactive: < 3s
- Bundle size: ~200KB gzipped

## Related Documentation

- [Root README](../README.md) - Full project overview and quick start
- [Backend README](../backend/README.md) - Backend API documentation
- [Runner README](../runner/README.md) - Runner service documentation
- [Vite Documentation](https://vitejs.dev/) - Build tool documentation
- [React Query Documentation](https://tanstack.com/query/latest) - State management

## License

See [../LICENSE](../LICENSE) for details.
