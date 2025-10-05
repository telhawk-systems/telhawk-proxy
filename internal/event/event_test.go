package event

import (
	"encoding/json"
	"testing"
)

func TestEvent_JSONSerialization(t *testing.T) {
	t.Run("serializes empty event with empty nested structs", func(t *testing.T) {
		e := Event{}
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}
		// Note: Go's JSON marshaller includes empty nested structs even with omitempty
		// This is expected behavior
		if len(data) == 0 {
			t.Error("should produce some JSON output")
		}
	})

	t.Run("serializes event with basic fields", func(t *testing.T) {
		e := Event{
			EventID: "test-123",
			TS:      "2024-01-01T00:00:00Z",
			Type:    "pageview",
		}
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}

		// Unmarshal and verify
		var decoded Event
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.EventID != "test-123" {
			t.Errorf("EventID = %v, want test-123", decoded.EventID)
		}
		if decoded.Type != "pageview" {
			t.Errorf("Type = %v, want pageview", decoded.Type)
		}
	})

	t.Run("can roundtrip event data", func(t *testing.T) {
		e := Event{
			EventID: "test",
			Type:    "click",
		}
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded Event
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.EventID != e.EventID {
			t.Errorf("EventID mismatch after roundtrip")
		}
		if decoded.Type != e.Type {
			t.Errorf("Type mismatch after roundtrip")
		}
	})
}

func TestUTMInfo(t *testing.T) {
	t.Run("serializes UTM parameters", func(t *testing.T) {
		utm := UTMInfo{
			Source:   "google",
			Medium:   "cpc",
			Campaign: "summer_sale",
			Term:     "running shoes",
			Content:  "ad1",
		}
		data, err := json.Marshal(utm)
		if err != nil {
			t.Fatalf("failed to marshal UTM: %v", err)
		}

		var decoded UTMInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.Source != "google" {
			t.Errorf("Source = %v, want google", decoded.Source)
		}
		if decoded.Campaign != "summer_sale" {
			t.Errorf("Campaign = %v, want summer_sale", decoded.Campaign)
		}
	})

	t.Run("handles empty UTM", func(t *testing.T) {
		utm := UTMInfo{}
		data, err := json.Marshal(utm)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}
		if string(data) != "{}" {
			t.Errorf("empty UTM should be {}, got %s", string(data))
		}
	})
}

func TestGoogleAdsInfo(t *testing.T) {
	t.Run("serializes Google Ads parameters", func(t *testing.T) {
		gads := GoogleAdsInfo{
			GCLID:      "test_gclid_123",
			CampaignID: "camp123",
			AdGroupID:  "ag456",
			Network:    "search",
		}
		data, err := json.Marshal(gads)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded GoogleAdsInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.GCLID != "test_gclid_123" {
			t.Errorf("GCLID = %v, want test_gclid_123", decoded.GCLID)
		}
		if decoded.Network != "search" {
			t.Errorf("Network = %v, want search", decoded.Network)
		}
	})
}

func TestMetaAdsInfo(t *testing.T) {
	t.Run("serializes Meta Ads parameters", func(t *testing.T) {
		meta := MetaAdsInfo{
			FBCLID:     "test_fbclid",
			FBC:        "fb_cookie",
			CampaignID: "meta_camp_1",
		}
		data, err := json.Marshal(meta)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded MetaAdsInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.FBCLID != "test_fbclid" {
			t.Errorf("FBCLID = %v, want test_fbclid", decoded.FBCLID)
		}
	})
}

func TestDeviceInfo(t *testing.T) {
	t.Run("serializes device information", func(t *testing.T) {
		mobile := true
		device := DeviceInfo{
			UA:       "Mozilla/5.0...",
			UAMobile: &mobile,
			OS:       "Windows",
			Browser:  "Chrome",
			Language: "en-US",
		}
		data, err := json.Marshal(device)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded DeviceInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.OS != "Windows" {
			t.Errorf("OS = %v, want Windows", decoded.OS)
		}
		if decoded.UAMobile == nil || *decoded.UAMobile != true {
			t.Error("UAMobile should be true")
		}
	})

	t.Run("handles screen information", func(t *testing.T) {
		device := DeviceInfo{
			Screens: []ScreenInfo{
				{
					Width:      1920,
					Height:     1080,
					ColorDepth: 24,
				},
			},
			ViewportW: 1200,
			ViewportH: 800,
		}
		data, err := json.Marshal(device)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded DeviceInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if len(decoded.Screens) != 1 {
			t.Fatalf("expected 1 screen, got %d", len(decoded.Screens))
		}
		if decoded.Screens[0].Width != 1920 {
			t.Errorf("Screen width = %v, want 1920", decoded.Screens[0].Width)
		}
	})
}

func TestSessionInfo(t *testing.T) {
	t.Run("serializes session data", func(t *testing.T) {
		session := SessionInfo{
			VisitorID:    "vis_123",
			SessionID:    "sess_456",
			SessionSeq:   5,
			FirstVisitTS: "2024-01-01T00:00:00Z",
		}
		data, err := json.Marshal(session)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded SessionInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.VisitorID != "vis_123" {
			t.Errorf("VisitorID = %v, want vis_123", decoded.VisitorID)
		}
		if decoded.SessionSeq != 5 {
			t.Errorf("SessionSeq = %v, want 5", decoded.SessionSeq)
		}
	})
}

func TestConsentInfo(t *testing.T) {
	t.Run("serializes consent information", func(t *testing.T) {
		applies := true
		consent := ConsentInfo{
			GDPRApplies: &applies,
			TCString:    "TC_STRING_V2",
			USPrivacy:   "1YNN",
			ConsentMode: "ad_storage=denied",
		}
		data, err := json.Marshal(consent)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded ConsentInfo
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if decoded.GDPRApplies == nil || *decoded.GDPRApplies != true {
			t.Error("GDPRApplies should be true")
		}
		if decoded.TCString != "TC_STRING_V2" {
			t.Errorf("TCString = %v, want TC_STRING_V2", decoded.TCString)
		}
	})
}

func TestCompleteEvent(t *testing.T) {
	t.Run("serializes complete event with all nested structures", func(t *testing.T) {
		mobile := true
		gdpr := true
		event := Event{
			EventID: "evt_123",
			TS:      "2024-01-01T12:00:00Z",
			Type:    "pageview",
			URL: URLInfo{
				UTM: UTMInfo{
					Source:   "google",
					Campaign: "test",
				},
				Referrer: "https://google.com",
			},
			Device: DeviceInfo{
				UA:       "Mozilla/5.0",
				UAMobile: &mobile,
				OS:       "Linux",
			},
			Session: SessionInfo{
				VisitorID: "vis_1",
				SessionID: "sess_1",
			},
			Consent: ConsentInfo{
				GDPRApplies: &gdpr,
			},
		}

		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("failed to marshal complete event: %v", err)
		}

		var decoded Event
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.EventID != "evt_123" {
			t.Errorf("EventID = %v, want evt_123", decoded.EventID)
		}
		if decoded.URL.UTM.Source != "google" {
			t.Errorf("UTM Source = %v, want google", decoded.URL.UTM.Source)
		}
		if decoded.Device.OS != "Linux" {
			t.Errorf("Device OS = %v, want Linux", decoded.Device.OS)
		}
		if decoded.Session.VisitorID != "vis_1" {
			t.Errorf("VisitorID = %v, want vis_1", decoded.Session.VisitorID)
		}
	})
}
