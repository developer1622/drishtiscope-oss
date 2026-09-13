# DrishtiScope Backend Daemon

The **DrishtiScope Backend** is a high-performance, single-binary Go daemon that extracts kernel and process telemetry from the Linux host, aggregates agent vitals, writes time-series snapshots to an embedded SQLite database, and broadcasts live telemetry over WebSockets.

---

## 🏛️ Architecture & Directory Structure

```
backend/
├── cmd/agentscope/
│   └── main.go                 # Process entry point, flag parsing, signal handling, fan-out
├── internal/
│   ├── realengine/
│   │   ├── engine.go           # 100% Real-world Linux /proc & host kernel telemetry collector
│   │   └── engine_test.go      # Unit tests for RealEngine
│   ├── ebpfagent/
│   │   ├── agent.go            # Cilium eBPF collection loader & ring-buffer consumer (Linux)
│   │   ├── agent_windows.go    # Experimental Microsoft eBPF for Windows stub
│   │   ├── agent_darwin.go     # macOS stub
│   │   └── agent_test.go       # eBPF tests
│   ├── enrich/
│   │   ├── proc.go             # /proc/[pid] parser: stat, statm, status, io, fd, net/tcp
│   │   └── syscallnames.go     # Linux x86_64 syscall number-to-name lookup table (0–450)
│   ├── agg/
│   │   ├── snapshot.go         # SnapshotBuilder, rolling series ring buffers (60 points)
│   │   └── rates.go            # Sliding window exponential rate counters (EWA 1s / 5s)
│   ├── api/
│   │   ├── server.go           # HTTP server routing, timeouts, middleware, SPA file server
│   │   ├── handlers.go         # REST endpoints: /api/health, /api/snapshot, /api/target, /ws
│   │   ├── chat.go             # AI Copilot handler with local rule engine & LLM provider dispatch
│   │   └── middleware.go       # CORS, loopback enforcement, rate limiter, security headers
│   ├── hub/
│   │   ├── hub.go              # WebSocket broadcast hub with slow-client disconnection
│   │   └── client.go           # nhooyr.io/websocket client handler and writePump
│   ├── storage/
│   │   ├── sqlite.go           # Embedded SQLite time-series storage with WAL mode
│   │   └── async.go            # Asynchronous buffered writer to prevent disk blocking
│   ├── protocol/
│   │   └── message.go          # Wire protocol definitions: Hello, Snapshot, Event, Envelope
│   ├── config/
│   │   └── config.go           # Configuration loader (env vars & CLI flags overlay)
│   └── mock/
│       ├── generator.go        # Synthetic demonstration generator (for offline dev/tests)
│       └── realproc.go         # Helper utilities for process candidate discovery
└── bpf/
    └── agent.bpf.c             # Dual-licensed (GPL-2.0/MIT) eBPF C program (CO-RE)
```

---

## ⚙️ The Dual-Engine Model

DrishtiScope incorporates two distinct collection engines to deliver maximum fidelity on any machine:

### 1. Real Engine (`mode=real`)
Designed to run with **zero root privileges** on developer laptops, workstations, and WSL2 environments.
- Scans `/proc` at each snapshot interval (default: 1.5s or 400ms).
- Calculates exact CPU% from `/proc/[pid]/stat` user and kernel time ticks.
- Reads resident memory (RSS) and virtual memory (VMS) from `/proc/[pid]/statm`.
- Resolves open file descriptors and physical paths via `/proc/[pid]/fd/*`.
- Parses network socket connections from `/proc/[pid]/net/tcp` and `/proc/[pid]/net/tcp6`.
- Calculates disk I/O rates and physical syscall throughput from `/proc/[pid]/io`.
- Computes runqueue scheduler latency from `/proc/[pid]/schedstat`.

### 2. eBPF Engine (`mode=ebpf`)
Engaged when running with `CAP_BPF` / `CAP_SYS_ADMIN` or as `root`.
- Attaches to kernel tracepoints:
  - `raw_syscalls/sys_enter` & `sys_exit`: Low-overhead syscall latency sampling.
  - `sched/sched_process_exec`: Process lifecycle and execution tracking.
  - `sched/sched_process_exit`: Immediate process termination capture.
- Emits binary event records through an in-kernel BPF Ring Buffer (`BPF_MAP_TYPE_RINGBUF`, 16MB) directly into the Go userspace handler.

### 3. Auto Mode (`mode=auto`, default)
Attempts eBPF probe attachment first. If the kernel denies unprivileged BPF loading, it automatically falls back to **Real Engine** (`mode=real`) without generating synthetic mock data.

---

## 📡 REST & WebSocket API Endpoints

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/api/health` | `GET` | Health check reporting mode (`real` or `ebpf`), uptime, client count, and target PID. |
| `/api/meta` | `GET` | System metadata: hostname, kernel version, schema version, and target process. |
| `/api/snapshot` | `GET` | The most recent complete snapshot of KPIs, processes, files, sockets, and timeline. |
| `/api/target` | `POST` | Dynamically updates the active target process by PID or comm name (`{"pid": 113971, "comm": "agy"}`). |
| `/api/history` | `GET` | Historical time-series points from the embedded SQLite database. |
| `/api/events/history` | `GET` | Recent security, lifecycle, and file events from SQLite storage. |
| `/api/v1/chat` | `POST` | AI Observability Copilot endpoint: evaluates live snapshots and returns diagnoses. |
| `/ws` | `GET` | WebSocket endpoint: upgrades connection and streams snapshots (400ms–2s) and events. |
| `/metrics` | `GET` | Prometheus OpenMetrics scraping endpoint. |
| `/healthz` | `GET` | Kubernetes liveness and readiness probe endpoint (returns `200 OK`). |
| `/api/v1/traces/perfetto` | `GET` | Chrome/Perfetto Trace Event Format JSON exporter for [ui.perfetto.dev](https://ui.perfetto.dev). |
| `/api/v1/logs` | `GET` | Structured `LogEntry[]` logs with severity levels. |

---

## 🛠️ Building & Running

### Prerequisites
- Go 1.22+
- Linux (x86_64 or aarch64), WSL2, macOS, or Windows

### Commands

```bash
# Run all tests with coverage
cd backend
go test -v -cover ./...

# Build the native binary
go build -ldflags="-s -w" -o drishtiscope ./cmd/agentscope

# Run in Real Mode targeting an autonomous agent
./drishtiscope -mode=real -comm=agy

# Run with eBPF privileges
sudo ./drishtiscope -mode=ebpf -comm=codex

# Serve prebuilt frontend dist directly
./drishtiscope -mode=real -static=../frontend/dist
```

---

## 🔒 Security & Loopback Defaults

- **OWASP Compliance**: By default, DrishtiScope listens only on local interfaces (`127.0.0.1:8080`). Remote interfaces require passing `AUTH_TOKEN` in the environment to enforce HTTP Bearer authentication and WebSocket token validation.
- **Payload Privacy**: Zero payload interception. No prompt text, tokens, or model response strings are stored or transmitted.
