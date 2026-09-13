package audit

import (
	"strings"
	"testing"
)

func TestStripSecrets(t *testing.T) {
	got := StripSecrets("ws://localhost:8080/ws?access_token=supersecret&x=1")
	if got == "" || strings.Contains(got, "supersecret") {
		t.Fatalf("leaked: %s", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatalf("expected redaction: %s", got)
	}

	// Test unparseable url
	unparseable := StripSecrets("http://[fe80::1%25eth0-broken")
	if unparseable != "[unparseable]" && !strings.Contains(unparseable, "http") {
		t.Fatalf("unexpected unparseable handling: %s", unparseable)
	}
}

func TestSanitizeNewlinesAndLength(t *testing.T) {
	if strings.Contains(sanitize("a\r\nb"), "\n") || strings.Contains(sanitize("a\r\nb"), "\r") {
		t.Fatal("newline or carriage return found")
	}

	longStr := strings.Repeat("x", 250)
	sanitized := sanitize(longStr)
	if len(sanitized) > 205 || !strings.HasSuffix(sanitized, "…") {
		t.Fatalf("expected truncated string, got len %d", len(sanitized))
	}
}

func TestEvent(t *testing.T) {
	Event("login", "127.0.0.1", "ok", "user logged in")
	Event("overflow\n", "10.0.0.1\r", "fail", strings.Repeat("y", 250))
}
