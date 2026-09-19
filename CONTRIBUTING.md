# Contributing to DrishtiScope (दृष्टिScope)

Thank you for your interest in contributing to **DrishtiScope**! DrishtiScope is an open-source, high-performance, real-time observability platform designed to monitor, trace, and understand **any running process** on Linux and Windows (e.g. backend web services, compilers, developer CLI utilities, background workers, and AI agent runtimes) at the operating system level without requiring code changes to the target process.

We welcome contributions from developers, Linux systems engineers, kernel enthusiasts, site reliability specialists, and open-source contributors worldwide.

---

## 🧭 Table of Contents
1. [Code of Conduct](#-code-of-conduct)
2. [Architecture Overview](#-architecture-overview)
3. [Development Environment Setup](#-development-environment-setup)
4. [Running DrishtiScope Locally](#-running-drishtiscope-locally)
5. [Frontend Development Guidelines](#-frontend-development-guidelines)
6. [Backend Development Guidelines](#-backend-development-guidelines)
7. [Adding New Metrics to the Metric Encyclopedia](#-adding-new-metrics-to-the-metric-encyclopedia)
8. [Automated Testing & Coverage (>90%)](#-automated-testing--coverage-90)
9. [Submitting a Pull Request](#-submitting-a-pull-request)
10. [Release Engineering with GoReleaser](#-release-engineering-with-goreleaser)

---

## 🤝 Code of Conduct
We are dedicated to providing a welcoming, inclusive, and harassment-free environment for everyone. Please treat all contributors and maintainers with kindness, respect, and empathy.

---

## 🏛️ Architecture Overview

```
                      ┌─────────────────────────────────┐
                      │        Target Process           │
                      │  (e.g., node, python, worker)   │
                      └──────────────┬──────────────────┘
                                     │
         ┌───────────────────────────┴──────────────────────────┐
         │                                                      │
         ▼                                                      ▼
  [Linux Kernel Space]                                  [Userspace /proc]
  eBPF Tracepoints:                                     /proc/[pid]/stat, status,
  • raw_syscalls:sys_enter/sys_exit                      statm, io, fd, net/tcp
  • sched_process_exec / sched_process_exit              Real process scanning
         │                                                      │
         └───────────────────────────┬──────────────────────────┘
                                     │
                                     ▼
                     ┌───────────────────────────────┐
                     │    DrishtiScope Go Backend    │
                     │    (:8080 HTTP & WebSocket)   │
                     ├───────────────────────────────┤
                     │ • Real-time Sliding Window    │
                     │ • Embedded SQLite TSDB (WAL)  │
                     │ • Prometheus /metrics         │
                     │ • Perfetto & Structured Logs  │
                     │ • Copilot Diagnostic API      │
                     └──────────────┬────────────────┘
                                    │ WebSocket (400ms ticks)
                                    ▼
                     ┌───────────────────────────────┐
                     │   DrishtiScope React SPA      │
                     │   (TypeScript + Tailwind)     │
                     ├───────────────────────────────┤
                     │ 1. Process Story (Activity)   │
                     │ 2. Overview (Signals & SLO)   │
                     │ 3. Execution & CPU (Traces)   │
                     │ 4. System Metrics (Telemetry) │
                     │ 5. Security & Logs (Audit)    │
                     │ • Metric Help Encyclopedia    │
                     │ • Interactive Copilot Drawer  │
                     └───────────────────────────────┘
```

---

## 🛠️ Development Environment Setup

### Prerequisites
- **Go**: Version 1.27 or newer (`go version`)
- **Node.js**: Version 20 or newer (`node -v`)
- **npm**: Version 9 or newer (`npm -v`)
- **Linux** (native or WSL2 on Windows 10/11) with kernel headers for eBPF compilation (optional for mock/proc modes)
- **GoReleaser**: v2.0+ (optional, for release builds)

---

## 🚀 Running DrishtiScope Locally

### 1. Clone Repository
```bash
git clone https://github.com/developer1622/drishtiscope-oss.git
cd drishtiscope-oss
```

### 2. Run Backend
In one terminal:
```bash
cd backend
go run ./cmd/agentscope
```
The server will start on `http://127.0.0.1:8080`.
By default, it automatically scans for running processes (such as `python`, `node`, `my-service`) and collects live kernel vitals.

### 3. Run Frontend
In a second terminal:
```bash
cd frontend
npm install
npm run dev
```
Open `http://localhost:5173` in your browser. The Vite dev server will connect to `ws://127.0.0.1:8080/ws` with real-time updates.

---

## 💻 Frontend Development Guidelines

- **Tech Stack**: React 18, TypeScript, Tailwind CSS, Recharts, Lucide Icons, Zustand.
- **Anti-Flicker Mode**: Ensure all Recharts components use `isAnimationActive={!antiFlicker}` (retrieved from `useScopeStore`) to prevent UI stutter and eye-strain during 400ms streaming updates.
- **Type Safety**: No `any` types. Run `npx tsc --noEmit` before submitting.
- **Tone & Scope**:
  - DrishtiScope is an open-source, vendor-neutral tool.
  - Avoid mentioning specific commercial products or comparing against vendor tools in UI copy.
  - The tool is dedicated to **Universal Process Observability** across any Linux or Windows workload.
- **Process Hierarchy**:
  - Always allow users to click any process in the hierarchy or process table and toggle it as the active target using `setSelectedPid()` and `setTarget()`.

---

## ⚙️ Backend Development Guidelines

- **Package Structure**:
  - `cmd/agentscope/main.go`: Entry point, lifecycle management, graceful shutdown.
  - `internal/api/`: HTTP endpoints (`/api/snapshot`, `/api/target`, `/api/chat`, `/api/history`, `/metrics`, `/healthz`).
  - `internal/agg/`: Ring buffers, exponential moving averages, snapshot aggregation.
  - `internal/enrich/`: `/proc` parsers (`stat`, `statm`, `io`, `fd`, `net/tcp`).
  - `internal/mock/realproc.go`: Real process inspector for running application environments.
  - `internal/ebpfagent/`: Cilium eBPF probe loader and ringbuf reader.
  - `internal/storage/`: SQLite TSDB engine with WAL checkpointing.
- **Code Style**:
  - Follow standard Go idioms (`gofmt -s -w .`, `go vet ./...`).
  - Keep allocations on the hot path minimal.
  - Never swallow errors silently; log or handle them properly.

---

## ❓ Adding New Metrics to the Metric Encyclopedia

DrishtiScope features a built-in, engineer-friendly **Metric Encyclopedia Modal** accessible via colorful question mark badges throughout the dashboard.

To add or update metric explanations:
1. Open `frontend/src/data/metricDocs.ts`.
2. Add an entry conforming to the `MetricDoc` interface:
   ```typescript
   export interface MetricDoc {
     id: string;
     title: string;
     category: 'Core Agent Signals' | 'Compute & CPU' | 'Memory & RAM' | 'Storage & Disk' | 'Network & Sockets' | 'Processes & Threads' | 'Security & Audit' | 'Kernel & eBPF';
     themeColor: 'cyan' | 'blue' | 'emerald' | 'amber' | 'rose' | 'purple';
     shortDefinition: string;
     juniorAdminExplanation: string; // Plain-English analogy for engineers
     howToRead: {
       unit: string;
       healthy: string;
       warning: string;
       critical: string;
     };
     whyAiAgentsCare: string; // Operational impact on the process and system
     linuxAdminCommand: string; // Copyable Linux command to verify independently
     kernelDataSource: string; // Kernel file or eBPF probe source
   }
   ```
3. Mount a `<MetricHelpButton metricId="your_metric_id" />` adjacent to the KPI card or chart.

---

## 🧪 Automated Testing & Coverage (>90%)

Every PR must pass existing tests and maintain high test coverage:

### Running Backend Tests
```bash
cd backend
go test -v -cover ./...
```
To generate HTML coverage reports:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Building Frontend
```bash
cd frontend
npm run build
```

### Running E2E Playwright Tests

DrishtiScope ships a full Playwright E2E suite covering all 5 dashboard tabs, the metric help encyclopedia, the copilot drawer, the process search omnibox, and telemetry mode switching.

```bash
# Start the backend in mock mode (for reproducible tests)
cd backend && go run ./cmd/agentscope -mode=mock -addr=127.0.0.1:8080 &

# In a separate terminal, run Playwright
cd frontend
npm run test:e2e
```

To view an interactive HTML test report after a run:
```bash
npx playwright show-report
```

The CI pipeline also runs Playwright automatically on every PR via the `playwright` job in `.github/workflows/ci.yml`.

---

## 🚢 Submitting a Pull Request

1. Fork the repository on GitHub.
2. Create a feature branch: `git checkout -b feature/process-memory-tracing`.
3. Commit your changes with clear, descriptive commit messages:
   - `feat: add thread pool latency histogram to StoryTab`
   - `fix: prevent layout shift on high-frequency socket reconnect`
4. Run tests and verify builds pass locally:
   ```bash
   cd backend && go test ./...
   cd ../frontend && npm run build
   ```
5. Push to your fork and submit a PR to `main`.
6. Maintainers will review your PR and provide constructive feedback.

---

## 📦 Release Engineering with GoReleaser

DrishtiScope uses [GoReleaser](https://goreleaser.com) for cross-compiling releases for Linux, Windows, and macOS (amd64 and arm64).

To verify the release pipeline locally:
```bash
# Verify configuration
goreleaser check

# Dry-run snapshot build for all 6 platforms
goreleaser build --snapshot --clean
```

---

Thank you for helping improve DrishtiScope and contributing to open-source systems observability!
