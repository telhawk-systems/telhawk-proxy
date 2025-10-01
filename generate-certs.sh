#!/bin/bash
# Generate self-signed SSL certificates for testing HTTPS

set -e

echo "Generating self-signed SSL certificates for testing..."

# Generate private key
openssl genrsa -out server.key 2048

# Generate certificate signing request
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Generate self-signed certificate
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt

# Clean up CSR file
rm server.csr

echo "SSL certificates generated:"
echo "  server.key - Private key"
echo "  server.crt - Certificate"
echo ""
echo "To use HTTPS, set these environment variables:"
echo "  export ENABLE_HTTPS=true"
echo "  export SSL_CERT_FILE=./server.crt"
echo "  export SSL_KEY_FILE=./server.key"
echo ""
echo "Note: These are self-signed certificates for testing only."
echo "For production, use certificates from a trusted Certificate Authority."