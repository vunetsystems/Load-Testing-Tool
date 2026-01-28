package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Collector collects metrics
type Collector struct {
	mu sync.RWMutex
	
	// Counters
	generatedError int64
	generatedTrans int64
	generatedBoth  int64
	sent           int64
	failed         int64
	
	// Latency tracking
	latencies []time.Duration
	
	// EPS tracking
	startTime     time.Time
	lastEPSCheck  time.Time
	lastSentCount int64
	currentEPS    float64
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
	now := time.Now()
	return &Collector{
		latencies:    make([]time.Duration, 0, 1000),
		startTime:    now,
		lastEPSCheck: now,
	}
}

// IncrementGenerated increments the generated counter
func (c *Collector) IncrementGenerated(msgType string) {
	switch msgType {
	case "error":
		atomic.AddInt64(&c.generatedError, 1)
	case "trans":
		atomic.AddInt64(&c.generatedTrans, 1)
	case "both":
		atomic.AddInt64(&c.generatedBoth, 1)
	}
}

// IncrementSent increments the sent counter
func (c *Collector) IncrementSent() {
	atomic.AddInt64(&c.sent, 1)
}

// IncrementFailed increments the failed counter
func (c *Collector) IncrementFailed() {
	atomic.AddInt64(&c.failed, 1)
}

// RecordLatency records a latency measurement
func (c *Collector) RecordLatency(latency time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.latencies = append(c.latencies, latency)
	
	// Keep only last 1000 latencies
	if len(c.latencies) > 1000 {
		c.latencies = c.latencies[len(c.latencies)-1000:]
	}
}

// UpdateEPS updates the current EPS calculation
func (c *Collector) UpdateEPS() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(c.lastEPSCheck).Seconds()
	
	if elapsed >= 1.0 {
		currentSent := atomic.LoadInt64(&c.sent)
		sentSinceLastCheck := currentSent - c.lastSentCount
		c.currentEPS = float64(sentSinceLastCheck) / elapsed
		c.lastSentCount = currentSent
		c.lastEPSCheck = now
	}
}

// GetStats returns current statistics
func (c *Collector) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return Stats{
		GeneratedError: atomic.LoadInt64(&c.generatedError),
		GeneratedTrans: atomic.LoadInt64(&c.generatedTrans),
		GeneratedBoth:  atomic.LoadInt64(&c.generatedBoth),
		Sent:           atomic.LoadInt64(&c.sent),
		Failed:         atomic.LoadInt64(&c.failed),
		CurrentEPS:     c.currentEPS,
		Uptime:         time.Since(c.startTime),
	}
}

// Stats represents collected statistics
type Stats struct {
	GeneratedError int64
	GeneratedTrans int64
	GeneratedBoth  int64
	Sent           int64
	Failed         int64
	CurrentEPS     float64
	Uptime         time.Duration
}

// TotalGenerated returns total messages generated
func (s *Stats) TotalGenerated() int64 {
	return s.GeneratedError + s.GeneratedTrans + s.GeneratedBoth
}
