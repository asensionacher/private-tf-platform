#!/bin/bash
# Generates a self-signed CA and server certificates for the registry hostnames.
# Run this once from the repo root:
#   bash scripts/gen-certs.sh [REGISTRY_HOST]
# Default REGISTRY_HOST: registry.lan

set -euo pipefail

REGISTRY_HOST="${1:-registry.lan}"
CERTS_DIR="$(cd "$(dirname "$0")/.." && pwd)/nginx/certs"
mkdir -p "$CERTS_DIR"

echo "Generating certs for: $REGISTRY_HOST, api.$REGISTRY_HOST, tf.$REGISTRY_HOST"
echo "Output dir: $CERTS_DIR"

# --- CA ---
openssl genrsa -out "$CERTS_DIR/ca.key" 4096

openssl req -x509 -new -nodes \
  -key "$CERTS_DIR/ca.key" \
  -sha256 -days 3650 \
  -subj "/CN=IAC Registry CA/O=IAC Registry/C=US" \
  -out "$CERTS_DIR/ca.crt"

# --- Server key ---
openssl genrsa -out "$CERTS_DIR/server.key" 2048

# --- CSR + SAN config covering all three hostnames ---
cat > "$CERTS_DIR/server.cnf" <<EOF
[req]
default_bits       = 2048
prompt             = no
default_md         = sha256
distinguished_name = dn
req_extensions     = req_ext

[dn]
CN = $REGISTRY_HOST

[req_ext]
subjectAltName = @alt_names

[alt_names]
DNS.1 = $REGISTRY_HOST
DNS.2 = api.$REGISTRY_HOST
DNS.3 = tf.$REGISTRY_HOST
EOF

openssl req -new \
  -key "$CERTS_DIR/server.key" \
  -config "$CERTS_DIR/server.cnf" \
  -out "$CERTS_DIR/server.csr"

# --- Sign with CA ---
cat > "$CERTS_DIR/server_ext.cnf" <<EOF
subjectAltName = DNS:$REGISTRY_HOST,DNS:api.$REGISTRY_HOST,DNS:tf.$REGISTRY_HOST
EOF

openssl x509 -req \
  -in "$CERTS_DIR/server.csr" \
  -CA "$CERTS_DIR/ca.crt" \
  -CAkey "$CERTS_DIR/ca.key" \
  -CAcreateserial \
  -days 3650 \
  -sha256 \
  -extfile "$CERTS_DIR/server_ext.cnf" \
  -out "$CERTS_DIR/server.crt"

# Cleanup intermediates
rm -f "$CERTS_DIR/server.csr" "$CERTS_DIR/server.cnf" "$CERTS_DIR/server_ext.cnf" "$CERTS_DIR/ca.srl"

echo ""
echo "Done. Files generated:"
echo "  $CERTS_DIR/ca.crt      <- trust this on your machines"
echo "  $CERTS_DIR/server.crt"
echo "  $CERTS_DIR/server.key"
echo ""
echo "To trust the CA on this machine:"
echo "  sudo cp $CERTS_DIR/ca.crt /usr/local/share/ca-certificates/iac-registry-ca.crt"
echo "  sudo update-ca-certificates"
