# Future Enhancements

This document outlines planned features and improvements for the GoTrack analytics platform.

## Configuration Management

### YAML Configuration Support
Add support for YAML-based configuration files to complement environment variables. This will enable more complex configuration scenarios while maintaining backward compatibility with the existing environment variable approach.

**Priority:** Medium  
**Status:** Planned

## Network & Proxy Configuration

### Trusted Proxy Configuration
Implement configurable IP range whitelisting to specify which proxies are trusted to forward traffic. This should support:
- Multiple IP ranges/CIDR blocks
- Option to trust all proxies when behind a secure firewall
- Configurable behavior for untrusted proxies

**Priority:** High  
**Status:** Planned

### Multi-Proxy Header Support
Extend support for various proxy headers used by different CDN providers:
- CloudFlare: `CF-Connecting-IP`
- CloudFront: `CloudFront-Viewer-Address`
- Standard: `X-Forwarded-For`, `X-Real-IP`
- Configurable priority order for header resolution

**Priority:** High  
**Status:** Planned

### Multiple Relay Targets
Support forwarding traffic to multiple downstream targets simultaneously, enabling:
- A/B testing scenarios
- Multi-environment deployments
- Backup/failover configurations

**Priority:** Medium  
**Status:** Planned

## Data Collection Enhancements

### Geolocation Headers
Capture and log geolocation headers provided by CDN providers (CloudFlare, CloudFront) to enrich request metadata without requiring separate geolocation services.

**Priority:** Medium  
**Status:** Planned

### Simplified Payload Structure
Refactor the event payload structure to follow a client-agnostic model where the JavaScript client reports raw telemetry data without interpretation:

```json
{
  "results": {
    "sound_channels": 4,
    "gpu": "NVIDIA GeForce RTX 3080",
    "window_width": 1920,
    "window_height": 1080,
    "webgl_vendor": "NVIDIA Corporation",
    "webgl_renderer": "NVIDIA GeForce RTX 3080/PCIe/SSE2"
  }
}
```

This approach:
- Treats the client as a minimal data collection agent
- Presents as general telemetry rather than bot detection
- Enables flexible server-side analysis
- Supports arbitrary nesting and custom metrics

**Priority:** High  
**Status:** Planned

### Remove Deprecated Fields
Remove or deprecate fields that provide limited value:
- `header_fingerprint` - Headers vary per request, making fingerprinting unreliable
- `payload_entropy` - Provides minimal actionable insights
- `request_size` - Redundant information already available in logs
- `user_agent_analysis` - Replace with raw User-Agent storage for downstream processing

**Priority:** Medium  
**Status:** Planned

## Client-Side Detection

### Ad Blocker Detection via /px.gif
Utilize the pixel endpoint (`/px.gif`) primarily for ad blocker detection. Successful pixel loads can serve as a positive signal for legitimate traffic, as most bots don't employ ad blocking.

**Priority:** Medium  
**Status:** Planned

### Simplified Collection Endpoint
When not operating in middleware mode, hardcode the collection endpoint to `location.href` rather than allowing configuration. This simplifies the client implementation and reduces attack surface.

**Priority:** Low  
**Status:** Planned

## IP Intelligence

### Port Scanning
Perform lightweight port checks on connecting IPs to detect common service ports:
- Port 22 (SSH)
- Port 80 (HTTP)
- Port 443 (HTTPS)
- Port 8080 (HTTP Alternate)
- Port 8081 (HTTP Alternate)

Open ports on client IPs may indicate hosting infrastructure or proxies rather than end-user devices.

**Priority:** Low  
**Status:** Research Phase

### Reverse DNS Resolution
Capture and store reverse DNS information up to the second-level domain (e.g., `google.com`). This provides context about the origin of requests without requiring external APIs.

**Example:** `208.79.209.138` resolves to `whatsmyip.org`

**Priority:** Medium  
**Status:** Planned

### WHOIS Integration
Integrate with commercial WHOIS data providers to enrich IP information with ownership and registration data. This is valuable for security analysis but requires careful implementation due to:
- Rate limiting and terms of service restrictions
- Commercial licensing requirements
- Cost considerations (e.g., WhoisXMLAPI: $30/2,000 requests)

Recommended providers:
- WhoisXMLAPI
- DomainTools
- IPinfo
- IPWHOIS

Implementation should support:
- Multiple provider backends
- Configurable fallbacks
- Caching to minimize API calls
- Direct database storage for enrichment data

**Priority:** Low  
**Status:** Research Phase  
**Notes:** Requires commercial agreements and careful rate limit management

### IP Reputation Services
Integrate with IP reputation services to identify known malicious sources, proxies, VPNs, and hosting providers. This provides immediate context for traffic analysis and anomaly detection.

**Priority:** Medium  
**Status:** Planned

## Headless Browser Detection

### Fingerprinting Research
Conduct testing against common headless browser evasion tools to identify detection patterns:
- HeadlessX
- Lightpanda
- Playwright with stealth plugins
- undetected-chromedriver
- puppeteer-extra-plugin-stealth

Goal is to document behavioral quirks and detection vectors that can be implemented in the client-side telemetry.

**Priority:** Medium  
**Status:** Research Phase

---

**Note:** Priorities and timelines are subject to change based on user feedback and operational requirements.