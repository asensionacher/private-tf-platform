#!/bin/bash
# ==============================================================================
# Environment Configuration Validation Script
# ==============================================================================
# This script validates the .env file for production deployment
# Usage: ./scripts/validate-env.sh
# Exit codes: 0 = success, 1 = validation failed
# ==============================================================================

set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
ERRORS=0
WARNINGS=0
CHECKS=0

echo -e "${BLUE}=================================${NC}"
echo -e "${BLUE}Environment Validation Starting${NC}"
echo -e "${BLUE}=================================${NC}"
echo ""

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo -e "${RED}✗ ERROR: .env file not found${NC}"
    echo -e "${YELLOW}  Run: cp .env.example .env${NC}"
    echo -e "${YELLOW}  Or run: ./scripts/setup-env.sh${NC}"
    exit 1
fi

echo -e "${GREEN}✓ .env file found${NC}"
echo ""

# Source the .env file
set -a
source .env
set +a

# Function to check required variable
check_required() {
    local var_name=$1
    local var_value="${!var_name}"
    CHECKS=$((CHECKS + 1))
    
    if [ -z "$var_value" ]; then
        echo -e "${RED}✗ ERROR: $var_name is not set${NC}"
        ERRORS=$((ERRORS + 1))
        return 1
    fi
    return 0
}

# Function to check for insecure default values
check_not_default() {
    local var_name=$1
    local var_value="${!var_name}"
    local pattern=$2
    CHECKS=$((CHECKS + 1))
    
    if [[ "$var_value" =~ $pattern ]]; then
        echo -e "${RED}✗ ERROR: $var_name contains insecure default value${NC}"
        echo -e "${YELLOW}  Current value: $var_value${NC}"
        echo -e "${YELLOW}  This must be changed before production deployment${NC}"
        ERRORS=$((ERRORS + 1))
        return 1
    fi
    return 0
}

# Function to check string length
check_min_length() {
    local var_name=$1
    local var_value="${!var_name}"
    local min_length=$2
    CHECKS=$((CHECKS + 1))
    
    if [ ${#var_value} -lt $min_length ]; then
        echo -e "${RED}✗ ERROR: $var_name is too short (minimum $min_length characters)${NC}"
        echo -e "${YELLOW}  Current length: ${#var_value}${NC}"
        ERRORS=$((ERRORS + 1))
        return 1
    fi
    return 0
}

# Function to warn about insecure configuration
check_warning() {
    local var_name=$1
    local var_value="${!var_name}"
    local pattern=$2
    local message=$3
    CHECKS=$((CHECKS + 1))
    
    if [[ "$var_value" =~ $pattern ]]; then
        echo -e "${YELLOW}⚠ WARNING: $var_name - $message${NC}"
        echo -e "${YELLOW}  Current value: $var_value${NC}"
        WARNINGS=$((WARNINGS + 1))
        return 1
    fi
    return 0
}

echo -e "${BLUE}Checking Security Configuration...${NC}"
echo ""

# Check ENCRYPTION_KEY
if check_required "ENCRYPTION_KEY"; then
    check_not_default "ENCRYPTION_KEY" "REQUIRED_CHANGE_ME|change-this"
    check_min_length "ENCRYPTION_KEY" 32
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ ENCRYPTION_KEY is set and appears secure${NC}"
    fi
fi

echo ""

# Check POSTGRES_PASSWORD
if check_required "POSTGRES_PASSWORD"; then
    check_not_default "POSTGRES_PASSWORD" "REQUIRED_CHANGE_ME|registry-password-change-me|password|admin"
    check_min_length "POSTGRES_PASSWORD" 16
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ POSTGRES_PASSWORD is set and appears secure${NC}"
    fi
fi

echo ""
echo -e "${BLUE}Checking Host Configuration...${NC}"
echo ""

# Check host configuration
if check_required "FRONTEND_HOST"; then
    echo -e "${GREEN}✓ FRONTEND_HOST is set: $FRONTEND_HOST${NC}"
    if [[ "$FRONTEND_HOST" == "localhost" ]]; then
        echo -e "${YELLOW}  Note: Using localhost (development mode)${NC}"
    fi
fi

if check_required "BACKEND_HOST"; then
    echo -e "${GREEN}✓ BACKEND_HOST is set: $BACKEND_HOST${NC}"
fi

if check_required "REGISTRY_HOST"; then
    echo -e "${GREEN}✓ REGISTRY_HOST is set: $REGISTRY_HOST${NC}"
fi

echo ""
echo -e "${BLUE}Checking Port Configuration...${NC}"
echo ""

# Check port configuration
if check_required "FRONTEND_PORT"; then
    echo -e "${GREEN}✓ FRONTEND_PORT is set: $FRONTEND_PORT${NC}"
fi

if check_required "BACKEND_PORT"; then
    echo -e "${GREEN}✓ BACKEND_PORT is set: $BACKEND_PORT${NC}"
fi

if check_required "RUNNER_PORT"; then
    echo -e "${GREEN}✓ RUNNER_PORT is set: $RUNNER_PORT${NC}"
fi

echo ""
echo -e "${BLUE}Checking Service URLs...${NC}"
echo ""

# Check service URLs
if check_required "RUNNER_URL"; then
    echo -e "${GREEN}✓ RUNNER_URL is set: $RUNNER_URL${NC}"
fi

if check_required "REGISTRY_HOST"; then
    echo -e "${GREEN}✓ REGISTRY_HOST is set: $REGISTRY_HOST${NC}"
fi

echo ""
echo -e "${BLUE}Checking CORS Configuration...${NC}"
echo ""

# Check CORS configuration
if check_required "ALLOWED_ORIGINS"; then
    if [[ "$ALLOWED_ORIGINS" == "*" ]]; then
        if [[ "$FRONTEND_HOST" != "localhost" ]] && [[ "$BACKEND_HOST" != "localhost" ]]; then
            echo -e "${YELLOW}⚠ WARNING: ALLOWED_ORIGINS is set to wildcard (*) in production${NC}"
            echo -e "${YELLOW}  This is a security risk. Specify exact origins instead.${NC}"
            echo -e "${YELLOW}  Example: https://privatetf.local,https://www.privatetf.local${NC}"
            WARNINGS=$((WARNINGS + 1))
        else
            echo -e "${GREEN}✓ ALLOWED_ORIGINS is set to * (development mode)${NC}"
        fi
    else
        echo -e "${GREEN}✓ ALLOWED_ORIGINS is configured with specific domains${NC}"
    fi
fi

echo ""
echo -e "${BLUE}Checking Security Warnings...${NC}"
echo ""

# Check if PostgreSQL port is exposed
if [ ! -z "$POSTGRES_PORT_EXTERNAL" ]; then
    echo -e "${YELLOW}⚠ WARNING: POSTGRES_PORT_EXTERNAL is set${NC}"
    echo -e "${YELLOW}  PostgreSQL will be exposed on host port $POSTGRES_PORT_EXTERNAL${NC}"
    echo -e "${YELLOW}  Consider commenting this out for better security${NC}"
    WARNINGS=$((WARNINGS + 1))
else
    echo -e "${GREEN}✓ PostgreSQL port is not exposed externally (secure)${NC}"
fi

# Check if using localhost in production-like setup
if [[ "$FRONTEND_HOST" == "localhost" ]] && [[ ! "$ALLOWED_ORIGINS" == "*" ]]; then
    echo -e "${YELLOW}⚠ WARNING: Using localhost with restricted CORS${NC}"
    echo -e "${YELLOW}  This might be intentional for development, but verify for production${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""
echo -e "${BLUE}=================================${NC}"
echo -e "${BLUE}Validation Summary${NC}"
echo -e "${BLUE}=================================${NC}"
echo ""
echo -e "Total checks performed: $CHECKS"

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✓ All checks passed!${NC}"
    echo -e "${GREEN}✓ Configuration appears production-ready${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "  1. Review configuration one more time"
    echo -e "  2. Backup your .env file securely"
    echo -e "  3. Deploy with: docker-compose up -d"
    echo ""
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}⚠ Validation completed with $WARNINGS warning(s)${NC}"
    echo -e "${YELLOW}  Review the warnings above${NC}"
    echo -e "${YELLOW}  Configuration may still be usable but consider addressing warnings${NC}"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Validation failed with $ERRORS error(s) and $WARNINGS warning(s)${NC}"
    echo -e "${RED}  Please fix the errors above before deployment${NC}"
    echo ""
    echo -e "${BLUE}Quick fixes:${NC}"
    echo -e "  - Generate ENCRYPTION_KEY: openssl rand -base64 32"
    echo -e "  - Generate POSTGRES_PASSWORD: openssl rand -base64 24"
    echo -e "  - Or use automated setup: ./scripts/setup-env.sh"
    echo ""
    exit 1
fi
