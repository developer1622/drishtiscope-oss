package mock

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

// MockGenerator simulates realistic process trees and telemetry for
// target processes (such as "agy" or "payments-agent").
type MockGenerator struct {
	cfg *config.Config

	// rolling series (max 60 points each)
	cpuSeries []agg.SeriesPoint
	ioSeries  []agg.IOPoint
	netSeries []agg.NetPoint

	// flows table — maintained as a slice, entries age out
	flows []agg.NetFlow

	// cumulative byte counters
	totalReadBytes  int64
	totalWriteBytes int64
	totalTxBytes    int64
	totalRxBytes    int64

	// tick counter
	tick int

	// pseudo-random seed state
	rng *rand.Rand
}

var (
	agyFiles = []string{
		"/home/ramum/.gemini/antigravity-cli/brain/10549066/transcript.jsonl",
		"/home/ramum/agent-tracking/tracker.bpf.c",
		"/home/ramum/agent-tracking/main.go",
		"/sys/kernel/debug/tracing/events/raw_syscalls/enable",
		"/proc/self/status",
		"/proc/self/stat",
		"/home/ramum/.local/bin/agy",
		"/var/run/docker.sock",
	}

	paymentsFiles = []string{
		"/etc/resolv.conf",
		"/var/lib/agent/ledger.db",
		"/var/lib/agent/transactions.log",
		"/tmp/scratch-9999",
		"/proc/self/status",
		"/proc/self/maps",
		"/etc/ssl/certs/ca-certificates.crt",
		"/var/run/secrets/token",
	}

	mockSyscalls = []string{
		"epoll_pwait", "futex", "read", "write", "sendto",
		"recvfrom", "nanosleep", "clock_gettime", "getpid", "openat",
		"close", "madvise", "mmap", "munmap", "bpf",
	}

	syscallWeights = []float64{
		480, 390, 210, 180, 95,
		85, 80, 65, 45, 35,
		30, 25, 20, 15, 12,
	}

	staticFlowDst = []struct {
		ip   string
		port int
	}{
		{"10.0.0.10", 443},
		{"1.1.1.1", 53},
		{"10.0.1.50", 5432},
		{"127.0.0.1", 45907},
	}
)

type procTemplate struct {
	PID     int
	TGID    int
	PPID    int
	Comm    string
	Cmdline string
	Exe     string
	State   string
	BaseCPU float64
	Threads int
	BaseRSS int64
	OpenFDs int
}

func getMockProcessList() []procTemplate {
	return []procTemplate{
		{
			PID:     22786,
			TGID:    22786,
			PPID:    1001,
			Comm:    "agy",
			Cmdline: "/home/ramum/.local/bin/agy --server --tracking=ebpf",
			Exe:     "/home/ramum/.local/bin/agy",
			State:   "R",
			BaseCPU: 28.4,
			Threads: 26,
			BaseRSS: 184 * 1024 * 1024,
			OpenFDs: 42,
		},
		{
			PID:     9999,
			TGID:    9999,
			PPID:    1001,
			Comm:    "payments-agent",
			Cmdline: "/usr/bin/payments-agent --config /etc/agent.yaml --workers 16",
			Exe:     "/usr/bin/payments-agent",
			State:   "S",
			BaseCPU: 16.2,
			Threads: 16,
			BaseRSS: 52 * 1024 * 1024,
			OpenFDs: 34,
		},
		{
			PID:     24648,
			TGID:    24648,
			PPID:    24579,
			Comm:    "node",
			Cmdline: "/home/ramum/.vscode-server/bin/node --type=extensionHost",
			Exe:     "/home/ramum/.vscode-server/bin/node",
			State:   "S",
			BaseCPU: 4.7,
			Threads: 14,
			BaseRSS: 480 * 1024 * 1024,
			OpenFDs: 68,
		},
		{
			PID:     1420,
			TGID:    1420,
			PPID:    1,
			Comm:    "dockerd",
			Cmdline: "/usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock",
			Exe:     "/usr/bin/dockerd",
			State:   "S",
			BaseCPU: 3.8,
			Threads: 38,
			BaseRSS: 96 * 1024 * 1024,
			OpenFDs: 85,
		},
		{
			PID:     1821,
			TGID:    1821,
			PPID:    1,
			Comm:    "kubelet",
			Cmdline: "/usr/bin/kubelet --config=/var/lib/kubelet/config.yaml --node-ip=172.19.0.4",
			Exe:     "/usr/bin/kubelet",
			State:   "S",
			BaseCPU: 3.5,
			Threads: 22,
			BaseRSS: 84 * 1024 * 1024,
			OpenFDs: 54,
		},
	}
}

func NewMockGenerator(cfg *config.Config) *MockGenerator {
	comm := cfg.TargetComm()
	if comm == "" {
		comm = "agy"
	}
	return &MockGenerator{
		cfg:       cfg,
		cpuSeries: make([]agg.SeriesPoint, 0, 60),
		ioSeries:  make([]agg.IOPoint, 0, 60),
		netSeries: make([]agg.NetPoint, 0, 60),
		flows:     buildInitialFlows(comm),
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func buildInitialFlows(comm string) []agg.NetFlow {
	flows := make([]agg.NetFlow, 0, 4)
	sports := []int{49201, 49202, 49210, 49215}
	for i, dst := range staticFlowDst {
		flows = append(flows, agg.NetFlow{
			Src:     "192.168.1.100",
			Dst:     dst.ip,
			Sport:   sports[i],
			Dport:   dst.port,
			Proto:   "tcp",
			State:   "ESTABLISHED",
			BytesTx: 0,
			BytesRx: 0,
			PID:     22786,
			Comm:    comm,
		})
	}
	return flows
}

func (g *MockGenerator) Run(ctx context.Context, snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) {
	ticker := time.NewTicker(time.Duration(g.cfg.SnapshotMs) * time.Millisecond)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			g.tick++
			elapsedS := t.Sub(startTime).Seconds()

			currentPid, currentComm := g.cfg.Target()
			if currentComm == "" {
				currentComm = "agy"
			}

			// ── Build process table with multiple active processes ──────────
			templates := getMockProcessList()
			var processes []agg.ProcessRow
			var activeTemplate *procTemplate

			for i := range templates {
				tpl := &templates[i]
				// Match active process
				isTarget := false
				if currentPid > 0 && tpl.PID == currentPid {
					isTarget = true
				} else if currentPid == 0 && (strings.EqualFold(tpl.Comm, currentComm) || strings.HasPrefix(strings.ToLower(tpl.Comm), strings.ToLower(currentComm))) {
					isTarget = true
				}

				// Calculate oscillating CPU per process
				var pCpu float64
				if isTarget {
					activeTemplate = tpl
					pCpu = tpl.BaseCPU +
						10.0*math.Sin(elapsedS/6.0) +
						4.0*math.Sin(elapsedS/2.5) +
						g.rng.Float64()*4.0 - 2.0
				} else {
					pCpu = tpl.BaseCPU + 2.0*math.Sin(elapsedS/12.0+float64(tpl.PID)) + g.rng.Float64()*1.5 - 0.75
				}
				if pCpu < 0.2 {
					pCpu = 0.2
				}
				if pCpu > 95.0 {
					pCpu = 95.0
				}

				pRss := tpl.BaseRSS + int64(g.rng.Float64()*5*1024*1024)
				pThreads := tpl.Threads + int(g.rng.Float64()*3-1)
				if pThreads < 1 {
					pThreads = 1
				}
				pFds := tpl.OpenFDs + int(g.rng.Float64()*4-2)

				processes = append(processes, agg.ProcessRow{
					PID:       tpl.PID,
					TGID:      tpl.TGID,
					PPID:      tpl.PPID,
					Comm:      tpl.Comm,
					Cmdline:   tpl.Cmdline,
					Exe:       tpl.Exe,
					UID:       1000,
					State:     tpl.State,
					Threads:   pThreads,
					CPUPct:    pCpu,
					RSSBytes:  pRss,
					VMSBytes:  pRss * 4,
					OpenFDs:   pFds,
					CtxSw:     int64(g.tick*50 + tpl.PID),
					StartTime: startTime.Format(time.RFC3339),
				})
			}

			// If target wasn't one of the presets, add a custom row for it
			if activeTemplate == nil {
				targetPID := currentPid
				if targetPID <= 0 {
					targetPID = 31042
				}
				customCpu := 18.0 + 8.0*math.Sin(elapsedS/7.0) + g.rng.Float64()*3.0
				customRss := int64(64 * 1024 * 1024)
				customRow := agg.ProcessRow{
					PID:       targetPID,
					TGID:      targetPID,
					PPID:      1001,
					Comm:      currentComm,
					Cmdline:   fmt.Sprintf("/usr/bin/%s --active", currentComm),
					Exe:       fmt.Sprintf("/usr/bin/%s", currentComm),
					UID:       1000,
					State:     "R",
					Threads:   12,
					CPUPct:    customCpu,
					RSSBytes:  customRss,
					VMSBytes:  customRss * 4,
					OpenFDs:   30,
					CtxSw:     int64(g.tick * 60),
					StartTime: startTime.Format(time.RFC3339),
				}
				processes = append([]agg.ProcessRow{customRow}, processes...)
				activeTemplate = &procTemplate{
					PID:     targetPID,
					Comm:    currentComm,
					BaseCPU: customCpu,
					Threads: 12,
					BaseRSS: customRss,
					OpenFDs: 30,
				}
			}

			// Target-specific metrics
			targetRow := processes[0]
			for _, p := range processes {
				if (currentPid > 0 && p.PID == currentPid) || (currentPid == 0 && strings.EqualFold(p.Comm, currentComm)) {
					targetRow = p
					break
				}
			}

			cpu := targetRow.CPUPct
			threads := targetRow.Threads
			rssBytes := targetRow.RSSBytes
			openFDs := targetRow.OpenFDs

			// ── IO rates ──────────────────────────────────────────────────
			rBps := 14240.0 + g.rng.Float64()*180*1024
			wBps := 6120.0 + g.rng.Float64()*40*1024
			g.totalReadBytes += int64(rBps * float64(g.cfg.SnapshotMs) / 1000.0)
			g.totalWriteBytes += int64(wBps * float64(g.cfg.SnapshotMs) / 1000.0)

			// ── Network rates ─────────────────────────────────────────────
			txBps := 24480.0 + g.rng.Float64()*160*1024
			rxBps := 12240.0 + g.rng.Float64()*80*1024
			g.totalTxBytes += int64(txBps * float64(g.cfg.SnapshotMs) / 1000.0)
			g.totalRxBytes += int64(rxBps * float64(g.cfg.SnapshotMs) / 1000.0)

			// ── Update flow byte counters ─────────────────────────────────
			for i := range g.flows {
				g.flows[i].Comm = targetRow.Comm
				g.flows[i].PID = targetRow.PID
				bTx := int64(g.rng.Float64() * txBps * float64(g.cfg.SnapshotMs) / 1000.0 / float64(len(g.flows)))
				bRx := int64(g.rng.Float64() * rxBps * float64(g.cfg.SnapshotMs) / 1000.0 / float64(len(g.flows)))
				g.flows[i].BytesTx += bTx
				g.flows[i].BytesRx += bRx
			}

			// ── Syscall rates ─────────────────────────────────────────────
			totalSyscallRate := 850.0 + cpu*40.0 + g.rng.Float64()*150.0
			syscallStats := buildSyscallStats(totalSyscallRate, g.rng)
			errRate := g.rng.Float64() * 0.4

			// ── File access rates tailored to target ──────────────────────
			filesTop := buildFileStatsForComm(targetRow.Comm, g.rng)

			// ── Update rolling series ─────────────────────────────────────
			tMs := t.UnixMilli()
			g.cpuSeries = appendCapped(g.cpuSeries, 60, agg.SeriesPoint{T: tMs, CPUPct: cpu})
			g.ioSeries = appendIOCapped(g.ioSeries, 60, agg.IOPoint{T: tMs, RBPS: rBps, WBPS: wBps})
			g.netSeries = appendNetCapped(g.netSeries, 60, agg.NetPoint{T: tMs, TxBPS: txBps, RxBPS: rxBps})

			// ── SRE Golden Signals & Monarch Vitals ─────────────────────────
			p50 := 1.15 + (g.rng.Float64()*0.4 - 0.2)
			p90 := 4.80 + (g.rng.Float64()*1.2 - 0.6)
			p99 := 21.5 + (g.rng.Float64()*5.0 - 2.5)
			if errRate > 5.0 {
				p99 += 15.0
			}
			sloAvail := 99.95 - (errRate / (totalSyscallRate + 1.0) * 1.5)
			if sloAvail < 98.0 {
				sloAvail = 98.0
			}
			errorBudget := 94.2 - (errRate * 0.4)
			if errorBudget < 10.0 {
				errorBudget = 10.0
			}
			burnRate := 0.72 + (errRate * 0.15)
			runqueueLat := 0.85 + (cpu / 100.0 * 1.4)
			saturationPct := (cpu/100.0)*65.0 + float64(openFDs)/1024.0*35.0

			// ── Assemble snapshot ─────────────────────────────────────────
			snap := &agg.Snapshot{
				Meta: agg.SnapshotMeta{
					DroppedEvents: 0,
					EventRate:     totalSyscallRate / 10.0,
					UptimeS:       elapsedS,
					Target: agg.Target{
						PID:  targetRow.PID,
						Comm: targetRow.Comm,
					},
				},
				Processes: processes,
				KPIs: agg.KPIs{
					CPUPct:            cpu,
					Threads:           threads,
					RSSBytes:          rssBytes,
					OpenFDs:           openFDs,
					NetBpsTx:          txBps,
					NetBpsRx:          rxBps,
					DiskBpsR:          rBps,
					DiskBpsW:          wBps,
					SyscallsPerSec:    totalSyscallRate,
					ErrSyscallsPerSec: errRate,
					ConnectsPerSec:    g.connectsPerSec(),
				},
				SRE: agg.SREMetrics{
					LatencyP50Us:      p50,
					LatencyP90Us:      p90,
					LatencyP99Us:      p99,
					SLOAvailability:   sloAvail,
					ErrorBudgetPct:    errorBudget,
					BurnRate:          burnRate,
					RunqueueLatencyUs: runqueueLat,
					SaturationPct:     saturationPct,
				},
				SyscallsTop: syscallStats,
				FilesTop:    filesTop,
				Flows:       append([]agg.NetFlow(nil), g.flows...),
				CPUSeries:   append([]agg.SeriesPoint(nil), g.cpuSeries...),
				IOSeries:    append([]agg.IOPoint(nil), g.ioSeries...),
				NetSeries:   append([]agg.NetPoint(nil), g.netSeries...),
				Timeline:    nil,
			}

			// ── Emit events matching target process ─────────────────────────
			g.emitEventsForTarget(events, targetRow, elapsedS)

			// ── Send snapshot (non-blocking) ──────────────────────────────
			select {
			case snapshots <- snap:
			default:
			}
		}
	}
}

func (g *MockGenerator) connectsPerSec() float64 {
	if g.tick%75 == 0 {
		return 0.2
	}
	return 0.02
}

func (g *MockGenerator) emitEventsForTarget(events chan<- agg.EventRow, target agg.ProcessRow, elapsedS float64) {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	comm := target.Comm
	pid := target.PID

	isAgy := strings.EqualFold(comm, "agy") || strings.Contains(strings.ToLower(comm), "agy")

	// Outbound connect every ~4s
	if g.tick%(int(4000/g.cfg.SnapshotMs)) == 0 {
		dst := staticFlowDst[g.rng.Intn(len(staticFlowDst))]
		title := "outbound connect"
		detail := fmt.Sprintf("→ %s:%d via tcp", dst.ip, dst.port)
		if isAgy {
			title = "telemetry streaming"
			detail = fmt.Sprintf("agy: stream batch to %s:%d", dst.ip, dst.port)
		}
		send(events, agg.EventRow{
			ID:       uid(),
			TS:       nowStr,
			Severity: "info",
			Category: "network",
			PID:      pid,
			Comm:     comm,
			Title:    title,
			Detail:   detail,
			Attrs: map[string]interface{}{
				"dst": dst.ip, "dport": dst.port, "proto": "tcp", "state": "ESTABLISHED",
			},
		})
	}

	// File open every ~3s
	if g.tick%(int(3000/g.cfg.SnapshotMs)) == 0 {
		var path string
		if isAgy {
			path = agyFiles[g.rng.Intn(len(agyFiles))]
		} else {
			path = paymentsFiles[g.rng.Intn(len(paymentsFiles))]
		}
		send(events, agg.EventRow{
			ID:       uid(),
			TS:       nowStr,
			Severity: "info",
			Category: "file",
			PID:      pid,
			Comm:     comm,
			Title:    "file open",
			Detail:   path,
			Attrs:    map[string]interface{}{"path": path, "flags": "O_RDONLY"},
		})
	}

	// agy-specific syscall event batch or permission check
	if isAgy && g.tick%(int(15000/g.cfg.SnapshotMs)) == 0 {
		send(events, agg.EventRow{
			ID:       uid(),
			TS:       nowStr,
			Severity: "info",
			Category: "syscall",
			PID:      pid,
			Comm:     comm,
			Title:    "epoll_pwait batch flush",
			Detail:   fmt.Sprintf("agy handled %d syscalls in 2.0s window", 1200+g.rng.Intn(400)),
			Attrs:    map[string]interface{}{"syscall": "epoll_pwait", "ret": 281, "batch_size": 15},
		})
	}

	// Security alert every ~60s
	if g.tick%(int(60000/g.cfg.SnapshotMs)) == 0 && g.tick > 0 {
		detail := "/etc/shadow — EACCES (uid=1000)"
		if isAgy {
			detail = "/proc/kallsyms — CAP_BPF / root required"
		}
		send(events, agg.EventRow{
			ID:       uid(),
			TS:       nowStr,
			Severity: "warn",
			Category: "file",
			PID:      pid,
			Comm:     comm,
			Title:    "permission denied",
			Detail:   detail,
			Attrs:    map[string]interface{}{"path": "/etc/shadow", "flags": "O_RDONLY", "errno": -13},
		})
	}

	// Child exec every ~90s
	if g.tick%(int(90000/g.cfg.SnapshotMs)) == 0 && g.tick > 0 {
		childComm := "agent-plugin"
		childExe := "/usr/lib/agent/plugins/agent-plugin"
		if isAgy {
			childComm = "agy-tracer"
			childExe = "/home/ramum/agent-tracking/agy-tracer"
		}
		send(events, agg.EventRow{
			ID:       uid(),
			TS:       nowStr,
			Severity: "info",
			Category: "process",
			PID:      pid + 100 + g.rng.Intn(50),
			Comm:     childComm,
			Title:    "child exec",
			Detail:   childExe,
			Attrs:    map[string]interface{}{"ppid": pid, "exe": childExe},
		})
	}

	// CPU spike event every ~45s
	if g.tick%(int(45000/g.cfg.SnapshotMs)) == 0 && g.tick > 0 {
		send(events, agg.EventRow{
			ID:       uid(),
			TS:       nowStr,
			Severity: "info",
			Category: "compute",
			PID:      pid,
			Comm:     comm,
			Title:    "compute burst",
			Detail:   fmt.Sprintf("%.1f%% CPU for %dms", target.CPUPct, 300+g.rng.Intn(600)),
			Attrs:    map[string]interface{}{"cpu_pct": target.CPUPct, "uptime_s": elapsedS},
		})
	}
}

func send(ch chan<- agg.EventRow, e agg.EventRow) {
	select {
	case ch <- e:
	default:
	}
}

func uid() string {
	return fmt.Sprintf("evt-%d", time.Now().UnixNano())
}

func buildSyscallStats(total float64, rng *rand.Rand) []agg.SyscallStat {
	weightSum := 0.0
	for _, w := range syscallWeights {
		weightSum += w
	}
	stats := make([]agg.SyscallStat, len(mockSyscalls))
	for i, name := range mockSyscalls {
		share := syscallWeights[i] / weightSum
		jitter := 0.9 + rng.Float64()*0.2
		countS := total * share * jitter
		errS := 0.0
		if name == "openat" || name == "read" {
			errS = rng.Float64() * 0.1
		}
		stats[i] = agg.SyscallStat{Name: name, CountS: countS, ErrorsS: errS}
	}
	return stats
}

func buildFileStatsForComm(comm string, rng *rand.Rand) []agg.FileStat {
	if strings.EqualFold(comm, "agy") || strings.Contains(strings.ToLower(comm), "agy") {
		return []agg.FileStat{
			{Path: "/home/ramum/.gemini/antigravity-cli/brain/transcript.jsonl", OpsS: 18.5 + rng.Float64()*6, BytesS: 16384 + rng.Float64()*8192, Errors: 0},
			{Path: "/home/ramum/agent-tracking/tracker.bpf.c", OpsS: 6.2 + rng.Float64()*2, BytesS: 4096, Errors: 0},
			{Path: "/sys/kernel/debug/tracing/trace_pipe", OpsS: 24.1 + rng.Float64()*10, BytesS: 32768 + rng.Float64()*16384, Errors: 0},
			{Path: "/proc/self/status", OpsS: 8.0 + rng.Float64()*3, BytesS: 1024, Errors: 0},
			{Path: "/home/ramum/.local/bin/agy", OpsS: 1.2 + rng.Float64(), BytesS: 65536, Errors: 0},
			{Path: "/proc/kallsyms", OpsS: 0.1, BytesS: 0, Errors: 1},
		}
	}
	return []agg.FileStat{
		{Path: "/var/lib/agent/ledger.db", OpsS: 14.3 + rng.Float64()*5, BytesS: 8192 + rng.Float64()*4096, Errors: 0},
		{Path: "/var/lib/agent/transactions.log", OpsS: 9.1 + rng.Float64()*3, BytesS: 2048 + rng.Float64()*1024, Errors: 0},
		{Path: "/etc/resolv.conf", OpsS: 2.0 + rng.Float64(), BytesS: 512, Errors: 0},
		{Path: "/proc/self/status", OpsS: 6.0 + rng.Float64()*2, BytesS: 256, Errors: 0},
		{Path: "/etc/ssl/certs/ca-certificates.crt", OpsS: 1.1 + rng.Float64()*0.5, BytesS: 32768, Errors: 0},
		{Path: "/etc/shadow", OpsS: 0.1, BytesS: 0, Errors: 1},
	}
}

func appendCapped(s []agg.SeriesPoint, cap int, p agg.SeriesPoint) []agg.SeriesPoint {
	s = append(s, p)
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}

func appendIOCapped(s []agg.IOPoint, cap int, p agg.IOPoint) []agg.IOPoint {
	s = append(s, p)
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}

func appendNetCapped(s []agg.NetPoint, cap int, p agg.NetPoint) []agg.NetPoint {
	s = append(s, p)
	if len(s) > cap {
		s = s[len(s)-cap:]
	}
	return s
}
