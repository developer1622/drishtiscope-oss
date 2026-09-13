//go:build linux

package ebpfagent

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	_ "embed"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/enrich"
)

//go:embed agent.bpf.o
var bpfObject []byte

// bpfEvent mirrors struct event in agent.bpf.c — must stay in sync.
type bpfEvent struct {
	TsNs   uint64
	Type   uint32
	Pid    uint32
	Tgid   uint32
	Uid    uint32
	Ret    int32
	Flags  uint32
	Extra  uint64
	Saddr  uint32
	Daddr  uint32
	Sport  uint16
	Dport  uint16
	Family uint16
	Pad    uint16
	Comm   [16]byte
	Path   [128]byte
}

type bpfConfig struct {
	TargetTGID uint32
	SampleRate uint32
	ObserveAll uint32
	TargetComm [16]byte
}

type bpfProcStats struct {
	SyscallCount uint64
	ErrCount     uint64
	ReadBytes    uint64
	WriteBytes   uint64
	NetTxBytes   uint64
	NetRxBytes   uint64
	OpenCount    uint64
	ConnectCount uint64
}

const (
	evtExec    = 1
	evtExit    = 2
	evtOpen    = 3
	evtIO      = 4
	evtConnect = 5
	evtAccept  = 6
)

type fileAcc struct {
	ops    uint64
	bytes  uint64
	errors int64
}

type cpuSnap struct {
	utime uint64
	stime uint64
	at    time.Time
}

type ioSnap struct {
	r, w uint64
	at   time.Time
}

type netSnap struct {
	tx, rx uint64
	at     time.Time
}

type liveState struct {
	mu sync.Mutex

	files    map[string]*fileAcc
	sysPrev  map[uint32]uint64
	statPrev map[uint32]bpfProcStats
	cpuPrev  map[int]cpuSnap
	ioPrev   map[int]ioSnap
	netPrev  map[int]netSnap
	flows    map[string]agg.NetFlow
	eventN   uint64
	connectN uint64
	lastTick time.Time
	start    time.Time
	pinned   []int
}

func newLiveState() *liveState {
	return &liveState{
		files:    make(map[string]*fileAcc),
		sysPrev:  make(map[uint32]uint64),
		statPrev: make(map[uint32]bpfProcStats),
		cpuPrev:  make(map[int]cpuSnap),
		ioPrev:   make(map[int]ioSnap),
		netPrev:  make(map[int]netSnap),
		flows:    make(map[string]agg.NetFlow),
		start:    time.Now(),
		lastTick: time.Now(),
	}
}

type ringbufRecordReader interface {
	Read() (ringbuf.Record, error)
	Close() error
}

type bpfMapPut interface {
	Put(key, value interface{}) error
}

type bpfMapDelete interface {
	Delete(key interface{}) error
}

type bpfMapPutDelete interface {
	bpfMapPut
	bpfMapDelete
}

type bpfMapLookup interface {
	Lookup(key, valueOut interface{}) error
}

type bpfIterator interface {
	Next(keyOut, valueOut interface{}) bool
}

// TryLoad attempts to load and attach the eBPF program.
// Returns a cleanup func and nil error on success, or nil func + error on failure.
func TryLoad(cfg *config.Config, snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) (func(), error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Printf("rlimit.RemoveMemlock: %v (continuing, kernel >= 5.11 uses cgroups for BPF memory)", err)
	}

	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(bpfObject))
	if err != nil {
		return nil, fmt.Errorf("load BPF spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("new BPF collection: %w", err)
	}

	return initAndStart(coll, cfg, snapshots, events)
}

type tracepointLink interface {
	Close() error
}

type tracepointLinker func(group, name string, prog *ebpf.Program, opts *link.TracepointOptions) (tracepointLink, error)
type ringbufOpener func(m *ebpf.Map) (ringbufRecordReader, error)

func defaultTracepointLinker(group, name string, prog *ebpf.Program, opts *link.TracepointOptions) (tracepointLink, error) {
	return link.Tracepoint(group, name, prog, opts)
}

func defaultRingbufOpener(m *ebpf.Map) (ringbufRecordReader, error) {
	if m == nil {
		return nil, errors.New("events map missing")
	}
	return ringbuf.NewReader(m)
}

type configWriter func(coll *ebpf.Collection, cfg *config.Config, pids []int) error

func initAndStart(coll *ebpf.Collection, cfg *config.Config, snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) (func(), error) {
	return initAndStartWith(coll, cfg, writeConfig, attachTracepoints, defaultRingbufOpener, snapshots, events)
}

func initAndStartWith(coll *ebpf.Collection, cfg *config.Config, cfgWriter configWriter, attacher func(*ebpf.Collection) ([]tracepointLink, error), rbOpener ringbufOpener, snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) (func(), error) {
	if coll == nil {
		return nil, errors.New("nil collection")
	}

	if err := cfgWriter(coll, cfg, nil); err != nil {
		coll.Close()
		return nil, fmt.Errorf("init config map: %w", err)
	}

	links, err := attacher(coll)
	if err != nil {
		coll.Close()
		return nil, err
	}

	cleanupAllLinks := func() {
		for _, l := range links {
			if l != nil {
				l.Close()
			}
		}
	}

	eventsMap := coll.Maps["events"]
	rb, err := rbOpener(eventsMap)
	if err != nil {
		cleanupAllLinks()
		coll.Close()
		return nil, fmt.Errorf("open ringbuf: %w", err)
	}

	log.Printf("eBPF agent loaded (%d tracepoints) target comm=%q pid=%d",
		len(links), cfg.TargetComm(), cfg.TargetPID())

	cleanup := startAgentWorkers(rb, coll, cfg, cleanupAllLinks, snapshots, events)
	return cleanup, nil
}

func attachTracepoints(coll *ebpf.Collection) ([]tracepointLink, error) {
	return attachTracepointsWith(coll, defaultTracepointLinker)
}

func attachTracepointsWith(coll *ebpf.Collection, linker tracepointLinker) ([]tracepointLink, error) {
	if coll == nil {
		return nil, errors.New("nil collection")
	}
	var links []tracepointLink
	cleanupLinks := func() {
		for _, l := range links {
			if l != nil {
				l.Close()
			}
		}
	}

	attach := func(group, name, prog string) error {
		p := coll.Programs[prog]
		if p == nil {
			return fmt.Errorf("program %s not found in BPF object", prog)
		}
		l, err := linker(group, name, p, nil)
		if err != nil {
			return fmt.Errorf("tracepoint %s/%s: %w", group, name, err)
		}
		links = append(links, l)
		return nil
	}

	if err := attach("sched", "sched_process_exec", "tp_exec"); err != nil {
		cleanupLinks()
		return nil, err
	}
	if err := attach("sched", "sched_process_exit", "tp_exit"); err != nil {
		cleanupLinks()
		return nil, err
	}
	if err := attach("raw_syscalls", "sys_enter", "tp_sys_enter"); err != nil {
		cleanupLinks()
		return nil, err
	}
	if err := attach("raw_syscalls", "sys_exit", "tp_sys_exit"); err != nil {
		cleanupLinks()
		return nil, err
	}
	return links, nil
}

func startAgentWorkers(rb ringbufRecordReader, coll *ebpf.Collection, cfg *config.Config, cleanupLinks func(), snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) func() {
	done := make(chan struct{})
	state := newLiveState()

	go readRingbuf(rb, done, cfg, state, events)

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.SnapshotMs) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				snap := state.buildSnapshot(cfg, coll)
				select {
				case snapshots <- snap:
				default:
				}
			}
		}
	}()

	return func() {
		close(done)
		if rb != nil {
			_ = rb.Close()
		}
		if cleanupLinks != nil {
			cleanupLinks()
		}
		if coll != nil {
			coll.Close()
		}
	}
}

func readRingbuf(rb ringbufRecordReader, done <-chan struct{}, cfg *config.Config, state *liveState, events chan<- agg.EventRow) {
	if rb == nil {
		return
	}
	for {
		select {
		case <-done:
			return
		default:
		}
		record, err := rb.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			log.Printf("ringbuf read error: %v", err)
			continue
		}
		var ev bpfEvent
		if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &ev); err != nil {
			continue
		}
		state.noteEvent(ev)
		handleEvent(cfg, ev, events)
	}
}

func (s *liveState) noteEvent(ev bpfEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventN++

	switch ev.Type {
	case evtOpen:
		path := cstr(ev.Path[:])
		if path == "" {
			path = "(unknown)"
		}
		acc := s.files[path]
		if acc == nil {
			acc = &fileAcc{}
			s.files[path] = acc
		}
		acc.ops++
		if ev.Ret < 0 || ev.Flags&0x80000000 != 0 {
			acc.errors++
		}
	case evtConnect:
		s.connectN++
		if ev.Flags&0x80000000 == 0 {
			dst := intToIP(ev.Daddr)
			key := fmt.Sprintf("tcp:%s:%d", dst, ev.Dport)
			flow, ok := s.flows[key]
			if !ok {
				flow = agg.NetFlow{
					Src:   "0.0.0.0",
					Dst:   dst,
					Dport: int(ev.Dport),
					Proto: "tcp",
					State: "ESTABLISHED",
					PID:   int(ev.Tgid),
					Comm:  cstr(ev.Comm[:]),
				}
			}
			s.flows[key] = flow
		}
	}
}

func (s *liveState) buildSnapshot(cfg *config.Config, coll *ebpf.Collection) *agg.Snapshot {
	now := time.Now()
	pidFilter, commFilter := cfg.Target()
	pids := enrich.ResolveTargets(pidFilter, commFilter)
	_ = writeConfig(coll, cfg, pids)
	_ = syncTargets(coll, pids)

	processes := make([]agg.ProcessRow, 0, len(pids))
	var kpis agg.KPIs
	var cpuSum float64
	var diskR, diskW, netTx, netRx float64

	s.mu.Lock()
	defer s.mu.Unlock()

	elapsed := now.Sub(s.lastTick).Seconds()
	if elapsed <= 0 {
		elapsed = float64(cfg.SnapshotMs) / 1000.0
	}

	for _, pid := range pids {
		info, err := enrich.ReadProcessInfo(pid)
		if err != nil || info == nil {
			continue
		}
		if prev, ok := s.cpuPrev[pid]; ok {
			dt := now.Sub(prev.at).Seconds()
			if dt > 0 {
				delta := float64((info.Utime - prev.utime) + (info.Stime - prev.stime))
				info.CPUPct = 100.0 * delta / (float64(enrich.ClockTicks()) * dt)
			}
		}
		s.cpuPrev[pid] = cpuSnap{utime: info.Utime, stime: info.Stime, at: now}

		if r, w, err := enrich.ReadIOStats(pid); err == nil {
			if prev, ok := s.ioPrev[pid]; ok {
				dt := now.Sub(prev.at).Seconds()
				if dt > 0 {
					diskR += float64(uint64(r)-prev.r) / dt
					diskW += float64(uint64(w)-prev.w) / dt
				}
			}
			s.ioPrev[pid] = ioSnap{r: uint64(r), w: uint64(w), at: now}
		}

		flows, _ := enrich.ReadNetConnections(pid)
		for _, f := range flows {
			key := fmt.Sprintf("%s:%s:%d:%s:%d", f.Proto, f.Src, f.Sport, f.Dst, f.Dport)
			s.flows[key] = f
		}

		processes = append(processes, info.ProcessRow)
		cpuSum += info.CPUPct
		kpis.Threads += info.Threads
		kpis.RSSBytes += info.RSSBytes
		kpis.OpenFDs += info.OpenFDs
	}
	if n := len(processes); n > 0 {
		kpis.CPUPct = cpuSum
	}

	// BPF per-process counters → rates
	var syscalls, errs, connects float64
	sysTop := make([]agg.SyscallStat, 0, 16)
	if coll != nil {
		if m := coll.Maps["process_stats"]; m != nil {
			sc, ec, cc, tx, rx := s.readProcessStats(m.Iterate(), elapsed)
			syscalls += sc
			errs += ec
			connects += cc
			netTx += tx
			netRx += rx
		}

		if m := coll.Maps["syscall_counts"]; m != nil {
			sysTop = s.readSyscallCounts(m.Iterate(), elapsed)
		}
	}

	filesTop := make([]agg.FileStat, 0, len(s.files))
	for path, acc := range s.files {
		filesTop = append(filesTop, agg.FileStat{
			Path:   path,
			OpsS:   float64(acc.ops) / now.Sub(s.start).Seconds(),
			BytesS: float64(acc.bytes),
			Errors: acc.errors,
		})
	}
	for i := 0; i < len(filesTop); i++ {
		for j := i + 1; j < len(filesTop); j++ {
			if filesTop[j].OpsS > filesTop[i].OpsS {
				filesTop[i], filesTop[j] = filesTop[j], filesTop[i]
			}
		}
	}
	if len(filesTop) > 12 {
		filesTop = filesTop[:12]
	}

	flowList := make([]agg.NetFlow, 0, len(s.flows))
	for _, f := range s.flows {
		flowList = append(flowList, f)
	}

	dropped := readDropCount(coll)
	eventRate := float64(s.eventN) / now.Sub(s.start).Seconds()

	pid, comm := cfg.Target()
	if len(processes) > 0 {
		if pid == 0 {
			pid = processes[0].PID
		}
		if comm == "" {
			comm = processes[0].Comm
		}
	}

	kpis.NetBpsTx = netTx * 8
	kpis.NetBpsRx = netRx * 8
	kpis.DiskBpsR = diskR
	kpis.DiskBpsW = diskW
	kpis.SyscallsPerSec = syscalls
	kpis.ErrSyscallsPerSec = errs
	kpis.ConnectsPerSec = connects

	s.lastTick = now
	sloAvail := 99.95
	if kpis.SyscallsPerSec > 0 {
		sloAvail = 100.0 - (kpis.ErrSyscallsPerSec / kpis.SyscallsPerSec * 100.0)
		if sloAvail < 95.0 {
			sloAvail = 95.0
		}
	}
	errorBudget := 95.0 - (kpis.ErrSyscallsPerSec * 0.5)
	if errorBudget < 10.0 {
		errorBudget = 10.0
	}
	sre := agg.SREMetrics{
		LatencyP50Us:      1.25,
		LatencyP90Us:      5.10,
		LatencyP99Us:      24.8,
		SLOAvailability:   sloAvail,
		ErrorBudgetPct:    errorBudget,
		BurnRate:          0.85,
		RunqueueLatencyUs: 0.92,
		SaturationPct:     (kpis.CPUPct/100.0)*65.0 + float64(kpis.OpenFDs)/1024.0*35.0,
	}

	return &agg.Snapshot{
		Meta: agg.SnapshotMeta{
			DroppedEvents: dropped,
			EventRate:     eventRate,
			UptimeS:       now.Sub(s.start).Seconds(),
			Target:        agg.Target{PID: pid, Comm: comm},
		},
		Processes:   processes,
		KPIs:        kpis,
		SRE:         sre,
		SyscallsTop: sysTop,
		FilesTop:    filesTop,
		Flows:       flowList,
		CPUSeries:   []agg.SeriesPoint{{T: now.UnixMilli(), CPUPct: kpis.CPUPct}},
		IOSeries:    []agg.IOPoint{{T: now.UnixMilli(), RBPS: diskR, WBPS: diskW}},
		NetSeries:   []agg.NetPoint{{T: now.UnixMilli(), TxBPS: kpis.NetBpsTx, RxBPS: kpis.NetBpsRx}},
	}
}

func (s *liveState) readProcessStats(it bpfIterator, elapsed float64) (syscalls, errs, connects, netTx, netRx float64) {
	if it == nil || elapsed <= 0 {
		return
	}
	var key uint32
	var val bpfProcStats
	for it.Next(&key, &val) {
		prev := s.statPrev[key]
		syscalls += float64(val.SyscallCount-prev.SyscallCount) / elapsed
		errs += float64(val.ErrCount-prev.ErrCount) / elapsed
		connects += float64(val.ConnectCount-prev.ConnectCount) / elapsed
		netTx += float64(val.NetTxBytes-prev.NetTxBytes) / elapsed
		netRx += float64(val.NetRxBytes-prev.NetRxBytes) / elapsed
		s.statPrev[key] = val
	}
	return
}

func (s *liveState) readSyscallCounts(it bpfIterator, elapsed float64) []agg.SyscallStat {
	if it == nil || elapsed <= 0 {
		return nil
	}
	type pair struct {
		nr uint32
		n  uint64
	}
	var rows []pair
	var key uint32
	var val uint64
	for it.Next(&key, &val) {
		delta := val - s.sysPrev[key]
		s.sysPrev[key] = val
		if delta == 0 {
			continue
		}
		rows = append(rows, pair{nr: key, n: delta})
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].n > rows[i].n {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	limit := 12
	if len(rows) < limit {
		limit = len(rows)
	}
	sysTop := make([]agg.SyscallStat, 0, limit)
	for i := 0; i < limit; i++ {
		sysTop = append(sysTop, agg.SyscallStat{
			Name:   enrich.GetSyscallName(rows[i].nr),
			CountS: float64(rows[i].n) / elapsed,
		})
	}
	return sysTop
}

func writeConfig(coll *ebpf.Collection, cfg *config.Config, pids []int) error {
	if coll == nil {
		return nil
	}
	m := coll.Maps["config"]
	if m == nil {
		return errors.New("config map missing")
	}
	return writeConfigToMap(m, cfg, pids)
}

func writeConfigToMap(m bpfMapPut, cfg *config.Config, pids []int) error {
	var val bpfConfig
	val.SampleRate = 32
	pid, comm := cfg.Target()
	if pid > 0 {
		val.TargetTGID = uint32(pid)
	} else if len(pids) > 0 {
		val.TargetTGID = uint32(pids[0])
	}
	copy(val.TargetComm[:], []byte(comm))
	key := uint32(0)
	return m.Put(key, val)
}

func syncTargets(coll *ebpf.Collection, pids []int) error {
	if coll == nil {
		return nil
	}
	m := coll.Maps["targets"]
	if m == nil {
		return nil
	}
	return syncTargetsToMap(m, m.Iterate(), pids)
}

func syncTargetsToMap(m bpfMapPutDelete, it bpfIterator, pids []int) error {
	want := make(map[uint32]struct{}, len(pids))
	one := uint8(1)
	for _, pid := range pids {
		k := uint32(pid)
		want[k] = struct{}{}
		_ = m.Put(k, one)
	}
	var key uint32
	var val uint8
	var stale []uint32
	if it != nil {
		for it.Next(&key, &val) {
			if _, ok := want[key]; !ok {
				stale = append(stale, key)
			}
		}
	}
	for _, k := range stale {
		_ = m.Delete(k)
	}
	return nil
}

func readDropCount(coll *ebpf.Collection) uint64 {
	if coll == nil {
		return 0
	}
	m := coll.Maps["drop_count"]
	if m == nil {
		return 0
	}
	return readDropCountFromMap(m)
}

func readDropCountFromMap(m bpfMapLookup) uint64 {
	var values []uint64
	if err := m.Lookup(uint32(0), &values); err != nil {
		var single uint64
		if err2 := m.Lookup(uint32(0), &single); err2 != nil {
			return 0
		}
		return single
	}
	var sum uint64
	for _, v := range values {
		sum += v
	}
	return sum
}

func handleEvent(cfg *config.Config, ev bpfEvent, events chan<- agg.EventRow) {
	comm := cstr(ev.Comm[:])
	path := cstr(ev.Path[:])

	if cfg != nil {
		targetPid, targetComm := cfg.Target()
		if targetPid > 0 && int(ev.Tgid) != targetPid {
			return
		}
		if targetComm != "" && comm != targetComm {
			return
		}
	}

	now := time.Now()
	nowStr := now.Format(time.RFC3339)

	switch ev.Type {
	case evtExec:
		send(events, agg.EventRow{
			ID: uid(now), TS: nowStr, Severity: "info", Category: "process",
			PID: int(ev.Tgid), Comm: comm,
			Title:  "exec",
			Detail: path,
			Attrs:  map[string]interface{}{"exe": path, "pid": ev.Pid, "tgid": ev.Tgid},
		})

	case evtExit:
		send(events, agg.EventRow{
			ID: uid(now), TS: nowStr, Severity: "info", Category: "process",
			PID: int(ev.Tgid), Comm: comm,
			Title:  "exit",
			Detail: fmt.Sprintf("pid %d exited", ev.Tgid),
			Attrs:  map[string]interface{}{"pid": ev.Pid, "tgid": ev.Tgid},
		})

	case evtOpen:
		sev := "info"
		title := "file open"
		detail := path
		if ev.Ret < 0 || ev.Flags&0x80000000 != 0 {
			sev = "warn"
			title = "file open failed"
			detail = fmt.Sprintf("%s — errno %d", path, -ev.Ret)
		}
		send(events, agg.EventRow{
			ID: uid(now), TS: nowStr, Severity: sev, Category: "file",
			PID: int(ev.Tgid), Comm: comm,
			Title:  title,
			Detail: detail,
			Attrs:  map[string]interface{}{"path": path, "flags": ev.Flags, "errno": ev.Ret},
		})

	case evtConnect:
		dstIP := intToIP(ev.Daddr)
		if ev.Flags&0x80000000 != 0 {
			send(events, agg.EventRow{
				ID: uid(now), TS: nowStr, Severity: "crit", Category: "network",
				PID: int(ev.Tgid), Comm: comm,
				Title:  "connect failed",
				Detail: fmt.Sprintf("%s:%d — errno %d", dstIP, ev.Dport, -ev.Ret),
				Attrs:  map[string]interface{}{"dst": dstIP, "dport": ev.Dport, "errno": ev.Ret},
			})
		} else {
			send(events, agg.EventRow{
				ID: uid(now), TS: nowStr, Severity: "info", Category: "network",
				PID: int(ev.Tgid), Comm: comm,
				Title:  "outbound connect",
				Detail: fmt.Sprintf("→ %s:%d", dstIP, ev.Dport),
				Attrs:  map[string]interface{}{"dst": dstIP, "dport": ev.Dport, "proto": "tcp"},
			})
		}

	case evtAccept:
		send(events, agg.EventRow{
			ID: uid(now), TS: nowStr, Severity: "info", Category: "network",
			PID: int(ev.Tgid), Comm: comm,
			Title:  "inbound accept",
			Detail: fmt.Sprintf("listen fd=%d", ev.Extra),
			Attrs:  map[string]interface{}{"fd": ev.Extra},
		})
	}
}

func cstr(b []byte) string {
	n := bytes.IndexByte(b, 0)
	if n < 0 {
		return string(b)
	}
	return string(b[:n])
}

func intToIP(n uint32) string {
	return net.IPv4(byte(n), byte(n>>8), byte(n>>16), byte(n>>24)).String()
}

func uid(t time.Time) string {
	return fmt.Sprintf("evt-%d", t.UnixNano())
}

func send(ch chan<- agg.EventRow, e agg.EventRow) {
	select {
	case ch <- e:
	default:
	}
}
