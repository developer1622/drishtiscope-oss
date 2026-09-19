# DrishtiScope Architecture Blueprint

This document details the architectural design, kernel interaction model, time-series storage, WebSocket protocol, and frontend data pipeline of **DrishtiScope (दृष्टिScope)**.

> [!NOTE]
> **Universal Process Observability**: DrishtiScope is engineered to observe **any process** executing on the host operating system—including backend web services, background workers, compilers, developer CLI utilities, and AI agent runtimes.

---

## 1. High-Level System Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│                             Host Operating System                          │
│                                                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                     Target Process                                   │  │
│  │       (e.g., node, python, go-service, compiler, worker, app)        │  │
│  │                                                                      │  │
│  │   • execve() / vfork() subprocesses  • Network sockets (TCP/TLS)     │  │
│  │   • read/write workspace files       • epoll_wait & futex loops      │  │
│  └──────────────────┬─────────────────────────────────┬─────────────────┘  │
│                     │                                 │                    │
│                     ▼                                 ▼                    │
│       [ Kernel Space Instrumentation ]       [ Userspace /proc Read ]      │
│       eBPF CO-RE Tracepoints (Privileged):   Real Mode (Zero Root):        │
│       • raw_syscalls:sys_enter/exit          • /proc/[pid]/stat (CPU)      │
│       • sched:sched_process_exec/exit        • /proc/[pid]/statm (RSS)     │
│       • BPF Ring Buffer (16 MB)              • /proc/[pid]/fd/* (Files)    │
│                                              • /proc/[pid]/net/tcp (Flows) │
│                     │                        • /proc/[pid]/io (Disk/Sys)   │
│                     │                        • /proc/[pid]/schedstat       │
│                     └────────────────┬───────┘                             │
│                                      ▼                                     │
│                     ┌─────────────────────────────────┐                    │
│                     │    DrishtiScope Go Daemon       │                    │
│                     │             (:8080)             │                    │
│                     ├─────────────────────────────────┤                    │
│                     │ • Ingest & Inode Cross-Matcher  │                    │
│                     │ • Sliding Window Rate Counters  │                    │
│                     │ • Rolling Series Ring Buffers   │                    │
│                     │ • Asynchronous SQLite WAL Store │                    │
│                     │ • WebSocket Broadcast Hub       │                    │
│                     │ • Local Rule Diagnostic Engine  │                    │
│                     └────────────────┬────────────────┘                    │
│                                      │                                     │
└──────────────────────────────────────┼─────────────────────────────────────┘
                                       │
                    WebSocket & REST   │ HTTP (:8080 / :5173)
                                       ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                          DrishtiScope Web UI                               │
│                                                                            │
│   • Header: Unified Omnibox, Stream Controls (Live/Paused), Theme Selector │
│   • Ribbon: 8 KPI Vitals with SVG Sparklines & (?) Help Glyphs             │
│   • Tab 1: Process Story (Chronological Timeline & Operational Status)     │
│   • Tab 2: Overview (Golden Signals: Latency Quantiles, Traffic Waveform)  │
│   • Tab 3: Execution & CPU (Continuous Flamegraph, Perfetto Export)         │
│   • Tab 4: System Metrics (Metrics Query Language MQL, Subsystem Donuts)   │
│   • Tab 5: Security & Logs (Workload Profile Radar, Denial Audit, Logs)     │
│   • Dynamic Graph Scratchpad & Linux Playground                            │
│   • Observability Copilot Chat Drawer                                      │
│   • Sticky Footer: Event Rate, Ingest Counters, Motto & Health Signals     │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Ingestion Pipeline & Dual-Engine Fallback

DrishtiScope uses a dual-engine architecture to operate seamlessly across both enterprise servers and developer laptops:

### Kernel Tracepoint Engine (`mode=ebpf`)
When executed with `CAP_BPF` or `root`:
- Program compiled via Clang/LLVM with BPF CO-RE (`agent.bpf.c`).
- Maps:
  - `events`: `BPF_MAP_TYPE_RINGBUF` (16MB capacity).
  - `start_times`: `BPF_MAP_TYPE_HASH` for nanosecond system call latency measurement.
  - `config`: `BPF_MAP_TYPE_ARRAY` storing target TGID filters.
- Probe points:
  - `tracepoint/raw_syscalls/sys_enter`: Records start timestamp into `start_times`.
  - `tracepoint/raw_syscalls/sys_exit`: Calculates latency, samples every Nth event to prevent ringbuffer overflows.
  - `tracepoint/sched/sched_process_exec`: Tracks child subprocess execution.
  - `tracepoint/sched/sched_process_exit`: Tracks process termination.

### Userspace Telemetry Engine (`mode=real`)
When executed in unprivileged environments (e.g. standard user on WSL2, Docker without `--privileged`):
- Operates with zero root privileges by reading `/proc`:
  - CPU calculation: Measures delta between consecutive samples of `utime + stime` divided by sample duration in seconds and `ClockTicks()` (USER_HZ, 100).
  - Memory: Multiplies page counts in `/proc/[pid]/statm` by system page size (4096 bytes).
  - File Descriptors: Evaluates symlinks in `/proc/[pid]/fd/` and eliminates `/dev/null` or pseudo-terminals to highlight actual code files, databases, and configuration paths.
  - Network Sockets: Decodes hex socket entries in `/proc/[pid]/net/tcp` and `/proc/[pid]/net/tcp6`, matching socket inodes from open file descriptors.
  - Disk I/O & Syscalls: Reads byte counters and syscall invocations (`syscr`, `syscw`) from `/proc/[pid]/io`.
  - SRE Scheduler Delay: Derives runqueue latency from `/proc/[pid]/schedstat`.

---

## 3. Wire Protocol (Envelope Format)

Every WebSocket message and snapshot response conforms to the Version 1 Envelope format:

```json
{
  "v": 1,
  "kind": "snapshot",
  "ts": "2026-09-13T16:05:20Z",
  "mode": "real",
  "payload": {
    "meta": {
      "dropped_events": 0,
      "event_rate": 14.5,
      "uptime_s": 285.4,
      "target": { "pid": 1234, "comm": "my-service" }
    },
    "kpis": {
      "cpu_pct": 34.66,
      "threads": 18,
      "rss_bytes": 731238400,
      "open_fds": 80,
      "net_bps_tx": 32907,
      "net_bps_rx": 29869,
      "disk_bps_r": 0,
      "disk_bps_w": 0,
      "syscalls_per_sec": 14.5,
      "err_syscalls_per_sec": 0.174,
      "connects_per_sec": 7
    },
    "sre": {
      "latency_p50_us": 98.4,
      "latency_p90_us": 225.8,
      "latency_p99_us": 465.1,
      "slo_availability": 99.98,
      "error_budget_pct": 94.2,
      "burn_rate": 0.08,
      "runqueue_latency_us": 136.7,
      "saturation_pct": 34.9
    },
    "processes": [...],
    "files_top": [...],
    "flows": [...],
    "cpu_series": [...],
    "io_series": [...],
    "net_series": [...],
    "timeline": [...]
  }
}
```

---

## 4. Time-Series Storage (Embedded SQLite WAL)

- **Database Engine**: Modern SQLite with Write-Ahead Logging (`PRAGMA journal_mode=WAL;`).
- **Asynchronous Decoupling**: Snapshots and events are enqueued into a buffered channel (`asyncStore`). An asynchronous writer goroutine flushes records in batches to disk, isolating real-time WebSocket broadcast loops from disk latency spikes.
- **Query APIs**:
  - `/api/history?limit=100`: Retrieves chronological series points for CPU, memory, and I/O.
  - `/api/events/history?limit=50`: Retrieves recent audit events, file touches, and socket connections.

---

## 5. Security & Network Isolation

1. **Loopback Only**: By default, DrishtiScope binds strictly to `127.0.0.1:8080`.
2. **Bearer Authentication**: When binding to public interfaces, `AUTH_TOKEN` is mandatory. Unauthenticated requests receive HTTP 401/403.
3. **Payload Sanitization**: DrishtiScope adheres to a strict zero-payload principle. It does not inspect TLS contents, prompt text, or completion tokens.
