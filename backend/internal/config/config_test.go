package config

import (
	"testing"
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
}

func TestWSOriginPatterns(t *testing.T) {
	c := Load()
	c.AllowInsecureWS = false
	p := c.WSOriginPatterns()
	if len(p) == 0 {
		t.Fatal("empty patterns")
	}
}
