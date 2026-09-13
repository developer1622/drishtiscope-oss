import React from 'react';
import { SnapshotMeta } from '../types/protocol';
import { Activity, Clock, AlertTriangle, Zap } from 'lucide-react';

export function StatusBar({ meta, schema }: { meta: SnapshotMeta | undefined, schema: number | undefined }) {
  if (!meta) return <div className="h-8 bg-panel border-t border-border" />;

  return (
    <div className="h-8 bg-panel border-t border-border flex items-center px-4 text-xs font-mono text-muted justify-between z-20">
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-2">
          <Zap size={14} className="text-cyan" />
          <span>v{schema || 1}</span>
        </div>
        <div className="flex items-center gap-2">
          <Clock size={14} />
          <span>Up {Math.max(0, meta.uptime_s).toFixed(0)}s</span>
        </div>
        <div className="flex items-center gap-2">
          <Activity size={14} className="text-green" />
          <span>{(meta.event_rate || 0).toFixed(1)} ev/s</span>
        </div>
        {meta.dropped_events > 0 && (
          <div className="flex items-center gap-2 text-rose">
            <AlertTriangle size={14} />
            <span>{meta.dropped_events} dropped</span>
          </div>
        )}
      </div>
      <div>
        AgentScope EBPF
      </div>
    </div>
  );
}
