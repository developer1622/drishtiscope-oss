import React from 'react';
import { TabType } from '../store/useScopeStore';
import { MetricHelpButton } from './MetricHelpModal';
import {
  Bot,
  Activity,
  Layers,
  Cpu,
  ShieldCheck,
  HelpCircle,
  Sparkles,
  Compass,
} from 'lucide-react';

/**
 * TabBanner Component
 * Provides a high-visibility, informative card at the top of each view explaining
 * what the tab contains, what questions it answers, and how engineers can interpret its data.
 */
export function TabBanner({ activeTab }: { activeTab: TabType }) {
  const infoMap = {
    story: {
      title: 'Process Story — Chronological Activity & AI Verdict',
      subtitle: 'What has the target AI agent executed since startup?',
      desc: 'Chronological timeline of tool invocations, files touched, network endpoints opened, and an AI execution verdict to determine whether the agent is healthy, reasoning, or stalled on I/O.',
      icon: <Bot size={18} className="text-cyan" />,
      badge: 'Activity & Narrative',
      badgeColor: 'bg-cyan/15 text-cyan border-cyan/30',
      helpMetricId: 'scratchpad_sandbox',
    },
    basic: {
      title: 'Overview — Core Agent Golden Signals & Reliability Vitals',
      subtitle: 'How healthy and fast is the agent at the kernel level?',
      desc: 'Continuous tracking of microsecond syscall latency quantiles (P50, P90, P99), system call traffic (RPS), SLO availability compliance, error budget burn rate, and synchronized waveforms.',
      icon: <Activity size={18} className="text-[#4285F4]" />,
      badge: 'Vitals & Golden Signals',
      badgeColor: 'bg-blue-500/15 text-[#4285F4] border-blue-500/30',
      helpMetricId: 'slo_availability',
    },
    medium: {
      title: 'Execution & CPU — Call Trees, Traces & Process Hierarchy',
      subtitle: 'Where are CPU cycles and thread scheduling moments spent?',
      desc: 'On-CPU stack profiling flamegraph breakdown (event loops, futex synchronization, token decoding), multi-lane Perfetto trace timeline, top 15 syscalls, and the active Linux process hierarchy.',
      icon: <Layers size={18} className="text-amber" />,
      badge: 'Call Trees & Trace',
      badgeColor: 'bg-amber/15 text-amber border-amber/30',
      helpMetricId: 'flamegraph_cpu',
    },
    advanced: {
      title: 'System Metrics — Deep Telemetry, MQL & Subsystems',
      subtitle: 'What are the memory, storage, and socket characteristics?',
      desc: 'Declarative Metrics Query Language (MQL) query bar, syscall functional categories donut (IO, Sync, Net, Mem, Proc), physical RSS vs cache allocation, storage IOPS trendlines, and TCP socket spectrum.',
      icon: <Cpu size={18} className="text-purple-400" />,
      badge: 'Telemetry & MQL',
      badgeColor: 'bg-purple-500/15 text-purple-400 border-purple-500/30',
      helpMetricId: 'mql_query',
    },
    complete: {
      title: 'Security & Logs — Audit Sandbox & Chronicle Center',
      subtitle: 'Are there unauthorized access attempts or suspicious operations?',
      desc: 'Structured JSON log explorer with severity filtering, Linux security sandbox audit (tracking EACCES permission denials and restricted path access), and 6-axis AI agent workload radar footprint.',
      icon: <ShieldCheck size={18} className="text-emerald-400" />,
      badge: 'Audit & Sandbox',
      badgeColor: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
      helpMetricId: 'ai_workload_radar',
    },
  }[activeTab];

  return (
    <div className="w-full bg-panel border border-border rounded-xl p-3.5 sm:p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 text-xs shadow-sm transition-all">
      <div className="flex items-start sm:items-center gap-3">
        <div className="w-10 h-10 rounded-xl bg-panel2 border border-border flex items-center justify-center shrink-0 shadow-inner">
          {infoMap.icon}
        </div>
        <div>
          <div className="flex items-center gap-2 flex-wrap">
            <h2 className="text-sm font-bold text-txt tracking-tight">{infoMap.title}</h2>
            <span className={`text-[10px] font-mono px-2 py-0.5 rounded-full border ${infoMap.badgeColor}`}>
              {infoMap.badge}
            </span>
            <MetricHelpButton metricId={infoMap.helpMetricId} color="cyan" size={13} />
          </div>
          <p className="text-xs text-muted mt-0.5 leading-relaxed">
            <span className="text-txt/80 font-medium hidden sm:inline">{infoMap.subtitle} </span>
            {infoMap.desc}
          </p>
        </div>
      </div>

      <div className="hidden lg:flex items-center gap-1.5 text-[11px] font-mono text-cyan shrink-0 px-2.5 py-1 rounded-lg bg-panel2 border border-border/70">
        <Compass size={13} className="text-cyan" />
        <span>Linux Observability View</span>
      </div>
    </div>
  );
}
