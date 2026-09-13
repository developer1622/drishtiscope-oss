package agg

import (
	"sync"
	"time"
)

type Counter struct {
	mu         sync.Mutex
	lastUpdate time.Time
	currentSec uint64
	rate1s     float64
	rate5s     float64
}

func NewCounter() *Counter {
	return &Counter{
		lastUpdate: time.Now(),
	}
}

func (c *Counter) Add(delta uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.currentSec += delta
	c.update()
}

func (c *Counter) update() {
	now := time.Now()
	elapsed := now.Sub(c.lastUpdate).Seconds()

	if elapsed >= 1.0 {
		// Update EWAs
		// For 1s EWA, alpha = 1.0 (essentially instantaneous)
		// For 5s EWA, alpha = 2 / (5 + 1) = 1/3

		rate := float64(c.currentSec) / elapsed

		c.rate1s = rate

		alpha := 1.0 / 3.0
		if c.rate5s == 0 {
			c.rate5s = rate
		} else {
			c.rate5s = (alpha * rate) + ((1.0 - alpha) * c.rate5s)
		}

		c.currentSec = 0
		c.lastUpdate = now
	}
}

func (c *Counter) Rate1s() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.update()
	return c.rate1s
}

func (c *Counter) Rate5s() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.update()
	return c.rate5s
}

type RateMap struct {
	mu       sync.RWMutex
	counters map[string]*Counter
}

func NewRateMap() *RateMap {
	return &RateMap{
		counters: make(map[string]*Counter),
	}
}

func (rm *RateMap) Add(key string, delta uint64) {
	rm.mu.RLock()
	c, ok := rm.counters[key]
	rm.mu.RUnlock()

	if !ok {
		rm.mu.Lock()
		c, ok = rm.counters[key]
		if !ok {
			c = NewCounter()
			rm.counters[key] = c
		}
		rm.mu.Unlock()
	}

	c.Add(delta)
}

func (rm *RateMap) Rate1s(key string) float64 {
	rm.mu.RLock()
	c, ok := rm.counters[key]
	rm.mu.RUnlock()

	if !ok {
		return 0
	}
	return c.Rate1s()
}
