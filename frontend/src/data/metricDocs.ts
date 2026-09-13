export interface MetricDoc {
  id: string;
  title: string;
  category: 'SRE Golden Signals' | 'Compute & CPU' | 'Memory & RAM' | 'Storage & Disk' | 'Network & Sockets' | 'Processes & Threads' | 'Security & Audit' | 'Kernel & eBPF';
  themeColor: 'cyan' | 'blue' | 'emerald' | 'amber' | 'rose' | 'purple';
  shortDefinition: string;
  juniorAdminExplanation: string;
  howToRead: {
    unit: string;
    healthy: string;
    warning: string;
    critical: string;
  };
  whyAiAgentsCare: string;
  linuxAdminCommand: string;
  kernelDataSource: string;
}

export const METRIC_DOCS: Record<string, MetricDoc> = {
  cpu_pct: {
    id: 'cpu_pct',
    title: 'CPU Utilization (%)',
    category: 'Compute & CPU',
    themeColor: 'cyan',
    shortDefinition: 'Percentage of total CPU core capacity consumed by the observed target process and all its child threads.',
    juniorAdminExplanation:
      'Think of the CPU like the engine of a car. A reading of 25% means one-fourth of a CPU core is actively calculating instructions. On multi-core systems (e.g. 8 cores), a multi-threaded process can reach up to 800% if all cores are running full speed.',
    howToRead: {
      unit: '% (Percentage of 1 Core)',
      healthy: '5% – 60%: Normal steady-state processing.',
      warning: '60% – 90%: Heavy workload, token generation, or compilation.',
      critical: '> 95%: Core starvation, thread spinning, or infinite loop.',
    },
    whyAiAgentsCare:
      'Autonomous AI agents (like agy or grok) spike CPU during local prompt evaluation, tool compilation, vector embeddings, and regex parsing over large context windows.',
    linuxAdminCommand: 'pidstat -u 1 5 -p <PID>',
    kernelDataSource: '/proc/[pid]/stat (fields 14 utime & 15 stime divided by USER_HZ ticks)',
  },

  threads: {
    id: 'threads',
    title: 'Thread Count (LWP)',
    category: 'Processes & Threads',
    themeColor: 'blue',
    shortDefinition: 'Number of active Light-Weight Processes (kernel threads) spawned under this process group/TGID.',
    juniorAdminExplanation:
      'A thread is an individual worker inside a program sharing the same memory. When a process needs to do multiple things simultaneously (like streaming tokens on one thread while writing to disk on another), it creates threads.',
    howToRead: {
      unit: 'Count (Threads)',
      healthy: '4 – 32: Normal Go/Node/Python worker pools.',
      warning: '33 – 128: Elevated concurrency or worker queue buildup.',
      critical: '> 256: Thread leak, OS thread exhaustion risk (EAGAIN).',
    },
    whyAiAgentsCare:
      'Go-based agents use Go runtime goroutines mapped across M:N OS threads; Node/Python agents spawn worker threads for background tool execution.',
    linuxAdminCommand: 'ls -1 /proc/<PID>/task | wc -l',
    kernelDataSource: '/proc/[pid]/status ("Threads:" line)',
  },

  rss_bytes: {
    id: 'rss_bytes',
    title: 'Resident Set Size (RSS Memory)',
    category: 'Memory & RAM',
    themeColor: 'purple',
    shortDefinition: 'The actual amount of physical RAM (not virtual swap or shared files) currently allocated to this process.',
    juniorAdminExplanation:
      'RSS is the real RAM the process is occupying in your physical RAM chips right now. Unlike Virtual Memory (VMS), which is just promised address space, RSS will trigger the Linux Out-Of-Memory (OOM) Killer if it grows too large!',
    howToRead: {
      unit: 'Bytes (KiB / MiB / GiB)',
      healthy: '< 300 MiB: Lightweight CLI agent.',
      warning: '300 MiB – 1.5 GiB: Loaded LLM context window or large git workspace.',
      critical: '> 2.0 GiB: High risk of Linux kernel OOM killer terminating the agent.',
    },
    whyAiAgentsCare:
      'AI agents cache huge conversation transcripts, git diffs, AST parse trees, and vector caches in RAM. If memory leaks occur, the kernel kills the agent with SIGKILL (OOM-kill).',
    linuxAdminCommand: 'ps -o pid,comm,rss,vsz -p <PID>',
    kernelDataSource: '/proc/[pid]/statm (field 2 page count multiplied by 4096 bytes)',
  },

  open_fds: {
    id: 'open_fds',
    title: 'Open File Descriptors (FDs)',
    category: 'Storage & Disk',
    themeColor: 'amber',
    shortDefinition: 'The number of open handles to files, sockets, pipes, and event loops held by the process.',
    juniorAdminExplanation:
      'In Linux, "everything is a file". Every file read, network socket, pipe, and epoll instance consumes one File Descriptor (FD). Each process has an OS limit (`ulimit -n`, typically 1024). Exceeding this limit causes "Too many open files" errors.',
    howToRead: {
      unit: 'Count (File Descriptors)',
      healthy: '< 64: Normal process operations.',
      warning: '65 – 500: High connection count or multiple open workspace files.',
      critical: '> 900 (near 1024): Imminent EMFILE crash ("Too many open files").',
    },
    whyAiAgentsCare:
      'Agents open dozens of files in repositories, maintain WebSocket streams, and create pipes for child processes. Unclosed files will crash tool execution.',
    linuxAdminCommand: 'ls -l /proc/<PID>/fd | wc -l',
    kernelDataSource: 'Count of entries in directory /proc/[pid]/fd/',
  },

  syscalls_per_sec: {
    id: 'syscalls_per_sec',
    title: 'System Calls / Sec (Traffic RPS)',
    category: 'SRE Golden Signals',
    themeColor: 'cyan',
    shortDefinition: 'The frequency of transitions from userspace into kernel space (sys_enter tracepoints) per second.',
    juniorAdminExplanation:
      'Every time a program wants to interact with the outside world (read a file, write to a socket, sleep, or lock a mutex), it must ask the Linux kernel via a "syscall". Higher numbers mean the program is doing intensive system work.',
    howToRead: {
      unit: 'Syscalls/sec (Hz)',
      healthy: '50 – 800: Normal interactive agent execution.',
      warning: '800 – 3,000: Intensive disk I/O, network streaming, or thread synchronization.',
      critical: '> 5,000: Syscall flood, hot futex spinning, or unbuffered write loop.',
    },
    whyAiAgentsCare:
      'Observing syscall velocity reveals whether the agent is blocked waiting for user input or rapidly streaming responses through epoll_wait and futex.',
    linuxAdminCommand: 'perf stat -e "raw_syscalls:sys_enter" -p <PID> -- sleep 2',
    kernelDataSource: 'eBPF tracepoint/raw_syscalls/sys_enter counter',
  },

  err_syscalls_per_sec: {
    id: 'err_syscalls_per_sec',
    title: 'Syscall Errors / Sec',
    category: 'SRE Golden Signals',
    themeColor: 'rose',
    shortDefinition: 'Number of syscalls per second that returned negative errno codes (e.g. EACCES, ENOENT, ECONNREFUSED).',
    juniorAdminExplanation:
      'When a syscall fails (e.g. trying to open a file that does not exist or connecting to a port that is closed), the kernel returns an error code like -ENOENT (-2) or -EACCES (-13). Some errors (like EAGAIN) are benign, but bursts of errors signal bugs or security blocks.',
    howToRead: {
      unit: 'Errors/sec',
      healthy: '< 1.0/s: Clean execution with few missing file lookups.',
      warning: '1.0 – 10.0/s: Retries, polling, or non-blocking socket EAGAIN.',
      critical: '> 10.0/s: Missing dependencies, refused connections, or permission denials.',
    },
    whyAiAgentsCare:
      'Bursts of errors reveal when an agent is searching for missing tools, failing to authenticate, or being restricted by sandbox security policies.',
    linuxAdminCommand: 'strace -e "trace=openat,connect" -c -p <PID>',
    kernelDataSource: 'eBPF tracepoint/raw_syscalls/sys_exit with ret < 0',
  },

  net_throughput: {
    id: 'net_throughput',
    title: 'Network Throughput (TX + RX)',
    category: 'Network & Sockets',
    themeColor: 'emerald',
    shortDefinition: 'Aggregate rate of network bytes transmitted (TX) and received (RX) across all active sockets.',
    juniorAdminExplanation:
      'This measures how much data the process is uploading and downloading over network cards and localhost sockets. It combines outbound model queries and incoming token streams.',
    howToRead: {
      unit: 'bps / KB/s / MB/s',
      healthy: '10 KB/s – 250 KB/s: Typical LLM streaming token rates.',
      warning: '250 KB/s – 5 MB/s: Downloading large dependency packages or docker layers.',
      critical: '> 20 MB/s: Unexpected massive data egress or network flooding.',
    },
    whyAiAgentsCare:
      'AI agents stream prompt completions from remote APIs and communicate with local language server protocols (LSP) and databases.',
    linuxAdminCommand: 'nethogs or iftop -P -p <PID>',
    kernelDataSource: '/proc/[pid]/net/dev and eBPF socket sendto/recvfrom accounting',
  },

  disk_io: {
    id: 'disk_io',
    title: 'Disk Storage I/O Rate (Read + Write)',
    category: 'Storage & Disk',
    themeColor: 'amber',
    shortDefinition: 'Throughput in bytes per second read from and written to permanent storage drives.',
    juniorAdminExplanation:
      'This measures disk drive activity. High read rates happen when loading code files; high write rates occur when logging or saving files to disk. Linux caches files in RAM, so disk I/O represents actual disk access.',
    howToRead: {
      unit: 'Bytes/s (KB/s / MB/s)',
      healthy: '< 500 KB/s: Normal workspace editing and logging.',
      warning: '500 KB/s – 10 MB/s: Heavy git operations or database checkpointing.',
      critical: '> 50 MB/s: High storage latency risk; can freeze system responsiveness.',
    },
    whyAiAgentsCare:
      'AI coding agents frequently read multi-megabyte source trees and append dense event logs into SQLite WAL files.',
    linuxAdminCommand: 'iotop -p <PID> -b -n 2',
    kernelDataSource: '/proc/[pid]/io (read_bytes and write_bytes deltas)',
  },

  p50_latency: {
    id: 'p50_latency',
    title: 'P50 Median Latency',
    category: 'SRE Golden Signals',
    themeColor: 'blue',
    shortDefinition: 'The 50th percentile of kernel syscall execution time (median response time).',
    juniorAdminExplanation:
      'Half of all system calls finish faster than this number. In Linux, typical memory and epoll syscalls complete in 1 to 2 microseconds (µs). If P50 is high, the entire operating system is dragging.',
    howToRead: {
      unit: 'Microseconds (µs)',
      healthy: '< 2.5 µs: Fast kernel execution.',
      warning: '2.5 µs – 10 µs: Moderate kernel lock contention.',
      critical: '> 25 µs: Severe CPU throttling or heavy memory swapping.',
    },
    whyAiAgentsCare:
      'Ensures the agent is not experiencing fundamental OS latency when making basic system calls.',
    linuxAdminCommand: 'perf top -p <PID>',
    kernelDataSource: 'eBPF histogram tracking sys_enter to sys_exit duration',
  },

  p90_latency: {
    id: 'p90_latency',
    title: 'P90 Syscall Latency',
    category: 'SRE Golden Signals',
    themeColor: 'purple',
    shortDefinition: 'The 90th percentile latency: 90% of system calls complete faster than this value.',
    juniorAdminExplanation:
      'While P50 shows the average speed, P90 exposes slower calls (such as file reads waiting for disk or network socket lookups) without being skewed by rare outliers.',
    howToRead: {
      unit: 'Microseconds (µs)',
      healthy: '< 8.0 µs: Excellent kernel responsiveness.',
      warning: '8.0 µs – 30 µs: Noticeable I/O delays.',
      critical: '> 100 µs: Blocked on disk I/O or waiting for network ACKs.',
    },
    whyAiAgentsCare:
      'Provides early warning before tail latency harms real-time streaming interactions.',
    linuxAdminCommand: 'syscount -d 5 -p <PID>',
    kernelDataSource: 'eBPF duration histogram 90th percentile',
  },

  p99_latency: {
    id: 'p99_latency',
    title: 'P99 Tail Latency (Worst 1%)',
    category: 'SRE Golden Signals',
    themeColor: 'rose',
    shortDefinition: 'The 99th percentile: only 1 in 100 system calls experiences latency higher than this number.',
    juniorAdminExplanation:
      'Tail latency represents the "worst case" moments. Even if an agent runs fast 99% of the time, high P99 latency causes stuttering, dropped connections, and perceived freezes.',
    howToRead: {
      unit: 'Microseconds (µs)',
      healthy: '< 35 µs: High performance with no major stalls.',
      warning: '35 µs – 150 µs: Occasional disk flush or mutex contention pauses.',
      critical: '> 500 µs: Kernel lockups, thread deadlocks, or disk stalls.',
    },
    whyAiAgentsCare:
      'AI agents experience user-facing lag when tail latency spikes, often caused by synchronizing large SQLite WAL checkpoints or slow DNS queries.',
    linuxAdminCommand: 'strace -T -p <PID> -o /tmp/trace.log',
    kernelDataSource: 'eBPF duration histogram 99th percentile bucket',
  },

  slo_availability: {
    id: 'slo_availability',
    title: 'SLO Availability (%)',
    category: 'SRE Golden Signals',
    themeColor: 'emerald',
    shortDefinition: 'Service Level Objective compliance score calculated as (Successful Syscalls / Total Syscalls) * 100%.',
    juniorAdminExplanation:
      'Google SRE standard metric indicating what fraction of system interactions succeeded without kernel errors. Standard enterprise targets aim for "Three Nines" (99.9%).',
    howToRead: {
      unit: '% (Target >= 99.9%)',
      healthy: '>= 99.90%: Meeting enterprise Google SRE reliability targets.',
      warning: '99.00% – 99.89%: Error budget being consumed.',
      critical: '< 99.00%: SLO breach! Immediate engineering attention required.',
    },
    whyAiAgentsCare:
      'Monitors whether autonomous background agents run reliably without silent systemic errors or intermittent network dropouts.',
    linuxAdminCommand: 'curl -s http://localhost:8080/metrics | grep agentscope_sre_slo_availability',
    kernelDataSource: 'Calculated over rolling 60-second sliding window',
  },

  error_budget: {
    id: 'error_budget',
    title: 'Error Budget Remaining (%)',
    category: 'SRE Golden Signals',
    themeColor: 'emerald',
    shortDefinition: 'The remaining allowable unreliability buffer for the current 30-day operating window.',
    juniorAdminExplanation:
      'In Google SRE philosophy, 100% reliability is the wrong target because it stifles innovation. The Error Budget defines how many errors the service is allowed to have before new deployments must be paused to focus on stability.',
    howToRead: {
      unit: '% (Remaining Buffer)',
      healthy: '> 70%: Plenty of error budget remaining; safe to deploy.',
      warning: '20% – 70%: Budget depleting; monitor agent failure causes.',
      critical: '< 20%: Budget exhausted! Halt feature changes and fix bugs.',
    },
    whyAiAgentsCare:
      'Allows autonomous agents to self-throttle their execution if their error budget drops too quickly.',
    linuxAdminCommand: 'curl -s http://localhost:8080/metrics | grep agentscope_sre_error_budget',
    kernelDataSource: 'Derived from SLO target delta integrated over uptime',
  },

  burn_rate: {
    id: 'burn_rate',
    title: 'Error Budget Burn Rate',
    category: 'SRE Golden Signals',
    themeColor: 'amber',
    shortDefinition: 'Rate at which the error budget is currently being consumed compared to the allowable baseline rate.',
    juniorAdminExplanation:
      'A burn rate of 1.0x means you will exhaust exactly 100% of your error budget in 30 days. A burn rate of 2.0x means you will run out in 15 days. A burn rate of 14.4x means your budget will vanish in just 2 days!',
    howToRead: {
      unit: 'Multiplier (x)',
      healthy: '< 1.0x: Sustainable error consumption.',
      warning: '1.0x – 2.5x: Faster than acceptable; investigate root cause.',
      critical: '> 3.0x: Rapid budget burn; triggers SRE pager alerts.',
    },
    whyAiAgentsCare:
      'Alerts operators instantly if an agent starts failing repeatedly before an entire outage unfolds.',
    linuxAdminCommand: 'curl -s http://localhost:8080/metrics | grep agentscope_sre_burn_rate',
    kernelDataSource: 'Integrated error rate / permitted error rate ratio',
  },

  runqueue_latency: {
    id: 'runqueue_latency',
    title: 'Runqueue Latency',
    category: 'Compute & CPU',
    themeColor: 'blue',
    shortDefinition: 'Time that runnable threads spend sitting on the CPU scheduler runqueue waiting for a core to become free.',
    juniorAdminExplanation:
      'When your process is ready to work, but all CPU cores are busy doing other tasks, the kernel puts the threads in a waiting line called the "runqueue". This latency measures how long threads wait in line before getting CPU execution time.',
    howToRead: {
      unit: 'Microseconds (µs)',
      healthy: '< 1.5 µs: Cores available immediately; zero queuing.',
      warning: '1.5 µs – 8.0 µs: Mild CPU scheduling contention.',
      critical: '> 15.0 µs: Severe CPU contention; noisy neighbor processes.',
    },
    whyAiAgentsCare:
      'High runqueue latency explains why an agent feels sluggish even when its own CPU % seems modest (indicating host CPU over-subscription).',
    linuxAdminCommand: 'runqlen or uptime (checking 1-min load average vs CPU count)',
    kernelDataSource: 'Kernel CFS (Completely Fair Scheduler) runqueue delta',
  },

  saturation: {
    id: 'saturation',
    title: 'Resource Saturation (%)',
    category: 'SRE Golden Signals',
    themeColor: 'amber',
    shortDefinition: 'Composite metric evaluating CPU capacity, scheduler runqueue wait, and file descriptor limits.',
    juniorAdminExplanation:
      'Saturation tells you how close the system or process is to 100% maximum capacity. Even before reaching 100%, queueing begins and performance drops sharply.',
    howToRead: {
      unit: '% (Composite Saturation)',
      healthy: '< 35%: System has plenty of headroom.',
      warning: '35% – 75%: Moderate load; capacity planning recommended.',
      critical: '> 80%: System saturated; requests will queue and degrade.',
    },
    whyAiAgentsCare:
      'Provides a single high-level health score indicating whether the agent is overloading its Linux environment.',
    linuxAdminCommand: 'sar -q 1 3',
    kernelDataSource: 'Weighted combination of CPU %, runqueue latency, and FD usage',
  },

  pid: {
    id: 'pid',
    title: 'Process ID (PID)',
    category: 'Processes & Threads',
    themeColor: 'blue',
    shortDefinition: 'Unique integer assigned by the Linux kernel to identify the active process.',
    juniorAdminExplanation:
      'Every running program in Linux has a unique number called a PID. You use this number with tools like `kill <PID>`, `top -p <PID>`, or `strace -p <PID>`.',
    howToRead: {
      unit: 'Integer (1 – 4194304)',
      healthy: 'Stable PID indicates a long-running, healthy process.',
      warning: 'PID changing frequently indicates process crashes and restarts.',
      critical: 'PID 1 is init/systemd.',
    },
    whyAiAgentsCare:
      'DrishtiScope dynamically filters eBPF tracepoint probes using this PID so you only see telemetry for your chosen AI agent.',
    linuxAdminCommand: 'pgrep -a <comm-name>',
    kernelDataSource: '/proc/[pid] directory name',
  },

  process_state: {
    id: 'process_state',
    title: 'Process State (R, S, D, Z)',
    category: 'Processes & Threads',
    themeColor: 'cyan',
    shortDefinition: 'The current operating state of the process in the Linux scheduler.',
    juniorAdminExplanation:
      'Linux processes transition through states: **R** (Running/Runnable) is actively calculating; **S** (Interruptible Sleep) is waiting for network/user input; **D** (Uninterruptible Sleep) is waiting on disk/hardware; **Z** (Zombie) is dead but waiting for parent to collect exit code.',
    howToRead: {
      unit: 'State Code',
      healthy: 'R (Running) or S (Sleeping).',
      warning: 'D (Uninterruptible Sleep): Process waiting for disk; cannot be killed.',
      critical: 'Z (Zombie): Parent process failed to wait() for exited child.',
    },
    whyAiAgentsCare:
      'Agents waiting for remote LLM API streaming spend time in S state; heavy compilation puts them in R; disk bottlenecks show as D.',
    linuxAdminCommand: 'ps -o pid,comm,stat -p <PID>',
    kernelDataSource: '/proc/[pid]/stat (field 3 state character)',
  },

  tcp_state: {
    id: 'tcp_state',
    title: 'TCP Socket State',
    category: 'Network & Sockets',
    themeColor: 'emerald',
    shortDefinition: 'The standard RFC 793 TCP finite state machine status of open network connections.',
    juniorAdminExplanation:
      'TCP connections move through states: **ESTABLISHED** (active data exchange), **LISTEN** (waiting for inbound connections), **TIME_WAIT** (gracefully closed, waiting for lingering packets), and **CLOSE_WAIT** (remote closed, local waiting to close).',
    howToRead: {
      unit: 'State String',
      healthy: 'ESTABLISHED: Healthy live socket streaming data.',
      warning: 'TIME_WAIT: Normal after socket close, lasts ~60 seconds.',
      critical: 'CLOSE_WAIT accumulation: Application bug — failed to call close() on socket.',
    },
    whyAiAgentsCare:
      'AI agents open HTTP/2 and WebSocket connections to model providers. Accumulation of CLOSE_WAIT indicates socket leaks that will exhaust ports.',
    linuxAdminCommand: 'ss -tanp | grep <PID>',
    kernelDataSource: '/proc/[pid]/net/tcp matched with /proc/[pid]/fd socket inodes',
  },

  file_ops: {
    id: 'file_ops',
    title: 'File Access Rate (Ops/s)',
    category: 'Storage & Disk',
    themeColor: 'amber',
    shortDefinition: 'Frequency of openat, read, write, and close operations performed against specific filesystem paths.',
    juniorAdminExplanation:
      'Shows which specific files the process is touching the most. For example, high ops on `transcript.jsonl` indicates prompt logging, while ops on `ledger.db` indicates state checkpointing.',
    howToRead: {
      unit: 'Ops/sec',
      healthy: '1 – 50 ops/s: Normal file reads and writes.',
      warning: '50 – 500 ops/s: High frequency logging or scanning directories.',
      critical: '> 1,000 ops/s: Unbuffered loop repeatedly opening the same file.',
    },
    whyAiAgentsCare:
      'Helps discover if an agent is repeatedly re-reading huge context files instead of caching them in memory.',
    linuxAdminCommand: 'fatrace -c -p <PID>',
    kernelDataSource: 'eBPF tracepoint/syscalls/sys_enter_openat captured paths',
  },

  chronicle_security: {
    id: 'chronicle_security',
    title: 'Chronicle Security Sandbox Audits',
    category: 'Security & Audit',
    themeColor: 'rose',
    shortDefinition: 'Kernel-level audit of security violations, unauthorized path accesses, and refused outbound connections.',
    juniorAdminExplanation:
      'Inspired by Google Chronicle Security Command Center, this audits whether an autonomous agent attempted to access sensitive system files (like `/etc/shadow` or `/proc/kallsyms`) or connect to blocked external IP addresses.',
    howToRead: {
      unit: 'Audit Events',
      healthy: '0 Violations: Clean sandboxed execution.',
      warning: 'Occasional EACCES on non-critical files.',
      critical: 'Attempts to read /etc/shadow or load unapproved kernel BPF programs.',
    },
    whyAiAgentsCare:
      'Autonomous agents run untrusted shell tools. Sandbox auditing guarantees that prompt injections cannot escalate privileges on the host.',
    linuxAdminCommand: 'dmesg -T | grep -i "denied"',
    kernelDataSource: 'eBPF security sandbox filter on openat & connect return codes',
  },

  perfetto_trace: {
    id: 'perfetto_trace',
    title: 'Perfetto Kernel Trace Timeline',
    category: 'Kernel & eBPF',
    themeColor: 'purple',
    shortDefinition: 'High-resolution multi-lane event timeline format fully compatible with ui.perfetto.dev.',
    juniorAdminExplanation:
      'Perfetto is Google\'s open-source performance instrumentation tool. It displays execution tracks (threads, syscalls, files, and network flows) on a synchronized timeline down to microsecond accuracy.',
    howToRead: {
      unit: 'Timeline Lanes',
      healthy: 'Evenly distributed execution lanes with no prolonged thread locks.',
      warning: 'Gaps where threads stall waiting for slow external responses.',
      critical: 'Extended red blocks representing blocked I/O or mutex contention.',
    },
    whyAiAgentsCare:
      'Allows engineers to export traces via 1-click download and visualize exact execution timelines in ui.perfetto.dev.',
    linuxAdminCommand: 'curl -s http://localhost:8080/api/v1/traces/perfetto -o trace.json',
    kernelDataSource: 'Exported from DrishtiScope memory ring buffer in Perfetto JSON format',
  },

  mql_query: {
    id: 'mql_query',
    title: 'Cloud Monitoring MQL Engine',
    category: 'Kernel & eBPF',
    themeColor: 'blue',
    shortDefinition: 'Google Cloud Monitoring Query Language (MQL) syntax for filtering and aggregating agent telemetry.',
    juniorAdminExplanation:
      'MQL is Google\'s declarative query language for cloud observability. It allows writing expressive queries like `fetch process::cpu_utilization | filter comm == "agy" | group_by 1m, mean` to query operational metrics.',
    howToRead: {
      unit: 'Query String',
      healthy: 'Presets provided for CPU, Memory, Syscalls, and Storage I/O.',
      warning: 'Custom queries should match target process fields.',
      critical: 'Syntax errors will show validation hints.',
    },
    whyAiAgentsCare:
      'Enables enterprise SREs to integrate DrishtiScope telemetry into existing Google Cloud monitoring dashboards and alerting policies.',
    linuxAdminCommand: 'curl -s http://localhost:8080/api/history?comm=<comm>',
    kernelDataSource: 'DrishtiScope SQLite TSDB query engine',
  },

  event_rate: {
    id: 'event_rate',
    title: 'Event Ingest Rate',
    category: 'Kernel & eBPF',
    themeColor: 'cyan',
    shortDefinition: 'The throughput of kernel tracepoint records delivered from eBPF ring buffer to userspace.',
    juniorAdminExplanation:
      'Measures the ingestion speed of DrishtiScope\'s daemon. It shows how many events per second the Go engine is consuming, enriching, and streaming to the web dashboard.',
    howToRead: {
      unit: 'Events/sec',
      healthy: '50 – 500 ev/s: Standard operational telemetry stream.',
      warning: '500 – 2,000 ev/s: High activity burst.',
      critical: '> 3,000 ev/s: Ring buffer nearing saturation limit.',
    },
    whyAiAgentsCare:
      'Ensures the observability platform itself remains lightweight and does not add overhead to the running AI agent.',
    linuxAdminCommand: 'curl -s http://localhost:8080/api/meta',
    kernelDataSource: 'Userspace event counter divided by elapsed uptime',
  },

  dropped_events: {
    id: 'dropped_events',
    title: 'Dropped Kernel Events',
    category: 'Kernel & eBPF',
    themeColor: 'rose',
    shortDefinition: 'Count of tracepoint events discarded by the kernel because the 16MB eBPF ring buffer was full.',
    juniorAdminExplanation:
      'When the kernel generates events faster than the userspace Go daemon can read them, the ring buffer fills up and drops events to protect system stability. Ideally, this number should always be zero.',
    howToRead: {
      unit: 'Count (Dropped Records)',
      healthy: '0 Dropped: Complete, 100% loss-free telemetry.',
      warning: '1 – 100 Dropped: Brief burst during extreme syscall spikes.',
      critical: '> 1,000 Dropped: Daemon cannot keep up; increase SNAPSHOT_MS or buffer size.',
    },
    whyAiAgentsCare:
      'A zero dropped event count guarantees that no security incidents or critical syscall anomalies were missed.',
    linuxAdminCommand: 'bpftool map dump name drop_count',
    kernelDataSource: 'BPF_MAP_TYPE_PERCPU_ARRAY drop_count in agent.bpf.c',
  },

  vms_bytes: {
    id: 'vms_bytes',
    title: 'Virtual Memory Size (VMS / VSZ)',
    category: 'Memory & RAM',
    themeColor: 'purple',
    shortDefinition: 'Total amount of virtual memory address space mapped by the process, including shared libraries and reserved heap.',
    juniorAdminExplanation:
      'Virtual memory is what the program *thinks* it has reserved. In Go and modern runtimes, runtimes reserve large 64-bit address space arenas up front, so VMS is often 4x to 10x larger than the actual physical RAM (RSS) being used. It is completely normal for VMS to be large.',
    howToRead: {
      unit: 'Bytes (MiB / GiB)',
      healthy: '100 MiB – 4 GiB: Standard Go / Rust / C++ runtime address allocation.',
      warning: '4 GiB – 32 GiB: Large memory-mapped database or tensor arrays.',
      critical: '> 64 GiB on 32-bit machines or address space exhaustion.',
    },
    whyAiAgentsCare:
      'Memory mmap allocations for large LLM model weights or SQLite WAL files appear in VMS before being touched into physical RSS memory.',
    linuxAdminCommand: 'ps -o pid,comm,vsz,rss -p <PID>',
    kernelDataSource: '/proc/[pid]/statm (field 1 size multiplied by 4096 bytes)',
  },

  ctx_switches: {
    id: 'ctx_switches',
    title: 'Context Switches / Sec',
    category: 'Processes & Threads',
    themeColor: 'blue',
    shortDefinition: 'Rate at which the Linux CPU scheduler saves one thread state and restores another to share the core.',
    juniorAdminExplanation:
      'A context switch is like a worker putting down one task and picking up another. **Voluntary** switches happen when a thread finishes or waits for I/O (good). **Involuntary** switches happen when the OS forcibly preempts a thread because its time slice expired (can cause overhead if too high).',
    howToRead: {
      unit: 'Switches/sec',
      healthy: '500 – 4,000/s: Balanced thread scheduling and asynchronous I/O.',
      warning: '4,000 – 15,000/s: High concurrency contention or frequent mutex locking.',
      critical: '> 30,000/s: "Thrashing" — CPU spends more time switching tasks than doing actual work!',
    },
    whyAiAgentsCare:
      'Go and Rust agent runtimes manage user-space lightweight tasks (goroutines/tokio). High kernel context switches indicate OS thread lock contention.',
    linuxAdminCommand: 'pidstat -w 1 5 -p <PID>',
    kernelDataSource: '/proc/[pid]/status (voluntary_ctxt_switches + nonvoluntary_ctxt_switches)',
  },

  page_faults: {
    id: 'page_faults',
    title: 'Page Faults / Sec',
    category: 'Memory & RAM',
    themeColor: 'cyan',
    shortDefinition: 'Rate of virtual memory page translations handled by the kernel memory subsystem.',
    juniorAdminExplanation:
      'A **minor page fault** is fast and normal: the program accesses memory it reserved, and the OS allocates a physical RAM page without reading disk. A **major page fault** is slow: the page must be read from disk (swap or file), which can take milliseconds.',
    howToRead: {
      unit: 'Faults/sec',
      healthy: '< 50/s minor, 0 major: Smooth memory allocation.',
      warning: '50 – 500/s minor: Active heap expansion during tool execution.',
      critical: '> 5 major/s: System is swapping to disk (RAM exhaustion)!',
    },
    whyAiAgentsCare:
      'Major page faults freeze agent token generation while the OS fetches memory from disk swap.',
    linuxAdminCommand: 'sar -B 1 3',
    kernelDataSource: '/proc/[pid]/stat (fields 10 minflt & 12 majflt)',
  },

  tcp_flows: {
    id: 'tcp_flows',
    title: 'TCP Network Flows & Sockets',
    category: 'Network & Sockets',
    themeColor: 'emerald',
    shortDefinition: 'Real-time table of active socket descriptors mapped to local and remote IP addresses, ports, and states.',
    juniorAdminExplanation:
      'Every outbound HTTP request, WebSocket stream, or database query has an entry here. DrishtiScope inspects `/proc/[pid]/fd` socket inodes and correlates them with the kernel TCP table to show exact connections.',
    howToRead: {
      unit: 'Flow Table',
      healthy: 'ESTABLISHED connections with consistent TX and RX bytes.',
      warning: 'Repeated SYN_SENT indicating network timeouts or firewall drops.',
      critical: 'Accumulation of unclosed sockets or connection storms.',
    },
    whyAiAgentsCare:
      'Autonomous agents connect to cloud LLMs (OpenAI, Anthropic, Gemini, Grok) and local tool endpoints. Flow monitoring exposes connection health.',
    linuxAdminCommand: 'ss -tanp | grep <PID>',
    kernelDataSource: '/proc/[pid]/net/tcp matched with /proc/[pid]/fd socket inodes',
  },

  sqlite_tsdb: {
    id: 'sqlite_tsdb',
    title: 'SQLite TSDB Telemetry Storage',
    category: 'Storage & Disk',
    themeColor: 'amber',
    shortDefinition: 'Zero-dependency embedded time-series database with Write-Ahead Logging (WAL) for persistent telemetry.',
    juniorAdminExplanation:
      'Unlike cloud agents that depend on massive external servers, DrishtiScope embeds a blazing-fast local SQLite database. Snapshots and kernel events are atomically written with WAL mode, giving you historical playback with zero configuration.',
    howToRead: {
      unit: 'Records & File Size',
      healthy: 'Sub-millisecond write latency, automatic rolling retention.',
      warning: 'WAL file growing if checkpointing is delayed.',
      critical: 'Disk full preventing persistence.',
    },
    whyAiAgentsCare:
      'Guarantees audit trails and post-mortem incident playback even if network connectivity is severed.',
    linuxAdminCommand: 'sqlite3 drishtiscope.db "SELECT count(*), datetime(min(t)/1000, \'unixepoch\') FROM snapshots;"',
    kernelDataSource: 'drishtiscope.db SQLite WAL persistence engine',
  },

  ai_workload_radar: {
    id: 'ai_workload_radar',
    title: 'AI Agent Workload Profile (Radar)',
    category: 'SRE Golden Signals',
    themeColor: 'purple',
    shortDefinition: 'Multi-dimensional footprint comparing autonomous agent compute, memory, I/O, and syscall intensity against standard daemons.',
    juniorAdminExplanation:
      'Autonomous AI agents have unique behavioral signatures: they stream network tokens, execute frequent sub-process forks, maintain large context caches in memory, and perform bursts of file operations. The radar chart shows how your agent behaves across 6 key axes.',
    howToRead: {
      unit: 'Intensity Score (0 – 100)',
      healthy: 'Balanced polygon without extreme spikes on Security Sensitivity.',
      warning: 'High Syscall Volatility (> 85) during tight loops.',
      critical: 'Security Sensitivity > 75: Agent attempting forbidden operations.',
    },
    whyAiAgentsCare:
      'Helps operators distinguish between normal agent reasoning bursts vs runaway tool loops.',
    linuxAdminCommand: 'top -p <PID> -b -n 1',
    kernelDataSource: 'Aggregate metrics normalized against baseline kernel profile',
  },

  flamegraph_cpu: {
    id: 'flamegraph_cpu',
    title: 'Cloud Profiler: On-CPU Flame Graph',
    category: 'Compute & CPU',
    themeColor: 'cyan',
    shortDefinition: 'Kernel and userspace call-stack symbol sampling at 100Hz to pinpoint where CPU cycles are spent.',
    juniorAdminExplanation:
      'Instead of guessing why an agent is taking CPU, a flame graph records the exact function call tree. Wide bars mean more CPU time was spent in that function (e.g. `runtime.futex` vs `agent.token_stream`).',
    howToRead: {
      unit: '% CPU Share',
      healthy: 'Work distributed across expected functions (event loop, JSON parsing).',
      warning: 'Single function consuming > 70% of CPU.',
      critical: 'Spinlock or deadlock loops dominating the profile.',
    },
    whyAiAgentsCare:
      'Allows optimizing prompt token decoding, tool serialization, and goroutine synchronization.',
    linuxAdminCommand: 'perf record -F 99 -p <PID> -g -- sleep 5 && perf report',
    kernelDataSource: 'eBPF perf_event instruction pointer sampling',
  },

  comm: {
    id: 'comm',
    title: 'Command Name (comm)',
    category: 'Processes & Threads',
    themeColor: 'cyan',
    shortDefinition: 'Short 16-character task name of the executable stored in task_struct->comm.',
    juniorAdminExplanation:
      'In Linux, the kernel limits the task name to 16 characters (TASK_COMM_LEN). This is the name shown in top, ps, and /proc/[pid]/comm (e.g. "agy", "node", "python3", "grok"). It is what you match when targeting processes.',
    howToRead: {
      unit: 'String (up to 16 chars)',
      healthy: 'Matches expected binary name (e.g. agy, python3, go).',
      warning: 'Truncated names when executable name exceeds 15 characters.',
      critical: 'Disguised process names used by malware or unrecognized binaries.',
    },
    whyAiAgentsCare:
      'DrishtiScope targets agents by their comm name or PID, filtering kernel tracepoints to observe only that specific AI agent.',
    linuxAdminCommand: 'cat /proc/<PID>/comm',
    kernelDataSource: '/proc/[pid]/comm and task_struct->comm in eBPF',
  },
};
