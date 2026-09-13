package realengine

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

func TestRealEngineCollection(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 100
	_ = cfg.SetTarget(os.Getpid(), "go")

	engine := NewEngine(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	snapshots := make(chan *agg.Snapshot, 10)
	events := make(chan agg.EventRow, 50)

	go engine.Run(ctx, snapshots, events)

	select {
	case snap := <-snapshots:
		if snap == nil {
			t.Fatal("expected non-nil snapshot")
		}
		if len(snap.Processes) == 0 {
			t.Fatal("expected at least one process from /proc")
		}
		if snap.KPIs.Threads <= 0 {
			t.Errorf("expected positive threads, got %d", snap.KPIs.Threads)
		}
		if snap.Meta.Target.PID <= 0 {
			t.Errorf("expected positive PID, got %d", snap.Meta.Target.PID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for realengine snapshot")
	}
}
