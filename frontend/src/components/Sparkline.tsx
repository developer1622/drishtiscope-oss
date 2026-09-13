import React from 'react';

export function Sparkline({ data, width = 100, height = 30, color = '#3dd68c', filled = true }: { data: number[], width?: number, height?: number, color?: string, filled?: boolean }) {
  if (!data || data.length === 0) return null;
  
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  
  const pts = data.map((d, i) => {
    const x = (i / (data.length - 1)) * width;
    const y = height - ((d - min) / range) * height;
    return `${x},${y}`;
  });

  const polylineStr = pts.join(' ');
  const polygonStr = `0,${height} ${polylineStr} ${width},${height}`;

  return (
    <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" className="overflow-visible">
      {filled && (
        <polygon points={polygonStr} fill={color} opacity={0.2} />
      )}
      <polyline points={polylineStr} fill="none" stroke={color} strokeWidth={1.5} vectorEffect="non-scaling-stroke" />
    </svg>
  );
}
