package detection

import (
	"sync"
	"time"
)

// TimingTracker stores and analyzes request timing patterns
type TimingTracker interface {
	RecordRequest(ip string, timestamp time.Time)
	GetLastRequest(ip string) (time.Time, bool)
}

// MemoryTimingTracker implements TimingTracker using in-memory storage
// Note: In production, consider using Redis or a database for distributed tracking
type MemoryTimingTracker struct {
	mu           sync.RWMutex
	lastRequests map[string]time.Time
}

// NewMemoryTimingTracker creates a new in-memory timing tracker
func NewMemoryTimingTracker() *MemoryTimingTracker {
	return &MemoryTimingTracker{
		lastRequests: make(map[string]time.Time),
	}
}

// RecordRequest records the timestamp of a request from the given IP
func (t *MemoryTimingTracker) RecordRequest(ip string, timestamp time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastRequests[ip] = timestamp
}

// GetLastRequest retrieves the last request time for the given IP
func (t *MemoryTimingTracker) GetLastRequest(ip string) (time.Time, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	lastTime, exists := t.lastRequests[ip]
	return lastTime, exists
}

// DefaultTracker is the global timing tracker instance
// This maintains backward compatibility with the original global variable
var DefaultTracker = NewMemoryTimingTracker()
