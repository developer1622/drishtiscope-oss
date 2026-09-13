//go:build darwin

package ebpfagent

import (
	"errors"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

// TryLoad on macOS (Darwin). Darwin does not implement native kernel eBPF tracepoints.
// Real eBPF tracing requires Linux (kernel >= 5.8) or Windows (with github.com/microsoft/ebpf-for-windows).
func TryLoad(cfg *config.Config, snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) (func(), error) {
	return nil, errors.New("native eBPF tracing requires Linux (kernel >= 5.8) or Windows (with github.com/microsoft/ebpf-for-windows); macOS does not provide in-kernel eBPF runtime. Operating in synthetic mock engine mode")
}
