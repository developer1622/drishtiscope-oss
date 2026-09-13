import React from 'react';

export function ModeChip({ mode }: { mode: 'ebpf' | 'real' | 'mock' | null }) {
  if (!mode) return <div className="skeleton w-24 h-6 rounded-full" />;
  
  if (mode === 'ebpf') {
    return (
      <div
        className="flex items-center gap-1.5 px-2.5 py-1 bg-green/10 border border-green/30 rounded-full text-green text-[11px] font-bold tracking-wide shadow-sm"
        title="Active Linux Kernel eBPF Probes Attached"
      >
        <div className="w-2 h-2 rounded-full bg-green pulse-dot" />
        EBPF LIVE
      </div>
    );
  }

  if (mode === 'real') {
    return (
      <div
        className="flex items-center gap-1.5 px-2.5 py-1 bg-cyan/10 border border-cyan/30 rounded-full text-cyan text-[11px] font-bold tracking-wide shadow-sm"
        title="Live Host Process & Kernel Telemetry (/proc + sys)"
      >
        <div className="w-2 h-2 rounded-full bg-cyan pulse-dot" />
        REAL LIVE
      </div>
    );
  }

  return (
    <div
      className="flex items-center gap-1.5 px-2.5 py-1 bg-amber/10 border border-amber/30 rounded-full text-amber text-[11px] font-bold tracking-wide"
      title="Synthetic Demonstration Data"
    >
      <div className="w-2 h-2 rounded-full bg-amber" />
      MOCK
    </div>
  );
}
