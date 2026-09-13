export function fmtBytes(n: number): string {
  if (!n || n === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  const i = Math.min(sizes.length - 1, Math.max(0, Math.floor(Math.log(Math.abs(n)) / Math.log(k))));
  return parseFloat((n / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function fmtBps(n: number): string {
  if (!n || n === 0) return '0 bps';
  const k = 1000;
  const sizes = ['bps', 'Kbps', 'Mbps', 'Gbps'];
  const i = Math.min(sizes.length - 1, Math.max(0, Math.floor(Math.log(Math.abs(n)) / Math.log(k))));
  return parseFloat((n / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function fmtPct(n: number): string {
  return n.toFixed(1) + '%';
}

export function fmtNs(n: number): string {
  if (n < 1000) return n + 'ns';
  if (n < 1000000) return (n / 1000).toFixed(1) + 'µs';
  if (n < 1000000000) return (n / 1000000).toFixed(1) + 'ms';
  return (n / 1000000000).toFixed(2) + 's';
}

export function fmtRelTime(ts: string): string {
  const diff = Date.now() - new Date(ts).getTime();
  const sec = Math.floor(diff / 1000);
  if (sec < 60) return sec + 's ago';
  const min = Math.floor(sec / 60);
  if (min < 60) return min + 'm ago';
  const hr = Math.floor(min / 60);
  if (hr < 24) return hr + 'h ago';
  return Math.floor(hr / 24) + 'd ago';
}

export function padPid(pid: number): string {
  return pid.toString().padStart(6, ' ');
}

const COPILOT_CMDLINE =
  /@github\/copilot|copilot-linux|copilot-darwin|copilot-win32|copilot-language-server/i;

/** Stable agent identity when Linux comm is a generic thread name (e.g. MainThread). */
export function agentLabel(comm: string, cmdline?: string): string {
  if (cmdline && COPILOT_CMDLINE.test(cmdline)) {
    return 'copilot';
  }
  return comm;
}
