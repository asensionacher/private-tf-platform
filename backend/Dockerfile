FROM golang:1.24-alpine AS builder

# Install git for go modules
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (no CGO needed with modernc.org/sqlite)
RUN CGO_ENABLED=0 GOOS=linux go build -o iac-tool .

# Final stage
FROM alpine:3.19

# Install git for fetching tags, openssh for git ssh, and gnupg for signing
RUN apk add --no-cache git ca-certificates openssh-client gnupg

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/iac-tool .

# Create data directory for SQLite database and GPG
RUN mkdir -p /app/data /app/data/gpg

# Expose port
EXPOSE 9080

# Run the application
CMD ["./iac-tool"]
