package agg

import (
	"sync"
	"testing"
	"time"
)

func TestCounterRatesAfterWindow(t *testing.T) {
	c := &Counter{lastUpdate: time.Now().Add(-2 * time.Second)}
	c.Add(200)
	r1 := c.Rate1s()
	if r1 < 50 || r1 > 200 {
		t.Fatalf("rate1s=%v", r1)
	}
	r5 := c.Rate5s()
	if r5 <= 0 {
		t.Fatalf("rate5s=%v", r5)
	}
}

func TestRateMapUnknownKey(t *testing.T) {
	rm := NewRateMap()
	if rm.Rate1s("missing") != 0 {
		t.Fatal("missing key")
	}
}

func TestRateMapConcurrentAdd(t *testing.T) {
	rm := NewRateMap()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				rm.Add("openat", 1)
			}
		}()
	}
	wg.Wait()
}
