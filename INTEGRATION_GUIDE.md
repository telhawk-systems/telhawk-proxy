# GoTrack JS ↔ Go Integration Guide

This guide explains how to configure the JavaScript security pixel and Go application to work together seamlessly.

## 🏗️ Architecture Overview

```
┌─────────────────┐    HTTP POST /collect     ┌──────────────────┐
│  JS Pixel      │─────────────────────────►│  Go Server       │
│  (Browser)      │                          │  (:19890)        │
│                 │    HTTP GET /px.gif      │                  │
│                 │─────────────────────────►│                  │
└─────────────────┘         (fallback)       └──────────────────┘
                                                       │
                                                       ▼
                                              ┌──────────────────┐
                                              │  Event Sinks     │
                                              │  - Log Files     │
                                              │  - Kafka         │
                                              │  - PostgreSQL    │
                                              └──────────────────┘
```

## 🔧 Configuration

### Go Server Configuration

The Go server is configured via environment variables:

```bash
# Server settings
SERVER_ADDR=":19890"                    # Port to listen on
TRUST_PROXY=false                       # Honor X-Forwarded-For headers

# Output sinks (comma-separated)
OUTPUTS="log"                           # Available: log, kafka, postgres
LOG_PATH="./events.ndjson"              # Log file location

# Security
MAX_BODY_BYTES=1048576                  # 1MB max payload size
IP_HASH_SECRET=""                       # Optional IP hashing secret

# Testing
TEST_MODE=false                         # Generate test events on startup
```

### JavaScript Pixel Configuration

The JS pixel is configured when initializing:

```javascript
// Basic initialization
GoTrack.init();

// With custom endpoint
GoTrack.config.endpoint = "https://your-domain.com/collect";
GoTrack.init();

// With additional options
GoTrack.config = {
  endpoint: "https://analytics.yoursite.com/collect",
  version: 1
};
```

## 📊 Event Structure

The JS pixel sends events in the Go Event format:

```json
{
  "event_id": "evt_1696035000_abc123",
  "ts": "2025-09-29T13:40:00.000Z",
  "type": "pageview",
  "url": {
    "referrer": "https://google.com/search",
    "referrer_hostname": "google.com",
    "raw_query": "?utm_source=google&utm_medium=cpc"
  },
  "route": {
    "domain": "example.com",
    "path": "/products/item-123",
    "title": "Product Page - Item 123",
    "protocol": "https"
  },
  "device": {
    "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)...",
    "language": "en-US",
    "languages": ["en-US", "en"],
    "viewport_w": 1920,
    "viewport_h": 1080,
    "device_pixel_ratio": 2.0,
    "hardware_concurrency": 8,
    "cookie_enabled": true,
    "storage_available": true,
    "screens": [{
      "width": 1920,
      "height": 1080,
      "colorDepth": 24,
      "pixelDepth": 24
    }]
  },
  "session": {
    "session_id": "sess_1696035000_xyz789"
  },
  "server": {
    "bot_score": 0,
    "bot_reasons": []
  }
}
```

## 🚀 Deployment Options

### Option 1: Development Setup

For local development and testing:

```bash
# Terminal 1: Start Go server
cd /path/to/hello-go
OUTPUTS=log LOG_PATH=./events.ndjson SERVER_ADDR=":19890" ./gotrack

# Terminal 2: Serve test page (optional)
python3 -m http.server 8080
# Then visit: http://localhost:8080/test-pixel.html
```

### Option 2: Production Setup

For production deployment:

```bash
# Build Go binary
go build -o gotrack ./cmd/gotrack

# Run with production config
OUTPUTS="log,kafka,postgres" \
SERVER_ADDR=":19890" \
TRUST_PROXY=true \
KAFKA_BROKERS="kafka1:9092,kafka2:9092" \
KAFKA_TOPIC="analytics.events" \
PG_DSN="postgres://user:pass@db:5432/analytics" \
./gotrack
```

### Option 3: Docker Deployment

Use the provided Docker setup:

```bash
# Build and run with Docker Compose
docker-compose up --build

# Or build manually
docker build -t gotrack .
docker run -p 19890:19890 -e OUTPUTS=log gotrack
```

## 🔒 CORS Configuration

The Go server includes CORS headers for browser compatibility:

```go
// Applied automatically
Access-Control-Allow-Origin: *
Access-Control-Allow-Headers: Content-Type
Access-Control-Allow-Methods: GET, POST, OPTIONS
```

For production, consider restricting origins:

```go
// In production, modify cors() function to:
w.Header().Set("Access-Control-Allow-Origin", "https://yourdomain.com")
```

## 📡 Transport Methods

The JS pixel uses multiple transport methods with fallback:

1. **Primary**: `fetch()` with `keepalive: true`
2. **Fallback**: `navigator.sendBeacon()`  
3. **Final Fallback**: `<img>` pixel via `/px.gif`

Example transport configuration:

```javascript
// Primary method - JSON to /collect
fetch('/collect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(eventData),
  keepalive: true
});

// Fallback method - query params to /px.gif
const img = new Image(1, 1);
img.src = `/px.gif?e=pageview&sid=${sessionId}&ua=${encodeURIComponent(userAgent)}`;
```

## 🧪 Testing Integration

### 1. Manual Testing

```bash
# Test /collect endpoint
curl -X POST -H "Content-Type: application/json" \
  -d '{"event_id":"test","ts":"2025-09-29T12:00:00Z","type":"pageview","device":{"ua":"test"}}' \
  http://localhost:19890/collect

# Test /px.gif endpoint  
curl "http://localhost:19890/px.gif?e=pageview&url=https://example.com"

# Check server health
curl http://localhost:19890/healthz
```

### 2. Browser Testing

Open the test page: `test-pixel.html`

This page will:
- ✅ Initialize the pixel
- ✅ Send test events  
- ✅ Display collected environment data
- ✅ Test both transport methods
- ✅ Show server responses

### 3. Automated Testing

```bash
# Run Go tests
go test ./...

# Run JS tests (if available)
cd js && npm test
```

## 📊 Monitoring

### Server Logs

Monitor server activity:
```bash
# Watch request logs (stdout)
tail -f /var/log/gotrack/app.log

# Watch event logs (NDJSON)
tail -f ./events.ndjson | jq '.'
```

### Metrics

Access Prometheus metrics:
```bash
curl http://localhost:19890/metrics
```

Key metrics:
- `gotrack_requests_total{endpoint="/collect"}`
- `gotrack_requests_total{endpoint="/px.gif"}`
- `gotrack_events_processed_total`
- `gotrack_sink_errors_total`

## 🔍 Troubleshooting

### Common Issues

**1. CORS Errors**
```
Access to fetch at 'http://localhost:19890/collect' from origin 'http://localhost:8080' has been blocked by CORS policy
```
**Solution**: CORS is already configured in the Go server. Make sure the server is running and CORS middleware is applied.

**2. Content-Type Errors**
```
HTTP 415: content-type must be application/json
```
**Solution**: Ensure JS sends `Content-Type: application/json` header.

**3. JSON Parsing Errors**
```
HTTP 400: invalid json object
```
**Solution**: Verify the payload matches the Go Event structure.

**4. Connection Refused**
```
Failed to fetch: net::ERR_CONNECTION_REFUSED
```
**Solution**: Ensure Go server is running on the correct port (19890 by default).

### Debug Mode

Enable debug logging:
```bash
# Go server debug
LOG_LEVEL=debug ./gotrack

# Browser console
// Check for errors in browser dev tools
console.log('GoTrack debug:', window.GoTrack);
```

## 🚀 Next Steps

1. **Customize Bot Detection**: Modify `js/src/detect/` to add custom detection rules
2. **Add Authentication**: Implement API keys or JWT tokens for production
3. **Configure Sinks**: Set up Kafka/PostgreSQL for scalable event processing  
4. **Set up Monitoring**: Use Grafana dashboards for real-time analytics
5. **Deploy**: Use Docker or Kubernetes for production deployment

## 📚 Related Documentation

- [Go Server README](README.md)
- [JS Pixel Documentation](js/README.md) 
- [Deployment Guide](DEPLOYMENT.md)
- [Test Mode Guide](TESTMODE.md)

---

✅ **Integration Complete!** The JS security pixel and Go backend are now configured to work together seamlessly.