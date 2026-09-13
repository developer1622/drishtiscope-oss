import React from 'react';
import { ProcessRow } from '../types/protocol';
import { fmtPct, fmtBytes, padPid } from '../utils/format';
import { Badge } from './Badge';
import { Crosshair } from 'lucide-react';

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
  const sorted = [...processes].sort((a, b) => b.cpu_pct - a.cpu_pct);

  const handleRowClick = (p: ProcessRow) => {
    onSelect(p.pid);
    if (onTarget) {
      onTarget(p.comm, p.pid);
    }
  };

  return (
    <div className="w-full text-sm text-left overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="text-muted border-b border-border text-xs sticky top-0 bg-panel z-10">
            <th className="py-2 px-3 font-normal text-right">PID</th>
            <th className="py-2 px-3 font-normal">COMM</th>
            <th className="py-2 px-3 font-normal text-right">CPU%</th>
            <th className="py-2 px-3 font-normal text-right">RSS</th>
            <th className="py-2 px-3 font-normal text-right">THR</th>
            <th className="py-2 px-3 font-normal text-right">FDS</th>
            <th className="py-2 px-3 font-normal text-center">ST</th>
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
            const isTarget =
              activeTargetComm &&
              (p.comm === activeTargetComm ||
                p.comm.toLowerCase() === activeTargetComm.toLowerCase());
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
                  className="py-1.5 px-3 font-mono text-txt truncate max-w-[140px] font-medium"
                  title={p.cmdline || p.comm}
                >
                  <div className="flex items-center gap-1.5">
                    {p.comm}
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
                    className={`text-[11px] font-mono px-2 py-0.5 rounded border transition-all flex items-center gap-1 mx-auto ${
                      isTarget
                        ? 'bg-cyan/10 border-cyan text-cyan'
                        : 'border-border text-muted hover:border-cyan hover:text-txt group-hover:border-border/80'
                    }`}
                  >
                    <Crosshair size={11} />
                    {isTarget ? 'Active' : 'Pick'}
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
