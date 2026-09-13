import React from 'react';
import { MetricHelpButton } from './MetricHelpModal';

export function Panel({
  title,
  subtitle,
  children,
  className = '',
  action,
  helpMetricId,
  helpColor,
}: {
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  action?: React.ReactNode;
  helpMetricId?: string;
  helpColor?: 'cyan' | 'blue' | 'emerald' | 'amber' | 'rose' | 'purple';
}) {
  return (
    <div className={`bg-panel border border-border rounded-xl p-4 flex flex-col gap-2 ${className}`}>
      {(title || subtitle || action) && (
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 mb-2">
          <div>
            {title && (
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-txt text-sm sm:text-base">{title}</h3>
                {helpMetricId && (
                  <MetricHelpButton metricId={helpMetricId} color={helpColor || 'cyan'} size={13} />
                )}
              </div>
            )}
            {subtitle && <span className="text-xs text-muted block mt-0.5">{subtitle}</span>}
          </div>
          {action && <div className="shrink-0">{action}</div>}
        </div>
      )}
      <div className="flex-1 min-h-0 overflow-auto">
        {children}
      </div>
    </div>
  );
}
