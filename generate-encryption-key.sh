#!/bin/bash

# Script to generate a secure encryption key for the Terraform Private Registry

echo "Generating a secure encryption key..."
echo ""

# Generate a random 32-byte key and encode it in base64
KEY=$(openssl rand -base64 32)

echo "Your encryption key has been generated:"
echo ""
echo "ENCRYPTION_KEY=$KEY"
echo ""
echo "Add this to your .env file or export it as an environment variable."
echo ""
echo "For Docker Compose, create a .env file in the project root:"
echo "  echo \"ENCRYPTION_KEY=$KEY\" > .env"
echo ""
echo "Then restart your containers:"
echo "  docker compose down && docker compose up -d"
echo ""
