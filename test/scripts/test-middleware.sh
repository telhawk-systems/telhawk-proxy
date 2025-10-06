#!/bin/bash
# Comprehensive test script for GoTrack middleware mode

set -e

echo "=== GoTrack Middleware Mode Test ==="
echo

# Build the application
echo "1. Building GoTrack..."
go build -o ./gotrack ./cmd/gotrack
echo "✓ Build complete"
echo

# Test normal mode first
echo "2. Testing normal (non-middleware) mode..."
OUTPUTS=log SERVER_ADDR=:19892 timeout 3 ./gotrack &
SERVER_PID=$!
sleep 1

NORMAL_HEALTH=$(curl -s http://localhost:19892/healthz 2>/dev/null || echo "failed")
NORMAL_404=$(curl -s -w "%{http_code}" http://localhost:19892/nonexistent 2>/dev/null | tail -c 3)

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

if [[ "$NORMAL_HEALTH" == "ok" && "$NORMAL_404" == "404" ]]; then
    echo "✓ Normal mode working (health OK, 404 for unknown paths)"
else
    echo "✗ Normal mode failed"
    exit 1
fi
echo

# Start a destination server for proxy testing
echo "3. Starting destination server on port 8082..."
cd /tmp && python3 -m http.server 8082 &
DEST_PID=$!
cd - >/dev/null
sleep 2

# Test that destination server is working  
DEST_TEST=$(curl -s -w "%{http_code}" http://localhost:8082/ 2>/dev/null | tail -c 3)
if [[ "$DEST_TEST" != "200" ]]; then
    echo "✗ Destination server failed to start"
    kill $DEST_PID 2>/dev/null || true
    exit 1
fi
echo "✓ Destination server ready"
echo

# Test middleware mode
echo "4. Testing middleware mode..."
FORWARD_DESTINATION=http://localhost:8082 \
OUTPUTS=log \
SERVER_ADDR=:19893 \
timeout 5 ./gotrack &
MIDDLEWARE_PID=$!
sleep 2

# Test tracking endpoints (should be handled locally)
MIDDLEWARE_HEALTH=$(curl -s http://localhost:19893/healthz 2>/dev/null || echo "failed")
PIXEL_RESULT=$(curl -s -w "%{http_code}" http://localhost:19893/px.gif?e=test 2>/dev/null | tail -c 3)

# Test non-tracking endpoints (should be proxied)
PROXY_RESULT=$(curl -s -w "%{http_code}" http://localhost:19893/ 2>/dev/null | tail -c 3)
PROXY_404=$(curl -s -w "%{http_code}" http://localhost:19893/nonexistent 2>/dev/null | tail -c 3)

kill $MIDDLEWARE_PID 2>/dev/null || true
wait $MIDDLEWARE_PID 2>/dev/null || true

if [[ "$MIDDLEWARE_HEALTH" == "ok" ]]; then
    echo "✓ Middleware tracking endpoints working"
else
    echo "✗ Middleware tracking endpoints failed"
    kill $DEST_PID 2>/dev/null || true
    exit 1
fi

if [[ "$PIXEL_RESULT" == "200" ]]; then
    echo "✓ Pixel endpoint working"
else
    echo "✗ Pixel endpoint failed"
    kill $DEST_PID 2>/dev/null || true
    exit 1
fi

if [[ "$PROXY_RESULT" == "200" ]]; then
    echo "✓ Proxy functionality working"
else
    echo "✗ Proxy functionality failed"
    kill $DEST_PID 2>/dev/null || true
    exit 1
fi

if [[ "$PROXY_404" == "404" ]]; then
    echo "✓ Proxy 404 handling working"
else
    echo "✗ Proxy 404 handling failed"
    kill $DEST_PID 2>/dev/null || true
    exit 1
fi

# Clean up
kill $DEST_PID 2>/dev/null || true

echo
echo "=== All middleware tests passed! ==="
echo
echo "Configuration examples:"
echo
echo "Normal mode (default):"
echo "  ./gotrack"
echo
echo "Middleware mode:"
echo "  export FORWARD_DESTINATION=http://your-backend:3000"
echo "  ./gotrack"
echo
echo "Environment variables:"
echo "  FORWARD_DESTINATION   - Target server for non-tracking requests"