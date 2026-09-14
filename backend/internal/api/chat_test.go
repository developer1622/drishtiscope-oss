package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/hub"
	"github.com/agentscope/agentscope/internal/storage"
)

func TestGeminiDirect(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not set; skipping live cloud test")
	}

	msgs := []chatMsg{
		{Role: "system", Content: "You are DrishtiScope Agent Copilot."},
		{Role: "user", Content: "Give a 1-sentence performance tip for Linux."},
	}

	reply, model, err := callGemini(context.Background(), key, msgs, "")
	if err != nil {
		t.Logf("callGemini non-fatal: %v", err)
		return
	}
	t.Logf("Success! Model=%s, Reply=%s", model, reply)
}

func setupTestServer() (*Server, *agg.SnapshotBuilder) {
	cfg := config.Load()
	_ = cfg.SetTarget(1234, "agy")
	h := hub.NewHub(32)
	builder := agg.NewSnapshotBuilder()

	snap := &agg.Snapshot{
		Meta: agg.SnapshotMeta{
			DroppedEvents: 0,
			EventRate:     250.0,
			UptimeS:       120.0,
			Target: agg.Target{
				PID:  1234,
				Comm: "agy",
			},
		},
		Processes: []agg.ProcessRow{
			{
				PID:      1234,
				Comm:     "agy",
				Cmdline:  "/usr/bin/agy --server",
				CPUPct:   42.5,
				RSSBytes: 256 * 1024 * 1024,
				Threads:  18,
				OpenFDs:  38,
			},
		},
		KPIs: agg.KPIs{
			CPUPct:            42.5,
			Threads:           18,
			RSSBytes:          256 * 1024 * 1024,
			OpenFDs:           38,
			NetBpsTx:          15000,
			NetBpsRx:          45000,
			DiskBpsR:          12000,
			DiskBpsW:          8000,
			SyscallsPerSec:    1450,
			ErrSyscallsPerSec: 1.8,
			ConnectsPerSec:    2.5,
		},
		SRE: agg.SREMetrics{
			SLOAvailability:    99.92,
			ErrorBudgetPct:     88.4,
			BurnRate:           0.85,
			RunqueueLatencyUs:  0.92,
			LatencyP50Us:       1.24,
			LatencyP90Us:       4.85,
			LatencyP99Us:       22.5,
			SaturationPct:      22.0,
		},
		SyscallsTop: []agg.SyscallStat{
			{Name: "futex", CountS: 620, ErrorsS: 0},
			{Name: "epoll_wait", CountS: 410, ErrorsS: 0},
			{Name: "openat", CountS: 45, ErrorsS: 2},
		},
		FilesTop: []agg.FileStat{
			{Path: "/var/lib/agent/ledger.db", OpsS: 12.0, BytesS: 4096, Errors: 0},
			{Path: "/etc/shadow", OpsS: 0.1, BytesS: 0, Errors: 1},
		},
		Flows: []agg.NetFlow{
			{Src: "127.0.0.1", Dst: "10.0.0.10", Sport: 44212, Dport: 443, Proto: "tcp", State: "ESTABLISHED", BytesTx: 10240, BytesRx: 65536, PID: 1234, Comm: "agy"},
		},
	}
	builder.Update(snap)

	srv := NewServer(cfg, h, builder, "mock", "", storage.Nop{})
	return srv, builder
}

func TestHandleChat_InvalidJSON(t *testing.T) {
	srv, _ := setupTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader("invalid-json"))
	rec := httptest.NewRecorder()

	srv.handleChat(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestHandleChat_RuleBasedQueries(t *testing.T) {
	srv, _ := setupTestServer()

	queries := []string{
		"Why is CPU high?",
		"Is there a memory leak in RSS?",
		"Show me network connections and open sockets",
		"What syscall errors are failing?",
		"Security audit on restricted files like shadow",
		"How do I profile with perf and bpftool?",
		"General optimization advice",
	}

	for _, q := range queries {
		chatReq := chatRequest{
			Model: "rule-engine",
			Messages: []chatMsg{
				{Role: "user", Content: q},
			},
			IncludeSnapshot: true,
		}
		body, _ := json.Marshal(chatReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		srv.handleChat(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for query %q, got %d", q, rec.Code)
		}

		var resp chatResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("failed to unmarshal response for %q: %v", q, err)
		}
		if resp.Reply == "" {
			t.Errorf("expected non-empty reply for query %q", q)
		}
	}
}

func TestHandleChat_EmptySnapshot(t *testing.T) {
	cfg := config.Load()
	h := hub.NewHub(32)
	emptyBuilder := agg.NewSnapshotBuilder()
	srv := NewServer(cfg, h, emptyBuilder, "mock", "", storage.Nop{})

	chatReq := chatRequest{
		Messages: []chatMsg{
			{Role: "user", Content: "Hello agent"},
		},
		IncludeSnapshot: true,
	}
	body, _ := json.Marshal(chatReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleChat(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestHandleMetaAndHello(t *testing.T) {
	srv, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	rec := httptest.NewRecorder()
	srv.handleMeta(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	h := srv.buildHello()
	if h.Target.Comm != "agy" {
		t.Fatalf("expected target comm agy, got %s", h.Target.Comm)
	}
}

func TestClientIPEdgeCases(t *testing.T) {
	// XFF header
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Forwarded-For", "192.168.1.50, 10.0.0.1")
	if ip := clientIP(req1, true); ip != "192.168.1.50" {
		t.Errorf("expected 192.168.1.50, got %s", ip)
	}

	// X-Real-IP header
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Real-IP", "172.16.0.25")
	if ip := clientIP(req2, true); ip != "172.16.0.25" {
		t.Errorf("expected 172.16.0.25, got %s", ip)
	}

	// Malformed RemoteAddr without port
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "custom-host-string"
	if ip := clientIP(req3, false); ip != "custom-host-string" {
		t.Errorf("expected custom-host-string, got %s", ip)
	}

	// Invalid loopback
	if isLoopbackIP("not-an-ip") {
		t.Error("expected false for not-an-ip")
	}
}
