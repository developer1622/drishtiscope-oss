export interface WsMessage {
  v: number;
  kind: 'hello' | 'snapshot' | 'event' | 'heartbeat' | 'error';
  ts: string;
  mode: 'ebpf' | 'real' | 'mock';
  payload: HelloPayload | Snapshot | EventRow | HeartbeatPayload | null;
}

export interface HelloPayload {
  schema: number;
  hostname: string;
  kernel: string;
  target: { pid: number; comm: string };
  ebpf_fail_reason?: string;
}

export interface Snapshot {
  meta: SnapshotMeta;
  processes: ProcessRow[];
  kpis: KPIs;
  sre?: SREMetrics;
  syscalls_top: SyscallStat[];
  files_top: FileStat[];
  flows: NetFlow[];
  cpu_series: SeriesPoint[];
  io_series: IOPoint[];
  net_series: NetPoint[];
  timeline: EventRow[];
}

export interface SREMetrics {
  latency_p50_us: number;
  latency_p90_us: number;
  latency_p99_us: number;
  slo_availability: number;
  error_budget_pct: number;
  burn_rate: number;
  runqueue_latency_us: number;
  saturation_pct: number;
}

export interface SnapshotMeta {
  dropped_events: number;
  event_rate: number;
  uptime_s: number;
  target: { pid: number; comm: string };
}

export interface ProcessRow {
  pid: number;
  tgid: number;
  ppid: number;
  comm: string;
  cmdline: string;
  exe: string;
  uid: number;
  state: string;
  threads: number;
  cpu_pct: number;
  rss_bytes: number;
  vms_bytes: number;
  open_fds: number;
  ctx_switches: number;
  start_time: string;
}

export interface KPIs {
  cpu_pct: number;
  threads: number;
  rss_bytes: number;
  open_fds: number;
  net_bps_tx: number;
  net_bps_rx: number;
  disk_bps_r: number;
  disk_bps_w: number;
  syscalls_per_sec: number;
  err_syscalls_per_sec: number;
  connects_per_sec: number;
}

export interface SyscallStat {
  name: string;
  count_s: number;
  errors_s: number;
}

export interface FileStat {
  path: string;
  ops_s: number;
  bytes_s: number;
  errors: number;
}

export interface NetFlow {
  src: string;
  dst: string;
  sport: number;
  dport: number;
  proto: string;
  state: string;
  bytes_tx: number;
  bytes_rx: number;
  pid: number;
  comm: string;
}

export interface SeriesPoint {
  t: number; // unix ms
  cpu_pct: number;
}

export interface IOPoint {
  t: number;
  r_bps: number;
  w_bps: number;
}

export interface NetPoint {
  t: number;
  tx_bps: number;
  rx_bps: number;
}

export interface EventRow {
  id: string;
  ts: string;
  severity: 'info' | 'warn' | 'crit';
  category: 'process' | 'syscall' | 'file' | 'network' | 'compute';
  pid: number;
  comm: string;
  title: string;
  detail: string;
  attrs: Record<string, unknown>;
}

export interface HeartbeatPayload {
  uptime_s: number;
}

export type ConnectionStatus = 'connecting' | 'connected' | 'reconnecting' | 'offline';
