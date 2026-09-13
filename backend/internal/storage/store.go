package storage

import (
	"context"

	"github.com/agentscope/agentscope/internal/agg"
)

// Store is the persistence port. Tests use Memory; production uses SQLite.
type Store interface {
	RecordSnapshot(snap *agg.Snapshot) error
	RecordEvent(e agg.EventRow) error
	QueryHistory(ctx context.Context, limit int, comm string) ([]SnapshotHistoryPoint, error)
	QueryEvents(ctx context.Context, limit int) ([]agg.EventRow, error)
	Close() error
}

// SnapshotHistoryPoint is a lightweight time-series entry for historical graphs.
type SnapshotHistoryPoint struct {
	ID       int64   `json:"id"`
	T        int64   `json:"t"`
	Comm     string  `json:"comm"`
	PID      int     `json:"pid"`
	CPUPct   float64 `json:"cpu_pct"`
	Threads  int     `json:"threads"`
	RSSBytes int64   `json:"rss_bytes"`
	OpenFDs  int     `json:"open_fds"`
	SyscallS float64 `json:"syscall_s"`
	ErrS     float64 `json:"err_s"`
	NetTxBps float64 `json:"net_tx_bps"`
	NetRxBps float64 `json:"net_rx_bps"`
	DiskRBps float64 `json:"disk_r_bps"`
	DiskWBps float64 `json:"disk_w_bps"`
}

// Nop is a Store that discards writes and returns empty reads.
type Nop struct{}

func (Nop) RecordSnapshot(*agg.Snapshot) error { return nil }
func (Nop) RecordEvent(agg.EventRow) error     { return nil }
func (Nop) QueryHistory(context.Context, int, string) ([]SnapshotHistoryPoint, error) {
	return []SnapshotHistoryPoint{}, nil
}
func (Nop) QueryEvents(context.Context, int) ([]agg.EventRow, error) {
	return []agg.EventRow{}, nil
}
func (Nop) Close() error { return nil }
