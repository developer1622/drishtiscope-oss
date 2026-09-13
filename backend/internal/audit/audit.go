// Package audit emits security-relevant events without secrets or tokens.
package audit

import (
	"log"
	"net/url"
	"strings"
)

// Event writes a structured audit line. detail must not contain secrets.
func Event(action, ip, result, detail string) {
	log.Printf("audit action=%s ip=%s result=%s detail=%s",
		sanitize(action), sanitize(ip), sanitize(result), sanitize(detail))
}

// StripSecrets removes credential material from URLs before logging.
func StripSecrets(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[unparseable]"
	}
	q := u.Query()
	for _, k := range []string{"access_token", "token", "auth", "password", "api_key"} {
		if q.Has(k) {
			q.Set(k, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	u.User = nil
	return u.String()
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
