# Frequently Asked Questions (FAQ) — DrishtiScope

This document answers the 15 most critical technical, architectural, and operational questions about **DrishtiScope (दृष्टिScope)**, the real-time, kernel-grounded observability console for autonomous AI agents and LLM runtimes.

---

### 1. What makes DrishtiScope fundamentally different from application-level LLM tools like LangSmith, Langfuse, or Phoenix?

**Application tracers observe what the agent *said* it did; DrishtiScope observes what the host process *actually did*.**

Tools like LangSmith, Langfuse, and Arize Phoenix operate at **Layer A (Application / SDK)**. They require you to instrument the code (`@observe()`), wrap your OpenAI/Anthropic client, or route requests through a reverse proxy. They track prompts, completions, tokens, and span waterfalls.

However, modern autonomous coding agents (`codex`, Claude Code, `agy`, Copilot, Grok, Ollama) do not behave like simple REST microservices. They:
- Rapidly spawn subprocesses (`bash`, `git`, `docker`, `cargo`, `python`) via `execve`/`vfork` bursts.
- Stream tokens over long-lived TLS sockets, where stalled network frames cause perceived "infinite thinking".
- Read and rewrite hundreds of workspace files, touch SQLite databases, and manage lock files.
- Consume CPU and memory on the host, occasionally leaking file descriptors or getting blocked on kernel scheduling.

DrishtiScope operates at **Layer C (System-Level Agent Observability)**. It hooks into the Linux kernel (via eBPF tracepoints and live `/proc` scanning) to observe the host effects of closed-source binaries without requiring code modification, SDKs, or cloud proxies.

---

### 2. How does DrishtiScope observe closed-source AI agents (`codex`, Claude Code, Copilot, `agy`) without code changes?

DrishtiScope uses external kernel-level and operating system instrumentation:
1. **Target Discovery**: DrishtiScope inspects `/proc` to match process names (`comm`) or command-line strings (`cmdline`). For instance, targeting `agy`, `codex`, or `copilot` automatically identifies the primary process identifier (PID) and tracks its entire descendant process tree (`ppid` hierarchy).
2. **Process Accounting**: It reads `/proc/[pid]/stat`, `/proc/[pid]/status`, and `/proc/[pid]/statm` to track scheduling ticks, thread lifecycles, memory mappings (VMS/RSS), and context switches.
3. **I/O & Socket Resolution**: It reads `/proc/[pid]/io` for physical read/write throughput and syscall counts (`syscr`/`syscw`). It inspects `/proc/[pid]/fd/*` symlinks and cross-references socket inodes against `/proc/[pid]/net/tcp` and `tcp6` to display active network flows and endpoints.
4. **Kernel Tracepoints (when privileged)**: When kernel capabilities are present, eBPF programs attach to `raw_syscalls:sys_enter`, `raw_syscalls:sys_exit`, and `sched:sched_process_exec` to sample syscall latency with sub-microsecond precision.

---

### 3. What is the difference between eBPF mode and Real `/proc` mode? Does DrishtiScope require `root` or `sudo`?

**No, root or sudo is not required for daily development.** DrishtiScope features a resilient **Zero-Root Real Mode**:

- **Real Mode (`mode=real`)**: Designed for locked-down laptops, enterprise developer workstations, and standard WSL2 environments without root privileges. DrishtiScope polls `/proc/[pid]/*` and host sysfs interfaces at configurable intervals (default 1.5s or 400ms). It computes exact delta ticks for CPU%, physical I/O bytes, active open files, thread counts, and network flows without requiring any elevated permissions or root privileges.
- **eBPF Mode (`mode=ebpf`)**: Activated when run with `CAP_BPF` / `CAP_SYS_ADMIN` or as `root`. Loads a dual-licensed eBPF program into kernel space to capture raw kernel tracepoints, ring buffer events, and sub-microsecond syscall latency distributions.
- **Auto Mode (`mode=auto`, default)**: Attempts to attach eBPF probes first. If the kernel restricts unprivileged BPF loading (e.g., `unprivileged_bpf_disabled=2`), it seamlessly engages Real Mode with zero interruption and zero synthetic fallback.

---

### 4. Does DrishtiScope intercept or store prompts, completions, or LLM tokens? What is the privacy posture?

**No. By design, DrishtiScope does not sniff TLS payloads or store prompt text.**

Unlike tools that perform TLS decryption (uprobes on `SSL_write`/`SSL_read`), DrishtiScope strictly observes **host effects and system metadata**:
- File paths accessed (e.g., `/home/user/project/main.go`, `database.db`).
- Socket connection metadata (remote IP, port 443, connection state, byte rates).
- Process execution metadata (command lines, parent/child relationships, exit codes).
- Resource telemetry (CPU%, RSS memory, scheduler latency).

**Privacy & Air-Gap Guarantee**: Telemetry is stored locally in an embedded SQLite database (`drishtiscope.db` in WAL mode). No telemetry, code snippets, or metrics are ever sent to an external SaaS, third party, or cloud service.

---

### 5. How does DrishtiScope compare to AgentSight? When should an engineering team use which?

[AgentSight](https://github.com/eunomia-bpf/agentsight) (developed by eunomia-bpf, arXiv:2508.02736) is DrishtiScope's closest peer in Layer C. However, they serve distinct operational needs:

| Dimension | AgentSight | DrishtiScope |
| :--- | :--- | :--- |
| **Primary Philosophy** | Boundary tracing: Captures prompt *intent* (via TLS interception) and kernel *effect*. | SRE Control Room: Focuses on process health, golden signals, and system bottlenecks. |
| **TLS Sniffing** | Decrypts and captures plaintext prompts and completions from HTTPS streams. | **Zero payload sniffing.** Tracks socket endpoints, states, and byte rates without reading text. |
| **Default Interface** | CLI-first (`top`, `record`, `report serve`). | Always-on interactive Web SPA with 5 real-time tabs, 15+ charts, and 5 themes. |
| **Privileges** | Typically requires `sudo` for SSL uprobe attachment and eBPF capture. | Dual-engine: eBPF when privileged, zero-root Real `/proc` mode on laptops and WSL2. |
| **Operational Surface** | Session replay and token flamegraphs. | SRE Golden Signals, P50/P90/P99 latency, Prometheus `/metrics`, Perfetto export, Metric Encyclopedia. |

**Recommendation**: Use AgentSight when you need forensic auditing of prompts sent to the model. Use DrishtiScope when operating agent processes in production, debugging stalls, monitoring resource consumption, and conducting SRE observability without violating data privacy policies.

---

### 6. What is the CPU, memory, and kernel overhead of running DrishtiScope?

DrishtiScope is engineered in Go 1.27+ and compiled to a lightweight, static native binary (~18 MB):
- **CPU Overhead**: In Real `/proc` mode with a 1.5s refresh rate, background CPU consumption is under **0.5%–1.2%** of a single core. In high-frequency eBPF mode, kernel tracepoint sampling introduces less than **1.5%** overhead.
- **Memory Footprint**: The Go daemon uses **18 MB to 35 MB RSS**.
- **I/O Impact**: The embedded SQLite database utilizes write-ahead logging (WAL mode) with passive background checkpoints, writing only consolidated state snapshots to disk without blocking the main event loop.

---

### 7. How does the Anti-Flicker mode and stream rate selector work, and why is it important for operations?

In live observability tools, high-frequency WebSocket streams (e.g. 200ms–400ms updates) cause visual elements, numbers, and graphs to constantly re-render, creating severe eye strain for engineers.

DrishtiScope provides a dedicated **Anti-Flicker System**:
1. **Refresh Rate Throttling**: The header dropdown offers preset stream intervals:
   - `500ms` (Rapid diagnostics)
   - `1s` (Balanced monitoring)
   - `2s` (Calm default)
   - `5s` (Relaxed long-term observation)
   - `Manual` (Pushes frozen until explicitly refreshed)
2. **Anti-Flicker Toggle (Smooth vs Rapid)**:
   - **Smooth Mode**: Dampens rapid numerical jitter, disables abrupt CSS pulse animations, and applies smooth transitions to charts and progress bars.
   - **Rapid Mode**: Delivers raw, instantaneous tick-by-tick kernel events for latency-sensitive debugging.

---

### 8. What are the 5 tabs in DrishtiScope, and what specific engineering question does each tab answer?

1. **Process Story (`Activity`)**: *"What has this agent done chronologically since it started?"*  
   Displays a timeline of file touches, socket connections, subprocess executions, and the real-time AI Executive Verdict (e.g., *Active Code Generation*, *High I/O & Socket Activity*, *Idle Event Loop*). Includes one-click copyable Linux diagnostics (`strace`, `pidstat`, `lsof`).
2. **Overview (`Vitals`)**: *"Is the agent healthy according to SRE Golden Signals?"*  
   Visualizes core vitals: Syscall latency quantiles (P50, P90, P99), traffic waveform, active threads, open file descriptors, and SLO error budget burn rate.
3. **Execution & CPU (`Call Trees`)**: *"Where is the agent spending its CPU cycles?"*  
   Presents continuous profiling flamegraphs, Go/C/Rust call trees, and single-click export to [ui.perfetto.dev](https://ui.perfetto.dev).
4. **System Metrics (`Telemetry`)**: *"How does the agent impact underlying host subsystems?"*  
   Includes a Metrics Query Language (MQL) console, subsystem breakdown donut charts (CPU, Memory, Disk, Network), and physical disk IOPS graphs.
5. **Security & Logs (`Audit`)**: *"Did the agent violate security boundaries or access sensitive files?"*  
   Displays an AI Workload Security Radar, permission denial audits (`EACCES`, `EPERM`), refused outbound connections, and structured runtime logs.

---

### 9. How do the 5 color themes work (Light, Dark, Ubuntu, Unix, Purple), and why is Light Mode the default?

DrishtiScope supports 5 themes controlled via CSS custom properties and instant theme switching:
- **☀️ Light Mode (Default)**: Clean, high-contrast palette with soft slate borders (`#f8fafc` background, `#0f172a` text). Optimized for bright daytime office environments and extended reading comfort without dark-mode eye fatigue.
- **🌙 Dark Mode**: Classic nocturnal control room palette (`#07080d` background, `#0e1118` panels, cyan/emerald accents).
- **🟠 Ubuntu Mode**: Canonical-inspired warm aubergine and deep purple tones (`#2c001e` background, `#dd4814` Ubuntu orange accents).
- **📟 Unix Mode**: Retro green-screen terminal aesthetic (`#0a0f0d` background, `#00ff66` glowing phosphor accents).
- **🔮 Purple Mode**: Cyber synthwave neon aesthetic (`#0d0b18` background, `#a855f7` vibrant violet accents).

Users can switch themes at any time using the header theme selector dropdown or the palette shortcut.

---

### 10. What is the Dynamic Graph Scratchpad and Linux Playground, and how do custom graphs get generated?

The **Dynamic Graph Scratchpad** (located at the bottom of the dashboard) allows engineers to dynamically instantiate custom visualizations on demand:
- **Built-in Visual Presets**: Single-click buttons to add specialized charts, such as:
  - *Syscall Latency Quantile Curve (P50 vs P90 vs P99)*
  - *Context Switches vs Runqueue Latency*
  - *Bidirectional Network Throughput (Tx vs Rx)*
  - *Thread Count vs RSS Memory Usage*
  - *Page Fault Dynamics (Minor vs Major Faults)*
- **Interactive AI Prompting**: Asking the AI Copilot to generate a specific graph (e.g., *"Show me thread count vs memory growth"*) dynamically injects and mounts the requested chart into the scratchpad.
- **Interactive Removal**: Each scratch graph can be inspected, compared, or closed independently.

---

### 11. How does the AI Observability Copilot work, and does it function without an external LLM API key?

**Yes, the Copilot works 100% locally out of the box without any API key.**

- **Local Deterministic Diagnostic Engine**: When no API key is configured, the Copilot executes a rule-based Linux diagnostic engine that evaluates the live snapshot (CPU saturation, open FD count, P99 syscall latency, error rates) and provides actionable CLI triage steps (`iotop`, `ss -tpe`, `pidstat`).
- **Cloud LLM Integration (Optional)**: If you provide an optional API key (`GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, or `GROQ_API_KEY`), the Copilot injects the exact, current host snapshot (KPIs, active files, open sockets, top syscalls) as structured context, enabling deep natural-language root-cause analysis (RCA). All credentials are read strictly from server environment variables at runtime—zero API keys are hardcoded, cached, or exposed to the client.

---

### 12. Can DrishtiScope export telemetry to standard observability stacks like Prometheus, Grafana, and Perfetto?

Yes. DrishtiScope provides enterprise-ready endpoints:
- **Prometheus (`/metrics`)**: Serves OpenMetrics-compliant gauges and counters for CPU, RSS, open file descriptors, network bytes, and syscall rates. Ready for scraping by Prometheus, VictoriaMetrics, or Datadog agents.
- **Health Probes (`/healthz`, `/livez`, `/readyz`)**: Standard Kubernetes/Borg probes returning HTTP 200 for container orchestrators.
- **Perfetto Traces (`/api/v1/traces/perfetto`)**: Emits Chrome/Perfetto Trace Event Format JSON. Can be dragged and dropped directly into [ui.perfetto.dev](https://ui.perfetto.dev) for nanosecond timeline inspection.
- **Structured Logs (`/api/v1/logs`)**: Delivers structured JSON log events for ingestion into Loki, Elasticsearch, or Cloud Logging.

---

### 13. Does DrishtiScope run on Windows Subsystem for Linux (WSL2), native Windows, or macOS?

**Yes. DrishtiScope is cross-platform with platform-native adaptations:**
- **Linux & WSL2**: Full primary support. In WSL2, DrishtiScope runs in Real Mode (`mode=real`), parsing `/proc` and host network stacks smoothly. In Linux with kernel headers and root privileges, it attaches directly to eBPF tracepoints (`mode=ebpf`).
- **Native Windows**: Compiles natively with Go (`drishtiscope.exe`). Seamlessly discovers real Windows processes using native Windows process scanning (`tasklist.exe` / Win32 process APIs) and interfaces with [Microsoft eBPF for Windows](WINDOWS_EBPF.md) (`ebpfcore.sys`). Serves both the WebSocket telemetry and the bundled React console on `http://localhost:8080`.
- **macOS (Darwin)**: Compiles natively with Go and provides developer demonstration telemetry and synthetic sampling for local UI testing.

---

### 14. How does DrishtiScope persist telemetry data, and how is SQLite time-series data retained?

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
