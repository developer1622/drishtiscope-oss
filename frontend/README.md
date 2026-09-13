# DrishtiScope Frontend Console

The **DrishtiScope Frontend** is a modern, high-density observability console built with **React 18**, **TypeScript**, **Tailwind CSS**, **Recharts**, and **Zustand**. It provides sub-second live telemetry rendering, interactive metric help popups, an AI copilot chat drawer, and 5 ergonomic color themes with Light Mode as default.

---

## 🏛️ Directory Structure & Key Modules

```
frontend/
├── src/
│   ├── components/
│   │   ├── Header.tsx              # Single straight-line header with Omnibox, Theme, Rate, and Stream controls
│   │   ├── StatusBar.tsx           # Sticky bottom bar with dropped events, ingest rate, motto & kernel info
│   │   ├── MetricHelpModal.tsx     # Popup modal explaining metrics with plain English analogies & verification CLI commands
│   │   ├── GraphScratchpad.tsx     # Dynamic graph scratchpad & Linux playground for on-demand metric curves
│   │   ├── ChatDrawer.tsx          # Floating AI Observability Copilot chat drawer with live snapshot context
│   │   ├── ModeChip.tsx            # Pulsing status chip: REAL LIVE (cyan), EBPF LIVE (green), or MOCK (amber)
│   │   ├── KPITile.tsx             # High-density vital tiles with SVG sparklines and threshold indicators
│   │   ├── ProcessTable.tsx        # Real-time process listing with CPU%, RSS, state, FDs, and "Set Target" action
│   │   ├── TabBanner.tsx           # Educational banner at the top of each tab explaining its purpose
│   │   └── tabs/
│   │       ├── StoryTab.tsx        # Tab 1: Chronological process story, AI verdict, and one-click Linux diagnostics
│   │       ├── BasicTab.tsx        # Tab 2: Overview of SRE Golden Signals, latency quantiles, and waveform
│   │       ├── MediumTab.tsx       # Tab 3: Execution call trees, continuous flamegraph profiler, Perfetto export
│   │       ├── AdvancedTab.tsx     # Tab 4: Metrics Query Language (MQL) console, subsystem donuts, disk IOPS
│   │       └── CompletePictureTab.tsx # Tab 5: Security workload radar, permission denial audit, system-wide tree
│   ├── pages/
│   │   └── Dashboard.tsx           # Main application view assembling Header, Ribbon, Tabs, Scratchpad, and StatusBar
│   ├── store/
│   │   └── useScopeStore.ts        # Central Zustand state: snapshot, events, connection, theme, tab, scratchpad
│   ├── data/
│   │   └── metricDocs.ts           # Metric Encyclopedia database: analogies, thresholds, CLI commands, kernel source
│   ├── styles/
│   │   └── globals.css             # CSS custom properties for 5 themes (Light, Dark, Ubuntu, Unix, Purple)
│   ├── types/
│   │   └── protocol.ts             # TypeScript interfaces mirroring Go WebSocket & REST protocol
│   └── ws/
│       └── client.ts               # Resilient WebSocket client with exponential backoff and heartbeat monitoring
├── index.html                      # HTML root with localized JetBrains Mono and Inter fonts
├── tailwind.config.ts              # Tailwind CSS configuration mapping design-token CSS variables
└── vite.config.ts                  # Vite build configuration with proxy rules for backend API
```

---

## 🎨 The 5 Theme System (Light Mode Default)

DrishtiScope ships with 5 themes designed for diverse lighting environments:

1. **☀️ Light Mode (Default)**:
   - High-contrast day palette designed for comfortable reading in bright offices.
   - Clean slate borders (`#e2e8f0`), neutral backgrounds (`#f8fafc`), dark navy text (`#0f172a`).
2. **🌙 Dark Mode**:
   - Classic nocturnal control room aesthetic with obsidian panels (`#0e1118`) and cyan/emerald highlights.
3. **🟠 Ubuntu Mode**:
   - Warm Canonical aubergine (`#2c001e`) and dark mahogany tones accented with Ubuntu orange (`#dd4814`).
4. **📟 Unix Mode**:
   - Retro 1980s green-screen CRT terminal aesthetic with glowing phosphor accents (`#00ff66`).
5. **🔮 Purple Mode**:
   - Cyberpunk synthwave palette with deep violet backgrounds (`#0d0b18`) and neon purple accents (`#a855f7`).

Theme selection is persistent in `localStorage` and switchable instantly via the header dropdown.

---

## ⚡ Anti-Flicker & Stream Control

High-frequency telemetry streams (e.g. 200ms–400ms updates) often cause screen flicker and eye strain. DrishtiScope solves this with:

- **Stream Refresh Rate Dropdown**: Choose between `500ms` (Rapid), `1s` (Balanced), `2s` (Calm default), `5s` (Relaxed), or `Manual` (Pause live updates).
- **Anti-Flicker Smooth Toggle**:
  - In **Smooth Mode**, rapid numerical jitter is damped, pulse animations are suppressed, and visual transitions are smoothed.
  - In **Rapid Mode**, every tick is rendered instantaneously for raw micro-profiling.
- **Stream Freeze / Live Button**: Freeze the dashboard at any millisecond to inspect and copy state without data advancing.

---

## ❓ Metric Encyclopedia `(?)`

Every single KPI tile, chart header, and stream control features a colorful question mark glyph `(?)`. Clicking it opens the **Metric Help Modal**:
- **Plain English Analogy**: Explains the metric in intuitive terms (e.g., comparing runqueue latency to a grocery store checkout line).
- **Healthy / Warning / Critical Bands**: Specific numerical thresholds.
- **Why AI Agents Care**: Concrete impact on LLM loops, token generation, or tool calls.
- **Linux Verification Command**: Pre-formatted terminal commands (`pidstat`, `strace`, `ss`, `lsof`) with one-click copy.
- **Kernel Source Location**: Exact Linux kernel file where the metric originates (e.g. `kernel/sched/core.c`, `/proc/[pid]/io`).

---

## 📈 Dynamic Graph Scratchpad & Linux Playground

Located at the bottom of the dashboard, the **Graph Scratchpad** lets engineers dynamically mount custom charts:
- Pre-built presets:
  - *Syscall Latency Quantile Curve (P50 vs P90 vs P99)*
  - *Context Switches vs Runqueue Latency*
  - *Bidirectional Network Throughput (Tx vs Rx)*
  - *Thread Count vs RSS Memory Usage*
  - *Page Fault Dynamics (Minor vs Major Faults)*
- Dynamically triggered by the AI Copilot when users ask to visualize specific correlations.

---

## 🤖 AI Observability Copilot Chat Drawer

Clicking the floating bot icon in the bottom-right corner slides open the **Agent Copilot**:
- **Local Rule Engine**: Evaluates live snapshot metrics and diagnoses performance bottlenecks with zero API keys.
- **Natural Language Analysis**: With an optional API key configured (`GEMINI_API_KEY`, `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`), the Copilot analyzes memory growth, file descriptor leaks, or network stalls in context.

---

## 🧪 Development & Testing

```bash
cd frontend

# Install dependencies
npm ci

# Start development server with HMR
npm run dev

# Run TypeScript type check
npx tsc --noEmit

# Build production bundle
npm run build

# Run Playwright E2E and visual regression suite (from repository root)
NODE_PATH=frontend/node_modules node tests/ui_audit.cjs
```
