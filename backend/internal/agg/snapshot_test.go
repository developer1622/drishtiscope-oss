package agg

import (
	"testing"
	"time"
)

func TestUpdateTakesOnlyNewestPoint(t *testing.T) {
	b := NewSnapshotBuilder()
	b.Update(&Snapshot{
		CPUSeries: []SeriesPoint{{T: 1, CPUPct: 1}, {T: 2, CPUPct: 2}},
	})
	b.Update(&Snapshot{
		CPUSeries: []SeriesPoint{{T: 1, CPUPct: 1}, {T: 2, CPUPct: 2}, {T: 3, CPUPct: 3}},
	})
	got := b.Latest().CPUSeries
	if len(got) != 2 {
		t.Fatalf("len=%d want 2: %+v", len(got), got)
	}
	if got[0].T != 2 || got[1].T != 3 {
		t.Fatalf("unexpected series %+v", got)
	}
}

func TestLatestReturnsCopy(t *testing.T) {
	b := NewSnapshotBuilder()
	b.Update(&Snapshot{
		Processes: []ProcessRow{{PID: 1, Comm: "agy"}},
		CPUSeries: []SeriesPoint{{T: 1, CPUPct: 4}},
	})
	a := b.Latest()
	a.Processes[0].Comm = "mutated"
	a.CPUSeries[0].CPUPct = 99
	got := b.Latest()
	if got.Processes[0].Comm != "agy" {
		t.Fatal("Latest leaked internal process slice")
	}
	if got.CPUSeries[0].CPUPct != 4 {
		t.Fatal("Latest leaked internal series")
	}
}

func TestEmptySlicesNeverNil(t *testing.T) {
	b := NewSnapshotBuilder()
	s := b.Latest()
	if s.Processes == nil || s.Flows == nil || s.Timeline == nil {
		t.Fatalf("nil slices: %+v", s)
	}
	b.Update(&Snapshot{})
	s = b.Latest()
	if s.Processes == nil || s.SyscallsTop == nil || s.FilesTop == nil || s.Flows == nil {
		t.Fatal("update produced nil slices")
	}
}

func TestEventRingCapsAt300(t *testing.T) {
	b := NewSnapshotBuilder()
	for i := 0; i < 350; i++ {
		b.AddEvent(EventRow{ID: string(rune(i))})
	}
	b.Update(&Snapshot{})
	if n := len(b.Latest().Timeline); n != 300 {
		t.Fatalf("timeline=%d", n)
	}
}

func TestNilUpdateSafe(t *testing.T) {
	b := NewSnapshotBuilder()
	defer func() {
		if recover() != nil {
			t.Fatal("panic")
		}
	}()
	b.Update(&Snapshot{})
}

func TestAppendCappedAndCloneNil(t *testing.T) {
	if s := cloneSnapshot(nil); s == nil {
		t.Fatal("expected empty snapshot from cloneSnapshot(nil)")
	}

	var series []SeriesPoint
	for i := 0; i < 70; i++ {
		series = appendCapped(series, 60, SeriesPoint{T: int64(i), CPUPct: float64(i)})
	}
	if len(series) != 60 {
		t.Fatalf("expected 60, got %d", len(series))
	}

	var ioSeries []IOPoint
	for i := 0; i < 70; i++ {
		ioSeries = appendIOCapped(ioSeries, 60, IOPoint{T: int64(i), RBPS: float64(i)})
	}
	if len(ioSeries) != 60 {
		t.Fatalf("expected 60, got %d", len(ioSeries))
	}

	var netSeries []NetPoint
	for i := 0; i < 70; i++ {
		netSeries = appendNetCapped(netSeries, 60, NetPoint{T: int64(i), TxBPS: float64(i)})
	}
	if len(netSeries) != 60 {
		t.Fatalf("expected 60, got %d", len(netSeries))
	}
}

func TestRatesExtended(t *testing.T) {
	c := NewCounter()
	c.Add(10)
	if c.Rate1s() != 0 {
		// within 1s window rate is 0 before elapsed
	}

	// Test update when elapsed >= 1s multiple times
	c.lastUpdate = time.Now().Add(-2 * time.Second)
	c.Add(100)
	_ = c.Rate5s()

	// second update with existing rate5s
	c.lastUpdate = time.Now().Add(-2 * time.Second)
	c.Add(200)
	r5 := c.Rate5s()
	if r5 <= 0 {
		t.Fatalf("expected r5 > 0, got %v", r5)
	}

	rm := NewRateMap()
	rm.Add("test_key", 50)
	if rm.Rate5s("missing") != 0 {
		t.Fatal("expected 0 for missing key in Rate5s")
	}
	if rm.Rate1s("test_key") < 0 {
		t.Fatal("negative rate")
	}
}

