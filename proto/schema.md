# WebSocket Protocol — DrishtiScope v1

All WebSocket messages are JSON objects with this envelope:

```json
{
  "v": 1,
  "kind": "hello" | "snapshot" | "event" | "heartbeat" | "error",
  "ts": "2026-09-12T19:00:00Z",
  "mode": "ebpf" | "real" | "mock",
  "payload": { ... }
}
```

---

## Message Kinds

### `hello`
Sent immediately when a client connects.

```json
{
  "v": 1,
  "kind": "hello",
  "ts": "...",
  "mode": "real",
  "payload": {
    "schema": 1,
    "hostname": "dev-box",
    "kernel": "6.6.87.2-microsoft-standard-WSL2",
    "target": { "pid": 1234, "comm": "my-service" }
  }
}
```

---

### `snapshot`
Sent every `SNAPSHOT_MS` ms (default 400ms). Contains full current state.

```json
{
  "kind": "snapshot",
  "payload": {
    "meta": {
      "dropped_events": 0,
      "event_rate": 142.5,
      "uptime_s": 37.2,
      "target": { "pid": 1234, "comm": "my-service" }
    },
    "processes": [ { "pid": 1234, "tgid": 1234, "ppid": 1001, "comm": "my-service",
      "cmdline": "/usr/bin/my-service --config /etc/service.yaml",
      "exe": "/usr/bin/my-service", "uid": 1000, "state": "S",
      "threads": 18, "cpu_pct": 12.4, "rss_bytes": 52428800, "vms_bytes": 314572800,
      "open_fds": 34, "ctx_switches": 4421, "start_time": "2026-09-12T18:00:00Z" } ],
    "kpis": {
      "cpu_pct": 12.4,
      "threads": 18,
      "rss_bytes": 52428800,
      "open_fds": 34,
      "net_bps_tx": 45678.0,
      "net_bps_rx": 12345.0,
      "disk_bps_r": 8192.0,
      "disk_bps_w": 2048.0,
      "syscalls_per_sec": 1420.5,
      "err_syscalls_per_sec": 0.1,
      "connects_per_sec": 0.03
    },
    "syscalls_top": [
      { "name": "epoll_wait", "count_s": 420.1, "errors_s": 0.0 },
      { "name": "futex",      "count_s": 380.2, "errors_s": 0.0 }
    ],
    "files_top": [
      { "path": "/var/lib/service/data.db", "ops_s": 12.3, "bytes_s": 8192.0, "errors": 0 }
    ],
    "flows": [
      { "src": "192.168.1.100", "dst": "10.0.0.10", "sport": 49201, "dport": 443,
        "proto": "tcp", "state": "ESTABLISHED", "bytes_tx": 12345, "bytes_rx": 67890,
        "pid": 1234, "comm": "my-service" }
    ],
    "cpu_series":  [ { "t": 1726175200000, "cpu_pct": 12.4 } ],
    "io_series":   [ { "t": 1726175200000, "r_bps": 8192, "w_bps": 2048 } ],
    "net_series":  [ { "t": 1726175200000, "tx_bps": 45678, "rx_bps": 12345 } ],
    "timeline":    [ { "id": "evt-001", "ts": "...", "severity": "info",
                       "category": "network", "pid": 1234, "comm": "my-service",
                       "title": "outbound connect", "detail": "10.0.0.10:443 via tcp",
                       "attrs": { "dst": "10.0.0.10", "dport": 443, "proto": "tcp" } } ]
  }
}
```

---

### `event`
Emitted immediately for interesting discrete events (connect, exec, error, sensitive file access).

```json
{
  "kind": "event",
  "payload": {
    "id": "evt-abc123",
    "ts": "2026-09-12T19:04:12Z",
    "severity": "warn",
    "category": "file",
    "pid": 1234,
    "comm": "my-service",
    "title": "sensitive file open denied",
    "detail": "/etc/shadow — EACCES",
    "attrs": { "path": "/etc/shadow", "flags": "O_RDONLY", "errno": 13 }
  }
}
```

Severity levels: `info` | `warn` | `crit`
Categories: `process` | `syscall` | `file` | `network` | `compute`

---

### `heartbeat`
Sent every 5 seconds when no snapshot has been sent recently.

```json
{ "kind": "heartbeat", "payload": { "uptime_s": 42.1 } }
```

---

### `error`
Server-side error notification.

```json
{ "kind": "error", "payload": { "message": "target pid 99999 not found" } }
```

---

## REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | `{ok:true, mode:"real", uptime_s:N, clients:N}` |
| GET | `/api/meta` | Same as hello payload |
| GET | `/api/snapshot` | Latest snapshot (for initial page load) |
| POST | `/api/target` | `{"pid":1234}` or `{"comm":"my-service"}` — updates filter |
| GET | `/ws` | WebSocket upgrade |
| GET | `/` | Serves React production build |

---

## Protocol Versioning

Field `"v": 1` is the schema version. Clients MUST ignore unknown `kind` values.
Future versions will increment `v` and announce in `hello.schema`.
