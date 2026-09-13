package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/hub"
	"github.com/agentscope/agentscope/internal/protocol"
	"github.com/agentscope/agentscope/internal/storage"
)

func TestRunFunc(t *testing.T) {
	t.Setenv("MODE", "invalid")
	if err := run(); err == nil {
		t.Fatal("expected error from run() with invalid mode")
	}
}

func TestRunAppInvalidConfig(t *testing.T) {
	cfg := config.Load()
	cfg.Mode = "invalid-mode"
	if err := runApp(context.Background(), cfg); err == nil {
		t.Fatal("expected error on invalid config mode")
	}
}

func TestRunAppMockMode(t *testing.T) {
	cfg := config.Load()
	cfg.Mode = "mock"
	cfg.HttpAddr = "127.0.0.1:0" // Random available port
	cfg.DBPath = filepath.Join(t.TempDir(), "test_main.db")
	cfg.SnapshotMs = 50

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	err := runApp(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAppAutoModeFallback(t *testing.T) {
	cfg := config.Load()
	cfg.Mode = "auto"
	cfg.HttpAddr = "0.0.0.0:0" // tests BindsAllInterfaces warning
	cfg.DBPath = filepath.Join(t.TempDir(), "test_auto.db")
	cfg.SnapshotMs = 50

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	err := runApp(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAppEBPFModeFailure(t *testing.T) {
	cfg := config.Load()
	cfg.Mode = "ebpf"
	cfg.HttpAddr = "127.0.0.1:0"
	cfg.DBPath = filepath.Join(t.TempDir(), "test_ebpf.db")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// When unprivileged, Mode="ebpf" fails with error
	_ = runApp(ctx, cfg)
}

func TestRunAppSQLiteDisabled(t *testing.T) {
	cfg := config.Load()
	cfg.Mode = "mock"
	cfg.HttpAddr = "127.0.0.1:0"
	// Invalid path to trigger sqlite disabled branch
	cfg.DBPath = "/nonexistent_dir_9999/cannot_create/db.sqlite"
	cfg.SnapshotMs = 50

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)

	_ = runApp(ctx, cfg)
}

func TestRunAppServerError(t *testing.T) {
	cfg := config.Load()
	cfg.Mode = "mock"
	cfg.HttpAddr = "127.0.0.1:-1" // Invalid port causes listener error
	cfg.DBPath = filepath.Join(t.TempDir(), "test_err.db")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runApp(ctx, cfg); err == nil {
		t.Fatal("expected error on invalid port")
	}
}

func TestFanout(t *testing.T) {
	h := hub.NewHub(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	builder := agg.NewSnapshotBuilder()
	store := storage.NewMemory()

	snaps := make(chan *agg.Snapshot, 10)
	events := make(chan agg.EventRow, 10)

	fanoutCtx, fanoutCancel := context.WithCancel(context.Background())
	go fanout(fanoutCtx, h, builder, store, snaps, events, protocol.ModeMock)

	snap := &agg.Snapshot{
		Meta: agg.SnapshotMeta{Target: agg.Target{Comm: "agy"}},
		KPIs: agg.KPIs{CPUPct: 10.0},
	}
	snaps <- snap

	evt := agg.EventRow{
		ID:    "e1",
		Comm:  "agy",
		Title: "exec",
	}
	events <- evt

	time.Sleep(50 * time.Millisecond)
	fanoutCancel()

	if builder.Latest() == nil {
		t.Fatal("expected latest snapshot in builder")
	}
	hist, _ := store.QueryHistory(context.Background(), 5, "agy")
	if len(hist) != 1 {
		t.Fatalf("expected 1 history in store, got %d", len(hist))
	}
}

func TestFanoutChannelClosed(t *testing.T) {
	h := hub.NewHub(16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	builder := agg.NewSnapshotBuilder()
	store := storage.NewMemory()

	snaps := make(chan *agg.Snapshot)
	events := make(chan agg.EventRow)

	close(snaps)
	// fanout returns when snapshots channel closes
	fanout(ctx, h, builder, store, snaps, events, protocol.ModeMock)

	snaps2 := make(chan *agg.Snapshot, 1)
	events2 := make(chan agg.EventRow)
	close(events2)
	fanout(ctx, h, builder, store, snaps2, events2, protocol.ModeMock)
}
