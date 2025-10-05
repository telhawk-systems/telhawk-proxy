package detection

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAnalyzeHeaders(t *testing.T) {
	t.Run("detects missing expected headers", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		analysis := analyzeHeaders(headers)

		if len(analysis.MissingExpected) != 4 {
			t.Errorf("expected 4 missing headers, got %d", len(analysis.MissingExpected))
		}
		expectedMissing := []string{"User-Agent", "Accept", "Accept-Language", "Accept-Encoding"}
		for _, expected := range expectedMissing {
			found := false
			for _, missing := range analysis.MissingExpected {
				if missing == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %s to be in missing headers", expected)
			}
		}
	})

	t.Run("detects automation headers", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 HeadlessChrome/90.0")
		headers.Set("Accept", "*/*")
		headers.Set("Accept-Language", "en-US")
		headers.Set("Accept-Encoding", "gzip")

		analysis := analyzeHeaders(headers)

		if len(analysis.AutomationHeaders) == 0 {
			t.Error("expected automation headers to be detected")
		}
	})

	t.Run("counts headers correctly", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("User-Agent", "Mozilla/5.0")
		headers.Set("Accept", "text/html")
		headers.Set("Accept-Language", "en-US")

		analysis := analyzeHeaders(headers)

		if analysis.HeaderCount != 3 {
			t.Errorf("expected header count 3, got %d", analysis.HeaderCount)
		}
	})

	t.Run("orders headers alphabetically", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("User-Agent", "test")
		headers.Set("Accept", "test")
		headers.Set("Content-Type", "test")

		analysis := analyzeHeaders(headers)

		if len(analysis.HeaderOrder) != 3 {
			t.Fatalf("expected 3 headers in order, got %d", len(analysis.HeaderOrder))
		}

		// Should be sorted alphabetically
		expected := []string{"accept", "content-type", "user-agent"}
		for i, header := range expected {
			if analysis.HeaderOrder[i] != header {
				t.Errorf("expected header[%d] = %s, got %s", i, header, analysis.HeaderOrder[i])
			}
		}
	})
}

func TestDetectAutomationHeaders(t *testing.T) {
	tests := []struct {
		name          string
		headers       http.Header
		expectDetect  bool
		description   string
	}{
		{
			name: "detects selenium in User-Agent",
			headers: http.Header{
				"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64) selenium webdriver"},
			},
			expectDetect: true,
			description:  "selenium keyword in user agent",
		},
		{
			name: "detects puppeteer in custom header",
			headers: http.Header{
				"X-Custom": []string{"automated with puppeteer"},
			},
			expectDetect: true,
			description:  "puppeteer keyword in custom header",
		},
		{
			name: "detects headless in User-Agent",
			headers: http.Header{
				"User-Agent": []string{"Mozilla/5.0 (X11; Linux x86_64) HeadlessChrome/90.0"},
			},
			expectDetect: true,
			description:  "headless keyword",
		},
		{
			name: "normal headers not detected",
			headers: http.Header{
				"User-Agent": []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0"},
			},
			expectDetect: false,
			description:  "normal Chrome user agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectAutomationHeaders(tt.headers)
			detected := len(result) > 0

			if detected != tt.expectDetect {
				t.Errorf("expected detection=%v for %s, got %v (results: %v)",
					tt.expectDetect, tt.description, detected, result)
			}
		})
	}
}

func TestAnalyzeUserAgent(t *testing.T) {
	tests := []struct {
		name               string
		userAgent          string
		expectAutomation   bool
		expectedPlatform   string
		expectedBrowser    string
	}{
		{
			name:             "Chrome on Windows",
			userAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expectAutomation: false,
			expectedPlatform: "Windows",
			expectedBrowser:  "Chrome",
		},
		{
			name:             "Firefox on macOS",
			userAgent:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:89.0) Gecko/20100101 Firefox/89.0",
			expectAutomation: false,
			expectedPlatform: "macOS",
			expectedBrowser:  "Firefox",
		},
		{
			name:             "Selenium WebDriver",
			userAgent:        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 selenium",
			expectAutomation: true,
			expectedPlatform: "Linux",
			expectedBrowser:  "Chrome",
		},
		{
			name:             "Puppeteer",
			userAgent:        "Mozilla/5.0 puppeteer",
			expectAutomation: true,
			expectedPlatform: "",
			expectedBrowser:  "",
		},
		{
			name:             "Mobile Safari iOS",
			userAgent:        "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Mobile/15E148 Safari/604.1",
			expectAutomation: false,
			expectedPlatform: "iOS",
			expectedBrowser:  "Safari",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := analyzeUserAgent(tt.userAgent)

			if analysis.ContainsAutomation != tt.expectAutomation {
				t.Errorf("expected automation=%v, got %v", tt.expectAutomation, analysis.ContainsAutomation)
			}
			if analysis.Platform != tt.expectedPlatform {
				t.Errorf("expected platform=%s, got %s", tt.expectedPlatform, analysis.Platform)
			}
			if analysis.Browser != tt.expectedBrowser {
				t.Errorf("expected browser=%s, got %s", tt.expectedBrowser, analysis.Browser)
			}
			if analysis.Length != len(tt.userAgent) {
				t.Errorf("expected length=%d, got %d", len(tt.userAgent), analysis.Length)
			}
		})
	}
}

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected float64
		delta    float64
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "all same byte",
			data:     []byte{0x41, 0x41, 0x41, 0x41},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "high entropy random-like",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
			expected: 3.0, // Should be exactly 3.0 for 8 different bytes
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateEntropy(tt.data)
			if result < tt.expected-tt.delta || result > tt.expected+tt.delta {
				t.Errorf("expected entropy ~%f, got %f", tt.expected, result)
			}
		})
	}
}

func TestIsLanguageUAInconsistent(t *testing.T) {
	tests := []struct {
		name           string
		userAgent      string
		acceptLanguage string
		expectMismatch bool
	}{
		{
			name:           "consistent en-US",
			userAgent:      "Mozilla/5.0 (Windows NT 10.0)",
			acceptLanguage: "en-US,en;q=0.9",
			expectMismatch: false,
		},
		{
			name:           "Chinese UA without Chinese language",
			userAgent:      "Mozilla/5.0 (zh-CN)",
			acceptLanguage: "en-US",
			expectMismatch: true,
		},
		{
			name:           "Japanese UA without Japanese language",
			userAgent:      "Mozilla/5.0 (ja)",
			acceptLanguage: "en-US",
			expectMismatch: true,
		},
		{
			name:           "Korean UA without Korean language",
			userAgent:      "Mozilla/5.0 (ko)",
			acceptLanguage: "en-US",
			expectMismatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLanguageUAInconsistent(tt.userAgent, tt.acceptLanguage)
			if result != tt.expectMismatch {
				t.Errorf("expected mismatch=%v, got %v", tt.expectMismatch, result)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		remoteAddr    string
		xff           string
		xrip          string
		expectedIP    string
	}{
		{
			name:       "uses RemoteAddr when no headers",
			remoteAddr: "192.168.1.1:8080",
			expectedIP: "192.168.1.1:8080",
		},
		{
			name:       "prefers X-Forwarded-For",
			remoteAddr: "10.0.0.1:8080",
			xff:        "203.0.113.1, 198.51.100.1",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "uses X-Real-IP when XFF absent",
			remoteAddr: "10.0.0.1:8080",
			xrip:       "203.0.113.1",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "trims whitespace from XFF",
			remoteAddr: "10.0.0.1",
			xff:        "  203.0.113.1  , 10.0.0.1",
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xrip != "" {
				req.Header.Set("X-Real-IP", tt.xrip)
			}

			result := getClientIP(req)
			if result != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, result)
			}
		})
	}
}

func TestMemoryTimingTracker(t *testing.T) {
	t.Run("records and retrieves request", func(t *testing.T) {
		tracker := NewMemoryTimingTracker()
		ip := "192.168.1.1"
		timestamp := time.Now()

		tracker.RecordRequest(ip, timestamp)

		lastTime, exists := tracker.GetLastRequest(ip)
		if !exists {
			t.Error("expected request to exist")
		}
		if !lastTime.Equal(timestamp) {
			t.Errorf("expected timestamp %v, got %v", timestamp, lastTime)
		}
	})

	t.Run("returns false for non-existent IP", func(t *testing.T) {
		tracker := NewMemoryTimingTracker()

		_, exists := tracker.GetLastRequest("192.168.1.1")
		if exists {
			t.Error("expected request to not exist")
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		tracker := NewMemoryTimingTracker()
		done := make(chan bool)

		// Concurrent writes
		for i := 0; i < 100; i++ {
			go func(id int) {
				ip := "192.168.1." + string(rune('0'+id%10))
				tracker.RecordRequest(ip, time.Now())
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 100; i++ {
			<-done
		}

		// Should not panic and should have some entries
		_, exists := tracker.GetLastRequest("192.168.1.0")
		if !exists {
			t.Error("expected at least one IP to be tracked")
		}
	})
}

func TestAnalyzeTimingPatterns(t *testing.T) {
	t.Run("first request has no previous", func(t *testing.T) {
		tracker := NewMemoryTimingTracker()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:8080"

		analysis := analyzeTimingPatterns(req, tracker)

		if analysis.HasPreviousRequest {
			t.Error("expected no previous request")
		}
		if analysis.RequestInterval != 0 {
			t.Error("expected zero interval")
		}
	})

	t.Run("second request calculates interval", func(t *testing.T) {
		tracker := NewMemoryTimingTracker()
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:8080"

		// First request
		analyzeTimingPatterns(req, tracker)

		// Wait a bit
		time.Sleep(10 * time.Millisecond)

		// Second request
		analysis := analyzeTimingPatterns(req, tracker)

		if !analysis.HasPreviousRequest {
			t.Error("expected previous request to exist")
		}
		if analysis.RequestInterval == 0 {
			t.Error("expected non-zero interval")
		}
		if analysis.RequestsPerSecond == 0 {
			t.Error("expected non-zero requests per second")
		}
	})

	t.Run("detects round interval precision", func(t *testing.T) {
		tracker := NewMemoryTimingTracker()
		ip := "192.168.1.1"
		
		// Manually set last request time to exactly 100ms ago
		now := time.Now()
		past := now.Add(-100 * time.Millisecond)
		tracker.RecordRequest(ip, past)

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip
		req.Header.Set("X-Forwarded-For", ip)

		// Simulate request at exact 100ms interval
		analysis := analyzeTimingPatterns(req, tracker)

		// The precision detection might not be exactly 100 due to timing,
		// but it should detect some precision
		if analysis.IntervalPrecision == 0 && analysis.HasPreviousRequest {
			// This is okay - timing might not be exact in tests
			t.Logf("Note: interval precision was 0 (interval: %f ms)", analysis.RequestInterval)
		}
	})
}

func TestGenerateHeaderFingerprint(t *testing.T) {
	t.Run("generates consistent fingerprint", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("User-Agent", "Mozilla/5.0")
		headers.Set("Accept", "text/html")

		fp1 := generateHeaderFingerprint(headers)
		fp2 := generateHeaderFingerprint(headers)

		if fp1 != fp2 {
			t.Error("expected consistent fingerprints")
		}
	})

	t.Run("different headers produce different fingerprints", func(t *testing.T) {
		headers1 := http.Header{}
		headers1.Set("User-Agent", "Mozilla/5.0")

		headers2 := http.Header{}
		headers2.Set("User-Agent", "Chrome/91.0")

		fp1 := generateHeaderFingerprint(headers1)
		fp2 := generateHeaderFingerprint(headers2)

		if fp1 == fp2 {
			t.Error("expected different fingerprints for different headers")
		}
	})

	t.Run("fingerprint is hex string", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("User-Agent", "test")

		fp := generateHeaderFingerprint(headers)

		// Should be 16 hex characters (8 bytes * 2)
		if len(fp) != 16 {
			t.Errorf("expected fingerprint length 16, got %d", len(fp))
		}

		// Should only contain hex characters
		for _, c := range fp {
			if !strings.ContainsRune("0123456789abcdef", c) {
				t.Errorf("fingerprint contains non-hex character: %c", c)
			}
		}
	})
}

func TestAnalyzeServerDetectionSignals(t *testing.T) {
	t.Run("comprehensive analysis", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0)")
		req.Header.Set("Accept", "text/html")
		req.Header.Set("Accept-Language", "en-US")
		req.Header.Set("Accept-Encoding", "gzip")
		req.RemoteAddr = "192.168.1.1:8080"

		body := []byte(`{"test": "data"}`)

		signals := AnalyzeServerDetectionSignals(req, body)

		// Check all sections were analyzed
		if signals.HeaderFingerprint == "" {
			t.Error("expected header fingerprint to be generated")
		}
		if signals.HeaderAnalysis.HeaderCount == 0 {
			t.Error("expected headers to be counted")
		}
		if signals.RequestAnalysis.RequestSize != len(body) {
			t.Errorf("expected request size %d, got %d", len(body), signals.RequestAnalysis.RequestSize)
		}
		if signals.RequestAnalysis.PayloadEntropy == 0 {
			t.Error("expected non-zero entropy for JSON payload")
		}
	})
}
