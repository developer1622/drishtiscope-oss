import React from 'react';
import { ProcessRow } from '../types/protocol';
import { fmtPct, fmtBytes, padPid, agentLabel } from '../utils/format';
import { Badge } from './Badge';
import { Crosshair } from 'lucide-react';
import { MetricHelpButton } from './MetricHelpModal';
import { useScopeStore } from '../store/useScopeStore';

export function ProcessTable({
  processes,
  selectedPid,
  activeTargetComm,
  onSelect,
  onTarget,
}: {
  processes: ProcessRow[];
  selectedPid: number | null;
  activeTargetComm?: string;
  onSelect: (pid: number) => void;
  onTarget?: (comm: string, pid: number) => void;
}) {
  const { antiFlicker } = useScopeStore();

  // If antiFlicker is active, sort target first, then stably by PID to prevent jitter
  const sorted = [...processes].sort((a, b) => {
    if (antiFlicker) {
      const aIsTarget = activeTargetComm && (a.comm === activeTargetComm || agentLabel(a.comm, a.cmdline).toLowerCase() === activeTargetComm.toLowerCase());
      const bIsTarget = activeTargetComm && (b.comm === activeTargetComm || agentLabel(b.comm, b.cmdline).toLowerCase() === activeTargetComm.toLowerCase());
      if (aIsTarget && !bIsTarget) return -1;
      if (!aIsTarget && bIsTarget) return 1;
      return a.pid - b.pid;
    }
    return b.cpu_pct - a.cpu_pct;
  });

  const handleRowClick = (p: ProcessRow) => {
    onSelect(p.pid);
    if (onTarget) {
      const label = agentLabel(p.comm, p.cmdline);
      // Cmdline-derived labels (copilot) must stay pid=0 so the backend
      // rediscovers the process after VS Code restarts it.
      onTarget(label, label !== p.comm ? 0 : p.pid);
    }
  };

  return (
    <div className="w-full text-sm text-left overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="text-muted border-b border-border text-xs sticky top-0 bg-panel z-10">
            <th className="py-2 px-3 font-normal text-right">
              <span className="inline-flex items-center gap-1">
                <span>PID</span>
                <MetricHelpButton metricId="pid" color="blue" size={11} />
              </span>
            </th>
            <th className="py-2 px-3 font-normal">
              <span className="inline-flex items-center gap-1">
                <span>COMM</span>
                <MetricHelpButton metricId="comm" color="cyan" size={11} />
              </span>
            </th>
            <th className="py-2 px-3 font-normal text-right">
              <span className="inline-flex items-center gap-1">
                <span>CPU%</span>
                <MetricHelpButton metricId="cpu_pct" color="cyan" size={11} />
              </span>
            </th>
            <th className="py-2 px-3 font-normal text-right">
              <span className="inline-flex items-center gap-1">
                <span>RSS</span>
                <MetricHelpButton metricId="rss_bytes" color="purple" size={11} />
              </span>
            </th>
            <th className="py-2 px-3 font-normal text-right">
              <span className="inline-flex items-center gap-1">
                <span>THR</span>
                <MetricHelpButton metricId="threads" color="blue" size={11} />
              </span>
            </th>
            <th className="py-2 px-3 font-normal text-right">
              <span className="inline-flex items-center gap-1">
                <span>FDS</span>
                <MetricHelpButton metricId="open_fds" color="amber" size={11} />
              </span>
            </th>
            <th className="py-2 px-3 font-normal text-center">
              <span className="inline-flex items-center gap-1">
                <span>ST</span>
                <MetricHelpButton metricId="process_state" color="emerald" size={11} />
              </span>
            </th>
            <th className="py-2 px-3 font-normal text-center">ACTION</th>
          </tr>
        </thead>
        <tbody>
          {sorted.length === 0 && (
            <tr>
              <td colSpan={8} className="py-8 text-center text-muted text-sm">
                No matching process. Set a comm or pid in the header.
              </td>
            </tr>
          )}
          {sorted.map((p) => {
            const label = agentLabel(p.comm, p.cmdline);
            const isTarget =
              !!activeTargetComm &&
              (p.comm === activeTargetComm ||
                p.comm.toLowerCase() === activeTargetComm.toLowerCase() ||
                label.toLowerCase() === activeTargetComm.toLowerCase() ||
                (!!p.cmdline &&
                  p.cmdline.toLowerCase().includes(activeTargetComm.toLowerCase())));
            const isSelected = p.pid === selectedPid || isTarget;
            const stateColor =
              p.state === 'R'
                ? 'green'
                : p.state === 'D'
                ? 'amber'
                : p.state === 'Z'
                ? 'rose'
                : 'muted';

            return (
              <tr
                key={p.pid}
                onClick={() => handleRowClick(p)}
                className={`border-b border-border/50 hover:bg-panel2 cursor-pointer transition-colors group ${
                  isSelected
                    ? 'bg-panel2 border-l-2 border-l-cyan'
                    : 'border-l-2 border-l-transparent'
                }`}
              >
                <td className="py-1.5 px-3 font-mono text-muted text-right whitespace-pre">
                  {padPid(p.pid)}
                </td>
                <td
                  className="py-1.5 px-3 font-mono text-txt truncate max-w-[180px] font-medium"
                  title={p.cmdline || p.comm}
                >
                  <div className="flex items-center gap-1.5">
                    <span className="truncate">{label}</span>
                    {label !== p.comm && (
                      <span className="text-[10px] text-muted font-normal shrink-0">{p.comm}</span>
                    )}
                    {isTarget && (
                      <span className="text-[10px] px-1.5 py-0.2 rounded bg-cyan/20 text-cyan border border-cyan/40">
                        TARGET
                      </span>
                    )}
                  </div>
                </td>
                <td className="py-1.5 px-3 text-right font-mono text-cyan">
                  {fmtPct(p.cpu_pct)}
                </td>
                <td className="py-1.5 px-3 text-right font-mono text-muted">
                  {fmtBytes(p.rss_bytes)}
                </td>
                <td className="py-1.5 px-3 text-right text-muted">{p.threads}</td>
                <td className="py-1.5 px-3 text-right text-muted">{p.open_fds}</td>
                <td className="py-1.5 px-3 text-center">
                  <Badge label={p.state} color={stateColor} className="font-mono" />
                </td>
                <td className="py-1.5 px-3 text-center">
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleRowClick(p);
                    }}
                    title={isTarget ? `Currently observed active agent: ${label}` : `Click to switch active focus to ${label} (PID ${p.pid})`}
                    className={`text-[11px] font-mono px-2.5 py-1 rounded-md border transition-all flex items-center gap-1.5 mx-auto ${
                      isTarget
                        ? 'bg-cyan/20 border-cyan/50 text-cyan font-bold shadow-sm'
                        : 'bg-panel2 border-border text-muted hover:border-cyan/50 hover:text-txt hover:bg-cyan/10'
                    }`}
                  >
                    <Crosshair size={12} className={isTarget ? 'text-cyan animate-pulse' : 'text-muted'} />
                    <span>{isTarget ? 'Active' : 'Set Active'}</span>
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
