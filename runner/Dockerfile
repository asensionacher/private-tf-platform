# Stage 1: Build the runner
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o iac-runner .

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install base dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    python3 \
    py3-pip \
    bash \
    curl \
    unzip \
    groff \
    less \
    jq \
    wget

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

# Install AWS CLI v2
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
    unzip awscliv2.zip && \
    ./aws/install && \
    rm -rf awscliv2.zip aws

# Install Azure CLI
RUN apk add --no-cache --virtual .build-deps gcc musl-dev python3-dev libffi-dev openssl-dev cargo make && \
    pip3 install --break-system-packages --no-cache-dir azure-cli && \
    apk del .build-deps

# Copy the runner binary from builder
COPY --from=builder /build/iac-runner /usr/local/bin/iac-runner

# Create working directory with proper permissions
RUN mkdir -p /tmp/iac-deployments && \
    chmod 777 /tmp/iac-deployments

# Expose port
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/iac-runner"]
