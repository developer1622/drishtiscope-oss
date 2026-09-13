package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/hub"
	"github.com/agentscope/agentscope/internal/storage"
)

func localReq(method, path string, body string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	r.RemoteAddr = "127.0.0.1:4321"
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func testServer(t *testing.T) (*Server, *storage.Memory) {
	t.Helper()
	cfg := config.Load()
	cfg.AllowInsecureWS = false
	cfg.CORSOrigins = []string{"http://localhost:5173"}
	_ = cfg.SetTarget(0, "agy")
	h := hub.NewHub(8)
	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)
	t.Cleanup(cancel)
	mem := storage.NewMemory()
	b := agg.NewSnapshotBuilder()
	b.Update(&agg.Snapshot{
		Meta:      agg.SnapshotMeta{Target: agg.Target{Comm: "agy", PID: 7}},
		Processes: []agg.ProcessRow{{PID: 7, Comm: "agy"}},
		KPIs:      agg.KPIs{CPUPct: 3.5},
	})
	return NewServer(cfg, h, b, "mock", "", mem), mem
}

func TestHealthAndMeta(t *testing.T) {
	s, _ := testServer(t)
	h := s.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, localReq(http.MethodGet, "/api/health", ""))
	if rr.Code != 200 {
		t.Fatalf("health %d %s", rr.Code, rr.Body)
	}
	if !strings.Contains(rr.Body.String(), `"ok":true`) {
		t.Fatal(rr.Body.String())
	}
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing security header")
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, localReq(http.MethodPost, "/api/health", ""))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405 got %d", rr.Code)
	}
}

func TestSnapshot(t *testing.T) {
	s, _ := testServer(t)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, localReq(http.MethodGet, "/api/snapshot", ""))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	var env map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env["kind"] != "snapshot" || env["mode"] != "mock" {
		t.Fatalf("%v", env)
	}
}

func TestTargetValidation(t *testing.T) {
	s, _ := testServer(t)
	h := s.Handler()

	post := func(body string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, localReq(http.MethodPost, "/api/target", body))
		return rr
	}

	if rr := post(`{}`); rr.Code != 400 {
		t.Fatalf("empty: %d %s", rr.Code, rr.Body)
	}
	if rr := post(`{"pid":-1}`); rr.Code != 400 {
		t.Fatalf("neg pid: %d %s", rr.Code, rr.Body)
	}
	if rr := post(`{"comm":"../etc/passwd"}`); rr.Code != 400 {
		t.Fatalf("path comm: %d %s", rr.Code, rr.Body)
	}
	rr := post(`{"comm":"nginx"}`)
	if rr.Code != 200 {
		t.Fatalf("nginx: %d %s", rr.Code, rr.Body)
	}
	pid, comm := s.cfg.Target()
	if comm != "nginx" || pid != 0 {
		t.Fatalf("target %d %s", pid, comm)
	}
	rr = post(`{"pid":99}`)
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	if s.cfg.TargetPID() != 99 {
		t.Fatal("pid not set")
	}
}

func TestHistoryUsesStore(t *testing.T) {
	s, mem := testServer(t)
	_ = mem.RecordSnapshot(&agg.Snapshot{
		Meta: agg.SnapshotMeta{Target: agg.Target{Comm: "agy", PID: 7}},
		KPIs: agg.KPIs{CPUPct: 8},
	})
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, localReq(http.MethodGet, "/api/history?limit=10&comm=agy", ""))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	var points []storage.SnapshotHistoryPoint
	if err := json.Unmarshal(rr.Body.Bytes(), &points); err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].CPUPct != 8 {
		t.Fatalf("%+v", points)
	}
}

func TestCORSAllowlist(t *testing.T) {
	s, _ := testServer(t)
	h := s.Handler()

	req := localReq(http.MethodGet, "/api/health", "")
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("cors %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}

	req = localReq(http.MethodGet, "/api/health", "")
	req.Header.Set("Origin", "http://evil.example")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("evil origin reflected")
	}
}

func TestOversizedBodyRejected(t *testing.T) {
	s, _ := testServer(t)
	s.cfg.MaxBodyBytes = 32
	body := bytes.Repeat([]byte("a"), 64)
	req := httptest.NewRequest(http.MethodPost, "/api/target", bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("want 400 got %d %s", rr.Code, rr.Body)
	}
}

func TestSPARejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := spaHandler(dir)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.URL.Path = "/foo/../../../etc/passwd"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("got %d body %s", rr.Code, rr.Body)
	}
}

func TestEventsHistory(t *testing.T) {
	s, mem := testServer(t)
	_ = mem.RecordEvent(agg.EventRow{ID: "e1", Title: "exec", Comm: "agy"})
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, localReq(http.MethodGet, "/api/events/history?limit=5", ""))
	if rr.Code != 200 {
		t.Fatal(rr.Body.String())
	}
	var ev []agg.EventRow
	if err := json.Unmarshal(rr.Body.Bytes(), &ev); err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 || ev[0].ID != "e1" {
		t.Fatalf("%+v", ev)
	}
}

func TestLoopbackOnlyWithoutToken(t *testing.T) {
	s, _ := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	req.RemoteAddr = "203.0.113.9:9"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("got %d %s", rr.Code, rr.Body)
	}

	health := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	health.RemoteAddr = "203.0.113.9:9"
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, health)
	if rr.Code != 200 {
		t.Fatalf("health must stay public, got %d", rr.Code)
	}
}

func TestBearerAuth(t *testing.T) {
	s, _ := testServer(t)
	s.cfg.AuthToken = "0123456789abcdef"
	h := s.Handler()

	unauth := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	unauth.RemoteAddr = "203.0.113.9:9"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, unauth)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauth %d %s", rr.Code, rr.Body)
	}

	wrong := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	wrong.RemoteAddr = "203.0.113.9:9"
	wrong.Header.Set("Authorization", "Bearer wrong-token-value")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, wrong)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token %d", rr.Code)
	}

	ok := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	ok.RemoteAddr = "203.0.113.9:9"
	ok.Header.Set("Authorization", "Bearer 0123456789abcdef")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, ok)
	if rr.Code != 200 {
		t.Fatalf("bearer %d %s", rr.Code, rr.Body)
	}
}

func TestContentTypeEnforced(t *testing.T) {
	s, _ := testServer(t)
	req := localReq(http.MethodPost, "/api/target", `{"comm":"agy"}`)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestXForwardedForIgnoredByDefault(t *testing.T) {
	s, _ := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	req.RemoteAddr = "203.0.113.9:9"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("xff spoof got %d", rr.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	s, _ := testServer(t)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, localReq(http.MethodGet, "/api/health", ""))
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("csp")
	}
	if rr.Header().Get("Permissions-Policy") == "" {
		t.Fatal("permissions-policy")
	}
}
