package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/hub"
	"github.com/agentscope/agentscope/internal/storage"
)

func TestBorgHealthEndpoints(t *testing.T) {
	s, _ := testServer(t)
	h := s.Handler()

	for _, path := range []string{"/healthz", "/livez", "/readyz"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, localReq(http.MethodGet, path, ""))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s returned %d, want 200", path, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "ok") {
			t.Fatalf("%s body want 'ok', got %q", path, rr.Body.String())
		}
	}
}

func TestPrometheusMetricsEndpoint(t *testing.T) {
	s, _ := testServer(t)
	// Update snapshot with rich SRE metrics
	s.builder.Update(&agg.Snapshot{
		Meta: agg.SnapshotMeta{Target: agg.Target{Comm: "agy", PID: 22786}},
		KPIs: agg.KPIs{
			CPUPct:            34.5,
			Threads:           28,
			RSSBytes:          184000000,
			OpenFDs:           45,
			SyscallsPerSec:    1450.2,
			ErrSyscallsPerSec: 1.5,
			NetBpsTx:          12000,
			NetBpsRx:          24000,
			DiskBpsR:          8000,
			DiskBpsW:          4000,
		},
		SRE: agg.SREMetrics{
			LatencyP50Us:      1.25,
			LatencyP90Us:      4.95,
			LatencyP99Us:      22.4,
			SLOAvailability:   99.96,
			ErrorBudgetPct:    94.5,
			BurnRate:          0.72,
			RunqueueLatencyUs: 0.88,
			SaturationPct:     24.2,
		},
	})

	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, localReq(http.MethodGet, "/metrics", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics returned %d", rr.Code)
	}

	body := rr.Body.String()
	expectedMetrics := []string{
		"drishti_target_info",
		"drishti_cpu_usage_ratio",
		"drishti_memory_rss_bytes",
		"drishti_open_fds",
		"drishti_threads_total",
		"drishti_syscalls_rate",
		"drishti_syscall_errors_rate",
		"drishti_network_transmit_bytes_per_second",
		"drishti_network_receive_bytes_per_second",
		"drishti_disk_read_bytes_per_second",
		"drishti_disk_write_bytes_per_second",
		"drishti_slo_availability_ratio",
		"drishti_slo_error_budget_ratio",
		"drishti_burn_rate",
		"drishti_runqueue_latency_microseconds",
		"drishti_latency_microseconds",
		"drishti_saturation_ratio",
		"drishti_active_ws_clients",
		"drishti_uptime_seconds",
	}

	for _, m := range expectedMetrics {
		if !strings.Contains(body, m) {
			t.Errorf("missing metric %s in /metrics response", m)
		}
	}

	// Test with nil snapshot builder
	emptyServer := NewServer(config.Load(), hub.NewHub(4), agg.NewSnapshotBuilder(), "mock", "", storage.NewMemory())
	rrEmpty := httptest.NewRecorder()
	emptyServer.Handler().ServeHTTP(rrEmpty, localReq(http.MethodGet, "/metrics", ""))
	if rrEmpty.Code != http.StatusOK {
		t.Fatalf("metrics with empty builder returned %d", rrEmpty.Code)
	}
}

func TestPerfettoTracesEndpoint(t *testing.T) {
	s, _ := testServer(t)

	// Case 1: with populated timeline
	s.builder.AddEvent(agg.EventRow{
		ID:       "evt-1",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Severity: "info",
		Category: "syscall",
		PID:      22786,
		Comm:     "agy",
		Title:    "sys_enter_read",
		Detail:   "fd=3",
		Attrs:    map[string]any{"fd": 3},
	})
	s.builder.AddEvent(agg.EventRow{
		ID:       "evt-2",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Severity: "warn",
		Category: "network",
		PID:      22786,
		Comm:     "agy",
		Title:    "connect",
		Detail:   "10.0.0.1:443",
		Attrs:    map[string]any{"dst": "10.0.0.1:443"},
	})
	s.builder.AddEvent(agg.EventRow{
		ID:       "evt-3",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Severity: "crit",
		Category: "file",
		PID:      22786,
		Comm:     "agy",
		Title:    "openat",
		Detail:   "/etc/shadow",
		Attrs:    map[string]any{"path": "/etc/shadow"},
	})
	s.builder.AddEvent(agg.EventRow{
		ID:       "evt-4",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Severity: "info",
		Category: "compute",
		PID:      22786,
		Comm:     "agy",
		Title:    "token_eval",
		Detail:   "tokens=512",
		Attrs:    map[string]any{"tokens": 512},
	})

	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, localReq(http.MethodGet, "/api/v1/traces/perfetto", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("perfetto returned %d", rr.Code)
	}

	var traceResp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &traceResp); err != nil {
		t.Fatalf("unmarshal perfetto trace: %v", err)
	}
	events, ok := traceResp["traceEvents"].([]any)
	if !ok || len(events) < 4 {
		t.Fatalf("expected at least 4 trace events, got %d", len(events))
	}

	// Case 2: empty timeline
	emptyServer := NewServer(config.Load(), hub.NewHub(4), agg.NewSnapshotBuilder(), "mock", "", storage.NewMemory())
	rrEmpty := httptest.NewRecorder()
	emptyServer.Handler().ServeHTTP(rrEmpty, localReq(http.MethodGet, "/api/v1/traces/perfetto", ""))
	if rrEmpty.Code != http.StatusOK {
		t.Fatalf("perfetto with empty timeline returned %d", rrEmpty.Code)
	}
}

func TestCloudLoggingEndpoint(t *testing.T) {
	s, _ := testServer(t)

	s.builder.AddEvent(agg.EventRow{
		ID:       "log-1",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Severity: "warn",
		Category: "syscall",
		PID:      22786,
		Comm:     "agy",
		Title:    "EACCES",
		Detail:   "permission denied on /etc/shadow",
		Attrs:    map[string]any{"err": 13},
	})
	s.builder.AddEvent(agg.EventRow{
		ID:       "log-2",
		TS:       time.Now().UTC().Format(time.RFC3339),
		Severity: "crit",
		Category: "process",
		PID:      22786,
		Comm:     "agy",
		Title:    "SIGSEGV",
		Detail:   "segfault at 0x0",
		Attrs:    map[string]any{"sig": 11},
	})

	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, localReq(http.MethodGet, "/api/v1/logs", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("cloud logging returned %d", rr.Code)
	}

	var logs []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &logs); err != nil {
		t.Fatalf("unmarshal cloud logs: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d", len(logs))
	}
	if logs[0]["severity"] != "WARNING" && logs[0]["severity"] != "ERROR" {
		t.Errorf("expected severity WARNING or ERROR, got %v", logs[0]["severity"])
	}

	// Test with empty timeline
	emptyServer := NewServer(config.Load(), hub.NewHub(4), agg.NewSnapshotBuilder(), "mock", "", storage.NewMemory())
	rrEmpty := httptest.NewRecorder()
	emptyServer.Handler().ServeHTTP(rrEmpty, localReq(http.MethodGet, "/api/v1/logs", ""))
	if rrEmpty.Code != http.StatusOK {
		t.Fatalf("cloud logs empty returned %d", rrEmpty.Code)
	}
}

func TestTargetHandlerErrors(t *testing.T) {
	s, _ := testServer(t)
	h := s.Handler()

	// 1. Unsupported media type
	rBadCt := httptest.NewRequest(http.MethodPost, "/api/target", strings.NewReader(`{}`))
	rBadCt.RemoteAddr = "127.0.0.1:1234"
	rBadCt.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, rBadCt)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}

	// 2. Invalid JSON
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, localReq(http.MethodPost, "/api/target", "not-json"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	// 3. Empty target object
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, localReq(http.MethodPost, "/api/target", "{}"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty object, got %d", rr.Code)
	}

	// 4. Target with comm only
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, localReq(http.MethodPost, "/api/target", `{"comm":"payments-agent"}`))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for comm-only, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServerStartShutdown(t *testing.T) {
	cfg := config.Load()
	cfg.HttpAddr = "127.0.0.1:0" // OS-assigned port
	h := hub.NewHub(4)
	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)
	defer cancel()

	b := agg.NewSnapshotBuilder()
	s := NewServer(cfg, h, b, "mock", "", storage.NewMemory())

	ctxServer, cancelServer := context.WithCancel(context.Background())
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- s.Start(ctxServer)
	}()

	// Allow server to bind
	time.Sleep(50 * time.Millisecond)

	// Cancel and check graceful shutdown
	cancelServer()
	select {
	case err := <-serverErrCh:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("server shutdown returned unexpected err: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shutdown gracefully within timeout")
	}
}

func TestFindStaticDirAndSPAHandler(t *testing.T) {
	// Test findStaticDir
	tmp := t.TempDir()
	indexFile := filepath.Join(tmp, "index.html")
	_ = os.WriteFile(indexFile, []byte("<!DOCTYPE html><html><body>Test SPA</body></html>"), 0644)

	found := findStaticDir(tmp)
	if found != tmp {
		t.Fatalf("expected %s, got %s", tmp, found)
	}

	if findStaticDir("/nonexistent-dir-for-sure-xyz") != "" {
		t.Fatal("expected empty string for nonexistent directory")
	}

	// Test spaHandler
	h := spaHandler(tmp)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/deep/link", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("spa fallback returned %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Test SPA") {
		t.Fatalf("unexpected spa response: %s", rr.Body.String())
	}
}

func TestKernelStringAndAtoi(t *testing.T) {
	k := kernelString()
	if k == "" {
		t.Fatal("kernelString returned empty")
	}

	if atoi("123") != 123 {
		t.Fatalf("expected 123, got %d", atoi("123"))
	}
	if atoi("abc") != 0 {
		t.Fatalf("expected 0, got %d", atoi("abc"))
	}
}
