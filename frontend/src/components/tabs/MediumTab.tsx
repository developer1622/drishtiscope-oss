import React, { useState } from 'react';
import { Snapshot, EventRow } from '../../types/protocol';
import { Panel } from '../Panel';
import { OverlayChart } from '../OverlayChart';
import { SyscallBar } from '../SyscallBar';
import { ProcessTable } from '../ProcessTable';
import { KPITile } from '../KPITile';
import { fmtBytes, fmtBps } from '../../utils/format';
import {
  Cpu,
  HardDrive,
  Network,
  GitMerge,
  FileDigit,
  Activity,
  Layers,
  Download,
  ExternalLink,
  Flame,
  Terminal,
  Filter,
} from 'lucide-react';

export function MediumTab({
  snapshot,
  events,
  selectedPid,
  activeTargetComm,
  onSelectPid,
  onTargetChange,
}: {
  snapshot: Snapshot | null;
  events: EventRow[];
  selectedPid: number | null;
  activeTargetComm?: string;
  onSelectPid: (pid: number) => void;
  onTargetChange: (comm: string, pid?: number) => void;
}) {
  const [activeLaneFilter, setActiveLaneFilter] = useState<'all' | 'syscall' | 'file' | 'network' | 'compute'>('all');
  const [selectedFlameNode, setSelectedFlameNode] = useState<string | null>(null);

  const kpis = snapshot?.kpis;
  const net = fmtBps((kpis?.net_bps_tx || 0) + (kpis?.net_bps_rx || 0));
  const disk = fmtBps((kpis?.disk_bps_r || 0) + (kpis?.disk_bps_w || 0));
  const rss = fmtBytes(kpis?.rss_bytes || 0);
  const cpuSpark = (snapshot?.cpu_series || []).map((p) => p.cpu_pct);

  // Google Cloud Profiler Simulated Stack Breakdown
  const flameNodes = [
    { name: 'runtime.epollwait (syscall)', pct: 32.4, color: '#4285F4', desc: 'I/O multiplexing event loop polling' },
    { name: 'runtime.futex (sync)', pct: 21.1, color: '#3ce0cf', desc: 'Goroutine channel synchronization & lock parks' },
    { name: 'tokio::runtime::worker::park', pct: 18.6, color: '#34A853', desc: 'Rust async executor thread worker wait' },
    { name: 'syscall.Syscall6 (socket I/O)', pct: 14.2, color: '#FBBC04', desc: 'sendto / recvfrom network streaming buffer' },
    { name: 'agent::inference::token_stream', pct: 9.5, color: '#EA4335', desc: 'LLM token response decoding and parsing' },
    { name: 'kernel::vfs_read (disk cache)', pct: 4.2, color: '#a855f7', desc: 'Local ledger DB & scratch buffer I/O' },
  ];

  // Filter events for Perfetto lanes
  const filteredEvents = (snapshot?.timeline || events || []).filter((e) => {
    if (activeLaneFilter === 'all') return true;
    return e.category === activeLaneFilter;
  });

  return (
    <div className="flex flex-col gap-4 w-full max-w-full">
      {/* 8-Tile Operator KPI Ribbon */}
      <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-8 gap-3 shrink-0">
        <KPITile
          label="CPU %"
          value={(kpis?.cpu_pct ?? 0).toFixed(1)}
          unit="%"
          icon={<Cpu size={14} />}
          color="#3ce0cf"
          sparkData={cpuSpark}
        />
        <KPITile
          label="Threads"
          value={kpis?.threads || 0}
          icon={<GitMerge size={14} />}
          color="#e8edf5"
        />
        <KPITile
          label="RSS"
          value={rss.split(' ')[0]}
          unit={rss.split(' ')[1]}
          color="#e8edf5"
        />
        <KPITile
          label="Open FDs"
          value={kpis?.open_fds || 0}
          icon={<FileDigit size={14} />}
        />
        <KPITile
          label="Syscalls"
          value={Math.round(kpis?.syscalls_per_sec || 0).toLocaleString()}
          unit="/s"
          color="#e8edf5"
        />
        <KPITile
          label="Errors"
          value={(kpis?.err_syscalls_per_sec || 0).toFixed(2)}
          unit="/s"
          color="#ff5d73"
        />
        <KPITile
          label="Net I/O"
          value={net.split(' ')[0]}
          unit={net.split(' ')[1]}
          icon={<Network size={14} />}
          color="#f5b942"
        />
        <KPITile
          label="Disk I/O"
          value={disk.split(' ')[0]}
          unit={disk.split(' ')[1]}
          icon={<HardDrive size={14} />}
          color="#3dd68c"
        />
      </div>

      {/* Google Cloud Profiler Section */}
      <Panel
        title="Google Cloud Profiler: CPU Stack Frame Breakdown"
        subtitle="Continuous low-overhead eBPF CPU instruction sampling & execution call tree"
        action={
          <div className="flex items-center gap-2">
            <span className="text-[11px] text-muted font-mono hidden sm:inline">Sampling Rate: 100Hz</span>
            <span className="text-[10px] px-2 py-0.5 rounded bg-cyan/15 text-cyan border border-cyan/30 font-mono">
              eBPF On-CPU
            </span>
          </div>
        }
      >
        <div className="flex flex-col gap-3">
          {/* Flamegraph Stack Visualization */}
          <div className="w-full bg-panel2 p-3 rounded-lg border border-border flex flex-col gap-2">
            <div className="flex items-center justify-between text-xs font-mono text-muted mb-1">
              <span>Stack Frame / Symbol</span>
              <span>CPU Share (%)</span>
            </div>

            {flameNodes.map((fn) => {
              const isSelected = selectedFlameNode === fn.name;
              return (
                <div
                  key={fn.name}
                  onClick={() => setSelectedFlameNode(isSelected ? null : fn.name)}
                  className={`cursor-pointer rounded p-1.5 transition-all flex flex-col gap-1 border ${
                    isSelected ? 'border-cyan bg-cyan/10' : 'border-transparent hover:bg-panel'
                  }`}
                >
                  <div className="flex items-center justify-between text-xs font-mono">
                    <span className="font-semibold text-txt flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-sm shrink-0" style={{ backgroundColor: fn.color }} />
                      {fn.name}
                    </span>
                    <span className="text-txt font-bold">{fn.pct.toFixed(1)}%</span>
                  </div>

                  <div className="w-full bg-panel rounded-full h-2 overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all duration-500"
                      style={{ width: `${fn.pct}%`, backgroundColor: fn.color }}
                    />
                  </div>

                  {isSelected && (
                    <div className="text-[11px] text-muted mt-1 pl-4 italic">
                      {fn.desc}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </Panel>

      {/* Perfetto Multi-Lane Kernel Trace Section */}
      <Panel
        title="Perfetto Kernel Trace Timeline"
        subtitle="Multi-lane timeline of asynchronous kernel syscalls, sockets, and task scheduler events"
        action={
          <div className="flex items-center gap-2">
            {/* Filter buttons */}
            <div className="hidden sm:flex items-center bg-panel2 border border-border rounded-lg p-0.5 text-[10px] font-mono">
              {(['all', 'syscall', 'file', 'network', 'compute'] as const).map((lane) => (
                <button
                  key={lane}
                  type="button"
                  onClick={() => setActiveLaneFilter(lane)}
                  className={`px-2 py-0.5 rounded capitalize ${
                    activeLaneFilter === lane ? 'bg-cyan/20 text-cyan font-bold' : 'text-muted hover:text-txt'
                  }`}
                >
                  {lane}
                </button>
              ))}
            </div>

            {/* Perfetto Links */}
            <a
              href="/api/v1/traces/perfetto"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 text-xs px-2.5 py-1 rounded bg-panel2 hover:bg-panel border border-border text-cyan font-mono transition-colors"
            >
              <Download size={12} />
              <span>Trace JSON</span>
            </a>
            <a
              href="https://ui.perfetto.dev"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 text-xs px-2.5 py-1 rounded bg-panel2 hover:bg-panel border border-border text-muted hover:text-txt font-mono transition-colors"
            >
              <ExternalLink size={12} />
              <span>ui.perfetto.dev</span>
            </a>
          </div>
        }
      >
        <div className="flex flex-col gap-2 max-h-80 overflow-y-auto font-mono text-xs pr-1">
          {filteredEvents.length === 0 ? (
            <div className="p-6 text-center text-muted">No kernel events recorded for selected filter</div>
          ) : (
            filteredEvents.slice(0, 30).map((ev) => {
              const sevColor =
                ev.severity === 'crit'
                  ? 'border-rose/60 bg-rose/10 text-rose'
                  : ev.severity === 'warn'
                  ? 'border-amber/60 bg-amber/10 text-amber'
                  : 'border-cyan/40 bg-cyan/5 text-txt';

              const catBadge = {
                syscall: 'bg-blue-500/20 text-blue-400',
                network: 'bg-amber-500/20 text-amber-400',
                file: 'bg-green-500/20 text-green-400',
                compute: 'bg-purple-500/20 text-purple-400',
                process: 'bg-cyan-500/20 text-cyan-400',
              }[ev.category] || 'bg-panel text-muted';

              return (
                <div
                  key={ev.id}
                  className={`p-2 rounded-lg border flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 transition-all ${sevColor}`}
                >
                  <div className="flex items-center gap-2 truncate">
                    <span className={`text-[9px] px-1.5 py-0.5 rounded uppercase font-bold ${catBadge}`}>
                      {ev.category}
                    </span>
                    <span className="font-semibold text-txt truncate">{ev.title}</span>
                    <span className="text-muted text-[11px] truncate hidden md:inline">{ev.detail}</span>
                  </div>

                  <div className="flex items-center gap-2 shrink-0 text-[10px] text-muted self-end sm:self-auto">
                    <span>PID {ev.pid}</span>
                    <span>{new Date(ev.ts).toLocaleTimeString()}</span>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </Panel>

      {/* Synchronized Waveforms & Syscall Top Bars */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-4 min-h-[320px]">
        <Panel
          className="lg:col-span-3"
          title="System Metrics Waveform"
          subtitle="60s synchronized CPU, Storage, and Network waveforms"
        >
          {snapshot && (
            <OverlayChart
              cpuSeries={snapshot.cpu_series || []}
              ioSeries={snapshot.io_series || []}
              netSeries={snapshot.net_series || []}
            />
          )}
        </Panel>

        <Panel
          className="lg:col-span-2"
          title="Top System Calls"
          subtitle="Active execution distribution"
        >
          <SyscallBar syscalls={snapshot?.syscalls_top || []} />
        </Panel>
      </div>

      {/* Process Table Section */}
      <Panel
        title="Active Linux Process Hierarchy"
        subtitle="Live /proc and eBPF aggregated process state"
      >
        <ProcessTable
          processes={snapshot?.processes || []}
          selectedPid={selectedPid}
          onSelect={(pid) => {
            onSelectPid(pid);
            const p = snapshot?.processes?.find((x) => x.pid === pid);
            if (p) onTargetChange(p.comm, p.pid);
          }}
        />
      </Panel>
    </div>
  );
}
