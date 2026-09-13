package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
)

type errorStore struct {
	Nop
}

func (errorStore) RecordSnapshot(*agg.Snapshot) error {
	return errors.New("snapshot error")
}

func (errorStore) RecordEvent(agg.EventRow) error {
	return errors.New("event error")
}

func TestAsyncStorage(t *testing.T) {
	mem := NewMemory()
	async := WrapAsync(mem, 16)
	defer async.Close()

	snap := sampleSnap("agy", 25.0)
	if err := async.RecordSnapshot(snap); err != nil {
		t.Fatal(err)
	}

	evt := agg.EventRow{
		ID:       "evt-async-1",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Severity: "info",
		Category: "process",
		PID:      1234,
		Comm:     "agy",
		Title:    "exec",
		Detail:   "/bin/agy",
	}
	if err := async.RecordEvent(evt); err != nil {
		t.Fatal(err)
	}

	// Give async loop a moment to drain
	time.Sleep(50 * time.Millisecond)

	hist, err := async.QueryHistory(context.Background(), 10, "agy")
	if err != nil || len(hist) != 1 {
		t.Fatalf("expected 1 history point, got %d, err=%v", len(hist), err)
	}

	events, err := async.QueryEvents(context.Background(), 10)
	if err != nil || len(events) != 1 {
		t.Fatalf("expected 1 event, got %d, err=%v", len(events), err)
	}
}

func TestAsyncStorageDefaultsAndErrors(t *testing.T) {
	// Tests default queue size (0 -> 256)
	errSt := errorStore{}
	async := WrapAsync(errSt, 0)
	defer async.Close()

	_ = async.RecordSnapshot(sampleSnap("agy", 1.0))
	_ = async.RecordEvent(agg.EventRow{ID: "err-evt"})
	time.Sleep(50 * time.Millisecond)
}

func TestAsyncQueueOverflow(t *testing.T) {
	mem := NewMemory()
	async := WrapAsync(mem, 1)

	// Close immediately so loop doesn't read
	_ = async.Close()

	// Writing to full/stopped channel should drop gracefully without panicking
	for i := 0; i < 10; i++ {
		_ = async.RecordSnapshot(sampleSnap("agy", float64(i)))
		_ = async.RecordEvent(agg.EventRow{ID: "e", Comm: "agy"})
	}
}

func TestSQLitePruningAndThrottling(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_prune.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Nil snapshot should return nil
	if err := db.RecordSnapshot(nil); err != nil {
		t.Fatal(err)
	}

	// Rapid snapshot records should be throttled (< 1500ms)
	s1 := sampleSnap("agy", 10.0)
	if err := db.RecordSnapshot(s1); err != nil {
		t.Fatal(err)
	}
	s2 := sampleSnap("agy", 20.0)
	// This second record should be throttled
	if err := db.RecordSnapshot(s2); err != nil {
		t.Fatal(err)
	}

	// History should only have 1 point because second was throttled
	hist, err := db.QueryHistory(context.Background(), 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 {
		t.Fatalf("expected 1 record due to throttling, got %d", len(hist))
	}

	// Query with comm filter
	histComm, err := db.QueryHistory(context.Background(), 10, "agy")
	if err != nil || len(histComm) != 1 {
		t.Fatalf("expected 1 record for comm agy, got %d", len(histComm))
	}

	// Record 40 events to trigger pruneEvery (32) and WAL checkpoint
	for i := 0; i < 40; i++ {
		err := db.RecordEvent(agg.EventRow{
			ID:       time.Now().Format("20060102150405.000000000"),
			TS:       time.Now().UTC().Format(time.RFC3339),
			Severity: "info",
			Category: "process",
			PID:      1000 + i,
			Comm:     "agy",
			Title:    "event",
			Detail:   "test",
			Attrs:    map[string]interface{}{"idx": i},
		})
		if err != nil {
			t.Fatalf("event %d failed: %v", i, err)
		}
	}

	// Query events
	evts, err := db.QueryEvents(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(evts) != 20 {
		t.Fatalf("expected 20 events, got %d", len(evts))
	}
}

func TestSQLiteDefaultPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	db, err := Open("")
	if err != nil {
		t.Fatalf("open default db failed: %v", err)
	}
	_ = db.Close()
}

func TestMemoryStoreLimitsAndCaps(t *testing.T) {
	m := NewMemory()
	// Test nil snapshot
	if err := m.RecordSnapshot(nil); err != nil {
		t.Fatal(err)
	}

	// Set low limits to test capping
	m.maxSnap = 2
	m.maxEvt = 2

	_ = m.RecordSnapshot(sampleSnap("agy", 1.0))
	_ = m.RecordSnapshot(sampleSnap("agy", 2.0))
	_ = m.RecordSnapshot(sampleSnap("agy", 3.0))

	if len(m.snapshots) != 2 {
		t.Fatalf("expected 2 snapshots after cap, got %d", len(m.snapshots))
	}

	_ = m.RecordEvent(agg.EventRow{ID: "e1", Title: "1", Comm: "agy"})
	_ = m.RecordEvent(agg.EventRow{ID: "e2", Title: "2", Comm: "agy"})
	_ = m.RecordEvent(agg.EventRow{ID: "e3", Title: "3", Comm: "agy"})

	if len(m.events) != 2 {
		t.Fatalf("expected 2 events after cap, got %d", len(m.events))
	}

	evts, err := m.QueryEvents(context.Background(), 5)
	if err != nil || len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d, err=%v", len(evts), err)
	}
	_ = m.Close()
}

func TestNopStoreAllMethods(t *testing.T) {
	var n Nop
	if err := n.RecordSnapshot(sampleSnap("x", 1)); err != nil {
		t.Fatal(err)
	}
	if err := n.RecordEvent(agg.EventRow{ID: "nop"}); err != nil {
		t.Fatal(err)
	}
	h, err := n.QueryHistory(context.Background(), 1, "")
	if err != nil || len(h) != 0 {
		t.Fatalf("%v %v", h, err)
	}
	evs, err := n.QueryEvents(context.Background(), 1)
	if err != nil || len(evs) != 0 {
		t.Fatalf("%v %v", evs, err)
	}
	if err := n.Close(); err != nil {
		t.Fatal(err)
	}
}
