import React from 'react';
import { FileStat } from '../types/protocol';
import { fmtBytes } from '../utils/format';
import { MetricHelpButton } from './MetricHelpModal';

export function FileTable({ files }: { files: FileStat[] }) {
  const maxOps = Math.max(...files.map(f => f.ops_s), 1);

  return (
    <div className="w-full text-sm text-left overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="text-muted border-b border-border text-xs sticky top-0 bg-panel z-10">
            <th className="py-2 px-3 font-normal w-1/2">
              <div className="inline-flex items-center gap-1">
                <span>PATH</span>
                <MetricHelpButton metricId="file_ops" color="cyan" size={12} title="Filesystem path inspected by eBPF openat" />
              </div>
            </th>
            <th className="py-2 px-3 font-normal w-1/4">
              <div className="inline-flex items-center gap-1">
                <span>OPS/S</span>
                <MetricHelpButton metricId="file_ops" color="amber" size={12} title="File operations frequency" />
              </div>
            </th>
            <th className="py-2 px-3 font-normal text-right">
              <div className="inline-flex items-center justify-end gap-1 w-full">
                <span>BYTES/S</span>
                <MetricHelpButton metricId="disk_io" color="emerald" size={12} title="I/O throughput in bytes per second" />
              </div>
            </th>
            <th className="py-2 px-3 font-normal text-right">
              <div className="inline-flex items-center justify-end gap-1 w-full">
                <span>ERRS</span>
                <MetricHelpButton metricId="err_syscalls_per_sec" color="rose" size={12} title="Failed I/O syscalls (ENOENT, EACCES)" />
              </div>
            </th>
          </tr>
        </thead>
        <tbody>
          {files.map((f, i) => {
            const pct = (f.ops_s / maxOps) * 100;
            return (
              <tr key={i} className="border-b border-border/50 hover:bg-panel2 transition-colors">
                <td className="py-1.5 px-3 font-mono text-txt truncate max-w-[200px]" title={f.path} style={{ direction: 'rtl', textAlign: 'left' }}>
                  <bdi>{f.path}</bdi>
                </td>
                <td className="py-1.5 px-3">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-cyan w-10 text-right">{f.ops_s.toFixed(1)}</span>
                    <div className="flex-1 h-1.5 bg-panel2 rounded-full overflow-hidden">
                      <div className="h-full bg-cyan/60" style={{ width: `${pct}%` }} />
                    </div>
                  </div>
                </td>
                <td className="py-1.5 px-3 text-right font-mono text-muted">{fmtBytes(f.bytes_s)}</td>
                <td className={`py-1.5 px-3 text-right font-mono ${f.errors > 0 ? 'text-rose' : 'text-muted/30'}`}>
                  {f.errors}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
