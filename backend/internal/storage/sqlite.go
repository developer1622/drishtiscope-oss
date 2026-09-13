package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

const (
	maxStoredSnapshots = 1000
	maxStoredEvents    = 2000
	pruneEvery         = 32
)

// DB is a WAL SQLite time-series store.
type DB struct {
	db           *sql.DB
	mu           sync.Mutex // Close + prune
	writes       atomic.Uint64
	lastSnapTime atomic.Int64
	insSnap      *sql.Stmt
	insEvent     *sql.Stmt
}

func Open(dbPath string) (*DB, error) {
	if dbPath == "" {
		dbPath = config.DefaultDBPath
	}
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("mkdir db dir: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := os.Chmod(dbPath, 0o600); err != nil {
		log.Printf("warning: chmod db 0600: %v", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)

	s := &DB{db: db}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	if err := s.prepare(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("prepare: %w", err)
	}
	log.Printf("SQLite TSDB ready at %s (WAL)", dbPath)
	return s, nil
}

func (s *DB) initSchema() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts INTEGER NOT NULL,
		target_comm TEXT NOT NULL,
		target_pid INTEGER NOT NULL,
		cpu_pct REAL NOT NULL,
		threads INTEGER NOT NULL,
		rss_bytes INTEGER NOT NULL,
		open_fds INTEGER NOT NULL,
		syscalls_per_sec REAL NOT NULL,
		err_syscalls_per_sec REAL NOT NULL,
		net_bps_tx REAL NOT NULL,
		net_bps_rx REAL NOT NULL,
		disk_bps_r REAL NOT NULL,
		disk_bps_w REAL NOT NULL,
		raw_json TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts);
	CREATE INDEX IF NOT EXISTS idx_snapshots_comm ON snapshots(target_comm);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		ts TEXT NOT NULL,
		ts_unix INTEGER NOT NULL,
		severity TEXT NOT NULL,
		category TEXT NOT NULL,
		pid INTEGER NOT NULL,
		comm TEXT NOT NULL,
		title TEXT NOT NULL,
		detail TEXT NOT NULL,
		attrs_json TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts_unix);
	CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);
	`)
	return err
}

func (s *DB) prepare() error {
	var err error
	s.insSnap, err = s.db.Prepare(`
		INSERT INTO snapshots (
			ts, target_comm, target_pid, cpu_pct, threads, rss_bytes, open_fds,
			syscalls_per_sec, err_syscalls_per_sec, net_bps_tx, net_bps_rx,
			disk_bps_r, disk_bps_w, raw_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	s.insEvent, err = s.db.Prepare(`
		INSERT OR REPLACE INTO events (
			id, ts, ts_unix, severity, category, pid, comm, title, detail, attrs_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	return err
}

func (s *DB) RecordSnapshot(snap *agg.Snapshot) error {
	if snap == nil {
		return nil
	}
	nowMs := time.Now().UnixMilli()
	// Throttle to 1 write per 1500ms to keep WAL writes minimal while preserving trends
	if prev := s.lastSnapTime.Load(); prev > 0 && (nowMs-prev) < 1500 {
		return nil
	}
	s.lastSnapTime.Store(nowMs)

	// Strip bulky nested series and timeline arrays to prevent unbounded growth in raw_json
	snapCopy := *snap
	snapCopy.Timeline = nil
	snapCopy.CPUSeries = nil
	snapCopy.IOSeries = nil
	snapCopy.NetSeries = nil

	raw, err := json.Marshal(&snapCopy)
	if err != nil {
		return err
	}
	_, err = s.insSnap.Exec(
		nowMs,
		snap.Meta.Target.Comm,
		snap.Meta.Target.PID,
		snap.KPIs.CPUPct,
		snap.KPIs.Threads,
		snap.KPIs.RSSBytes,
		snap.KPIs.OpenFDs,
		snap.KPIs.SyscallsPerSec,
		snap.KPIs.ErrSyscallsPerSec,
		snap.KPIs.NetBpsTx,
		snap.KPIs.NetBpsRx,
		snap.KPIs.DiskBpsR,
		snap.KPIs.DiskBpsW,
		string(raw),
	)
	if err != nil {
		return err
	}
	s.maybePrune()
	return nil
}

func (s *DB) RecordEvent(e agg.EventRow) error {
	attrsRaw, err := json.Marshal(e.Attrs)
	if err != nil {
		attrsRaw = []byte("{}")
	}
	_, err = s.insEvent.Exec(
		e.ID, e.TS, time.Now().UnixMilli(),
		e.Severity, e.Category, e.PID, e.Comm, e.Title, e.Detail, string(attrsRaw),
	)
	if err != nil {
		return err
	}
	s.maybePrune()
	return nil
}

func (s *DB) maybePrune() {
	n := s.writes.Add(1)
	if n%pruneEvery != 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, _ = s.db.Exec(`DELETE FROM snapshots WHERE id IN (
		SELECT id FROM snapshots ORDER BY id DESC LIMIT -1 OFFSET ?
	)`, maxStoredSnapshots)
	_, _ = s.db.Exec(`DELETE FROM events WHERE ts_unix IN (
		SELECT ts_unix FROM events ORDER BY ts_unix DESC LIMIT -1 OFFSET ?
	)`, maxStoredEvents)
	// Execute passive WAL checkpoint to flush committed WAL frames and prevent unbounded WAL file growth
	_, _ = s.db.Exec(`PRAGMA wal_checkpoint(PASSIVE)`)
}

func (s *DB) QueryHistory(ctx context.Context, limit int, comm string) ([]SnapshotHistoryPoint, error) {
	limit = config.ClampLimit(limit, 100, 1000)
	var (
		rows *sql.Rows
		err  error
	)
	if comm != "" {
		comm = config.SanitizeComm(comm)
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, ts, target_comm, target_pid, cpu_pct, threads, rss_bytes, open_fds,
			       syscalls_per_sec, err_syscalls_per_sec, net_bps_tx, net_bps_rx, disk_bps_r, disk_bps_w
			FROM snapshots WHERE target_comm = ? ORDER BY id DESC LIMIT ?`, comm, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, ts, target_comm, target_pid, cpu_pct, threads, rss_bytes, open_fds,
			       syscalls_per_sec, err_syscalls_per_sec, net_bps_tx, net_bps_rx, disk_bps_r, disk_bps_w
			FROM snapshots ORDER BY id DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []SnapshotHistoryPoint
	for rows.Next() {
		var p SnapshotHistoryPoint
		if err := rows.Scan(
			&p.ID, &p.T, &p.Comm, &p.PID, &p.CPUPct, &p.Threads, &p.RSSBytes, &p.OpenFDs,
			&p.SyscallS, &p.ErrS, &p.NetTxBps, &p.NetRxBps, &p.DiskRBps, &p.DiskWBps,
		); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	if points == nil {
		points = []SnapshotHistoryPoint{}
	}
	return points, nil
}

func (s *DB) QueryEvents(ctx context.Context, limit int) ([]agg.EventRow, error) {
	limit = config.ClampLimit(limit, 50, 500)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, ts, severity, category, pid, comm, title, detail, attrs_json
		FROM events ORDER BY ts_unix DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []agg.EventRow
	for rows.Next() {
		var e agg.EventRow
		var attrsRaw string
		if err := rows.Scan(
			&e.ID, &e.TS, &e.Severity, &e.Category, &e.PID, &e.Comm, &e.Title, &e.Detail, &attrsRaw,
		); err != nil {
			return nil, err
		}
		var attrs map[string]interface{}
		if attrsRaw != "" {
			_ = json.Unmarshal([]byte(attrsRaw), &attrs)
		}
		e.Attrs = attrs
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if events == nil {
		events = []agg.EventRow{}
	}
	return events, nil
}

func (s *DB) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.insSnap != nil {
		_ = s.insSnap.Close()
	}
	if s.insEvent != nil {
		_ = s.insEvent.Close()
	}
	return s.db.Close()
}
