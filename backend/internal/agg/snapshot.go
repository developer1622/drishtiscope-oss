package agg

import (
	"sync"
)

type Snapshot struct {
	Meta        SnapshotMeta  `json:"meta"`
	Processes   []ProcessRow  `json:"processes"`
	KPIs        KPIs          `json:"kpis"`
	SRE         SREMetrics    `json:"sre"`
	SyscallsTop []SyscallStat `json:"syscalls_top"`
	FilesTop    []FileStat    `json:"files_top"`
	Flows       []NetFlow     `json:"flows"`
	CPUSeries   []SeriesPoint `json:"cpu_series"`
	IOSeries    []IOPoint     `json:"io_series"`
	NetSeries   []NetPoint    `json:"net_series"`
	Timeline    []EventRow    `json:"timeline"`
}

type SREMetrics struct {
	LatencyP50Us      float64 `json:"latency_p50_us"`
	LatencyP90Us      float64 `json:"latency_p90_us"`
	LatencyP99Us      float64 `json:"latency_p99_us"`
	SLOAvailability   float64 `json:"slo_availability"`
	ErrorBudgetPct    float64 `json:"error_budget_pct"`
	BurnRate          float64 `json:"burn_rate"`
	RunqueueLatencyUs float64 `json:"runqueue_latency_us"`
	SaturationPct     float64 `json:"saturation_pct"`
}

type SnapshotMeta struct {
	DroppedEvents uint64  `json:"dropped_events"`
	EventRate     float64 `json:"event_rate"`
	UptimeS       float64 `json:"uptime_s"`
	Target        Target  `json:"target"`
}

type Target struct {
	PID  int    `json:"pid"`
	Comm string `json:"comm"`
}

type ProcessRow struct {
	PID       int     `json:"pid"`
	TGID      int     `json:"tgid"`
	PPID      int     `json:"ppid"`
	Comm      string  `json:"comm"`
	Cmdline   string  `json:"cmdline"`
	Exe       string  `json:"exe"`
	UID       int     `json:"uid"`
	State     string  `json:"state"`
	Threads   int     `json:"threads"`
	CPUPct    float64 `json:"cpu_pct"`
	RSSBytes  int64   `json:"rss_bytes"`
	VMSBytes  int64   `json:"vms_bytes"`
	OpenFDs   int     `json:"open_fds"`
	CtxSw     int64   `json:"ctx_switches"`
	StartTime string  `json:"start_time"`
}

type KPIs struct {
	CPUPct            float64 `json:"cpu_pct"`
	Threads           int     `json:"threads"`
	RSSBytes          int64   `json:"rss_bytes"`
	OpenFDs           int     `json:"open_fds"`
	NetBpsTx          float64 `json:"net_bps_tx"`
	NetBpsRx          float64 `json:"net_bps_rx"`
	DiskBpsR          float64 `json:"disk_bps_r"`
	DiskBpsW          float64 `json:"disk_bps_w"`
	SyscallsPerSec    float64 `json:"syscalls_per_sec"`
	ErrSyscallsPerSec float64 `json:"err_syscalls_per_sec"`
	ConnectsPerSec    float64 `json:"connects_per_sec"`
}

type SyscallStat struct {
	Name    string  `json:"name"`
	CountS  float64 `json:"count_s"`
	ErrorsS float64 `json:"errors_s"`
}

type FileStat struct {
	Path   string  `json:"path"`
	OpsS   float64 `json:"ops_s"`
	BytesS float64 `json:"bytes_s"`
	Errors int64   `json:"errors"`
}

type NetFlow struct {
	Src     string `json:"src"`
	Dst     string `json:"dst"`
	Sport   int    `json:"sport"`
	Dport   int    `json:"dport"`
	Proto   string `json:"proto"`
	State   string `json:"state"`
	BytesTx int64  `json:"bytes_tx"`
	BytesRx int64  `json:"bytes_rx"`
	PID     int    `json:"pid"`
	Comm    string `json:"comm"`
}

type SeriesPoint struct {
	T      int64   `json:"t"` // unix ms
	CPUPct float64 `json:"cpu_pct"`
}

type IOPoint struct {
	T    int64   `json:"t"`
	RBPS float64 `json:"r_bps"`
	WBPS float64 `json:"w_bps"`
}

type NetPoint struct {
	T     int64   `json:"t"`
	TxBPS float64 `json:"tx_bps"`
	RxBPS float64 `json:"rx_bps"`
}

type EventRow struct {
	ID       string                 `json:"id"`
	TS       string                 `json:"ts"`
	Severity string                 `json:"severity"` // info|warn|crit
	Category string                 `json:"category"` // process|syscall|file|network|compute
	PID      int                    `json:"pid"`
	Comm     string                 `json:"comm"`
	Title    string                 `json:"title"`
	Detail   string                 `json:"detail"`
	Attrs    map[string]interface{} `json:"attrs"`
}

type SnapshotBuilder struct {
	mu        sync.Mutex
	latest    *Snapshot
	cpuSeries []SeriesPoint
	ioSeries  []IOPoint
	netSeries []NetPoint
	events    []EventRow
}

func emptySnapshot() *Snapshot {
	return &Snapshot{
		Processes:   []ProcessRow{},
		SyscallsTop: []SyscallStat{},
		FilesTop:    []FileStat{},
		Flows:       []NetFlow{},
		CPUSeries:   []SeriesPoint{},
		IOSeries:    []IOPoint{},
		NetSeries:   []NetPoint{},
		Timeline:    []EventRow{},
	}
}

func NewSnapshotBuilder() *SnapshotBuilder {
	return &SnapshotBuilder{
		latest:    emptySnapshot(),
		cpuSeries: make([]SeriesPoint, 0, 60),
		ioSeries:  make([]IOPoint, 0, 60),
		netSeries: make([]NetPoint, 0, 60),
		events:    make([]EventRow, 0, 300),
	}
}

func (b *SnapshotBuilder) Update(s *Snapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Producers may send either a single new point or a full rolling window.
	// Always ingest only the newest point so we never double-accumulate.
	if n := len(s.CPUSeries); n > 0 {
		b.cpuSeries = appendCapped(b.cpuSeries, 60, s.CPUSeries[n-1])
	}
	if n := len(s.IOSeries); n > 0 {
		b.ioSeries = appendIOCapped(b.ioSeries, 60, s.IOSeries[n-1])
	}
	if n := len(s.NetSeries); n > 0 {
		b.netSeries = appendNetCapped(b.netSeries, 60, s.NetSeries[n-1])
	}

	b.latest = s
	b.latest.CPUSeries = append([]SeriesPoint(nil), b.cpuSeries...)
	b.latest.IOSeries = append([]IOPoint(nil), b.ioSeries...)
	b.latest.NetSeries = append([]NetPoint(nil), b.netSeries...)
	b.latest.Timeline = append([]EventRow(nil), b.events...)
	if b.latest.Processes == nil {
		b.latest.Processes = []ProcessRow{}
	}
	if b.latest.SyscallsTop == nil {
		b.latest.SyscallsTop = []SyscallStat{}
	}
	if b.latest.FilesTop == nil {
		b.latest.FilesTop = []FileStat{}
	}
	if b.latest.Flows == nil {
		b.latest.Flows = []NetFlow{}
	}
}

func (b *SnapshotBuilder) Latest() *Snapshot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return cloneSnapshot(b.latest)
}

func cloneSnapshot(s *Snapshot) *Snapshot {
	if s == nil {
		return emptySnapshot()
	}
	out := *s
	out.Processes = append([]ProcessRow{}, s.Processes...)
	out.SyscallsTop = append([]SyscallStat{}, s.SyscallsTop...)
	out.FilesTop = append([]FileStat{}, s.FilesTop...)
	out.Flows = append([]NetFlow{}, s.Flows...)
	out.CPUSeries = append([]SeriesPoint{}, s.CPUSeries...)
	out.IOSeries = append([]IOPoint{}, s.IOSeries...)
	out.NetSeries = append([]NetPoint{}, s.NetSeries...)
	out.Timeline = append([]EventRow{}, s.Timeline...)
	return &out
}

func appendCapped(s []SeriesPoint, cap int, p SeriesPoint) []SeriesPoint {
	s = append(s, p)
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}

func appendIOCapped(s []IOPoint, cap int, p IOPoint) []IOPoint {
	s = append(s, p)
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}

func appendNetCapped(s []NetPoint, cap int, p NetPoint) []NetPoint {
	s = append(s, p)
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}

func (b *SnapshotBuilder) AddEvent(e EventRow) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events = append(b.events, e)
	if len(b.events) > 300 {
		b.events = b.events[len(b.events)-300:]
	}
}
