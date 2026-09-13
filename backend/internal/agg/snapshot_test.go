package agg

import "testing"

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
