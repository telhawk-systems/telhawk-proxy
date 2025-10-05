#!/bin/bash
# Quick HTTPS test script for GoTrack

set -e

echo "=== GoTrack HTTPS Setup Test ==="
echo

# Build the application
echo "1. Building GoTrack..."
go build -o ./gotrack ./cmd/gotrack
echo "✓ Build complete"
echo

# Generate certificates if they don't exist
if [[ ! -f "server.crt" || ! -f "server.key" ]]; then
    echo "2. Generating SSL certificates..."
    ./generate-certs.sh > /dev/null 2>&1
    echo "✓ SSL certificates generated"
else
    echo "2. SSL certificates already exist"
    echo "✓ Using existing certificates"
fi
echo

# Test HTTP mode
echo "3. Testing HTTP mode..."
OUTPUTS=log SERVER_ADDR=:19890 timeout 3 ./gotrack &
SERVER_PID=$!
sleep 1

HTTP_RESULT=$(curl -s http://localhost:19890/healthz 2>/dev/null || echo "failed")
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

if [[ "$HTTP_RESULT" == "ok" ]]; then
    echo "✓ HTTP mode working"
else
    echo "✗ HTTP mode failed"
    exit 1
fi
echo

# Test HTTPS mode
echo "4. Testing HTTPS mode..."
ENABLE_HTTPS=true SSL_CERT_FILE=./server.crt SSL_KEY_FILE=./server.key OUTPUTS=log SERVER_ADDR=:19891 timeout 3 ./gotrack &
SERVER_PID=$!
sleep 1

HTTPS_RESULT=$(curl -k -s https://localhost:19891/healthz 2>/dev/null || echo "failed")
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

if [[ "$HTTPS_RESULT" == "ok" ]]; then
    echo "✓ HTTPS mode working"
else
    echo "✗ HTTPS mode failed"
    exit 1
fi
echo

echo "=== All tests passed! ==="
echo
echo "To run GoTrack with HTTPS:"
echo "  export ENABLE_HTTPS=true"
echo "  export SSL_CERT_FILE=./server.crt"  
echo "  export SSL_KEY_FILE=./server.key"
echo "  ./gotrack"
echo
echo "Environment variables for HTTPS:"
echo "  ENABLE_HTTPS     - Set to 'true' to enable HTTPS"
echo "  SSL_CERT_FILE    - Path to SSL certificate file" 
echo "  SSL_KEY_FILE     - Path to SSL private key file"