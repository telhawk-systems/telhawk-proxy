package event

import "revinar.io/go.track/internal/event/detection"

// High-level envelope. Optional fields are omitted when empty.
type Event struct {
	EventID string `json:"event_id,omitempty"`
	TS      string `json:"ts,omitempty"`   // ISO8601
	Type    string `json:"type,omitempty"` // "pageview", "click", etc.

	URL     URLInfo     `json:"url,omitempty"`
	Route   RouteInfo   `json:"route,omitempty"`
	Device  DeviceInfo  `json:"device,omitempty"`
	Session SessionInfo `json:"session,omitempty"`
	Server  ServerMeta  `json:"server,omitempty"`
	Consent ConsentInfo `json:"consent,omitempty"`
}

// --- URL / attribution ---

type URLInfo struct {
	UTM       UTMInfo           `json:"utm,omitempty"`
	Google    GoogleAdsInfo     `json:"google,omitempty"`
	Meta      MetaAdsInfo       `json:"meta,omitempty"`
	Microsoft MicrosoftAdsInfo  `json:"microsoft,omitempty"`
	OtherIDs  map[string]string `json:"other_click_ids,omitempty"` // ttclid, li_fat_id, epik, twclid, etc.

	Referrer         string `json:"referrer,omitempty"`
	ReferrerHostname string `json:"referrer_hostname,omitempty"`
	RawQuery         string `json:"raw_query,omitempty"`
	QuerySize        int    `json:"query_size,omitempty"`
}

type UTMInfo struct {
	Source     string `json:"source,omitempty"`
	Medium     string `json:"medium,omitempty"`
	Campaign   string `json:"campaign,omitempty"`
	Term       string `json:"term,omitempty"`
	Content    string `json:"content,omitempty"`
	ID         string `json:"id,omitempty"`
	CampaignID string `json:"campaign_id,omitempty"`
}

type GoogleAdsInfo struct {
	GCLID  string `json:"gclid,omitempty"`
	GCLSRC string `json:"gclsrc,omitempty"`
	GBRAID string `json:"gbraid,omitempty"`
	WBRAID string `json:"wbraid,omitempty"`

	CampaignID string `json:"campaign_id,omitempty"`
	AdGroupID  string `json:"ad_group_id,omitempty"`
	AdID       string `json:"ad_id,omitempty"`
	KeywordID  string `json:"keyword_id,omitempty"`

	// Common ValueTrack extras (optional)
	MatchType string `json:"matchtype,omitempty"`
	Network   string `json:"network,omitempty"`
	Device    string `json:"device,omitempty"`
	Placement string `json:"placement,omitempty"`
}

type MetaAdsInfo struct {
	FBCLID     string `json:"fbclid,omitempty"`
	FBC        string `json:"fbc,omitempty"`
	FBP        string `json:"fbp,omitempty"`
	CampaignID string `json:"campaign_id,omitempty"`
	AdSetID    string `json:"adset_id,omitempty"`
	AdID       string `json:"ad_id,omitempty"`
}

type MicrosoftAdsInfo struct {
	MSCLKID string `json:"msclkid,omitempty"`
}

// --- Route ---

type RouteInfo struct {
	Domain       string            `json:"domain,omitempty"`
	Path         string            `json:"path,omitempty"`
	FullPath     string            `json:"fullPath,omitempty"`
	Hash         string            `json:"hash,omitempty"`
	CanonicalURL string            `json:"canonical_url,omitempty"`
	Title        string            `json:"title,omitempty"`
	Protocol     string            `json:"protocol,omitempty"`
	Query        map[string]string `json:"query,omitempty"`
}

// --- Device ---

type DeviceInfo struct {
	UA       string   `json:"ua,omitempty"`
	UABrands []string `json:"ua_brands,omitempty"`
	UAMobile *bool    `json:"ua_mobile,omitempty"`

	OS              string   `json:"os,omitempty"`
	Browser         string   `json:"browser,omitempty"`
	Language        string   `json:"language,omitempty"`
	Languages       []string `json:"languages,omitempty"`
	TZ              string   `json:"tz,omitempty"`
	TZOffsetMinutes int      `json:"tz_offset_minutes,omitempty"`

	ViewportW        int     `json:"viewport_w,omitempty"`
	ViewportH        int     `json:"viewport_h,omitempty"`
	DevicePixelRatio float64 `json:"device_pixel_ratio,omitempty"`

	HardwareConcurrency int `json:"hardware_concurrency,omitempty"`
	DeviceMemoryGB      int `json:"device_memory,omitempty"`
	MaxTouchPoints      int `json:"maxTouchPoints,omitempty"`

	PrefersColorScheme   string `json:"prefers_color_scheme,omitempty"`
	PrefersReducedMotion *bool  `json:"prefers_reduced_motion,omitempty"`

	CookieEnabled    *bool `json:"cookie_enabled,omitempty"`
	StorageAvailable *bool `json:"storage_available,omitempty"`

	NetworkEffectiveType string  `json:"network_effective_type,omitempty"`
	NetworkDownlinkMbps  float64 `json:"network_downlink,omitempty"`
	NetworkRTT           int     `json:"network_rtt,omitempty"`
	NetworkSaveData      *bool   `json:"network_save_data,omitempty"`

	GPU      string       `json:"gpu,omitempty"`
	Monitors int          `json:"monitors,omitempty"`
	Screens  []ScreenInfo `json:"screens,omitempty"`
}

type ScreenInfo struct {
	Width       int `json:"width,omitempty"`
	Height      int `json:"height,omitempty"`
	AvailWidth  int `json:"availWidth,omitempty"`
	AvailHeight int `json:"availHeight,omitempty"`
	ColorDepth  int `json:"colorDepth,omitempty"`
	PixelDepth  int `json:"pixelDepth,omitempty"`
}

// --- Session / Event meta ---

type SessionInfo struct {
	VisitorID    string `json:"visitor_id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	SessionStart string `json:"session_start_ts,omitempty"`
	SessionSeq   int    `json:"session_seq,omitempty"`
	FirstVisitTS string `json:"first_visit_ts,omitempty"`
}

// --- Server enrich ---

type ServerMeta struct {
	IP        string                           `json:"ip_hash,omitempty"`   // hash of client IP (if enabled)
	Geo       map[string]string                `json:"geo,omitempty"`       // coarse {country,region,city}
	Detection detection.ServerDetectionSignals `json:"detection,omitempty"` // Raw detection signals
}

// --- Consent ---

type ConsentInfo struct {
	GDPRApplies *bool  `json:"gdpr_applies,omitempty"`
	TCString    string `json:"tc_string,omitempty"`    // IAB TCF v2
	USPrivacy   string `json:"us_privacy,omitempty"`   // CCPA/US privacy
	GPPString   string `json:"gpp_string,omitempty"`   // GPP if present
	ConsentMode string `json:"consent_mode,omitempty"` // e.g., "ad_storage=denied,analytics_storage=granted"
}
