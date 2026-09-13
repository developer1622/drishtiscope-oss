// SPDX-License-Identifier: GPL-2.0 OR MIT
/*
 * AgentScope eBPF tracer — agent.bpf.c
 *
 * Observes a target process across:
 *   - Process lifecycle: exec, exit
 *   - Syscall counts / errors / IO bytes (raw_syscalls)
 *   - File activity: openat path + errno
 *   - Networking: connect (IPv4 dest) + accept
 *
 * Filtering: userspace pins a tgid and/or a comm prefix in `config`,
 * and may add extra tgids (children) to the `targets` hash.
 * If nothing is pinned, only exec events are emitted so userspace can
 * discover the process starting up — we never attach as "trace everything".
 *
 * Compile (x86_64):
 *   clang -g -O2 -target bpf -D__TARGET_ARCH_x86 \
 *     -I/usr/include/x86_64-linux-gnu -I/usr/include \
 *     -c agent.bpf.c -o agent.bpf.o
 *
 * The userspace loader embeds the resulting object.
 */

#include <linux/bpf.h>
#include <linux/types.h>
#include <bpf/bpf_helpers.h>

char _license[] SEC("license") = "GPL";

/* ─── Syscall numbers ────────────────────────────────────────────────────── */
#if defined(__TARGET_ARCH_arm64)
#define SYS_READ     63
#define SYS_WRITE    64
#define SYS_OPENAT   56
#define SYS_CONNECT  203
#define SYS_ACCEPT   202
#define SYS_ACCEPT4  242
#define SYS_SENDTO   206
#define SYS_RECVFROM 207
#define SYS_SENDMSG  211
#define SYS_RECVMSG  212
#define SYS_WRITEV   66
#define SYS_READV    65
#define SYS_PREAD64  67
#define SYS_PWRITE64 68
#else
/* x86_64 */
#define SYS_READ     0
#define SYS_WRITE    1
#define SYS_OPENAT   257
#define SYS_CONNECT  42
#define SYS_ACCEPT   43
#define SYS_ACCEPT4  288
#define SYS_SENDTO   44
#define SYS_RECVFROM 45
#define SYS_SENDMSG  46
#define SYS_RECVMSG  47
#define SYS_WRITEV   20
#define SYS_READV    19
#define SYS_PREAD64  17
#define SYS_PWRITE64 18
#endif

/* ─── Event types (must match userspace bpfEvent) ────────────────────────── */
#define EVT_EXEC      1
#define EVT_EXIT      2
#define EVT_OPEN      3
#define EVT_IO        4
#define EVT_CONNECT   5
#define EVT_ACCEPT    6

#define AF_INET 2
#define EINPROGRESS 115

/* ─── Shared event struct — keep in sync with Go bpfEvent ────────────────── */
struct event {
	__u64 ts_ns;
	__u32 type;
	__u32 pid;
	__u32 tgid;
	__u32 uid;
	__s32 ret;
	__u32 flags;
	__u64 extra;
	__u32 saddr;
	__u32 daddr;
	__u16 sport;
	__u16 dport;
	__u16 family;
	__u16 pad;
	char  comm[16];
	char  path[128];
};

struct config_t {
	__u32 target_tgid;     /* 0 = unset */
	__u32 sample_rate;     /* reserved */
	__u32 observe_all;     /* 1 = match every process (dangerous) */
	char  target_comm[16]; /* prefix; empty = unset */
};

struct proc_stats {
	__u64 syscall_count;
	__u64 err_count;
	__u64 read_bytes;
	__u64 write_bytes;
	__u64 net_tx_bytes;
	__u64 net_rx_bytes;
	__u64 open_count;
	__u64 connect_count;
};

struct open_state {
	__u32 flags;
	char  path[128];
};

struct connect_state {
	__u32 daddr;
	__u16 dport;
	__u16 family;
	__u32 fd;
};

struct sockaddr_in_min {
	__u16 sin_family;
	__u16 sin_port;
	__u32 sin_addr;
};

/* Tracepoint common header (8 bytes) + payload. */
struct sys_enter_ctx {
	__u16 common_type;
	__u8  common_flags;
	__u8  common_preempt_count;
	__s32 common_pid;
	__s64 id;
	__u64 args[6];
};

struct sys_exit_ctx {
	__u16 common_type;
	__u8  common_flags;
	__u8  common_preempt_count;
	__s32 common_pid;
	__s64 id;
	__s64 ret;
};

/* sched_process_exec: __string filename is a __data_loc at offset 8. */
struct exec_ctx {
	__u16 common_type;
	__u8  common_flags;
	__u8  common_preempt_count;
	__s32 common_pid;
	__u32 data_loc_filename;
	__s32 pid;
	__s32 old_pid;
};

/* ─── Maps ───────────────────────────────────────────────────────────────── */

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, struct config_t);
} config SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 16 * 1024 * 1024);
} events SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 1024);
	__type(key, __u32);
	__type(value, struct proc_stats);
} process_stats SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 512);
	__type(key, __u32);
	__type(value, __u64);
} syscall_counts SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__uint(max_entries, 1);
	__type(key, __u32);
	__type(value, __u64);
} drop_count SEC(".maps");

/* Extra tgids (children) pinned by userspace. */
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 256);
	__type(key, __u32);
	__type(value, __u8);
} targets SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 32768);
	__type(key, __u32);
	__type(value, struct open_state);
} open_inflight SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, 32768);
	__type(key, __u32);
	__type(value, struct connect_state);
} connect_inflight SEC(".maps");

/* ─── Helpers ────────────────────────────────────────────────────────────── */

static __always_inline int comm_prefix_match(const char want[16]) {
	char comm[16] = {};
	if (want[0] == 0)
		return 0;
	bpf_get_current_comm(&comm, sizeof(comm));
#pragma unroll
	for (int i = 0; i < 16; i++) {
		char c = want[i];
		if (c == 0)
			return 1;
		if (comm[i] != c)
			return 0;
	}
	return 1;
}

static __always_inline int is_target(__u32 tgid) {
	__u32 key = 0;
	struct config_t *cfg = bpf_map_lookup_elem(&config, &key);
	if (!cfg)
		return 0;
	if (cfg->observe_all)
		return 1;
	if (cfg->target_tgid != 0 && cfg->target_tgid == tgid)
		return 1;
	if (comm_prefix_match(cfg->target_comm))
		return 1;
	__u8 *hit = bpf_map_lookup_elem(&targets, &tgid);
	return hit != NULL;
}

static __always_inline void record_drop(void) {
	__u32 key = 0;
	__u64 *cnt = bpf_map_lookup_elem(&drop_count, &key);
	if (cnt)
		(*cnt)++;
}

static __always_inline struct event *
alloc_event(__u32 type, __u32 pid, __u32 tgid) {
	struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		record_drop();
		return NULL;
	}
	e->ts_ns = bpf_ktime_get_ns();
	e->type = type;
	e->pid = pid;
	e->tgid = tgid;
	e->uid = (__u32)(bpf_get_current_uid_gid() & 0xFFFFFFFF);
	e->ret = 0;
	e->flags = 0;
	e->extra = 0;
	e->saddr = 0;
	e->daddr = 0;
	e->sport = 0;
	e->dport = 0;
	e->family = 0;
	e->pad = 0;
	e->path[0] = '\0';
	bpf_get_current_comm(&e->comm, sizeof(e->comm));
	return e;
}

static __always_inline struct proc_stats *get_stats(__u32 tgid) {
	struct proc_stats *ps = bpf_map_lookup_elem(&process_stats, &tgid);
	if (ps)
		return ps;
	struct proc_stats empty = {};
	bpf_map_update_elem(&process_stats, &tgid, &empty, BPF_NOEXIST);
	return bpf_map_lookup_elem(&process_stats, &tgid);
}

static __always_inline void bump_syscall(__u32 nr) {
	__u64 *c = bpf_map_lookup_elem(&syscall_counts, &nr);
	if (c) {
		__sync_fetch_and_add(c, 1);
		return;
	}
	__u64 one = 1;
	bpf_map_update_elem(&syscall_counts, &nr, &one, BPF_NOEXIST);
}

/* ───────────────────────────────────────────────────────────────────────────
 * TRACEPOINT: sched/sched_process_exec
 * Always emitted (unfiltered) so userspace can catch the target starting.
 * ─────────────────────────────────────────────────────────────────────────── */
SEC("tracepoint/sched/sched_process_exec")
int tp_exec(struct exec_ctx *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid = (__u32)(pid_tgid >> 32);
	__u32 pid = (__u32)pid_tgid;

	struct event *e = alloc_event(EVT_EXEC, pid, tgid);
	if (!e)
		return 0;

	/* __data_loc: low 16 bits = offset from start of ctx. */
	__u32 loc = ctx->data_loc_filename;
	__u32 off = loc & 0xFFFF;
	if (off != 0 && off < 4096) {
		char *p = (char *)ctx + off;
		bpf_probe_read_kernel_str(e->path, sizeof(e->path), p);
	}

	bpf_ringbuf_submit(e, 0);
	return 0;
}

SEC("tracepoint/sched/sched_process_exit")
int tp_exit(void *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid = (__u32)(pid_tgid >> 32);
	__u32 pid = (__u32)pid_tgid;

	if (!is_target(tgid))
		return 0;

	struct event *e = alloc_event(EVT_EXIT, pid, tgid);
	if (!e)
		return 0;
	bpf_ringbuf_submit(e, 0);

	bpf_map_delete_elem(&process_stats, &tgid);
	bpf_map_delete_elem(&targets, &tgid);
	return 0;
}

/* ───────────────────────────────────────────────────────────────────────────
 * TRACEPOINT: raw_syscalls/sys_enter
 * Cheap counters + capture of openat/connect/accept arguments.
 * ─────────────────────────────────────────────────────────────────────────── */
SEC("tracepoint/raw_syscalls/sys_enter")
int tp_sys_enter(struct sys_enter_ctx *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid = (__u32)(pid_tgid >> 32);
	__u32 tid = (__u32)pid_tgid;

	if (!is_target(tgid))
		return 0;

	__u32 nr = (__u32)ctx->id;

	struct proc_stats *ps = get_stats(tgid);
	if (ps)
		__sync_fetch_and_add(&ps->syscall_count, 1);
	bump_syscall(nr);

	if (nr == SYS_OPENAT) {
		struct open_state st = {};
		st.flags = (__u32)ctx->args[2];
		const char *fn = (const char *)ctx->args[1];
		bpf_probe_read_user_str(st.path, sizeof(st.path), fn);
		bpf_map_update_elem(&open_inflight, &tid, &st, BPF_ANY);
		if (ps)
			__sync_fetch_and_add(&ps->open_count, 1);

		struct event *e = alloc_event(EVT_OPEN, tid, tgid);
		if (e) {
			e->flags = st.flags;
			__builtin_memcpy(e->path, st.path, sizeof(e->path));
			bpf_ringbuf_submit(e, 0);
		}
		return 0;
	}

	if (nr == SYS_CONNECT) {
		struct sockaddr_in_min sin = {};
		if (bpf_probe_read_user(&sin, sizeof(sin), (void *)ctx->args[1]) != 0)
			return 0;
		if (sin.sin_family != AF_INET)
			return 0;

		struct connect_state st = {};
		st.family = sin.sin_family;
		st.daddr = sin.sin_addr;
		st.dport = ((sin.sin_port >> 8) & 0xff) | ((sin.sin_port & 0xff) << 8);
		st.fd = (__u32)ctx->args[0];
		bpf_map_update_elem(&connect_inflight, &tid, &st, BPF_ANY);
		if (ps)
			__sync_fetch_and_add(&ps->connect_count, 1);
		return 0;
	}

	if (nr == SYS_ACCEPT || nr == SYS_ACCEPT4) {
		struct event *e = alloc_event(EVT_ACCEPT, tid, tgid);
		if (e) {
			e->extra = ctx->args[0];
			bpf_ringbuf_submit(e, 0);
		}
	}

	return 0;
}

SEC("tracepoint/raw_syscalls/sys_exit")
int tp_sys_exit(struct sys_exit_ctx *ctx) {
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	__u32 tgid = (__u32)(pid_tgid >> 32);
	__u32 tid = (__u32)pid_tgid;

	if (!is_target(tgid))
		return 0;

	__u32 nr = (__u32)ctx->id;
	__s64 ret = ctx->ret;

	struct proc_stats *ps = bpf_map_lookup_elem(&process_stats, &tgid);

	if (ret < 0 && ps)
		__sync_fetch_and_add(&ps->err_count, 1);

	if (ps && ret > 0) {
		if (nr == SYS_READ || nr == SYS_PREAD64 || nr == SYS_READV)
			__sync_fetch_and_add(&ps->read_bytes, (__u64)ret);
		if (nr == SYS_WRITE || nr == SYS_PWRITE64 || nr == SYS_WRITEV)
			__sync_fetch_and_add(&ps->write_bytes, (__u64)ret);
		if (nr == SYS_SENDTO || nr == SYS_SENDMSG)
			__sync_fetch_and_add(&ps->net_tx_bytes, (__u64)ret);
		if (nr == SYS_RECVFROM || nr == SYS_RECVMSG)
			__sync_fetch_and_add(&ps->net_rx_bytes, (__u64)ret);
	}

	if (nr == SYS_OPENAT) {
		struct open_state *st = bpf_map_lookup_elem(&open_inflight, &tid);
		if (ret < 0 && st) {
			struct event *e = alloc_event(EVT_OPEN, tid, tgid);
			if (e) {
				e->ret = (__s32)ret;
				e->flags = st->flags | 0x80000000;
				__builtin_memcpy(e->path, st->path, sizeof(e->path));
				bpf_ringbuf_submit(e, 0);
			}
		}
		bpf_map_delete_elem(&open_inflight, &tid);
		return 0;
	}

	if (nr == SYS_CONNECT) {
		struct connect_state *st = bpf_map_lookup_elem(&connect_inflight, &tid);
		if (st) {
			/* Skip EINPROGRESS — the handshake is still in flight. */
			if (ret != -EINPROGRESS) {
				struct event *e = alloc_event(EVT_CONNECT, tid, tgid);
				if (e) {
					e->ret = (__s32)ret;
					e->daddr = st->daddr;
					e->dport = st->dport;
					e->family = st->family;
					e->extra = st->fd;
					if (ret < 0)
						e->flags = 0x80000000;
					bpf_ringbuf_submit(e, 0);
				}
			}
		}
		bpf_map_delete_elem(&connect_inflight, &tid);
	}

	return 0;
}
