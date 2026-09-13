package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/api"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/ebpfagent"
	"github.com/agentscope/agentscope/internal/hub"
	"github.com/agentscope/agentscope/internal/mock"
	"github.com/agentscope/agentscope/internal/protocol"
	"github.com/agentscope/agentscope/internal/realengine"
	"github.com/agentscope/agentscope/internal/storage"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	if err := run(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func run() error {
	cfg := config.Load()
	cfg.ParseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runApp(ctx, cfg)
}

func runApp(ctx context.Context, cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if !cfg.HasAuth() {
		log.Printf("WARNING: AUTH_TOKEN is unset — API is loopback-only (OWASP A01/A07). Set AUTH_TOKEN to expose it.")
	}
	if cfg.BindsAllInterfaces() && !cfg.HasAuth() {
		log.Printf("WARNING: listening on %s without AUTH_TOKEN; remote clients will receive 403", cfg.HttpAddr)
	}

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	h := hub.NewHub(cfg.MaxWSClients)
	go h.Run(runCtx)

	var store storage.Store = storage.Nop{}
	if db, err := storage.Open(cfg.DBPath); err != nil {
		log.Printf("sqlite disabled: %v", err)
	} else {
		store = storage.WrapAsync(db, 256)
		defer func() { _ = store.Close() }()
	}

	builder := agg.NewSnapshotBuilder()
	snapshots := make(chan *agg.Snapshot, 32)
	events := make(chan agg.EventRow, 512)

	mode := protocol.ModeReal
	var modeReason string
	var cleanup func()

	if cfg.Mode == "mock" {
		mode = protocol.ModeMock
		go mock.NewMockGenerator(cfg).Run(runCtx, snapshots, events)
	} else if cfg.Mode == "ebpf" {
		cfn, err := ebpfagent.TryLoad(cfg, snapshots, events)
		if err != nil {
			return err
		}
		cleanup = cfn
		mode = protocol.ModeEBPF
	} else if cfg.Mode == "real" {
		mode = protocol.ModeReal
		go realengine.NewEngine(cfg).Run(runCtx, snapshots, events)
	} else {
		// Default mode: "auto"
		// Attempt eBPF probe attachment first; if unprivileged or fails, run RealEngine Linux /proc collector
		cfn, err := ebpfagent.TryLoad(cfg, snapshots, events)
		if err == nil {
			cleanup = cfn
			mode = protocol.ModeEBPF
			log.Printf("eBPF kernel probes successfully loaded and attached to kernel tracepoints")
		} else {
			modeReason = err.Error()
			mode = protocol.ModeReal
			log.Printf("eBPF probe attach requires CAP_BPF/root (%v); active in Real-Time Linux /proc Engine (mode=real)", err)
			go realengine.NewEngine(cfg).Run(runCtx, snapshots, events)
		}
	}
	if cleanup != nil {
		defer cleanup()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fanout(runCtx, h, builder, store, snapshots, events, mode)
	}()

	srv := api.NewServer(cfg, h, builder, mode, modeReason, store)
	log.Printf("DrishtiScope listening on %s (mode=%s comm=%q pid=%d db=%q)",
		cfg.HttpAddr, mode, cfg.TargetComm(), cfg.TargetPID(), cfg.DBPath)
	err := srv.Start(runCtx)
	runCancel()
	wg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	log.Printf("shutdown complete")
	return nil
}

func fanout(
	ctx context.Context,
	h *hub.Hub,
	builder *agg.SnapshotBuilder,
	store storage.Store,
	snapshots <-chan *agg.Snapshot,
	events <-chan agg.EventRow,
	mode string,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-snapshots:
			if !ok {
				return
			}
			builder.Update(snap)
			_ = store.RecordSnapshot(snap)
			if b, err := protocol.Marshal(protocol.KindSnapshot, mode, snap); err == nil {
				h.Broadcast(b)
			}
		case evt, ok := <-events:
			if !ok {
				return
			}
			builder.AddEvent(evt)
			_ = store.RecordEvent(evt)
			if b, err := protocol.Marshal(protocol.KindEvent, mode, evt); err == nil {
				h.Broadcast(b)
			}
		}
	}
}
