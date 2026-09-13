package enrich

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
)

// ProcessInfo is /proc-derived process state plus CPU accounting fields.
type ProcessInfo struct {
	agg.ProcessRow
	Utime uint64
	Stime uint64
}

var (
	clkOnce  sync.Once
	clkTicks int64 = 100
)

// ClockTicks returns USER_HZ (usually 100).
func ClockTicks() int64 {
	clkOnce.Do(func() {
		// Go doesn't expose sysconf easily without cgo; 100 is correct on
		// virtually every Linux distro. /proc/pid/stat is in these ticks.
		clkTicks = 100
	})
	return clkTicks
}

// ResolveTargets returns PIDs to watch: an explicit pid (plus children),
// or every process whose comm starts with prefix (plus children).
func ResolveTargets(pid int, commPrefix string) []int {
	if pid > 0 {
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err != nil {
			return nil
		}
		return expandDescendants([]int{pid})
	}
	if commPrefix == "" {
		return nil
	}
	roots := findByComm(commPrefix)
	if len(roots) == 0 {
		return nil
	}
	return expandDescendants(roots)
}

func findByComm(prefix string) []int {
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range ents {
		pid, ok := parsePIDName(e.Name())
		if !ok {
			continue
		}
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
		if err != nil {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(string(b)), prefix) {
			out = append(out, pid)
		}
	}
	return out
}

func expandDescendants(roots []int) []int {
	ppidOf := map[int]int{}
	ents, err := os.ReadDir("/proc")
	if err != nil {
		return roots
	}
	for _, e := range ents {
		pid, ok := parsePIDName(e.Name())
		if !ok {
			continue
		}
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		_, ppid, _, _, _, _, ok := parseStat(string(b))
		if !ok {
			continue
		}
		ppidOf[pid] = ppid
	}
	want := map[int]struct{}{}
	for _, r := range roots {
		want[r] = struct{}{}
	}
	changed := true
	for changed {
		changed = false
		for pid, ppid := range ppidOf {
			if _, got := want[pid]; got {
				continue
			}
			if _, ok := want[ppid]; ok {
				want[pid] = struct{}{}
				changed = true
			}
		}
	}
	out := make([]int, 0, len(want))
	for pid := range want {
		out = append(out, pid)
	}
	return out
}

func parsePIDName(name string) (int, bool) {
	if name == "" || name[0] < '0' || name[0] > '9' {
		return 0, false
	}
	n, err := strconv.Atoi(name)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func ReadProcessInfo(pid int) (*ProcessInfo, error) {
	pidStr := strconv.Itoa(pid)
	base := "/proc/" + pidStr

	pr := &ProcessInfo{}
	pr.PID = pid
	pr.TGID = pid

	if data, err := os.ReadFile(base + "/comm"); err == nil {
		pr.Comm = strings.TrimSpace(string(data))
	}

	if data, err := os.ReadFile(base + "/cmdline"); err == nil {
		pr.Cmdline = strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
	}

	if target, err := os.Readlink(base + "/exe"); err == nil {
		pr.Exe = target
	}

	if data, err := os.ReadFile(base + "/stat"); err == nil {
		state, ppid, utime, stime, threads, startTicks, ok := parseStat(string(data))
		if ok {
			pr.State = state
			pr.PPID = ppid
			pr.Utime = utime
			pr.Stime = stime
			pr.Threads = threads
			pr.StartTime = formatStart(startTicks)
		}
	}

	if data, err := os.ReadFile(base + "/status"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "Tgid:") {
				fmt.Sscanf(line, "Tgid:\t%d", &pr.TGID)
			} else if strings.HasPrefix(line, "Uid:") {
				fmt.Sscanf(line, "Uid:\t%d", &pr.UID)
			} else if strings.HasPrefix(line, "voluntary_ctxt_switches:") {
				fmt.Sscanf(line, "voluntary_ctxt_switches:\t%d", &pr.CtxSw)
			}
		}
	}

	if fdDir, err := os.ReadDir(base + "/fd"); err == nil {
		pr.OpenFDs = len(fdDir)
	}

	if data, err := os.ReadFile(base + "/statm"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) >= 2 {
			if vms, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
				pr.VMSBytes = vms * 4096
			}
			if rss, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				pr.RSSBytes = rss * 4096
			}
		}
	}

	return pr, nil
}

// parseStat extracts fields from /proc/pid/stat after the comm in parentheses.
func parseStat(data string) (state string, ppid int, utime, stime uint64, threads int, startTicks uint64, ok bool) {
	rparen := strings.LastIndex(data, ")")
	if rparen < 0 || rparen+2 >= len(data) {
		return
	}
	fields := strings.Fields(data[rparen+2:])
	// 0 state, 1 ppid, 11 utime, 12 stime, 17 num_threads, 19 starttime
	if len(fields) < 20 {
		return
	}
	state = fields[0]
	ppid, _ = strconv.Atoi(fields[1])
	utime, _ = strconv.ParseUint(fields[11], 10, 64)
	stime, _ = strconv.ParseUint(fields[12], 10, 64)
	threads, _ = strconv.Atoi(fields[17])
	startTicks, _ = strconv.ParseUint(fields[19], 10, 64)
	ok = true
	return
}

func formatStart(startTicks uint64) string {
	btime := bootTime()
	if btime == 0 {
		return ""
	}
	sec := btime + int64(startTicks)/ClockTicks()
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}

var (
	bootOnce sync.Once
	btime    int64
)

func bootTime() int64 {
	bootOnce.Do(func() {
		f, err := os.Open("/proc/stat")
		if err != nil {
			return
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "btime ") {
				fmt.Sscanf(sc.Text(), "btime %d", &btime)
				return
			}
		}
	})
	return btime
}

func ReadIOStats(pid int) (int64, int64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/io", pid))
	if err != nil {
		return 0, 0, err
	}
	var readBytes, writeBytes int64
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "read_bytes:") {
			fmt.Sscanf(line, "read_bytes: %d", &readBytes)
		} else if strings.HasPrefix(line, "write_bytes:") {
			fmt.Sscanf(line, "write_bytes: %d", &writeBytes)
		}
	}
	return readBytes, writeBytes, nil
}

var tcpStates = map[string]string{
	"01": "ESTABLISHED",
	"02": "SYN_SENT",
	"03": "SYN_RECV",
	"04": "FIN_WAIT1",
	"05": "FIN_WAIT2",
	"06": "TIME_WAIT",
	"07": "CLOSE",
	"08": "CLOSE_WAIT",
	"09": "LAST_ACK",
	"0A": "LISTEN",
	"0B": "CLOSING",
}

// ReadNetConnections returns TCP/UDP sockets owned by pid (inode-matched).
func ReadNetConnections(pid int) ([]agg.NetFlow, error) {
	inodes := socketInodes(pid)
	if len(inodes) == 0 {
		return nil, nil
	}
	comm := ""
	if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		comm = strings.TrimSpace(string(b))
	}
	var out []agg.NetFlow
	out = append(out, parseSockTable(fmt.Sprintf("/proc/%d/net/tcp", pid), "tcp", pid, comm, inodes)...)
	out = append(out, parseSockTable(fmt.Sprintf("/proc/%d/net/tcp6", pid), "tcp", pid, comm, inodes)...)
	out = append(out, parseSockTable(fmt.Sprintf("/proc/%d/net/udp", pid), "udp", pid, comm, inodes)...)
	return out, nil
}

func socketInodes(pid int) map[uint64]struct{} {
	ents, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return nil
	}
	out := make(map[uint64]struct{})
	for _, e := range ents {
		target, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, e.Name()))
		if err != nil {
			continue
		}
		if !strings.HasPrefix(target, "socket:[") {
			continue
		}
		s := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
		n, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			out[n] = struct{}{}
		}
	}
	return out
}

func parseSockTable(path, proto string, pid int, comm string, inodes map[uint64]struct{}) []agg.NetFlow {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []agg.NetFlow
	sc := bufio.NewScanner(f)
	if sc.Scan() {
		// header
	}
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 10 {
			continue
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		if _, ok := inodes[inode]; !ok {
			continue
		}
		src, sport, err1 := parseHexIPPort(fields[1])
		dst, dport, err2 := parseHexIPPort(fields[2])
		if err1 != nil || err2 != nil {
			continue
		}
		st := tcpStates[strings.ToUpper(fields[3])]
		if proto == "udp" && st == "" {
			st = "UDP"
		}
		out = append(out, agg.NetFlow{
			Src: src, Dst: dst, Sport: sport, Dport: dport,
			Proto: proto, State: st, PID: pid, Comm: comm,
		})
	}
	return out
}

func parseHexIPPort(s string) (string, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("bad addr %q", s)
	}
	port64, err := strconv.ParseUint(parts[1], 16, 16)
	if err != nil {
		return "", 0, err
	}
	ipHex := parts[0]
	switch len(ipHex) {
	case 8:
		n, err := strconv.ParseUint(ipHex, 16, 32)
		if err != nil {
			return "", 0, err
		}
		ip := net.IPv4(byte(n), byte(n>>8), byte(n>>16), byte(n>>24))
		return ip.String(), int(port64), nil
	case 32:
		// IPv6: eight little-endian 32-bit words. Show compressed form.
		b := make([]byte, 16)
		for i := 0; i < 4; i++ {
			word, err := strconv.ParseUint(ipHex[i*8:(i+1)*8], 16, 32)
			if err != nil {
				return "", 0, err
			}
			b[i*4+0] = byte(word)
			b[i*4+1] = byte(word >> 8)
			b[i*4+2] = byte(word >> 16)
			b[i*4+3] = byte(word >> 24)
		}
		return net.IP(b).String(), int(port64), nil
	default:
		return "", 0, fmt.Errorf("unsupported ip hex len %d", len(ipHex))
	}
}
