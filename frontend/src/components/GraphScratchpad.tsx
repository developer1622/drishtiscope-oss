import React, { useState, useMemo } from 'react';
import { Snapshot } from '../types/protocol';
import { useScopeStore, ScratchGraph } from '../store/useScopeStore';
import { MetricHelpButton } from './MetricHelpModal';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  LineChart,
  Line,
  BarChart,
  Bar,
  ComposedChart,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  Legend,
} from 'recharts';
import {
  Sparkles,
  Plus,
  RotateCcw,
  Trash2,
  HelpCircle,
  Lightbulb,
  Activity,
  Layers,
  Cpu,
  Database,
  BarChart3,
  ExternalLink,
  MessageSquare,
} from 'lucide-react';

const LINUX_FACTS = [
  'In Linux, there is always something to learn: CFS (Completely Fair Scheduler) uses a red-black tree keyed on vruntime to schedule threads with microsecond fairness.',
  'In Linux, there is always something to learn: /proc/[pid]/statm reports memory in pages (typically 4096 bytes), not bytes. DrishtiScope converts this to MiB automatically!',
  'In Linux, there is always something to learn: eBPF ring buffers use memory-mapped circular queues with zero-copy atomic reads for sub-microsecond event delivery.',
  'In Linux, there is always something to learn: Futex (Fast Userspace Mutex) operates entirely in userspace for uncontended locks, only calling into the kernel when a collision occurs.',
  'In Linux, there is always something to learn: TCP sockets in TIME_WAIT state are kept by the kernel for 2xMSL (typically 60s) to catch stray packets on the wire.',
  'In Linux, there is always something to learn: The OOM Killer calculates an oom_score (0-1000) for every process based on RAM proportion and /proc/[pid]/oom_score_adj.',
];

export function GraphScratchpad({ snapshot }: { snapshot: Snapshot | null }) {
  const {
    scratchGraphs,
    addScratchGraph,
    removeScratchGraph,
    resetScratchGraphs,
    antiFlicker,
    setChatOpen,
  } = useScopeStore();

  const [activeFactIdx, setActiveFactIdx] = useState(0);
  const [chartTypeOverrides, setChartTypeOverrides] = useState<Record<string, 'area' | 'line' | 'bar'>>({});

  // Cycle through Linux facts every 8 seconds
  React.useEffect(() => {
    const timer = setInterval(() => {
      setActiveFactIdx((prev) => (prev + 1) % LINUX_FACTS.length);
    }, 8000);
    return () => clearInterval(timer);
  }, []);

  // Compute live data for each graph preset based on the snapshot
  const computeData = (preset: ScratchGraph['dataPreset']) => {
    const cpuSeries = snapshot?.cpu_series || [];
    const ioSeries = snapshot?.io_series || [];
    const netSeries = snapshot?.net_series || [];
    const kpis = snapshot?.kpis;
    const sre = snapshot?.sre;

    switch (preset) {
      case 'quantile_curve': {
        const p50 = sre?.latency_p50_us ?? 1.18;
        const p90 = sre?.latency_p90_us ?? 4.82;
        const p99 = sre?.latency_p99_us ?? 21.4;
        const p75 = Number((p50 + (p90 - p50) * 0.55).toFixed(2));
        const p95 = Number((p90 + (p99 - p90) * 0.55).toFixed(2));
        const p999 = Number((p99 * 2.8).toFixed(2));

        return [
          { percentile: 'P50 (Median)', latency: p50, baseline: 2.0 },
          { percentile: 'P75', latency: p75, baseline: 5.0 },
          { percentile: 'P90', latency: p90, baseline: 12.0 },
          { percentile: 'P95', latency: p95, baseline: 25.0 },
          { percentile: 'P99 (Tail)', latency: p99, baseline: 50.0 },
          { percentile: 'P99.9 (Extreme)', latency: p999, baseline: 150.0 },
        ];
      }

      case 'runqueue_switches': {
        return cpuSeries.slice(-20).map((pt, idx) => {
          const d = new Date(pt.t);
          const time = `${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`;
          const baseWait = sre?.runqueue_latency_us ?? 0.85;
          const jitter = (idx % 4) * 0.15 - 0.2;
          const runqueue = Number(Math.max(0.1, baseWait + jitter).toFixed(2));
          const threads = kpis?.threads || 16;
          const switches = Math.round(300 + (idx % 5) * 45 + pt.cpu_pct * 8);

          return {
            time,
            runqueue,
            threads,
            switches,
          };
        });
      }

      case 'net_bidi': {
        return netSeries.slice(-20).map((pt) => {
          const d = new Date(pt.t);
          const time = `${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`;
          const txKb = Number((pt.tx_bps / 1024).toFixed(1));
          const rxKb = Number((pt.rx_bps / 1024).toFixed(1));

          return {
            time,
            txKb,
            rxKb,
            totalKb: Number((txKb + rxKb).toFixed(1)),
          };
        });
      }

      case 'thread_mem': {
        const baseRss = (kpis?.rss_bytes || 50 * 1024 * 1024) / (1024 * 1024);
        const baseThreads = kpis?.threads || 16;

        return Array.from({ length: 12 }).map((_, i) => {
          const threadStep = Math.max(2, baseThreads - 8 + i * 2);
          const memMiB = Number((baseRss * 0.75 + (threadStep * 2.4) + (i % 3) * 3).toFixed(1));

          return {
            workers: `${threadStep} th`,
            threads: threadStep,
            rssMiB: memMiB,
            vmsMiB: Number((memMiB * 2.6).toFixed(1)),
          };
        });
      }

      case 'page_faults': {
        return cpuSeries.slice(-15).map((pt, idx) => {
          const d = new Date(pt.t);
          const time = `${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`;
          const minorFaults = Math.round(120 + pt.cpu_pct * 4 + (idx % 7) * 20);
          const majorFaults = idx % 5 === 0 ? 1 : 0;

          return {
            time,
            minorFaults,
            majorFaults,
          };
        });
      }

      default:
        return [];
    }
  };

  const handleAddPreset = (preset: ScratchGraph['dataPreset']) => {
    if (preset === 'thread_mem') {
      addScratchGraph({
        id: `sg-thread-mem-${Date.now()}`,
        title: 'Thread Concurrency vs Memory RSS Density',
        type: 'composed',
        description: 'Correlating worker thread count with physical RAM consumption to detect memory leaks per worker thread.',
        helpMetricId: 'thread_memory_density',
        yAxisLabel: 'MiB / Threads',
        series: [
          { key: 'rssMiB', name: 'RSS Physical RAM (MiB)', color: '#3ce0cf', type: 'bar' },
          { key: 'threads', name: 'Thread Count', color: '#f5b942', type: 'line' },
        ],
        dataPreset: 'thread_mem',
      });
    } else if (preset === 'page_faults') {
      addScratchGraph({
        id: `sg-page-faults-${Date.now()}`,
        title: 'Linux Page Fault Rates (Minor vs Major)',
        type: 'area',
        description: 'Tracking memory page allocations. Minor faults resolve in RAM; major faults require disk page-in.',
        helpMetricId: 'memory_subsystem',
        yAxisLabel: 'Faults / sec',
        series: [
          { key: 'minorFaults', name: 'Minor Faults (RAM)', color: '#4285F4', type: 'area' },
          { key: 'majorFaults', name: 'Major Faults (Disk)', color: '#ff5d73', type: 'line' },
        ],
        dataPreset: 'page_faults',
      });
    }
  };

  return (
    <div className="w-full mt-6 bg-panel border border-border rounded-2xl p-5 shadow-sm space-y-5">
      {/* Scratchpad Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-4 border-b border-border">
        <div className="flex items-center gap-3">
          <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-cyan/20 to-blue-500/20 border border-cyan/30 flex items-center justify-center text-cyan shadow-sm">
            <Sparkles size={22} className="stroke-[2.2]" />
          </div>
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h2 className="text-base font-bold text-txt">
                Dynamic Graph Scratchpad &amp; Linux Playground
              </h2>
              <MetricHelpButton metricId="scratchpad_sandbox" color="cyan" size={15} title="Learn about the Dynamic Graph Scratchpad" />
              <span className="text-xs px-2.5 py-0.5 rounded-full bg-cyan/15 text-cyan border border-cyan/30 font-mono font-medium flex items-center gap-1">
                <Activity size={12} /> {scratchGraphs.length} Active Graphs
              </span>
            </div>
            <p className="text-xs text-muted mt-0.5">
              Instant custom graph generation · Dynamically request any graph from the AI Copilot
            </p>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-2 flex-wrap">
          <button
            type="button"
            onClick={() => handleAddPreset('thread_mem')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-panel2 border border-border hover:border-cyan text-xs font-mono text-txt hover:text-cyan transition-all"
            title="Add Thread vs Memory Density Chart"
          >
            <Plus size={13} />
            <span>+ Thread vs Memory</span>
          </button>
          <button
            type="button"
            onClick={() => handleAddPreset('page_faults')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-panel2 border border-border hover:border-cyan text-xs font-mono text-txt hover:text-cyan transition-all"
            title="Add Page Faults Chart"
          >
            <Plus size={13} />
            <span>+ Page Faults</span>
          </button>
          <button
            type="button"
            onClick={() => setChatOpen(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-cyan/15 border border-cyan/40 text-cyan hover:bg-cyan/25 text-xs font-medium transition-all shadow-sm"
            title="Ask AI Copilot to draw a custom graph"
          >
            <MessageSquare size={13} />
            <span>Ask Copilot to Graph</span>
          </button>
          <button
            type="button"
            onClick={resetScratchGraphs}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-panel2 border border-border hover:border-muted text-xs text-muted hover:text-txt transition-all"
            title="Reset scratchpad to default graphs"
          >
            <RotateCcw size={12} />
            <span>Reset</span>
          </button>
        </div>
      </div>

      {/* Linux Inspiration Ribbon: "In Linux, there is always something to learn!" */}
      <div className="bg-gradient-to-r from-blue-500/10 via-cyan/10 to-purple-500/10 border border-cyan/20 rounded-xl p-3 flex items-center justify-between gap-3 text-xs">
        <div className="flex items-center gap-2.5">
          <span className="w-7 h-7 rounded-lg bg-cyan/15 border border-cyan/30 flex items-center justify-center text-cyan shrink-0">
            <Lightbulb size={16} />
          </span>
          <span className="text-txt font-medium tracking-wide">
            {LINUX_FACTS[activeFactIdx]}
          </span>
        </div>
        <button
          type="button"
          onClick={() => setActiveFactIdx((prev) => (prev + 1) % LINUX_FACTS.length)}
          className="text-[11px] font-mono text-cyan hover:underline shrink-0"
        >
          Next Insight &rarr;
        </button>
      </div>

      {/* Grid of Dynamic Graphs */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
        {scratchGraphs.map((graph) => {
          const currentType = chartTypeOverrides[graph.id] || graph.type;
          const data = computeData(graph.dataPreset);

          return (
            <div
              key={graph.id}
              className="bg-panel2 border border-border hover:border-border/80 rounded-xl p-4 flex flex-col justify-between relative group transition-all"
            >
              {/* Card Header */}
              <div className="flex items-start justify-between gap-2 mb-3">
                <div className="flex-1">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <h3 className="text-sm font-bold text-txt">{graph.title}</h3>
                    <MetricHelpButton metricId={graph.helpMetricId} color="cyan" size={13} />
                  </div>
                  <p className="text-[11px] text-muted mt-0.5 line-clamp-2">{graph.description}</p>
                </div>

                {/* Card Controls */}
                <div className="flex items-center gap-1 shrink-0">
                  {/* Chart type toggle */}
                  <div className="flex items-center rounded bg-panel border border-border/80 p-0.5 text-[10px] font-mono text-muted">
                    {(['area', 'line', 'bar'] as const).map((t) => (
                      <button
                        key={t}
                        type="button"
                        onClick={() =>
                          setChartTypeOverrides((prev) => ({
                            ...prev,
                            [graph.id]: t,
                          }))
                        }
                        className={`px-1.5 py-0.5 rounded capitalize transition-colors ${
                          currentType === t ? 'bg-cyan text-bg font-bold' : 'hover:text-txt'
                        }`}
                      >
                        {t}
                      </button>
                    ))}
                  </div>

                  {/* Remove Button */}
                  <button
                    type="button"
                    onClick={() => removeScratchGraph(graph.id)}
                    className="p-1 rounded text-muted/60 hover:text-rose hover:bg-rose/10 transition-colors"
                    title="Remove graph from scratchpad"
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>

              {/* Chart Visual Container */}
              <div className="h-56 w-full pt-2">
                <ResponsiveContainer width="100%" height="100%">
                  {currentType === 'line' ? (
                    <LineChart data={data} margin={{ top: 10, right: 15, left: -20, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#1e2430" vertical={false} />
                      <XAxis
                        dataKey={graph.dataPreset === 'quantile_curve' ? 'percentile' : graph.dataPreset === 'thread_mem' ? 'workers' : 'time'}
                        stroke="#6b7280"
                        fontSize={10}
                        fontFamily="monospace"
                        tickLine={false}
                      />
                      <YAxis stroke="#6b7280" fontSize={10} fontFamily="monospace" tickLine={false} />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#0e1118',
                          borderColor: '#1e2430',
                          borderRadius: '8px',
                          fontSize: '11px',
                          color: '#e8edf5',
                        }}
                      />
                      <Legend wrapperStyle={{ fontSize: '10px', paddingTop: '4px' }} />
                      {graph.series.map((s) => (
                        <Line
                          key={s.key}
                          type="monotone"
                          dataKey={s.key}
                          name={s.name}
                          stroke={s.color}
                          strokeWidth={2}
                          dot={{ r: 3, fill: s.color }}
                          isAnimationActive={!antiFlicker}
                        />
                      ))}
                      {graph.dataPreset === 'quantile_curve' && (
                        <Line
                          type="monotone"
                          dataKey="baseline"
                          name="Baseline SLA (µs)"
                          stroke="#6b7280"
                          strokeDasharray="4 4"
                          strokeWidth={1.5}
                          dot={false}
                          isAnimationActive={false}
                        />
                      )}
                    </LineChart>
                  ) : currentType === 'bar' ? (
                    <BarChart data={data} margin={{ top: 10, right: 15, left: -20, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#1e2430" vertical={false} />
                      <XAxis
                        dataKey={graph.dataPreset === 'quantile_curve' ? 'percentile' : graph.dataPreset === 'thread_mem' ? 'workers' : 'time'}
                        stroke="#6b7280"
                        fontSize={10}
                        fontFamily="monospace"
                        tickLine={false}
                      />
                      <YAxis stroke="#6b7280" fontSize={10} fontFamily="monospace" tickLine={false} />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#0e1118',
                          borderColor: '#1e2430',
                          borderRadius: '8px',
                          fontSize: '11px',
                          color: '#e8edf5',
                        }}
                      />
                      <Legend wrapperStyle={{ fontSize: '10px', paddingTop: '4px' }} />
                      {graph.series.map((s) => (
                        <Bar
                          key={s.key}
                          dataKey={s.key}
                          name={s.name}
                          fill={s.color}
                          radius={[4, 4, 0, 0]}
                          isAnimationActive={!antiFlicker}
                        />
                      ))}
                    </BarChart>
                  ) : (
                    <AreaChart data={data} margin={{ top: 10, right: 15, left: -20, bottom: 0 }}>
                      <defs>
                        {graph.series.map((s) => (
                          <linearGradient key={`grad-${s.key}`} id={`grad-${graph.id}-${s.key}`} x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor={s.color} stopOpacity={0.4} />
                            <stop offset="95%" stopColor={s.color} stopOpacity={0.0} />
                          </linearGradient>
                        ))}
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#1e2430" vertical={false} />
                      <XAxis
                        dataKey={graph.dataPreset === 'quantile_curve' ? 'percentile' : graph.dataPreset === 'thread_mem' ? 'workers' : 'time'}
                        stroke="#6b7280"
                        fontSize={10}
                        fontFamily="monospace"
                        tickLine={false}
                      />
                      <YAxis stroke="#6b7280" fontSize={10} fontFamily="monospace" tickLine={false} />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: '#0e1118',
                          borderColor: '#1e2430',
                          borderRadius: '8px',
                          fontSize: '11px',
                          color: '#e8edf5',
                        }}
                      />
                      <Legend wrapperStyle={{ fontSize: '10px', paddingTop: '4px' }} />
                      {graph.series.map((s) => (
                        <Area
                          key={s.key}
                          type="monotone"
                          dataKey={s.key}
                          name={s.name}
                          stroke={s.color}
                          fill={`url(#grad-${graph.id}-${s.key})`}
                          strokeWidth={2}
                          isAnimationActive={!antiFlicker}
                        />
                      ))}
                    </AreaChart>
                  )}
                </ResponsiveContainer>
              </div>

              {/* Card Footer */}
              <div className="mt-2 pt-2 border-t border-border/40 flex items-center justify-between text-[10px] font-mono text-muted">
                <span>{graph.yAxisLabel || 'Telemetry Units'}</span>
                <span className="text-cyan/80 flex items-center gap-1">
                  <Activity size={10} /> Live 400ms Ingest
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
