FROM node:20-alpine

# Build arguments for Vite environment variables
ARG VITE_REGISTRY_HOST=registry.local
ARG VITE_REGISTRY_PORT=9080
ARG VITE_API_BASE_URL=http://localhost:9080/api
ENV VITE_REGISTRY_HOST=$VITE_REGISTRY_HOST
ENV VITE_REGISTRY_PORT=$VITE_REGISTRY_PORT
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

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

# Install serve globally for production serving
RUN npm install -g serve

ENV PORT=3000

EXPOSE 3000

CMD ["sh", "-c", "serve -s dist -l $PORT"]
