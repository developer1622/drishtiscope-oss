import React, { useState } from 'react';
import { EventRow } from '../types/protocol';
import { fmtRelTime } from '../utils/format';
import { Badge } from './Badge';

export function TimelineEvent({ event }: { event: EventRow }) {
  const [expanded, setExpanded] = useState(false);
  
  const colors = {
    info: 'bg-cyan',
    warn: 'bg-amber',
    crit: 'bg-rose crit-pulse',
  };

  const badgeColors = {
    process: 'cyan',
    syscall: 'amber',
    file: 'green',
    network: 'cyan',
    compute: 'muted'
  } as const;

  return (
    <div className="flex flex-col gap-1 py-2 border-b border-border/30 last:border-0 hover:bg-panel2/50 px-2 rounded transition-colors cursor-pointer" onClick={() => setExpanded(!expanded)}>
      <div className="flex items-start gap-3">
        <div className="flex flex-col items-center gap-1 mt-1 shrink-0 w-16">
          <div className={`w-2.5 h-2.5 rounded-full ${colors[event.severity]}`} />
          <span className="text-[10px] text-muted font-mono">{fmtRelTime(event.ts)}</span>
        </div>
        
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <Badge label={event.category} color={badgeColors[event.category] || 'muted'} />
            <span className="font-mono text-muted text-xs">[{event.pid}] {event.comm}</span>
            <span className="text-sm font-medium text-txt">{event.title}</span>
          </div>
          <div className="text-sm text-muted break-words">
            {event.detail}
          </div>
        </div>
      </div>
      
      {expanded && event.attrs && Object.keys(event.attrs).length > 0 && (
        <div className="ml-16 mt-2 p-2 bg-bg border border-border rounded text-xs font-mono text-muted overflow-x-auto">
          <pre>{JSON.stringify(event.attrs, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}
