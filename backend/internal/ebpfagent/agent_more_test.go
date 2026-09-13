//go:build linux

package ebpfagent

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

type mockRingbufReader struct {
	records []ringbuf.Record
	idx     int
	closed  bool
}

func (m *mockRingbufReader) Read() (ringbuf.Record, error) {
	if m.closed {
		return ringbuf.Record{}, ringbuf.ErrClosed
	}
	if m.idx >= len(m.records) {
		m.closed = true
		return ringbuf.Record{}, ringbuf.ErrClosed
	}
	r := m.records[m.idx]
	m.idx++
	return r, nil
}

func (m *mockRingbufReader) Close() error {
	m.closed = true
	return nil
}

type mockMap struct {
	data map[interface{}]interface{}
}

func newMockMap() *mockMap {
	return &mockMap{data: make(map[interface{}]interface{})}
}

func (m *mockMap) Put(key, value interface{}) error {
	m.data[key] = value
	return nil
}

func (m *mockMap) Delete(key interface{}) error {
	delete(m.data, key)
	return nil
}

func (m *mockMap) Lookup(key, valueOut interface{}) error {
	v, ok := m.data[key]
	if !ok {
		return errors.New("not found")
	}
	switch out := valueOut.(type) {
	case *[]uint64:
		if s, ok := v.([]uint64); ok {
			*out = s
			return nil
		}
	case *uint64:
		if u, ok := v.(uint64); ok {
			*out = u
			return nil
		}
	}
	return errors.New("type mismatch")
}

type mockIterator struct {
	keys   []interface{}
	values []interface{}
	idx    int
}

func (it *mockIterator) Next(keyOut, valueOut interface{}) bool {
	if it.idx >= len(it.keys) {
		return false
	}
	k := it.keys[it.idx]
	v := it.values[it.idx]
	it.idx++

	if kp, ok := keyOut.(*uint32); ok {
		*kp = k.(uint32)
	}
	if vp, ok := valueOut.(*uint8); ok {
		*vp = v.(uint8)
	}
	if vp, ok := valueOut.(*uint64); ok {
		*vp = v.(uint64)
	}
	if vp, ok := valueOut.(*bpfProcStats); ok {
		*vp = v.(bpfProcStats)
	}
	return true
}

func encodeEvent(ev bpfEvent) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, &ev)
	return buf.Bytes()
}

func TestWriteConfigToMap(t *testing.T) {
	cfg := config.Load()
	_ = cfg.SetTarget(1234, "test-comm")
	m := newMockMap()

	if err := writeConfigToMap(m, cfg, nil); err != nil {
		t.Fatal(err)
	}

	val, ok := m.data[uint32(0)]
	if !ok {
		t.Fatal("expected key 0 in mock map")
	}
	bpfCfg := val.(bpfConfig)
	if bpfCfg.TargetTGID != 1234 {
		t.Fatalf("expected target TGID 1234, got %d", bpfCfg.TargetTGID)
	}
	if cstr(bpfCfg.TargetComm[:]) != "test-comm" {
		t.Fatalf("expected comm test-comm, got %s", cstr(bpfCfg.TargetComm[:]))
	}

	// Test with TargetTGID from pids slice when pid is 0
	_ = cfg.SetTarget(0, "other")
	if err := writeConfigToMap(m, cfg, []int{5678}); err != nil {
		t.Fatal(err)
	}
	bpfCfg2 := m.data[uint32(0)].(bpfConfig)
	if bpfCfg2.TargetTGID != 5678 {
		t.Fatalf("expected target TGID 5678, got %d", bpfCfg2.TargetTGID)
	}

	// Test writeConfig with nil and missing map
	if err := writeConfig(nil, cfg, nil); err != nil {
		t.Fatal(err)
	}
	emptyColl := &ebpf.Collection{Maps: map[string]*ebpf.Map{}}
	if err := writeConfig(emptyColl, cfg, nil); err == nil {
		t.Fatal("expected error for missing config map")
	}
}

func TestSyncTargetsToMap(t *testing.T) {
	m := newMockMap()
	m.data[uint32(100)] = uint8(1)
	m.data[uint32(200)] = uint8(1)

	it := &mockIterator{
		keys:   []interface{}{uint32(100), uint32(200)},
		values: []interface{}{uint8(1), uint8(1)},
	}

	// Want 100 and 300; 200 is stale and should be deleted
	if err := syncTargetsToMap(m, it, []int{100, 300}); err != nil {
		t.Fatal(err)
	}

	if _, ok := m.data[uint32(100)]; !ok {
		t.Fatal("expected 100 retained")
	}
	if _, ok := m.data[uint32(300)]; !ok {
		t.Fatal("expected 300 added")
	}
	if _, ok := m.data[uint32(200)]; ok {
		t.Fatal("expected 200 deleted")
	}

	// Test syncTargets with nil and empty coll
	if err := syncTargets(nil, []int{1}); err != nil {
		t.Fatal(err)
	}
	emptyColl := &ebpf.Collection{Maps: map[string]*ebpf.Map{}}
	if err := syncTargets(emptyColl, []int{1}); err != nil {
		t.Fatal(err)
	}
}

func TestReadDropCountFromMap(t *testing.T) {
	m := newMockMap()
	m.data[uint32(0)] = []uint64{10, 20, 30}

	count := readDropCountFromMap(m)
	if count != 60 {
		t.Fatalf("expected sum 60, got %d", count)
	}

	// Single value fallback
	m.data[uint32(0)] = uint64(42)
	count2 := readDropCountFromMap(m)
	if count2 != 42 {
		t.Fatalf("expected 42, got %d", count2)
	}

	// Missing key
	delete(m.data, uint32(0))
	if readDropCountFromMap(m) != 0 {
		t.Fatal("expected 0 for missing key")
	}

	// Test readDropCount with nil and empty coll
	if readDropCount(nil) != 0 {
		t.Fatal("expected 0 for nil coll")
	}
	emptyColl := &ebpf.Collection{Maps: map[string]*ebpf.Map{}}
	if readDropCount(emptyColl) != 0 {
		t.Fatal("expected 0 for missing drop_count map")
	}
}

func TestReadRingbufAndHandleEvents(t *testing.T) {
	cfg := config.Load()
	_ = cfg.SetTarget(1234, "test-comm")
	state := newLiveState()
	events := make(chan agg.EventRow, 32)
	done := make(chan struct{})

	// 1. exec event
	var evExec bpfEvent
	evExec.Type = evtExec
	evExec.Tgid = 1234
	copy(evExec.Comm[:], "test-comm\x00")
	copy(evExec.Path[:], "/bin/test\x00")

	// 2. exit event
	var evExit bpfEvent
	evExit.Type = evtExit
	evExit.Tgid = 1234
	copy(evExit.Comm[:], "test-comm\x00")

	// 3. open failed event
	var evOpenFail bpfEvent
	evOpenFail.Type = evtOpen
	evOpenFail.Tgid = 1234
	copy(evOpenFail.Comm[:], "test-comm\x00")
	copy(evOpenFail.Path[:], "/etc/shadow\x00")
	evOpenFail.Ret = -13
	evOpenFail.Flags = 0x80000000

	// 4. connect failed event
	var evConnFail bpfEvent
	evConnFail.Type = evtConnect
	evConnFail.Tgid = 1234
	copy(evConnFail.Comm[:], "test-comm\x00")
	evConnFail.Daddr = 0x0100000a
	evConnFail.Dport = 443
	evConnFail.Ret = -111
	evConnFail.Flags = 0x80000000

	// 5. accept event
	var evAccept bpfEvent
	evAccept.Type = evtAccept
	evAccept.Tgid = 1234
	copy(evAccept.Comm[:], "test-comm\x00")
	evAccept.Extra = 7

	mockRb := &mockRingbufReader{
		records: []ringbuf.Record{
			{RawSample: encodeEvent(evExec)},
			{RawSample: encodeEvent(evExit)},
			{RawSample: encodeEvent(evOpenFail)},
			{RawSample: encodeEvent(evConnFail)},
			{RawSample: encodeEvent(evAccept)},
			{RawSample: []byte("too short")}, // corrupt sample
		},
	}

	readRingbuf(mockRb, done, cfg, state, events)

	if len(events) != 5 {
		t.Fatalf("expected 5 events dispatched, got %d", len(events))
	}

	// Test nil reader does not hang or panic
	readRingbuf(nil, done, cfg, state, events)
}

func TestBuildSnapshotComprehensive(t *testing.T) {
	selfPID := os.Getpid()
	cfg := config.Load()
	_ = cfg.SetTarget(selfPID, "")
	cfg.SnapshotMs = 100

	s := newLiveState()
	s.start = time.Now().Add(-10 * time.Second)
	s.lastTick = time.Now().Add(-1 * time.Second)

	// Populate cpuPrev and ioPrev to trigger delta calculation branches
	s.cpuPrev[selfPID] = cpuSnap{utime: 10, stime: 20, at: time.Now().Add(-500 * time.Millisecond)}
	s.ioPrev[selfPID] = ioSnap{r: 1000, w: 500, at: time.Now().Add(-500 * time.Millisecond)}

	// Note 15 different files to exercise sorting and truncation limit of 12
	for i := 0; i < 15; i++ {
		var ev bpfEvent
		ev.Type = evtOpen
		copy(ev.Path[:], []byte("/tmp/test_file_"+string(rune('a'+i))+"\x00"))
		s.noteEvent(ev)
	}

	// Note a successful connect
	var evConn bpfEvent
	evConn.Type = evtConnect
	evConn.Tgid = uint32(selfPID)
	copy(evConn.Comm[:], "test\x00")
	evConn.Daddr = 0x0100007f
	evConn.Dport = 8080
	s.noteEvent(evConn)

	snap := s.buildSnapshot(cfg, nil)
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}

	if len(snap.Processes) == 0 {
		t.Fatal("expected at least 1 process for self PID")
	}
	if len(snap.FilesTop) > 12 {
		t.Fatalf("expected FilesTop truncated to <= 12, got %d", len(snap.FilesTop))
	}
	if snap.SRE.SLOAvailability <= 0 {
		t.Errorf("expected positive SLO availability, got %f", snap.SRE.SLOAvailability)
	}
	if snap.Meta.Target.PID != selfPID {
		t.Fatalf("expected target PID %d, got %d", selfPID, snap.Meta.Target.PID)
	}
}

func TestStartAgentWorkers(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 20
	_ = cfg.SetTarget(os.Getpid(), "")

	snaps := make(chan *agg.Snapshot, 10)
	events := make(chan agg.EventRow, 10)

	var evExec bpfEvent
	evExec.Type = evtExec
	evExec.Tgid = uint32(os.Getpid())
	copy(evExec.Comm[:], "self\x00")

	mockRb := &mockRingbufReader{
		records: []ringbuf.Record{
			{RawSample: encodeEvent(evExec)},
		},
	}

	cleanedLinks := false
	cleanupLinks := func() {
		cleanedLinks = true
	}

	cleanup := startAgentWorkers(mockRb, nil, cfg, cleanupLinks, snaps, events)

	// Wait for at least one snapshot and event
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	select {
	case <-snaps:
	case <-ctx.Done():
		t.Fatal("timeout waiting for snapshot from worker")
	}

	cleanup()

	if !cleanedLinks {
		t.Fatal("expected cleanupLinks to be called")
	}
	if !mockRb.closed {
		t.Fatal("expected mock ringbuf to be closed")
	}
}

func TestTryLoadNonRoot(t *testing.T) {
	cfg := config.Load()
	snaps := make(chan *agg.Snapshot, 1)
	events := make(chan agg.EventRow, 1)

	cleanup, err := TryLoad(cfg, snaps, events)
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
	} else {
		// Expected when running unprivileged without CAP_BPF
		if cleanup != nil {
			t.Fatal("expected nil cleanup on error")
		}
	}
}

func TestInitAndStartErrors(t *testing.T) {
	cfg := config.Load()
	snaps := make(chan *agg.Snapshot, 1)
	events := make(chan agg.EventRow, 1)

	// nil collection
	if _, err := initAndStart(nil, cfg, snaps, events); err == nil {
		t.Fatal("expected error for nil collection")
	}

	// empty collection (missing config map)
	emptyColl := &ebpf.Collection{Maps: map[string]*ebpf.Map{}}
	if _, err := initAndStart(emptyColl, cfg, snaps, events); err == nil {
		t.Fatal("expected error for empty collection")
	}
}

func TestAttachTracepointsErrors(t *testing.T) {
	emptyColl := &ebpf.Collection{
		Programs: map[string]*ebpf.Program{},
	}
	if _, err := attachTracepoints(emptyColl); err == nil {
		t.Fatal("expected error for missing tracepoint programs")
	}
}

func TestReadProcessStats(t *testing.T) {
	s := newLiveState()

	// Initial reading
	it := &mockIterator{
		keys: []interface{}{uint32(1001)},
		values: []interface{}{
			bpfProcStats{
				SyscallCount: 100,
				ErrCount:     5,
				ConnectCount: 2,
				NetTxBytes:   1000,
				NetRxBytes:   2000,
			},
		},
	}

	sc, ec, cc, tx, rx := s.readProcessStats(it, 1.0)
	if sc != 100 || ec != 5 || cc != 2 || tx != 1000 || rx != 2000 {
		t.Fatalf("unexpected initial rates: sc=%f ec=%f cc=%f tx=%f rx=%f", sc, ec, cc, tx, rx)
	}

	// Second reading with delta
	it2 := &mockIterator{
		keys: []interface{}{uint32(1001)},
		values: []interface{}{
			bpfProcStats{
				SyscallCount: 150,
				ErrCount:     7,
				ConnectCount: 3,
				NetTxBytes:   1500,
				NetRxBytes:   2500,
			},
		},
	}

	sc2, ec2, cc2, tx2, rx2 := s.readProcessStats(it2, 1.0)
	if sc2 != 50 || ec2 != 2 || cc2 != 1 || tx2 != 500 || rx2 != 500 {
		t.Fatalf("unexpected delta rates: sc=%f ec=%f cc=%f tx=%f rx=%f", sc2, ec2, cc2, tx2, rx2)
	}

	// Nil iterator or non-positive elapsed
	s.readProcessStats(nil, 1.0)
	s.readProcessStats(it, 0)
}

func TestReadSyscallCounts(t *testing.T) {
	s := newLiveState()

	it := &mockIterator{
		keys:   []interface{}{uint32(0), uint32(1), uint32(257)},
		values: []interface{}{uint64(50), uint64(30), uint64(10)},
	}

	top := s.readSyscallCounts(it, 1.0)
	if len(top) != 3 {
		t.Fatalf("expected 3 syscall stats, got %d", len(top))
	}
	if top[0].Name != "read" || top[0].CountS != 50 {
		t.Fatalf("expected read as top, got %+v", top[0])
	}
	if top[1].Name != "write" || top[1].CountS != 30 {
		t.Fatalf("expected write as second, got %+v", top[1])
	}
	if top[2].Name != "openat" || top[2].CountS != 10 {
		t.Fatalf("expected openat as third, got %+v", top[2])
	}

	// Nil iterator or non-positive elapsed
	if s.readSyscallCounts(nil, 1.0) != nil {
		t.Fatal("expected nil for nil iterator")
	}
	if s.readSyscallCounts(it, 0) != nil {
		t.Fatal("expected nil for non-positive elapsed")
	}
}

func TestSendNonBlocking(t *testing.T) {
	ch := make(chan agg.EventRow, 1)
	send(ch, agg.EventRow{ID: "1"})
	// Next send should drop without blocking
	send(ch, agg.EventRow{ID: "2"})
	if len(ch) != 1 {
		t.Fatalf("expected 1 item, got %d", len(ch))
	}
}

type mockLink struct {
	closed bool
}

func (m *mockLink) Close() error {
	m.closed = true
	return nil
}

func TestInitAndStartWithPluggable(t *testing.T) {
	cfg := config.Load()
	cfg.SnapshotMs = 20
	snaps := make(chan *agg.Snapshot, 10)
	events := make(chan agg.EventRow, 10)

	// Attacher error
	errAttacher := func(*ebpf.Collection) ([]tracepointLink, error) {
		return nil, errors.New("attach failed")
	}
	cfgWriterOK := func(coll *ebpf.Collection, cfg *config.Config, pids []int) error {
		return nil
	}
	cfgWriterErr := func(coll *ebpf.Collection, cfg *config.Config, pids []int) error {
		return errors.New("config write error")
	}

	// 1. nil coll
	if _, err := initAndStartWith(nil, cfg, cfgWriterOK, errAttacher, nil, snaps, events); err == nil {
		t.Fatal("expected error for nil coll")
	}

	coll := &ebpf.Collection{
		Maps: map[string]*ebpf.Map{},
	}

	// 2. Config writer error
	if _, err := initAndStartWith(coll, cfg, cfgWriterErr, errAttacher, nil, snaps, events); err == nil {
		t.Fatal("expected error on config write failure")
	}

	// 3. Attacher error
	if _, err := initAndStartWith(coll, cfg, cfgWriterOK, errAttacher, nil, snaps, events); err == nil {
		t.Fatal("expected error on attacher failure")
	}

	// 4. Ringbuf opener error
	mockAttacher := func(*ebpf.Collection) ([]tracepointLink, error) {
		return []tracepointLink{&mockLink{}}, nil
	}
	errOpener := func(*ebpf.Map) (ringbufRecordReader, error) {
		return nil, errors.New("ringbuf opener failed")
	}
	if _, err := initAndStartWith(coll, cfg, cfgWriterOK, mockAttacher, errOpener, snaps, events); err == nil {
		t.Fatal("expected error on ringbuf opener failure")
	}

	// 5. Successful start and cleanup
	mockOpener := func(*ebpf.Map) (ringbufRecordReader, error) {
		return &mockRingbufReader{}, nil
	}
	cleanup, err := initAndStartWith(coll, cfg, cfgWriterOK, mockAttacher, mockOpener, snaps, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cleanup()
}

func TestAttachTracepointsWithBranches(t *testing.T) {
	// nil collection
	if _, err := attachTracepointsWith(nil, nil); err == nil {
		t.Fatal("expected error for nil collection")
	}

	// Missing tp_exec program
	emptyColl := &ebpf.Collection{Programs: map[string]*ebpf.Program{}}
	if _, err := attachTracepointsWith(emptyColl, nil); err == nil {
		t.Fatal("expected error for missing tp_exec")
	}

	collWithProgs := &ebpf.Collection{
		Programs: map[string]*ebpf.Program{
			"tp_exec":      {},
			"tp_exit":      {},
			"tp_sys_enter": {},
			"tp_sys_exit":  {},
		},
	}

	// Linker success for all 4
	mockLinker := func(group, name string, prog *ebpf.Program, opts *link.TracepointOptions) (tracepointLink, error) {
		return &mockLink{}, nil
	}
	links, err := attachTracepointsWith(collWithProgs, mockLinker)
	if err != nil || len(links) != 4 {
		t.Fatalf("expected 4 links, got %d, err=%v", len(links), err)
	}

	// Linker error on tp_exit
	failExitLinker := func(group, name string, prog *ebpf.Program, opts *link.TracepointOptions) (tracepointLink, error) {
		if name == "sched_process_exit" {
			return nil, errors.New("tp_exit failed")
		}
		return &mockLink{}, nil
	}
	if _, err := attachTracepointsWith(collWithProgs, failExitLinker); err == nil {
		t.Fatal("expected error for tp_exit failure")
	}

	// Linker error on tp_sys_enter
	failEnterLinker := func(group, name string, prog *ebpf.Program, opts *link.TracepointOptions) (tracepointLink, error) {
		if name == "sys_enter" {
			return nil, errors.New("sys_enter failed")
		}
		return &mockLink{}, nil
	}
	if _, err := attachTracepointsWith(collWithProgs, failEnterLinker); err == nil {
		t.Fatal("expected error for sys_enter failure")
	}

	// Linker error on tp_sys_exit
	failSysExitLinker := func(group, name string, prog *ebpf.Program, opts *link.TracepointOptions) (tracepointLink, error) {
		if name == "sys_exit" {
			return nil, errors.New("sys_exit failed")
		}
		return &mockLink{}, nil
	}
	if _, err := attachTracepointsWith(collWithProgs, failSysExitLinker); err == nil {
		t.Fatal("expected error for sys_exit failure")
	}
}

func TestDefaultRingbufOpenerNil(t *testing.T) {
	if _, err := defaultRingbufOpener(nil); err == nil {
		t.Fatal("expected error for nil map")
	}
}

func TestReadRingbufTransientError(t *testing.T) {
	cfg := config.Load()
	state := newLiveState()
	events := make(chan agg.EventRow, 10)
	done := make(chan struct{})

	var ev bpfEvent
	ev.Type = evtExec
	copy(ev.Comm[:], "test\x00")

	// Create reader that returns a transient error first, then a valid record, then closes
	mockRb := &mockRingbufReaderWithError{
		records: []ringbuf.Record{
			{RawSample: encodeEvent(ev)},
		},
	}

	readRingbuf(mockRb, done, cfg, state, events)
}

type mockRingbufReaderWithError struct {
	records []ringbuf.Record
	idx     int
	closed  bool
	errored bool
}

func (m *mockRingbufReaderWithError) Read() (ringbuf.Record, error) {
	if !m.errored {
		m.errored = true
		return ringbuf.Record{}, errors.New("transient read error")
	}
	if m.closed || m.idx >= len(m.records) {
		return ringbuf.Record{}, ringbuf.ErrClosed
	}
	r := m.records[m.idx]
	m.idx++
	return r, nil
}

func (m *mockRingbufReaderWithError) Close() error {
	m.closed = true
	return nil
}

func TestBuildSnapshotEdgeCases(t *testing.T) {
	cfg := config.Load()
	// Set target comm with no pid and no comm to exercise comm == ""
	_ = cfg.SetTarget(os.Getpid(), "")
	s := newLiveState()
	s.start = time.Now().Add(-5 * time.Second)
	// Set lastTick to now to exercise elapsed <= 0 fallback
	s.lastTick = time.Now()

	snap := s.buildSnapshot(cfg, nil)
	if snap == nil {
		t.Fatal("expected snapshot")
	}
}


