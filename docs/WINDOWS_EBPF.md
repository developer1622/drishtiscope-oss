# Microsoft eBPF for Windows Integration Guide

DrishtiScope provides cross-platform telemetry observability across Linux, Windows, and macOS. This document outlines how DrishtiScope integrates with **[Microsoft eBPF for Windows](https://github.com/microsoft/ebpf-for-windows)**.

> [!NOTE]
> **Universal Process Monitoring on Windows**: On Windows 10/11 and Windows Server, DrishtiScope can monitor **any active process** (e.g. `powershell.exe`, `node.exe`, `python.exe`, or custom backend services) using native Win32 process scanning (`tasklist.exe`), with optional kernel probe support when the Microsoft eBPF for Windows driver is installed.

---

## 1. Architectural Overview

```
                     ┌──────────────────────────────────────────────┐
                     │          DrishtiScope UI (React/TS)          │
                     │  Process Signals • Memory • Net • Files      │
                     └──────────────────────▲───────────────────────┘
                                            │ WebSocket JSON (:8080/ws)
                     ┌──────────────────────┴───────────────────────┐
                     │       DrishtiScope Go Engine (Pure Go)       │
                     │      CGO_ENABLED=0 • modernc.org/sqlite      │
                     └──────▲───────────────────────────────▲───────┘
                            │                               │
               Windows Driver Detection               Fallback Engine
                            │                               │
        ┌───────────────────┴──────────────────┐    ┌───────┴──────────────┐
        │       \\.\EbpfCoreDevice             │    │ Multi-Profile Mock   │
        │    Windows eBPF Core Driver          │    │ python / node / app  │
        └───────────────────▲──────────────────┘    └──────────────────────┘
                            │
        ┌───────────────────┴──────────────────┐
        │      ebpfcore.sys / NetEbpfExt       │
        │   Microsoft eBPF for Windows Engine   │
        └──────────────────────────────────────┘
```

### Component Breakdown
1. **Windows Platform Driver Interface (`agent_windows.go`)**:
   - Built with the `//go:build windows` build constraint.
   - At startup, probes for the presence of the Microsoft eBPF execution driver via `CreateFile` on `\\.\EbpfCoreDevice`.
   - Queries the Windows Service Control Manager (`sc.exe query ebpfcore`) to confirm `ebpfcore.sys` service state.
2. **Graceful Dual-Mode Fallback**:
   - When the eBPF drivers and execution device are active, attaches to kernel/driver hooks.
   - When the driver is not installed or execution privileges are limited, safely falls back to DrishtiScope's Real Engine (via `tasklist.exe` and ETW bridge) or synthetic telemetry engine while notifying the dashboard via WebSocket metadata.
3. **CGO-Free Cross-Compilation**:
   - Built with `CGO_ENABLED=0` to ensure true zero-dependency cross-compilation from Linux/CI hosts to Windows (`windows/amd64`, `windows/arm64`) and macOS (`darwin/amd64`, `darwin/arm64`).
   - SQLite TSDB utilizes `modernc.org/sqlite` (pure Go implementation of SQLite), completely removing external C compiler dependencies.

---

## 2. Installing Microsoft eBPF for Windows (Optional)

To run with real eBPF kernel hooks on Windows 10/11 or Windows Server 2022+:

### Step 1: Install eBPF for Windows MSI Release
Download and install the official signed MSI from [microsoft/ebpf-for-windows releases](https://github.com/microsoft/ebpf-for-windows/releases):

```powershell
# In an Administrative PowerShell:
Invoke-WebRequest -Uri https://github.com/microsoft/ebpf-for-windows/releases/download/v0.17.1/ebpf-for-windows.msi -OutFile ebpf-for-windows.msi
Start-Process msiexec.exe -ArgumentList '/i ebpf-for-windows.msi /quiet /qn /norestart' -Wait
```

### Step 2: Verify Driver Status
```powershell
# Check driver service status
sc.exe query ebpfcore
sc.exe query netebpfext

# Verify device node creation
Get-Item -LiteralPath "\\.\EbpfCoreDevice" -ErrorAction SilentlyContinue
```

### Step 3: Run DrishtiScope
Extract the Windows release archive from GoReleaser (`drishtiscope_windows_amd64.zip`):

```powershell
.\drishtiscope.exe --addr 127.0.0.1:8080 --comm powershell
```

---

## 3. GoReleaser Multi-Platform Build Pipeline

DrishtiScope packages releases across 6 target platforms using [GoReleaser](https://goreleaser.com):

| Operating System | Architecture | Binary Name | Archive Format |
| :--- | :--- | :--- | :--- |
| **Linux** | `amd64` | `drishtiscope` | `.tar.gz` |
| **Linux** | `arm64` | `drishtiscope` | `.tar.gz` |
| **Windows** | `amd64` | `drishtiscope.exe` | `.zip` |
| **Windows** | `arm64` | `drishtiscope.exe` | `.zip` |
| **macOS (Darwin)** | `amd64` | `drishtiscope` | `.tar.gz` |
| **macOS (Darwin)** | `arm64` (Apple Silicon) | `drishtiscope` | `.tar.gz` |

### Reproducing Releases Locally
```bash
# Check configuration
goreleaser check

# Build release artifacts snapshot
goreleaser build --snapshot --clean
```

---

## 4. Test Coverage Standard

Every Go package in DrishtiScope strictly maintains **>90% automated statement test coverage**:

```bash
go test -cover ./...
```
- `cmd/agentscope`: **93.9%**
- `internal/agg`: **97.8%**
- `internal/api`: **96.0%**
- `internal/audit`: **100.0%**
- `internal/config`: **94.0%**
- `internal/ebpfagent`: **91.7%**
- `internal/enrich`: **95.3%**
- `internal/hub`: **94.0%**
- `internal/mock`: **97.4%**
- `internal/protocol`: **92.3%**
- `internal/storage`: **90.8%**
