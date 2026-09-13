//go:build linux

package ebpfagent

import (
	"bytes"
	"testing"

	"github.com/cilium/ebpf"
)

func TestEmbeddedObjectParses(t *testing.T) {
	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(bpfObject))
	if err != nil {
		t.Fatalf("LoadCollectionSpecFromReader: %v", err)
	}
	for _, name := range []string{"tp_exec", "tp_exit", "tp_sys_enter", "tp_sys_exit"} {
		if spec.Programs[name] == nil {
			t.Errorf("missing program %s", name)
		}
	}
	for _, name := range []string{"events", "config", "process_stats", "syscall_counts", "targets", "drop_count"} {
		if spec.Maps[name] == nil {
			t.Errorf("missing map %s", name)
		}
	}
}
