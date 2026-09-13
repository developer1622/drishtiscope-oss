package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/agentscope/agentscope/internal/agg"
)

func sampleSnap(comm string, cpu float64) *agg.Snapshot {
	return &agg.Snapshot{
		Meta: agg.SnapshotMeta{Target: agg.Target{PID: 7, Comm: comm}},
		KPIs: agg.KPIs{CPUPct: cpu, Threads: 4, RSSBytes: 1024, SyscallsPerSec: 10},
	}
}

func TestMemoryStoreRoundTrip(t *testing.T) {
	m := NewMemory()
	if err := m.RecordSnapshot(sampleSnap("agy", 12)); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordEvent(agg.EventRow{ID: "e1", Title: "exec", Comm: "agy"}); err != nil {
		t.Fatal(err)
	}
	hist, err := m.QueryHistory(context.Background(), 10, "agy")
	if err != nil || len(hist) != 1 || hist[0].CPUPct != 12 {
		t.Fatalf("hist=%v err=%v", hist, err)
	}
	ev, err := m.QueryEvents(context.Background(), 10)
	if err != nil || len(ev) != 1 || ev[0].ID != "e1" {
		t.Fatalf("ev=%v err=%v", ev, err)
	}
	other, err := m.QueryHistory(context.Background(), 10, "nginx")
	if err != nil || len(other) != 0 {
		t.Fatalf("filter: %v %v", other, err)
	}
}

func TestSQLiteStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.RecordSnapshot(sampleSnap("agy", 9.5)); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordEvent(agg.EventRow{
		ID: "evt-1", TS: "2026-01-01T00:00:00Z", Severity: "info",
		Category: "process", PID: 7, Comm: "agy", Title: "exec", Detail: "/bin/agy",
		Attrs: map[string]interface{}{"exe": "/bin/agy"},
	}); err != nil {
		t.Fatal(err)
	}

	hist, err := db.QueryHistory(context.Background(), 10, "agy")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 || hist[0].Comm != "agy" || hist[0].CPUPct != 9.5 {
		t.Fatalf("hist %+v", hist)
	}
	ev, err := db.QueryEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 || ev[0].Title != "exec" {
		t.Fatalf("events %+v", ev)
	}
	if ev[0].Attrs["exe"] != "/bin/agy" {
		t.Fatalf("attrs %+v", ev[0].Attrs)
	}
}

func TestNopStore(t *testing.T) {
	var n Nop
	if err := n.RecordSnapshot(sampleSnap("x", 1)); err != nil {
		t.Fatal(err)
	}
	h, err := n.QueryHistory(context.Background(), 1, "")
	if err != nil || len(h) != 0 {
		t.Fatalf("%v %v", h, err)
	}
}

func TestMemoryClampsLimit(t *testing.T) {
	m := NewMemory()
	for i := 0; i < 20; i++ {
		_ = m.RecordSnapshot(sampleSnap("agy", float64(i)))
	}
	h, err := m.QueryHistory(context.Background(), 5, "")
	if err != nil || len(h) != 5 {
		t.Fatalf("len=%d err=%v", len(h), err)
	}
}
