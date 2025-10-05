package detection

import (
	"net/http"
)

// AnalyzeServerDetectionSignals performs comprehensive server-side detection data collection
func AnalyzeServerDetectionSignals(r *http.Request, body []byte) ServerDetectionSignals {
	return AnalyzeServerDetectionSignalsWithTracker(r, body, DefaultTracker)
}

// AnalyzeServerDetectionSignalsWithTracker performs detection with a custom timing tracker
// This allows for dependency injection and better testability
func AnalyzeServerDetectionSignalsWithTracker(
	r *http.Request,
	body []byte,
	tracker TimingTracker,
) ServerDetectionSignals {
	signals := ServerDetectionSignals{}

	// Analyze HTTP headers
	signals.HeaderAnalysis = analyzeHeaders(r.Header)
	signals.HeaderFingerprint = generateHeaderFingerprint(r.Header)

	// Analyze request payload
	signals.RequestAnalysis = analyzeRequest(r, body)

	// Analyze timing patterns
	signals.TimingAnalysis = analyzeTimingPatterns(r, tracker)

	return signals
}
