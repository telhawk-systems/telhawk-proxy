package event

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shortontech/gotrack/internal/event/detection"
	"github.com/shortontech/gotrack/pkg/config"
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

	parseUTMParams(q, e)
	parseGoogleParams(q, e)
	parseMetaParams(q, e)
	parseMicrosoftParams(q, e)
	parseOtherClickIDs(q, e)
}

func parseUTMParams(q url.Values, e *Event) {
	setIfEmpty(&e.URL.UTM.Source, q.Get("utm_source"))
	setIfEmpty(&e.URL.UTM.Medium, q.Get("utm_medium"))
	setIfEmpty(&e.URL.UTM.Campaign, q.Get("utm_campaign"))
	setIfEmpty(&e.URL.UTM.Term, q.Get("utm_term"))
	setIfEmpty(&e.URL.UTM.Content, q.Get("utm_content"))
	setIfEmpty(&e.URL.UTM.ID, q.Get("utm_id"))
	setIfEmpty(&e.URL.UTM.CampaignID, q.Get("utm_campaign_id"))
}

func parseGoogleParams(q url.Values, e *Event) {
	setIfEmpty(&e.URL.Google.GCLID, q.Get("gclid"))
	setIfEmpty(&e.URL.Google.GCLSRC, q.Get("gclsrc"))
	setIfEmpty(&e.URL.Google.GBRAID, q.Get("gbraid"))
	setIfEmpty(&e.URL.Google.WBRAID, q.Get("wbraid"))
	setIfEmpty(&e.URL.Google.CampaignID, q.Get("campaignid"))
	setIfEmpty(&e.URL.Google.AdGroupID, q.Get("adgroupid"))
	setIfEmpty(&e.URL.Google.AdID, q.Get("creative"))
	setIfEmpty(&e.URL.Google.KeywordID, q.Get("keyword"))
	setIfEmpty(&e.URL.Google.MatchType, q.Get("matchtype"))
	setIfEmpty(&e.URL.Google.Network, q.Get("network"))
	setIfEmpty(&e.URL.Google.Device, q.Get("device"))
	setIfEmpty(&e.URL.Google.Placement, q.Get("placement"))
}

func parseMetaParams(q url.Values, e *Event) {
	setIfEmpty(&e.URL.Meta.FBCLID, q.Get("fbclid"))
	setIfEmpty(&e.URL.Meta.FBC, q.Get("fbc"))
	setIfEmpty(&e.URL.Meta.FBP, q.Get("fbp"))
	setIfEmpty(&e.URL.Meta.CampaignID, q.Get("campaign_id"))
	setIfEmpty(&e.URL.Meta.AdSetID, q.Get("adset_id"))
	setIfEmpty(&e.URL.Meta.AdID, q.Get("ad_id"))
}

func parseMicrosoftParams(q url.Values, e *Event) {
	setIfEmpty(&e.URL.Microsoft.MSCLKID, q.Get("msclkid"))
}

func parseOtherClickIDs(q url.Values, e *Event) {
	if e.URL.OtherIDs == nil {
		e.URL.OtherIDs = map[string]string{}
	}
	copyIf(q, e.URL.OtherIDs, "ttclid", "li_fat_id", "epik", "twclid", "dclid")
}

func setIfEmpty(dst *string, value string) {
	if *dst == "" {
		*dst = value
	}
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
