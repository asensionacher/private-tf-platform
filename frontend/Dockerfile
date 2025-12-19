FROM node:20-alpine AS builder

# Build arguments for Vite environment variables
ARG VITE_REGISTRY_HOST=registry.local
ARG VITE_REGISTRY_PORT=9080
ENV VITE_REGISTRY_HOST=$VITE_REGISTRY_HOST
ENV VITE_REGISTRY_PORT=$VITE_REGISTRY_PORT

WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# Copy package files
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile

# Copy source code
COPY . .

# Build the application
RUN pnpm run build

# Final stage - serve with nginx
FROM nginx:alpine

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
