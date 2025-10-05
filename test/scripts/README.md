# Test Scripts

This directory contains manual test scripts and HTML test pages for testing various features of gotrack.

## Shell Scripts

- **test-hmac.sh** - Tests HMAC authentication functionality
- **test-https.sh** - Tests HTTPS/TLS configuration
- **test-integration.sh** - Integration tests for the full system
- **test-middleware.sh** - Tests middleware proxy functionality
- **test-pixel-injection.sh** - Tests automatic pixel injection into HTML
- **test-url-preservation.sh** - Tests URL parameter preservation

## HTML Test Pages

- **test-enhanced-detection.html** - Tests enhanced bot/crawler detection
- **test-pixel.html** - Basic pixel tracking test page

## Usage

Make sure the gotrack server is running before executing these scripts:

```bash
# Start the server (from project root)
go run ./cmd/gotrack

# Run a test script
./test/scripts/test-integration.sh
```

Most scripts require curl and will test against `http://localhost:19890` by default.
