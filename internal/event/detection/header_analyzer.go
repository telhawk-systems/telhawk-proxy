package detection

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// analyzeHeaders performs comprehensive HTTP header analysis
func analyzeHeaders(headers http.Header) HeaderAnalysis {
	analysis := HeaderAnalysis{
		MissingExpected:    []string{},
		AutomationHeaders:  []string{},
		InconsistentValues: []string{},
		HeaderOrder:        []string{},
		HeaderCount:        len(headers),
	}

	// Collect header order for fingerprinting
	for key := range headers {
		analysis.HeaderOrder = append(analysis.HeaderOrder, strings.ToLower(key))
	}
	sort.Strings(analysis.HeaderOrder)

	// Detect automation headers
	analysis.AutomationHeaders = detectAutomationHeaders(headers)

	// Check for missing expected headers
	analysis.MissingExpected = checkMissingHeaders(headers)

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

// detectAutomationHeaders checks for automation-specific headers and values
func detectAutomationHeaders(headers http.Header) []string {
	var automationHeaders []string

	// Check for automation tool signatures in ANY header value
	automationKeywords := []string{"headless", "selenium", "webdriver", "puppeteer", "playwright"}
	
	for header, values := range headers {
		for _, value := range values {
			lowerValue := strings.ToLower(value)
			for _, keyword := range automationKeywords {
				if strings.Contains(lowerValue, keyword) {
					automationHeaders = append(automationHeaders, fmt.Sprintf("%s: %s", header, value))
					break // Don't duplicate the same header
				}
			}
		}
	}

	// Check for automation-specific headers
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

			// Check for specific suspicious values
			if len(suspiciousValues) > 0 {
				for _, suspicious := range suspiciousValues {
					if strings.Contains(lowerValue, suspicious) {
						automationHeaders = append(automationHeaders, fmt.Sprintf("%s: %s", header, value))
						break
					}
				}
			} else {
				// If no specific values, presence of the header itself is suspicious
				automationHeaders = append(automationHeaders, fmt.Sprintf("%s: %s", header, value))
			}
		}
	}

	return automationHeaders
}

// checkMissingHeaders checks for missing expected headers
func checkMissingHeaders(headers http.Header) []string {
	var missing []string
	expectedHeaders := []string{"User-Agent", "Accept", "Accept-Language", "Accept-Encoding"}

	for _, expected := range expectedHeaders {
		if headers.Get(expected) == "" {
			missing = append(missing, expected)
		}
	}

	return missing
}

// isLanguageUAInconsistent checks if Accept-Language and User-Agent are inconsistent
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
