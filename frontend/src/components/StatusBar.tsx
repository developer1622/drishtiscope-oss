import React from 'react';
import { SnapshotMeta } from '../types/protocol';
import { Activity, Clock, AlertTriangle, Zap } from 'lucide-react';
import { MetricHelpButton } from './MetricHelpModal';

export function StatusBar({ meta, schema }: { meta: SnapshotMeta | undefined, schema: number | undefined }) {
  if (!meta) return <div className="h-8 bg-panel border-t border-border" />;

  return (
    <div className="h-8 bg-panel border-t border-border flex items-center px-4 text-xs font-mono text-muted justify-between z-20">
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-1.5">
          <Zap size={14} className="text-cyan" />
          <span>v{schema || 1}</span>
        </div>
        <div className="flex items-center gap-1.5">
          <Clock size={14} />
          <span>Up {Math.max(0, meta.uptime_s).toFixed(0)}s</span>
        </div>
        <div className="flex items-center gap-1.5">
          <Activity size={14} className="text-green" />
          <span>{(meta.event_rate || 0).toFixed(1)} ev/s</span>
          <MetricHelpButton metricId="event_rate" color="cyan" size={11} title="Kernel tracepoint ingest rate" />
        </div>
        <div className="flex items-center gap-1.5">
          {meta.dropped_events > 0 ? (
            <span className="text-rose flex items-center gap-1">
              <AlertTriangle size={14} />
              <span>{meta.dropped_events} dropped</span>
            </span>
          ) : (
            <span className="text-green/80 flex items-center gap-1">
              <span>0 dropped</span>
            </span>
          )}
          <MetricHelpButton metricId="dropped_events" color={meta.dropped_events > 0 ? "rose" : "emerald"} size={11} title="Kernel ring buffer dropped events" />
        </div>
      </div>
      <div className="text-[11px] text-muted">
        DrishtiScope eBPF Engine
      </div>
    </div>
  );
}
