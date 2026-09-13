package audit

import "testing"

func TestStripSecrets(t *testing.T) {
	got := StripSecrets("ws://localhost:8080/ws?access_token=supersecret&x=1")
	if got == "" || contains(got, "supersecret") {
		t.Fatalf("leaked: %s", got)
	}
	if !contains(got, "REDACTED") {
		t.Fatalf("expected redaction: %s", got)
	}
}

func TestSanitizeNewlines(t *testing.T) {
	if contains(sanitize("a\nb"), "\n") {
		t.Fatal("newline")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
