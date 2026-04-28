package metrics

import (
	"log"
	"sync/atomic"
	"time"
)

type Collector struct {
	generated   int64
	sent        int64
	failed      int64
	startTime   time.Time
	lastLogTime time.Time
}

func NewCollector() *Collector {
	now := time.Now()
	return &Collector{
		startTime:   now,
		lastLogTime: now,
	}
}

func (c *Collector) IncrementGenerated() {
	atomic.AddInt64(&c.generated, 1)
}

func (c *Collector) IncrementSent() {
	atomic.AddInt64(&c.sent, 1)
}

func (c *Collector) IncrementFailed() {
	atomic.AddInt64(&c.failed, 1)
}

func (c *Collector) GetStats() (generated, sent, failed int64, currentEPS float64, uptime float64) {
	generated = atomic.LoadInt64(&c.generated)
	sent = atomic.LoadInt64(&c.sent)
	failed = atomic.LoadInt64(&c.failed)
	
	uptime = time.Since(c.startTime).Seconds()
	if uptime > 0 {
		currentEPS = float64(generated) / uptime
	}
	
	return
}

func (c *Collector) StartPeriodicLogging(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			generated, sent, failed, currentEPS, uptime := c.GetStats()
			log.Printf("Metrics | generated=%d sent=%d failed=%d current_eps=%.2f uptime=%.1fs",
				generated, sent, failed, currentEPS, uptime)
		}
	}()
}
