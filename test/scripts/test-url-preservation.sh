#!/bin/bash
# Test URL parameter preservation in pixel injection

set -e

echo "=== GoTrack URL Parameter Preservation Test ==="
echo

# Build GoTrack
echo "1. Building GoTrack..."
go build -o ./gotrack ./cmd/gotrack
echo "✓ Build complete"
echo

# Start test server
echo "2. Starting test server..."
cat > /tmp/url_test_server.py << 'EOF'
import http.server
import socketserver

class TestHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith('/test'):
            self.send_response(200)
            self.send_header('Content-Type', 'text/html; charset=utf-8')
            self.end_headers()
            html = '''<!DOCTYPE html>
<html>
<head><title>URL Test</title></head>
<body><h1>URL Parameter Test</h1></body>
</html>'''
            self.wfile.write(html.encode())
        else:
            self.send_error(404)

with socketserver.TCPServer(("", 8091), TestHandler) as httpd:
    httpd.serve_forever()
EOF

cd /tmp && python3 url_test_server.py &
TEST_SERVER_PID=$!
cd - >/dev/null
sleep 2
echo "✓ Test server started"
echo

# Start GoTrack
echo "3. Starting GoTrack with auto-injection..."
FORWARD_DESTINATION=http://localhost:8091 \
OUTPUTS=log \
SERVER_ADDR=:19911 \
timeout 10 ./gotrack &
GOTRACK_PID=$!
sleep 2

# Test cases
echo "4. Testing URL parameter preservation..."

# Test 1: Simple UTM parameters
echo "   Test 1: UTM parameters"
RESPONSE=$(curl -s "http://localhost:19911/test.html?utm_source=google&utm_medium=cpc&utm_campaign=summer")
PIXEL_URL=$(echo "$RESPONSE" | grep -o 'px\.gif[^"]*')
DECODED=$(python3 -c "
import urllib.parse
url = '$PIXEL_URL'
params = urllib.parse.parse_qs(url.split('?', 1)[1])
print(urllib.parse.unquote(params['url'][0]))
")

if [[ "$DECODED" == *"utm_source=google"* && "$DECODED" == *"utm_medium=cpc"* && "$DECODED" == *"utm_campaign=summer"* ]]; then
    echo "   ✓ UTM parameters preserved"
else
    echo "   ✗ UTM parameters missing: $DECODED"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
fi

# Test 2: Complex parameters with special characters
echo "   Test 2: Complex parameters"
RESPONSE=$(curl -s "http://localhost:19911/test.html?q=hello%20world&ref=site.com%2Fpage&id=123&debug=true")
PIXEL_URL=$(echo "$RESPONSE" | grep -o 'px\.gif[^"]*')
DECODED=$(python3 -c "
import urllib.parse
url = '$PIXEL_URL'
params = urllib.parse.parse_qs(url.split('?', 1)[1])
print(urllib.parse.unquote(params['url'][0]))
")

if [[ "$DECODED" == *"q=hello world"* && "$DECODED" == *"ref=site.com/page"* && "$DECODED" == *"id=123"* ]]; then
    echo "   ✓ Complex parameters preserved"
else
    echo "   ✗ Complex parameters missing: $DECODED"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
fi

# Test 3: No parameters (should work without errors)
echo "   Test 3: No parameters"
RESPONSE=$(curl -s "http://localhost:19911/test.html")
PIXEL_URL=$(echo "$RESPONSE" | grep -o 'px\.gif[^"]*')
DECODED=$(python3 -c "
import urllib.parse
url = '$PIXEL_URL'
params = urllib.parse.parse_qs(url.split('?', 1)[1])
print(urllib.parse.unquote(params['url'][0]))
")

if [[ "$DECODED" == "/test.html" ]]; then
    echo "   ✓ No parameters handled correctly"
else
    echo "   ✗ No parameters case failed: $DECODED"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
fi

# Test 4: Mixed tracking parameters
echo "   Test 4: Mixed tracking parameters"
RESPONSE=$(curl -s "http://localhost:19911/test.html?utm_source=facebook&fbclid=ABC123&gclid=XYZ789&msclkid=DEF456")
PIXEL_URL=$(echo "$RESPONSE" | grep -o 'px\.gif[^"]*')
DECODED=$(python3 -c "
import urllib.parse
url = '$PIXEL_URL'
params = urllib.parse.parse_qs(url.split('?', 1)[1])
print(urllib.parse.unquote(params['url'][0]))
")

if [[ "$DECODED" == *"fbclid=ABC123"* && "$DECODED" == *"gclid=XYZ789"* && "$DECODED" == *"msclkid=DEF456"* ]]; then
    echo "   ✓ Mixed tracking parameters preserved"
else
    echo "   ✗ Mixed tracking parameters missing: $DECODED"
    kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
    exit 1
fi

# Clean up
kill $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
wait $GOTRACK_PID $TEST_SERVER_PID 2>/dev/null || true
rm -f /tmp/url_test_server.py

echo
echo "=== All URL parameter preservation tests passed! ==="
echo
echo "Summary:"
echo "✓ UTM parameters (utm_source, utm_medium, utm_campaign) preserved"
echo "✓ Click IDs (fbclid, gclid, msclkid) preserved"
echo "✓ Complex parameters with special characters preserved"
echo "✓ URLs without parameters handled correctly"
echo "✓ Full query string passed to pixel tracking endpoint"
echo
echo "Bug fix verified: Auto-injected pixels now capture complete URLs!"