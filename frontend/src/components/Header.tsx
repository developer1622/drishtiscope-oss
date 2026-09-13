import React, { useState } from 'react';
import { ModeChip } from './ModeChip';
import { HelloPayload, ConnectionStatus, ProcessRow } from '../types/protocol';
import { agentLabel } from '../utils/format';
import { useScopeStore, TabType, ThemeMode } from '../store/useScopeStore';
import { apiHeaders } from '../utils/api';
import { MetricHelpButton } from './MetricHelpModal';
import {
  Search,
  Server,
  Crosshair,
  Sun,
  Moon,
  Layers,
  Activity,
  Cpu,
  ShieldCheck,
  Pause,
  Play,
  Download,
  ChevronRight,
  Terminal,
  RefreshCw,
  Zap,
  Shield,
  Bot,
  Palette,
} from 'lucide-react';

export function Header({
  mode,
  hello,
  status,
  targetLabel,
  processes = [],
  onTargetSelect,
}: {
  mode: 'ebpf' | 'real' | 'mock' | null;
  hello: HelloPayload | null;
  status: ConnectionStatus;
  targetLabel?: string;
  processes?: ProcessRow[];
  onTargetSelect?: (comm: string, pid?: number) => void;
}) {
  const [targetInput, setTargetInput] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const {
    theme,
    setTheme,
    activeTab,
    setActiveTab,
    isPaused,
    togglePause,
    timeRange,
    setTimeRange,
    refreshRate,
    setRefreshRate,
    antiFlicker,
    toggleAntiFlicker,
    setSnapshot,
  } = useScopeStore();

  const [loadingSnap, setLoadingSnap] = useState(false);

  const loadSnapshot = async () => {
    setLoadingSnap(true);
    try {
      const res = await fetch('/api/snapshot', { headers: apiHeaders() });
      if (res.ok) {
        const data = await res.json();
        if (data && data.payload) {
          setSnapshot(data.payload);
        }
      }
    } catch (e) {
      console.error('Snapshot error:', e);
    } finally {
      setTimeout(() => setLoadingSnap(false), 300);
    }
  };

  const applyTarget = async (comm: string, pid?: number) => {
    setBusy(true);
    setErr(null);
    const body = pid !== undefined && pid > 0 ? { comm, pid } : { comm, pid: 0 };
    try {
      const res = await fetch('/api/target', {
        method: 'POST',
        headers: apiHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        setErr(`target failed (${res.status})`);
      } else {
        setTargetInput('');
        if (onTargetSelect) onTargetSelect(comm, pid);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'network error');
    } finally {
      setBusy(false);
    }
  };

  const handleTargetSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const raw = targetInput.trim();
    if (!raw) return;
    if (/^\d+$/.test(raw)) {
      await applyTarget('', Number(raw));
    } else {
      await applyTarget(raw, 0);
    }
  };

  const statusColors = {
    connected: 'bg-green',
    connecting: 'bg-amber',
    reconnecting: 'bg-amber',
    offline: 'bg-rose',
  };

  const tabs: { id: TabType; label: string; icon: React.ReactNode; badge: string; description: string }[] = [
    {
      id: 'story',
      label: 'Process Story',
      icon: <Bot size={14} />,
      badge: 'Activity',
      description: 'Chronological timeline of tool executions, files touched, sockets opened, and AI verdict.',
    },
    {
      id: 'basic',
      label: 'Overview',
      icon: <Activity size={14} />,
      badge: 'Vitals',
      description: 'Core Agent Golden Signals: Syscall latency quantiles (P50/P90/P99), traffic RPS, and error budget.',
    },
    {
      id: 'medium',
      label: 'Execution & CPU',
      icon: <Layers size={14} />,
      badge: 'Call Trees',
      description: 'Continuous on-CPU flamegraph stack profiling, Perfetto trace timeline, and live Linux process hierarchy.',
    },
    {
      id: 'advanced',
      label: 'System Metrics',
      icon: <Cpu size={14} />,
      badge: 'Telemetry',
      description: 'Deep operational telemetry: Metrics Query Language (MQL) console, syscall categories, and storage IOPS.',
    },
    {
      id: 'complete',
      label: 'Security & Logs',
      icon: <ShieldCheck size={14} />,
      badge: 'Audit',
      description: 'Security & audit center: Structured JSON logs, Linux sandbox audit, and 6-axis AI workload radar.',
    },
  ];

  const themeOptions: { id: ThemeMode; label: string; icon: React.ReactNode; title: string }[] = [
    { id: 'light', label: 'Light', icon: <Sun size={13} />, title: 'Light Mode' },
    { id: 'dark', label: 'Dark', icon: <Moon size={13} />, title: 'Dark Mode' },
    { id: 'ubuntu', label: 'Ubuntu', icon: <span className="text-[11px] font-bold leading-none">U</span>, title: 'Ubuntu Mode' },
    { id: 'unix', label: 'Unix', icon: <Terminal size={13} />, title: 'Unix/Terminal Mode' },
  ];

  const uniqueProcesses = Array.from(
    new Map(processes.map((p) => [agentLabel(p.comm, p.cmdline) + ':' + p.pid, p])).values()
  );
  const currentTargetValue = uniqueProcesses.find((p) => {
    const label = agentLabel(p.comm, p.cmdline).toLowerCase();
    const target = (targetLabel || '').toLowerCase();
    return label === target || p.comm.toLowerCase() === target || target.includes(p.comm.toLowerCase()) || target.includes(String(p.pid));
  });

  return (
    <header className="bg-panel border-b border-border z-30 sticky top-0 shrink-0 w-full transition-colors shadow-sm">
      {/* DrishtiScope Top Bar */}
      <div className="h-14 flex items-center justify-between px-3 sm:px-4 gap-2 w-full overflow-x-auto scrollbar-none flex-nowrap">
        {/* Brand (Left) */}
        <div className="flex items-center gap-2 sm:gap-2.5 shrink-0 flex-nowrap">
          {/* DrishtiScope Spectrum Emblem */}
          <div className="flex items-center gap-2 text-txt font-bold text-base sm:text-lg tracking-tight shrink-0">
            <div className="flex items-center gap-1.5 p-1.5 rounded-lg bg-panel2 border border-border shrink-0">
              <div className="flex items-center gap-0.5">
                <span className="w-2.5 h-2.5 rounded-full bg-cyan" title="eBPF Kernel Trace" />
                <span className="w-2.5 h-2.5 rounded-full bg-rose" title="Security Alerts" />
                <span className="w-2.5 h-2.5 rounded-full bg-amber" title="Performance Metrics" />
                <span className="w-2.5 h-2.5 rounded-full bg-green" title="Health Signals" />
              </div>
            </div>

            <div className="hidden sm:flex flex-col shrink-0">
              <div className="flex items-center gap-1.5 leading-none">
                <span className="text-txt font-bold tracking-tight">DrishtiScope</span>
                <span className="text-[10px] font-semibold px-1.5 py-0.2 rounded bg-cyan/15 text-cyan border border-cyan/30">
                  दृष्टि
                </span>
              </div>
              <span className="text-[10px] font-normal text-muted leading-tight hidden lg:inline">
                Real-Time Agentic AI & LLM Process Observability
              </span>
            </div>
          </div>

          <div className="shrink-0">
            <ModeChip mode={mode} />
          </div>
        </div>

        {/* Unified Controls (Right) - All in a Single Straight Line */}
        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0 justify-end ml-auto flex-nowrap whitespace-nowrap">
          {/* Single Unified Process Target Omnibox (Only 1 Search Option) */}
          <div className="relative flex items-center bg-panel2 border border-border focus-within:border-cyan focus-within:ring-1 focus-within:ring-cyan rounded-lg px-2.5 py-1 text-xs font-mono text-txt shadow-sm transition-all gap-1.5 shrink-0">
            <Search size={13} className="text-cyan shrink-0" />
            <form onSubmit={handleTargetSubmit} className="flex items-center min-w-0">
              <input
                type="text"
                list="header-process-options"
                placeholder={targetLabel ? `${targetLabel} (/)` : 'Target PID or process... (/)'}
                className="bg-transparent text-xs text-txt placeholder:text-muted/70 focus:outline-none w-20 sm:w-36 md:w-48 font-mono leading-none truncate"
                value={targetInput}
                onChange={(e) => {
                  const val = e.target.value;
                  setTargetInput(val);
                  // Immediate apply if user selects a datalist item or typed an exact PID
                  const matched = processes.find(
                    (p) => String(p.pid) === val.trim() || p.comm.toLowerCase() === val.trim().toLowerCase()
                  );
                  if (matched) {
                    const label = agentLabel(matched.comm, matched.cmdline);
                    applyTarget(label, matched.pid);
                  }
                }}
                disabled={busy}
              />
            </form>
            <datalist id="header-process-options">
              {uniqueProcesses.map((p) => {
                const label = agentLabel(p.comm, p.cmdline);
                return (
                  <option key={p.pid} value={String(p.pid)}>
                    PID {p.pid} · {label}
                  </option>
                );
              })}
              <option value="agy">agy — Autonomous Pair Programming Agent</option>
              <option value="codex">codex — AI Code Execution Host</option>
            </datalist>
            {err && (
              <div className="absolute right-0 top-full mt-1 text-[10px] text-rose bg-panel border border-border rounded px-2 py-0.5 shadow z-50 whitespace-nowrap">
                {err}
              </div>
            )}
          </div>

          {/* Anti-Flicker Smooth Mode Toggle */}
          <button
            type="button"
            onClick={toggleAntiFlicker}
            title={antiFlicker ? 'Anti-Flicker is ON (Smooth, zero eye-strain). Click to enable high-FPS flicker mode.' : 'Anti-Flicker is OFF (High-FPS animation mode). Click to enable smooth eye-comfort mode.'}
            className={`flex items-center gap-1 px-2 py-1.5 rounded-lg text-xs font-mono border transition-all shrink-0 whitespace-nowrap ${
              antiFlicker
                ? 'bg-green/15 text-green border-green/40 shadow-sm'
                : 'bg-panel2 text-muted hover:text-amber border-border'
            }`}
          >
            {antiFlicker ? <Shield size={12} className="text-green" /> : <Zap size={12} className="text-amber" />}
            <span className="hidden xl:inline">{antiFlicker ? 'Smooth' : 'Rapid'}</span>
          </button>

          {/* Refresh Rate Selector */}
          <div className="hidden md:flex items-center bg-panel2 border border-border rounded-lg px-2 py-1 text-xs font-mono text-txt shrink-0 whitespace-nowrap">
            <span className="text-muted hidden 2xl:inline text-[10px] mr-1">Rate:</span>
            <select
              aria-label="Refresh Rate / Flicker Control"
              value={refreshRate}
              onChange={(e) => setRefreshRate(e.target.value as any)}
              className="bg-transparent text-txt text-xs font-mono focus:outline-none cursor-pointer pr-1"
              title="Select telemetry refresh rate to reduce flicker & eye strain"
            >
              <option value="2000" className="bg-panel text-txt">2s (Calm)</option>
              <option value="1000" className="bg-panel text-txt">1s (Balanced)</option>
              <option value="500" className="bg-panel text-txt">500ms (Rapid)</option>
              <option value="5000" className="bg-panel text-txt">5s (Relaxed)</option>
              <option value="manual" className="bg-panel text-txt">Manual (No stream)</option>
            </select>
          </div>

          {/* Stream Pause / Play Toggle with Question Mark (?) Help */}
          <div className="flex items-center gap-1 bg-panel2 border border-border rounded-lg px-2 py-0.5 shrink-0 flex-nowrap whitespace-nowrap">
            <button
              type="button"
              onClick={togglePause}
              title={isPaused ? 'Resume live kernel stream' : 'Pause stream to freeze and inspect snapshot'}
              className={`flex items-center gap-1 px-1.5 py-1 rounded-md text-xs font-mono border transition-all shrink-0 whitespace-nowrap ${
                isPaused
                  ? 'bg-amber/15 text-amber border-amber/40 animate-pulse'
                  : 'bg-transparent text-muted hover:text-txt border-transparent'
              }`}
            >
              {isPaused ? <Play size={12} className="text-amber" /> : <Pause size={12} />}
              <span className="whitespace-nowrap">{isPaused ? 'Paused' : 'Live'}</span>
            </button>
            <MetricHelpButton
              metricId="stream_live_pause"
              color="amber"
              size={11}
              title="What does Live / Paused mean? (Click to view guide)"
            />
          </div>

          {/* Time Window Selector with Question Mark (?) Help - Strictly Single Line */}
          <div className="hidden sm:flex items-center bg-panel2 border border-border rounded-lg p-0.5 text-[11px] font-mono gap-0.5 shrink-0 flex-nowrap whitespace-nowrap">
            {['1m', '5m', '15m', '1h'].map((tr) => (
              <button
                key={tr}
                type="button"
                onClick={() => setTimeRange(tr)}
                className={`px-1.5 py-0.5 rounded transition-colors shrink-0 whitespace-nowrap ${
                  timeRange === tr ? 'bg-cyan/20 text-cyan font-bold' : 'text-muted hover:text-txt'
                }`}
                title={`Observation time window: ${tr}`}
              >
                {tr}
              </button>
            ))}
            <div className="w-px h-3 bg-border mx-0.5 shrink-0" />
            <MetricHelpButton
              metricId="time_window_range"
              color="cyan"
              size={11}
              title="What do 1m, 5m, 15m, 1h mean? (Click to view guide)"
            />
          </div>

          {/* Theme Mode Dropdown Selector */}
          <div className="flex items-center bg-panel2 border border-border rounded-lg px-2 py-1 text-xs font-mono text-txt shrink-0 whitespace-nowrap">
            <Palette size={12} className="text-cyan mr-1.5 shrink-0" />
            <select
              aria-label="Select Theme Mode"
              value={theme}
              onChange={(e) => setTheme(e.target.value as ThemeMode)}
              className="bg-transparent text-txt text-xs font-mono focus:outline-none cursor-pointer pr-1"
              title="Choose visual mode (Light is default)"
            >
              <option value="light" className="bg-panel text-txt">☀️ Light</option>
              <option value="dark" className="bg-panel text-txt">🌙 Dark</option>
              <option value="ubuntu" className="bg-panel text-txt">🟠 Ubuntu</option>
              <option value="unix" className="bg-panel text-txt">📟 Unix</option>
              <option value="purple" className="bg-panel text-txt">🔮 Purple</option>
            </select>
          </div>

          {/* Connection Pill */}
          <div className="flex items-center gap-1.5 text-[10px] font-mono text-muted bg-panel2 px-2 py-1.5 rounded-lg border border-border shrink-0 whitespace-nowrap">
            <div className={`w-2 h-2 rounded-full ${statusColors[status]}`} />
            <span className="hidden sm:inline">{status.toUpperCase()}</span>
          </div>
        </div>
      </div>

      {/* Navigation Tabs Bar */}
      <div className="flex items-center gap-1 sm:gap-2 px-3 sm:px-4 py-1.5 border-t border-border bg-panel2/70 overflow-x-auto scrollbar-none">
        {tabs.map((tab) => {
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-medium transition-all shrink-0 ${
                isActive
                  ? 'bg-cyan/15 text-cyan border border-cyan/40 shadow-sm font-semibold'
                  : 'text-muted hover:text-txt hover:bg-panel border border-transparent'
              }`}
            >
              {tab.icon}
              <span>{tab.label}</span>
              <span
                className={`text-[9px] px-1.5 py-0.2 rounded font-mono ${
                  isActive ? 'bg-cyan/25 text-cyan' : 'bg-border/60 text-muted'
                }`}
              >
                {tab.badge}
              </span>
            </button>
          );
        })}
      </div>
    </header>
  );
}
