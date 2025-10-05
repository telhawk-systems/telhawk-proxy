package detection

import (
	"math"
	"net/http"
)

// analyzeRequest performs request payload analysis
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

// calculateEntropy calculates the Shannon entropy of the data
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
