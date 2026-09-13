package enrich

import (
	"os"
	"testing"
)

func TestParseStat(t *testing.T) {
	line := "1234 (payments agent) S 1 1234 1234 0 -1 4194304 0 0 0 0 10 20 0 0 20 0 8 0 111 123456 77 18446744073709551615"
	state, ppid, utime, stime, threads, startTicks, ok := parseStat(line)
	if !ok {
		t.Fatal("parseStat failed")
	}
	if state != "S" || ppid != 1 {
		t.Fatalf("state=%s ppid=%d", state, ppid)
	}
	if utime != 10 || stime != 20 {
		t.Fatalf("utime=%d stime=%d", utime, stime)
	}
	if threads != 8 {
		t.Fatalf("threads=%d", threads)
	}
	if startTicks != 111 {
		t.Fatalf("startTicks=%d", startTicks)
	}
}

func TestParseStatRejectsShort(t *testing.T) {
	if _, _, _, _, _, _, ok := parseStat("12 (x) S"); ok {
		t.Fatal("expected failure")
	}
	if _, _, _, _, _, _, ok := parseStat("no-parens"); ok {
		t.Fatal("expected failure")
	}
}

func TestParseHexIPPort(t *testing.T) {
	ip, port, err := parseHexIPPort("0100007F:1F90")
	if err != nil {
		t.Fatal(err)
	}
	if ip != "127.0.0.1" || port != 8080 {
		t.Fatalf("got %s:%d", ip, port)
	}
	if _, _, err := parseHexIPPort("bad"); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := parseHexIPPort("GG:80"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseHexIPPortV6(t *testing.T) {
	// ::1 port 80 — four little-endian 32-bit words of zeros except last.
	ip, port, err := parseHexIPPort("00000000000000000000000001000000:0050")
	if err != nil {
		t.Fatal(err)
	}
	if port != 80 {
		t.Fatalf("port %d", port)
	}
	if ip == "" {
		t.Fatal("empty ip")
	}
}

func TestGetSyscallName(t *testing.T) {
	if GetSyscallName(257) != "openat" {
		t.Fatal(GetSyscallName(257))
	}
	if GetSyscallName(9999) != "sys_9999" {
		t.Fatal(GetSyscallName(9999))
	}
}

func TestParsePIDName(t *testing.T) {
	if n, ok := parsePIDName("42"); !ok || n != 42 {
		t.Fatalf("42 -> %d %v", n, ok)
	}
	if _, ok := parsePIDName("net"); ok {
		t.Fatal("net should not parse")
	}
	if _, ok := parsePIDName(""); ok {
		t.Fatal("empty")
	}
	if _, ok := parsePIDName("0"); ok {
		t.Fatal("pid 0")
	}
}

func TestTCPStateMap(t *testing.T) {
	if tcpStates["01"] != "ESTABLISHED" || tcpStates["0A"] != "LISTEN" {
		t.Fatal(tcpStates)
	}
}

func TestReadProcessInfoSelf(t *testing.T) {
	info, err := ReadProcessInfo(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if info.PID != os.Getpid() {
		t.Fatalf("pid %d", info.PID)
	}
	if info.Comm == "" {
		t.Fatal("empty comm")
	}
	if info.Threads < 1 {
		t.Fatal("threads")
	}
}
