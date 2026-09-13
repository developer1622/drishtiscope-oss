import React from 'react';
import { SyscallStat } from '../types/protocol';

export function SyscallBar({ syscalls }: { syscalls: SyscallStat[] }) {
  const max = Math.max(...syscalls.map(s => s.count_s), 1);

  return (
    <div className="flex flex-col gap-1 overflow-y-auto">
      {syscalls.map(sc => {
        const pct = (sc.count_s / max) * 100;
        return (
          <div key={sc.name} className="flex items-center gap-2 text-[11px] h-[22px] group relative">
            <div className="w-24 shrink-0 font-mono text-muted truncate text-right">{sc.name}</div>
            <div className="flex-1 h-3 bg-panel2 rounded relative overflow-hidden flex items-center">
              <div 
                className="h-full bg-cyan/40 group-hover:bg-cyan/60 transition-all duration-500 ease-out absolute left-0 top-0"
                style={{ width: `${pct}%` }}
              />
              <span className="relative z-10 ml-2 text-txt font-mono">{sc.count_s.toFixed(0)}</span>
            </div>
            {sc.errors_s > 0 && (
              <div className="w-16 shrink-0 flex items-center gap-1 text-rose" title={`${sc.errors_s} errors/s`}>
                <div className="w-2 h-2 rounded-full bg-rose shrink-0" />
                {sc.errors_s}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
