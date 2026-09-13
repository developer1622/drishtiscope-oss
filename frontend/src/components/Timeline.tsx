import React, { useEffect, useRef, useState } from 'react';
import { EventRow } from '../types/protocol';
import { TimelineEvent } from './TimelineEvent';

export function Timeline({ events }: { events: EventRow[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  const handleScroll = () => {
    if (!containerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  };

  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [events, autoScroll]);

  const visibleEvents = events.slice(-100); // Simple virtualization

  return (
    <div 
      ref={containerRef} 
      onScroll={handleScroll}
      className="flex-1 overflow-y-auto pr-2"
    >
      {visibleEvents.length === 0 ? (
        <div className="text-center text-muted text-sm py-8">No events yet...</div>
      ) : (
        <div className="flex flex-col">
          {visibleEvents.map((e, i) => (
            <TimelineEvent key={e.id || i} event={e} />
          ))}
        </div>
      )}
    </div>
  );
}
