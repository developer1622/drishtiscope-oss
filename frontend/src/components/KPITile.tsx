import React, { ReactNode } from 'react';
import { Sparkline } from './Sparkline';

export function KPITile({ label, value, unit, delta, sparkData, color = '#3dd68c', icon }: { label: string, value: string | number, unit?: string, delta?: number, sparkData?: number[], color?: string, icon?: ReactNode }) {
  return (
    <div className="bg-panel border border-border rounded-xl p-4 flex flex-col justify-between min-w-[150px] flex-1 relative overflow-hidden group">
      <div className="flex justify-between items-start mb-2">
        <span className="text-muted text-sm font-medium">{label}</span>
        {icon && <span className="text-muted opacity-50">{icon}</span>}
      </div>
      
      <div className="flex items-baseline gap-1 relative z-10">
        <span className="text-2xl font-bold text-txt tabular-nums">{value}</span>
        {unit && <span className="text-sm text-muted">{unit}</span>}
      </div>
      
      {delta !== undefined && (
        <div className={`text-xs mt-1 ${delta > 0 ? 'text-green' : delta < 0 ? 'text-rose' : 'text-muted'}`}>
          {delta > 0 ? '+' : ''}{delta}%
        </div>
      )}
      
      {sparkData && (
        <div className="absolute bottom-0 left-0 right-0 h-10 opacity-30 group-hover:opacity-60 transition-opacity">
          <Sparkline data={sparkData} width={200} height={40} color={color} />
        </div>
      )}
    </div>
  );
}
