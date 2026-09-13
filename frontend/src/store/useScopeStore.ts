import { create } from 'zustand';
import { Snapshot, EventRow, ConnectionStatus, HelloPayload } from '../types/protocol';

export type TabType = 'basic' | 'medium' | 'advanced' | 'complete';
export type ThemeMode = 'dark' | 'light';

interface ScopeStore {
  snapshot: Snapshot | null;
  events: EventRow[];
  connection: ConnectionStatus;
  mode: 'ebpf' | 'mock' | null;
  hello: HelloPayload | null;
  selectedPid: number | null;
  lastSnapshotAt: number | null;
  theme: ThemeMode;
  activeTab: TabType;
  isPaused: boolean;
  timeRange: string;
  activeMetricHelpId: string | null;

  setSnapshot: (s: Snapshot) => void;
  addEvent: (e: EventRow) => void;
  setConnection: (s: ConnectionStatus) => void;
  setMode: (m: 'ebpf' | 'mock' | null) => void;
  setHello: (h: HelloPayload) => void;
  setSelectedPid: (pid: number | null) => void;
  toggleTheme: () => void;
  setActiveTab: (t: TabType) => void;
  togglePause: () => void;
  setTimeRange: (tr: string) => void;
  openMetricHelp: (id: string) => void;
  closeMetricHelp: () => void;
}

export const useScopeStore = create<ScopeStore>((set) => ({
  snapshot: null,
  events: [],
  connection: 'connecting',
  mode: null,
  hello: null,
  selectedPid: null,
  lastSnapshotAt: null,
  theme: 'dark',
  activeTab: 'basic',
  isPaused: false,
  timeRange: 'live',
  activeMetricHelpId: null,

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
  toggleTheme: () =>
    set((state) => {
      const nextTheme = state.theme === 'dark' ? 'light' : 'dark';
      if (nextTheme === 'light') {
        document.documentElement.classList.add('light');
        document.documentElement.classList.remove('dark');
      } else {
        document.documentElement.classList.remove('light');
        document.documentElement.classList.add('dark');
      }
      return { theme: nextTheme };
    }),
  setActiveTab: (t) => set({ activeTab: t }),
}));
