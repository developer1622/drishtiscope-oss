import React from 'react';
import { NetFlow } from '../types/protocol';
import { fmtBytes } from '../utils/format';
import { Badge } from './Badge';

export function FlowTable({ flows }: { flows: NetFlow[] }) {
  const sorted = [...flows].sort((a, b) => (b.bytes_tx + b.bytes_rx) - (a.bytes_tx + a.bytes_rx));

  return (
    <div className="w-full text-sm text-left overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="text-muted border-b border-border text-xs sticky top-0 bg-panel z-10">
            <th className="py-2 px-3 font-normal">COMM</th>
            <th className="py-2 px-3 font-normal">PROTO</th>
            <th className="py-2 px-3 font-normal">SRC</th>
            <th className="py-2 px-3 font-normal">DST</th>
            <th className="py-2 px-3 font-normal text-right">TX</th>
            <th className="py-2 px-3 font-normal text-right">RX</th>
            <th className="py-2 px-3 font-normal text-center">STATE</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((f, i) => {
            const stateColor = f.state === 'ESTABLISHED' ? 'green' : f.state === 'SYN_SENT' ? 'amber' : 'muted';
            return (
              <tr key={i} className="border-b border-border/50 hover:bg-panel2 transition-colors">
                <td className="py-1.5 px-3 font-mono text-txt">{f.comm}</td>
                <td className="py-1.5 px-3 font-mono text-muted text-xs uppercase">{f.proto}</td>
                <td className="py-1.5 px-3 font-mono text-muted text-xs">{f.src}:{f.sport}</td>
                <td className="py-1.5 px-3 font-mono text-muted text-xs">{f.dst}:{f.dport}</td>
                <td className="py-1.5 px-3 text-right font-mono text-cyan">{fmtBytes(f.bytes_tx)}</td>
                <td className="py-1.5 px-3 text-right font-mono text-green">{fmtBytes(f.bytes_rx)}</td>
                <td className="py-1.5 px-3 text-center">
                  {f.state && <Badge label={f.state} color={stateColor} />}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
