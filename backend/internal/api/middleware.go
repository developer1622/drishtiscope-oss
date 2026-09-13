package api

import (
	"net/http"
	"strings"

	"github.com/agentscope/agentscope/internal/audit"
)

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, s.cfg.TrustProxy)

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("X-DNS-Prefetch-Control", "off")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; img-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/ws" {
			w.Header().Set("Cache-Control", "no-store")
		}

		origin := r.Header.Get("Origin")
		if origin != "" && s.cfg.OriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
		} else if origin != "" {
			audit.Event("cors_reject", ip, "deny", origin)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if s.cfg.MaxBodyBytes > 0 && r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes)
		}

		if strings.HasPrefix(r.URL.Path, "/api") || r.URL.Path == "/ws" {
			if !s.apiLimit.Allow(ip) {
				audit.Event("rate_limit", ip, "deny", r.Method+" "+r.URL.Path)
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
		}

		if !isPublicPath(r.URL.Path) {
			if s.cfg.HasAuth() {
				if !tokenOK(bearerToken(r), s.cfg.AuthToken) {
					if !s.authLimit.Allow(ip) {
						audit.Event("auth_lockout", ip, "deny", r.URL.Path)
						writeError(w, http.StatusTooManyRequests, "too many auth failures")
						return
					}
					audit.Event("auth_fail", ip, "deny", r.Method+" "+r.URL.Path)
					w.Header().Set("WWW-Authenticate", `Bearer realm="drishtiscope"`)
					writeError(w, http.StatusUnauthorized, "unauthorized")
					return
				}
			} else if !isLoopbackIP(ip) {
				// A01 deny-by-default: without AUTH_TOKEN the API is loopback-only.
				audit.Event("access_reject", ip, "deny", r.Method+" "+r.URL.Path)
				writeError(w, http.StatusForbidden, "set AUTH_TOKEN to expose this API")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				audit.Event("panic", clientIP(r, false), "error", "recovered")
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func allowMethods(methods ...string) func(http.HandlerFunc) http.HandlerFunc {
	allowed := make(map[string]struct{}, len(methods))
	for _, m := range methods {
		allowed[m] = struct{}{}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if _, ok := allowed[r.Method]; !ok {
				w.Header().Set("Allow", strings.Join(methods, ", "))
				writeError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			next(w, r)
		}
	}
}
