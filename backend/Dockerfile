FROM golang:1.24-alpine AS builder

# Install git for go modules
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY backend/ ./

# Build the application (no CGO needed with modernc.org/sqlite)
RUN CGO_ENABLED=0 GOOS=linux go build -o iac-tool .

# Final stage
FROM alpine:3.19

# Install minimal dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    openssh-client \
    gnupg \
    curl

WORKDIR /app

# Copy the backend binary
COPY --from=builder /app/iac-tool .

# Create data directory for SQLite database and GPG
RUN mkdir -p /app/data /app/data/gpg

# Expose port
EXPOSE 9080

# Run the application
CMD ["./iac-tool"]
