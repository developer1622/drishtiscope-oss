import React from 'react';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function Badge({ label, color = 'muted', className }: { label: React.ReactNode, color?: 'cyan'|'green'|'amber'|'rose'|'muted', className?: string }) {
  const colors = {
    cyan: 'bg-cyan/10 text-cyan border-cyan/20',
    green: 'bg-green/10 text-green border-green/20',
    amber: 'bg-amber/10 text-amber border-amber/20',
    rose: 'bg-rose/10 text-rose border-rose/20',
    muted: 'bg-border text-muted border-border',
  };
  return (
    <span className={cn('px-2 py-0.5 text-xs rounded border inline-flex items-center', colors[color], className)}>
      {label}
    </span>
  );
}
