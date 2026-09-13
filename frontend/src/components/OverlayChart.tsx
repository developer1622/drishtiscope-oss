import React from 'react';
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { SeriesPoint, IOPoint, NetPoint } from '../types/protocol';

interface OverlayChartProps {
  cpuSeries: SeriesPoint[];
  ioSeries: IOPoint[];
  netSeries: NetPoint[];
}

export function OverlayChart({ cpuSeries, ioSeries, netSeries }: OverlayChartProps) {
  // Merge data based on timestamp
  const dataMap = new Map<number, any>();
  
  const merge = (series: any[], mapFn: (d: any) => any) => {
    series.forEach(d => {
      const existing = dataMap.get(d.t) || { t: d.t };
      dataMap.set(d.t, { ...existing, ...mapFn(d) });
    });
  };

  merge(cpuSeries, d => ({ cpu: d.cpu_pct }));
  merge(ioSeries, d => ({ io: d.r_bps + d.w_bps }));
  merge(netSeries, d => ({ net: d.tx_bps + d.rx_bps }));

  const data = Array.from(dataMap.values()).sort((a, b) => a.t - b.t);

  const formatTime = (ts: number) => {
    const d = new Date(ts);
    return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`;
  };

  return (
    <div className="w-full h-64">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3ce0cf" stopOpacity={0.3}/>
              <stop offset="95%" stopColor="#3ce0cf" stopOpacity={0}/>
            </linearGradient>
            <linearGradient id="colorIo" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3dd68c" stopOpacity={0.3}/>
              <stop offset="95%" stopColor="#3dd68c" stopOpacity={0}/>
            </linearGradient>
            <linearGradient id="colorNet" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#f5b942" stopOpacity={0.3}/>
              <stop offset="95%" stopColor="#f5b942" stopOpacity={0}/>
            </linearGradient>
          </defs>
          <XAxis 
            dataKey="t" 
            tickFormatter={formatTime} 
            stroke="#8b95a8" 
            fontSize={11} 
            tickMargin={8}
            minTickGap={30}
          />
          <YAxis yAxisId="left" stroke="#8b95a8" fontSize={11} tickFormatter={v => `${v}%`} />
          <YAxis yAxisId="right" orientation="right" stroke="#8b95a8" fontSize={11} tickFormatter={v => v > 1000000 ? `${(v/1000000).toFixed(1)}M` : `${(v/1000).toFixed(0)}K`} />
          <Tooltip 
            contentStyle={{ backgroundColor: '#0e1118', border: '1px solid #1e2430', borderRadius: '8px' }}
            labelFormatter={(label) => formatTime(label as number)}
          />
          <Area yAxisId="left" type="monotone" dataKey="cpu" stroke="#3ce0cf" fillOpacity={1} fill="url(#colorCpu)" isAnimationActive={false} />
          <Area yAxisId="right" type="monotone" dataKey="io" stroke="#3dd68c" fillOpacity={1} fill="url(#colorIo)" isAnimationActive={false} />
          <Area yAxisId="right" type="monotone" dataKey="net" stroke="#f5b942" fillOpacity={1} fill="url(#colorNet)" isAnimationActive={false} />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
