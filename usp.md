# DrishtiScope — System Design, Motivation & Architectural Value

> **A Lightweight, System-Level Process Observability Console**  
> Observing kernel interactions, system calls, resource utilization, and runtime behavior of any running process — without code modification, without proxies, and keeping all telemetry 100% local.

---

## 1. Project Background & Author's Motivation

DrishtiScope (दृष्टिScope) was developed as an independent engineering, research, and learning project to deeply understand **operating system internals and process lifecycles**. 

When building and running modern software—whether it is a backend web service, a local developer CLI, a database engine, a compiler pipeline, a background worker daemon, or an autonomous AI agent runtime—it is often difficult to know what the process is actually doing beneath the application code. 

Most conventional application monitoring frameworks focus on what the software voluntarily logs through an SDK or decorator. While this application-level tracing is extremely valuable and well-engineered, there is another fundamental layer: **the host kernel layer**.

The goal of creating DrishtiScope was simple and educational:
1. **Learn low-level systems programming**: Explore how Linux kernel mechanisms (`/proc`, eBPF tracepoints, scheduling statistics, socket tables) and Windows system interfaces (`tasklist`, Win32 APIs, ETW bridges) actually capture process behavior.
2. **Observe any process transparently**: Enable developers and system administrators to inspect the real-time execution of *any* target process without requiring source code modifications, custom instrumentation, or third-party cloud connections.
3. **Understand resource dynamics**: Directly measure system call throughput, microsecond-level latency, resident memory footprint, open file descriptors, and active socket endpoints as they happen in real time.

---

## 2. Universal Scope: Observing Any Process

Although the project was initially inspired by observing autonomous coding agents and LLM runtimes (which execute rapid tool bursts, spawn child shells, and touch hundreds of files), **DrishtiScope is completely universal**. 

It can observe and analyze any process executing on the host system:
- **Backend Services & Web APIs**: Node.js, Go, Python, Java, or Rust web servers.
- **Developer Tools & Compilers**: `gcc`, `clang`, `cargo`, `go build`, `git`, or `npm`.
- **Database & Storage Daemons**: Embedded databases, key-value stores, or message queues.
- **Autonomous AI Agents & Script Runtimes**: Local model runners, coding assistants, and CLI automation tools.
- **System Utilities & Shell Pipelines**: Background worker scripts, container utilities, and administrative tools.

Operators simply provide the process ID (`-pid=1234`) or process name (`-comm=my-process`), and DrishtiScope begins live, non-invasive observation.

---

## 3. The Three Observability Layers

To understand where DrishtiScope fits, it helps to look at observability in three complementary layers:

```
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER A — Application & SDK Tracing                                     │
│  • What it observes: Internal functions, application logs, spans.        │
│  • How it works: Requires SDK imports, library wrappers, or API proxies. │
│  • Best suited for: Application business logic, user journey tracing.    │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER B — Fleet & Cluster Monitoring                                    │
│  • What it observes: Machine-wide metrics, network meshes, node health.   │
│  • How it works: Cluster agents, central log forwarders, daemon sets.    │
│  • Best suited for: Large distributed production clusters & fleet alerts.│
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER C — System-Level Process Observability (DrishtiScope)             │
│  • What it observes: Exact process behavior, system calls, file touches, │
│    socket connections, CPU ticks, and scheduler delay for a target PID.  │
│  • How it works: Kernel tracepoints (eBPF) and standard /proc scanning.  │
│  • Best suited for: Deep local process debugging, performance triage,    │
│    understanding host interactions without modifying source code.        │
└──────────────────────────────────────────────────────────────────────────┘
```

> [!NOTE]
> **Mutual Respect for Existing Tools**: Layer A and Layer B tools in the industry represent decades of combined engineering excellence and solve critical enterprise challenges. DrishtiScope does not attempt to replace or compete with application-level tracers or enterprise monitoring stacks. Instead, it serves as an educational and operational microscope for inspecting what an individual process is doing at the operating system boundary.

---

## 4. Key Technical Capabilities

### 4.1 Non-Invasive Process Attachment
- Operates strictly from userspace or kernel tracepoints outside the target process.
- No SDK installation, no language-specific packages, and no code rebuilds required.
- Capable of observing closed-source binaries, third-party executables, and standard shell utilities.

### 4.2 Dual-Engine Design (eBPF + Zero-Root /proc)
- **eBPF Engine (`mode=ebpf`)**: When run with administrative privileges (`CAP_BPF` or `root`), attaches to kernel tracepoints (`raw_syscalls:sys_enter/exit`, `sched_process_exec/exit`) for sub-microsecond latency measurement.
- **Real Engine (`mode=real`)**: Designed for standard workstations and locked-down developer environments. Reads `/proc/[pid]/*` files (`stat`, `statm`, `io`, `fd`, `net/tcp`) without needing `root` or `sudo`.
- **Auto Fallback (`mode=auto`)**: Automatically checks system capabilities. If privileged eBPF is unavailable, it gracefully defaults to the real `/proc` engine with zero interruption.

### 4.3 Process Story & Timeline
- Generates a chronological stream of significant process events: child process creation (`execve`), file access (`openat`, `write`), socket connections (`connect`), and permission errors (`EACCES`).
- Provides a high-level operational assessment (e.g., *Active Computation*, *Heavy File & Socket I/O*, *Waiting on Network / Idle Event Loop*).

### 4.4 SRE Golden Signals for Single Processes
Adapts the classic Google Site Reliability Engineering (SRE) Golden Signals to individual processes:
- **Latency**: P50, P90, and P99 system call latency quantiles.
- **Traffic**: System call rate (calls per second) and network socket throughput (bytes per second).
- **Errors**: Non-zero kernel error return rate (`EACCES`, `EPERM`, `ECONNREFUSED`).
- **Saturation**: CPU core utilization percentage, CFS scheduler runqueue delay, and open file descriptor consumption against system limits.

### 4.5 Self-Contained, Single-Binary Architecture
- Packaged as a single static binary (~18 MB) with zero runtime dependencies.
- Embeds the compiled web UI, an embedded SQLite time-series database (WAL mode), standard Prometheus `/metrics`, Kubernetes health probes (`/healthz`, `/livez`, `/readyz`), and Perfetto trace export (`/api/v1/traces/perfetto`).

### 4.6 Strict Privacy & Zero-Payload Design
- DrishtiScope is strictly an **operational and performance monitor**.
- It **does not inspect, record, or decrypt network payloads**, prompt text, source code files, or user data.
- It only records system metadata: process names, file paths, socket addresses/ports, and performance counters.
- All collected telemetry remains 100% local on the machine in an embedded SQLite file (`drishtiscope.db`). No telemetry is ever transmitted to any remote cloud server.

---

## 5. Practical Use Cases & Problem Scenarios

Here are common real-world debugging scenarios where host-level process observability is helpful:

### Scenario 1: A process is consuming high CPU, but logs show nothing
- **Challenge**: A background worker or compiler tool is pegging a CPU core at 99%, but has not emitted any log lines.
- **DrishtiScope View**: The **Execution & CPU** tab and call tree reveal whether the process is spinning in user code, looping on regular expressions, or stuck in system call loops. The user can export a trace to Perfetto for nanosecond timeline inspection.

### Scenario 2: Investigating file and disk bottlenecks
- **Challenge**: A build job or data processor is running unusually slow.
- **DrishtiScope View**: The **Process Story** and **Telemetry** tabs display exact real-time physical read/write throughput, open file descriptors, and specific workspace files currently being accessed.

### Scenario 3: Identifying permission denials and sandbox boundaries
- **Challenge**: A script or container process exits with an ambiguous error code.
- **DrishtiScope View**: The **Security & Audit** tab captures kernel error codes (such as `EACCES` or `EPERM`) with the exact file path or socket address that was rejected, without having to run heavy forensic tools.

### Scenario 4: Laptop development without elevated permissions
- **Challenge**: Developers often work on corporate laptops or shared servers where root/sudo access is strictly restricted.
- **DrishtiScope View**: In `mode=real`, DrishtiScope gathers comprehensive process metrics via standard unprivileged `/proc` access, giving developers immediate visibility on their local machine.

---

## 6. What DrishtiScope is NOT (Limitations & Boundaries)

To maintain absolute clarity, here is what DrishtiScope is explicitly not designed to do:

| Domain | DrishtiScope's Scope | Dedicated Tools |
| :--- | :--- | :--- |
| **Application-Level Business Spans** | Observes operating system effects. Does not trace internal application functions. | Application-level tracing libraries and APMs |
| **Payload Content Inspection** | Strictly payload-agnostic. Does not inspect payload text or decryption keys. | Network packet analyzers or TLS proxy loggers |
| **Cluster-Wide Distributed Tracing** | Designed for single-node process inspection. | Distributed tracing and fleet monitoring platforms |
| **Kernel-Level Inline Enforcement** | Passive observability only. Does not block or terminate system calls. | Linux Security Modules (LSM), SELinux, AppArmor |
| **Long-Term Enterprise Warehousing** | Retains recent rolling history in a local SQLite file. | Centralized time-series databases and data warehouses |

---

## 7. Educational Value: Metric Encyclopedia

To make low-level operating system metrics approachable for developers of all experience levels, DrishtiScope includes a built-in **Metric Encyclopedia**:
- Every KPI card and chart has an interactive `(?)` help button.
- Provides simple analogies explaining what each metric means (e.g., comparing scheduler runqueue latency to waiting at a checkout counter).
- Lists healthy, warning, and critical reference thresholds.
- Provides copyable terminal commands (`pidstat`, `strace`, `lsof`, `ss`) so engineers can independently verify the numbers on their own terminal.
- Points to the exact Linux kernel source files where the counters originate (e.g., `kernel/sched/core.c`, `/proc/[pid]/io`).

---

## 8. Important Caveats & Disclaimers

> [!CAUTION]
> **Independent Open-Source Work**: This project was developed strictly by the author in an individual capacity for learning and technical research. It is completely independent and has no official endorsement, sponsorship, or affiliation with any organization or commercial entity.

> [!NOTE]
> **Kernel Version Compatibility**: System-level metrics depend on the host operating system. Linux kernels 5.8+ provide the best eBPF capabilities. On earlier kernels or unprivileged containers, the `/proc` engine provides the primary data source. On Windows, metrics are collected via native Win32 process APIs, with optional support for the open-source Microsoft eBPF for Windows driver.

> [!TIP]
> **Local Testing**: Users are encouraged to run and test DrishtiScope in local or non-production environments to explore operating system dynamics and process observability safely.
