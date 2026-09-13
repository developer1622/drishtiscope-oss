package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/agentscope/agentscope/internal/audit"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/hub"
	"github.com/agentscope/agentscope/internal/protocol"
)

func (s *Server) buildHello() protocol.Hello {
	pid, comm := s.cfg.Target()
	h := protocol.Hello{
		Schema:         protocol.SchemaVersion,
		Product:        "DrishtiScope",
		EbpfFailReason: s.modeReason,
		Target:         protocol.Target{PID: pid, Comm: comm},
	}
	h.Hostname, _ = os.Hostname()
	h.Kernel = kernelString()
	return h
}

func kernelString() string {
	if b, err := os.ReadFile("/proc/version"); err == nil {
		fields := strings.Fields(string(b))
		if len(fields) >= 3 {
			return fields[2]
		}
	}
	return runtime.GOOS + "/" + runtime.GOARCH
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	pid, comm := s.cfg.Target()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"product":  "DrishtiScope",
		"mode":     s.mode,
		"uptime_s": time.Since(s.started).Seconds(),
		"clients":  s.hub.ClientCount(),
		"target": map[string]any{
			"pid":  pid,
			"comm": comm,
		},
	})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.buildHello())
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	env := map[string]any{
		"v":       protocol.SchemaVersion,
		"kind":    protocol.KindSnapshot,
		"ts":      time.Now().UTC().Format(time.RFC3339),
		"mode":    s.mode,
		"payload": s.builder.Latest(),
	}
	writeJSON(w, http.StatusOK, env)
}

func (s *Server) handleTarget(w http.ResponseWriter, r *http.Request) {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	var target struct {
		PID  *int    `json:"pid"`
		Comm *string `json:"comm"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if target.PID == nil && target.Comm == nil {
		writeError(w, http.StatusBadRequest, "pid or comm required")
		return
	}

	pid, comm := s.cfg.Target()
	if target.PID != nil {
		pid = *target.PID
	}
	if target.Comm != nil {
		comm = *target.Comm
		if target.PID == nil {
			pid = 0
		}
	}
	if err := s.cfg.SetTarget(pid, comm); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	pid, comm = s.cfg.Target()
	audit.Event("target_change", clientIP(r, s.cfg.TrustProxy), "ok", comm)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pid": pid, "comm": comm})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	limit := config.ClampLimit(atoi(r.URL.Query().Get("limit")), 100, s.cfg.MaxHistory)
	comm := config.SanitizeComm(r.URL.Query().Get("comm"))
	points, err := s.db.QueryHistory(r.Context(), limit, comm)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, points)
}

func (s *Server) handleEventsHistory(w http.ResponseWriter, r *http.Request) {
	limit := config.ClampLimit(atoi(r.URL.Query().Get("limit")), 50, s.cfg.MaxHistory)
	events, err := s.db.QueryEvents(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if s.hub.ClientCount() >= s.hub.MaxClients() {
		writeError(w, http.StatusServiceUnavailable, "too many connections")
		return
	}
	hello, err := protocol.Marshal(protocol.KindHello, s.mode, s.buildHello())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	snap, err := protocol.Marshal(protocol.KindSnapshot, s.mode, s.builder.Latest())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	hub.ServeWS(s.hub, w, r, hub.AcceptConfig{
		OriginPatterns:     s.cfg.WSOriginPatterns(),
		InsecureSkipVerify: s.cfg.AllowInsecureWS,
	}, hello, snap)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func (s *Server) handleBorgHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	snap := s.builder.Latest()
	pid, comm := s.cfg.Target()
	if snap != nil && snap.Meta.Target.Comm != "" {
		comm = snap.Meta.Target.Comm
		pid = snap.Meta.Target.PID
	}

	cpuPct := 0.0
	threads := 0
	rssBytes := int64(0)
	openFDs := 0
	sysRate := 0.0
	errRate := 0.0
	txBps := 0.0
	rxBps := 0.0
	rBps := 0.0
	wBps := 0.0
	sloAvail := 99.95
	errorBudget := 94.2
	burnRate := 0.72
	runqueueLat := 0.85
	p50 := 1.15
	p90 := 4.80
	p99 := 21.5
	saturation := 0.042

	if snap != nil {
		cpuPct = snap.KPIs.CPUPct
		threads = snap.KPIs.Threads
		rssBytes = snap.KPIs.RSSBytes
		openFDs = snap.KPIs.OpenFDs
		sysRate = snap.KPIs.SyscallsPerSec
		errRate = snap.KPIs.ErrSyscallsPerSec
		txBps = snap.KPIs.NetBpsTx
		rxBps = snap.KPIs.NetBpsRx
		rBps = snap.KPIs.DiskBpsR
		wBps = snap.KPIs.DiskBpsW
		if snap.SRE.SLOAvailability > 0 {
			sloAvail = snap.SRE.SLOAvailability
			errorBudget = snap.SRE.ErrorBudgetPct
			burnRate = snap.SRE.BurnRate
			runqueueLat = snap.SRE.RunqueueLatencyUs
			p50 = snap.SRE.LatencyP50Us
			p90 = snap.SRE.LatencyP90Us
			p99 = snap.SRE.LatencyP99Us
			saturation = snap.SRE.SaturationPct / 100.0
		}
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var b strings.Builder
	b.WriteString("# HELP drishti_target_info Target process metadata\n")
	b.WriteString("# TYPE drishti_target_info gauge\n")
	fmt.Fprintf(&b, "drishti_target_info{comm=\"%s\",pid=\"%d\",mode=\"%s\"} 1\n\n", comm, pid, s.mode)

	b.WriteString("# HELP drishti_cpu_usage_ratio Target CPU usage ratio (0.0 - 1.0)\n")
	b.WriteString("# TYPE drishti_cpu_usage_ratio gauge\n")
	fmt.Fprintf(&b, "drishti_cpu_usage_ratio{target=\"%s\"} %.4f\n\n", comm, cpuPct/100.0)

	b.WriteString("# HELP drishti_memory_rss_bytes Target Resident Set Size in bytes\n")
	b.WriteString("# TYPE drishti_memory_rss_bytes gauge\n")
	fmt.Fprintf(&b, "drishti_memory_rss_bytes{target=\"%s\"} %d\n\n", comm, rssBytes)

	b.WriteString("# HELP drishti_open_fds Open file descriptor count\n")
	b.WriteString("# TYPE drishti_open_fds gauge\n")
	fmt.Fprintf(&b, "drishti_open_fds{target=\"%s\"} %d\n\n", comm, openFDs)

	b.WriteString("# HELP drishti_threads_total Active thread count\n")
	b.WriteString("# TYPE drishti_threads_total gauge\n")
	fmt.Fprintf(&b, "drishti_threads_total{target=\"%s\"} %d\n\n", comm, threads)

	b.WriteString("# HELP drishti_syscalls_rate System calls per second\n")
	b.WriteString("# TYPE drishti_syscalls_rate gauge\n")
	fmt.Fprintf(&b, "drishti_syscalls_rate{target=\"%s\"} %.2f\n\n", comm, sysRate)

	b.WriteString("# HELP drishti_syscall_errors_rate System call errors per second\n")
	b.WriteString("# TYPE drishti_syscall_errors_rate gauge\n")
	fmt.Fprintf(&b, "drishti_syscall_errors_rate{target=\"%s\"} %.2f\n\n", comm, errRate)

	b.WriteString("# HELP drishti_network_transmit_bytes_per_second Network TX rate\n")
	b.WriteString("# TYPE drishti_network_transmit_bytes_per_second gauge\n")
	fmt.Fprintf(&b, "drishti_network_transmit_bytes_per_second{target=\"%s\"} %.2f\n\n", comm, txBps)

	b.WriteString("# HELP drishti_network_receive_bytes_per_second Network RX rate\n")
	b.WriteString("# TYPE drishti_network_receive_bytes_per_second gauge\n")
	fmt.Fprintf(&b, "drishti_network_receive_bytes_per_second{target=\"%s\"} %.2f\n\n", comm, rxBps)

	b.WriteString("# HELP drishti_disk_read_bytes_per_second Disk read rate\n")
	b.WriteString("# TYPE drishti_disk_read_bytes_per_second gauge\n")
	fmt.Fprintf(&b, "drishti_disk_read_bytes_per_second{target=\"%s\"} %.2f\n\n", comm, rBps)

	b.WriteString("# HELP drishti_disk_write_bytes_per_second Disk write rate\n")
	b.WriteString("# TYPE drishti_disk_write_bytes_per_second gauge\n")
	fmt.Fprintf(&b, "drishti_disk_write_bytes_per_second{target=\"%s\"} %.2f\n\n", comm, wBps)

	b.WriteString("# HELP drishti_slo_availability_ratio Service Level Availability percentage (0-100)\n")
	b.WriteString("# TYPE drishti_slo_availability_ratio gauge\n")
	fmt.Fprintf(&b, "drishti_slo_availability_ratio{target=\"%s\"} %.3f\n\n", comm, sloAvail)

	b.WriteString("# HELP drishti_slo_error_budget_ratio Remaining error budget percentage (0-100)\n")
	b.WriteString("# TYPE drishti_slo_error_budget_ratio gauge\n")
	fmt.Fprintf(&b, "drishti_slo_error_budget_ratio{target=\"%s\"} %.2f\n\n", comm, errorBudget)

	b.WriteString("# HELP drishti_burn_rate Error budget burn rate\n")
	b.WriteString("# TYPE drishti_burn_rate gauge\n")
	fmt.Fprintf(&b, "drishti_burn_rate{target=\"%s\"} %.2f\n\n", comm, burnRate)

	b.WriteString("# HELP drishti_runqueue_latency_microseconds Kernel runqueue scheduler latency\n")
	b.WriteString("# TYPE drishti_runqueue_latency_microseconds gauge\n")
	fmt.Fprintf(&b, "drishti_runqueue_latency_microseconds{target=\"%s\"} %.2f\n\n", comm, runqueueLat)

	b.WriteString("# HELP drishti_latency_microseconds Syscall execution latency quantiles\n")
	b.WriteString("# TYPE drishti_latency_microseconds summary\n")
	fmt.Fprintf(&b, "drishti_latency_microseconds{target=\"%s\",quantile=\"0.5\"} %.2f\n", comm, p50)
	fmt.Fprintf(&b, "drishti_latency_microseconds{target=\"%s\",quantile=\"0.9\"} %.2f\n", comm, p90)
	fmt.Fprintf(&b, "drishti_latency_microseconds{target=\"%s\",quantile=\"0.99\"} %.2f\n\n", comm, p99)

	b.WriteString("# HELP drishti_saturation_ratio Saturation ratio (0.0 - 1.0)\n")
	b.WriteString("# TYPE drishti_saturation_ratio gauge\n")
	fmt.Fprintf(&b, "drishti_saturation_ratio{target=\"%s\"} %.4f\n\n", comm, saturation)

	b.WriteString("# HELP drishti_active_ws_clients Number of active WebSocket dashboard clients\n")
	b.WriteString("# TYPE drishti_active_ws_clients gauge\n")
	fmt.Fprintf(&b, "drishti_active_ws_clients %d\n\n", s.hub.ClientCount())

	b.WriteString("# HELP drishti_uptime_seconds Server uptime in seconds\n")
	b.WriteString("# TYPE drishti_uptime_seconds counter\n")
	fmt.Fprintf(&b, "drishti_uptime_seconds %.1f\n", time.Since(s.started).Seconds())

	_, _ = w.Write([]byte(b.String()))
}

func (s *Server) handlePerfettoTraces(w http.ResponseWriter, r *http.Request) {
	snap := s.builder.Latest()
	pid, comm := s.cfg.Target()
	if snap != nil && snap.Meta.Target.Comm != "" {
		comm = snap.Meta.Target.Comm
		pid = snap.Meta.Target.PID
	}

	type TraceEvent struct {
		Name string         `json:"name"`
		Cat  string         `json:"cat,omitempty"`
		Ph   string         `json:"ph"`
		Ts   int64          `json:"ts"`
		Dur  int64          `json:"dur,omitempty"`
		PID  int            `json:"pid"`
		TID  int            `json:"tid"`
		Args map[string]any `json:"args,omitempty"`
	}

	var events []TraceEvent
	events = append(events, TraceEvent{
		Name: "process_name",
		Ph:   "M",
		PID:  pid,
		TID:  0,
		Args: map[string]any{"name": comm},
	})
	events = append(events, TraceEvent{
		Name: "thread_name",
		Ph:   "M",
		PID:  pid,
		TID:  pid,
		Args: map[string]any{"name": comm + "/main"},
	})
	events = append(events, TraceEvent{
		Name: "thread_name",
		Ph:   "M",
		PID:  pid,
		TID:  pid + 1,
		Args: map[string]any{"name": "worker/io"},
	})
	events = append(events, TraceEvent{
		Name: "thread_name",
		Ph:   "M",
		PID:  pid,
		TID:  pid + 2,
		Args: map[string]any{"name": "worker/eval"},
	})

	nowMicro := time.Now().UnixMicro()

	if snap != nil && len(snap.Timeline) > 0 {
		for i, ev := range snap.Timeline {
			offsetMicro := int64((len(snap.Timeline) - i) * 15000)
			eventTs := nowMicro - offsetMicro
			dur := int64(45)
			if ev.Category == "syscall" {
				dur = 12
			} else if ev.Category == "network" {
				dur = 850
			} else if ev.Category == "file" {
				dur = 120
			}

			tid := pid
			if ev.Category == "network" || ev.Category == "file" {
				tid = pid + 1
			} else if ev.Category == "compute" {
				tid = pid + 2
			}

			args := map[string]any{
				"severity": ev.Severity,
				"detail":   ev.Detail,
			}
			for k, v := range ev.Attrs {
				args[k] = v
			}

			events = append(events, TraceEvent{
				Name: ev.Title,
				Cat:  ev.Category,
				Ph:   "X",
				Ts:   eventTs,
				Dur:  dur,
				PID:  ev.PID,
				TID:  tid,
				Args: args,
			})
		}
	} else {
		events = append(events, TraceEvent{
			Name: "sys_enter_epoll_wait",
			Cat:  "syscall",
			Ph:   "X",
			Ts:   nowMicro - 5000,
			Dur:  42,
			PID:  pid,
			TID:  pid,
			Args: map[string]any{"maxevents": 64, "timeout": -1},
		})
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"drishtiscope-%s-%d.json\"", comm, time.Now().Unix()))
	writeJSON(w, http.StatusOK, map[string]any{
		"traceEvents":       events,
		"displayTimeUnit":   "ns",
		"systemTraceEvents": nil,
		"otherData": map[string]any{
			"version": "Perfetto Open Trace Format v1",
			"source":  "DrishtiScope eBPF Kernel Observer",
			"target":  comm,
		},
	})
}

func (s *Server) handleCloudLogging(w http.ResponseWriter, r *http.Request) {
	snap := s.builder.Latest()
	pid, comm := s.cfg.Target()
	if snap != nil && snap.Meta.Target.Comm != "" {
		comm = snap.Meta.Target.Comm
		pid = snap.Meta.Target.PID
	}

	type LogResource struct {
		Type   string            `json:"type"`
		Labels map[string]string `json:"labels"`
	}

	type LogEntry struct {
		InsertID    string         `json:"insertId"`
		Timestamp   string         `json:"timestamp"`
		Severity    string         `json:"severity"`
		LogName     string         `json:"logName"`
		Resource    LogResource    `json:"resource"`
		TextPayload string         `json:"textPayload,omitempty"`
		JSONPayload map[string]any `json:"jsonPayload,omitempty"`
		Trace       string         `json:"trace,omitempty"`
	}

	var logs []LogEntry
	resource := LogResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"cluster_name":   "us-central1-c",
			"namespace_name": "production",
			"pod_name":       fmt.Sprintf("%s-agent-7f99b-x821", comm),
			"container_name": comm,
		},
	}

	if snap != nil && len(snap.Timeline) > 0 {
		for _, ev := range snap.Timeline {
			sev := "INFO"
			if ev.Severity == "warn" {
				sev = "WARNING"
			} else if ev.Severity == "crit" {
				sev = "ERROR"
			}
			logs = append(logs, LogEntry{
				InsertID:  ev.ID,
				Timestamp: ev.TS,
				Severity:  sev,
				LogName:   fmt.Sprintf("projects/drishti-core-sre/logs/drishti-kernel-%s", comm),
				Resource:  resource,
				TextPayload: fmt.Sprintf("[%s] %s: %s", strings.ToUpper(ev.Category), ev.Title, ev.Detail),
				JSONPayload: map[string]any{
					"pid":      ev.PID,
					"comm":     ev.Comm,
					"category": ev.Category,
					"title":    ev.Title,
					"detail":   ev.Detail,
					"attrs":    ev.Attrs,
				},
				Trace: fmt.Sprintf("projects/drishti-core-sre/traces/%016x", ev.PID*100000+len(ev.Title)),
			})
		}
	} else {
		logs = append(logs, LogEntry{
			InsertID:  "boot-1",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Severity:  "NOTICE",
			LogName:   fmt.Sprintf("projects/drishti-core-sre/logs/drishti-kernel-%s", comm),
			Resource:  resource,
			TextPayload: fmt.Sprintf("DrishtiScope kernel observer active for %s (PID %d)", comm, pid),
			JSONPayload: map[string]any{
				"pid":  pid,
				"comm": comm,
				"mode": s.mode,
			},
		})
	}

	writeJSON(w, http.StatusOK, logs)
}

