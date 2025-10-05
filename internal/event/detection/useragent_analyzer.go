package detection

import (
	"strings"
)

// analyzeUserAgent performs detailed user-agent string analysis
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

	// Extract basic platform info
	analysis.Platform = extractPlatform(lowerUA)

	// Extract basic browser info
	analysis.Browser = extractBrowser(lowerUA)

	return analysis
}

// extractPlatform extracts platform information from user-agent string
func extractPlatform(lowerUA string) string {
	// Check mobile platforms first (iOS UAs contain "Mac OS X")
	if strings.Contains(lowerUA, "iphone") || strings.Contains(lowerUA, "ipad") {
		return "iOS"
	} else if strings.Contains(lowerUA, "android") {
		return "Android"
	} else if strings.Contains(lowerUA, "windows") {
		return "Windows"
	} else if strings.Contains(lowerUA, "mac") {
		return "macOS"
	} else if strings.Contains(lowerUA, "linux") {
		return "Linux"
	}
	return ""
}

// extractBrowser extracts browser information from user-agent string
func extractBrowser(lowerUA string) string {
	if strings.Contains(lowerUA, "chrome") && !strings.Contains(lowerUA, "edge") {
		return "Chrome"
	} else if strings.Contains(lowerUA, "firefox") {
		return "Firefox"
	} else if strings.Contains(lowerUA, "safari") && !strings.Contains(lowerUA, "chrome") {
		return "Safari"
	} else if strings.Contains(lowerUA, "edge") {
		return "Edge"
	}
	return ""
}
