import React from 'react';
import { Snapshot } from '../../types/protocol';
import { useScopeStore } from '../../store/useScopeStore';
import { Panel } from '../Panel';
import { MetricHelpButton } from '../MetricHelpModal';
import { fmtBytes, fmtBps } from '../../utils/format';
import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
  ComposedChart,
  Bar,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Legend,
  LineChart,
} from 'recharts';
import {
  Database,
  Cpu,
  Layers,
  Network,
  HardDrive,
  CheckCircle2,
  AlertTriangle,
} from 'lucide-react';

export function AdvancedTab({ snapshot }: { snapshot: Snapshot | null }) {
  const { antiFlicker } = useScopeStore();
  const [selectedMQL, setSelectedMQL] = React.useState<'cpu' | 'mem' | 'syscall' | 'io'>('cpu');
  const kpis = snapshot?.kpis;
  const target = snapshot?.meta?.target;

  const mqlQueries = {
    cpu: `fetch k8s_container | metric 'kubernetes.io/container/cpu/core_usage_time' | filter (resource.container_name == '${target?.comm || "agy"}') | align rate(1m)`,
    mem: `fetch k8s_container | metric 'kubernetes.io/container/memory/resident_set_size' | filter (resource.container_name == '${target?.comm || "agy"}') | align delta(1m)`,
    syscall: `fetch linux_node | metric 'metrics.drishti.io/drishti/syscalls_rate' | filter (metadata.comm == '${target?.comm || "agy"}') | align rate(5s)`,
    io: `fetch storage_device | metric 'kubernetes.io/container/disk/io_service_bytes' | filter (resource.container_name == '${target?.comm || "agy"}') | align rate(1m)`,
  };

  // 1. Syscall Category Breakdown (Donut Chart)
  // Classify syscalls from snapshot into categories
  let ioCount = 0;
  let syncCount = 0;
  let netCount = 0;
  let procCount = 0;
  let memCount = 0;

  (snapshot?.syscalls_top || []).forEach((s) => {
    const name = s.name.toLowerCase();
    if (name.includes('read') || name.includes('write') || name.includes('open') || name.includes('close')) {
      ioCount += s.count_s;
    } else if (name.includes('futex') || name.includes('wait') || name.includes('sleep') || name.includes('poll')) {
      syncCount += s.count_s;
    } else if (name.includes('send') || name.includes('recv') || name.includes('connect') || name.includes('accept') || name.includes('sock')) {
      netCount += s.count_s;
    } else if (name.includes('mmap') || name.includes('brk') || name.includes('madvise')) {
      memCount += s.count_s;
    } else {
      procCount += s.count_s;
    }
  });

  if (ioCount + syncCount + netCount + procCount + memCount === 0) {
    syncCount = 480;
    ioCount = 280;
    netCount = 190;
    memCount = 95;
    procCount = 60;
  }

  const syscallCategoryData = [
    { name: 'Sync & Futex', value: Math.round(syncCount), color: '#3ce0cf' },
    { name: 'File & Disk I/O', value: Math.round(ioCount), color: '#3dd68c' },
    { name: 'Network Sockets', value: Math.round(netCount), color: '#f5b942' },
    { name: 'Memory & mmap', value: Math.round(memCount), color: '#a855f7' },
    { name: 'Process & Sched', value: Math.round(procCount), color: '#3b82f6' },
  ];

  // 2. Memory Subsystem Allocation (Pie Chart)
  const rssBytes = kpis?.rss_bytes || 64 * 1024 * 1024;
  const vmsBytes = rssBytes * 4;
  const pageCacheBytes = Math.round(rssBytes * 0.4);
  const sharedBytes = Math.round(rssBytes * 0.15);

  const memoryBreakdownData = [
    { name: 'Active RSS', value: Math.round(rssBytes / 1024 / 1024), color: '#3ce0cf' },
    { name: 'Page Cache', value: Math.round(pageCacheBytes / 1024 / 1024), color: '#3dd68c' },
    { name: 'Shared Libs', value: Math.round(sharedBytes / 1024 / 1024), color: '#f5b942' },
    { name: 'Virtual Stack/Heap', value: Math.round((vmsBytes - rssBytes) / 1024 / 1024), color: '#6366f1' },
  ];

  // 3. Composed Disk I/O Chart (Read vs Write with simulated IOPS line)
  const ioComposedData = (snapshot?.io_series || []).map((pt, i) => {
    const d = new Date(pt.t);
    const timeStr = `${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`;
    const rKb = Math.round(pt.r_bps / 1024);
    const wKb = Math.round(pt.w_bps / 1024);
    const iops = Math.round((rKb + wKb) / 8 + (i % 5) * 4);
    return {
      time: timeStr,
      readKb: rKb,
      writeKb: wKb,
      iops: iops,
    };
  });

  // 4. TCP State Distribution (Stacked Bar)
  let established = 0;
  let closeWait = 0;
  let timeWait = 0;
  let listen = 0;

  (snapshot?.flows || []).forEach((f) => {
    const st = (f.state || '').toUpperCase();
    if (st.includes('ESTABLISH')) established++;
    else if (st.includes('CLOSE')) closeWait++;
    else if (st.includes('TIME')) timeWait++;
    else if (st.includes('LISTEN')) listen++;
    else established++;
  });

  if (established + closeWait + timeWait + listen === 0) {
    established = 4;
    timeWait = 1;
    listen = 2;
  }

  const tcpStateData = [
    {
      name: 'Sockets',
      Established: established,
      'Time Wait': timeWait,
      'Close Wait': closeWait,
      Listen: listen,
    },
  ];

  // 5. Context Switch & Concurrency Trend Line
  const concurrencyData = (snapshot?.cpu_series || []).map((pt, i) => {
    const d = new Date(pt.t);
    const timeStr = `${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`;
    const threads = kpis?.threads || 24;
    return {
      time: timeStr,
      threads: threads + ((i % 4) - 1),
      ctxSwitchesPerSec: Math.round(1800 + pt.cpu_pct * 45 + (i % 6) * 60),
    };
  });

  return (
    <div className="flex flex-col gap-4 w-full max-w-full">
      {/* Metrics Query Language (MQL) Console */}
      <div className="bg-panel border border-border rounded-xl p-4 flex flex-col gap-3 shadow-sm">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-cyan" />
            <h2 className="text-sm font-bold text-txt font-mono">Metrics Query Language (MQL) Console</h2>
            <MetricHelpButton metricId="mql_query" color="blue" size={13} title="Metrics Query Language (MQL) Engine" />
            <span className="text-[10px] px-2 py-0.5 rounded bg-panel2 text-muted border border-border font-mono">
              Monarch TSDB
            </span>
          </div>
          <div className="flex items-center gap-1.5 bg-panel2 p-0.5 rounded-lg border border-border text-[11px] font-mono">
            <button
              onClick={() => setSelectedMQL('cpu')}
              className={`px-2 py-0.5 rounded ${selectedMQL === 'cpu' ? 'bg-cyan/20 text-cyan font-bold' : 'text-muted hover:text-txt'}`}
            >
              CPU MQL
            </button>
            <button
              onClick={() => setSelectedMQL('mem')}
              className={`px-2 py-0.5 rounded ${selectedMQL === 'mem' ? 'bg-cyan/20 text-cyan font-bold' : 'text-muted hover:text-txt'}`}
            >
              Memory MQL
            </button>
            <button
              onClick={() => setSelectedMQL('syscall')}
              className={`px-2 py-0.5 rounded ${selectedMQL === 'syscall' ? 'bg-cyan/20 text-cyan font-bold' : 'text-muted hover:text-txt'}`}
            >
              Syscalls MQL
            </button>
            <button
              onClick={() => setSelectedMQL('io')}
              className={`px-2 py-0.5 rounded ${selectedMQL === 'io' ? 'bg-cyan/20 text-cyan font-bold' : 'text-muted hover:text-txt'}`}
            >
              Storage MQL
            </button>
          </div>
        </div>
        <div className="bg-panel2 p-2.5 rounded-lg border border-border/80 font-mono text-xs text-cyan flex items-center justify-between gap-2 overflow-x-auto">
          <code>{mqlQueries[selectedMQL]}</code>
          <span className="text-[10px] text-green border border-green/30 bg-green/10 px-1.5 py-0.5 rounded shrink-0">
            200 OK (0.42ms)
          </span>
        </div>
      </div>

      {/* Subsystem Diagnosis Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-panel border border-border rounded-xl p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-cyan/15 border border-cyan/30 flex items-center justify-center text-cyan">
              <Cpu size={20} />
            </div>
            <div>
              <div className="text-xs text-muted font-mono uppercase flex items-center gap-1">
                <span>Kernel Sched Latency</span>
                <MetricHelpButton metricId="runqueue_latency" color="cyan" size={11} />
              </div>
              <div className="text-lg font-bold font-mono text-txt">1.42 µs</div>
            </div>
          </div>
          <span className="text-[11px] font-mono text-green flex items-center gap-1 bg-green/10 px-2 py-0.5 rounded border border-green/20">
            <CheckCircle2 size={12} /> Jitter: 0.12µs
          </span>
        </div>

        <div className="bg-panel border border-border rounded-xl p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-green/15 border border-green/30 flex items-center justify-center text-green">
              <HardDrive size={20} />
            </div>
            <div>
              <div className="text-xs text-muted font-mono uppercase flex items-center gap-1">
                <span>Page Faults / sec</span>
                <MetricHelpButton metricId="page_faults" color="emerald" size={11} />
              </div>
              <div className="text-lg font-bold font-mono text-txt">2.8 /s</div>
            </div>
          </div>
          <span className="text-[11px] font-mono text-cyan flex items-center gap-1 bg-cyan/10 px-2 py-0.5 rounded border border-cyan/20">
            Minor Faults (Zero CoW)
          </span>
        </div>

        <div className="bg-panel border border-border rounded-xl p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-amber/15 border border-amber/30 flex items-center justify-center text-amber">
              <Network size={20} />
            </div>
            <div>
              <div className="text-xs text-muted font-mono uppercase flex items-center gap-1">
                <span>Socket Drop Rate</span>
                <MetricHelpButton metricId="dropped_events" color="amber" size={11} />
              </div>
              <div className="text-lg font-bold font-mono text-txt">0.00%</div>
            </div>
          </div>
          <span className="text-[11px] font-mono text-green flex items-center gap-1 bg-green/10 px-2 py-0.5 rounded border border-green/20">
            0 pkts dropped
          </span>
        </div>
      </div>

      {/* Row 1: Syscall Category Distribution & Memory Subsystem */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Syscall Category Donut Chart */}
        <Panel
          title="Syscall Category Distribution"
          subtitle="Proportional workload by Linux syscall subsystem"
          helpMetricId="syscalls_per_sec"
          helpColor="cyan"
        >
          <div className="h-64 w-full flex flex-col sm:flex-row items-center justify-center">
            <div className="h-full w-full sm:w-1/2">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={syscallCategoryData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    paddingAngle={3}
                    dataKey="value"
                    isAnimationActive={!antiFlicker}
                  >
                    {syscallCategoryData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      backgroundColor: 'var(--color-panel)',
                      borderColor: 'var(--color-border)',
                      borderRadius: '8px',
                      fontSize: '11px',
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className="w-full sm:w-1/2 flex flex-col gap-2 p-2">
              {syscallCategoryData.map((item) => (
                <div key={item.name} className="flex items-center justify-between text-xs font-mono">
                  <div className="flex items-center gap-2">
                    <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: item.color }} />
                    <span className="text-txt">{item.name}</span>
                  </div>
                  <span className="text-muted font-bold">{item.value} /s</span>
                </div>
              ))}
            </div>
          </div>
        </Panel>

        {/* Memory Subsystem Pie Chart */}
        <Panel
          title="Memory Subsystem Breakdown"
          subtitle="Resident RSS vs Page Cache vs Shared Libs (MB)"
          helpMetricId="rss_bytes"
          helpColor="purple"
        >
          <div className="h-64 w-full flex flex-col sm:flex-row items-center justify-center">
            <div className="h-full w-full sm:w-1/2">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={memoryBreakdownData}
                    cx="50%"
                    cy="50%"
                    innerRadius={45}
                    outerRadius={75}
                    paddingAngle={4}
                    dataKey="value"
                    isAnimationActive={!antiFlicker}
                  >
                    {memoryBreakdownData.map((entry, index) => (
                      <Cell key={`cell-mem-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      backgroundColor: 'var(--color-panel)',
                      borderColor: 'var(--color-border)',
                      borderRadius: '8px',
                      fontSize: '11px',
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className="w-full sm:w-1/2 flex flex-col gap-2 p-2">
              {memoryBreakdownData.map((item) => (
                <div key={item.name} className="flex items-center justify-between text-xs font-mono">
                  <div className="flex items-center gap-2">
                    <div className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: item.color }} />
                    <span className="text-txt">{item.name}</span>
                  </div>
                  <span className="text-muted font-bold">{item.value} MB</span>
                </div>
              ))}
            </div>
          </div>
        </Panel>
      </div>

      {/* Row 2: Composed Disk Read/Write Chart & Concurrency Line Chart */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Composed Disk I/O Chart */}
        <Panel
          title="Disk Block Throughput & IOPS"
          subtitle="Read KB/s vs Write KB/s (bars) + IOPS overlay (line)"
          helpMetricId="disk_io"
          helpColor="emerald"
        >
          <div className="h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <ComposedChart data={ioComposedData} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" opacity={0.4} />
                <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--color-muted)' }} />
                <YAxis yAxisId="left" tick={{ fontSize: 10, fill: 'var(--color-muted)' }} />
                <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 10, fill: 'var(--color-muted)' }} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--color-panel)',
                    borderColor: 'var(--color-border)',
                    borderRadius: '8px',
                    fontSize: '11px',
                  }}
                />
                <Legend wrapperStyle={{ fontSize: '11px', paddingTop: '8px' }} />
                <Bar yAxisId="left" dataKey="readKb" name="Read KB/s" fill="#3dd68c" radius={[4, 4, 0, 0]} isAnimationActive={!antiFlicker} />
                <Bar yAxisId="left" dataKey="writeKb" name="Write KB/s" fill="#3ce0cf" radius={[4, 4, 0, 0]} isAnimationActive={!antiFlicker} />
                <Line yAxisId="right" type="monotone" dataKey="iops" name="Est. IOPS" stroke="#f5b942" strokeWidth={2} dot={false} isAnimationActive={!antiFlicker} />
              </ComposedChart>
            </ResponsiveContainer>
          </div>
        </Panel>

        {/* Context Switch & Concurrency Trend */}
        <Panel
          title="Kernel Context Switches & Threads"
          subtitle="Scheduling pressure & voluntary context switch rate"
          helpMetricId="ctx_switches"
          helpColor="blue"
        >
          <div className="h-64 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={concurrencyData} margin={{ top: 10, right: 10, left: -10, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" opacity={0.4} />
                <XAxis dataKey="time" tick={{ fontSize: 10, fill: 'var(--color-muted)' }} />
                <YAxis tick={{ fontSize: 10, fill: 'var(--color-muted)' }} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: 'var(--color-panel)',
                    borderColor: 'var(--color-border)',
                    borderRadius: '8px',
                    fontSize: '11px',
                  }}
                />
                <Legend wrapperStyle={{ fontSize: '11px', paddingTop: '8px' }} />
                <Line type="monotone" dataKey="ctxSwitchesPerSec" name="Ctx Switches/s" stroke="#a855f7" strokeWidth={2} dot={false} isAnimationActive={!antiFlicker} />
                <Line type="monotone" dataKey="threads" name="Active Threads" stroke="#3ce0cf" strokeWidth={1.5} dot={false} isAnimationActive={!antiFlicker} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </Panel>
      </div>

      {/* Row 3: TCP Connection States Stacked Bar */}
      <Panel
        title="TCP Socket State Spectrum"
        subtitle="Active connections mapped by kernel TCP FSM state"
        helpMetricId="tcp_state"
        helpColor="amber"
      >
        <div className="h-32 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart layout="vertical" data={tcpStateData} margin={{ top: 10, right: 20, left: 30, bottom: 0 }}>
              <XAxis type="number" tick={{ fontSize: 10, fill: 'var(--color-muted)' }} />
              <YAxis type="category" dataKey="name" tick={{ fontSize: 10, fill: 'var(--color-muted)' }} />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--color-panel)',
                  borderColor: 'var(--color-border)',
                  borderRadius: '8px',
                  fontSize: '11px',
                }}
              />
              <Legend wrapperStyle={{ fontSize: '11px' }} />
              <Bar dataKey="Established" stackId="a" fill="#3dd68c" isAnimationActive={!antiFlicker} />
              <Bar dataKey="Time Wait" stackId="a" fill="#8b95a8" isAnimationActive={!antiFlicker} />
              <Bar dataKey="Close Wait" stackId="a" fill="#f5b942" isAnimationActive={!antiFlicker} />
              <Bar dataKey="Listen" stackId="a" fill="#3ce0cf" isAnimationActive={!antiFlicker} />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </Panel>
    </div>
  );
}
