package config

import (
	"fmt"
	"strings"
	"unicode"
)

// SanitizeComm trims and truncates a Linux comm prefix. Returns empty if the
// value contains disallowed runes (control chars / path separators).
func SanitizeComm(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\x00")
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '/' || r == '\\' || r == 0 || unicode.IsControl(r) {
			return ""
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' || r == ':' || r == '+' {
			b.WriteRune(r)
		} else if r == ' ' {
			// Linux comm can contain spaces; keep a single underscore instead
			// so BPF prefix matching stays well-defined.
			b.WriteByte('_')
		} else {
			return ""
		}
	}
	out := b.String()
	if len(out) > MaxCommLen {
		out = out[:MaxCommLen]
	}
	return out
}

func ValidatePID(pid int) error {
	if pid < 0 || pid > MaxPID {
		return fmt.Errorf("pid out of range (0..%d)", MaxPID)
	}
	return nil
}

func ClampLimit(n, fallback, max int) int {
	if n <= 0 {
		return fallback
	}
	if n > max {
		return max
	}
	return n
}
