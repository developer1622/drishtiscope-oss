package mock

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

func TestMockGeneratorWithCustomTargetComm(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 20
	if err := cfg.SetTarget(0, "custom-worker"); err != nil {
		t.Fatal(err)
	}

	g := NewMockGenerator(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	snaps := make(chan *agg.Snapshot, 10)
	evts := make(chan agg.EventRow, 32)
	go g.Run(ctx, snaps, evts)

	select {
	case snap := <-snaps:
		if snap.Meta.Target.Comm != "custom-worker" {
			t.Fatalf("expected custom-worker comm, got %s", snap.Meta.Target.Comm)
		}
		if len(snap.Processes) == 0 {
			t.Fatal("expected processes in snapshot")
		}
		if snap.Processes[0].Comm != "custom-worker" {
			t.Fatalf("expected custom-worker as first row, got %s", snap.Processes[0].Comm)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for snapshot")
	}
}

func TestMockGeneratorCopilotCmdlineTarget(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 20
	if err := cfg.SetTarget(0, "copilot"); err != nil {
		t.Fatal(err)
	}

	g := NewMockGenerator(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	snaps := make(chan *agg.Snapshot, 10)
	evts := make(chan agg.EventRow, 32)
	go g.Run(ctx, snaps, evts)

	select {
	case snap := <-snaps:
		if snap.Meta.Target.Comm != "copilot" {
			t.Fatalf("expected filter comm copilot, got %s", snap.Meta.Target.Comm)
		}
		found := false
		for _, p := range snap.Processes {
			if strings.Contains(strings.ToLower(p.Cmdline), "copilot") {
				found = true
				if p.PID != 58785 {
					t.Fatalf("expected copilot pid 58785, got %d", p.PID)
				}
				if p.Comm != "MainThread" {
					t.Fatalf("expected kernel comm MainThread, got %s", p.Comm)
				}
			}
		}
		if !found {
			t.Fatal("expected a process whose cmdline contains copilot")
		}
		if snap.Meta.Target.PID != 58785 {
			t.Fatalf("expected resolved pid 58785, got %d", snap.Meta.Target.PID)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for snapshot")
	}
}

func TestMockGeneratorWithTargetPID(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 20
	// Target node PID 24648
	if err := cfg.SetTarget(24648, ""); err != nil {
		t.Fatal(err)
	}

	g := NewMockGenerator(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	snaps := make(chan *agg.Snapshot, 10)
	evts := make(chan agg.EventRow, 32)
	go g.Run(ctx, snaps, evts)

	select {
	case snap := <-snaps:
		if snap.Meta.Target.PID != 24648 {
			t.Fatalf("expected PID 24648, got %d", snap.Meta.Target.PID)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for snapshot")
	}
}

func TestMockGeneratorEventTriggers(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 100 // tick = 100ms
	_ = cfg.SetTarget(22786, "agy")

	g := NewMockGenerator(cfg)
	events := make(chan agg.EventRow, 100)

	// Test connect event trigger (tick % (4000/100) == 0 -> tick = 40)
	g.tick = 40
	target := agg.ProcessRow{PID: 22786, Comm: "agy", CPUPct: 45.0}
	g.emitEventsForTarget(events, target, 10.0)

	// Test file open trigger (tick = 30)
	g.tick = 30
	g.emitEventsForTarget(events, target, 10.0)

	// Test agy-specific batch (tick = 15000/100 = 150)
	g.tick = 150
	g.emitEventsForTarget(events, target, 15.0)

	// Test security alert (tick = 60000/100 = 600)
	g.tick = 600
	g.emitEventsForTarget(events, target, 60.0)

	// Test child exec (tick = 90000/100 = 900)
	g.tick = 900
	g.emitEventsForTarget(events, target, 90.0)

	// Test CPU burst (tick = 45000/100 = 450)
	g.tick = 450
	g.emitEventsForTarget(events, target, 45.0)

	// Non-agy targets
	targetOther := agg.ProcessRow{PID: 9999, Comm: "payments-agent", CPUPct: 20.0}
	g.tick = 40
	g.emitEventsForTarget(events, targetOther, 10.0)
	g.tick = 30
	g.emitEventsForTarget(events, targetOther, 10.0)
	g.tick = 600
	g.emitEventsForTarget(events, targetOther, 60.0)
	g.tick = 900
	g.emitEventsForTarget(events, targetOther, 90.0)

	if len(events) < 5 {
		t.Fatalf("expected multiple events, got %d", len(events))
	}
}

func TestAppendCappedFunctions(t *testing.T) {
	// SeriesPoint cap
	var sp []agg.SeriesPoint
	for i := 0; i < 10; i++ {
		sp = appendCapped(sp, 5, agg.SeriesPoint{T: int64(i), CPUPct: float64(i)})
	}
	if len(sp) != 5 {
		t.Fatalf("expected len 5, got %d", len(sp))
	}
	if sp[0].T != 5 || sp[4].T != 9 {
		t.Fatalf("unexpected slice window: %v", sp)
	}

	// IOPoint cap
	var iop []agg.IOPoint
	for i := 0; i < 10; i++ {
		iop = appendIOCapped(iop, 5, agg.IOPoint{T: int64(i), RBPS: float64(i)})
	}
	if len(iop) != 5 {
		t.Fatalf("expected len 5, got %d", len(iop))
	}

	// NetPoint cap
	var np []agg.NetPoint
	for i := 0; i < 10; i++ {
		np = appendNetCapped(np, 5, agg.NetPoint{T: int64(i), TxBPS: float64(i)})
	}
	if len(np) != 5 {
		t.Fatalf("expected len 5, got %d", len(np))
	}
}

func TestConnectsPerSec(t *testing.T) {
	g := NewMockGenerator(config.Load())
	g.tick = 75
	if g.connectsPerSec() != 0.2 {
		t.Fatalf("expected 0.2, got %f", g.connectsPerSec())
	}
	g.tick = 74
	if g.connectsPerSec() != 0.02 {
		t.Fatalf("expected 0.02, got %f", g.connectsPerSec())
	}
}

func TestUid(t *testing.T) {
	id1 := uid()
	id2 := uid()
	if id1 == "" || id2 == "" {
		t.Fatal("empty uid")
	}
}

func TestBuildSyscallStatsErrors(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	stats := buildSyscallStats(5000.0, rng)
	if len(stats) == 0 {
		t.Fatal("empty syscall stats")
	}
}
