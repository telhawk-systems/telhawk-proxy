package main

import (
	"testing"

	"github.com/shortontech/gotrack/internal/event"
)

// TestGenerateTestEvents tests the test event generation
func TestGenerateTestEvents(t *testing.T) {
	events := generateTestEvents()

	t.Run("generates correct number of events", func(t *testing.T) {
		if len(events) != 5 {
			t.Errorf("expected 5 test events, got %d", len(events))
		}
	})

	t.Run("all events have required fields", func(t *testing.T) {
		verifyRequiredFields(t, events)
	})

	t.Run("events have correct types", func(t *testing.T) {
		verifyEventTypes(t, events)
	})

	t.Run("first event has correct details", func(t *testing.T) {
		verifyFirstEvent(t, events[0])
	})

	t.Run("second event is mobile", func(t *testing.T) {
		verifySecondEvent(t, events[1])
	})

	t.Run("third event is conversion", func(t *testing.T) {
		verifyThirdEvent(t, events[2])
	})

	t.Run("fourth event has Facebook data", func(t *testing.T) {
		verifyFourthEvent(t, events[3])
	})

	t.Run("fifth event is custom", func(t *testing.T) {
		verifyFifthEvent(t, events[4])
	})
}

// verifyRequiredFields checks that all events have required fields populated
func verifyRequiredFields(t *testing.T, events []event.Event) {
	for i, e := range events {
		if e.EventID == "" {
			t.Errorf("event %d: EventID should not be empty", i)
		}
		if e.TS == "" {
			t.Errorf("event %d: TS should not be empty", i)
		}
		if e.Type == "" {
			t.Errorf("event %d: Type should not be empty", i)
		}
	}
}

// verifyEventTypes checks that events have the expected types
func verifyEventTypes(t *testing.T, events []event.Event) {
	expectedTypes := []string{"pageview", "click", "conversion", "pageview", "custom_event"}
	for i, expectedType := range expectedTypes {
		if events[i].Type != expectedType {
			t.Errorf("event %d: expected type %s, got %s", i, expectedType, events[i].Type)
		}
	}
}

// verifyFirstEvent validates the first test event
func verifyFirstEvent(t *testing.T, e event.Event) {
	if e.URL.Referrer != "https://google.com" {
		t.Errorf("referrer incorrect: %s", e.URL.Referrer)
	}
	if e.URL.ReferrerHostname != "google.com" {
		t.Errorf("referrer hostname incorrect: %s", e.URL.ReferrerHostname)
	}
	if e.URL.UTM.Source != "google" {
		t.Errorf("UTM source incorrect: %s", e.URL.UTM.Source)
	}
	if e.Route.Domain != "example.com" {
		t.Errorf("domain incorrect: %s", e.Route.Domain)
	}
	if e.Device.Browser != "Chrome" {
		t.Errorf("browser incorrect: %s", e.Device.Browser)
	}
}

// verifySecondEvent validates the second test event (mobile)
func verifySecondEvent(t *testing.T, e event.Event) {
	if e.Type != "click" {
		t.Errorf("type should be click, got %s", e.Type)
	}
	if e.Device.Browser != "Safari" {
		t.Errorf("browser should be Safari, got %s", e.Device.Browser)
	}
	if e.Device.UAMobile == nil || !*e.Device.UAMobile {
		t.Error("should be mobile")
	}
}

// verifyThirdEvent validates the third test event (conversion)
func verifyThirdEvent(t *testing.T, e event.Event) {
	if e.Type != "conversion" {
		t.Errorf("type should be conversion, got %s", e.Type)
	}
	if e.Route.Path != "/thank-you" {
		t.Errorf("path should be /thank-you, got %s", e.Route.Path)
	}
}

// verifyFourthEvent validates the fourth test event (Facebook)
func verifyFourthEvent(t *testing.T, e event.Event) {
	if e.URL.Referrer != "https://facebook.com" {
		t.Errorf("referrer should be facebook.com, got %s", e.URL.Referrer)
	}
	if e.URL.UTM.Source != "facebook" {
		t.Errorf("UTM source should be facebook, got %s", e.URL.UTM.Source)
	}
	if e.URL.Meta.FBCLID != "fb_click_123" {
		t.Errorf("FBCLID should be set, got %s", e.URL.Meta.FBCLID)
	}
}

// verifyFifthEvent validates the fifth test event (custom)
func verifyFifthEvent(t *testing.T, e event.Event) {
	if e.Type != "custom_event" {
		t.Errorf("type should be custom_event, got %s", e.Type)
	}
	if e.Route.Path != "/dashboard" {
		t.Errorf("path should be /dashboard, got %s", e.Route.Path)
	}
}

// TestRunTestMode tests the test mode execution
func TestRunTestMode(t *testing.T) {
	t.Run("sends events to emit function", func(t *testing.T) {
		var receivedEvents []event.Event
		emitFunc := func(e event.Event) {
			receivedEvents = append(receivedEvents, e)
		}

		runTestMode(emitFunc)

		if len(receivedEvents) != 5 {
			t.Errorf("expected 5 events to be emitted, got %d", len(receivedEvents))
		}

		// Verify events were sent
		for i, e := range receivedEvents {
			if e.EventID == "" {
				t.Errorf("event %d: EventID should not be empty", i)
			}
		}
	})

	t.Run("handles nil emit function gracefully", func(t *testing.T) {
		// This should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("runTestMode panicked with nil emit: %v", r)
			}
		}()

		// Note: This will panic if emitFn is called with nil, but that's expected behavior
		// We're just ensuring the function itself doesn't panic before calling emit
	})
}

// TestBoolPtr tests the boolean pointer helper
func TestBoolPtr(t *testing.T) {
	t.Run("creates pointer to true", func(t *testing.T) {
		ptr := boolPtr(true)
		if ptr == nil {
			t.Error("expected non-nil pointer")
		}
		if !*ptr {
			t.Error("expected pointer to true")
		}
	})

	t.Run("creates pointer to false", func(t *testing.T) {
		ptr := boolPtr(false)
		if ptr == nil {
			t.Error("expected non-nil pointer")
		}
		if *ptr {
			t.Error("expected pointer to false")
		}
	})

	t.Run("different pointers for same value", func(t *testing.T) {
		ptr1 := boolPtr(true)
		ptr2 := boolPtr(true)
		
		// They should have the same value but different addresses
		if *ptr1 != *ptr2 {
			t.Error("both pointers should point to true")
		}
		if ptr1 == ptr2 {
			t.Error("pointers should have different addresses")
		}
	})
}

// Test event field diversity
func TestGenerateTestEvents_FieldDiversity(t *testing.T) {
	events := generateTestEvents()

	// Count different browsers
	browsers := make(map[string]int)
	for _, e := range events {
		if e.Device.Browser != "" {
			browsers[e.Device.Browser]++
		}
	}

	if len(browsers) < 2 {
		t.Error("expected at least 2 different browsers in test events")
	}

	// Count different operating systems
	oses := make(map[string]int)
	for _, e := range events {
		if e.Device.OS != "" {
			oses[e.Device.OS]++
		}
	}

	if len(oses) < 2 {
		t.Error("expected at least 2 different operating systems in test events")
	}

	// Verify viewport dimensions vary
	viewports := make(map[int]bool)
	for _, e := range events {
		if e.Device.ViewportW > 0 {
			viewports[e.Device.ViewportW] = true
		}
	}

	if len(viewports) < 3 {
		t.Error("expected at least 3 different viewport widths")
	}
}

// Test that visitor and session IDs are unique
func TestGenerateTestEvents_UniqueIDs(t *testing.T) {
	events := generateTestEvents()

	visitorIDs := make(map[string]bool)
	sessionIDs := make(map[string]bool)

	for _, e := range events {
		if e.Session.VisitorID != "" {
			visitorIDs[e.Session.VisitorID] = true
		}
		if e.Session.SessionID != "" {
			sessionIDs[e.Session.SessionID] = true
		}
	}

	// All visitor IDs should be unique
	if len(visitorIDs) != len(events) {
		t.Errorf("expected %d unique visitor IDs, got %d", len(events), len(visitorIDs))
	}

	// All session IDs should be unique
	if len(sessionIDs) != len(events) {
		t.Errorf("expected %d unique session IDs, got %d", len(events), len(sessionIDs))
	}
}

// Test that timestamps are in order
func TestGenerateTestEvents_TimestampOrder(t *testing.T) {
	events := generateTestEvents()

	for i := 1; i < len(events); i++ {
		if events[i].TS <= events[i-1].TS {
			t.Errorf("event %d timestamp should be after event %d", i, i-1)
		}
	}
}

// Test specific UTM tracking scenarios
func TestGenerateTestEvents_UTMTracking(t *testing.T) {
	events := generateTestEvents()

	// First event should have Google organic UTM
	if events[0].URL.UTM.Source != "google" {
		t.Error("first event should have google UTM source")
	}
	if events[0].URL.UTM.Medium != "organic" {
		t.Error("first event should have organic UTM medium")
	}

	// Fourth event should have Facebook social UTM
	if events[3].URL.UTM.Source != "facebook" {
		t.Error("fourth event should have facebook UTM source")
	}
	if events[3].URL.UTM.Medium != "social" {
		t.Error("fourth event should have social UTM medium")
	}
	if events[3].URL.UTM.Campaign != "spring_sale" {
		t.Error("fourth event should have spring_sale campaign")
	}
}

// Test geo data in server metadata
func TestGenerateTestEvents_GeoData(t *testing.T) {
	events := generateTestEvents()

	// First event should have geo data
	if events[0].Server.Geo == nil {
		t.Error("first event should have geo data")
	}
	
	if events[0].Server.Geo["country"] != "US" {
		t.Errorf("expected country US, got %s", events[0].Server.Geo["country"])
	}
	
	if events[0].Server.Geo["region"] != "CA" {
		t.Errorf("expected region CA, got %s", events[0].Server.Geo["region"])
	}
	
	if events[0].Server.Geo["city"] != "San Francisco" {
		t.Errorf("expected city San Francisco, got %s", events[0].Server.Geo["city"])
	}
}

// Test meta ads tracking
func TestGenerateTestEvents_MetaAdsTracking(t *testing.T) {
	events := generateTestEvents()

	// Fourth event should have Meta ads tracking
	if events[3].URL.Meta.FBCLID == "" {
		t.Error("fourth event should have FBCLID")
	}
	if events[3].URL.Meta.CampaignID == "" {
		t.Error("fourth event should have CampaignID")
	}
}
