# DrishtiScope (दृष्टिScope)

<div align="center">

<img src="docs/images/drishtiscope_logo.png" alt="DrishtiScope Logo" width="220" />

<br/>

> **Sanskrit: दृष्टि (Drishti — Insight, Clear Seeing, Perception) + English: Scope (Observatory Instrument)**  
> **Real-Time Kernel-Grounded Process Observability for Autonomous AI Agents & LLM Runtimes.**  
> *Process observability for coding agents and AI runtimes — without an SDK, without a proxy, without sending telemetry to the cloud.*

[![CI](https://github.com/drishtiscope/drishtiscope/actions/workflows/ci.yml/badge.svg)](https://github.com/drishtiscope/drishtiscope/actions/workflows/ci.yml)
[![Docker](https://github.com/drishtiscope/drishtiscope/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/drishtiscope/drishtiscope/actions/workflows/docker-publish.yml)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react)](https://react.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/badge/Single_Binary-~18_MB-blue.svg)](https://github.com/drishtiscope/drishtiscope/releases)
[![Dual-Engine](https://img.shields.io/badge/Engine-eBPF_%2B_Real_/proc-cyan.svg)](docs/ARCHITECTURE.md)

</div>

---

## 📹 DrishtiScope Showcase Video

Watch the **1080p 60fps Full Walkthrough** of DrishtiScope observing an active autonomous pair-programming agent (`agy` PID 113971), navigating the 5 tabs, testing the 5 themes, and exploring the Linux Metric Encyclopedia:

> 🎬 **Direct Video Downloads**:
> - **MP4 Video (H.264)**: [videos/drishtiscope_showcase.mp4](videos/drishtiscope_showcase.mp4) (6.5 MB, Full HD 1080p, 60fps)
> - **WebM Video (VP9)**: [videos/drishtiscope_showcase.webm](videos/drishtiscope_showcase.webm) (6.9 MB)

---

## 🧭 The Gap Nobody in LLMOps Owns

Almost every tool marketed as "AI agent observability" in 2026 watches **what the application logged**. That is a useful layer, but it is **not the same layer as what the process actually did**.

An autonomous coding agent (`codex`, Claude Code, `agy`, Copilot, Grok, Ollama) does not behave like a standard microservice:

| Agent Behavior | What Application Tracers See | What DrishtiScope Sees (Kernel + /proc) |
| :--- | :--- | :--- |
| Spawns `bash`, `git`, `docker`, `python` in rapid bursts | A "tool call" span, *if* the SDK logged it | Exact child PIDs, PPID tree, cmdline, open FDs |
| Streams tokens over long-lived TLS connections | Latency and token count, *if* client was wrapped | `ESTABLISHED` sockets, TX/RX throughput, P99 syscall latency |
| Rewrites workspace files, touches SQLite, updates WAL | Nothing, unless the tool span logged paths | Real `openat`/`write` descriptors, active file locks |
| Sits in `epoll_wait` / `futex` between turns | "Idle" only if the application reported it | Runqueue latency, CPU %, syscall mix, stall vs wait |
| Hits `EACCES` on `/etc/shadow` or refused connect | Typically nothing | Security audit events, error rate, error budget burn |

```
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER A — Application / SDK  ("what the agent said it did")             │
│  LangSmith · Langfuse · Phoenix · Braintrust · Helicone · AgentOps       │
│  Needs: SDK, decorator, gateway, or vendor plugin                        │
│  Misses: child processes, syscalls, /proc, real sockets, kernel stalls   │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER B — Generic kernel / eBPF APM  ("what the machine did")           │
│  Pixie · Coroot · Grafana Beyla · Groundcover · Tetragon · Falco         │
│  Needs: cluster / root / DaemonSet, not agent-aware                      │
│  Misses: agent identity, process story, single-binary laptop use         │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER C — System-level agent observability  ("what THIS agent did")     │
│  DrishtiScope  →  SRE control room · eBPF + /proc · golden signals       │
│  Needs: zero instrumentation, attach to live PID or comm                 │
│  Sees: host effects of closed-source agents with SRE semantics           │
└──────────────────────────────────────────────────────────────────────────┘
```

For the complete competitive analysis and positioning matrix against 40+ industry tools, see the [DrishtiScope USP Brief](usp.md).

---

## 📸 Visual Tour & Core Modules

DrishtiScope provides a high-density, intuitive SRE control room for autonomous AI coding agents and LLM runtimes. Below is an overview of each core module, how it works, and how to use the console:

### 1. Tab 1: Process Story (`Activity`) — Turn-by-Turn Agent Activity Chronology
![Tab 1: Process Story](screenshots/tab_story_desktop.png)

- **What it provides**: Chronological event feed capturing every tool execution, child process fork (`bash`, `git`, `python`, `npm`), file modification, and socket connect.
- **AI Executive Verdict**: Synthesizes microsecond kernel metrics into human-readable agent states (*Active Code Generation*, *Intense File I/O & Socket Activity*, *Waiting on LLM Token Stream*, or *Idle Event Loop*).
- **One-Click Linux Triage**: Instantly copy diagnostic commands pre-configured with the agent's PID (`strace -p <PID>`, `pidstat -p <PID> 1`, `lsof -p <PID>`).
- **How to use**: Click the **Process Story** tab or press `1` to follow what the agent is doing second-by-second.

### 2. Tab 2: Overview (`Vitals`) — Core SRE Golden Signals & Reliability
![Tab 2: Overview Vitals](screenshots/tab_basic_desktop.png)

- **What it provides**: Continuous tracking of the 4 SRE Golden Signals:
  - **Latency**: Microsecond-precision syscall latency quantiles (P50 Median, P90, and P99 Tail Latency).
  - **Traffic**: Live system call throughput (Syscalls/sec) and bidirectional network rate (Kbps / Mbps).
  - **Errors**: Non-zero negative kernel return rates (`EACCES`, `EPERM`, `ECONNREFUSED`).
  - **Saturation**: CPU utilization percentage, open file descriptors (FDs / limit), and CFS scheduler runqueue wait time.
- **Monarch Synchronized Waveform**: Real-time 60-second correlation comparing Syscall RPS against P99 latency spikes.
- **SLO Error Budget & Burn Rate**: Tracks 99.9% SLO compliance and 30-day burn rate acceleration to catch runaway agent loops before resource exhaustion.
- **How to use**: Click the **Overview** tab or press `2` for an instant high-level health assessment.

### 3. Tab 3: Execution & CPU (`Call Trees`) — Continuous Flamegraph & Process Tree
![Tab 3: Execution & CPU](screenshots/tab_medium_desktop.png)

- **What it provides**: Continuous on-CPU stack trace profiling and execution call tree breakdown across runtime engines (Python, Node.js, Go, Rust, C++).
- **Single-Click Perfetto Export**: Export Chrome/Perfetto Trace Event JSON to inspect nanosecond timeline slices directly in [ui.perfetto.dev](https://ui.perfetto.dev).
- **Live Process Hierarchy**: Displays active child processes, execution states (`R` Running, `S` Sleeping, `D` Disk Sleep, `Z` Zombie), thread counts, and memory footprints.
- **How to use**: Click the **Execution & CPU** tab or press `3` to isolate CPU hot spots and inspect process trees.

### 4. Tab 4: System Metrics (`Telemetry`) — MQL Console & Deep Telemetry
![Tab 4: System Metrics](screenshots/tab_advanced_desktop.png)

- **What it provides**: Metrics Query Language (MQL) interactive console for ad-hoc querying and slicing of live metrics.
- **Subsystem Breakdown Donut Charts**: Immediate visual breakdown of resource consumption across CPU cores, Resident Memory (RSS), Disk I/O, and Network.
- **Physical Disk IOPS & Latency**: Real-time storage read/write rates and physical IOPS curves.
- **How to use**: Click the **System Metrics** tab or press `4` to run telemetry queries and inspect low-level hardware counters.

### 5. Tab 5: Security & Logs (`Audit`) — Workload Profile Radar & Sandbox Audit
![Tab 5: Security & Logs](screenshots/tab_complete_desktop.png)

- **What it provides**: 6-axis AI Workload Profile Radar evaluating CPU, Memory, Disk, Network, Concurrency, and Syscall intensity.
- **Permission Denial & Security Alerts**: Immediate capture of permission denials (`EACCES`, `EPERM`) and blocked network operations.
- **Structured Runtime Logs**: Multi-severity log explorer (`INFO`, `WARN`, `CRITICAL`) with search filtering and one-click JSON export.
- **How to use**: Click the **Security & Logs** tab or press `5` to audit agent behavior and inspect security boundaries.

### 6. Linux Metric Encyclopedia Modal & Terminal Command Cheat Sheet
![Linux Metric Encyclopedia](screenshots/modal_metric_help.png)

- **What it provides**: Every KPI tile, chart header, and telemetry mode chip across the dashboard features an interactive `(?)` glyph. Clicking it opens the **Metric Help Modal**:
  - **Intuitive Analogies**: Plain-English explanations (e.g. comparing CFS runqueue latency to retail checkout lines).
  - **Threshold Guidance**: Explicit bands for Normal/Healthy, Warning, and Critical/Danger states.
  - **Why AI Agents Care**: Concrete impact on LLM token generation, memory footprint, or tool execution stalls.
  - **Copyable Terminal Verification**: Direct copy of Linux commands (`pidstat`, `sar`, `ss`, `lsof`) to verify numbers on your terminal.
  - **Kernel Source Location**: Exact Linux kernel source files where telemetry originates (e.g. `kernel/sched/core.c`, `/proc/[pid]/io`).

### 7. AI Observability Copilot Chat Drawer
![AI Copilot Chat Drawer](screenshots/chat_drawer_open.png)

- **What it provides**: Floating AI observability assistant available from any tab by clicking the bottom-right Copilot button.
- **100% Local Rule Engine**: Evaluates live snapshot metrics and diagnoses performance bottlenecks with zero external API keys.
- **Dynamic Chart Generation**: Ask the Copilot to graph correlations (e.g. *"Graph thread count vs memory growth"*), and it mounts the chart directly into your **Dynamic Graph Scratchpad**.

---

## 🌍 Universal Compatibility: Multi-Mode, Multi-Platform & Multi-Arch

DrishtiScope is engineered from the ground up to run anywhere developers, SREs, and AI engineers work:

### 1. Three Ingestion Modes
- **EBPF LIVE (`mode=ebpf`)**: Nanosecond-precision Linux kernel tracepoints (`raw_syscalls:sys_enter`, `sys_exit`, `sched_process_exec`, `sched_process_exit`) via 16MB in-kernel BPF ring buffers. On Windows, integrates with [Microsoft eBPF for Windows](docs/WINDOWS_EBPF.md) (`ebpfcore.sys`).
- **REAL LIVE (`mode=real`)**: Zero-root telemetry collector. On Linux/WSL2, reads genuine `/proc/[pid]/*` and `sysfs` telemetry. On native Windows, scans live processes via Windows system APIs (`tasklist.exe` & ETW bridge). Operates with zero privileges on locked-down enterprise workstations and developer laptops.
- **MOCK (`mode=mock`)**: Deterministic synthetic simulation with realistic sinusoidal CPU oscillation, tool bursts, and socket events for offline testing, UI design, and CI workflows.
- **AUTO (`mode=auto`, default)**: Automatically probes host permissions; attaches eBPF if privileged, otherwise falls back gracefully to Real Mode with zero synthetic data.

### 2. Multi-Platform Support
- **Linux (x86_64 & aarch64)**: Full native kernel 5.8+ support with BTF/CO-RE and zero-root `/proc` engine.
- **Windows Subsystem for Linux (WSL2)**: 100% feature parity in both Real and eBPF modes.
- **Native Windows 10/11 & Windows Server**: Native `drishtiscope.exe` single-binary execution with Windows process discovery and Microsoft eBPF for Windows integration.
- **macOS (Darwin amd64 & Apple Silicon arm64)**: Native compilation with synthetic demonstration engine for local UI development.

### 3. Multi-Architecture Standalone Binaries
- **Pure Go Userspace + Embedded SQLite TSDB (`modernc.org/sqlite`)**: Zero CGO dependencies.
- Packaged as standalone ~18 MB single binaries across 6 architectures:
  - `linux/amd64`
  - `linux/arm64`
  - `windows/amd64` (`drishtiscope.exe`)
  - `windows/arm64` (`drishtiscope.exe`)
  - `darwin/amd64`
  - `darwin/arm64` (Apple Silicon)

---

## 🌟 Key Features

### 1. Dual-Engine Architecture: eBPF + Zero-Root Real Mode
- **eBPF Mode (`mode=ebpf`)**: Loads in-kernel tracepoints (`raw_syscalls:sys_enter`, `sys_exit`, `sched_process_exec`) for nanosecond latency profiling.
- **Zero-Root Real Mode (`mode=real`)**: Works immediately on locked-down laptops, enterprise workstations, and WSL2 without root or sudo. Polls `/proc/[pid]/*` and host sysfs interfaces to measure CPU ticks, memory, open files, network flows, and scheduler latency.
- **Auto Mode (`mode=auto`, default)**: Automatically detects kernel permissions; attaches eBPF if privileged, otherwise falls back gracefully to Real Mode with zero synthetic mock data.

### 2. 5 Dedicated Observability Tabs
- **Tab 1: Process Story (`Activity`)**: Chronological event feed, file touches, socket connections, and live AI Executive Verdict (*Active Code Generation*, *High I/O & Socket Activity*, *Idle Event Loop*). Includes one-click copyable Linux diagnostics (`strace`, `pidstat`, `lsof`).
- **Tab 2: Overview (`Vitals`)**: Core SRE Golden Signals: Syscall latency quantiles (P50, P90, P99), traffic waveform, active threads, open FDs, and SLO error budget burn rate.
- **Tab 3: Execution & CPU (`Call Trees`)**: Continuous flamegraph profiler, call tree drill-downs, and single-click export to [ui.perfetto.dev](https://ui.perfetto.dev).
- **Tab 4: System Metrics (`Telemetry`)**: Metrics Query Language (MQL) console, subsystem breakdown donut charts (CPU, Memory, Disk, Network), and physical disk IOPS graphs.
- **Tab 5: Security & Logs (`Audit`)**: AI Workload Security Radar, permission denial audits (`EACCES`, `EPERM`), refused outbound connections, and structured runtime logs.

### 3. 5 Ergonomic Themes (Light Mode Default)
Switch between 5 themes instantly from the top header:
- **☀️ Light Mode (Default)**: Clean, high-contrast palette with soft slate borders (`#f8fafc` background, `#0f172a` text). Optimized for bright daytime environments and reading comfort.
- **🌙 Dark Mode**: Classic nocturnal control room palette (`#07080d` background, `#0e1118` panels, cyan/emerald accents).
- **🟠 Ubuntu Mode**: Canonical-inspired warm aubergine (`#2c001e` background, `#dd4814` orange accents).
- **📟 Unix Mode**: Retro green-screen terminal aesthetic (`#0a0f0d` background, `#00ff66` phosphor accents).
- **🔮 Purple Mode**: Cyber synthwave neon aesthetic (`#0d0b18` background, `#a855f7` violet accents).

### 4. Live Executing Process Omnibox (`/`)
- **Instant Keyboard Navigation**: Press `/` anywhere in the dashboard to instantly focus the search omnibox.
- **Real-Time Suggestions**: Automatically ranks and surfaces executing processes by CPU and RSS memory.
- **Full Keyboard Navigation**: Cycle candidates with `ArrowDown` / `ArrowUp`, select with `Enter`, or dismiss with `Escape`.
- **Direct PID / Comm Targeting**: Target any process on the fly without restarting the daemon.

### 5. Anti-Flicker & Stream Rate Controls
High-frequency streams can strain an engineer's eyes. DrishtiScope provides:
- **Stream Rate Throttling**: Choose between `500ms` (Rapid), `1s` (Balanced), `2s` (Calm default), `5s` (Relaxed), or `Manual`.
- **Anti-Flicker Toggle**: Dampens rapid numerical jitter and suppresses abrupt pulse animations.
- **Live / Paused Freeze**: Pause the stream at any millisecond to inspect and copy state.

### 6. Metric Encyclopedia with Explanatory Glyphs `(?)`
Every KPI tile, chart header, and telemetry mode chip features an interactive `(?)` glyph. Clicking it opens the **Metric Help Modal** with:
- Plain English analogies (e.g., comparing runqueue latency to a grocery store checkout).
- Healthy, warning, and critical thresholds.
- Impact on AI agent token generation and tool execution.
- Copyable terminal verification commands (`pidstat`, `strace`, `ss`, `lsof`).
- Exact Linux kernel source code locations (e.g. `kernel/sched/core.c`, `/proc/[pid]/io`).

### 7. Dynamic Graph Scratchpad & Linux Playground
Mount specialized graphs on demand:
- *Syscall Latency Quantile Curve (P50 vs P90 vs P99)*
- *Context Switches vs Runqueue Latency*
- *Bidirectional Network Throughput (Tx vs Rx)*
- *Thread Count vs RSS Memory Usage*
- *Page Fault Dynamics (Minor vs Major Faults)*
- Custom graphs can also be created dynamically via the AI Copilot.

### 8. AI Observability Copilot Chat Drawer
- **100% Local Rule Engine**: Operates with zero external API keys, diagnosing bottlenecks from live snapshots locally.
- **Natural Language Analysis**: With an optional API key configured strictly via environment variables (`GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GROQ_API_KEY`), the Copilot analyzes memory growth, socket stalls, or file descriptor leaks in context. No keys are ever bundled or exposed.

### 9. Comprehensive Testing & Enterprise Exporters
- **Playwright E2E Suite**: 40 responsive tests covering Desktop Chrome and Mobile Chrome (`npm run test:e2e`).
- **Prometheus (`/metrics`)**: OpenMetrics standard gauges and counters.
- **Perfetto Traces (`/api/v1/traces/perfetto`)**: Direct import into [ui.perfetto.dev](https://ui.perfetto.dev).
- **Structured Logs (`/api/v1/logs`)**: Ingestible by Loki, Elasticsearch, or Cloud Logging.
- **Health Probes (`/healthz`, `/livez`, `/readyz`)**: Standard Kubernetes liveness and readiness probes.

---

## 🏛️ System Architecture

```mermaid
flowchart TD
    subgraph Host ["Linux Host (Kernel >= 5.8 / WSL2)"]
        Target["AI Agent Target Process (codex, agy, copilot, node, python)"]
        
        subgraph KernelSpace ["Kernel Space (eBPF & /proc)"]
            Tracepoints["sys_enter / sys_exit Tracepoints"]
            RingBuf["BPF Ring Buffer (16MB)"]
            ProcScan["/proc/[pid]/stat, status, io, fd, net/tcp"]
            Tracepoints --> RingBuf
            Target --> Tracepoints
            Target --> ProcScan
        end
        
        subgraph DrishtiScope ["DrishtiScope Go Daemon (:8080)"]
            Loader["eBPF Loader / Real /proc Engine"]
            ProcEnrich["/proc Inode & Socket Matcher"]
            Aggregator["Agent Golden Signals & Sliding Window Aggregator"]
            TSDB[("Embedded SQLite TSDB (WAL)")]
            Hub["WebSocket Broadcast Hub (400ms ticks)"]
            CopilotAPI["AI Copilot LLM Engine (Gemini / Claude / OpenAI)"]
            
            RingBuf --> Loader
            ProcScan --> Loader
            Loader --> Aggregator
            ProcEnrich --> Aggregator
            Aggregator --> TSDB
            Aggregator --> Hub
            Aggregator --> CopilotAPI
        end
        
        subgraph Endpoints ["DrishtiScope Endpoints"]
            Prometheus["/metrics (Prometheus)"]
            Healthz["/healthz, /livez, /readyz"]
            PerfettoExport["/api/v1/traces/perfetto"]
            LoggingExport["/api/v1/logs"]
            ChatEndpoint["/api/chat (Agent Copilot)"]
            
            Aggregator --> Prometheus
            Aggregator --> Healthz
            Aggregator --> PerfettoExport
            Aggregator --> LoggingExport
            CopilotAPI --> ChatEndpoint
        end
    end
    
    subgraph UI ["DrishtiScope Console (:5173 / :8080)"]
        Tab1["Tab 1: Process Story (Chronological Activity & Verdict)"]
        Tab2["Tab 2: Overview (Agent Golden Signals & Vitals)"]
        Tab3["Tab 3: Execution & CPU (Traces & Flamegraph)"]
        Tab4["Tab 4: System Metrics (Telemetry & MQL)"]
        Tab5["Tab 5: Security & Logs (Audit & Sandbox)"]
        HelpModal["Metric Encyclopedia Popup Modal"]
        ChatDrawer["Floating Agent Copilot Chat"]
        
        Hub --> Tab1
        Hub --> Tab2
        Hub --> Tab3
        Hub --> Tab4
        Hub --> Tab5
        Tab1 -.-> HelpModal
        Tab2 -.-> HelpModal
        ChatEndpoint <--> ChatDrawer
    end
```

---

## 🚀 Quickstart

### Option 1: Run Pre-Built Binary (Zero-Root Real Mode)

```bash
# Download the latest binary for your architecture
curl -sSL https://github.com/drishtiscope/drishtiscope/releases/latest/download/drishtiscope_linux_amd64.tar.gz | tar -xz

# Start DrishtiScope in Real Mode watching an agent (e.g. agy or codex)
./drishtiscope -mode=real -comm=agy

# Open dashboard
open http://localhost:8080
```

### Option 2: Run with Docker / Podman

```bash
docker run -d --name drishtiscope \
  --pid=host \
  --net=host \
  -v /proc:/proc:ro \
  ghcr.io/drishtiscope/drishtiscope:latest
```

### Option 3: Build from Source

```bash
# 1. Clone repository
git clone https://github.com/drishtiscope/drishtiscope.git
cd drishtiscope

# 2. Build frontend
cd frontend
npm ci
npm run build
cd ..

# 3. Build backend
cd backend
go build -ldflags="-s -w" -o ../drishtiscope ./cmd/agentscope
cd ..

# 4. Launch DrishtiScope
./drishtiscope -mode=real -comm=agy -static=frontend/dist
```

### Option 4: Run Natively on Windows

DrishtiScope runs natively on Windows 10/11 using pure Go, native Windows process scanning (`tasklist.exe`), and interfaces with [Microsoft eBPF for Windows](docs/WINDOWS_EBPF.md):

```powershell
# In PowerShell:
# 1. Build or download Windows binary
cd backend
go build -ldflags="-s -w" -o ../drishtiscope.exe ./cmd/agentscope
cd ..

# 2. Run DrishtiScope targeting your agent process (e.g. agy.exe, python.exe, powershell.exe)
.\drishtiscope.exe -mode=real -comm=agy

# 3. Open dashboard in your browser
Start-Process http://localhost:8080
```

---

## ❓ Frequently Asked Questions (FAQ)

Here are answers to the most common questions. For detailed explanations, see the full [docs/FAQ.md](docs/FAQ.md).

#### 1. What makes DrishtiScope fundamentally different from LangSmith, Langfuse, and Phoenix?
Application tracers observe what the agent *said* it did; DrishtiScope observes what the host process *actually did*. It tracks child `execve` bursts, file modifications, socket throughput, and kernel scheduling stalls on closed-source binaries without SDKs.

#### 2. How does DrishtiScope observe closed-source AI agents without code changes?
By reading live `/proc/[pid]/*` files (`stat`, `statm`, `status`, `io`, `fd`, `net/tcp`) and optionally attaching non-invasive eBPF kernel tracepoints to `raw_syscalls`.

#### 3. Does DrishtiScope require `root` or `sudo`?
No. In Real Mode (`mode=real`), DrishtiScope operates completely unprivileged without `sudo` on developer laptops and WSL2.

#### 4. Does DrishtiScope intercept or store prompts, completions, or LLM tokens?
No. DrishtiScope adheres to a strict zero-payload principle. It does not sniff TLS bodies or store prompt text. Only system metadata and resource metrics are recorded.

#### 5. How does DrishtiScope compare to AgentSight?
AgentSight uses TLS uprobes to capture prompts alongside syscalls. DrishtiScope is an SRE control room focused on process vitals, golden signals, and system bottlenecks without payload decryption.

#### 6. What is the CPU and memory overhead?
Under 1.2% CPU and 18–35 MB RSS in Real Mode. Under 1.5% CPU in high-frequency eBPF mode.

#### 7. How does Anti-Flicker mode work?
It throttles stream intervals (500ms to 5s) and smooths UI animations to eliminate eye strain from continuous WebSocket updates.

#### 8. What are the 5 tabs?
Process Story (Activity), Overview (Vitals), Execution & CPU (Flamegraphs), System Metrics (MQL), and Security & Logs (Audit).

#### 9. How do the 5 themes work?
Supports Light (Default), Dark, Ubuntu, Unix, and Purple themes via CSS custom properties.

#### 10. What is the Dynamic Graph Scratchpad?
An on-demand charting canvas at the bottom of the dashboard for visualizing metric correlations (latency quantiles, runqueue vs switches, network throughput).

#### 11. Does the AI Copilot require an external API key?
No. It includes a built-in deterministic rule engine that diagnoses bottlenecks locally without an API key.

#### 12. Can I export telemetry to Prometheus and Perfetto?
Yes. DrishtiScope provides `/metrics` for Prometheus scraping and `/api/v1/traces/perfetto` for [ui.perfetto.dev](https://ui.perfetto.dev).

#### 13. Does it run on WSL2, Windows, or macOS?
Yes. WSL2 is supported natively in Real Mode. Experimental Windows eBPF and Darwin synthetic modes are included.

#### 14. How is data persisted?
In an embedded SQLite database using WAL mode with asynchronous decoupled writes.

#### 15. How do I contribute?
Read [CONTRIBUTING.md](CONTRIBUTING.md) and submit a pull request!

---

## 🤝 Contributing

We welcome contributions from developers, Linux systems engineers, and AI researchers worldwide. Please see [CONTRIBUTING.md](CONTRIBUTING.md) for full development guidelines, testing standards, and architecture details.

---

## 📄 License

- DrishtiScope userspace Go daemon and React console are licensed under the [MIT License](LICENSE).
- The eBPF kernel program (`backend/bpf/agent.bpf.c`) is dual-licensed under **GPL-2.0 OR MIT**.
