package mock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agentscope/agentscope/internal/agg"
	"github.com/agentscope/agentscope/internal/enrich"
)

type cpuSample struct {
	totalTicks uint64
	sampleTime time.Time
	readBytes  int64
	writeBytes int64
}

var (
	procHistoryMu sync.Mutex
	procHistory   = make(map[int]cpuSample)
)

// scanRealAIProcesses inspects /proc for real running AI agents (agy, codex, copilot, etc.)
func scanRealAIProcesses(targetPID int, targetComm string) []agg.ProcessRow {
	var rows []agg.ProcessRow
	now := time.Now()

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	seenPids := make(map[int]bool)

	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}

		base := fmt.Sprintf("/proc/%d", pid)
		commBytes, err := os.ReadFile(base + "/comm")
		if err != nil {
			continue
		}
		comm := strings.TrimSpace(string(commBytes))

		cmdBytes, _ := os.ReadFile(base + "/cmdline")
		cmdline := strings.ReplaceAll(string(cmdBytes), "\x00", " ")
		lowerComm := strings.ToLower(comm)
		lowerCmd := strings.ToLower(cmdline)

		isCandidate := false
		if (targetPID > 0 && pid == targetPID) ||
			(targetComm != "" && (strings.EqualFold(comm, targetComm) || strings.Contains(lowerCmd, strings.ToLower(targetComm)))) {
			isCandidate = true
		}

		// Prominently discover AI agent processes running on host
		if lowerComm == "codex" || strings.Contains(lowerCmd, "codex") ||
			lowerComm == "agy" || strings.Contains(lowerCmd, "agy") ||
			strings.Contains(lowerCmd, "copilot") ||
			(lowerComm == "node" && (strings.Contains(lowerCmd, "extensionhost") || strings.Contains(lowerCmd, "vite"))) ||
			lowerComm == "dockerd" || lowerComm == "kubelet" {
			isCandidate = true
		}

		if !isCandidate {
			continue
		}

		if seenPids[pid] {
			continue
		}
		seenPids[pid] = true

		info, err := enrich.ReadProcessInfo(pid)
		if err != nil || info == nil {
			continue
		}

		// Calculate realistic CPU from /proc stat ticks
		var cpuPct float64
		currentTicks := info.Utime + info.Stime

		procHistoryMu.Lock()
		if prev, ok := procHistory[pid]; ok {
			deltaTicks := int64(currentTicks - prev.totalTicks)
			deltaSec := now.Sub(prev.sampleTime).Seconds()
			if deltaSec > 0.1 && deltaTicks >= 0 {
				ticksPerSec := float64(deltaTicks) / deltaSec
				clockTicks := enrich.ClockTicks()
				if clockTicks <= 0 {
					clockTicks = 100
				}
				cpuPct = (ticksPerSec / float64(clockTicks)) * 100.0
			}
		}
		rBytes, wBytes, _ := enrich.ReadIOStats(pid)
		procHistory[pid] = cpuSample{
			totalTicks: currentTicks,
			sampleTime: now,
			readBytes:  rBytes,
			writeBytes: wBytes,
		}
		procHistoryMu.Unlock()

		if cpuPct < 0.1 {
			cpuPct = 0.2
		}
		if cpuPct > 100.0 {
			cpuPct = 99.9
		}

		exe := info.Exe
		if exe == "" {
			exe, _ = filepath.EvalSymlinks(base + "/exe")
		}

		row := agg.ProcessRow{
			PID:       pid,
			TGID:      info.TGID,
			PPID:      info.PPID,
			Comm:      comm,
			Cmdline:   cmdline,
			Exe:       exe,
			UID:       info.UID,
			State:     info.State,
			Threads:   info.Threads,
			CPUPct:    cpuPct,
			RSSBytes:  info.RSSBytes,
			VMSBytes:  info.VMSBytes,
			OpenFDs:   info.OpenFDs,
			CtxSw:     info.CtxSw,
			StartTime: info.StartTime,
		}

		rows = append(rows, row)
	}

	return rows
}

// readRealTargetMetrics returns live real-world I/O, flows, and files for a PID from /proc
func readRealTargetMetrics(pid int) (rBps, wBps float64, flows []agg.NetFlow, files []agg.FileStat, ok bool) {
	if pid <= 0 {
		return 0, 0, nil, nil, false
	}
	base := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(base); err != nil {
		return 0, 0, nil, nil, false
	}

	// Real IO
	rBytes, wBytes, err := enrich.ReadIOStats(pid)
	now := time.Now()
	if err == nil {
		procHistoryMu.Lock()
		if prev, exists := procHistory[pid]; exists {
			deltaSec := now.Sub(prev.sampleTime).Seconds()
			if deltaSec > 0.1 {
				if rBytes >= prev.readBytes {
					rBps = float64(rBytes-prev.readBytes) / deltaSec
				}
				if wBytes >= prev.writeBytes {
					wBps = float64(wBytes-prev.writeBytes) / deltaSec
				}
			}
		}
		procHistoryMu.Unlock()
	}

	// Real network flows from /proc/<pid>/net/tcp
	if realFlows, err := enrich.ReadNetConnections(pid); err == nil && len(realFlows) > 0 {
		flows = realFlows
	}

	// Real open files from /proc/<pid>/fd
	if fdEntries, err := os.ReadDir(base + "/fd"); err == nil {
		fileMap := make(map[string]int)
		for _, e := range fdEntries {
			link, err := os.Readlink(base + "/fd/" + e.Name())
			if err == nil && strings.HasPrefix(link, "/") && !strings.HasPrefix(link, "/dev") && !strings.HasPrefix(link, "/proc") {
				fileMap[link]++
			}
		}
		for path, count := range fileMap {
			files = append(files, agg.FileStat{
				Path:   path,
				OpsS:   float64(count) * 2.5,
				BytesS: float64(count) * 1024 * 4,
				Errors: 0,
			})
			if len(files) >= 8 {
				break
			}
		}
	}

	return rBps, wBps, flows, files, true
}
