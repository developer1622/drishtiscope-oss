import React from 'react';
import { Snapshot, ProcessRow, EventRow } from '../../types/protocol';
import { Panel } from '../Panel';
import { MetricHelpButton } from '../MetricHelpModal';
import { fmtBytes, fmtBps, fmtPct, agentLabel } from '../../utils/format';
import {
  Bot,
  Activity,
  Cpu,
  HardDrive,
  Network,
  Clock,
  Terminal,
  FileCode,
  ShieldAlert,
  Zap,
  CheckCircle2,
  FolderOpen,
  ArrowRight,
  Copy,
  Check,
} from 'lucide-react';
import {
  AreaChart,
  Area,
  ResponsiveContainer,
  XAxis,
  YAxis,
  Tooltip,
} from 'recharts';

export function StoryTab({
  snapshot,
  events,
  targetProcess,
}: {
  snapshot: Snapshot | null;
  events: EventRow[];
  targetProcess?: ProcessRow;
}) {
  const [copiedCmd, setCopiedCmd] = React.useState<string | null>(null);

  const target = snapshot?.meta?.target;
  const kpis = snapshot?.kpis;
  const activePID = targetProcess?.pid || target?.pid || 0;
  const activeComm = targetProcess?.comm || target?.comm || 'unknown';
  const label = agentLabel(activeComm, targetProcess?.cmdline);

  const copyToClipboard = (cmd: string) => {
    navigator.clipboard.writeText(cmd);
    setCopiedCmd(cmd);
    setTimeout(() => setCopiedCmd(null), 2000);
  };

  // Classify process role
  const lowerComm = activeComm.toLowerCase();
  const lowerCmd = (targetProcess?.cmdline || '').toLowerCase();
  let agentRole = 'Linux Background Process';
  let agentBadge = 'Native Process';
  let roleColor = 'text-cyan border-cyan/40 bg-cyan/15';

  if (lowerComm.includes('codex') || lowerCmd.includes('codex')) {
    agentRole = 'OpenAI Codex Code Execution & Sandbox Host';
    agentBadge = 'AI Agent Host';
    roleColor = 'text-emerald border-emerald/40 bg-emerald/15';
  } else if (lowerComm.includes('agy') || lowerCmd.includes('agy')) {
    agentRole = 'Antigravity Autonomous Pair Programming Agent';
    agentBadge = 'Autonomous Agent';
    roleColor = 'text-cyan border-cyan/40 bg-cyan/15';
  } else if (lowerCmd.includes('copilot') || lowerComm.includes('copilot')) {
    agentRole = 'GitHub Copilot Language Engine & Completions Agent';
    agentBadge = 'LLM Assistant';
    roleColor = 'text-purple border-purple/40 bg-purple/15';
  } else if (lowerComm.includes('node') || lowerCmd.includes('node')) {
    agentRole = 'Node.js JavaScript Agent & Tool Worker';
    agentBadge = 'Runtime Worker';
    roleColor = 'text-amber border-amber/40 bg-amber/15';
  } else if (lowerComm.includes('python') || lowerCmd.includes('python')) {
    agentRole = 'Python LLM Pipeline & Inference Script';
    agentBadge = 'Python Agent';
    roleColor = 'text-blue border-blue/40 bg-blue/15';
  }

  // Filter events relevant to this target
  const processEvents = events.filter(
    (e) => e.pid === activePID || (activeComm && e.comm.toLowerCase().includes(activeComm.toLowerCase()))
  );

  // Compute behavioral verdict
  const cpu = kpis?.cpu_pct ?? targetProcess?.cpu_pct ?? 0;
  const openFds = kpis?.open_fds ?? targetProcess?.open_fds ?? 0;
  const syscalls = kpis?.syscalls_per_sec ?? 0;

  let verdict = 'Idle Event Loop — Waiting for user prompt or incoming message';
  let verdictColor = 'text-muted border-border bg-panel2';
  let verdictIcon = <Clock size={16} className="text-muted" />;

  if (cpu > 15) {
    verdict = 'Active Code Generation & Execution — Computing tasks across thread pool';
    verdictColor = 'text-cyan border-cyan/40 bg-cyan/15';
    verdictIcon = <Cpu size={16} className="text-cyan animate-pulse" />;
  } else if (openFds > 50) {
    verdict = 'High I/O & Socket Activity — Interacting with multiple tool handles & file descriptors';
    verdictColor = 'text-amber border-amber/40 bg-amber/15';
    verdictIcon = <Zap size={16} className="text-amber" />;
  } else if (syscalls > 1000) {
    verdict = 'Rapid Kernel Operations — Polling file descriptors and network buffers';
    verdictColor = 'text-emerald border-emerald/40 bg-emerald/15';
    verdictIcon = <Activity size={16} className="text-emerald" />;
  }

  // Files accessed by this process
  const filesTop = snapshot?.files_top || [];

  // Network flows for this process
  const flows = snapshot?.flows || [];

  // Diagnostic commands
  const diagCommands = [
    { label: 'Check File Descriptors', cmd: `lsof -p ${activePID}` },
    { label: 'Inspect Memory Vitals', cmd: `cat /proc/${activePID}/status | grep -E "(Vm|Rss|Threads)"` },
    { label: 'Trace Syscalls (3s)', cmd: `strace -p ${activePID} -s 128 -c` },
    { label: 'View Command & Args', cmd: `cat /proc/${activePID}/cmdline | tr '\\0' ' '` },
  ];

  return (
    <div className="flex flex-col gap-4">
      {/* ── 1. Executive Identity Banner ─────────────────────────────── */}
      <div className="bg-panel border border-border rounded-xl p-4 sm:p-5 flex flex-col md:flex-row items-start md:items-center justify-between gap-4 relative overflow-hidden">
        <div className="flex items-start gap-3.5 relative z-10">
          <div className="w-12 h-12 rounded-xl bg-panel2 border border-border flex items-center justify-center shrink-0 shadow-inner">
            <Bot size={24} className="text-cyan" />
          </div>
          <div className="flex flex-col">
            <div className="flex items-center gap-2 flex-wrap">
              <h2 className="text-lg sm:text-xl font-bold text-txt font-mono">{label}</h2>
              <span className={`text-[11px] font-mono font-semibold px-2 py-0.5 rounded-full border ${roleColor}`}>
                {agentBadge}
              </span>
              <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-panel2 text-muted border border-border">
                PID {activePID}
              </span>
              <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-panel2 text-muted border border-border">
                PPID {targetProcess?.ppid ?? 1}
              </span>
            </div>
            <p className="text-xs text-muted mt-1">{agentRole}</p>
            {targetProcess?.cmdline && (
              <p className="text-[11px] font-mono text-muted/80 mt-1 max-w-3xl truncate bg-panel2/60 px-2 py-1 rounded border border-border/40">
                {targetProcess.cmdline}
              </p>
            )}
          </div>
        </div>

        {/* Current State Verdict Pill */}
        <div className={`flex items-center gap-2.5 px-3 py-2 rounded-xl border ${verdictColor} shrink-0 max-w-md`}>
          {verdictIcon}
          <div className="flex flex-col">
            <span className="text-[10px] font-mono font-bold uppercase tracking-wider text-muted">Current Assessment</span>
            <span className="text-xs font-medium leading-snug">{verdict}</span>
          </div>
        </div>
      </div>

      {/* ── 2. Core Telemetry Ribbon ─────────────────────────────────── */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2.5">
        <div className="bg-panel border border-border rounded-xl p-3 flex flex-col justify-between">
          <span className="text-[11px] font-mono text-muted uppercase">CPU Usage</span>
          <div className="flex items-baseline gap-1 my-1">
            <span className="text-xl font-bold text-cyan font-mono">{fmtPct(cpu)}</span>
          </div>
          <span className="text-[10px] text-muted">Kernel + User space</span>
        </div>

        <div className="bg-panel border border-border rounded-xl p-3 flex flex-col justify-between">
          <span className="text-[11px] font-mono text-muted uppercase">Memory (RSS)</span>
          <div className="flex items-baseline gap-1 my-1">
            <span className="text-xl font-bold text-txt font-mono">
              {fmtBytes(kpis?.rss_bytes || targetProcess?.rss_bytes || 0)}
            </span>
          </div>
          <span className="text-[10px] text-muted">Resident in RAM</span>
        </div>

        <div className="bg-panel border border-border rounded-xl p-3 flex flex-col justify-between">
          <span className="text-[11px] font-mono text-muted uppercase">Active Threads</span>
          <div className="flex items-baseline gap-1 my-1">
            <span className="text-xl font-bold text-txt font-mono">
              {targetProcess?.threads ?? kpis?.threads ?? 1}
            </span>
          </div>
          <span className="text-[10px] text-muted">Worker goroutines/tasks</span>
        </div>

        <div className="bg-panel border border-border rounded-xl p-3 flex flex-col justify-between">
          <span className="text-[11px] font-mono text-muted uppercase">Open Handles</span>
          <div className="flex items-baseline gap-1 my-1">
            <span className="text-xl font-bold text-txt font-mono">
              {targetProcess?.open_fds ?? kpis?.open_fds ?? 0}
            </span>
          </div>
          <span className="text-[10px] text-muted">Files & TCP sockets</span>
        </div>

        <div className="bg-panel border border-border rounded-xl p-3 flex flex-col justify-between">
          <span className="text-[11px] font-mono text-muted uppercase">Disk I/O Rate</span>
          <div className="flex items-baseline gap-1 my-1">
            <span className="text-xl font-bold text-emerald font-mono">
              {fmtBps((kpis?.disk_bps_r || 0) + (kpis?.disk_bps_w || 0))}
            </span>
          </div>
          <span className="text-[10px] text-muted">Read / Write throughput</span>
        </div>

        <div className="bg-panel border border-border rounded-xl p-3 flex flex-col justify-between">
          <span className="text-[11px] font-mono text-muted uppercase">Network Rate</span>
          <div className="flex items-baseline gap-1 my-1">
            <span className="text-xl font-bold text-amber font-mono">
              {fmtBps((kpis?.net_bps_tx || 0) + (kpis?.net_bps_rx || 0))}
            </span>
          </div>
          <span className="text-[10px] text-muted">API traffic sent/received</span>
        </div>
      </div>

      {/* ── 3. What Has Happened So Far & Files Touched ──────────────── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Story Activity Timeline */}
        <Panel
          title="What This Process Has Done (Chronological Activity)"
          subtitle={`Events recorded for ${label} since monitoring began`}
        >
          <div className="flex flex-col gap-2.5 max-h-[380px] overflow-y-auto pr-1">
            {processEvents.length === 0 ? (
              <div className="py-10 text-center text-muted text-xs font-mono">
                Monitoring live process events... Operating within normal thresholds with zero kernel faults.
              </div>
            ) : (
              processEvents.slice(0, 15).map((evt, idx) => {
                const sevColor =
                  evt.severity === 'crit'
                    ? 'text-rose border-rose/40 bg-rose/10'
                    : evt.severity === 'warn'
                    ? 'text-amber border-amber/40 bg-amber/10'
                    : 'text-cyan border-cyan/40 bg-cyan/10';

                return (
                  <div
                    key={evt.id || idx}
                    className="p-2.5 rounded-lg bg-panel2 border border-border/70 flex items-start gap-2.5 text-xs font-mono"
                  >
                    <div className={`px-1.5 py-0.5 rounded text-[10px] font-bold border uppercase shrink-0 ${sevColor}`}>
                      {evt.severity}
                    </div>
                    <div className="flex flex-col flex-1 min-w-0">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-bold text-txt truncate">{evt.title}</span>
                        <span className="text-[10px] text-muted shrink-0">{evt.ts.slice(11, 19)}</span>
                      </div>
                      <p className="text-[11px] text-muted mt-0.5 leading-relaxed">{evt.detail}</p>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </Panel>

        {/* Files Touched & Workspaces */}
        <Panel
          title="Files & Workspaces Accessed"
          subtitle="Real-time open file handles, transcripts & database ledgers"
        >
          <div className="flex flex-col gap-2 max-h-[380px] overflow-y-auto">
            {filesTop.length === 0 ? (
              <div className="py-10 text-center text-muted text-xs font-mono">
                Scanning /proc/{activePID}/fd for active file handles...
              </div>
            ) : (
              filesTop.map((f, idx) => (
                <div
                  key={idx}
                  className="p-2.5 rounded-lg bg-panel2 border border-border/70 flex items-center justify-between gap-3 text-xs font-mono"
                >
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <FolderOpen size={14} className="text-cyan shrink-0" />
                    <span className="text-txt truncate font-medium" title={f.path}>
                      {f.path}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 shrink-0 text-muted text-[11px]">
                    <span>{f.ops_s.toFixed(0)} ops/s</span>
                    <span className="text-emerald">{fmtBps(f.bytes_s)}</span>
                  </div>
                </div>
              ))
            )}
          </div>
        </Panel>
      </div>

      {/* ── 4. Network Connections & Quick CLI Diagnostics ─────────── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Active Network Connections */}
        <Panel
          title="Network Endpoints & Tool Sockets"
          subtitle="Remote APIs, DNS resolvers & local IPC sockets"
        >
          <div className="flex flex-col gap-2 max-h-[260px] overflow-y-auto">
            {flows.length === 0 ? (
              <div className="py-8 text-center text-muted text-xs font-mono">
                No active external TCP flows. Process communicates via local pipes or stdin/stdout.
              </div>
            ) : (
              flows.map((fl, idx) => (
                <div
                  key={idx}
                  className="p-2 rounded-lg bg-panel2 border border-border/70 flex items-center justify-between text-xs font-mono"
                >
                  <div className="flex items-center gap-2 truncate">
                    <Network size={13} className="text-amber shrink-0" />
                    <span className="text-muted">{fl.src}</span>
                    <ArrowRight size={11} className="text-muted/60" />
                    <span className="text-txt font-bold">{fl.dst}:{fl.dport}</span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span className="text-[10px] px-1.5 py-0.5 rounded bg-panel border border-border text-emerald">
                      {fl.state || 'ESTABLISHED'}
                    </span>
                    <span className="text-[11px] text-muted">{fmtBytes(fl.bytes_tx + fl.bytes_rx)}</span>
                  </div>
                </div>
              ))
            )}
          </div>
        </Panel>

        {/* Quick Diagnostic CLI Commands */}
        <Panel
          title="One-Click Linux Diagnostic Commands"
          subtitle={`Inspect PID ${activePID} directly from terminal`}
        >
          <div className="flex flex-col gap-2">
            {diagCommands.map((item, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-2 rounded-lg bg-panel2 border border-border/70 gap-2 text-xs font-mono"
              >
                <div className="flex flex-col min-w-0 flex-1">
                  <span className="text-[10px] text-muted uppercase font-semibold">{item.label}</span>
                  <code className="text-cyan truncate text-[11px] select-all">{item.cmd}</code>
                </div>
                <button
                  type="button"
                  onClick={() => copyToClipboard(item.cmd)}
                  title="Copy command to clipboard"
                  className="px-2 py-1 rounded bg-panel hover:bg-border text-muted hover:text-txt border border-border transition-colors flex items-center gap-1 shrink-0 text-[10px]"
                >
                  {copiedCmd === item.cmd ? (
                    <>
                      <Check size={11} className="text-emerald" />
                      <span className="text-emerald">Copied</span>
                    </>
                  ) : (
                    <>
                      <Copy size={11} />
                      <span>Copy</span>
                    </>
                  )}
                </button>
              </div>
            ))}
          </div>
        </Panel>
      </div>
    </div>
  );
}
