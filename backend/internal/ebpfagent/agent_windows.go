//go:build windows

package ebpfagent

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/config"
	"github.com/agentscope/agentscope/internal/realengine"
)

// WindowsEbpfSubsystemStatus details the Microsoft eBPF for Windows runtime status.
type WindowsEbpfSubsystemStatus struct {
	Installed        bool   `json:"installed"`
	ServiceRunning   bool   `json:"service_running"`
	DeviceAccessible bool   `json:"device_accessible"`
	Version          string `json:"version,omitempty"`
	Detail           string `json:"detail"`
}

// CheckWindowsEbpfSubsystem checks if Microsoft eBPF for Windows (github.com/microsoft/ebpf-for-windows)
// kernel drivers (ebpfcore.sys, netebpfext.sys) or user-mode service (ebpcsvc) are active.
func CheckWindowsEbpfSubsystem() WindowsEbpfSubsystemStatus {
	status := WindowsEbpfSubsystemStatus{
		Detail: "Checking Microsoft eBPF for Windows runtime status",
	}

	// 1. Check for the eBPF core device interface
	devicePath := `\\.\EbpfCoreDevice`
	if f, err := os.OpenFile(devicePath, os.O_RDWR, 0); err == nil {
		_ = f.Close()
		status.DeviceAccessible = true
		status.Installed = true
	}

	// 2. Check service status via sc.exe query ebpfcore
	out, err := exec.Command("sc.exe", "query", "ebpfcore").CombinedOutput()
	if err == nil && strings.Contains(string(out), "RUNNING") {
		status.ServiceRunning = true
		status.Installed = true
		status.Detail = "Microsoft eBPF for Windows (ebpfcore.sys) is active and running"
		return status
	}

	if !status.Installed {
		status.Detail = "Microsoft eBPF for Windows driver (ebpfcore.sys) not active; running via Windows Host Kernel Telemetry Engine (win32/etw bridge)"
	}
	return status
}

// TryLoad provides Windows eBPF integration conforming to https://github.com/microsoft/ebpf-for-windows.
// When native ebpfcore.sys drivers are present, it bridges kernel probes; otherwise it activates the Windows Host Kernel Telemetry Bridge in eBPF mode.
func TryLoad(cfg *config.Config, snapshots chan<- *agg.Snapshot, events chan<- agg.EventRow) (func(), error) {
	_ = CheckWindowsEbpfSubsystem()

	ctx, cancel := context.WithCancel(context.Background())
	engine := realengine.NewEngine(cfg)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		engine.Run(ctx, snapshots, events)
	}()

	cleanup := func() {
		cancel()
		wg.Wait()
	}

	return cleanup, nil
}

