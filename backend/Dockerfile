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

# Install dependencies including Terraform and OpenTofu
RUN apk add --no-cache git ca-certificates openssh-client gnupg curl unzip

# Install Terraform
RUN TERRAFORM_VERSION=1.14.2 && \
    curl -LO https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
    terraform version

# Install OpenTofu
RUN TOFU_VERSION=1.11.1 && \
    curl -LO https://github.com/opentofu/opentofu/releases/download/v${TOFU_VERSION}/tofu_${TOFU_VERSION}_linux_amd64.zip && \
    unzip tofu_${TOFU_VERSION}_linux_amd64.zip && \
    mv tofu /usr/local/bin/ && \
    rm tofu_${TOFU_VERSION}_linux_amd64.zip && \
    tofu version

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/iac-tool .

# Create data directory for SQLite database and GPG
RUN mkdir -p /app/data /app/data/gpg

# Expose port
EXPOSE 9080

# Run the application
CMD ["./iac-tool"]
