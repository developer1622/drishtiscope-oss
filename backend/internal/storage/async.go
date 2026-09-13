package storage

import (
	"context"
	"log"

	"github.com/agentscope/agentscope/internal/agg"
)

// Async decouples the hot snapshot/event path from disk IO.
type Async struct {
	inner Store
	snaps chan *agg.Snapshot
	evts  chan agg.EventRow
	stop  chan struct{}
	done  chan struct{}
}

func WrapAsync(inner Store, queue int) *Async {
	if queue <= 0 {
		queue = 256
	}
	a := &Async{
		inner: inner,
		snaps: make(chan *agg.Snapshot, queue),
		evts:  make(chan agg.EventRow, queue),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go a.loop()
	return a
}

func (a *Async) loop() {
	defer close(a.done)
	for {
		select {
		case <-a.stop:
			return
		case snap := <-a.snaps:
			if err := a.inner.RecordSnapshot(snap); err != nil {
				log.Printf("storage: snapshot write: %v", err)
			}
		case e := <-a.evts:
			if err := a.inner.RecordEvent(e); err != nil {
				log.Printf("storage: event write: %v", err)
			}
		}
	}
}

func (a *Async) RecordSnapshot(snap *agg.Snapshot) error {
	select {
	case a.snaps <- snap:
	default:
		log.Printf("storage: snapshot queue full, dropping")
	}
	return nil
}

func (a *Async) RecordEvent(e agg.EventRow) error {
	select {
	case a.evts <- e:
	default:
		log.Printf("storage: event queue full, dropping")
	}
	return nil
}

func (a *Async) QueryHistory(ctx context.Context, limit int, comm string) ([]SnapshotHistoryPoint, error) {
	return a.inner.QueryHistory(ctx, limit, comm)
}

func (a *Async) QueryEvents(ctx context.Context, limit int) ([]agg.EventRow, error) {
	return a.inner.QueryEvents(ctx, limit)
}

func (a *Async) Close() error {
	select {
	case <-a.stop:
	default:
		close(a.stop)
	}
	<-a.done
	return a.inner.Close()
}
