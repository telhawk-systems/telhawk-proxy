package event

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/shortontech/gotrack/pkg/config"
)

func TestEnrichServerFields_Timestamp(t *testing.T) {
	t.Run("sets timestamp when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		e := &Event{}
		before := time.Now().UTC()
		EnrichServerFields(req, e, config.Config{})
		if e.TS == "" {
			t.Error("timestamp should be set")
		}
		ts, err := time.Parse(time.RFC3339Nano, e.TS)
		if err != nil {
			t.Errorf("timestamp should be valid RFC3339Nano: %v", err)
		}
		if ts.Before(before) || ts.After(time.Now().UTC().Add(time.Second)) {
			t.Errorf("timestamp %v should be recent", ts)
		}
	})

	t.Run("preserves existing timestamp", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		existingTS := "2024-01-01T12:00:00Z"
		e := &Event{TS: existingTS}
		EnrichServerFields(req, e, config.Config{})
		if e.TS != existingTS {
			t.Errorf("timestamp = %v, want %v", e.TS, existingTS)
		}
	})
}

func TestEnrichServerFields_EventType(t *testing.T) {
	t.Run("sets default type to pageview when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		e := &Event{}
		EnrichServerFields(req, e, config.Config{})
		if e.Type != "pageview" {
			t.Errorf("type = %v, want pageview", e.Type)
		}
	})

	t.Run("preserves existing event type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		e := &Event{Type: "click"}
		EnrichServerFields(req, e, config.Config{})
		if e.Type != "click" {
			t.Errorf("type = %v, want click", e.Type)
		}
	})
}

func TestEnrichServerFields_UserAgent(t *testing.T) {
	t.Run("extracts user agent when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 Test Browser")
		e := &Event{}
		EnrichServerFields(req, e, config.Config{})
		if e.Device.UA != "Mozilla/5.0 Test Browser" {
			t.Errorf("UA = %v, want Mozilla/5.0 Test Browser", e.Device.UA)
		}
	})

	t.Run("preserves existing user agent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 Test Browser")
		e := &Event{Device: DeviceInfo{UA: "Client UA"}}
		EnrichServerFields(req, e, config.Config{})
		if e.Device.UA != "Client UA" {
			t.Errorf("UA = %v, want Client UA", e.Device.UA)
		}
	})
}

func TestEnrichServerFields_Referrer(t *testing.T) {
	t.Run("extracts referrer when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Referer", "https://google.com/search")
		e := &Event{}
		EnrichServerFields(req, e, config.Config{})
		if e.URL.Referrer != "https://google.com/search" {
			t.Errorf("Referrer = %v, want https://google.com/search", e.URL.Referrer)
		}
		if e.URL.ReferrerHostname != "google.com" {
			t.Errorf("ReferrerHostname = %v, want google.com", e.URL.ReferrerHostname)
		}
	})

	t.Run("preserves existing referrer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Referer", "https://google.com")
		e := &Event{URL: URLInfo{Referrer: "https://existing.com"}}
		EnrichServerFields(req, e, config.Config{})
		if e.URL.Referrer != "https://existing.com" {
			t.Errorf("Referrer = %v, want https://existing.com", e.URL.Referrer)
		}
	})
}

func TestEnrichServerFields_Query(t *testing.T) {
	t.Run("extracts raw query when empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/?foo=bar&baz=qux", nil)
		e := &Event{}
		EnrichServerFields(req, e, config.Config{})
		if e.URL.RawQuery != "foo=bar&baz=qux" {
			t.Errorf("RawQuery = %v, want foo=bar&baz=qux", e.URL.RawQuery)
		}
		if e.URL.QuerySize != len("foo=bar&baz=qux") {
			t.Errorf("QuerySize = %v, want %v", e.URL.QuerySize, len("foo=bar&baz=qux"))
		}
	})

	t.Run("preserves existing raw query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/?foo=bar", nil)
		e := &Event{URL: URLInfo{RawQuery: "existing=query"}}
		EnrichServerFields(req, e, config.Config{})
		if e.URL.RawQuery != "existing=query" {
			t.Errorf("RawQuery = %v, want existing=query", e.URL.RawQuery)
		}
	})
}

func TestEnrichServerFields_ClientIP(t *testing.T) {
	t.Run("extracts client IP without proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		e := &Event{}
		EnrichServerFields(req, e, config.Config{TrustProxy: false})
		if e.Server.IP != "192.168.1.100" {
			t.Errorf("IP = %v, want 192.168.1.100", e.Server.IP)
		}
	})

	t.Run("extracts client IP from X-Forwarded-For when proxy trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")
		e := &Event{}
		EnrichServerFields(req, e, config.Config{TrustProxy: true})
		if e.Server.IP != "203.0.113.1" {
			t.Errorf("IP = %v, want 203.0.113.1", e.Server.IP)
		}
	})

	t.Run("extracts client IP from X-Real-IP when proxy trusted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("X-Real-IP", "203.0.113.5")
		e := &Event{}
		EnrichServerFields(req, e, config.Config{TrustProxy: true})
		if e.Server.IP != "203.0.113.5" {
			t.Errorf("IP = %v, want 203.0.113.5", e.Server.IP)
		}
	})
}

func TestEnrichServerFields_Detection(t *testing.T) {
	t.Run("sets server detection signals", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		e := &Event{}
		EnrichServerFields(req, e, config.Config{})
		if e.Server.Detection.HeaderFingerprint == "" {
			t.Error("detection header fingerprint should be set")
		}
	})
}

func assertUTMFields(t *testing.T, utm UTMInfo, expected map[string]string) {
	t.Helper()
	if expected["source"] != "" && utm.Source != expected["source"] {
		t.Errorf("UTM.Source = %v, want %v", utm.Source, expected["source"])
	}
	if expected["medium"] != "" && utm.Medium != expected["medium"] {
		t.Errorf("UTM.Medium = %v, want %v", utm.Medium, expected["medium"])
	}
	if expected["campaign"] != "" && utm.Campaign != expected["campaign"] {
		t.Errorf("UTM.Campaign = %v, want %v", utm.Campaign, expected["campaign"])
	}
	if expected["term"] != "" && utm.Term != expected["term"] {
		t.Errorf("UTM.Term = %v, want %v", utm.Term, expected["term"])
	}
	if expected["content"] != "" && utm.Content != expected["content"] {
		t.Errorf("UTM.Content = %v, want %v", utm.Content, expected["content"])
	}
	if expected["id"] != "" && utm.ID != expected["id"] {
		t.Errorf("UTM.ID = %v, want %v", utm.ID, expected["id"])
	}
	if expected["campaign_id"] != "" && utm.CampaignID != expected["campaign_id"] {
		t.Errorf("UTM.CampaignID = %v, want %v", utm.CampaignID, expected["campaign_id"])
	}
}

func assertGoogleFields(t *testing.T, google GoogleAdsInfo, expected map[string]string) {
	t.Helper()
	checks := map[string]struct{ got, want string }{
		"GCLID": {google.GCLID, expected["gclid"]}, "GCLSRC": {google.GCLSRC, expected["gclsrc"]},
		"GBRAID": {google.GBRAID, expected["gbraid"]}, "WBRAID": {google.WBRAID, expected["wbraid"]},
		"CampaignID": {google.CampaignID, expected["campaign_id"]}, "AdGroupID": {google.AdGroupID, expected["adgroup_id"]},
		"AdID": {google.AdID, expected["ad_id"]}, "KeywordID": {google.KeywordID, expected["keyword_id"]},
		"MatchType": {google.MatchType, expected["matchtype"]}, "Network": {google.Network, expected["network"]},
		"Device": {google.Device, expected["device"]}, "Placement": {google.Placement, expected["placement"]},
	}
	for field, v := range checks {
		if v.want != "" && v.got != v.want {
			t.Errorf("Google.%s = %v, want %v", field, v.got, v.want)
		}
	}
}

func assertMetaFields(t *testing.T, meta MetaAdsInfo, expected map[string]string) {
	t.Helper()
	checks := map[string]struct{ got, want string }{
		"FBCLID": {meta.FBCLID, expected["fbclid"]}, "FBC": {meta.FBC, expected["fbc"]},
		"FBP": {meta.FBP, expected["fbp"]}, "CampaignID": {meta.CampaignID, expected["campaign_id"]},
		"AdSetID": {meta.AdSetID, expected["adset_id"]}, "AdID": {meta.AdID, expected["ad_id"]},
	}
	for field, v := range checks {
		if v.want != "" && v.got != v.want {
			t.Errorf("Meta.%s = %v, want %v", field, v.got, v.want)
		}
	}
}

func TestParseUTMAndClickIDsFromRequest(t *testing.T) {
	t.Run("extracts all UTM parameters", func(t *testing.T) {
		reqURL := "/page?utm_source=google&utm_medium=cpc&utm_campaign=summer&utm_term=shoes&utm_content=ad1&utm_id=abc&utm_campaign_id=123"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}
		parseUTMAndClickIDsFromRequest(req, e)
		assertUTMFields(t, e.URL.UTM, map[string]string{
			"source": "google", "medium": "cpc", "campaign": "summer", "term": "shoes",
			"content": "ad1", "id": "abc", "campaign_id": "123",
		})
	})

	t.Run("preserves existing UTM parameters", func(t *testing.T) {
		reqURL := "/page?utm_source=google&utm_campaign=new"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{URL: URLInfo{UTM: UTMInfo{Source: "existing", Campaign: "existing"}}}
		parseUTMAndClickIDsFromRequest(req, e)
		assertUTMFields(t, e.URL.UTM, map[string]string{"source": "existing", "campaign": "existing"})
	})

	t.Run("extracts Google Ads parameters", func(t *testing.T) {
		reqURL := "/page?gclid=test123&gclsrc=aw.ds&gbraid=br123&wbraid=wb456"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}
		parseUTMAndClickIDsFromRequest(req, e)
		assertGoogleFields(t, e.URL.Google, map[string]string{
			"gclid": "test123", "gclsrc": "aw.ds", "gbraid": "br123", "wbraid": "wb456",
		})
	})

	t.Run("extracts Google Ads campaign details", func(t *testing.T) {
		reqURL := "/page?campaignid=c123&adgroupid=ag456&creative=cr789&keyword=test&matchtype=exact&network=search&device=mobile&placement=top"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}
		parseUTMAndClickIDsFromRequest(req, e)
		assertGoogleFields(t, e.URL.Google, map[string]string{
			"campaign_id": "c123", "adgroup_id": "ag456", "ad_id": "cr789", "keyword_id": "test",
			"matchtype": "exact", "network": "search", "device": "mobile", "placement": "top",
		})
	})

	t.Run("extracts Meta Ads parameters", func(t *testing.T) {
		reqURL := "/page?fbclid=fb123&fbc=cookie123&fbp=pixel456&campaign_id=c789&adset_id=as012&ad_id=ad345"
		req := httptest.NewRequest(http.MethodGet, reqURL, nil)
		e := &Event{}
		parseUTMAndClickIDsFromRequest(req, e)
		assertMetaFields(t, e.URL.Meta, map[string]string{
			"fbclid": "fb123", "fbc": "cookie123", "fbp": "pixel456",
			"campaign_id": "c789", "adset_id": "as012", "ad_id": "ad345",
		})
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
		expected := map[string]string{"ttclid": "tiktok123", "li_fat_id": "linkedin456", "epik": "pinterest789", "twclid": "twitter012", "dclid": "display345"}
		for key, want := range expected {
			if got := e.URL.OtherIDs[key]; got != want {
				t.Errorf("OtherIDs[%s] = %v, want %v", key, got, want)
			}
		}
	})

	t.Run("handles nil request URL", func(t *testing.T) {
		req := &http.Request{}
		e := &Event{}
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

		if e.TS == "" {
			t.Error("timestamp should be set")
		}
		if e.Type != "pageview" {
			t.Error("type should be pageview")
		}
		if e.Device.UA == "" {
			t.Error("user agent should be set")
		}
		if e.URL.Referrer != "https://google.com/search?q=test" {
			t.Error("referrer should be set correctly")
		}
		if e.URL.UTM.Source != "google" {
			t.Error("UTM source should be extracted")
		}
		if e.URL.Google.GCLID != "abc123" {
			t.Error("GCLID should be extracted")
		}
		if e.Server.IP != "198.51.100.1" {
			t.Error("IP should be set from X-Forwarded-For")
		}
	})

	t.Run("handles minimal request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/collect", nil)
		req.RemoteAddr = "192.168.1.1:54321"
		e := &Event{}
		cfg := config.Config{TrustProxy: false}
		EnrichServerFields(req, e, cfg)
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
