# DrishtiScope — Unique Selling Proposition

> **The SRE control room for autonomous AI agents.**
> Kernel-grounded process observability for coding agents and LLM runtimes — without an SDK, without a proxy, without sending telemetry to the cloud.

**Product:** DrishtiScope (दृष्टिScope)  
**Category:** System-level observability for autonomous AI agents  
**One-liner:** Watch what `codex`, `claude`, `agy`, `grok`, Copilot, Ollama, and any other agent process *actually does* on the host — syscalls, files, sockets, CPU, memory — in a 400 ms live dashboard, with a process story, SRE golden signals, and a copilot that has the snapshot in context.

**Audience for this doc:** founders, SREs, platform engineers, security reviewers, and anyone comparing “LLM observability” tools who needs a honest map of the market.

**Research date:** 13 September 2026.

---

## 0. How to read this document

This is a positioning and competitive-intelligence brief, not a feature dump.

| Section | What it answers |
|---|---|
| 1. The gap | Why existing tools miss the thing DrishtiScope watches |
| 2. Positioning map | Three layers of “AI observability” and where we sit |
| 3. The USP | Seven claims that survive a competitor check |
| 4. Full catalog | Every similar tool we found, grouped by job |
| 5. Head-to-head matrices | Feature tables, not slogans |
| 6. Worked examples | Same incident, four tools, four answers |
| 7. Honest gaps | What we do *not* sell |
| 8. Pitches | 15-second, 60-second, and slide-ready copy |
| 9. Sources | Where the competitive claims come from |

---

## 1. The gap nobody in LLMOps owns

Almost every product marketed as “AI agent observability” in 2026 watches **what the application logged**.

That is a useful layer. It is not the same layer as **what the process did**.

An autonomous coding agent (`codex`, Claude Code, Copilot, Grok, `agy`, a local Ollama worker) does not look like a REST microservice:

| Agent behaviour | What SDK tracers see | What the kernel sees |
|---|---|---|
| Spawn `bash`, `git`, `docker`, `python` in a burst of `execve` / `vfork` | A “tool call” span, if the framework emitted one | Real child PIDs, PPID tree, cmdline, open FDs |
| Stream tokens over a long-lived TLS socket | Token count and latency, if the SDK wrapped the client | `ESTABLISHED` sockets, TX/RX bytes, P99 syscall latency |
| Rewrite the workspace, write transcripts, touch SQLite | Nothing, unless the agent logged the path | `openat`, `write`, inode, recent file touches |
| Sit in `epoll_wait` / `futex` between user turns | “idle” only if the app said so | Runqueue, CPU %, syscall mix, stall vs. wait |
| Hit `EACCES` on `/etc/shadow` or a refused outbound connect | Often nothing | Security sandbox events, error rate, error budget burn |

SDK products (LangSmith, Langfuse, Phoenix, Helicone, Datadog LLM Observability) require **you to own the code**, wrap the LLM client, or put a **proxy in front of the model API**.

That fails for the workload DrishtiScope is built for:

1. **Closed-source CLIs** you did not write (`codex`, Claude Code, Copilot).
2. **Subprocesses that bypass instrumentation** (`bash` pipelines, compilers, docker).
3. **Host effects** that never appear in a GenAI span (file I/O, FD leaks, socket stalls).
4. **Air-gapped / local** use where you will not ship prompts to a SaaS.

The 2025–2026 research literature calls this the **semantic gap**: application traces show *intent*; the kernel shows *effect*. AgentSight (eunomia-bpf, arXiv:2508.02736) is the academic/engineering project that named it. DrishtiScope occupies the same gap, with a different product shape: **an SRE dashboard, not a `top`/`strace` recorder.**

---

## 2. Positioning map — three layers of AI observability

```
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER A — Application / SDK  ("what the agent said it did")             │
│  LangSmith · Langfuse · Phoenix · Braintrust · Helicone · AgentOps       │
│  Opik · Logfire · OpenLIT · Datadog LLM · New Relic AI Obs               │
│                                                                          │
│  Needs: SDK, decorator, gateway, or vendor plugin                        │
│  Sees:  prompts, tokens, tool spans, evals, cost                         │
│  Misses: child processes, syscalls, /proc, real sockets, kernel stalls   │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │  you still don't know if the
                                    │  agent is burning the disk,
                                    │  leaking FDs, or stalled on TLS
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER B — Generic kernel / eBPF APM  ("what the machine did")           │
│  Pixie · Coroot · Grafana Beyla · Groundcover · Metoro · Hubble          │
│  Falco · Tetragon · Tracee · Sysdig · Prometheus Node Exporter           │
│                                                                          │
│  Needs: cluster / root / DaemonSet, not agent-aware                      │
│  Sees:  HTTP, syscalls, service maps, security rules                     │
│  Misses: agent identity, process story, coding-agent verdicts,           │
│          junior-admin metric encyclopedia, single-binary laptop use      │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │  the missing product
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  LAYER C — System-level agent observability  ("what THIS agent did")     │
│                                                                          │
│  DrishtiScope     SRE control room · eBPF + /proc · golden signals       │
│  AgentSight       closest peer · eBPF + TLS sniff · TUI / record         │
│  OpenLIT coding   vendor CLI hooks · tokens & spend, not kernel          │
│  cctop / CTOP /   session-file TUIs · "htop for Claude" · no syscalls    │
│  agentop / Helix / abtop                                                 │
│                                                                          │
│  Needs: attach to a live process (or read session files)                 │
│  Sees:  the host effects of a closed-source agent, with agent semantics  │
└──────────────────────────────────────────────────────────────────────────┘
```

**DrishtiScope’s slot:** Layer C, SRE-shaped.

Not “another Langfuse.” Not “Pixie for Kubernetes.” Not “htop that greps `~/.claude`.”

> The live, kernel-grounded control room for one (or a handful of) autonomous agent processes on a Linux host — with a process story, Google SRE golden signals, Prometheus, Perfetto, an embedded TSDB, and a copilot that can see the snapshot.

---

## 3. The unique selling proposition

### 3.1 One sentence

**DrishtiScope is the only production-shaped, single-binary SRE dashboard that observes autonomous AI agent processes from the Linux kernel (eBPF + live `/proc`) with zero instrumentation, a chronological process story, golden-signal SLOs, and a telemetry-aware copilot — without sniffing TLS payloads or shipping data to a SaaS.**

### 3.2 Seven claims that actually differentiate

These are the claims that remain after checking the 40+ tools in section 4.

| # | Claim | Why it is not generic |
|---|---|---|
| 1 | **Zero-instrumentation, closed-source-agent native** | Works on `codex` / Claude / Copilot / Grok binaries you did not write. LangSmith/Langfuse/Phoenix cannot. |
| 2 | **Kernel ground truth, not session files** | eBPF tracepoints + `/proc/[pid]/{stat,status,io,fd,net/tcp}`. cctop/CTOP/OpenLIT coding read `~/.claude` JSON. Different physics. |
| 3 | **Process Story + executive verdict** | A chronological feed plus a live assessment (*Idle event loop* / *Active code generation* / *High I/O & sockets* / *Rapid kernel ops*). SDK tracers show span trees. Security tools show alerts. Nobody else narrates the *process*. |
| 4 | **SRE golden signals applied to the agent** | Latency (syscall P50/P90/P99), Traffic (RPS + net B/s), Errors (nonzero returns), Saturation (CPU, runqueue, FDs) + 99.9% SLO burn. Pixie/Coroot do this for HTTP services, not for a coding agent PID. |
| 5 | **Zero-root real mode** | `MODE=auto` tries eBPF, falls back to live `/proc` scanning without root. AgentSight’s deep capture wants `sudo`. Falco/Tetragon/Pixie want privileged DaemonSets. |
| 6 | **Self-contained 18 MB binary** | Embedded React SPA + SQLite WAL TSDB + Prometheus `/metrics` + `/healthz` + Perfetto export. No ClickHouse, Redis, S3, Grafana, or SaaS bill. Langfuse self-host is a multi-service stack. |
| 7 | **Privacy by architecture** | Observes *effects* (files, sockets, syscalls, CPU). Does **not** intercept TLS to read prompts. AgentSight’s differentiator is SSL/TLS plaintext capture. We deliberately do the opposite. |

### 3.3 Supporting product surface (the “control room”)

These are not unique one-by-one. The *bundle* is.

- **5 tabs:** Process Story → Overview (vitals) → Execution & CPU (call trees / Perfetto) → System Metrics (MQL) → Security & Logs.
- **Metric Encyclopedia:** every KPI has a `(?)` popup — plain-English analogy, healthy/warning/critical bands, “why AI agents care,” copyable `pidstat`/`strace`/`ss` command, exact kernel source.
- **Agent Copilot:** floating chat that injects the live snapshot (CPU, RSS, error rate, FDs, sockets) into Gemini / Claude / OpenAI, or runs rule-based local diagnostics with no key.
- **400 ms WebSocket ticks** with an Anti-Flicker mode so the control room does not strobe.
- **Hot-switch the observed PID** from the process tree (“Set Active”).
- **Exporters ops teams already use:** Prometheus OpenMetrics, Perfetto JSON (`ui.perfetto.dev`), structured `LogEntry` logs, liveness/readiness probes.

---

## 4. Full catalog of similar tools

Grouped by the job they actually do. “Similar” here means a buyer might confuse them with DrishtiScope in a 10-minute search.

### 4.1 Layer A — LLM / agent tracing, evals, cost (SDK or proxy)

These are the products every “best AI observability 2026” roundup ranks. They are **complements**, not substitutes, unless you own the agent source.

| Tool | Maker | License / host | What it is good at | Why it is not DrishtiScope |
|---|---|---|---|---|
| **LangSmith** | LangChain | Proprietary. Cloud; self-host is Enterprise. ~$39/seat/mo | Deepest LangChain/LangGraph traces, prompt hub, replay | Requires LangChain (or OTel ingest). No kernel, no closed-source CLI, no `/proc`. |
| **Langfuse** | ClickHouse (acquired Jan 2026) | MIT. Cloud or self-host. ~34.5k GitHub stars | Most adopted OSS LLM workbench: traces, prompts, evals, datasets | Self-host needs ClickHouse + Redis + S3. SDK/`@observe`. No eBPF, no process story. |
| **Arize Phoenix** | Arize | Elastic-2.0. Local or cloud | OTel + OpenInference, RAG evals, UMAP drift, one-container local debug | Eval/tracing toolkit. Not a host-process SRE console. |
| **Braintrust** | Braintrust | Proprietary. Cloud. ~$249/mo | Eval-first: datasets, scoring, CI for prompts | No kernel. No closed-source agent attach. |
| **Helicone** | Helicone | OSS + cloud. Proxy. from ~$20/mo | One-line base-URL swap, cost, caching | Sees provider HTTP. Misses `execve`, files, local tools. |
| **AgentOps** | AgentOps | MIT (agent-first, session replay) | CrewAI / AutoGen loops, time-travel replay | Framework SDK. Not eBPF. |
| **Comet Opik** | Comet | OSS + cloud. ~22k stars | Trace + eval + dashboards for RAG/agents | Application telemetry. |
| **Pydantic Logfire** | Pydantic | Commercial. OTel-native | Full-stack + AI spans in one trace, SQL-queryable | Needs OTel in your app. |
| **OpenLIT** | OpenLIT | OSS. OTel. ~2.8k stars | 70+ LLM integrations **and** a coding-agent CLI for Claude/Cursor/Codex spend | Coding mode is vendor-plugin + OTel, not kernel. Closest *product-adjacent* on coding agents, still Layer A. |
| **OpenLLMetry** | Traceloop | Apache-2.0 SDK | Auto-instrument OpenAI/Anthropic/LangChain → any OTel backend | A library, not a host dashboard. |
| **Langtrace** | Langtrace | AGPL app | OTel LLM tracer, light self-host | Same layer as Langfuse, thinner. |
| **Galileo** | Galileo | Commercial | Quality, hallucination, guardrails | Eval/quality, not process. |
| **Maxim AI** | Maxim | Commercial | Simulation → eval → observability + gateway | Agent-app lifecycle, not kernel. |
| **Portkey** | Portkey | Commercial | AI gateway, routing, fallbacks, basic logs | Proxy. |
| **HoneyHive** | HoneyHive | Commercial | Enterprise eval + tracing | SDK. |
| **LangWatch** | LangWatch | Commercial. from €59 | Tracing + eval | SDK. |
| **Laminar** | Laminar | OSS + cloud | OTel-native, chat-with-trace | Application traces. |
| **W&B Weave** | Weights & Biases / CoreWeave | Commercial | Experiment tracking + LLM traces | MLOps, not host SRE. |
| **PostHog AI** | PostHog | OSS + cloud | Product analytics + LLM traces | Product analytics company. |
| **Datadog LLM Observability** | Datadog | SaaS | Agent graphs for LangGraph/CrewAI/OpenAI Agents SDK; Bits AI SRE | Needs their SDK/agent. 10–60 s ingest. Cloud bill. No laptop single-binary. |
| **New Relic AI Observability** | New Relic | SaaS | LLM agents, cost, quality (GA June 2026) + SRE Agent | Same: APM vendor, instrumentation, cloud. |
| **SigNoz** | SigNoz | MIT + cloud | One backend for traces/logs/metrics; can ingest LLM OTel | Generic OTel. Not agent-process-aware. |
| **LiteLLM** | BerriAI | OSS. ~58k stars | Unified LLM proxy | Gateway, not observability of the *host process*. |
| **Promptfoo / DeepEval / RAGAS** | various | OSS | Prompt/RAG evaluation harnesses | Evals only. |

**Rule of thumb:** if the first setup step is `pip install …` and `init()`, it is Layer A.

### 4.2 Layer B — eBPF / kernel observability and runtime security (not agent-aware)

These *do* see the kernel. They are built for Kubernetes, HTTP microservices, or threat detection — not for “is my coding agent stalled.”

#### Observability / APM

| Tool | What it is | Agent-aware? | vs DrishtiScope |
|---|---|---|---|
| **Pixie** (CNCF) | In-cluster eBPF: full-body HTTP, service maps, PxL scripts, profiles | No | Cluster debugger. Heavy. No process story for `codex`. |
| **Coroot** | OSS eBPF APM: service maps, SLOs, cost, AI RCA. Percona edition for DBs | No | Excellent for Postgres/K8s. Wrong unit of observation (service, not agent PID). |
| **Grafana Beyla / OBI** | eBPF auto-instrumentation → Prometheus / OTel RED metrics | No | Exporter, no UI of its own. HTTP/gRPC, not coding-agent semantics. |
| **Groundcover** | eBPF sensor, BYOC, per-node pricing, Agent Mode | No (has LLM obs via eBPF HTTP, K8s) | Platform for clusters. Not a laptop control room. |
| **Metoro** | K8s eBPF + autonomous AI SRE (detect → RCA → fix PR) | No | AI *doing* SRE on the cluster, not SRE *on* the AI agent. |
| **Odigos** | Auto OTel instrumentation manager | No | Adoption tool. |
| **Parca / Grafana Pyroscope** | Continuous profiling (eBPF / language) | No | Flamegraphs only. We export Perfetto; they are profilers. |
| **Cilium Hubble** | L4/L7 network flows | No | Network, not process story. |
| **Prometheus Node Exporter + Grafana** | Host counters every 15–30 s | No | No per-agent story, no 400 ms WS, needs a stack. |
| **Elastic APM / Dynatrace OneAgent** | Classic APM | Partial LLM add-ons | Enterprise agents, not a local binary. |

#### Runtime security

| Tool | What it is | vs DrishtiScope |
|---|---|---|
| **Falco** (CNCF graduated) | eBPF/kmod syscall rules → alerts. Huge community ruleset | Detection engine. Alert table, not an SRE story. Needs FalcoSidekick for UX. Root. |
| **Tetragon** (Cilium) | eBPF observe **and in-kernel enforce** (kill/block) | Security control plane. Dangerous if misconfigured. No golden-signal dashboard for agents. |
| **Tracee** (Aqua) | Forensic eBPF, Rego/Go signatures | Forensics/security. |
| **Sysdig Secure / Sysdig Inspect** | Commercial Falco + capture files. Inspect is a *post-hoc* syscall GUI (tiles, sub-second timelines, process trees) | Closest *UI ancestor* for “drill into syscalls.” Forensic captures, not a live 400 ms agent control room. Enterprise priced. |
| **Datadog Security / Wiz runtime** | Vendor eBPF security | Cloud suite. |

#### Kernel power-user tools (not products)

`bpftrace`, BCC, `perf`, `strace`, `lsof`, `pidstat`, `ss`, Perfetto UI, `htop`/`btop`/`glances`/`atop`.

DrishtiScope **wraps this world** (copyable commands on every metric, Perfetto export) rather than replacing it.

### 4.3 Layer C — actually about coding / AI *agent processes*

This is the short list a sophisticated buyer should put next to DrishtiScope.

| Tool | Mechanism | UX | Unique angle | Gap vs DrishtiScope |
|---|---|---|---|---|
| **AgentSight** ([eunomia-bpf/agentsight](https://github.com/eunomia-bpf/agentsight), arXiv:2508.02736) | eBPF + **SSL/TLS uprobe sniff** + process/file tracepoints. Also reads native Claude/Codex/Gemini session files | CLI (`top`, `record`, `report`, `vis`), local web on `:7395`, Agent Nebula GIF, token flamegraphs, OTel GenAI export | **Closest peer.** Zero-SDK, closed-source CLIs, correlates *prompts* with *syscalls*. Academic paper, <3% overhead. MIT | Deep capture wants **sudo**. Product is recorder/TUI-first, not an SRE golden-signal control room. **Reads TLS plaintext** (prompts, responses) — a feature and a privacy/compliance problem. No metric encyclopedia, no SLO burn, no Prometheus-first ops surface, no zero-root `/proc` production mode as the default story |
| **OpenLIT coding-agent mode** | Vendor CLI plugins (`openlit coding install --vendor=claude-code\|cursor\|codex`) + OTel | Dashboards for cost, tokens, lines added/removed, accept rate | Team spend / productivity across Claude, Cursor, Codex | Not kernel. Depends on vendor telemetry. No syscall P99, no FD leaks, no Perfetto |
| **cctop** (several projects share the name: [cctop.app](https://cctop.app/), flolep2607, DeanLa, stefanprodan, Sixeight) | Reads `~/.claude`, `~/.codex`, pty/unix sockets, menubar jump-to-session | TUI or macOS menubar. “Which session is waiting on you?” | Session *orchestration* and token burn | Session files ≠ kernel. No eBPF, no golden signals, no security audit |
| **CTOP** ([aakashadesara/ctop](https://github.com/aakashadesara/ctop)) | Process table + session files for Claude/Codex/OpenCode/Devin | TUI “htop for agents” | CPU/mem + tokens + cost in one pane | Same: no syscall tracing, no story tab, no exporters |
| **agentop** | File watchers on `~/.claude` + `~/.codex` + process scan | Multi-panel TUI | Timeline of session events | Disk logs, not kernel |
| **Helix** | Rust TUI + MCP memory server | Session cards + persistent memory | Memory across sessions | Not observability of host effects |
| **abtop** (graykode) | Rust TUI | Claude & Codex: tokens, context, rate limits, ports | Lightweight | Same class as cctop |
| **llm-slo-ebpf-toolkit** | K8s eBPF: DNS, TCP retransmit, scheduling, TLS handshake vs OTel traces | Toolkit, Bayesian attribution | SLO for *LLM services* (TTFT), not coding-agent PIDs | Cluster reliability engineering, different user |

**If you only remember one competitor: AgentSight.**  
Everything else in Layer C is a session-file TUI or a spend dashboard.

---

## 5. Head-to-head matrices

### 5.1 The slide that should be in every pitch

| Capability | **DrishtiScope** | AgentSight | LangSmith / Langfuse / Phoenix | Pixie / Coroot / Beyla | Falco / Tetragon | cctop / CTOP / OpenLIT coding | Datadog / New Relic |
|---|---|---|---|---|---|---|---|
| Primary object | **One agent PID + children** | One agent session + TLS | LLM span tree | K8s service | Threat event | Coding session files | Host / service / LLM span |
| Needs code change | **No** | No | Yes (SDK/proxy) | No | No | No (vendor files) | Agent + SDK |
| Closed-source CLI | **Yes** | Yes | No | Accidental | Accidental | Yes (if it logs) | No |
| Kernel syscalls | **eBPF + /proc** | eBPF | No | eBPF (HTTP-centric) | eBPF | No | Partial |
| Reads prompt text | **No (by design)** | **Yes (TLS sniff)** | Yes (app logged it) | Sometimes HTTP body | No | Sometimes from logs | Yes if instrumented |
| Process story / verdict | **Yes** | Timeline of events | Span waterfall | Service map | Alert | Session status | APM trace |
| SRE golden signals + SLO burn | **Yes** | Resource metrics | Latency/tokens | RED for HTTP | No | CPU/mem only | Yes, for services |
| Zero-root mode | **Yes (`/proc`)** | Fallback snapshots | n/a | No | No | Yes | No |
| Single binary + embedded UI + TSDB | **Yes (~18 MB)** | CLI + optional UI + SQLite | Cloud or multi-service | Cluster | Agent + sidekick | TUI | SaaS |
| Update cadence | **~400 ms WebSocket** | Live TUI / record | Seconds–minutes | In-cluster live / scrape | Event stream | 1 s TUI | 10–60 s |
| Prometheus `/metrics` | **Yes** | Limited | Via OTel | Yes | Via exporters | No | Native |
| Perfetto export | **Yes** | Flamegraphs / pprof | No | Profiles | No | No | Proprietary |
| Junior-admin `(?)` encyclopedia | **Yes** | No | Docs site | No | Rule docs | No | Product docs |
| Telemetry-aware copilot | **Yes (snapshot injected)** | Secondary LLM on traces | Playgrounds | AI RCA (Coroot/Metoro) | No | No | Bits AI / SRE Agent |
| Privacy / air-gap | **Local SQLite, no SaaS** | Local SQLite; payloads sensitive | Cloud or self-host stack | In-cluster | Local | Local | Cloud |
| Root required for full fidelity | Optional | Typical for `record` | n/a | Yes | Yes | No | Agent |

### 5.2 vs AgentSight (the only true peer)

Use this table when an engineer says “isn’t that AgentSight?”

| Dimension | AgentSight | DrishtiScope | Who wins |
|---|---|---|---|
| Philosophy | Boundary tracing: **intent (TLS) + effect (kernel)** | Effect-first SRE: **golden signals + story**, no payload sniff | Different products |
| Capture of prompts / completions | First-class (sslsniff / uprobes, including statically linked BoringSSL) | Not in scope | AgentSight, if you *want* prompts |
| Privacy / compliance | Captured DBs contain prompts, headers, paths — treat as secrets | No prompt bodies; files/sockets/syscalls only | **DrishtiScope** |
| Default UX | `agentsight top` TUI, `record` then `report serve` | Always-on dark control-room SPA | **DrishtiScope** for SREs; AgentSight for CLI natives |
| Narrative | Event log, process tree, nebula file-replay GIF | **Executive verdict + chronological story + copyable diagnostics** | **DrishtiScope** |
| SRE toolkit | Metrics view, OTel GenAI export | SLO, error budget, P50/P90/P99 syscall latency, Prometheus, healthz, Perfetto | **DrishtiScope** |
| Privilege | sudo for eBPF record | auto → eBPF, else live `/proc` | **DrishtiScope** for locked-down laptops |
| Teaching surface | Docs + flamegraphs | Metric Encyclopedia on every tile | **DrishtiScope** |
| Academic backing | arXiv + ACM paper | Product/engineering | AgentSight |
| Token / cost from intercepted traffic | Yes | No (would need session files or TLS) | AgentSight |
| Windows / macOS | Session-file mode; Windows CI; eBPF is Linux | Explicit Windows eBPF-for-Windows detection + Darwin synthetic engine | Tie (different fallbacks) |

**Coexistence pitch:** run AgentSight when you need to *audit what was sent to the model*. Run DrishtiScope when you need to *operate the process* (latency, saturation, stalls, FD leaks, security effects) without holding prompt contents.

### 5.3 vs Langfuse / LangSmith (what buyers Google first)

**Example setup they expect**

```python
# Langfuse — you own this process
from langfuse import observe

@observe()
def my_agent(user_msg: str) -> str:
    return llm.invoke(user_msg)
```

**DrishtiScope setup for a binary you do not own**

```bash
# no SDK, no decorator, no proxy
./drishtiscope --mode=auto --comm=codex
# open http://localhost:8080
```

| Question | Langfuse / LangSmith | DrishtiScope |
|---|---|---|
| “Did the prompt hallucinate?” | Yes (evals, scores) | No |
| “How many tokens / $ did this chain cost?” | Yes | No |
| “Why is Codex pegged at 180% CPU and 4k FDs?” | No | Yes |
| “Which files did the agent just open?” | Only if the tool span logged the path | Live `files_top` from the kernel / proc |
| “Is the TLS stream to the model gateway stalled?” | App-level latency if instrumented | Socket states + syscall P99, even if the CLI is closed source |
| “Can I run this on an air-gapped laptop in one binary?” | Self-host a stack | Yes |

### 5.4 vs Pixie / Coroot / Falco (the eBPF crowd)

| Question | Pixie | Coroot | Falco | DrishtiScope |
|---|---|---|---|
| Unit of observation | Cluster / HTTP request | Service / DB | Rule match | **Agent PID** |
| Install | Helm, privileged | Helm / node agent | DaemonSet + rules | **One binary** |
| “Tell me the story of PID 14221 (`codex`)” | You write a PxL script | You click a service | You get an alert if a rule fires | **Tab 1 is that story** |
| Junior admin | PxL, K8s fluency | Service maps | Falco rule syntax | `(?)` encyclopedia + copyable `pidstat` |
| Security enforcement | No | No | Alert (Falco) / kill (Tetragon) | Audit tab, no inline kill |

### 5.5 vs cctop / CTOP / OpenLIT coding (the “htop for Claude” crowd)

These tools answer: **which session is waiting, and what did it cost?**

DrishtiScope answers: **is the process healthy, stalled, leaking, or hostile — at the kernel?**

| Signal | cctop / CTOP | OpenLIT coding | DrishtiScope |
|---|---|---|---|
| Tokens, context %, $ | Yes | Yes (team rollup) | No |
| “Waiting on you” | Yes (core UX) | Session list | Indirect (idle verdict) |
| Syscall P99, error budget | No | No | Yes |
| Child `execve` of `docker build` | No | Tool span if vendor logged it | Process tree + story events |
| Works if the vendor stops writing session JSON | No | No | Yes (kernel does not need their cooperation) |

---

## 6. Worked examples — same incident, four answers

These are the slides that make the USP concrete.

### Example 1 — “Codex is slow”

A developer says Codex has been “thinking” for 90 seconds.

| Tool | What you see | What you still don’t know |
|---|---|---|
| **LangSmith** | Nothing. Codex is not a LangChain app you instrumented. | Everything. |
| **cctop** | Session is `busy`, context 61%, last tool `shell`. | Whether it is compiling, waiting on TLS, or spinning. |
| **AgentSight** | TLS shows a long streaming completion; file events show `/tmp/build`. | SLO, error budget, P99 syscall latency as first-class SRE numbers. |
| **DrishtiScope** | Verdict: *High I/O & Socket Activity*. P99 syscall latency 12 ms (warn). `ESTABLISHED` to the model gateway. `files_top` shows a 400 MB object file. Copilot: “saturation is disk, not the model — `iotop -p <pid>`.” Copy `pidstat -d 1 5 -p <pid>`. | Prompt text (intentionally). Token $ (use cctop/OpenLIT beside it). |

**USP in one line:** we tell you *which subsystem* is slow, in SRE language, on a binary you did not write.

---

### Example 2 — “Did the agent touch secrets?”

An agent was allowed to run `bash`. Security wants an audit.

| Tool | Result |
|---|---|
| **Langfuse** | A tool span `run_terminal_cmd` with the command string *if* the harness logged it. No proof of what the kernel allowed. |
| **Falco** | Alert: `Write below etc` / `Read sensitive file` — if a rule is loaded. No golden-signal context, no story of the last 5 minutes. |
| **Tetragon** | Can **kill** the process on `security_file_open` of `/etc/shadow`. That is enforcement, not an observability product. |
| **AgentSight** | `report audit --json` lists file opens and spawns; TLS may show the prompt that caused it. |
| **DrishtiScope** | Security tab: `EACCES` on `/etc/shadow`, refused outbound connect, radar vs baseline daemons. Story tab lists the file touches and endpoints. Data never left the box. |

**USP in one line:** independent kernel audit of a closed-source agent, without becoming a blocking security product.

---

### Example 3 — “The agent looks idle but the fan is spinning”

| Tool | Result |
|---|---|
| **htop** | `codex` at 97% CPU. No idea if that is token decode, a runaway regex, or a stuck Go runtime. |
| **LangSmith** | No data. |
| **Phoenix** | No data unless you wrapped the runtime. |
| **DrishtiScope** | Tab 3 call tree: `runtime.epollwait` vs `regex.Match` vs `syscall.Syscall6`. Tab 2: Traffic RPS high, error ratio 0, saturation CPU 97% → verdict *Active Code Generation*. Perfetto export opens in `ui.perfetto.dev` with syscall + file + socket lanes. Encyclopedia `(?)` on CPU: healthy 5–60%, critical >95%, verify with `pidstat -u 1 5 -p <PID>`. |

**USP in one line:** we translate a hot PID into an execution story a junior admin can verify with stock Linux tools.

---

### Example 4 — “Laptop, no root, WSL2, 5 minutes”

| Tool | Friction |
|---|---|
| Pixie / Coroot / Groundcover | You do not have a cluster. |
| Falco | Needs root + rules + a sink. |
| Langfuse Cloud | You will upload traces of an internal agent. Legal says no. |
| AgentSight `record` | Wants sudo for eBPF; TLS capture may be out of policy. |
| **DrishtiScope** | `./drishtiscope --mode=auto --comm=agy` → `/proc` fallback, localhost UI, SQLite on disk, copilot works without an API key (rule engine). |

**USP in one line:** the only Layer-C tool that is a realistic **default** on a locked-down developer laptop.

---

### Example 5 — “Wire this into the existing ops stack”

```
DrishtiScope :8080
   ├── /                 live SPA (Story / Vitals / CPU / Telemetry / Audit)
   ├── /metrics          Prometheus scrape (OpenMetrics)
   ├── /healthz /livez /readyz
   ├── /api/v1/traces/perfetto   → ui.perfetto.dev
   ├── /api/v1/logs              structured LogEntry[]
   └── /api/chat                 copilot with live snapshot
```

Langfuse needs ClickHouse. Pixie stays in-cluster. cctop prints a TUI. Datadog wants a tenant.

**USP in one line:** drop-in scrape targets and probes, no new vendor backend.

---

## 7. What DrishtiScope is *not* (stay honest)

A USP that pretends to replace Langfuse will lose to Langfuse.

| Job | Use instead | Why |
|---|---|---|
| Prompt evals, datasets, LLM-as-judge | Langfuse, Phoenix, Braintrust, DeepEval | We do not score outputs |
| Token accounting and $ / developer | OpenLIT coding, cctop, Helicone | We do not parse vendor usage logs or TLS bodies |
| “Which Claude pane is waiting on me?” | cctop.app / CTOP | That is a session switcher |
| Cluster-wide HTTP APM | Pixie, Coroot, Groundcover, Datadog | We observe selected PIDs, not the mesh |
| Block a syscall in-kernel | Tetragon | We are observe-only |
| Capture the actual prompt | AgentSight, or an SDK tracer | Privacy choice |
| SIEM / SOC2 evidence warehouse | Falco + SIEM, Sysdig Secure | We are a host console with an audit tab |
| Continuous fleet profiling | Parca, Pyroscope | We export Perfetto for one target |

**Recommended pairing (the “and” slide):**

```
         ┌─────────────┐
         │  Langfuse   │  quality, prompts, evals   (if you own the app)
         └──────┬──────┘
                │
   ┌────────────┼────────────┐
   │            │            │
cctop      DrishtiScope   Falco/Tetragon
sessions   kernel SRE     runtime policy
& $        of the PID     (optional)
   │            │
   └──── AgentSight ────┘
         when you must
         see TLS payloads
```

DrishtiScope’s job in that picture: **the missing SRE pane on the agent process.**

---

## 8. Pitches and slide copy

### 8.1 15 seconds (elevator)

> LangSmith shows you the prompt. htop shows you a hot PID. DrishtiScope shows you the *process story* of a closed-source AI agent — live from the kernel, no SDK, one binary.

### 8.2 60 seconds (demo intro)

> Autonomous agents don’t fail like microservices. They spawn shells, stream tokens, rewrite your tree, and stall on sockets — and the popular “LLM observability” tools only see what the SDK logged. We attach eBPF (or plain `/proc` if you have no root) to `codex`, Claude, Copilot, Grok, whatever is on the box. In 400 milliseconds you get golden signals, an executive verdict, the files and endpoints it touched, Perfetto traces, Prometheus metrics, and a copilot that already has the snapshot. Nothing leaves the machine. We never sniff your prompts.

### 8.3 Positioning statement (classic)

> **For** SREs, platform engineers, and agent developers who run closed-source or rapidly-changing coding agents on Linux  
> **who** need to know whether the agent is healthy, stalled, leaking, or touching the wrong files  
> **DrishtiScope** is a single-binary, kernel-grounded observability console  
> **that** turns `/proc` and eBPF into an SRE control room with a process story, golden signals, and a telemetry-aware copilot  
> **unlike** LangSmith/Langfuse (SDK traces of prompts), Pixie/Falco (generic cluster eBPF), or cctop (session-file TUIs)  
> **and unlike** AgentSight, we observe *effects* rather than intercepting TLS, and we ship a production dashboard instead of a recorder.

### 8.4 Three-bullet slide

1. **See closed-source agents from the outside.** No SDK, no proxy, no cooperation from the vendor.
2. **Speak SRE, not span trees.** Golden signals, SLO burn, Perfetto, Prometheus, 400 ms live.
3. **Stay local and teachable.** SQLite on disk, `(?)` encyclopedia on every metric, copilot with the snapshot, optional zero-root.

### 8.5 Anti-positioning (what not to say)

| Don’t claim | Because |
|---|---|
| “We replace Langfuse” | We don’t do evals or prompt management |
| “We are the only eBPF agent observer” | AgentSight exists and is strong |
| “We capture prompts without instrumentation” | We deliberately don’t; that’s AgentSight |
| “Enterprise SIEM” | We are a host console |
| “Zero overhead” | eBPF is low, not free; `/proc` scraping has a tick cost |

### 8.6 Name, category, proof points

| Item | Copy |
|---|---|
| **Name** | DrishtiScope — दृष्टि (insight, clear seeing) + Scope |
| **Category label** | Agentic process observability / Kernel SRE for AI agents |
| **Not the category** | LLMOps, prompt management, APM, CNAPP |
| **Proof of “production-shaped”** | `/metrics`, `/healthz`, `/livez`, `/readyz`, Perfetto, structured logs, SQLite WAL, GoReleaser multi-arch, Docker/GHCR |
| **Proof of “agent-native”** | Auto-labels Codex, Antigravity (`agy`), Copilot, Node/Python workers; hot-switch PID; story verdicts |
| **Proof of “teachable”** | Metric Encyclopedia: analogy, bands, why-agents-care, verification command, kernel source |

---

## 9. Sources (competitive research)

Roundups and indexes consulted 13 Sep 2026:

- Aldric Research, *AI Observability & Evaluation Platforms (2026)* — LangSmith, Braintrust, Phoenix, Langfuse, Helicone, W&B Weave
- Aiprosol, *LLM Observability & Eval Index (2026)*
- TwoTail, *Best agent observability tools in 2026*
- morphllm, *AI Agent Observability Tools (2026): 12 platforms*
- PostHog, *The best AI observability tools, compared* (updated Jun 2026)
- GitHub topic `llm-observability` and [awesome-agentops-landscape](https://github.com/dyronrh/awesome-agentops-landscape)
- Metoro, *Top 8 eBPF Observability Tools in 2026* — Metoro, Coroot, Pixie, Anteon, Beyla, Odigos, Pyroscope, Parca
- NomadX / Decryption Digest — Falco vs Tetragon vs Tracee (2026)
- AgentSight product docs: [eunomia.dev/agentsight](https://eunomia.dev/agentsight/) and [github.com/eunomia-bpf/agentsight](https://github.com/eunomia-bpf/agentsight)
- Zheng et al., *AgentSight: System-Level Observability for AI Agents Using eBPF*, arXiv:2508.02736
- OpenLIT docs — coding-agent observability for Claude Code, Cursor, Codex
- cctop.app, CTOP, agentop, Helix, abtop project READMEs
- Datadog, *Monitor, troubleshoot, and improve AI agents*; New Relic AI Observability (June 2026)
- Grafana Beyla documentation; Percona Coroot Edition; Groundcover Flora/eBPF
- Sysdig Inspect (open-source syscall drill-down UI) — UI ancestor, not a competitor in the agent category

---

## 10. One-page leave-behind

```
DRISHTISCOPE
Kernel SRE for autonomous AI agents

THE PROBLEM
  LLM tracers see prompts. Cluster eBPF sees HTTP. Session TUIs see ~/.claude.
  Nobody operates the live Linux process of a closed-source coding agent.

THE PRODUCT
  One ~18 MB binary. eBPF when you have it, /proc when you don't.
  400 ms WebSocket control room:
    Story → Vitals (golden signals + SLO) → CPU/Perfetto → MQL → Audit
  Copilot that already holds the snapshot. Encyclopedia on every metric.
  Prometheus, healthz, Perfetto, SQLite. Nothing leaves the machine.

THE MOAT (what we will not give up)
  1. Closed-source attach, zero SDK
  2. Process story + verdict (not a span tree, not an alert)
  3. Golden signals on the agent PID
  4. Zero-root real mode
  5. Effects, not TLS payloads
  6. Single binary, embedded TSDB
  7. Teachable (junior-admin encyclopedia)

TRUE PEER
  AgentSight — use it when you must read prompts off the wire.
  Use us when you must run the process like an SRE.

NOT PEERS (common confusion)
  LangSmith / Langfuse / Phoenix     → own the app, score the prompt
  Pixie / Coroot / Falco             → cluster / security, not agent PID
  cctop / CTOP / OpenLIT coding      → sessions and dollars, not syscalls
```

---

*End of USP brief. Refresh the catalog when Langfuse, AgentSight, or OpenLIT ship a kernel control room — that is the only move that collapses this positioning.*
