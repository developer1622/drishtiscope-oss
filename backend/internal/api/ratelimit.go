package api

import (
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// limiter is a per-key token bucket (A06/A07: abuse and brute-force control).
type limiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	items  map[string]*bucket
	lastGC time.Time
}

func newLimiter(rate float64, burst int) *limiter {
	if rate <= 0 {
		rate = 40
	}
	if burst <= 0 {
		burst = 80
	}
	return &limiter{
		rate:   rate,
		burst:  float64(burst),
		items:  make(map[string]*bucket),
		lastGC: time.Now(),
	}
}

func (l *limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.gcLocked(now)

	b := l.items[key]
	if b == nil {
		b = &bucket{tokens: l.burst, last: now}
		l.items[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *limiter) gcLocked(now time.Time) {
	if now.Sub(l.lastGC) < time.Minute {
		return
	}
	l.lastGC = now
	for k, b := range l.items {
		if now.Sub(b.last) > 5*time.Minute {
			delete(l.items, k)
		}
	}
}
