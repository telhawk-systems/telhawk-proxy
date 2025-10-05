package detection

import (
	"net/http"
	"time"
)

// analyzeTimingPatterns analyzes request timing patterns
func analyzeTimingPatterns(r *http.Request, tracker TimingTracker) TimingAnalysis {
	analysis := TimingAnalysis{}

	clientIP := getClientIP(r)
	now := time.Now()

	if lastTime, exists := tracker.GetLastRequest(clientIP); exists {
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

	tracker.RecordRequest(clientIP, now)
	return analysis
}
