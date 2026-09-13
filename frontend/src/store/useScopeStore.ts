import { create } from 'zustand';
import { Snapshot, EventRow, ConnectionStatus, HelloPayload } from '../types/protocol';

export type TabType = 'story' | 'basic' | 'medium' | 'advanced' | 'complete';
export type ThemeMode = 'light' | 'dark' | 'ubuntu' | 'unix' | 'purple';
export type RefreshRate = '500' | '1000' | '2000' | '5000' | 'manual';

export interface ScratchGraph {
  id: string;
  title: string;
  type: 'area' | 'bar' | 'line' | 'composed';
  description: string;
  helpMetricId: string;
  yAxisLabel?: string;
  series: {
    key: string;
    name: string;
    color: string;
    type?: 'area' | 'bar' | 'line';
  }[];
  dataPreset: 'quantile_curve' | 'runqueue_switches' | 'net_bidi' | 'thread_mem' | 'page_faults';
}

interface ScopeStore {
  snapshot: Snapshot | null;
  events: EventRow[];
  connection: ConnectionStatus;
  mode: 'ebpf' | 'real' | 'mock' | null;
  hello: HelloPayload | null;
  selectedPid: number | null;
  lastSnapshotAt: number | null;
  theme: ThemeMode;
  activeTab: TabType;
  isPaused: boolean;
  timeRange: string;
  activeMetricHelpId: string | null;
  isChatOpen: boolean;
  refreshRate: RefreshRate;
  antiFlicker: boolean;
  scratchGraphs: ScratchGraph[];

  setSnapshot: (s: Snapshot) => void;
  addEvent: (e: EventRow) => void;
  setConnection: (s: ConnectionStatus) => void;
  setMode: (m: 'ebpf' | 'real' | 'mock' | null) => void;
  setHello: (h: HelloPayload) => void;
  setSelectedPid: (pid: number | null) => void;
  setTheme: (t: ThemeMode) => void;
  toggleTheme: () => void;
  setActiveTab: (t: TabType) => void;
  togglePause: () => void;
  setTimeRange: (tr: string) => void;
  openMetricHelp: (id: string) => void;
  closeMetricHelp: () => void;
  setChatOpen: (open: boolean) => void;
  toggleChat: () => void;
  setRefreshRate: (rate: RefreshRate) => void;
  toggleAntiFlicker: () => void;
  addScratchGraph: (graph: ScratchGraph) => void;
  removeScratchGraph: (id: string) => void;
  resetScratchGraphs: () => void;
}

const getInitialTheme = (): ThemeMode => {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('drishti_theme') as ThemeMode;
    if (saved && ['light', 'dark', 'ubuntu', 'unix', 'purple'].includes(saved)) {
      applyThemeClasses(saved);
      return saved;
    }
    applyThemeClasses('light');
  }
  return 'light';
};

const applyThemeClasses = (t: ThemeMode) => {
  if (typeof document !== 'undefined') {
    const html = document.documentElement;
    html.classList.remove('light', 'dark', 'ubuntu', 'unix', 'purple');
    html.classList.add(t);
  }
};

export const useScopeStore = create<ScopeStore>((set) => ({
  snapshot: null,
  events: [],
  connection: 'connecting',
  mode: null,
  hello: null,
  selectedPid: null,
  lastSnapshotAt: null,
  theme: getInitialTheme(),
  activeTab: 'basic',
  isPaused: false,
  timeRange: 'live',
  activeMetricHelpId: null,
  isChatOpen: false,
  refreshRate: (typeof window !== 'undefined' && (localStorage.getItem('drishti_refresh') as RefreshRate)) || '2000',
  antiFlicker: typeof window !== 'undefined' ? localStorage.getItem('drishti_antiflicker') !== 'false' : true,

  setSnapshot: (s) =>
    set((state) => {
      if (state.isPaused) return state;
      return { snapshot: s, lastSnapshotAt: Date.now() };
    }),
  addEvent: (e) =>
    set((state) => {
      if (state.isPaused) return state;
      const newEvents = [...state.events, e];
      if (newEvents.length > 300) {
        newEvents.shift();
      }
      return { events: newEvents };
    }),
  setConnection: (c) => set({ connection: c }),
  setMode: (m) => set({ mode: m }),
  setHello: (h) => set({ hello: h }),
  setSelectedPid: (pid) => set({ selectedPid: pid }),
  togglePause: () => set((state) => ({ isPaused: !state.isPaused })),
  setTimeRange: (tr) => set({ timeRange: tr }),
  openMetricHelp: (id) => set({ activeMetricHelpId: id }),
  closeMetricHelp: () => set({ activeMetricHelpId: null }),
  setChatOpen: (open) => set({ isChatOpen: open }),
  toggleChat: () => set((state) => ({ isChatOpen: !state.isChatOpen })),
  setRefreshRate: (rate) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('drishti_refresh', rate);
    }
    set({ refreshRate: rate });
  },
  toggleAntiFlicker: () =>
    set((state) => {
      const next = !state.antiFlicker;
      if (typeof window !== 'undefined') {
        localStorage.setItem('drishti_antiflicker', String(next));
      }
      return { antiFlicker: next };
    }),
  setTheme: (theme) => {
    applyThemeClasses(theme);
    localStorage.setItem('drishti_theme', theme);
    set({ theme });
  },
  toggleTheme: () =>
    set((state) => {
      const themes: ThemeMode[] = ['light', 'dark', 'ubuntu', 'unix'];
      const nextIdx = (themes.indexOf(state.theme) + 1) % themes.length;
      const nextTheme = themes[nextIdx];
      applyThemeClasses(nextTheme);
      localStorage.setItem('drishti_theme', nextTheme);
      return { theme: nextTheme };
    }),
  setActiveTab: (t) => set({ activeTab: t }),
  scratchGraphs: [
    {
      id: 'sg-quantile',
      title: 'Syscall Latency Quantile Distribution (P50 – P99.9)',
      type: 'line',
      description: 'Kernel latency step function across statistical percentiles to detect severe tail-latency drag.',
      helpMetricId: 'latency_quantile_curve',
      yAxisLabel: 'Latency (µs)',
      series: [
        { key: 'latency', name: 'Latency (µs)', color: '#a855f7', type: 'line' },
      ],
      dataPreset: 'quantile_curve',
    },
    {
      id: 'sg-runqueue',
      title: 'CFS Scheduler Runqueue & Thread Contention',
      type: 'composed',
      description: 'Correlating CPU scheduler wait line (runqueue) against active concurrency switches.',
      helpMetricId: 'scheduler_runqueue',
      yAxisLabel: 'Latency (µs) / Count',
      series: [
        { key: 'runqueue', name: 'Runqueue Wait (µs)', color: '#4285F4', type: 'area' },
        { key: 'threads', name: 'Active Threads', color: '#FBBC04', type: 'line' },
      ],
      dataPreset: 'runqueue_switches',
    },
    {
      id: 'sg-net-bidi',
      title: 'Bi-Directional Network Ingress vs Egress Streaming',
      type: 'area',
      description: 'Dual stream comparing outbound prompt payload (TX) against incoming LLM token stream (RX).',
      helpMetricId: 'net_stream_bidi',
      yAxisLabel: 'Throughput (KB/s)',
      series: [
        { key: 'rxKb', name: 'Ingress RX (KB/s)', color: '#34A853', type: 'area' },
        { key: 'txKb', name: 'Egress TX (KB/s)', color: '#EA4335', type: 'area' },
      ],
      dataPreset: 'net_bidi',
    },
  ],
  addScratchGraph: (graph) =>
    set((state) => {
      // Don't add duplicate IDs
      const filtered = state.scratchGraphs.filter((g) => g.id !== graph.id);
      return { scratchGraphs: [...filtered, graph] };
    }),
  removeScratchGraph: (id) =>
    set((state) => ({
      scratchGraphs: state.scratchGraphs.filter((g) => g.id !== id),
    })),
  resetScratchGraphs: () =>
    set({
      scratchGraphs: [
        {
          id: 'sg-quantile',
          title: 'Syscall Latency Quantile Distribution (P50 – P99.9)',
          type: 'line',
          description: 'Kernel latency step function across statistical percentiles to detect severe tail-latency drag.',
          helpMetricId: 'latency_quantile_curve',
          yAxisLabel: 'Latency (µs)',
          series: [
            { key: 'latency', name: 'Latency (µs)', color: '#a855f7', type: 'line' },
          ],
          dataPreset: 'quantile_curve',
        },
        {
          id: 'sg-runqueue',
          title: 'CFS Scheduler Runqueue & Thread Contention',
          type: 'composed',
          description: 'Correlating CPU scheduler wait line (runqueue) against active concurrency switches.',
          helpMetricId: 'scheduler_runqueue',
          yAxisLabel: 'Latency (µs) / Count',
          series: [
            { key: 'runqueue', name: 'Runqueue Wait (µs)', color: '#4285F4', type: 'area' },
            { key: 'threads', name: 'Active Threads', color: '#FBBC04', type: 'line' },
          ],
          dataPreset: 'runqueue_switches',
        },
        {
          id: 'sg-net-bidi',
          title: 'Bi-Directional Network Ingress vs Egress Streaming',
          type: 'area',
          description: 'Dual stream comparing outbound prompt payload (TX) against incoming LLM token stream (RX).',
          helpMetricId: 'net_stream_bidi',
          yAxisLabel: 'Throughput (KB/s)',
          series: [
            { key: 'rxKb', name: 'Ingress RX (KB/s)', color: '#34A853', type: 'area' },
            { key: 'txKb', name: 'Egress TX (KB/s)', color: '#EA4335', type: 'area' },
          ],
          dataPreset: 'net_bidi',
        },
      ],
    }),
}));
