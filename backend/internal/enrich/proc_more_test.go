package enrich

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClockTicks(t *testing.T) {
	ticks := ClockTicks()
	if ticks <= 0 {
		t.Fatalf("expected positive clock ticks, got %d", ticks)
	}
}

func TestResolveTargets(t *testing.T) {
	selfPID := os.Getpid()
	targets := ResolveTargets(selfPID, "")
	if len(targets) == 0 {
		t.Fatal("expected at least self PID in targets")
	}
	found := false
	for _, p := range targets {
		if p == selfPID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected self pid %d in targets %v", selfPID, targets)
	}

	// Non-existent PID
	empty := ResolveTargets(999999999, "")
	if len(empty) != 0 {
		t.Fatalf("expected empty for invalid pid, got %v", empty)
	}

	// Empty pid and empty comm
	empty2 := ResolveTargets(0, "")
	if len(empty2) != 0 {
		t.Fatalf("expected empty for (0, ''), got %v", empty2)
	}

	// Non-existent comm
	empty3 := ResolveTargets(0, "nonexistent-comm-xyz-987")
	if len(empty3) != 0 {
		t.Fatalf("expected empty for nonexistent comm, got %v", empty3)
	}

	// Real comm prefix (e.g. "go" or "bash" or "systemd" or self comm)
	selfInfo, err := ReadProcessInfo(selfPID)
	if err == nil && selfInfo.Comm != "" {
		byComm := ResolveTargets(0, selfInfo.Comm)
		if len(byComm) == 0 {
			t.Logf("comm %q had no matches", selfInfo.Comm)
		}
	}
}

func TestResolveTargetsCmdlineFallback(t *testing.T) {
	selfPID := os.Getpid()
	selfInfo, err := ReadProcessInfo(selfPID)
	if err != nil {
		t.Fatal(err)
	}
	if selfInfo.Cmdline == "" {
		t.Skip("empty cmdline")
	}

	// Go test binaries always get -test.* flags; comm is the binary name.
	const token = "-test."
	if !strings.Contains(selfInfo.Cmdline, token) {
		t.Skipf("cmdline %q does not contain %q", selfInfo.Cmdline, token)
	}
	if strings.HasPrefix(selfInfo.Comm, token) {
		t.Skipf("comm %q already matches token, cannot test fallback", selfInfo.Comm)
	}

	got := ResolveTargets(0, token)
	found := false
	for _, p := range got {
		if p == selfPID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected self pid %d via cmdline fallback, got %v (comm=%q cmdline=%q)", selfPID, got, selfInfo.Comm, selfInfo.Cmdline)
	}

	if got := ResolveTargets(0, "zzz-no-such-cmdline-token-xyz"); len(got) != 0 {
		t.Fatalf("expected empty for missing cmdline token, got %v", got)
	}
}

func TestExpandDescendants(t *testing.T) {
	// Root with no descendants
	res := expandDescendants([]int{999999999})
	if len(res) != 1 || res[0] != 999999999 {
		t.Fatalf("expected [999999999], got %v", res)
	}

	// Empty roots
	resEmpty := expandDescendants([]int{})
	if len(resEmpty) != 0 {
		t.Fatalf("expected empty, got %v", resEmpty)
	}
}

func TestReadProcessInfoInvalid(t *testing.T) {
	info, err := ReadProcessInfo(999999999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.PID != 999999999 {
		t.Fatalf("expected PID 999999999, got %d", info.PID)
	}
	if info.Comm != "" {
		t.Fatalf("expected empty comm for non-existent pid, got %s", info.Comm)
	}
}

func TestReadIOStats(t *testing.T) {
	// Test self pid (may succeed or fail if /proc/self/io requires privileges)
	_, _, _ = ReadIOStats(os.Getpid())

	// Test non-existent pid
	_, _, err := ReadIOStats(999999999)
	if err == nil {
		t.Fatal("expected error for non-existent pid")
	}
}

func TestReadNetConnections(t *testing.T) {
	// Non-existent pid
	flows, err := ReadNetConnections(999999999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flows) != 0 {
		t.Fatalf("expected 0 flows, got %d", len(flows))
	}

	// Self pid
	_, _ = ReadNetConnections(os.Getpid())
}

func TestSocketInodes(t *testing.T) {
	inodes := socketInodes(999999999)
	if len(inodes) != 0 {
		t.Fatalf("expected empty inodes, got %v", inodes)
	}
	_ = socketInodes(os.Getpid())
}

func TestParseSockTable(t *testing.T) {
	// Create mock tcp table file
	tmpDir := t.TempDir()
	tcpPath := filepath.Join(tmpDir, "tcp")
	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode                                                     
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 123456 1 0000000000000000 100 0 0 10 0                     
   1: 0100007F:1F90 0100007F:D431 01 00000000:00000000 00:00000000 00000000  1000        0 123457 1 0000000000000000 100 0 0 10 0                     
   2: BADFORMAT
   3: 0100007F:1F90 0100007F:D432 01 00000000:00000000 00:00000000 00000000  1000        0 999999 1 0000000000000000 100 0 0 10 0                     
`
	if err := os.WriteFile(tcpPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	inodes := map[uint64]struct{}{
		123456: {},
		123457: {},
	}

	flows := parseSockTable(tcpPath, "tcp", 1234, "test-comm", inodes)
	if len(flows) != 2 {
		t.Fatalf("expected 2 flows matching inodes, got %d", len(flows))
	}
	if flows[0].State != "LISTEN" {
		t.Fatalf("expected LISTEN, got %s", flows[0].State)
	}
	if flows[1].State != "ESTABLISHED" {
		t.Fatalf("expected ESTABLISHED, got %s", flows[1].State)
	}

	// Non-existent table file
	badFlows := parseSockTable(filepath.Join(tmpDir, "nonexistent"), "tcp", 1234, "test", inodes)
	if len(badFlows) != 0 {
		t.Fatalf("expected 0 flows, got %d", len(badFlows))
	}

	// UDP fallback state
	udpPath := filepath.Join(tmpDir, "udp")
	udpContent := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000  1000        0 123458 1 0000000000000000 100 0 0 10 0
   1: 0100007F:0035 00000000:0000 FF 00000000:00000000 00:00000000 00000000  1000        0 123459 1 0000000000000000 100 0 0 10 0
`
	if err := os.WriteFile(udpPath, []byte(udpContent), 0644); err != nil {
		t.Fatal(err)
	}
	udpInodes := map[uint64]struct{}{
		123458: {},
		123459: {},
	}
	uFlows := parseSockTable(udpPath, "udp", 1234, "test", udpInodes)
	if len(uFlows) != 2 {
		t.Fatalf("expected 2 udp flows, got %d", len(uFlows))
	}
	if uFlows[1].State != "UDP" {
		t.Fatalf("expected UDP fallback state, got %s", uFlows[1].State)
	}
}

func TestParseHexIPPortErrors(t *testing.T) {
	// Bad colon split
	if _, _, err := parseHexIPPort("1234"); err == nil {
		t.Fatal("expected error")
	}
	// Bad port hex
	if _, _, err := parseHexIPPort("0100007F:ZZZZ"); err == nil {
		t.Fatal("expected error")
	}
	// Bad IPv4 hex
	if _, _, err := parseHexIPPort("0100007Z:1F90"); err == nil {
		t.Fatal("expected error")
	}
	// Unsupported length
	if _, _, err := parseHexIPPort("0100:1F90"); err == nil {
		t.Fatal("expected error")
	}
	// Bad IPv6 hex character
	if _, _, err := parseHexIPPort("0000000000000000000000000100000Z:0050"); err == nil {
		t.Fatal("expected error")
	}
}

func TestFormatStartAndBootTime(t *testing.T) {
	s := formatStart(1000)
	// formatStart returns RFC3339 string if bootTime() != 0
	if bootTime() > 0 && s == "" {
		t.Fatal("expected non-empty formatted start time")
	}
	_ = bootTime()
}

func TestSyscallCoverage(t *testing.T) {
	// Test known numbers
	tests := []struct {
		nr   uint32
		want string
	}{
		{0, "read"},
		{1, "write"},
		{2, "open"},
		{3, "close"},
		{41, "socket"},
		{42, "connect"},
		{43, "accept"},
		{59, "execve"},
		{257, "openat"},
		{435, "clone3"},
		{9999, "sys_9999"},
	}
	for _, tt := range tests {
		got := GetSyscallName(tt.nr)
		if got != tt.want {
			t.Errorf("GetSyscallName(%d) = %s, want %s", tt.nr, got, tt.want)
		}
	}
	if len(SyscallNames) < 50 {
		t.Errorf("expected at least 50 syscalls in map, got %d", len(SyscallNames))
	}
}
