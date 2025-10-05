package event

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"revinar.io/go.track/pkg/config"
)

func TestEnrichServerFields(t *testing.T) {
	t.Run("sets timestamp when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		e := &Event{}
		cfg := config.Config{}

		before := time.Now().UTC()
		EnrichServerFields(req, e, cfg)
		after := time.Now().UTC()

		if e.TS == "" {
			t.Error("timestamp should be set")
		}

		// Parse and verify timestamp is within reasonable range
		ts, err := time.Parse(time.RFC3339Nano, e.TS)
		if err != nil {
			t.Errorf("timestamp should be valid RFC3339Nano: %v", err)
		}
		if ts.Before(before) || ts.After(after.Add(time.Second)) {
			t.Errorf("timestamp %v should be between %v and %v", ts, before, after)
		}
	})

	t.Run("preserves existing timestamp", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		existingTS := "2024-01-01T12:00:00Z"
		e := &Event{TS: existingTS}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.TS != existingTS {
			t.Errorf("timestamp = %v, want %v", e.TS, existingTS)
		}
	})

	t.Run("sets default type to pageview when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		e := &Event{}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.Type != "pageview" {
			t.Errorf("type = %v, want pageview", e.Type)
		}
	})

	t.Run("preserves existing event type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		e := &Event{Type: "click"}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.Type != "click" {
			t.Errorf("type = %v, want click", e.Type)
		}
	})

	t.Run("extracts user agent when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 Test Browser")
		e := &Event{}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.Device.UA != "Mozilla/5.0 Test Browser" {
			t.Errorf("UA = %v, want Mozilla/5.0 Test Browser", e.Device.UA)
		}
	})

	t.Run("preserves existing user agent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 Test Browser")
		e := &Event{Device: DeviceInfo{UA: "Client UA"}}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.Device.UA != "Client UA" {
			t.Errorf("UA = %v, want Client UA", e.Device.UA)
		}
	})

	t.Run("extracts referrer when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.Header.Set("Referer", "https://google.com/search")
		e := &Event{}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.URL.Referrer != "https://google.com/search" {
			t.Errorf("Referrer = %v, want https://google.com/search", e.URL.Referrer)
		}
		if e.URL.ReferrerHostname != "google.com" {
			t.Errorf("ReferrerHostname = %v, want google.com", e.URL.ReferrerHostname)
		}
	})

	t.Run("preserves existing referrer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.Header.Set("Referer", "https://google.com")
		e := &Event{URL: URLInfo{Referrer: "https://existing.com"}}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.URL.Referrer != "https://existing.com" {
			t.Errorf("Referrer = %v, want https://existing.com", e.URL.Referrer)
		}
	})

	t.Run("extracts raw query when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect?foo=bar&baz=qux", nil)
		e := &Event{}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.URL.RawQuery != "foo=bar&baz=qux" {
			t.Errorf("RawQuery = %v, want foo=bar&baz=qux", e.URL.RawQuery)
		}
		if e.URL.QuerySize != len("foo=bar&baz=qux") {
			t.Errorf("QuerySize = %v, want %v", e.URL.QuerySize, len("foo=bar&baz=qux"))
		}
	})

	t.Run("preserves existing raw query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect?foo=bar", nil)
		e := &Event{URL: URLInfo{RawQuery: "existing=query"}}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		if e.URL.RawQuery != "existing=query" {
			t.Errorf("RawQuery = %v, want existing=query", e.URL.RawQuery)
		}
	})

	t.Run("extracts client IP without proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		e := &Event{}
		cfg := config.Config{TrustProxy: false}

		EnrichServerFields(req, e, cfg)

		if e.Server.IP != "192.168.1.100" {
			t.Errorf("IP = %v, want 192.168.1.100", e.Server.IP)
		}
	})

	t.Run("extracts client IP from X-Forwarded-For when proxy trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
		e := &Event{}
		cfg := config.Config{TrustProxy: true}

		EnrichServerFields(req, e, cfg)

		if e.Server.IP != "203.0.113.1" {
			t.Errorf("IP = %v, want 203.0.113.1", e.Server.IP)
		}
	})

	t.Run("extracts client IP from X-Real-IP when proxy trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Real-IP", "203.0.113.5")
		e := &Event{}
		cfg := config.Config{TrustProxy: true}

		EnrichServerFields(req, e, cfg)

		if e.Server.IP != "203.0.113.5" {
			t.Errorf("IP = %v, want 203.0.113.5", e.Server.IP)
		}
	})

	t.Run("sets server detection signals", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		e := &Event{}
		cfg := config.Config{}

		EnrichServerFields(req, e, cfg)

		// Verify detection signals are populated (basic check)
		// Detection details are tested in detection package
		if e.Server.Detection.HeaderFingerprint == "" {
			t.Error("detection header fingerprint should be set")
		}
	})
}

func TestParseUTMAndClickIDsFromRequest(t *testing.T) {
	t.Run("extracts all UTM parameters", func(t *testing.T) {
		reqURL := "/page?utm_source=google&utm_medium=cpc&utm_campaign=summer&utm_term=shoes&utm_content=ad1&utm_id=abc&utm_campaign_id=123"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}

		parseUTMAndClickIDsFromRequest(req, e)

		if e.URL.UTM.Source != "google" {
			t.Errorf("UTM.Source = %v, want google", e.URL.UTM.Source)
		}
		if e.URL.UTM.Medium != "cpc" {
			t.Errorf("UTM.Medium = %v, want cpc", e.URL.UTM.Medium)
		}
		if e.URL.UTM.Campaign != "summer" {
			t.Errorf("UTM.Campaign = %v, want summer", e.URL.UTM.Campaign)
		}
		if e.URL.UTM.Term != "shoes" {
			t.Errorf("UTM.Term = %v, want shoes", e.URL.UTM.Term)
		}
		if e.URL.UTM.Content != "ad1" {
			t.Errorf("UTM.Content = %v, want ad1", e.URL.UTM.Content)
		}
		if e.URL.UTM.ID != "abc" {
			t.Errorf("UTM.ID = %v, want abc", e.URL.UTM.ID)
		}
		if e.URL.UTM.CampaignID != "123" {
			t.Errorf("UTM.CampaignID = %v, want 123", e.URL.UTM.CampaignID)
		}
	})

	t.Run("preserves existing UTM parameters", func(t *testing.T) {
		reqURL := "/page?utm_source=google&utm_campaign=new"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{
			URL: URLInfo{
				UTM: UTMInfo{
					Source:   "existing",
					Campaign: "existing",
				},
			},
		}

		parseUTMAndClickIDsFromRequest(req, e)

		if e.URL.UTM.Source != "existing" {
			t.Errorf("UTM.Source = %v, want existing", e.URL.UTM.Source)
		}
		if e.URL.UTM.Campaign != "existing" {
			t.Errorf("UTM.Campaign = %v, want existing", e.URL.UTM.Campaign)
		}
	})

	t.Run("extracts Google Ads parameters", func(t *testing.T) {
		reqURL := "/page?gclid=test123&gclsrc=aw.ds&gbraid=br123&wbraid=wb456"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}

		parseUTMAndClickIDsFromRequest(req, e)

		if e.URL.Google.GCLID != "test123" {
			t.Errorf("Google.GCLID = %v, want test123", e.URL.Google.GCLID)
		}
		if e.URL.Google.GCLSRC != "aw.ds" {
			t.Errorf("Google.GCLSRC = %v, want aw.ds", e.URL.Google.GCLSRC)
		}
		if e.URL.Google.GBRAID != "br123" {
			t.Errorf("Google.GBRAID = %v, want br123", e.URL.Google.GBRAID)
		}
		if e.URL.Google.WBRAID != "wb456" {
			t.Errorf("Google.WBRAID = %v, want wb456", e.URL.Google.WBRAID)
		}
	})

	t.Run("extracts Google Ads campaign details", func(t *testing.T) {
		reqURL := "/page?campaignid=c123&adgroupid=ag456&creative=cr789&keyword=test&matchtype=exact&network=search&device=mobile&placement=top"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}

		parseUTMAndClickIDsFromRequest(req, e)

		if e.URL.Google.CampaignID != "c123" {
			t.Errorf("Google.CampaignID = %v, want c123", e.URL.Google.CampaignID)
		}
		if e.URL.Google.AdGroupID != "ag456" {
			t.Errorf("Google.AdGroupID = %v, want ag456", e.URL.Google.AdGroupID)
		}
		if e.URL.Google.AdID != "cr789" {
			t.Errorf("Google.AdID = %v, want cr789", e.URL.Google.AdID)
		}
		if e.URL.Google.KeywordID != "test" {
			t.Errorf("Google.KeywordID = %v, want test", e.URL.Google.KeywordID)
		}
		if e.URL.Google.MatchType != "exact" {
			t.Errorf("Google.MatchType = %v, want exact", e.URL.Google.MatchType)
		}
		if e.URL.Google.Network != "search" {
			t.Errorf("Google.Network = %v, want search", e.URL.Google.Network)
		}
		if e.URL.Google.Device != "mobile" {
			t.Errorf("Google.Device = %v, want mobile", e.URL.Google.Device)
		}
		if e.URL.Google.Placement != "top" {
			t.Errorf("Google.Placement = %v, want top", e.URL.Google.Placement)
		}
	})

	t.Run("extracts Meta Ads parameters", func(t *testing.T) {
		reqURL := "/page?fbclid=fb123&fbc=cookie123&fbp=pixel456&campaign_id=c789&adset_id=as012&ad_id=ad345"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}

		parseUTMAndClickIDsFromRequest(req, e)

		if e.URL.Meta.FBCLID != "fb123" {
			t.Errorf("Meta.FBCLID = %v, want fb123", e.URL.Meta.FBCLID)
		}
		if e.URL.Meta.FBC != "cookie123" {
			t.Errorf("Meta.FBC = %v, want cookie123", e.URL.Meta.FBC)
		}
		if e.URL.Meta.FBP != "pixel456" {
			t.Errorf("Meta.FBP = %v, want pixel456", e.URL.Meta.FBP)
		}
		if e.URL.Meta.CampaignID != "c789" {
			t.Errorf("Meta.CampaignID = %v, want c789", e.URL.Meta.CampaignID)
		}
		if e.URL.Meta.AdSetID != "as012" {
			t.Errorf("Meta.AdSetID = %v, want as012", e.URL.Meta.AdSetID)
		}
		if e.URL.Meta.AdID != "ad345" {
			t.Errorf("Meta.AdID = %v, want ad345", e.URL.Meta.AdID)
		}
	})

	t.Run("extracts Microsoft Ads parameters", func(t *testing.T) {
		reqURL := "/page?msclkid=ms123456"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}

		parseUTMAndClickIDsFromRequest(req, e)

		if e.URL.Microsoft.MSCLKID != "ms123456" {
			t.Errorf("Microsoft.MSCLKID = %v, want ms123456", e.URL.Microsoft.MSCLKID)
		}
	})

	t.Run("extracts other click IDs", func(t *testing.T) {
		reqURL := "/page?ttclid=tiktok123&li_fat_id=linkedin456&epik=pinterest789&twclid=twitter012&dclid=display345"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}

		parseUTMAndClickIDsFromRequest(req, e)

		if e.URL.OtherIDs["ttclid"] != "tiktok123" {
			t.Errorf("OtherIDs[ttclid] = %v, want tiktok123", e.URL.OtherIDs["ttclid"])
		}
		if e.URL.OtherIDs["li_fat_id"] != "linkedin456" {
			t.Errorf("OtherIDs[li_fat_id] = %v, want linkedin456", e.URL.OtherIDs["li_fat_id"])
		}
		if e.URL.OtherIDs["epik"] != "pinterest789" {
			t.Errorf("OtherIDs[epik] = %v, want pinterest789", e.URL.OtherIDs["epik"])
		}
		if e.URL.OtherIDs["twclid"] != "twitter012" {
			t.Errorf("OtherIDs[twclid] = %v, want twitter012", e.URL.OtherIDs["twclid"])
		}
		if e.URL.OtherIDs["dclid"] != "display345" {
			t.Errorf("OtherIDs[dclid] = %v, want display345", e.URL.OtherIDs["dclid"])
		}
	})

	t.Run("handles nil request URL", func(t *testing.T) {
		req := &http.Request{}
		e := &Event{}

		// Should not panic
		parseUTMAndClickIDsFromRequest(req, e)
	})

	t.Run("ignores whitespace in click IDs", func(t *testing.T) {
		reqURL := "/page?ttclid=%20%20&li_fat_id=%20%09%20&epik=valid123"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}

		parseUTMAndClickIDsFromRequest(req, e)

		if _, exists := e.URL.OtherIDs["ttclid"]; exists {
			t.Error("empty ttclid should not be added")
		}
		if _, exists := e.URL.OtherIDs["li_fat_id"]; exists {
			t.Error("whitespace-only li_fat_id should not be added")
		}
		if e.URL.OtherIDs["epik"] != "valid123" {
			t.Errorf("OtherIDs[epik] = %v, want valid123", e.URL.OtherIDs["epik"])
		}
	})
}

func TestCopyIf(t *testing.T) {
	t.Run("copies non-empty values", func(t *testing.T) {
		q := url.Values{
			"key1": []string{"value1"},
			"key2": []string{"value2"},
			"key3": []string{"value3"},
		}
		dst := map[string]string{}

		copyIf(q, dst, "key1", "key2")

		if dst["key1"] != "value1" {
			t.Errorf("dst[key1] = %v, want value1", dst["key1"])
		}
		if dst["key2"] != "value2" {
			t.Errorf("dst[key2] = %v, want value2", dst["key2"])
		}
		if _, exists := dst["key3"]; exists {
			t.Error("key3 should not be copied")
		}
	})

	t.Run("skips empty and whitespace values", func(t *testing.T) {
		q := url.Values{
			"empty":      []string{""},
			"whitespace": []string{"  \t  "},
			"valid":      []string{"value"},
		}
		dst := map[string]string{}

		copyIf(q, dst, "empty", "whitespace", "valid")

		if _, exists := dst["empty"]; exists {
			t.Error("empty key should not be copied")
		}
		if _, exists := dst["whitespace"]; exists {
			t.Error("whitespace key should not be copied")
		}
		if dst["valid"] != "value" {
			t.Errorf("dst[valid] = %v, want value", dst["valid"])
		}
	})

	t.Run("handles missing keys", func(t *testing.T) {
		q := url.Values{
			"exists": []string{"value"},
		}
		dst := map[string]string{}

		copyIf(q, dst, "exists", "missing")

		if dst["exists"] != "value" {
			t.Errorf("dst[exists] = %v, want value", dst["exists"])
		}
		if _, exists := dst["missing"]; exists {
			t.Error("missing key should not be copied")
		}
	})
}

func TestClientIPFromRequest(t *testing.T) {
	t.Run("returns RemoteAddr when proxy not trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		req.Header.Set("X-Real-IP", "203.0.113.2")

		ip := clientIPFromRequest(req, false)

		if ip != "192.168.1.100" {
			t.Errorf("ip = %v, want 192.168.1.100", ip)
		}
	})

	t.Run("returns first X-Forwarded-For IP when proxy trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 10.0.0.2")

		ip := clientIPFromRequest(req, true)

		if ip != "203.0.113.1" {
			t.Errorf("ip = %v, want 203.0.113.1", ip)
		}
	})

	t.Run("handles whitespace in X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "  203.0.113.1  , 198.51.100.1")

		ip := clientIPFromRequest(req, true)

		if ip != "203.0.113.1" {
			t.Errorf("ip = %v, want 203.0.113.1", ip)
		}
	})

	t.Run("returns X-Real-IP when no X-Forwarded-For and proxy trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Real-IP", "203.0.113.5")

		ip := clientIPFromRequest(req, true)

		if ip != "203.0.113.5" {
			t.Errorf("ip = %v, want 203.0.113.5", ip)
		}
	})

	t.Run("handles whitespace in X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Real-IP", "  203.0.113.5  ")

		ip := clientIPFromRequest(req, true)

		if ip != "203.0.113.5" {
			t.Errorf("ip = %v, want 203.0.113.5", ip)
		}
	})

	t.Run("falls back to RemoteAddr when headers empty and proxy trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"

		ip := clientIPFromRequest(req, true)

		if ip != "192.168.1.100" {
			t.Errorf("ip = %v, want 192.168.1.100", ip)
		}
	})

	t.Run("handles RemoteAddr without port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.100"

		ip := clientIPFromRequest(req, false)

		if ip != "192.168.1.100" {
			t.Errorf("ip = %v, want 192.168.1.100", ip)
		}
	})

	t.Run("returns full RemoteAddr when SplitHostPort fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "invalid::address::format"

		ip := clientIPFromRequest(req, false)

		if ip != "invalid::address::format" {
			t.Errorf("ip = %v, want invalid::address::format", ip)
		}
	})

	t.Run("prefers X-Forwarded-For over X-Real-IP when both present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		req.Header.Set("X-Real-IP", "203.0.113.2")

		ip := clientIPFromRequest(req, true)

		if ip != "203.0.113.1" {
			t.Errorf("ip = %v, want 203.0.113.1 (X-Forwarded-For should take precedence)", ip)
		}
	})

	t.Run("handles IPv6 addresses", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "[2001:db8::1]:12345"

		ip := clientIPFromRequest(req, false)

		if ip != "2001:db8::1" {
			t.Errorf("ip = %v, want 2001:db8::1", ip)
		}
	})
}

func TestEnrichServerFieldsIntegration(t *testing.T) {
	t.Run("enriches complete event with all parameters", func(t *testing.T) {
		reqURL := "/page?utm_source=google&utm_campaign=test&gclid=abc123&fbclid=fb456&ttclid=tt789&foo=bar"
		req := httptest.NewRequest(http.MethodPost, reqURL, nil)
		req.RemoteAddr = "203.0.113.1:12345"
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Referer", "https://google.com/search?q=test")
		req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.1")

		e := &Event{}
		cfg := config.Config{TrustProxy: true}

		EnrichServerFields(req, e, cfg)

		// Verify timestamp
		if e.TS == "" {
			t.Error("timestamp should be set")
		}

		// Verify type
		if e.Type != "pageview" {
			t.Errorf("type = %v, want pageview", e.Type)
		}

		// Verify device
		if e.Device.UA == "" {
			t.Error("user agent should be set")
		}

		// Verify referrer
		if e.URL.Referrer != "https://google.com/search?q=test" {
			t.Errorf("referrer = %v, want https://google.com/search?q=test", e.URL.Referrer)
		}
		if e.URL.ReferrerHostname != "google.com" {
			t.Errorf("referrer hostname = %v, want google.com", e.URL.ReferrerHostname)
		}

		// Verify raw query
		expectedQuery := "utm_source=google&utm_campaign=test&gclid=abc123&fbclid=fb456&ttclid=tt789&foo=bar"
		if e.URL.RawQuery != expectedQuery {
			t.Errorf("raw query = %v, want %v", e.URL.RawQuery, expectedQuery)
		}

		// Verify UTM
		if e.URL.UTM.Source != "google" {
			t.Errorf("UTM source = %v, want google", e.URL.UTM.Source)
		}
		if e.URL.UTM.Campaign != "test" {
			t.Errorf("UTM campaign = %v, want test", e.URL.UTM.Campaign)
		}

		// Verify Google Ads
		if e.URL.Google.GCLID != "abc123" {
			t.Errorf("GCLID = %v, want abc123", e.URL.Google.GCLID)
		}

		// Verify Meta Ads
		if e.URL.Meta.FBCLID != "fb456" {
			t.Errorf("FBCLID = %v, want fb456", e.URL.Meta.FBCLID)
		}

		// Verify other IDs
		if e.URL.OtherIDs["ttclid"] != "tt789" {
			t.Errorf("ttclid = %v, want tt789", e.URL.OtherIDs["ttclid"])
		}

		// Verify IP
		if e.Server.IP != "198.51.100.1" {
			t.Errorf("IP = %v, want 198.51.100.1", e.Server.IP)
		}

		// Verify detection signals are present
		if e.Server.Detection.HeaderFingerprint == "" {
			t.Error("detection header fingerprint should be set")
		}
	})

	t.Run("handles minimal request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.RemoteAddr = "192.168.1.1:54321"

		e := &Event{}
		cfg := config.Config{TrustProxy: false}

		EnrichServerFields(req, e, cfg)

		// Should have defaults
		if e.TS == "" {
			t.Error("timestamp should be set")
		}
		if e.Type != "pageview" {
			t.Errorf("type = %v, want pageview", e.Type)
		}
		if e.Server.IP != "192.168.1.1" {
			t.Errorf("IP = %v, want 192.168.1.1", e.Server.IP)
		}
	})
}
