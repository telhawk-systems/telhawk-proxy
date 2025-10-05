package detection

// ServerDetectionSignals represents raw server-side detection data
type ServerDetectionSignals struct {
	HeaderFingerprint string          `json:"header_fingerprint"`
	HeaderAnalysis    HeaderAnalysis  `json:"header_analysis"`
	RequestAnalysis   RequestAnalysis `json:"request_analysis"`
	TimingAnalysis    TimingAnalysis  `json:"timing_analysis"`
}

// HeaderAnalysis contains header-based detection signals
type HeaderAnalysis struct {
	MissingExpected    []string `json:"missing_expected"`
	AutomationHeaders  []string `json:"automation_headers"`
	InconsistentValues []string `json:"inconsistent_values"`
	HeaderOrder        []string `json:"header_order"`
	HeaderCount        int      `json:"header_count"`
}

// RequestAnalysis contains request payload analysis
type RequestAnalysis struct {
	PayloadEntropy    float64    `json:"payload_entropy"`
	RequestSize       int        `json:"request_size"`
	UserAgentAnalysis UAAnalysis `json:"user_agent_analysis"`
}

// UAAnalysis contains user-agent string analysis
type UAAnalysis struct {
	Length             int      `json:"length"`
	ContainsAutomation bool     `json:"contains_automation"`
	AutomationKeywords []string `json:"automation_keywords"`
	Platform           string   `json:"platform"`
	Browser            string   `json:"browser"`
}

// TimingAnalysis contains request timing pattern analysis
type TimingAnalysis struct {
	RequestInterval    float64 `json:"request_interval_ms"`
	IntervalPrecision  int     `json:"interval_precision"` // How precise the timing is (e.g., exact 100ms intervals)
	RequestsPerSecond  float64 `json:"requests_per_second"`
	HasPreviousRequest bool    `json:"has_previous_request"`
}
