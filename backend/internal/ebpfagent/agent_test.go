//go:build linux

package ebpfagent

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

func TestCstr(t *testing.T) {
	cases := []struct {
		in   []byte
		want string
	}{
		{[]byte("hello\x00world"), "hello"},
		{[]byte("exact"), "exact"},
		{[]byte("\x00initial"), ""},
		{[]byte{}, ""},
	}
	for _, tc := range cases {
		if got := cstr(tc.in); got != tc.want {
			t.Errorf("cstr(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIntToIP(t *testing.T) {
	// Little endian representation of 127.0.0.1 is 0x0100007f
	ip := intToIP(0x0100007f)
	if ip != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %s", ip)
	}

	// 10.0.0.1 -> 0x0100000a
	ip2 := intToIP(0x0100000a)
	if ip2 != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %s", ip2)
	}
}

func TestLiveStateNoteEvent(t *testing.T) {
	s := newLiveState()

	// 1. Note evtOpen success
	var evOpen bpfEvent
	evOpen.Type = evtOpen
	copy(evOpen.Path[:], "/etc/resolv.conf\x00")
	copy(evOpen.Comm[:], "agy\x00")
	evOpen.Ret = 3
	s.noteEvent(evOpen)

	if s.eventN != 1 {
		t.Fatalf("expected eventN=1, got %d", s.eventN)
	}
	if s.files["/etc/resolv.conf"] == nil || s.files["/etc/resolv.conf"].ops != 1 {
		t.Fatal("file stat not recorded")
	}

	// 2. Note evtOpen failure (EACCES)
	var evOpenFail bpfEvent
	evOpenFail.Type = evtOpen
	copy(evOpenFail.Path[:], "/etc/shadow\x00")
	evOpenFail.Ret = -13
	s.noteEvent(evOpenFail)

	if s.files["/etc/shadow"].errors != 1 {
		t.Fatal("file error not recorded")
	}

	// 3. Note evtConnect
	var evConn bpfEvent
	evConn.Type = evtConnect
	evConn.Daddr = 0x0100000a
	evConn.Dport = 443
	evConn.Tgid = 22786
	copy(evConn.Comm[:], "agy\x00")
	s.noteEvent(evConn)

	if s.connectN != 1 {
		t.Fatalf("expected connectN=1, got %d", s.connectN)
	}
	if len(s.flows) == 0 {
		t.Fatal("expected flow recorded")
	}
}

func TestHandleEvent(t *testing.T) {
	cfg := config.Load()
	_ = cfg.SetTarget(22786, "agy")
	events := make(chan agg.EventRow, 10)

	// Case 1: evtExec
	var evExec bpfEvent
	evExec.Type = evtExec
	evExec.Tgid = 22786
	evExec.Pid = 22786
	copy(evExec.Comm[:], "agy\x00")
	copy(evExec.Path[:], "/usr/bin/agy\x00")
	handleEvent(cfg, evExec, events)

	select {
	case e := <-events:
		if e.Category != "process" || e.Title != "exec" {
			t.Fatalf("unexpected event: %+v", e)
		}
	default:
		t.Fatal("expected exec event")
	}

	// Case 2: evtExit
	var evExit bpfEvent
	evExit.Type = evtExit
	evExit.Tgid = 22786
	copy(evExit.Comm[:], "agy\x00")
	evExit.Ret = 0
	handleEvent(cfg, evExit, events)

	select {
	case e := <-events:
		if e.Category != "process" || e.Title != "exit" {
			t.Fatalf("unexpected event: %+v", e)
		}
	default:
		t.Fatal("expected exit event")
	}

	// Case 3: evtOpen permission denied
	var evOpen bpfEvent
	evOpen.Type = evtOpen
	evOpen.Tgid = 22786
	copy(evOpen.Comm[:], "agy\x00")
	copy(evOpen.Path[:], "/etc/shadow\x00")
	evOpen.Ret = -13
	handleEvent(cfg, evOpen, events)

	select {
	case e := <-events:
		if e.Category != "file" || e.Severity != "warn" {
			t.Fatalf("unexpected open event: %+v", e)
		}
	default:
		t.Fatal("expected open event")
	}

	// Case 4: evtConnect
	var evConn bpfEvent
	evConn.Type = evtConnect
	evConn.Tgid = 22786
	copy(evConn.Comm[:], "agy\x00")
	evConn.Daddr = 0x0100000a
	evConn.Dport = 443
	handleEvent(cfg, evConn, events)

	select {
	case e := <-events:
		if e.Category != "network" {
			t.Fatalf("unexpected connect event: %+v", e)
		}
	default:
		t.Fatal("expected connect event")
	}

	// Case 5: Non-matching comm ignored
	var evOther bpfEvent
	evOther.Type = evtExec
	evOther.Tgid = 9999
	copy(evOther.Comm[:], "other\x00")
	handleEvent(cfg, evOther, events)

	select {
	case e := <-events:
		t.Fatalf("expected no event for non-matching target, got %+v", e)
	default:
		// success
	}
}

func TestBpfEventBinarySize(t *testing.T) {
	var ev bpfEvent
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, &ev); err != nil {
		t.Fatalf("binary.Write bpfEvent failed: %v", err)
	}

	// Verify buffer size is consistent
	if buf.Len() < 160 {
		t.Fatalf("expected bpfEvent size >= 160, got %d", buf.Len())
	}

	var evRead bpfEvent
	if err := binary.Read(buf, binary.LittleEndian, &evRead); err != nil {
		t.Fatalf("binary.Read bpfEvent failed: %v", err)
	}
}

func TestLiveStateBuildSnapshot(t *testing.T) {
	cfg := config.Load()
	_ = cfg.SetTarget(0, "nonexistent-comm-xyz")
	s := newLiveState()
	s.start = time.Now().Add(-5 * time.Second)
	s.lastTick = time.Now().Add(-1 * time.Second)

	// Note some files and flows
	var evOpen bpfEvent
	evOpen.Type = evtOpen
	copy(evOpen.Path[:], "/var/log/test.log\x00")
	s.noteEvent(evOpen)

	snap := s.buildSnapshot(cfg, nil)
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if snap.SRE.SLOAvailability <= 0 {
		t.Errorf("expected SRE availability > 0, got %v", snap.SRE.SLOAvailability)
	}
}
