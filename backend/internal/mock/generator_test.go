package mock

import (
	"context"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

func TestMockGeneratorEmitsSnapshot(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 40
	if err := cfg.SetTarget(0, "agy"); err != nil {
		t.Fatal(err)
	}
	g := NewMockGenerator(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snaps := make(chan *agg.Snapshot, 8)
	evts := make(chan agg.EventRow, 32)
	go g.Run(ctx, snaps, evts)

	select {
	case snap := <-snaps:
		if snap.Meta.Target.Comm != "agy" {
			t.Fatalf("comm %q", snap.Meta.Target.Comm)
		}
		if len(snap.Processes) == 0 {
			t.Fatal("empty process table")
		}
		if snap.KPIs.SyscallsPerSec <= 0 {
			t.Fatal("syscalls")
		}
		if len(snap.SyscallsTop) == 0 || len(snap.FilesTop) == 0 || len(snap.Flows) == 0 {
			t.Fatal("missing tables")
		}
	case <-ctx.Done():
		t.Fatal("no snapshot produced")
	}
}

func TestBuildSyscallStats(t *testing.T) {
	stats := buildSyscallStats(1000, NewMockGenerator(config.Load()).rng)
	if len(stats) != len(mockSyscalls) {
		t.Fatalf("len %d", len(stats))
	}
	var sum float64
	for _, s := range stats {
		if s.Name == "" || s.CountS < 0 {
			t.Fatalf("%+v", s)
		}
		sum += s.CountS
	}
	if sum < 700 || sum > 1300 {
		t.Fatalf("sum=%v", sum)
	}
}

func TestBuildFileStatsForComm(t *testing.T) {
	agy := buildFileStatsForComm("agy", NewMockGenerator(config.Load()).rng)
	pay := buildFileStatsForComm("payments-agent", NewMockGenerator(config.Load()).rng)
	if len(agy) == 0 || len(pay) == 0 {
		t.Fatal("empty")
	}
	if agy[0].Path == pay[0].Path {
		t.Fatal("expected different file sets")
	}
}
