package storage

import (
	"context"
	"sync"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

// Memory is an in-process Store used by unit tests.
type Memory struct {
	mu        sync.Mutex
	nextID    int64
	snapshots []SnapshotHistoryPoint
	events    []agg.EventRow
	maxSnap   int
	maxEvt    int
}

func NewMemory() *Memory {
	return &Memory{maxSnap: 5000, maxEvt: 5000, nextID: 1}
}

func (m *Memory) RecordSnapshot(snap *agg.Snapshot) error {
	if snap == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots = append(m.snapshots, SnapshotHistoryPoint{
		ID:       m.nextID,
		T:        time.Now().UnixMilli(),
		Comm:     snap.Meta.Target.Comm,
		PID:      snap.Meta.Target.PID,
		CPUPct:   snap.KPIs.CPUPct,
		Threads:  snap.KPIs.Threads,
		RSSBytes: snap.KPIs.RSSBytes,
		OpenFDs:  snap.KPIs.OpenFDs,
		SyscallS: snap.KPIs.SyscallsPerSec,
		ErrS:     snap.KPIs.ErrSyscallsPerSec,
		NetTxBps: snap.KPIs.NetBpsTx,
		NetRxBps: snap.KPIs.NetBpsRx,
		DiskRBps: snap.KPIs.DiskBpsR,
		DiskWBps: snap.KPIs.DiskBpsW,
	})
	m.nextID++
	if len(m.snapshots) > m.maxSnap {
		m.snapshots = m.snapshots[len(m.snapshots)-m.maxSnap:]
	}
	return nil
}

func (m *Memory) RecordEvent(e agg.EventRow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
	if len(m.events) > m.maxEvt {
		m.events = m.events[len(m.events)-m.maxEvt:]
	}
	return nil
}

func (m *Memory) QueryHistory(_ context.Context, limit int, comm string) ([]SnapshotHistoryPoint, error) {
	limit = config.ClampLimit(limit, 100, 1000)
	m.mu.Lock()
	defer m.mu.Unlock()
	var filtered []SnapshotHistoryPoint
	for _, p := range m.snapshots {
		if comm != "" && p.Comm != comm {
			continue
		}
		filtered = append(filtered, p)
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	out := make([]SnapshotHistoryPoint, len(filtered))
	copy(out, filtered)
	return out, nil
}

func (m *Memory) QueryEvents(_ context.Context, limit int) ([]agg.EventRow, error) {
	limit = config.ClampLimit(limit, 50, 500)
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(m.events)
	if n > limit {
		n = limit
	}
	src := m.events[len(m.events)-n:]
	out := make([]agg.EventRow, len(src))
	copy(out, src)
	// newest first, matching SQLite ORDER BY ts_unix DESC
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (m *Memory) Close() error { return nil }
