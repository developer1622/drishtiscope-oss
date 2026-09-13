import React, { useState } from 'react';
import { ModeChip } from './ModeChip';
import { HelloPayload, ConnectionStatus, ProcessRow } from '../types/protocol';
import { useScopeStore, TabType } from '../store/useScopeStore';
import { apiHeaders } from '../utils/api';
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
  ExternalLink,
  ChevronRight,
  Sparkles,
} from 'lucide-react';

export function Header({
  mode,
  hello,
  status,
  targetLabel,
  processes = [],
  onTargetSelect,
}: {
  mode: 'ebpf' | 'mock' | null;
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
    toggleTheme,
    activeTab,
    setActiveTab,
    isPaused,
    togglePause,
    timeRange,
    setTimeRange,
  } = useScopeStore();

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

  const tabs: { id: TabType; label: string; icon: React.ReactNode; badge: string }[] = [
    { id: 'basic', label: 'SRE Golden Signals', icon: <Activity size={14} />, badge: 'Monarch Vitals' },
    { id: 'medium', label: 'Cloud Profiler & Trace', icon: <Layers size={14} />, badge: 'Perfetto UI' },
    { id: 'advanced', label: 'Cloud Monitoring', icon: <Cpu size={14} />, badge: 'MQL Deep' },
    { id: 'complete', label: 'Cloud Logging & Chronicle', icon: <ShieldCheck size={14} />, badge: 'Security Audit' },
  ];

  const uniqueProcesses = Array.from(
    new Map(processes.map((p) => [p.comm, p])).values()
  );

  return (
    <header className="bg-panel border-b border-border z-30 sticky top-0 shrink-0 w-full transition-colors shadow-sm">
      {/* Google Cloud Console Top Bar */}
      <div className="h-14 flex items-center justify-between px-3 sm:px-4 gap-2">
        {/* Brand, Google Emblem & Project Breadcrumbs */}
        <div className="flex items-center gap-2 sm:gap-3 shrink-0">
          {/* Google 4-Color Accent Emblem */}
          <div className="flex items-center gap-2 text-txt font-bold text-base sm:text-lg tracking-tight">
            <div className="flex items-center gap-1.5 p-1.5 rounded-lg bg-panel2 border border-border">
              <div className="flex items-center gap-0.5">
                <span className="w-2.5 h-2.5 rounded-full bg-[#4285F4]" title="Google Blue" />
                <span className="w-2.5 h-2.5 rounded-full bg-[#EA4335]" title="Google Red" />
                <span className="w-2.5 h-2.5 rounded-full bg-[#FBBC04]" title="Google Yellow" />
                <span className="w-2.5 h-2.5 rounded-full bg-[#34A853]" title="Google Green" />
              </div>
            </div>

            <div className="flex flex-col">
              <div className="flex items-center gap-1.5 leading-none">
                <span className="text-txt font-bold tracking-tight">DrishtiScope</span>
                <span className="text-[10px] font-semibold px-1.5 py-0.2 rounded bg-cyan/15 text-cyan border border-cyan/30">
                  दृष्टि
                </span>
              </div>
              <span className="text-[10px] font-normal text-muted leading-tight hidden md:inline">
                Google SRE Kernel Observability
              </span>
            </div>
          </div>

          <div className="w-px h-5 bg-border mx-1 hidden lg:block" />

          {/* Google Cloud Style Project Breadcrumb */}
          <div className="hidden xl:flex items-center gap-1 text-[11px] font-mono text-muted bg-panel2/80 px-2.5 py-1 rounded-md border border-border">
            <span className="text-cyan font-medium">google-internal-sre</span>
            <ChevronRight size={12} className="opacity-50" />
            <span className="text-txt">wsl2-production</span>
            <ChevronRight size={12} className="opacity-50" />
            <span className="text-amber truncate max-w-[120px]">{targetLabel || 'agy'}</span>
          </div>

          <ModeChip mode={mode} />

          {/* Hostname & Kernel */}
          {hello && (
            <div
              className="hidden 2xl:flex items-center gap-1.5 text-[11px] font-mono text-muted bg-panel2 px-2.5 py-1 rounded-full border border-border max-w-[240px] truncate"
              title={`${hello.hostname} | ${hello.kernel}`}
            >
              <Server size={12} className="shrink-0 text-cyan" />
              <span className="truncate">{hello.hostname}</span>
              <span className="opacity-40">|</span>
              <span className="truncate opacity-75">{hello.kernel}</span>
            </div>
          )}
        </div>

        {/* Omnibox, Process Selector, Live Controls & Google Tools */}
        <div className="flex items-center gap-2 sm:gap-2.5">
          {/* Quick Target Selector */}
          <div className="flex items-center gap-1.5 bg-panel2 border border-border rounded-lg px-2.5 py-1 text-xs font-mono text-txt">
            <Crosshair size={13} className="text-cyan shrink-0" />
            <span className="text-muted hidden md:inline text-[11px]">Target:</span>
            <select
              aria-label="Pick target process"
              className="bg-transparent text-txt text-xs font-mono focus:outline-none cursor-pointer pr-1 max-w-[100px] sm:max-w-[140px] truncate"
              value={targetLabel?.split(' ')[0] || ''}
              onChange={(e) => {
                const val = e.target.value;
                if (!val) return;
                const matched = processes.find((p) => p.comm === val);
                applyTarget(val, matched ? matched.pid : 0);
              }}
              disabled={busy}
            >
              <option value="" disabled className="bg-panel text-muted">
                Choose Process...
              </option>
              {uniqueProcesses.map((p) => (
                <option key={p.comm} value={p.comm} className="bg-panel text-txt">
                  {p.comm} (PID {p.pid})
                </option>
              ))}
            </select>
          </div>

          {/* Google Cloud Omnibox Search Bar */}
          <form onSubmit={handleTargetSubmit} className="relative hidden lg:block">
            <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
            <input
              type="text"
              list="header-process-options"
              placeholder={targetLabel ? `Filter / Target (/)` : 'Search resource / PID (/)'}
              className="bg-panel2 border border-border text-xs text-txt rounded-lg pl-8 pr-3 py-1.5 w-36 xl:w-52 focus:outline-none focus:border-cyan focus:ring-1 focus:ring-cyan transition-all font-mono placeholder:font-sans placeholder:text-muted/70"
              value={targetInput}
              onChange={(e) => setTargetInput(e.target.value)}
              disabled={busy}
            />
            <datalist id="header-process-options">
              {uniqueProcesses.map((p) => (
                <option key={p.pid} value={p.comm}>
                  PID {p.pid} - {p.comm}
                </option>
              ))}
            </datalist>
            {err && <div className="absolute right-0 top-full mt-1 text-[10px] text-rose">{err}</div>}
          </form>

          {/* Stream Pause / Play Toggle */}
          <button
            type="button"
            onClick={togglePause}
            title={isPaused ? 'Resume live kernel stream' : 'Pause stream to inspect snapshot'}
            className={`flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-mono border transition-all ${
              isPaused
                ? 'bg-amber/15 text-amber border-amber/40 animate-pulse'
                : 'bg-panel2 text-muted hover:text-txt border-border'
            }`}
          >
            {isPaused ? <Play size={12} className="text-amber" /> : <Pause size={12} />}
            <span className="hidden sm:inline">{isPaused ? 'Paused' : 'Live'}</span>
          </button>

          {/* Time Window Selector */}
          <div className="hidden sm:flex items-center bg-panel2 border border-border rounded-lg p-0.5 text-[11px] font-mono">
            {['1m', '5m', '15m', '1h'].map((tr) => (
              <button
                key={tr}
                type="button"
                onClick={() => setTimeRange(tr)}
                className={`px-2 py-0.5 rounded ${
                  timeRange === tr ? 'bg-cyan/20 text-cyan font-bold' : 'text-muted hover:text-txt'
                }`}
              >
                {tr}
              </button>
            ))}
          </div>

          {/* Export Perfetto Trace Button */}
          <a
            href="/api/v1/traces/perfetto"
            target="_blank"
            rel="noreferrer"
            title="Download Google Perfetto / Chrome Trace Event JSON for ui.perfetto.dev"
            className="hidden md:flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-panel2 border border-border text-xs font-mono text-muted hover:text-cyan hover:border-cyan/40 transition-colors"
          >
            <Download size={12} className="text-cyan" />
            <span className="hidden xl:inline">Perfetto</span>
          </a>

          {/* Prometheus Metrics Button */}
          <a
            href="/metrics"
            target="_blank"
            rel="noreferrer"
            title="View Prometheus / OpenMetrics Monarch scraping endpoint"
            className="hidden md:flex items-center gap-1 px-2 py-1.5 rounded-lg bg-panel2 border border-border text-xs font-mono text-muted hover:text-amber hover:border-amber/40 transition-colors"
          >
            <span className="text-[10px] font-bold text-amber">/metrics</span>
          </a>

          {/* Theme Toggle (Dark / Light) */}
          <button
            type="button"
            onClick={toggleTheme}
            title={theme === 'dark' ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
            className="p-1.5 rounded-lg bg-panel2 border border-border text-muted hover:text-txt transition-colors"
          >
            {theme === 'dark' ? (
              <Sun size={15} className="text-amber" />
            ) : (
              <Moon size={15} className="text-cyan" />
            )}
          </button>

          {/* Connection Pill */}
          <div className="flex items-center gap-1.5 text-[10px] font-mono text-muted bg-panel2 px-2 py-1.5 rounded-lg border border-border">
            <div className={`w-2 h-2 rounded-full ${statusColors[status]}`} />
            <span className="hidden sm:inline">{status.toUpperCase()}</span>
          </div>
        </div>
      </div>

      {/* Google Cloud Operations Navigation Tabs Bar */}
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
