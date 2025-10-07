# Event Example - Complete Request Structure

This document shows what a fully populated GoTrack event looks like, including all possible HTTP headers, request payload, and server-enriched data.

---

## HTTP Request

### Request Line
```http
POST / HTTP/1.1
```

### Request Headers

#### Standard HTTP Headers
```http
Host: analytics.example.com
Content-Type: application/json
Content-Length: 2847
Accept: */*
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.9
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36
Referer: https://example.com/products/item-123?utm_source=google&utm_medium=cpc
Origin: https://example.com
Connection: keep-alive
Cache-Control: no-cache
```

#### GoTrack-Specific Headers
```http
X-GoTrack-HMAC: sha256=a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456
```

#### Proxy & CDN Headers (when TRUST_PROXY=true)
```http
X-Forwarded-For: 203.0.113.42, 198.51.100.1, 10.0.0.5
X-Real-IP: 203.0.113.42
X-Forwarded-Proto: https
X-Forwarded-Host: example.com
X-Forwarded-Port: 443
```

#### CloudFlare Headers (if behind CloudFlare CDN)
```http
CF-Ray: 84a1b2c3d4e5f678-ORD
CF-Connecting-IP: 203.0.113.42
CF-IPCountry: US
CF-Visitor: {"scheme":"https"}
CF-Request-ID: 0a1b2c3d4e5f6789
```

#### CloudFront Headers (if behind AWS CloudFront)
```http
CloudFront-Forwarded-Proto: https
CloudFront-Is-Desktop-Viewer: true
CloudFront-Is-Mobile-Viewer: false
CloudFront-Is-SmartTV-Viewer: false
CloudFront-Is-Tablet-Viewer: false
CloudFront-Viewer-Country: US
CloudFront-Viewer-Country-Region: CA
CloudFront-Viewer-City: San Francisco
CloudFront-Viewer-Latitude: 37.7749
CloudFront-Viewer-Longitude: -122.4194
```

#### Bot Detection Headers (from proxies/WAFs)
```http
X-Bot-Score: 0.15
X-Device-Type: desktop
```

#### Cookie Header
```http
Cookie: _gt_sid=sess_1234567890_abc123; _gt_vid=vis_9876543210_xyz789
```

---

## Request Body (JSON Payload)

```json
{
  "event_id": "evt_1735516800_a1b2c3d4e5",
  "ts": "2024-12-30T00:00:00.000Z",
  "type": "pageview",
  
  "url": {
    "utm": {
      "source": "google",
      "medium": "cpc",
      "campaign": "holiday_sale_2024",
      "term": "buy+shoes+online",
      "content": "textad1",
      "id": "utm_12345",
      "campaign_id": "67890"
    },
    "google": {
      "gclid": "CjwKCAiA1eKBhBZEiwAX3gglXYZ123456",
      "gclsrc": "aw.ds",
      "gbraid": "0AAAAAo1234567890",
      "wbraid": "0AAAAAp9876543210",
      "campaign_id": "12345678901",
      "ad_group_id": "98765432109",
      "ad_id": "567890123456",
      "keyword_id": "kwd-123456789",
      "matchtype": "e",
      "network": "g",
      "device": "c",
      "placement": "www.example.com"
    },
    "meta": {
      "fbclid": "IwAR1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7",
      "fbc": "fb.1.1234567890.IwAR1234567890",
      "fbp": "fb.1.1234567890123.1234567890",
      "campaign_id": "23851234567890123",
      "adset_id": "23851234567890124",
      "ad_id": "23851234567890125"
    },
    "microsoft": {
      "msclkid": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
    },
    "other_click_ids": {
      "ttclid": "tiktok_click_id_123456",
      "li_fat_id": "linkedin_click_id_789012",
      "epik": "pinterest_click_id_345678",
      "twclid": "twitter_click_id_901234",
      "sclid": "snapchat_click_id_567890"
    },
    "referrer": "https://www.google.com/search?q=buy+shoes+online",
    "referrer_hostname": "www.google.com",
    "raw_query": "?utm_source=google&utm_medium=cpc&utm_campaign=holiday_sale_2024&gclid=CjwKCAiA1eKBhBZEiwAX3gglXYZ123456",
    "query_size": 142
  },
  
  "route": {
    "domain": "example.com",
    "path": "/products/item-123",
    "fullPath": "/products/item-123?utm_source=google",
    "hash": "#reviews",
    "canonical_url": "https://example.com/products/item-123",
    "title": "Premium Running Shoes - Item 123 | Example Store",
    "protocol": "https",
    "query": {
      "utm_source": "google",
      "utm_medium": "cpc",
      "gclid": "CjwKCAiA1eKBhBZEiwAX3gglXYZ123456"
    }
  },
  
  "device": {
    "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "ua_brands": [
      "\"Not_A Brand\";v=\"8\"",
      "\"Chromium\";v=\"120\"",
      "\"Google Chrome\";v=\"120\""
    ],
    "ua_mobile": false,
    "os": "Windows",
    "browser": "Chrome",
    "language": "en-US",
    "languages": ["en-US", "en", "es"],
    "tz": "America/Los_Angeles",
    "tz_offset_minutes": -480,
    
    "viewport_w": 1920,
    "viewport_h": 937,
    "device_pixel_ratio": 2.0,
    
    "hardware_concurrency": 16,
    "device_memory": 8,
    "maxTouchPoints": 0,
    
    "prefers_color_scheme": "dark",
    "prefers_reduced_motion": false,
    
    "cookie_enabled": true,
    "storage_available": true,
    
    "network_effective_type": "4g",
    "network_downlink": 10.5,
    "network_rtt": 50,
    "network_save_data": false,
    
    "gpu": "ANGLE (NVIDIA, NVIDIA GeForce RTX 4090 (0x00002684) Direct3D11 vs_5_0 ps_5_0, D3D11)",
    "monitors": 2,
    "screens": [
      {
        "width": 3840,
        "height": 2160,
        "availWidth": 3840,
        "availHeight": 2120,
        "colorDepth": 24,
        "pixelDepth": 24
      },
      {
        "width": 1920,
        "height": 1080,
        "availWidth": 1920,
        "availHeight": 1040,
        "colorDepth": 24,
        "pixelDepth": 24
      }
    ]
  },
  
  "session": {
    "visitor_id": "vis_1734912000_xyz789abc",
    "session_id": "sess_1735516800_abc123xyz",
    "session_start_ts": "2024-12-30T00:00:00.000Z",
    "session_seq": 5,
    "first_visit_ts": "2024-12-23T14:30:00.000Z"
  },
  
  "server": {
    "ip_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "geo": {
      "country": "US",
      "region": "California",
      "city": "San Francisco",
      "postal_code": "94102",
      "latitude": "37.7749",
      "longitude": "-122.4194",
      "timezone": "America/Los_Angeles"
    },
    "detection": {
      "header_fingerprint": "h1:a1b2c3d4e5f6",
      "header_analysis": {
        "missing_expected": [],
        "automation_headers": [],
        "inconsistent_values": [],
        "header_order": [
          "Host",
          "Connection",
          "Content-Length",
          "User-Agent",
          "Content-Type",
          "Accept",
          "Origin",
          "Referer",
          "Accept-Encoding",
          "Accept-Language"
        ],
        "header_count": 10
      },
      "request_analysis": {
        "payload_entropy": 4.82,
        "request_size": 2847,
        "user_agent_analysis": {
          "length": 115,
          "contains_automation": false,
          "automation_keywords": [],
          "platform": "Windows",
          "browser": "Chrome"
        }
      },
      "timing_analysis": {
        "request_interval_ms": 3247.82,
        "interval_precision": 2,
        "requests_per_second": 0.31,
        "has_previous_request": true
      }
    }
  }
}
```

---

## Response

### Response Headers
```http
HTTP/1.1 200 OK
Content-Type: application/json
Content-Length: 27
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, OPTIONS
Date: Mon, 30 Dec 2024 00:00:00 GMT
Server: GoTrack/1.0
X-Request-ID: req_1735516800_def456
```

### Response Body
```json
{
  "status": "ok",
  "event_id": "evt_1735516800_a1b2c3d4e5"
}
```

---

## Alternative: Pixel Request (GET /px.gif)

For simple tracking without JavaScript, the pixel endpoint accepts URL parameters:

```http
GET /px.gif?e=pageview&url=https%3A%2F%2Fexample.com%2Fproducts%2Fitem-123&ref=https%3A%2F%2Fwww.google.com&utm_source=google&utm_medium=cpc&sid=sess_1735516800_abc123xyz&uid=vis_1734912000_xyz789abc HTTP/1.1
Host: analytics.example.com
User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36
Referer: https://example.com/products/item-123
Accept: image/webp,image/apng,image/*,*/*;q=0.8
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.9
```

Response:
```http
HTTP/1.1 200 OK
Content-Type: image/gif
Content-Length: 43
Cache-Control: no-store, no-cache, must-revalidate, private
Expires: 0
Pragma: no-cache
Access-Control-Allow-Origin: *

GIF89a...
```

---

## Notes

### Required Fields
Only a minimal subset is required for a valid event:
- `event_id` - Generated automatically if not provided
- `ts` - Generated automatically if not provided
- `type` - Defaults to "pageview" if not provided

All other fields are optional and enriched as available.

### Server-Side Enrichment
The following fields are added or enhanced by the GoTrack server:
- `server.ip_hash` - Hashed client IP (if `IP_HASH_SECRET` configured)
- `server.geo` - GeoIP lookup (if `GEOIP_DB` configured)
- `server.detection` - Bot detection signals from request analysis

### Privacy & Security
- IP addresses are hashed with a daily rotating salt when `IP_HASH_SECRET` is configured
- HMAC authentication prevents forged tracking data when `HMAC_SECRET` is configured
- Cookies are httpOnly when set by the server
- No PII (personally identifiable information) is collected by default

### CloudFlare & CloudFront Headers
When `TRUST_PROXY=true`, GoTrack will extract geolocation from CDN headers:
- CloudFlare: `CF-IPCountry`, etc.
- CloudFront: `CloudFront-Viewer-Country`, `CloudFront-Viewer-City`, etc.

**Note:** These headers are currently **not fully implemented** - see [MISSING_FEATURES.md](MISSING_FEATURES.md) for planned enhancements.

### Multiple Targets
Currently, GoTrack forwards requests to a single `FORWARD_DESTINATION`. Support for multiple relay targets is planned - see [MISSING_FEATURES.md](MISSING_FEATURES.md).

### Data Sinks
Events can be simultaneously written to multiple sinks:
- **Log** - NDJSON format to file (rotating logs supported)
- **Kafka** - High-throughput message queue with at-least-once delivery
- **PostgreSQL** - JSONB storage with GIN indexes for fast querying

Configure via `OUTPUTS` environment variable: `OUTPUTS=log,kafka,postgres`
