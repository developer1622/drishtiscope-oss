package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultHTTPAddr    = ":8080"
	DefaultComm        = "agy"
	DefaultMode        = "auto"
	DefaultSnapshotMs  = 1500
	DefaultDBPath      = "drishtiscope.db"
	MaxCommLen         = 15 // TASK_COMM_LEN-1
	MaxPID             = 4_194_304
	DefaultMaxBody     = 64 << 10
	DefaultMaxWS       = 128
	DefaultMaxHistory  = 500
	DefaultShutdownSec = 10
)

// Config is process-wide settings. Target PID/comm are mutex-protected because
// the HTTP API can change them while the eBPF/mock loops read them.
type Config struct {
	mu sync.RWMutex

	HttpAddr   string
	Mode       string
	SnapshotMs int
	StaticDir  string
	DBPath     string

	CORSOrigins     []string
	AllowInsecureWS bool
	MaxBodyBytes    int64
	MaxWSClients    int
	MaxHistory      int
	ShutdownTimeout time.Duration

	// AuthToken, when set, is required as Bearer (or WS access_token) for
	// every route except GET /api/health. Never log this value.
	AuthToken      string
	TrustProxy     bool
	RateLimitRPS   float64
	RateLimitBurst int

	targetPid  int
	targetComm string
}

func Load() *Config {
	cfg := &Config{
		HttpAddr:        envOr("HTTP_ADDR", DefaultHTTPAddr),
		Mode:            strings.ToLower(envOr("MODE", DefaultMode)),
		SnapshotMs:      envInt("SNAPSHOT_MS", DefaultSnapshotMs),
		targetPid:       envInt("TARGET_PID", 0),
		targetComm:      envOr("TARGET_COMM", DefaultComm),
		StaticDir:       envOr("STATIC_DIR", ""),
		DBPath:          envOr("DB_PATH", DefaultDBPath),
		CORSOrigins:     parseOrigins(envOr("CORS_ORIGINS", defaultOrigins())),
		AllowInsecureWS: envBool("WS_INSECURE", false),
		MaxBodyBytes:    int64(envInt("MAX_BODY_BYTES", DefaultMaxBody)),
		MaxWSClients:    envInt("MAX_WS_CLIENTS", DefaultMaxWS),
		MaxHistory:      envInt("MAX_HISTORY", DefaultMaxHistory),
		ShutdownTimeout: time.Duration(envInt("SHUTDOWN_TIMEOUT_SEC", DefaultShutdownSec)) * time.Second,
		AuthToken:       os.Getenv("AUTH_TOKEN"),
		TrustProxy:      envBool("TRUST_PROXY", false),
		RateLimitRPS:    float64(envInt("RATE_LIMIT_RPS", 40)),
		RateLimitBurst:  envInt("RATE_LIMIT_BURST", 80),
	}
	cfg.normalize()
	return cfg
}

func defaultOrigins() string {
	return strings.Join([]string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
	}, ",")
}

func (c *Config) normalize() {
	if c.SnapshotMs < 50 {
		c.SnapshotMs = DefaultSnapshotMs
	}
	if c.SnapshotMs > 10_000 {
		c.SnapshotMs = 10_000
	}
	if c.MaxBodyBytes <= 0 {
		c.MaxBodyBytes = DefaultMaxBody
	}
	if c.MaxWSClients <= 0 {
		c.MaxWSClients = DefaultMaxWS
	}
	if c.MaxHistory <= 0 || c.MaxHistory > 5000 {
		c.MaxHistory = DefaultMaxHistory
	}
	if c.ShutdownTimeout <= 0 {
		c.ShutdownTimeout = DefaultShutdownSec * time.Second
	}
	if c.RateLimitRPS <= 0 {
		c.RateLimitRPS = 40
	}
	if c.RateLimitBurst <= 0 {
		c.RateLimitBurst = 80
	}
	c.targetComm = SanitizeComm(c.targetComm)
	if c.targetPid < 0 || c.targetPid > MaxPID {
		c.targetPid = 0
	}
}

// Validate returns a user-facing error if the config cannot be used.
func (c *Config) Validate() error {
	switch c.Mode {
	case "auto", "ebpf", "real", "mock":
	default:
		return fmt.Errorf("invalid mode %q (want auto|ebpf|real|mock)", c.Mode)
	}
	if c.HttpAddr == "" {
		return fmt.Errorf("http listen address is empty")
	}
	if c.SnapshotMs < 50 || c.SnapshotMs > 10_000 {
		return fmt.Errorf("snapshot-ms out of range")
	}
	if c.AuthToken != "" && len(c.AuthToken) < 16 {
		return fmt.Errorf("AUTH_TOKEN must be at least 16 characters")
	}
	return nil
}

var flagsOnce sync.Once

// ParseFlags overlays CLI flags on environment defaults. Call once from main.
func (c *Config) ParseFlags() {
	flagsOnce.Do(func() {
		pid := c.targetPid
		comm := c.targetComm
		insecure := c.AllowInsecureWS
		flag.StringVar(&c.HttpAddr, "addr", c.HttpAddr, "HTTP/WS listen address")
		flag.StringVar(&c.Mode, "mode", c.Mode, "auto | ebpf | real | mock")
		flag.StringVar(&comm, "comm", comm, "process comm prefix to watch")
		flag.IntVar(&pid, "pid", pid, "process pid to watch (0 = match by comm)")
		flag.IntVar(&c.SnapshotMs, "snapshot-ms", c.SnapshotMs, "snapshot interval in milliseconds")
		flag.StringVar(&c.StaticDir, "static", c.StaticDir, "directory of built React UI (optional)")
		flag.StringVar(&c.DBPath, "db", c.DBPath, "path to SQLite TSDB file")
		flag.BoolVar(&insecure, "ws-insecure", insecure, "skip WebSocket origin checks (dev only)")
		flag.Parse()
		c.AllowInsecureWS = insecure
		c.Mode = strings.ToLower(c.Mode)
		_ = c.SetTarget(pid, comm)
		c.normalize()
	})
}

func (c *Config) TargetPID() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.targetPid
}

func (c *Config) TargetComm() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.targetComm
}

func (c *Config) Target() (pid int, comm string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.targetPid, c.targetComm
}

// SetTarget validates and installs a new filter. Empty comm keeps the previous comm
// when only pid is being set; pid 0 means "match by comm".
func (c *Config) SetTarget(pid int, comm string) error {
	if err := ValidatePID(pid); err != nil {
		return err
	}
	if comm != "" {
		comm = SanitizeComm(comm)
		if comm == "" {
			return fmt.Errorf("invalid process comm")
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.targetPid = pid
	if comm != "" {
		c.targetComm = comm
	}
	return nil
}

func (c *Config) SnapshotInterval() time.Duration {
	return time.Duration(c.SnapshotMs) * time.Millisecond
}

func (c *Config) HasAuth() bool {
	return c.AuthToken != ""
}

// BindsAllInterfaces reports whether HttpAddr listens beyond loopback.
func (c *Config) BindsAllInterfaces() bool {
	addr := c.HttpAddr
	if strings.HasPrefix(addr, ":") || strings.HasPrefix(addr, "0.0.0.0:") || strings.HasPrefix(addr, "[::]:") {
		return true
	}
	return false
}

func (c *Config) OriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	for _, o := range c.CORSOrigins {
		if o == "*" || strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}

func (c *Config) WSOriginPatterns() []string {
	if c.AllowInsecureWS {
		return []string{"*"}
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add("localhost:*")
	add("127.0.0.1:*")
	for _, o := range c.CORSOrigins {
		if o == "*" {
			add("*")
			continue
		}
		o = strings.TrimPrefix(o, "https://")
		o = strings.TrimPrefix(o, "http://")
		if o != "" {
			add(o)
		}
	}
	return out
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
