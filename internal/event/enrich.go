package event

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"revinar.io/go.track/internal/event/detection"
	"revinar.io/go.track/pkg/config"
)

// Normalize fields that the server can set/augment safely.
func EnrichServerFields(r *http.Request, e *Event, cfg config.Config) {
	if e.TS == "" {
		e.TS = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if e.Type == "" {
		e.Type = "pageview"
	}
	// UA
	if e.Device.UA == "" {
		e.Device.UA = r.UserAgent()
	}
	// Referrer
	if e.URL.Referrer == "" {
		e.URL.Referrer = r.Referer()
		if u, err := url.Parse(e.URL.Referrer); err == nil && u != nil {
			e.URL.ReferrerHostname = u.Hostname()
		}
	}
	// Raw query size (if not already set)
	if e.URL.RawQuery == "" && r.URL != nil {
		e.URL.RawQuery = r.URL.RawQuery
		e.URL.QuerySize = len(e.URL.RawQuery)
	}

	// Parse common UTM/click-ids from URL if client didn't supply
	parseUTMAndClickIDsFromRequest(r, e)

	// IP hashing (coarse privacy)
	e.Server.IP = clientIPFromRequest(r, cfg.TrustProxy)

	// Server-side detection signals (raw data, no scoring)
	body := []byte{} // TODO: Pass actual body if available
	e.Server.Detection = detection.AnalyzeServerDetectionSignals(r, body)
}

// Extract UTM & known click ids directly from the request URL (server-side fallback).
func parseUTMAndClickIDsFromRequest(r *http.Request, e *Event) {
	if r.URL == nil {
		return
	}
	q := r.URL.Query()

	// UTM
	if e.URL.UTM.Source == "" {
		e.URL.UTM.Source = q.Get("utm_source")
	}
	if e.URL.UTM.Medium == "" {
		e.URL.UTM.Medium = q.Get("utm_medium")
	}
	if e.URL.UTM.Campaign == "" {
		e.URL.UTM.Campaign = q.Get("utm_campaign")
	}
	if e.URL.UTM.Term == "" {
		e.URL.UTM.Term = q.Get("utm_term")
	}
	if e.URL.UTM.Content == "" {
		e.URL.UTM.Content = q.Get("utm_content")
	}
	if e.URL.UTM.ID == "" {
		e.URL.UTM.ID = q.Get("utm_id")
	}
	if e.URL.UTM.CampaignID == "" {
		e.URL.UTM.CampaignID = q.Get("utm_campaign_id")
	}

	// Google
	if e.URL.Google.GCLID == "" {
		e.URL.Google.GCLID = q.Get("gclid")
	}
	if e.URL.Google.GCLSRC == "" {
		e.URL.Google.GCLSRC = q.Get("gclsrc")
	}
	if e.URL.Google.GBRAID == "" {
		e.URL.Google.GBRAID = q.Get("gbraid")
	}
	if e.URL.Google.WBRAID == "" {
		e.URL.Google.WBRAID = q.Get("wbraid")
	}

	if e.URL.Google.CampaignID == "" {
		e.URL.Google.CampaignID = q.Get("campaignid")
	}
	if e.URL.Google.AdGroupID == "" {
		e.URL.Google.AdGroupID = q.Get("adgroupid")
	}
	if e.URL.Google.AdID == "" {
		e.URL.Google.AdID = q.Get("creative")
	}
	if e.URL.Google.KeywordID == "" {
		e.URL.Google.KeywordID = q.Get("keyword")
	}

	if e.URL.Google.MatchType == "" {
		e.URL.Google.MatchType = q.Get("matchtype")
	}
	if e.URL.Google.Network == "" {
		e.URL.Google.Network = q.Get("network")
	}
	if e.URL.Google.Device == "" {
		e.URL.Google.Device = q.Get("device")
	}
	if e.URL.Google.Placement == "" {
		e.URL.Google.Placement = q.Get("placement")
	}

	// Meta
	if e.URL.Meta.FBCLID == "" {
		e.URL.Meta.FBCLID = q.Get("fbclid")
	}
	if e.URL.Meta.FBC == "" {
		e.URL.Meta.FBC = q.Get("fbc")
	}
	if e.URL.Meta.FBP == "" {
		e.URL.Meta.FBP = q.Get("fbp")
	}
	if e.URL.Meta.CampaignID == "" {
		e.URL.Meta.CampaignID = q.Get("campaign_id")
	}
	if e.URL.Meta.AdSetID == "" {
		e.URL.Meta.AdSetID = q.Get("adset_id")
	}
	if e.URL.Meta.AdID == "" {
		e.URL.Meta.AdID = q.Get("ad_id")
	}

	// Microsoft
	if e.URL.Microsoft.MSCLKID == "" {
		e.URL.Microsoft.MSCLKID = q.Get("msclkid")
	}

	// Other common click ids
	if e.URL.OtherIDs == nil {
		e.URL.OtherIDs = map[string]string{}
	}
	copyIf(q, e.URL.OtherIDs, "ttclid", "li_fat_id", "epik", "twclid", "dclid")
}

func copyIf(q url.Values, dst map[string]string, keys ...string) {
	for _, k := range keys {
		if v := strings.TrimSpace(q.Get(k)); v != "" {
			dst[k] = v
		}
	}
}

func clientIPFromRequest(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return ip
			}
		}
		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			return strings.TrimSpace(xrip)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
