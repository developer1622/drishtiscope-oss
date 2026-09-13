package config

import (
	"os"
	"testing"
	"time"
)

func TestSanitizeComm(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"agy", "agy"},
		{"  nginx ", "nginx"},
		{"payments-agent", "payments-agent"},
		{"a_very_long_process_name", "a_very_long_pro"},
		{"../etc/passwd", ""},
		{"agy\x00", "agy"},
		{"foo bar", "foo_bar"},
		{"", ""},
		{"node:16", "node:16"},
	}
	for _, tc := range tests {
		if got := SanitizeComm(tc.in); got != tc.want {
			t.Errorf("SanitizeComm(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidatePID(t *testing.T) {
	if err := ValidatePID(0); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePID(1); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePID(-1); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidatePID(MaxPID + 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestClampLimit(t *testing.T) {
	if ClampLimit(0, 100, 500) != 100 {
		t.Fatal("fallback")
	}
	if ClampLimit(50, 100, 500) != 50 {
		t.Fatal("passthrough")
	}
	if ClampLimit(9999, 100, 500) != 500 {
		t.Fatal("max")
	}
}

func TestSetTargetConcurrent(t *testing.T) {
	c := Load()
	if err := c.SetTarget(42, "nginx"); err != nil {
		t.Fatal(err)
	}
	pid, comm := c.Target()
	if pid != 42 || comm != "nginx" {
		t.Fatalf("got %d %q", pid, comm)
	}
	if err := c.SetTarget(0, "agy"); err != nil {
		t.Fatal(err)
	}
	if c.TargetPID() != 0 || c.TargetComm() != "agy" {
		t.Fatal("comm update")
	}
	if err := c.SetTarget(-3, "x"); err == nil {
		t.Fatal("bad pid")
	}
	if err := c.SetTarget(1, "../x"); err == nil {
		t.Fatal("bad comm")
	}
}

func TestValidateMode(t *testing.T) {
	c := Load()
	c.Mode = "nope"
	if err := c.Validate(); err == nil {
		t.Fatal("expected invalid mode")
	}
	c.Mode = "mock"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}

	c.HttpAddr = ""
	if err := c.Validate(); err == nil {
		t.Fatal("expected empty HttpAddr error")
	}
	c.HttpAddr = ":8080"

	c.SnapshotMs = 10
	if err := c.Validate(); err == nil {
		t.Fatal("expected snapshot-ms out of range error")
	}
	c.SnapshotMs = 400
}

func TestNormalizeEdgeCases(t *testing.T) {
	c := &Config{
		SnapshotMs:      10, // < 50
		MaxBodyBytes:    -1,
		MaxWSClients:    -1,
		MaxHistory:      -1,
		ShutdownTimeout: -1,
		RateLimitRPS:    -1,
		RateLimitBurst:  -1,
		targetPid:       -1,
		targetComm:      "bad/comm",
	}
	c.normalize()

	if c.SnapshotMs != DefaultSnapshotMs {
		t.Errorf("expected default snapshot ms, got %d", c.SnapshotMs)
	}
	if c.MaxBodyBytes != DefaultMaxBody {
		t.Errorf("expected default body bytes, got %d", c.MaxBodyBytes)
	}
	if c.MaxWSClients != DefaultMaxWS {
		t.Errorf("expected default ws clients, got %d", c.MaxWSClients)
	}
	if c.MaxHistory != DefaultMaxHistory {
		t.Errorf("expected default max history, got %d", c.MaxHistory)
	}
	if c.ShutdownTimeout != DefaultShutdownSec*time.Second {
		t.Errorf("expected default shutdown timeout, got %v", c.ShutdownTimeout)
	}
	if c.RateLimitRPS != 40 {
		t.Errorf("expected rate limit 40, got %v", c.RateLimitRPS)
	}
	if c.RateLimitBurst != 80 {
		t.Errorf("expected burst 80, got %d", c.RateLimitBurst)
	}
	if c.targetPid != 0 {
		t.Errorf("expected target pid 0, got %d", c.targetPid)
	}

	// Test upper boundaries
	c.SnapshotMs = 20_000
	c.MaxHistory = 10_000
	c.targetPid = MaxPID + 100
	c.normalize()
	if c.SnapshotMs != 10_000 {
		t.Errorf("expected capped 10_000, got %d", c.SnapshotMs)
	}
	if c.MaxHistory != DefaultMaxHistory {
		t.Errorf("expected default history, got %d", c.MaxHistory)
	}
	if c.targetPid != 0 {
		t.Errorf("expected reset pid, got %d", c.targetPid)
	}
}

func TestOriginAllowed(t *testing.T) {
	c := Load()
	c.AllowInsecureWS = false
	c.CORSOrigins = []string{"http://localhost:5173"}
	if !c.OriginAllowed("http://localhost:5173") {
		t.Fatal("allowlisted origin")
	}
	if c.OriginAllowed("http://evil.example") {
		t.Fatal("rejected origin")
	}
	c.AllowInsecureWS = true
	if c.OriginAllowed("http://evil.example") {
		t.Fatal("WS insecure must not widen CORS")
	}
}

func TestAuthTokenMinLength(t *testing.T) {
	c := Load()
	c.Mode = "mock"
	c.AuthToken = "short"
	if err := c.Validate(); err == nil {
		t.Fatal("short token")
	}
	c.AuthToken = "0123456789abcdef"
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestBindsAllInterfaces(t *testing.T) {
	c := Load()
	c.HttpAddr = ":8080"
	if !c.BindsAllInterfaces() {
		t.Fatal(":8080")
	}
	c.HttpAddr = "127.0.0.1:8080"
	if c.BindsAllInterfaces() {
		t.Fatal("loopback")
	}
	c.HttpAddr = "0.0.0.0:8080"
	if !c.BindsAllInterfaces() {
		t.Fatal("0.0.0.0:8080 should bind all interfaces")
	}
}

func TestWSOriginPatterns(t *testing.T) {
	c := Load()
	c.AllowInsecureWS = false
	p := c.WSOriginPatterns()
	if len(p) == 0 {
		t.Fatal("empty patterns")
	}
}

func TestEnvHelpers(t *testing.T) {
	os.Setenv("TEST_STRING_VAR", "hello")
	defer os.Unsetenv("TEST_STRING_VAR")
	if envOr("TEST_STRING_VAR", "def") != "hello" {
		t.Fatal("envOr failed")
	}
	if envOr("NON_EXISTENT_VAR_XYZ", "def") != "def" {
		t.Fatal("envOr fallback failed")
	}

	os.Setenv("TEST_INT_VAR", "42")
	defer os.Unsetenv("TEST_INT_VAR")
	if envInt("TEST_INT_VAR", 0) != 42 {
		t.Fatal("envInt failed")
	}
	os.Setenv("TEST_BAD_INT_VAR", "not-a-number")
	defer os.Unsetenv("TEST_BAD_INT_VAR")
	if envInt("TEST_BAD_INT_VAR", 99) != 99 {
		t.Fatal("envInt fallback failed")
	}

	os.Setenv("TEST_BOOL_VAR", "true")
	defer os.Unsetenv("TEST_BOOL_VAR")
	if !envBool("TEST_BOOL_VAR", false) {
		t.Fatal("envBool true failed")
	}
	os.Setenv("TEST_BOOL_1", "1")
	defer os.Unsetenv("TEST_BOOL_1")
	if !envBool("TEST_BOOL_1", false) {
		t.Fatal("envBool 1 failed")
	}
	if envBool("NON_EXISTENT_BOOL", true) != true {
		t.Fatal("envBool fallback failed")
	}

	origins := parseOrigins(" http://a.com, http://b.com , ")
	if len(origins) != 2 || origins[0] != "http://a.com" || origins[1] != "http://b.com" {
		t.Fatalf("parseOrigins failed: %v", origins)
	}

	// ParseFlags test
	c := Load()
	c.ParseFlags()
}
