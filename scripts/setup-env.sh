#!/bin/bash
# ==============================================================================
# Environment Configuration Setup Script
# ==============================================================================
# This script creates a production-ready .env file with secure defaults
# Usage: ./scripts/setup-env.sh
# ==============================================================================

set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}IAC Platform - Environment Setup${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# Check if .env already exists
if [ -f ".env" ]; then
    echo -e "${YELLOW}WARNING: .env file already exists!${NC}"
    echo -e "${YELLOW}This script will create a backup and overwrite it.${NC}"
    echo ""
    read -p "Do you want to continue? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Setup cancelled.${NC}"
        exit 1
    fi
    
    # Backup existing .env
    BACKUP_FILE=".env.backup.$(date +%Y%m%d_%H%M%S)"
    cp .env "$BACKUP_FILE"
    echo -e "${GREEN}✓ Backed up existing .env to $BACKUP_FILE${NC}"
    echo ""
fi

echo -e "${CYAN}This script will help you create a production-ready .env file.${NC}"
echo -e "${CYAN}Press Enter to use default values shown in brackets.${NC}"
echo ""

# ==============================================================================
# Generate secure secrets
# ==============================================================================

echo -e "${BLUE}Generating secure secrets...${NC}"
echo ""

# Generate ENCRYPTION_KEY
if command -v openssl &> /dev/null; then
    ENCRYPTION_KEY=$(openssl rand -base64 32)
    echo -e "${GREEN}✓ Generated secure ENCRYPTION_KEY${NC}"
else
    echo -e "${YELLOW}⚠ openssl not found, using fallback method${NC}"
    ENCRYPTION_KEY=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
fi

# Generate POSTGRES_PASSWORD
if command -v openssl &> /dev/null; then
    POSTGRES_PASSWORD=$(openssl rand -base64 24)
    echo -e "${GREEN}✓ Generated secure POSTGRES_PASSWORD${NC}"
else
    POSTGRES_PASSWORD=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9!@#$%^&*' | fold -w 24 | head -n 1)
fi

echo ""

# ==============================================================================
# Environment Selection
# ==============================================================================

echo -e "${BLUE}Environment Configuration${NC}"
echo ""
echo "1) Production (recommended for deployment)"sudo ss -ltnp | grep :3000
echo "2) Development (localhost with relaxed security)"
echo ""
read -p "Select environment [1]: " ENV_CHOICE
ENV_CHOICE=${ENV_CHOICE:-1}

if [ "$ENV_CHOICE" == "1" ]; then
    ENVIRONMENT="production"
    echo -e "${GREEN}✓ Production environment selected${NC}"
else
    ENVIRONMENT="development"
    echo -e "${YELLOW}✓ Development environment selected${NC}"
fi

echo ""

# ==============================================================================
# Domain/Host Configuration
# ==============================================================================

echo -e "${BLUE}Domain/Host Configuration${NC}"
echo ""

if [ "$ENVIRONMENT" == "production" ]; then
    read -p "Enter your domain name [privatetf.local]: " DOMAIN
    DOMAIN=${DOMAIN:-privatetf.local}
    
    FRONTEND_HOST=$DOMAIN
    BACKEND_HOST=$DOMAIN
    REGISTRY_HOST=$DOMAIN
    
    echo -e "${GREEN}✓ Using domain: $DOMAIN${NC}"
    
    # CORS Configuration for production
    read -p "Use HTTPS? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        PROTOCOL="https"
        ALLOWED_ORIGINS="https://$DOMAIN"
    else
        PROTOCOL="http"
        ALLOWED_ORIGINS="http://$DOMAIN"
    fi
    
    echo -e "${GREEN}✓ CORS configured for: $ALLOWED_ORIGINS${NC}"
    
else
    # Development defaults
    FRONTEND_HOST="localhost"
    BACKEND_HOST="localhost"
    REGISTRY_HOST="localhost"
    ALLOWED_ORIGINS="*"
    PROTOCOL="http"
    
    echo -e "${YELLOW}✓ Using localhost (development mode)${NC}"
fi

echo ""

# ==============================================================================
# Port Configuration
# ==============================================================================

echo -e "${BLUE}Port Configuration${NC}"
echo ""

read -p "Frontend port [3000]: " FRONTEND_PORT
FRONTEND_PORT=${FRONTEND_PORT:-3000}

read -p "Backend port [9080]: " BACKEND_PORT
BACKEND_PORT=${BACKEND_PORT:-9080}

read -p "Runner port [8080]: " RUNNER_PORT
RUNNER_PORT=${RUNNER_PORT:-8080}

VITE_DEV_PORT=5173

echo -e "${GREEN}✓ Ports configured${NC}"
echo ""

# PostgreSQL external port
if [ "$ENVIRONMENT" == "production" ]; then
    read -p "Expose PostgreSQL externally? (not recommended) (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "PostgreSQL external port [5432]: " POSTGRES_PORT_EXTERNAL
        POSTGRES_PORT_EXTERNAL=${POSTGRES_PORT_EXTERNAL:-5432}
        EXPOSE_POSTGRES="yes"
    else
        EXPOSE_POSTGRES="no"
    fi
else
    EXPOSE_POSTGRES="no"
fi

# ==============================================================================
# Service URLs
# ==============================================================================

RUNNER_URL="http://runner:8080"

if [ "$ENVIRONMENT" == "production" ] && [ "$PROTOCOL" == "https" ]; then
    REGISTRY_HOST="https://$DOMAIN"
else
    REGISTRY_HOST="http://registry.local:9080"
fi

VITE_API_BASE_URL=""

# ==============================================================================
# Create .env file
# ==============================================================================

echo -e "${BLUE}Creating .env file...${NC}"
echo ""

cat > .env << EOF
# ==============================================================================
# IAC Platform - Environment Configuration
# ==============================================================================
# Generated by setup-env.sh on $(date)
# Environment: $ENVIRONMENT
# ==============================================================================

# ==============================================================================
# SECURITY CONFIGURATION
# ==============================================================================

# Encryption key (auto-generated)
ENCRYPTION_KEY=$ENCRYPTION_KEY

# PostgreSQL password (auto-generated)
POSTGRES_PASSWORD=$POSTGRES_PASSWORD

# ==============================================================================
# DOMAIN/HOST CONFIGURATION
# ==============================================================================

FRONTEND_HOST=$FRONTEND_HOST
BACKEND_HOST=$BACKEND_HOST
REGISTRY_HOST=$REGISTRY_HOST

# ==============================================================================
# PORT CONFIGURATION
# ==============================================================================

FRONTEND_PORT=$FRONTEND_PORT
BACKEND_PORT=$BACKEND_PORT
RUNNER_PORT=$RUNNER_PORT
VITE_DEV_PORT=$VITE_DEV_PORT
EOF

# Add PostgreSQL external port if enabled
if [ "$EXPOSE_POSTGRES" == "yes" ]; then
    echo "POSTGRES_PORT_EXTERNAL=$POSTGRES_PORT_EXTERNAL" >> .env
else
    echo "# POSTGRES_PORT_EXTERNAL=5432" >> .env
fi

cat >> .env << EOF

# ==============================================================================
# SERVICE URLs
# ==============================================================================

RUNNER_URL=$RUNNER_URL
REGISTRY_HOST=$REGISTRY_HOST

# ==============================================================================
# FRONTEND CONFIGURATION
# ==============================================================================

VITE_API_BASE_URL=$VITE_API_BASE_URL

# ==============================================================================
# CORS CONFIGURATION
# ==============================================================================

ALLOWED_ORIGINS=$ALLOWED_ORIGINS

# ==============================================================================
# End of configuration
# ==============================================================================
EOF

echo -e "${GREEN}✓ .env file created successfully!${NC}"
echo ""

# ==============================================================================
# Validation
# ==============================================================================

echo -e "${BLUE}Validating configuration...${NC}"
echo ""

if [ -x "./scripts/validate-env.sh" ]; then
    ./scripts/validate-env.sh
    VALIDATION_RESULT=$?
else
    echo -e "${YELLOW}⚠ Validation script not found or not executable${NC}"
    VALIDATION_RESULT=0
fi

echo ""

# ==============================================================================
# Summary
# ==============================================================================

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}Setup Complete!${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo -e "${GREEN}Configuration Summary:${NC}"
echo -e "  Environment: $ENVIRONMENT"
echo -e "  Domain: $DOMAIN"
echo -e "  Frontend: $PROTOCOL://$FRONTEND_HOST:$FRONTEND_PORT"
echo -e "  Backend: $PROTOCOL://$BACKEND_HOST:$BACKEND_PORT"
echo -e "  PostgreSQL exposed: $EXPOSE_POSTGRES"
echo ""

if [ "$ENVIRONMENT" == "production" ]; then
    echo -e "${YELLOW}IMPORTANT - Production Checklist:${NC}"
    echo -e "${YELLOW}  [ ] Review .env file for any additional customization${NC}"
    echo -e "${YELLOW}  [ ] Backup .env file securely (it contains secrets!)${NC}"
    echo -e "${YELLOW}  [ ] Configure reverse proxy (nginx/traefik/caddy)${NC}"
    echo -e "${YELLOW}  [ ] Set up SSL/TLS certificates${NC}"
    echo -e "${YELLOW}  [ ] Configure firewall rules${NC}"
    echo -e "${YELLOW}  [ ] Set up monitoring and logging${NC}"
    echo -e "${YELLOW}  [ ] Plan backup strategy${NC}"
    echo ""
    echo -e "${CYAN}For detailed deployment instructions, see:${NC}"
    echo -e "  docs/DEPLOYMENT.md"
    echo ""
fi

echo -e "${CYAN}Next steps:${NC}"
echo -e "  1. Review your .env file:"
echo -e "     ${BLUE}cat .env${NC}"
echo ""
echo -e "  2. Start the services:"
echo -e "     ${BLUE}docker-compose up -d${NC}"
echo ""
echo -e "  3. Check service status:"
echo -e "     ${BLUE}docker-compose ps${NC}"
echo ""
echo -e "  4. View logs:"
echo -e "     ${BLUE}docker-compose logs -f${NC}"
echo ""

if [ $VALIDATION_RESULT -ne 0 ]; then
    echo -e "${RED}⚠ Validation warnings detected. Please review them above.${NC}"
    echo ""
fi

echo -e "${GREEN}Setup completed successfully!${NC}"
