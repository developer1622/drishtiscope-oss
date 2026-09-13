//go:build !linux && !windows && !darwin

package ebpfagent

import (
	"errors"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
)

// TryLoad is the fallback stub for non-Linux, non-Windows, non-macOS platforms.
func TryLoad(cfg *config.Config, snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) (func(), error) {
	return nil, errors.New("eBPF is supported on Linux (kernel >= 5.8) and Windows (with github.com/microsoft/ebpf-for-windows)")
}
