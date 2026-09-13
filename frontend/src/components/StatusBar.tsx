import React from 'react';
import { SnapshotMeta } from '../types/protocol';
import { Activity, Clock, AlertTriangle, Zap, Cpu } from 'lucide-react';
import { MetricHelpButton } from './MetricHelpModal';

/**
 * StatusBar Component
 * Pinned footer displaying live kernel ingest rate, schema version, uptime,
 * dropped eBPF ring buffer events, and engine status.
 * Engineered to remain 100% visible and responsive across all screen dimensions.
 */
export function StatusBar({
  meta,
  schema,
}: {
  meta: SnapshotMeta | undefined;
  schema: number | undefined;
}) {
  if (!meta) {
    return (
      <footer className="shrink-0 w-full min-h-[2.25rem] bg-panel border-t border-border flex items-center px-4 text-xs font-mono text-muted z-30">
        <span className="text-muted/60">Connecting to DrishtiScope telemetry daemon...</span>
      </footer>
    );
  }

  const uptimeStr = Math.max(0, meta.uptime_s).toFixed(0);
  const eventRateStr = (meta.event_rate || 0).toFixed(1);
  const dropped = meta.dropped_events || 0;

  return (
    <footer
      aria-label="System status bar"
      className="shrink-0 w-full min-h-[2.25rem] bg-panel border-t border-border flex items-center justify-between px-3 sm:px-4 py-1 text-xs font-mono text-muted z-30 gap-2 sm:gap-4 overflow-x-auto scrollbar-none shadow-md"
    >
      {/* Left Telemetry Indicators */}
      <div className="flex items-center gap-3 sm:gap-5 shrink-0">
        {/* Protocol Schema Version */}
        <div className="flex items-center gap-1.5 px-2 py-0.5 rounded bg-panel2 border border-border/80 text-[11px]" title={`DrishtiScope Schema Version ${schema || 1}`}>
          <Zap size={13} className="text-cyan shrink-0" />
          <span className="text-txt font-semibold">v{schema || 1}</span>
        </div>

        {/* System Uptime */}
        <div className="flex items-center gap-1.5 text-[11px]" title={`Continuous Uptime: ${uptimeStr} seconds`}>
          <Clock size={13} className="text-muted shrink-0" />
          <span>Up <strong className="text-txt">{uptimeStr}s</strong></span>
        </div>

        {/* Kernel Tracepoint Ingest Rate */}
        <div className="flex items-center gap-1 text-[11px]" title={`eBPF Ring Buffer Ingest Rate: ${eventRateStr} events/sec`}>
          <Activity size={13} className="text-green shrink-0 animate-pulse" />
          <span className="text-txt font-semibold">{eventRateStr}</span>
          <span className="text-muted">ev/s</span>
          <MetricHelpButton metricId="event_rate" color="cyan" size={12} title="Kernel tracepoint ingest rate" />
        </div>

        {/* Ring Buffer Dropped Events */}
        <div className="flex items-center gap-1 text-[11px]">
          {dropped > 0 ? (
            <span className="text-rose flex items-center gap-1 font-bold">
              <AlertTriangle size={13} />
              <span>{dropped} dropped</span>
            </span>
          ) : (
            <span className="text-emerald-400/90 flex items-center gap-1 font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0" />
              <span>0 dropped</span>
            </span>
          )}
          <MetricHelpButton
            metricId="dropped_events"
            color={dropped > 0 ? 'rose' : 'emerald'}
            size={12}
            title="Kernel ring buffer dropped events"
          />
        </div>
      </div>

      {/* Right Engine Badge */}
      <div className="flex items-center gap-2 shrink-0 text-[11px]">
        <div className="hidden md:flex items-center gap-1.5 px-2 py-0.5 rounded bg-panel2 border border-border text-muted">
          <span className="w-2 h-2 rounded-full bg-green animate-pulse shrink-0" />
          <span className="text-txt font-medium">DrishtiScope eBPF Engine</span>
        </div>
        <span className="text-muted/60 hidden lg:inline">·</span>
        <span className="text-cyan/80 text-[10px] hidden lg:inline">
          💡 In Linux, there is always something to learn!
        </span>
      </div>
    </footer>
  );
}
