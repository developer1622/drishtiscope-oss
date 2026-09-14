import React, { useState, useEffect, useRef } from 'react';
import { ModeChip } from './ModeChip';
import { Logo } from './Logo';
import { HelloPayload, ConnectionStatus, ProcessRow } from '../types/protocol';
import { agentLabel, fmtBytes } from '../utils/format';
import { useScopeStore, TabType, ThemeMode } from '../store/useScopeStore';
import { apiHeaders } from '../utils/api';
import { MetricHelpButton } from './MetricHelpModal';
import {
  Search,
  X,
  Check,
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
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const searchContainerRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Close suggestions when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        searchContainerRef.current &&
        !searchContainerRef.current.contains(e.target as Node)
      ) {
        setIsSearchOpen(false);
        setHighlightedIndex(-1);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

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
        setSearchQuery('');
        setIsSearchOpen(false);
        setHighlightedIndex(-1);
        if (onTargetSelect) onTargetSelect(comm, pid);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'network error');
    } finally {
      setBusy(false);
    }
  };

  const selectProcess = async (p: ProcessRow) => {
    const label = agentLabel(p.comm, p.cmdline);
    await applyTarget(label, p.pid);
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

  // Filter suggestions of currently executing processes based on search query
  const query = searchQuery.trim().toLowerCase();
  const currentTargetClean = (targetLabel || '').toLowerCase();

  // If no processes currently tracked, provide fallback agent suggestions so user can easily click-to-target
  const displayProcesses: ProcessRow[] =
    processes.length > 0
      ? processes
      : [
          {
            pid: 143399,
            tgid: 143399,
            ppid: 1,
            comm: 'agy',
            cmdline: 'agy --conversation=active',
            exe: '/usr/local/bin/agy',
            uid: 1000,
            cpu_pct: 1.2,
            rss_bytes: 604471296,
            vms_bytes: 2417885184,
            state: 'S',
            threads: 17,
            open_fds: 72,
            ctx_switches: 12400,
            start_time: '12:00:00',
          },
          {
            pid: 1042,
            tgid: 1042,
            ppid: 1,
            comm: 'python3',
            cmdline: 'python3 -m agentscope.runtime',
            exe: '/usr/bin/python3',
            uid: 1000,
            cpu_pct: 0.8,
            rss_bytes: 314572800,
            vms_bytes: 1258291200,
            state: 'S',
            threads: 8,
            open_fds: 45,
            ctx_switches: 8420,
            start_time: '12:05:00',
          },
          {
            pid: 2190,
            tgid: 2190,
            ppid: 1,
            comm: 'node',
            cmdline: 'node server.js',
            exe: '/usr/local/bin/node',
            uid: 1000,
            cpu_pct: 0.5,
            rss_bytes: 157286400,
            vms_bytes: 629145600,
            state: 'S',
            threads: 11,
            open_fds: 28,
            ctx_switches: 5120,
            start_time: '12:10:00',
          },
        ];

  const filteredProcesses = displayProcesses
    .filter((p) => {
      if (!query) return true;
      const label = agentLabel(p.comm, p.cmdline).toLowerCase();
      const pidStr = String(p.pid);
      const comm = p.comm.toLowerCase();
      const cmdline = (p.cmdline || '').toLowerCase();
      return (
        label.includes(query) ||
        pidStr.includes(query) ||
        comm.includes(query) ||
        cmdline.includes(query)
      );
    })
    .sort((a, b) => {
      // Prioritize currently observed target
      const aIsTarget =
        currentTargetClean.includes(String(a.pid)) ||
        currentTargetClean.includes(a.comm.toLowerCase());
      const bIsTarget =
        currentTargetClean.includes(String(b.pid)) ||
        currentTargetClean.includes(b.comm.toLowerCase());
      if (aIsTarget && !bIsTarget) return -1;
      if (!aIsTarget && bIsTarget) return 1;

      // Prioritize CPU usage, then RSS
      if (b.cpu_pct !== a.cpu_pct) return b.cpu_pct - a.cpu_pct;
      return b.rss_bytes - a.rss_bytes;
    });

  const handleSearchKeyDown = (e: React.KeyboardEvent) => {
    if (!isSearchOpen) {
      if (e.key === 'ArrowDown' || e.key === 'Enter') {
        setIsSearchOpen(true);
        e.preventDefault();
      }
      return;
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setHighlightedIndex((prev) =>
        prev < filteredProcesses.length - 1 ? prev + 1 : 0
      );
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setHighlightedIndex((prev) =>
        prev > 0 ? prev - 1 : filteredProcesses.length - 1
      );
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (highlightedIndex >= 0 && highlightedIndex < filteredProcesses.length) {
        selectProcess(filteredProcesses[highlightedIndex]);
      } else if (filteredProcesses.length === 1) {
        selectProcess(filteredProcesses[0]);
      } else if (searchQuery.trim()) {
        const raw = searchQuery.trim();
        if (/^\d+$/.test(raw)) {
          applyTarget('', Number(raw));
        } else {
          applyTarget(raw, 0);
        }
      }
    } else if (e.key === 'Escape') {
      setIsSearchOpen(false);
      setHighlightedIndex(-1);
    }
  };

  return (
    <header className="bg-panel border-b border-border z-30 sticky top-0 shrink-0 w-full transition-colors shadow-sm">
      {/* DrishtiScope Top Bar */}
      <div className="h-14 flex items-center justify-between px-3 sm:px-4 gap-2 w-full relative">
        {/* Brand & Mode Indicator (Left) - Single sleek line on all screen sizes */}
        <div className="flex items-center gap-2 sm:gap-2.5 shrink-0 flex-nowrap">
          <Logo size={28} />
          <div className="flex items-center gap-1.5 leading-none shrink-0">
            <span className="text-txt font-bold tracking-tight text-sm sm:text-base md:text-lg">
              DrishtiScope
            </span>
            <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded bg-cyan/15 text-cyan border border-cyan/30 font-mono">
              दृष्टि
            </span>
          </div>

          <div className="shrink-0 ml-0.5 sm:ml-1">
            <ModeChip mode={mode} />
          </div>
        </div>

        {/* Unified Controls (Right) - All in a Single Straight Line */}
        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0 justify-end ml-auto flex-nowrap whitespace-nowrap">
          {/* Executing Processes Search Omnibox & Live Suggestions Dropdown */}
          <div ref={searchContainerRef} className="relative shrink-0">
            <div
              className={`flex items-center bg-panel2 border rounded-lg px-2.5 py-1 text-xs font-mono text-txt shadow-sm transition-all gap-1.5 ${
                isSearchOpen
                  ? 'border-cyan ring-1 ring-cyan/40 bg-panel'
                  : 'border-border hover:border-border/80'
              }`}
            >
              <Search size={13} className="text-cyan shrink-0" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                placeholder={targetLabel ? `Search: ${targetLabel} (/)` : 'Search executing processes... (/)'}
                className="bg-transparent text-xs text-txt placeholder:text-muted/70 focus:outline-none w-28 sm:w-44 md:w-56 font-mono leading-none truncate"
                onFocus={() => {
                  setIsSearchOpen(true);
                  setHighlightedIndex(-1);
                }}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setIsSearchOpen(true);
                  setHighlightedIndex(-1);
                }}
                onKeyDown={handleSearchKeyDown}
                disabled={busy}
              />
              {busy ? (
                <div className="w-3 h-3 rounded-full border border-cyan border-t-transparent animate-spin shrink-0" />
              ) : searchQuery ? (
                <button
                  type="button"
                  onClick={() => {
                    setSearchQuery('');
                    setHighlightedIndex(-1);
                    searchInputRef.current?.focus();
                  }}
                  className="text-muted hover:text-txt p-0.5 rounded transition-colors shrink-0"
                  title="Clear search"
                >
                  <X size={12} />
                </button>
              ) : null}
            </div>

            {/* Suggestions Dropdown: Current Executing Processes */}
            {isSearchOpen && (
              <div className="absolute right-0 top-full mt-1.5 w-80 sm:w-96 max-h-80 bg-panel border border-border rounded-xl shadow-2xl z-50 flex flex-col overflow-hidden animate-in fade-in-50 zoom-in-95 duration-100">
                {/* Header */}
                <div className="px-3 py-2 bg-panel2/80 border-b border-border flex items-center justify-between gap-2 shrink-0">
                  <div className="flex items-center gap-1.5">
                    <Activity size={12} className="text-cyan" />
                    <span className="text-[11px] font-bold text-txt">Current Executing Processes</span>
                    <span className="text-[10px] px-1.5 py-0.2 rounded bg-cyan/15 text-cyan border border-cyan/30 font-mono">
                      {filteredProcesses.length}
                    </span>
                  </div>
                  <span className="text-[10px] text-muted font-mono hidden sm:inline">
                    ↑↓ navigate · ↵ select · esc close
                  </span>
                </div>

                {/* Suggestions List */}
                <div className="overflow-y-auto max-h-60 divide-y divide-border/30">
                  {filteredProcesses.length === 0 ? (
                    <div className="p-4 text-center text-xs font-mono text-muted">
                      No executing processes match "{searchQuery}"
                      {searchQuery.trim() && (
                        <button
                          type="button"
                          onClick={() => {
                            const raw = searchQuery.trim();
                            if (/^\d+$/.test(raw)) {
                              applyTarget('', Number(raw));
                            } else {
                              applyTarget(raw, 0);
                            }
                          }}
                          className="mt-2 block w-full py-1.5 px-2 rounded bg-panel2 hover:bg-cyan/20 text-cyan text-center border border-border transition-colors text-[11px]"
                        >
                          Target "{searchQuery.trim()}" directly →
                        </button>
                      )}
                    </div>
                  ) : (
                    filteredProcesses.slice(0, 25).map((p, idx) => {
                      const label = agentLabel(p.comm, p.cmdline);
                      const isHighlighted = idx === highlightedIndex;
                      const isTarget =
                        currentTargetClean.includes(String(p.pid)) ||
                        currentTargetClean.includes(p.comm.toLowerCase());

                      return (
                        <button
                          key={p.pid}
                          type="button"
                          onClick={() => selectProcess(p)}
                          onMouseEnter={() => setHighlightedIndex(idx)}
                          className={`w-full text-left px-3 py-2 flex items-center justify-between gap-2 text-xs font-mono transition-colors ${
                            isHighlighted
                              ? 'bg-cyan/15 text-txt'
                              : isTarget
                              ? 'bg-panel2/80 text-txt'
                              : 'hover:bg-panel2 text-muted hover:text-txt'
                          }`}
                        >
                          <div className="flex flex-col gap-0.5 min-w-0 flex-1">
                            <div className="flex items-center gap-1.5 truncate">
                              <span className="font-bold text-txt truncate">{label}</span>
                              <span className="text-[10px] px-1.5 py-0.2 rounded bg-panel border border-border text-cyan font-semibold shrink-0">
                                PID {p.pid}
                              </span>
                              {isTarget && (
                                <span className="text-[9px] px-1.5 py-0.2 rounded bg-green/15 text-green border border-green/30 font-semibold shrink-0 flex items-center gap-0.5">
                                  <Check size={10} /> Active
                                </span>
                              )}
                              <span
                                className={`text-[9px] px-1 py-0.2 rounded shrink-0 font-bold ${
                                  p.state === 'R'
                                    ? 'bg-green/20 text-green'
                                    : p.state === 'D'
                                    ? 'bg-amber/20 text-amber'
                                    : 'bg-panel border border-border text-muted'
                                }`}
                                title={`Process state: ${p.state}`}
                              >
                                {p.state}
                              </span>
                            </div>
                            <div className="text-[10px] text-muted truncate">
                              {p.cmdline || p.exe || p.comm}
                            </div>
                          </div>

                          <div className="flex flex-col items-end gap-0.5 shrink-0 text-[10px]">
                            <span className="font-bold text-cyan">{p.cpu_pct.toFixed(1)}% CPU</span>
                            <span className="text-muted">{fmtBytes(p.rss_bytes)}</span>
                          </div>
                        </button>
                      );
                    })
                  )}
                </div>

                {/* Footer status / count if truncated */}
                {filteredProcesses.length > 25 && (
                  <div className="px-3 py-1 bg-panel2/50 border-t border-border/50 text-[10px] text-muted font-mono text-center shrink-0">
                    Showing top 25 of {filteredProcesses.length} executing processes. Type to narrow search.
                  </div>
                )}
              </div>
            )}

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

          {/* Time Window Selector with Question Mark (?) Help - Visible Across All Screen Sizes */}
          <div className="flex items-center bg-panel2 border border-border rounded-lg p-0.5 text-[10px] sm:text-[11px] font-mono gap-0.5 shrink-0 flex-nowrap whitespace-nowrap">
            {['1m', '5m', '15m', '1h'].map((tr) => (
              <button
                key={tr}
                type="button"
                onClick={() => setTimeRange(tr)}
                className={`px-1 sm:px-1.5 py-0.5 rounded transition-colors shrink-0 whitespace-nowrap ${
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
