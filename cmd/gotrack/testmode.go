package main

import (
	"log"
	"time"

	"github.com/google/uuid"
	"revinar.io/go.track/internal/event"
)

// generateTestEvents creates sample events for testing sinks
func generateTestEvents() []event.Event {
	now := time.Now()

	events := []event.Event{
		{
			EventID: uuid.New().String(),
			TS:      now.Format(time.RFC3339),
			Type:    "pageview",
			URL: event.URLInfo{
				Referrer:         "https://google.com",
				ReferrerHostname: "google.com",
				UTM: event.UTMInfo{
					Source:   "google",
					Medium:   "organic",
					Campaign: "search",
				},
			},
			Route: event.RouteInfo{
				Domain:   "example.com",
				Path:     "/home",
				FullPath: "/home?utm_source=google",
				Title:    "Home Page",
				Protocol: "https",
			},
			Device: event.DeviceInfo{
				UA:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				Browser:   "Chrome",
				OS:        "Windows",
				Language:  "en-US",
				ViewportW: 1920,
				ViewportH: 1080,
			},
			Session: event.SessionInfo{
				VisitorID:    "visitor-" + uuid.New().String()[:8],
				SessionID:    "session-" + uuid.New().String()[:8],
				SessionStart: now.Add(-5 * time.Minute).Format(time.RFC3339),
				SessionSeq:   1,
			},
			Server: event.ServerMeta{
				IP: "203.0.113.42",
				Geo: map[string]string{
					"country": "US",
					"region":  "CA",
					"city":    "San Francisco",
				},
			},
		},
		{
			EventID: uuid.New().String(),
			TS:      now.Add(1 * time.Second).Format(time.RFC3339),
			Type:    "click",
			URL: event.URLInfo{
				Referrer: "https://example.com/home",
			},
			Route: event.RouteInfo{
				Domain:   "example.com",
				Path:     "/signup",
				FullPath: "/signup",
				Title:    "Sign Up",
				Protocol: "https",
			},
			Device: event.DeviceInfo{
				UA:        "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)",
				Browser:   "Safari",
				OS:        "iOS",
				UAMobile:  boolPtr(true),
				ViewportW: 375,
				ViewportH: 812,
			},
			Session: event.SessionInfo{
				VisitorID: "visitor-" + uuid.New().String()[:8],
				SessionID: "session-" + uuid.New().String()[:8],
			},
		},
		{
			EventID: uuid.New().String(),
			TS:      now.Add(2 * time.Second).Format(time.RFC3339),
			Type:    "conversion",
			Route: event.RouteInfo{
				Domain: "example.com",
				Path:   "/thank-you",
				Title:  "Thank You",
			},
			Device: event.DeviceInfo{
				UA:        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
				Browser:   "Firefox",
				OS:        "Linux",
				ViewportW: 1366,
				ViewportH: 768,
			},
			Session: event.SessionInfo{
				VisitorID: "visitor-" + uuid.New().String()[:8],
				SessionID: "session-" + uuid.New().String()[:8],
			},
		},
		{
			EventID: uuid.New().String(),
			TS:      now.Add(3 * time.Second).Format(time.RFC3339),
			Type:    "pageview",
			URL: event.URLInfo{
				Referrer: "https://facebook.com",
				UTM: event.UTMInfo{
					Source:   "facebook",
					Medium:   "social",
					Campaign: "spring_sale",
					Content:  "post_123",
				},
				Meta: event.MetaAdsInfo{
					FBCLID:     "fb_click_123",
					CampaignID: "camp_456",
				},
			},
			Route: event.RouteInfo{
				Domain: "example.com",
				Path:   "/products",
				Title:  "Products",
			},
			Device: event.DeviceInfo{
				UA:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
				Browser:   "Safari",
				OS:        "macOS",
				ViewportW: 1440,
				ViewportH: 900,
			},
			Session: event.SessionInfo{
				VisitorID: "visitor-" + uuid.New().String()[:8],
				SessionID: "session-" + uuid.New().String()[:8],
			},
		},
		{
			EventID: uuid.New().String(),
			TS:      now.Add(4 * time.Second).Format(time.RFC3339),
			Type:    "custom_event",
			Route: event.RouteInfo{
				Domain: "example.com",
				Path:   "/dashboard",
				Title:  "Dashboard",
			},
			Device: event.DeviceInfo{
				UA:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:91.0)",
				Browser:   "Firefox",
				OS:        "Windows",
				ViewportW: 1920,
				ViewportH: 1080,
			},
			Session: event.SessionInfo{
				VisitorID: "visitor-" + uuid.New().String()[:8],
				SessionID: "session-" + uuid.New().String()[:8],
			},
		},
	}

	return events
}

// runTestMode generates and sends test events
func runTestMode(emitFn func(event.Event)) {
	log.Println("🧪 TEST MODE: Generating test events...")

	events := generateTestEvents()

	for i, e := range events {
		log.Printf("📊 Sending test event %d/%d: %s (%s)", i+1, len(events), e.Type, e.EventID)
		emitFn(e)

		// Small delay between events to see them clearly in logs
		if i < len(events)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	log.Println("✅ TEST MODE: All test events sent!")
	log.Println("💡 Check your sinks:")
	log.Println("   - Log files: tail -f out/events.ndjson")
	log.Println("   - Kafka: ./deploy/manage.sh kafka-console")
	log.Println("   - PostgreSQL: ./deploy/manage.sh psql")
}

// Helper function for bool pointers
func boolPtr(b bool) *bool {
	return &b
}
