# OWASP Top 10:2025 controls

This file maps DrishtiScope to the [OWASP Top 10:2025](https://owasp.org/Top10/).
The HTTP API is a **local observability surface**. It is deny-by-default on the network.

| ID | Risk | What we do |
|----|------|------------|
| **A01** Broken Access Control (incl. SSRF) | Unauthenticated remote use of `/api/*` and `/ws` | Without `AUTH_TOKEN`, every route except `GET /api/health` is **loopback-only**. Mutations from non-loopback IPs are 403. `X-Forwarded-For` is ignored unless `TRUST_PROXY=1`. No user-controlled outbound fetch (no SSRF surface). |
| **A02** Security Misconfiguration | Open CORS, verbose errors, default bind | CORS allow-list (never widened by `--ws-insecure`). Security headers (CSP, frame deny, nosniff, Permissions-Policy, COOP). JSON errors do not leak SQL. Startup warns if bound to all interfaces without a token. |
| **A03** Software Supply Chain Failures | Compromised deps | `go.sum` + `package-lock.json` committed. `make vulncheck` runs `govulncheck`. |
| **A04** Cryptographic Failures | Token leakage, world-readable DB | Bearer compared with `crypto/subtle`. Tokens never logged (`access_token` redacted). SQLite file mode `0600`. HSTS set when TLS is terminated on the process. |
| **A05** Injection | SQLi, XSS, path, comm injection | Parameterized SQL. `DisallowUnknownFields` on JSON. Comm/pid sanitization. SPA rejects `..`. React does not use `dangerouslySetInnerHTML`. POST requires `application/json`. |
| **A06** Insecure Design | Abuse of a privileged local agent | Rate limits per IP. Max WS clients. Body size cap. Fail closed: short `AUTH_TOKEN` is a config error. |
| **A07** Authentication Failures | Missing/brute-force auth | Optional `AUTH_TOKEN` (min 16 chars). Constant-time compare. Auth failures audited and rate-limited. No default password. |
| **A08** Software or Data Integrity Failures | Tampered input | Unknown JSON fields rejected. BPF object is embedded at build time, not downloaded. |
| **A09** Security Logging & Alerting Failures | Silent attacks | `audit` lines for auth fail, lockout, CORS reject, rate limit, target change, panic. Secrets stripped. |
| **A10** Mishandling of Exceptional Conditions | Fail-open, leaked panics | Panic recovery returns generic 500. Auth/config errors fail closed. Timeouts and graceful shutdown. |

## Enable authentication

```bash
export AUTH_TOKEN="$(openssl rand -hex 16)"
export VITE_API_TOKEN="$AUTH_TOKEN"   # frontend
./drishtiscope --addr=127.0.0.1:8080
```

Clients send `Authorization: Bearer <token>`. Browsers on WebSocket use `?access_token=` (redacted in logs).
