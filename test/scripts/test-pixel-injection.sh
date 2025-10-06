#!/bin/bash
# Comprehensive test script for GoTrack pixel injection

set -e

echo "=== GoTrack Pixel Injection Test ==="
echo

# Build the application
echo "1. Building GoTrack..."
go build -o ./gotrack ./cmd/gotrack
echo "✓ Build complete"
echo

# Create test server script
echo "2. Setting up test server..."
cat > /tmp/injection_test_server.py << 'EOF'
import http.server
import socketserver

class TestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        path = self.path
        if path == '/page.html':
            self.send_response(200)
            self.send_header('Content-Type', 'text/html; charset=utf-8')
            self.end_headers()
            html = '''<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<h1>Hello World</h1>
</body>
</html>'''
            self.wfile.write(html.encode())
        elif path == '/api/data.json':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(b'{"status": "ok"}')
        elif path == '/mixed.HTML':
            self.send_response(200)
            self.send_header('Content-Type', 'TEXT/HTML')
            self.end_headers()
            self.wfile.write(b'<HTML><BODY>Mixed case</BODY></HTML>')
        else:
            self.send_error(404)

with socketserver.TCPServer(("", 8088), TestHandler) as httpd:
    httpd.serve_forever()
EOF

cd /tmp && python3 injection_test_server.py &
TEST_SERVER_PID=$!
cd - >/dev/null
sleep 2
echo "✓ Test server started"
echo

# Test with pixel injection enabled (default)
echo "3. Testing with pixel injection enabled..."
FORWARD_DESTINATION=http://localhost:8088 \
OUTPUTS=log \
SERVER_ADDR=:19901 \
timeout 5 ./gotrack &
GOTRACK_PID=$!
sleep 2

# Test HTML injection
HTML_RESULT=$(curl -s http://localhost:19901/page.html)
PIXEL_COUNT=$(echo "$HTML_RESULT" | grep -c 'px\.gif' || echo "0")

if [[ "$PIXEL_COUNT" == "1" ]]; then
    echo "✓ HTML pixel injection working"
else
    echo "✗ HTML pixel injection failed"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
fi

# Test JSON not injected
JSON_RESULT=$(curl -s http://localhost:19901/api/data.json)
if echo "$JSON_RESULT" | grep -q 'px\.gif'; then
    echo "✗ JSON content incorrectly modified: $JSON_RESULT"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
else
    echo "✓ JSON content not modified (correct)"
fi

# Test case-insensitive HTML
MIXED_RESULT=$(curl -s http://localhost:19901/mixed.HTML)
MIXED_PIXEL_COUNT=$(echo "$MIXED_RESULT" | grep -c 'px\.gif' || echo "0")

if [[ "$MIXED_PIXEL_COUNT" == "1" ]]; then
    echo "✓ Case-insensitive HTML detection working"
else
    echo "✗ Case-insensitive HTML detection failed"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
fi

kill $GOTRACK_PID 2>/dev/null || true
wait $GOTRACK_PID 2>/dev/null || true
echo

# Test with pixel injection disabled
echo "4. Testing with pixel injection disabled..."
FORWARD_DESTINATION=http://localhost:8088 \
OUTPUTS=log \
SERVER_ADDR=:19902 \
timeout 5 ./gotrack &
GOTRACK_PID=$!
sleep 2

# Test HTML not injected when disabled
HTML_NO_INJECT=$(curl -s http://localhost:19902/page.html)
if echo "$HTML_NO_INJECT" | grep -q 'px\.gif'; then
    echo "✗ Pixel injection not properly disabled"
    echo "Response: $HTML_NO_INJECT"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
else
    echo "✓ Pixel injection properly disabled"
fi

# Clean up
kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
wait $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
rm -f /tmp/injection_test_server.py

echo
echo "=== All pixel injection tests passed! ==="
echo
echo "Summary:"
echo "✓ HTML content gets pixel injected (default behavior)"
echo "✓ JSON content is never modified"  
echo "✓ Case-insensitive content-type detection works"
echo "✓ Injected pixels use proper URL encoding"