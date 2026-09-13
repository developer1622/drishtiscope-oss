package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/hub"
	"github.com/agentscope/agentscope/internal/protocol"
	"github.com/agentscope/agentscope/internal/storage"
)

type Server struct {
	cfg        *config.Config
	hub        *hub.Hub
	builder    *agg.SnapshotBuilder
	mode       string
	modeReason string
	db         storage.Store
	http       *http.Server
	started    time.Time

	apiLimit  *limiter
	authLimit *limiter

	hbStop chan struct{}
	hbOnce sync.Once
}

func NewServer(cfg *config.Config, h *hub.Hub, builder *agg.SnapshotBuilder, mode, modeReason string, db storage.Store) *Server {
	if db == nil {
		db = storage.Nop{}
	}
	return &Server{
		cfg:        cfg,
		hub:        h,
		builder:    builder,
		mode:       mode,
		modeReason: modeReason,
		db:         db,
		started:    time.Now(),
		apiLimit:   newLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst),
		authLimit:  newLimiter(0.2, 10), // ~12 auth failures/min
		hbStop:     make(chan struct{}),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	get := allowMethods(http.MethodGet)
	post := allowMethods(http.MethodPost)

	mux.HandleFunc("/api/health", get(s.handleHealth))
	mux.HandleFunc("/healthz", get(s.handleBorgHealth))
	mux.HandleFunc("/livez", get(s.handleBorgHealth))
	mux.HandleFunc("/readyz", get(s.handleBorgHealth))
	mux.HandleFunc("/metrics", get(s.handlePrometheusMetrics))
	mux.HandleFunc("/api/v1/traces/perfetto", get(s.handlePerfettoTraces))
	mux.HandleFunc("/api/v1/logs", get(s.handleCloudLogging))
	mux.HandleFunc("/api/meta", get(s.handleMeta))
	mux.HandleFunc("/api/snapshot", get(s.handleSnapshot))
	mux.HandleFunc("/api/target", post(s.handleTarget))
	mux.HandleFunc("/api/history", get(s.handleHistory))
	mux.HandleFunc("/api/events/history", get(s.handleEventsHistory))
	mux.HandleFunc("/ws", get(s.handleWS))

	if dir := findStaticDir(s.cfg.StaticDir); dir != "" {
		log.Printf("serving UI from %s", dir)
		mux.Handle("/", spaHandler(dir))
	}

	return recoverMiddleware(s.middleware(mux))
}

// Start binds and serves until ctx is cancelled, then shuts down gracefully.
func (s *Server) Start(ctx context.Context) error {
	s.http = &http.Server{
		Addr:              s.cfg.HttpAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// Read/Write timeouts must stay zero: WebSocket connections share this
		// server and would otherwise be killed mid-stream.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}
	go s.heartbeat()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("http listen %s", s.cfg.HttpAddr)
		errCh <- s.http.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return s.Shutdown()
	case err := <-errCh:
		s.stopHeartbeat()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (s *Server) Shutdown() error {
	s.stopHeartbeat()
	if s.http == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
	defer cancel()
	return s.http.Shutdown(ctx)
}

func (s *Server) stopHeartbeat() {
	s.hbOnce.Do(func() { close(s.hbStop) })
}

func (s *Server) heartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.hbStop:
			return
		case <-ticker.C:
			b, err := protocol.Marshal(protocol.KindHeartbeat, s.mode, protocol.HeartbeatPayload{
				UptimeS: time.Since(s.started).Seconds(),
			})
			if err != nil {
				continue
			}
			s.hub.Broadcast(b)
		}
	}
}

func findStaticDir(explicit string) string {
	candidates := []string{explicit}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "frontend", "dist"),
			filepath.Join(exeDir, "dist"),
			filepath.Join(exeDir, "..", "frontend", "dist"),
		)
	}
	wd, _ := os.Getwd()
	candidates = append(candidates,
		filepath.Join(wd, "frontend", "dist"),
		filepath.Join(wd, "..", "frontend", "dist"),
		filepath.Join(wd, "dist"),
	)
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(c, "index.html")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func spaHandler(dir string) http.Handler {
	root := http.Dir(dir)
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/ws" {
			http.NotFound(w, r)
			return
		}
		// Prevent path-escape even though http.Dir already blocks "..".
		if strings.Contains(r.URL.Path, "..") {
			http.NotFound(w, r)
			return
		}
		p := r.URL.Path
		if p == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}
		f, err := root.Open(p)
		if err != nil {
			if os.IsNotExist(err) {
				http.ServeFile(w, r, filepath.Join(dir, "index.html"))
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
