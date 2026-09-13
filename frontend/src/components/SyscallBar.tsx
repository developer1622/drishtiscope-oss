import React from 'react';
import { SyscallStat } from '../types/protocol';
import { MetricHelpButton } from './MetricHelpModal';

export function SyscallBar({ syscalls }: { syscalls: SyscallStat[] }) {
  const max = Math.max(...syscalls.map(s => s.count_s), 1);

  return (
    <div className="flex flex-col gap-1.5 overflow-y-auto">
      <div className="flex items-center justify-between text-[11px] font-mono text-muted pb-1 border-b border-border/50 px-1">
        <div className="inline-flex items-center gap-1">
          <span>SYSCALL</span>
          <MetricHelpButton metricId="syscalls_per_sec" color="cyan" size={12} title="System call execution rate" />
        </div>
        <div className="inline-flex items-center gap-2">
          <div className="inline-flex items-center gap-1">
            <span>COUNT/S</span>
            <MetricHelpButton metricId="syscalls_per_sec" color="blue" size={12} title="Syscall invocations per second" />
          </div>
          <div className="inline-flex items-center gap-1">
            <span>ERRS</span>
            <MetricHelpButton metricId="err_syscalls_per_sec" color="rose" size={12} title="Syscall errno return codes" />
          </div>
        </div>
      </div>
      {syscalls.map(sc => {
        const pct = (sc.count_s / max) * 100;
        return (
          <div key={sc.name} className="flex items-center gap-2 text-[11px] h-[22px] group relative">
            <div className="w-28 shrink-0 font-mono text-muted truncate text-right">{sc.name}</div>
            <div className="flex-1 h-3.5 bg-panel2 rounded relative overflow-hidden flex items-center">
              <div 
                className="h-full bg-cyan/40 group-hover:bg-cyan/60 transition-all duration-500 ease-out absolute left-0 top-0"
                style={{ width: `${pct}%` }}
              />
              <span className="relative z-10 ml-2 text-txt font-mono text-[10px]">{sc.count_s.toFixed(0)}/s</span>
            </div>
            {sc.errors_s > 0 ? (
              <div className="w-16 shrink-0 flex items-center justify-end gap-1 text-rose" title={`${sc.errors_s} errors/s`}>
                <div className="w-1.5 h-1.5 rounded-full bg-rose shrink-0" />
                <span>{sc.errors_s.toFixed(1)}</span>
              </div>
            ) : (
              <div className="w-16 shrink-0 text-right text-muted/30 font-mono text-[10px]">0</div>
            )}
          </div>
        );
      })}
    </div>
  );
}
