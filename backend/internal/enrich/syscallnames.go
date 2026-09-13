package enrich

import "fmt"

// SyscallNames is the x86_64 syscall table (subset of commonly seen calls).
// Unknown numbers render as sys_<nr>.
var SyscallNames = map[uint32]string{
	0: "read", 1: "write", 2: "open", 3: "close", 4: "stat", 5: "fstat", 6: "lstat", 7: "poll",
	8: "lseek", 9: "mmap", 10: "mprotect", 11: "munmap", 12: "brk", 13: "rt_sigaction", 14: "rt_sigprocmask", 15: "rt_sigreturn",
	16: "ioctl", 17: "pread64", 18: "pwrite64", 19: "readv", 20: "writev", 21: "access", 22: "pipe", 23: "select",
	24: "sched_yield", 25: "mremap", 26: "msync", 27: "mincore", 28: "madvise", 29: "shmget", 30: "shmat", 31: "shmctl",
	32: "dup", 33: "dup2", 34: "pause", 35: "nanosleep", 36: "getitimer", 37: "alarm", 38: "setitimer", 39: "getpid",
	40: "sendfile", 41: "socket", 42: "connect", 43: "accept", 44: "sendto", 45: "recvfrom", 46: "sendmsg", 47: "recvmsg",
	48: "shutdown", 49: "bind", 50: "listen", 51: "getsockname", 52: "getpeername", 53: "socketpair", 54: "setsockopt", 55: "getsockopt",
	56: "clone", 57: "fork", 58: "vfork", 59: "execve", 60: "exit", 61: "wait4", 62: "kill", 63: "uname",
	72: "fcntl", 73: "flock", 74: "fsync", 78: "getdents", 79: "getcwd", 80: "chdir", 82: "rename", 83: "mkdir",
	89: "readlink", 90: "chmod", 96: "gettimeofday", 97: "getrlimit", 102: "getuid", 104: "getgid",
	137: "statfs", 158: "arch_prctl", 186: "gettid", 202: "futex", 203: "sched_setaffinity", 204: "sched_getaffinity",
	217: "getdents64", 218: "set_tid_address", 228: "clock_gettime", 230: "clock_nanosleep",
	231: "exit_group", 232: "epoll_wait", 233: "epoll_ctl", 234: "tgkill", 247: "waitid",
	257: "openat", 262: "newfstatat", 263: "unlinkat", 267: "readlinkat", 269: "fchmodat", 270: "faccessat",
	271: "pselect6", 272: "ppoll", 281: "epoll_pwait", 288: "accept4", 291: "epoll_create1",
	292: "dup3", 293: "pipe2", 302: "prlimit64", 318: "getrandom", 332: "statx",
	334: "rseq", 435: "clone3",
}

func GetSyscallName(nr uint32) string {
	if name, ok := SyscallNames[nr]; ok {
		return name
	}
	return fmt.Sprintf("sys_%d", nr)
}
