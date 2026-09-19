# Frequently Asked Questions (FAQ) — DrishtiScope

This document answers the 15 most common technical, architectural, and operational questions about **DrishtiScope (दृष्टिScope)**, the real-time, kernel-grounded observability console for inspecting any running process on Linux and Windows.

---

### 1. What makes system-level process observability different from application-level tracing libraries?

**Application tracers observe what the application chose to record; system-level observability observes what the host operating system kernel actually executed.**

Application-level tracing libraries operate inside the software application. They require developers to import an SDK, wrap client calls, or route requests through a proxy. They are exceptionally good for tracking business transactions, user workflows, and internal function calls.

However, many modern workloads—such as automated CLI utilities, compilers, background daemons, web servers, and autonomous agent runtimes—interact heavily with the underlying host system:
- Spawning child subprocesses (`bash`, `git`, `python`, `npm`) via `execve`/`vfork`.
- Opening long-lived network sockets where network latency or dropped packets can cause application stalls.
- Modifying hundreds of project files, touching local databases, and managing file locks.
- Consuming host CPU and memory, occasionally leaking file descriptors, or experiencing CPU scheduling delays.

DrishtiScope operates at the **System and Operating System Layer**. It observes the host-level effects of any process by reading live kernel interfaces (`/proc` and optional eBPF tracepoints) without requiring any code changes, SDK imports, or cloud proxies.

---

### 2. How does DrishtiScope observe any running process without code changes or SDKs?

DrishtiScope inspects the process from the outside using standard operating system interfaces:
1. **Process Discovery**: DrishtiScope inspects `/proc` (or Windows process tables) to match process names (`comm`) or command-line strings (`cmdline`). When a process is selected, DrishtiScope identifies its PID and tracks its child process tree.
2. **Resource Accounting**: It reads `/proc/[pid]/stat`, `/proc/[pid]/status`, and `/proc/[pid]/statm` to track scheduling ticks, thread lifecycles, memory mappings (VMS/RSS), and context switches.
3. **I/O & Socket Resolution**: It reads `/proc/[pid]/io` for physical read/write throughput and system call counters (`syscr`/`syscw`). It inspects `/proc/[pid]/fd/*` symlinks and cross-references socket inodes against `/proc/[pid]/net/tcp` to display active network flows and endpoints.
4. **Kernel Tracepoints (when privileged)**: When kernel capabilities are present, eBPF programs attach to `raw_syscalls:sys_enter`, `raw_syscalls:sys_exit`, and `sched:sched_process_exec` to sample system call latency with sub-microsecond precision.

---

### 3. What is the difference between eBPF mode and Real `/proc` mode? Does DrishtiScope require `root` or `sudo`?

**No, root or sudo is not required for daily development.** DrishtiScope features a resilient **Zero-Root Real Mode**:

- **Real Mode (`mode=real`)**: Designed for developer laptops, workstations, and standard WSL2 environments without root privileges. DrishtiScope polls `/proc/[pid]/*` and host sysfs interfaces at configurable intervals (default 1.5s or 400ms). It computes exact delta ticks for CPU%, physical I/O bytes, active open files, thread counts, and network flows without requiring elevated permissions.
- **eBPF Mode (`mode=ebpf`)**: Activated when run with `CAP_BPF` / `CAP_SYS_ADMIN` or as `root`. Loads a dual-licensed eBPF program into kernel space to capture raw kernel tracepoints, ring buffer events, and sub-microsecond system call latency distributions.
- **Auto Mode (`mode=auto`, default)**: Attempts to attach eBPF probes first. If the kernel restricts unprivileged BPF loading, it seamlessly falls back to Real Mode with zero interruption and zero synthetic data.

---

### 4. Does DrishtiScope intercept or store network payloads, prompt text, or user secrets?

**No. By design, DrishtiScope does not inspect network payloads or store sensitive contents.**

DrishtiScope strictly observes **system metadata and resource metrics**:
- File paths accessed (e.g., `/home/user/project/main.go`, `config.yaml`).
- Socket connection metadata (remote IP, port number, connection state, byte rates).
- Process execution metadata (command lines, parent/child relationships, exit codes).
- Resource telemetry (CPU%, RSS memory, scheduler latency).

**Privacy Guarantee**: Telemetry is stored strictly on your local machine in an embedded SQLite database (`drishtiscope.db` in WAL mode). No telemetry, source code, or metrics are ever sent to any external server or cloud service.

---

### 5. How does DrishtiScope handle payload privacy compared to network packet capture tools?

Network packet inspection tools capture raw bytes and decrypt packets to inspect communication contents. While this is helpful for forensic security auditing, it introduces significant data privacy, credential leakage, and compliance risks.

DrishtiScope deliberately takes an **operational SRE approach**:
- It tracks socket endpoints, connection states (`ESTABLISHED`), and transfer rates without decrypting or inspecting the underlying payload.
- This allows engineers to diagnose network stalls, socket leaks, and connection saturation safely, even on systems handling confidential data, without violating organizational privacy boundaries.

---

### 6. What is the CPU, memory, and kernel overhead of running DrishtiScope?

DrishtiScope is engineered in Go and compiled to a lightweight, static native binary (~18 MB):
- **CPU Overhead**: In Real `/proc` mode with a 1.5s refresh rate, background CPU consumption is under **0.5%–1.2%** of a single core. In high-frequency eBPF mode, kernel tracepoint sampling introduces less than **1.5%** overhead.
- **Memory Footprint**: The Go daemon uses **18 MB to 35 MB RSS**.
- **I/O Impact**: The embedded SQLite database utilizes write-ahead logging (WAL mode) with passive background checkpoints, writing consolidated state snapshots to disk without blocking the main event loop.

---

### 7. How does the Anti-Flicker mode and stream rate selector work?

In live observability tools, high-frequency WebSocket streams (e.g. 200ms–400ms updates) cause visual elements, numbers, and graphs to constantly re-render, creating eye strain.

DrishtiScope provides an **Anti-Flicker System**:
1. **Refresh Rate Throttling**: The header dropdown offers preset stream intervals:
   - `500ms` (Rapid diagnostics)
   - `1s` (Balanced monitoring)
   - `2s` (Calm default)
   - `5s` (Relaxed long-term observation)
   - `Manual` (Pushes frozen until explicitly refreshed)
2. **Anti-Flicker Toggle (Smooth vs Rapid)**:
   - **Smooth Mode**: Dampens rapid numerical jitter, disables abrupt CSS pulse animations, and applies smooth transitions to charts.
   - **Rapid Mode**: Delivers raw, instantaneous tick-by-tick kernel events for latency-sensitive debugging.

---

### 8. What are the 5 tabs in DrishtiScope, and what specific engineering question does each tab answer?

1. **Process Story (`Activity`)**: *"What has this process done chronologically since it started?"*  
   Displays a timeline of file touches, socket connections, subprocess executions, and operational status. Includes one-click copyable Linux diagnostics (`strace`, `pidstat`, `lsof`).
2. **Overview (`Vitals`)**: *"Is the process healthy according to SRE Golden Signals?"*  
   Visualizes core vitals: System call latency quantiles (P50, P90, P99), traffic waveform, active threads, open file descriptors, and SLO error budget burn rate.
3. **Execution & CPU (`Call Trees`)**: *"Where is the process spending its CPU cycles?"*  
   Presents continuous profiling flamegraphs, call trees, and single-click export to [ui.perfetto.dev](https://ui.perfetto.dev).
4. **System Metrics (`Telemetry`)**: *"How does the process impact underlying host subsystems?"*  
   Includes a Metrics Query Language (MQL) console, subsystem breakdown donut charts (CPU, Memory, Disk, Network), and physical disk IOPS graphs.
5. **Security & Logs (`Audit`)**: *"Did the process violate security boundaries or hit permission errors?"*  
   Displays a Workload Security Radar, permission denial audits (`EACCES`, `EPERM`), refused outbound connections, and structured runtime logs.

---

### 9. How do the 5 color themes work (Light, Dark, Ubuntu, Unix, Purple), and why is Light Mode the default?

DrishtiScope supports 5 themes controlled via CSS custom properties:
- **☀️ Light Mode (Default)**: Clean, high-contrast palette with soft slate borders (`#f8fafc` background, `#0f172a` text). Optimized for bright daytime office environments and extended reading comfort.
- **🌙 Dark Mode**: Classic nocturnal control room palette (`#07080d` background, `#0e1118` panels, cyan/emerald accents).
- **🟠 Ubuntu Mode**: Warm aubergine and deep purple tones (`#2c001e` background, `#dd4814` orange accents).
- **📟 Unix Mode**: Retro green-screen terminal aesthetic (`#0a0f0d` background, `#00ff66` phosphor accents).
- **🔮 Purple Mode**: Cyber synthwave aesthetic (`#0d0b18` background, `#a855f7` violet accents).

Users can switch themes at any time using the header theme selector dropdown.

---

### 10. What is the Dynamic Graph Scratchpad and Linux Playground?

The **Dynamic Graph Scratchpad** (located at the bottom of the dashboard) allows engineers to dynamically instantiate custom visualizations on demand:
- **Built-in Presets**: Single-click buttons to add specialized charts, such as:
  - *Syscall Latency Quantile Curve (P50 vs P90 vs P99)*
  - *Context Switches vs Runqueue Latency*
  - *Bidirectional Network Throughput (Tx vs Rx)*
  - *Thread Count vs RSS Memory Usage*
  - *Page Fault Dynamics (Minor vs Major Faults)*
- **Interactive AI Prompting**: Asking the Copilot to generate a specific graph dynamically mounts the requested chart into the scratchpad.
- **Independent Inspection**: Each scratch graph can be inspected, compared, or closed independently.

---

### 11. How does the AI Observability Copilot work, and does it function without an external API key?

**Yes, the Copilot works 100% locally out of the box without any API key.**

- **Local Deterministic Diagnostic Engine**: When no API key is configured, the Copilot executes a rule-based Linux diagnostic engine that evaluates the live snapshot (CPU saturation, open FD count, P99 syscall latency, error rates) and provides actionable CLI triage steps (`iotop`, `ss -tpe`, `pidstat`).
- **External LLM Integration (Optional)**: If you provide an optional API key (`GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, or `GROQ_API_KEY`), the Copilot injects the current host snapshot (KPIs, active files, open sockets, top syscalls) as structured context for natural-language analysis. All credentials are read strictly from server environment variables at runtime—zero API keys are hardcoded, cached, or exposed to the client.

---

### 12. Can DrishtiScope export telemetry to standard observability stacks like Prometheus and Perfetto?

Yes. DrishtiScope provides standard export endpoints:
- **Prometheus (`/metrics`)**: Serves OpenMetrics-compliant gauges and counters for CPU, RSS, open file descriptors, network bytes, and syscall rates. Ready for scraping by Prometheus or any compatible collector.
- **Health Probes (`/healthz`, `/livez`, `/readyz`)**: Standard HTTP 200 endpoints for container orchestrators.
- **Perfetto Traces (`/api/v1/traces/perfetto`)**: Emits Chrome/Perfetto Trace Event Format JSON for drag-and-drop inspection in [ui.perfetto.dev](https://ui.perfetto.dev).
- **Structured Logs (`/api/v1/logs`)**: Delivers structured JSON log events for ingestion into standard log aggregators.

---

### 13. Does DrishtiScope run on Windows Subsystem for Linux (WSL2), native Windows, or macOS?

**Yes. DrishtiScope is cross-platform with platform-native adaptations:**
- **Linux & WSL2**: Full primary support. In WSL2, DrishtiScope runs in Real Mode (`mode=real`), parsing `/proc` and host network stacks smoothly. In Linux with kernel headers and root privileges, it attaches directly to eBPF tracepoints (`mode=ebpf`).
- **Native Windows**: Compiles natively with Go (`drishtiscope.exe`). Discovers real Windows processes using native Windows process scanning (`tasklist.exe` / Win32 APIs) and interfaces with the open-source [Microsoft eBPF for Windows](WINDOWS_EBPF.md) driver (`ebpfcore.sys`).
- **macOS (Darwin)**: Compiles natively with Go and provides developer demonstration telemetry and synthetic sampling for local UI testing.

---

### 14. How does DrishtiScope persist telemetry data, and how is time-series data retained?

- **Storage Engine**: Telemetry is written to an embedded SQLite database (`drishtiscope.db`) with Write-Ahead Logging (`PRAGMA journal_mode=WAL;`).
- **Asynchronous Ingestion**: Ingestion occurs asynchronously through an in-memory channel buffer (capacity 256) to ensure disk writes never block real-time WebSocket broadcasting.
- **Retention & History**: Historical snapshots are queryable via `/api/history?limit=N`. Snapshots older than the configured retention threshold are automatically pruned during passive checkpoints.

---

### 15. How do I contribute new kernel tracepoints, system metrics, or visual chart presets to DrishtiScope?

We welcome community contributions!
- **Add a Kernel Tracepoint**: Modify [`backend/bpf/agent.bpf.c`](file:///home/ramum/agentscope/backend/bpf/agent.bpf.c), declare the BPF program with `SEC("tracepoint/...")`, and update [`backend/internal/ebpfagent/agent.go`](file:///home/ramum/agentscope/backend/internal/ebpfagent/agent.go).
- **Add a Metric to the Encyclopedia**: Add a new entry to [`frontend/src/data/metricDocs.ts`](file:///home/ramum/agentscope/frontend/src/data/metricDocs.ts) with plain-English definitions, healthy/warning thresholds, and CLI verification commands.
- **Add a Visual Chart Preset**: Define the new graph in [`frontend/src/store/useScopeStore.ts`](file:///home/ramum/agentscope/frontend/src/store/useScopeStore.ts) and add the preset button in [`frontend/src/components/GraphScratchpad.tsx`](file:///home/ramum/agentscope/frontend/src/components/GraphScratchpad.tsx).
- **Run Tests**: Ensure `go test -v ./...` passes in `backend/`, `npm run build` succeeds in `frontend/`, and all Playwright end-to-end tests pass (`npm run test:e2e`). Refer to [`CONTRIBUTING.md`](file:///home/ramum/agentscope/CONTRIBUTING.md) for full guidelines.
