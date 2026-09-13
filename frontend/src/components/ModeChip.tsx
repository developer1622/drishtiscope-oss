import React from 'react';

export function ModeChip({ mode }: { mode: 'ebpf' | 'mock' | null }) {
  if (!mode) return <div className="skeleton w-24 h-6 rounded-full" />;
  
  if (mode === 'ebpf') {
    return (
      <div className="flex items-center gap-2 px-3 py-1 bg-green/10 border border-green/20 rounded-full text-green text-xs font-bold tracking-wide">
        <div className="w-2 h-2 rounded-full bg-green pulse-dot" />
        EBPF LIVE
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 px-3 py-1 bg-amber/10 border border-amber/20 rounded-full text-amber text-xs font-bold tracking-wide">
      <div className="w-2 h-2 rounded-full bg-amber" />
      MOCK
    </div>
  );
}
