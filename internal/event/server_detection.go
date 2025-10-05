package event

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ServerDetectionSignals represents raw server-side detection data
type ServerDetectionSignals struct {
	HeaderFingerprint string                 `json:"header_fingerprint"`
	HeaderAnalysis    HeaderAnalysis         `json:"header_analysis"`
	RequestAnalysis   RequestAnalysis        `json:"request_analysis"`
	TimingAnalysis    TimingAnalysis         `json:"timing_analysis"`
}

type HeaderAnalysis struct {
	MissingExpected     []string `json:"missing_expected"`
	AutomationHeaders   []string `json:"automation_headers"`
	InconsistentValues  []string `json:"inconsistent_values"`
	HeaderOrder         []string `json:"header_order"`
	HeaderCount         int      `json:"header_count"`
}

type RequestAnalysis struct {
	PayloadEntropy      float64  `json:"payload_entropy"`
	RequestSize         int      `json:"request_size"`
	UserAgentAnalysis   UAAnalysis `json:"user_agent_analysis"`
}

type UAAnalysis struct {
	Length              int      `json:"length"`
	ContainsAutomation  bool     `json:"contains_automation"`
	AutomationKeywords  []string `json:"automation_keywords"`
	Platform            string   `json:"platform"`
	Browser             string   `json:"browser"`
}

type TimingAnalysis struct {
	RequestInterval     float64 `json:"request_interval_ms"`
	IntervalPrecision   int     `json:"interval_precision"` // How precise the timing is (e.g., exact 100ms intervals)
	RequestsPerSecond   float64 `json:"requests_per_second"`
	HasPreviousRequest  bool    `json:"has_previous_request"`
}

// Global request timing tracker (in production, use Redis or database)
var lastRequestTimes = make(map[string]time.Time)

// AnalyzeServerDetectionSignals performs comprehensive server-side detection data collection
func AnalyzeServerDetectionSignals(r *http.Request, body []byte) ServerDetectionSignals {
	signals := ServerDetectionSignals{}

	// Analyze HTTP headers
	signals.HeaderAnalysis = analyzeHeaders(r.Header)
	signals.HeaderFingerprint = generateHeaderFingerprint(r.Header)

	// Analyze request payload
	signals.RequestAnalysis = analyzeRequest(r, body)

	// Analyze timing patterns
	signals.TimingAnalysis = analyzeTimingPatterns(r)

	return signals
}

func analyzeHeaders(headers http.Header) HeaderAnalysis {
	analysis := HeaderAnalysis{
		MissingExpected:   []string{},
		AutomationHeaders: []string{},
		InconsistentValues: []string{},
		HeaderOrder:       []string{},
		HeaderCount:       len(headers),
	}

	// Collect header order for fingerprinting
	for key := range headers {
		analysis.HeaderOrder = append(analysis.HeaderOrder, strings.ToLower(key))
	}
	sort.Strings(analysis.HeaderOrder)

	// Check for automation-specific headers and values
	automationIndicators := map[string][]string{
		"X-Requested-With":     {"xmlhttprequest"},
		"Purpose":              {"prefetch"},
		"X-Purpose":            {"preview"},
		"Sec-Fetch-Mode":       {"navigate", "cors", "no-cors"},
		"Chrome-Proxy":         {},
		"X-DevTools-Emulate-Network-Conditions-Client-Id": {},
	}

	for header, suspiciousValues := range automationIndicators {
		if value := headers.Get(header); value != "" {
			lowerValue := strings.ToLower(value)
			
			// Check for automation tool signatures in any header
			if strings.Contains(lowerValue, "headless") ||
				strings.Contains(lowerValue, "selenium") ||
				strings.Contains(lowerValue, "webdriver") ||
				strings.Contains(lowerValue, "puppeteer") ||
				strings.Contains(lowerValue, "playwright") {
				analysis.AutomationHeaders = append(analysis.AutomationHeaders, fmt.Sprintf("%s: %s", header, value))
			}

			// Check for specific suspicious values
			if len(suspiciousValues) > 0 {
				for _, suspicious := range suspiciousValues {
					if strings.Contains(lowerValue, suspicious) {
						analysis.AutomationHeaders = append(analysis.AutomationHeaders, fmt.Sprintf("%s: %s", header, value))
					}
				}
			}
		}
	}

	// Check for missing expected headers
	expectedHeaders := []string{"User-Agent", "Accept", "Accept-Language", "Accept-Encoding"}
	for _, expected := range expectedHeaders {
		if headers.Get(expected) == "" {
			analysis.MissingExpected = append(analysis.MissingExpected, expected)
		}
	}

	// Check for language/UA inconsistencies
	userAgent := headers.Get("User-Agent")
	acceptLanguage := headers.Get("Accept-Language")
	if userAgent != "" && acceptLanguage != "" {
		if isLanguageUAInconsistent(userAgent, acceptLanguage) {
			analysis.InconsistentValues = append(analysis.InconsistentValues, "language-ua-mismatch")
		}
	}

	return analysis
}

func analyzeRequest(r *http.Request, body []byte) RequestAnalysis {
	analysis := RequestAnalysis{
		RequestSize: len(body),
	}

	// Calculate payload entropy
	if len(body) > 0 {
		analysis.PayloadEntropy = calculateEntropy(body)
	}

	// Detailed User-Agent analysis
	analysis.UserAgentAnalysis = analyzeUserAgent(r.UserAgent())

	return analysis
}

func analyzeUserAgent(userAgent string) UAAnalysis {
	analysis := UAAnalysis{
		Length:             len(userAgent),
		AutomationKeywords: []string{},
	}

	lowerUA := strings.ToLower(userAgent)
	
	// Check for automation keywords
	automationKeywords := []string{
		"headless", "selenium", "webdriver", "puppeteer", 
		"playwright", "phantom", "jsdom", "nightmare",
		"chrome-headless", "automated", "bot", "crawler",
	}

	for _, keyword := range automationKeywords {
		if strings.Contains(lowerUA, keyword) {
			analysis.ContainsAutomation = true
			analysis.AutomationKeywords = append(analysis.AutomationKeywords, keyword)
		}
	}

	// Extract basic platform/browser info
	if strings.Contains(lowerUA, "windows") {
		analysis.Platform = "Windows"
	} else if strings.Contains(lowerUA, "mac") {
		analysis.Platform = "macOS"
	} else if strings.Contains(lowerUA, "linux") {
		analysis.Platform = "Linux"
	} else if strings.Contains(lowerUA, "android") {
		analysis.Platform = "Android"
	} else if strings.Contains(lowerUA, "iphone") || strings.Contains(lowerUA, "ipad") {
		analysis.Platform = "iOS"
	}

	if strings.Contains(lowerUA, "chrome") && !strings.Contains(lowerUA, "edge") {
		analysis.Browser = "Chrome"
	} else if strings.Contains(lowerUA, "firefox") {
		analysis.Browser = "Firefox"
	} else if strings.Contains(lowerUA, "safari") && !strings.Contains(lowerUA, "chrome") {
		analysis.Browser = "Safari"
	} else if strings.Contains(lowerUA, "edge") {
		analysis.Browser = "Edge"
	}

	return analysis
}

func analyzeTimingPatterns(r *http.Request) TimingAnalysis {
	analysis := TimingAnalysis{}

	clientIP := getClientIP(r)
	now := time.Now()
	
	if lastTime, exists := lastRequestTimes[clientIP]; exists {
		interval := now.Sub(lastTime)
		analysis.RequestInterval = float64(interval.Nanoseconds()) / 1e6 // Convert to milliseconds
		analysis.HasPreviousRequest = true
		analysis.RequestsPerSecond = 1000.0 / analysis.RequestInterval

		// Analyze interval precision (automation often has very precise timing)
		intervalMs := int64(interval.Nanoseconds() / 1e6)
		if intervalMs > 0 {
			// Check if it's a round number (100ms, 500ms, 1000ms, etc.)
			if intervalMs%1000 == 0 {
				analysis.IntervalPrecision = 1000
			} else if intervalMs%500 == 0 {
				analysis.IntervalPrecision = 500
			} else if intervalMs%100 == 0 {
				analysis.IntervalPrecision = 100
			} else if intervalMs%50 == 0 {
				analysis.IntervalPrecision = 50
			} else if intervalMs%10 == 0 {
				analysis.IntervalPrecision = 10
			}
		}
	}
	
	lastRequestTimes[clientIP] = now
	return analysis
}

func generateHeaderFingerprint(headers http.Header) string {
	// Create a fingerprint based on header names and order
	var headerParts []string
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)

	for _, key := range keys {
		// Include only the header name and first few chars of value for fingerprinting
		value := headers.Get(key)
		if len(value) > 20 {
			value = value[:20] + "..."
		}
		headerParts = append(headerParts, key+":"+value)
	}

	fingerprint := strings.Join(headerParts, "|")
	hash := sha256.Sum256([]byte(fingerprint))
	return hex.EncodeToString(hash[:8]) // First 8 bytes as hex
}

func calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// Calculate byte frequency
	freq := make(map[byte]int)
	for _, b := range data {
		freq[b]++
	}

	// Calculate Shannon entropy
	entropy := 0.0
	length := float64(len(data))
	for _, count := range freq {
		if count > 0 {
			p := float64(count) / length
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func isLanguageUAInconsistent(userAgent, acceptLanguage string) bool {
	ua := strings.ToLower(userAgent)
	lang := strings.ToLower(acceptLanguage)

	// Basic inconsistency checks
	if strings.Contains(ua, "zh-cn") && !strings.Contains(lang, "zh") {
		return true
	}
	if strings.Contains(ua, "ja") && !strings.Contains(lang, "ja") {
		return true
	}
	if strings.Contains(ua, "ko") && !strings.Contains(lang, "ko") {
		return true
	}

	return false
}

func getClientIP(r *http.Request) string {
	// Simple IP extraction - enhance based on your proxy setup
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	return r.RemoteAddr
}