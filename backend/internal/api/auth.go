package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) >= 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	// WebSocket browsers cannot set Authorization; query is accepted then
	// stripped from logs (see audit.StripSecrets).
	return r.URL.Query().Get("access_token")
}

func tokenOK(got, want string) bool {
	if want == "" {
		return true
	}
	if len(got) != len(want) {
		// still compare to keep timing flatter
		subtle.ConstantTimeCompare([]byte(want), []byte(want))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func isPublicPath(path string) bool {
	return path == "/api/health"
}
