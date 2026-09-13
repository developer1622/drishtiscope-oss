package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// ─── request / response types ────────────────────────────────────────────────

type chatMsg struct {
	Role    string `json:"role"`    // "user" | "assistant" | "system"
	Content string `json:"content"`
}

type chatRequest struct {
	Messages []chatMsg `json:"messages"`
	// optional context toggles
	IncludeSnapshot bool   `json:"include_snapshot"`
	Model           string `json:"model,omitempty"`
}

type chatResponse struct {
	Reply     string  `json:"reply"`
	Model     string  `json:"model"`
	TookMs    int64   `json:"took_ms"`
	Source    string  `json:"source"` // "llm" | "rule"
	Timestamp string  `json:"timestamp"`
}

// ─── handler ─────────────────────────────────────────────────────────────────

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json: " + err.Error()})
		return
	}
	if len(req.Messages) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "messages must not be empty"})
		return
	}

	start := time.Now()

	// Build a system context string from the latest snapshot
	sysCtx := s.buildAgentSystemContext(req.IncludeSnapshot)

	// Prepend DrishtiScope system prompt
	systemPrompt := chatMsg{
		Role: "system",
		Content: `You are DrishtiScope Agent Copilot — an expert Linux performance engineer, security analyst, and kernel observability specialist.

You are embedded inside DrishtiScope, an open-source eBPF-powered autonomous AI agent observability platform.

Your job is to:
1. Diagnose performance anomalies, memory leaks, CPU spikes, and security concerns for the observed process.
2. Suggest concrete Linux commands (ss, perf, strace, bpftool, lsof, cat /proc/<pid>/status, etc.)
3. Explain kernel metrics in plain language that any Linux engineer can understand.
4. Compare current telemetry against healthy baselines.
5. Be concise, actionable, and format commands in backticks.

You speak with authority but remain approachable. Never say "as an AI" or "I don't know" — always provide your best diagnosis.

` + sysCtx,
	}

	allMessages := make([]chatMsg, 0, len(req.Messages)+1)
	allMessages = append(allMessages, systemPrompt)
	allMessages = append(allMessages, req.Messages...)

	// Try LLM providers unless offline / rule-engine requested
	var reply, model, source string
	var err error

	if req.Model != "rule-engine" && req.Model != "offline" {
		// Gemini is tried first (highest priority)
		geminiKey := os.Getenv("GEMINI_API_KEY")
		if geminiKey == "" {
			geminiKey = ""
		}
		if geminiKey != "" {
			reply, model, err = callGemini(r.Context(), geminiKey, allMessages, req.Model)
			if err == nil {
				source = "llm"
			} else {
				log.Printf("[chat] gemini error: %v", err)
			}
		}

		if source == "" {
			if key := os.Getenv("OPENAI_API_KEY"); key != "" {
				reply, model, err = callOpenAI(r.Context(), key, allMessages, req.Model)
				if err == nil {
					source = "llm"
				}
			}
		}

		if source == "" {
			if key := os.Getenv("GROQ_API_KEY"); key != "" {
				reply, model, err = callGroq(r.Context(), key, allMessages, req.Model)
				if err == nil {
					source = "llm"
				}
			}
		}

		if source == "" {
			if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
				reply, model, err = callAnthropic(r.Context(), key, allMessages, req.Model)
				if err == nil {
					source = "llm"
				}
			}
		}
	}

	// Fallback: rule-based agent analysis
	if source == "" {
		userMsg := ""
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				userMsg = req.Messages[i].Content
				break
			}
		}
		reply = s.ruleBasedAnalysis(userMsg)
		model = "rule-engine-v1"
		source = "rule"
	}

	writeJSON(w, http.StatusOK, chatResponse{
		Reply:     reply,
		Model:     model,
		TookMs:    time.Since(start).Milliseconds(),
		Source:    source,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// ─── snapshot context builder ─────────────────────────────────────────────────

func (s *Server) buildAgentSystemContext(includeSnapshot bool) string {
	snap := s.builder.Latest()
	if snap == nil {
		return "⚠️ No live snapshot available yet. Answer general Linux/eBPF agent observability questions."
	}

	var sb strings.Builder
	sb.WriteString("## Live Telemetry Context\n")

	pid, comm := s.cfg.Target()
	sb.WriteString(fmt.Sprintf("**Target Process**: `%s` (PID %d) | **Mode**: `%s`\n", comm, pid, s.mode))
	sb.WriteString(fmt.Sprintf("**Uptime**: %.0fs | **Event Rate**: %.1f ev/s | **Dropped**: %d\n\n",
		snap.Meta.UptimeS, snap.Meta.EventRate, snap.Meta.DroppedEvents))

	if includeSnapshot && snap != nil {
		k := snap.KPIs
		sb.WriteString("### KPIs\n")
		sb.WriteString(fmt.Sprintf("- CPU: **%.1f%%** | RSS: **%.1f MiB** | FDs: **%d** | Threads: **%d**\n",
			k.CPUPct, float64(k.RSSBytes)/(1024*1024), k.OpenFDs, k.Threads))
		sb.WriteString(fmt.Sprintf("- Net TX: **%.1f KB/s** | Net RX: **%.1f KB/s**\n",
			k.NetBpsTx/1024, k.NetBpsRx/1024))
		sb.WriteString(fmt.Sprintf("- Disk R: **%.1f KB/s** | Disk W: **%.1f KB/s**\n",
			k.DiskBpsR/1024, k.DiskBpsW/1024))
		sb.WriteString(fmt.Sprintf("- Syscalls/s: **%.0f** | Err Syscalls/s: **%.0f** | Connects/s: **%.1f**\n\n",
			k.SyscallsPerSec, k.ErrSyscallsPerSec, k.ConnectsPerSec))
	}

	if len(snap.SyscallsTop) > 0 {
		sb.WriteString("### Top Syscalls\n")
		n := len(snap.SyscallsTop)
		if n > 8 {
			n = 8
		}
		for _, sc := range snap.SyscallsTop[:n] {
			sb.WriteString(fmt.Sprintf("- `%s`: %.0f/s (err: %.0f/s)\n", sc.Name, sc.CountS, sc.ErrorsS))
		}
		sb.WriteString("\n")
	}

	if len(snap.Flows) > 0 {
		sb.WriteString("### Active Network Flows\n")
		n := len(snap.Flows)
		if n > 5 {
			n = 5
		}
		for _, f := range snap.Flows[:n] {
			sb.WriteString(fmt.Sprintf("- `%s:%d → %s:%d` [%s %s]\n",
				f.Src, f.Sport, f.Dst, f.Dport, f.Proto, f.State))
		}
		sb.WriteString("\n")
	}

	if len(snap.Timeline) > 0 {
		sb.WriteString("### Recent Events (last 5)\n")
		events := snap.Timeline
		sort.Slice(events, func(i, j int) bool { return events[i].TS > events[j].TS })
		n := len(events)
		if n > 5 {
			n = 5
		}
		for _, ev := range events[:n] {
			sb.WriteString(fmt.Sprintf("- [%s] **%s** — %s\n", ev.Severity, ev.Title, ev.Detail))
		}
	}

	return sb.String()
}

// ─── rule-based fallback ──────────────────────────────────────────────────────

func (s *Server) ruleBasedAnalysis(userMsg string) string {
	snap := s.builder.Latest()
	q := strings.ToLower(userMsg)

	// High-CPU diagnosis
	if strings.Contains(q, "cpu") || strings.Contains(q, "slow") || strings.Contains(q, "performance") {
		if snap != nil && snap.KPIs.CPUPct > 70 {
			return fmt.Sprintf(
				"🔥 **High CPU Detected** — `%.1f%%` CPU usage is well above the healthy threshold (< 30%%).\n\n"+
					"**Likely causes for autonomous AI agents:**\n"+
					"1. Token generation loop spinning in tight loops\n"+
					"2. Tool call serialization in hot path (JSON encoding/decoding)\n"+
					"3. Goroutine scheduling contention (check `runtime.futex` in profiler)\n\n"+
					"**Immediate actions:**\n"+
					"```bash\n"+
					"# Profile on-CPU activity (10 seconds)\n"+
					"perf record -F 99 -p %d -g -- sleep 10 && perf report --stdio | head -50\n\n"+
					"# Check goroutine count\n"+
					"cat /proc/%d/status | grep Threads\n\n"+
					"# Watch CPU usage live\n"+
					"top -p %d -b -n 5\n"+
					"```\n\n"+
					"⚡ **Quick win**: Check if `epoll_wait` or `futex` dominate — that indicates I/O wait or lock contention, not actual compute.",
				snap.KPIs.CPUPct,
				snap.Meta.Target.PID,
				snap.Meta.Target.PID,
				snap.Meta.Target.PID,
			)
		}
		return "✅ **CPU looks healthy** — no spike detected in the current snapshot.\n\n**To investigate further:**\n```bash\nperf stat -p <PID> sleep 5\ncat /proc/<PID>/schedstat\n```"
	}

	// Memory diagnosis
	if strings.Contains(q, "memory") || strings.Contains(q, "mem") || strings.Contains(q, "rss") || strings.Contains(q, "leak") {
		if snap != nil && snap.KPIs.RSSBytes > 500*1024*1024 {
			gb := float64(snap.KPIs.RSSBytes) / (1024 * 1024 * 1024)
			return fmt.Sprintf(
				"⚠️ **High Memory Usage** — RSS is **%.2f GiB**. For an AI agent, this could indicate:\n\n"+
					"1. **Context cache accumulation** — LLM KV cache not being evicted\n"+
					"2. **Memory leak in tool result buffers** — large JSON responses retained\n"+
					"3. **mmap growth** — vector embeddings loaded into memory\n\n"+
					"**Diagnose:**\n"+
					"```bash\n"+
					"# Detailed memory map\n"+
					"cat /proc/<PID>/smaps_rollup\n\n"+
					"# Check for anon vs file-backed memory\n"+
					"cat /proc/<PID>/status | grep -E 'VmRSS|VmAnon|VmFile'\n\n"+
					"# Enable memory allocation tracking\n"+
					"valgrind --leak-check=full --log-file=leak.log <binary>\n"+
					"```", gb)
		}
		return fmt.Sprintf("✅ **Memory looks healthy** — RSS is **%.0f MiB**, within normal range for AI agents.\n\n```bash\n# Monitor memory over time\nwatch -n1 'cat /proc/<PID>/status | grep -E VmRSS|VmAnon'\n```",
			float64(func() int64 {
				if snap != nil {
					return snap.KPIs.RSSBytes
				}
				return 0
			}())/(1024*1024))
	}

	// Network diagnosis
	if strings.Contains(q, "network") || strings.Contains(q, "net") || strings.Contains(q, "socket") || strings.Contains(q, "connection") {
		if snap != nil {
			flows := len(snap.Flows)
			return fmt.Sprintf(
				"🌐 **Network Status** — **%d** active TCP flows observed.\n\n"+
					"**Healthy baseline for AI agents**: 2-6 connections (LLM API, tool endpoints, DNS).\n\n"+
					"```bash\n"+
					"# Show all connections for this process\n"+
					"ss -tanp | grep <PID>\n\n"+
					"# Check for connection leaks (TIME_WAIT accumulation)\n"+
					"ss -s\n\n"+
					"# DNS resolution latency\n"+
					"dig @1.1.1.1 api.openai.com +stats\n"+
					"```\n\n"+
					"**Tip**: If you see many `TIME_WAIT` sockets, the agent may be creating new TCP connections per request instead of reusing a connection pool.",
				flows)
		}
	}

	// File descriptors
	if strings.Contains(q, "fd") || strings.Contains(q, "file") || strings.Contains(q, "descriptor") || strings.Contains(q, "open") {
		if snap != nil && snap.KPIs.OpenFDs > 500 {
			return fmt.Sprintf(
				"🚨 **FD Leak Warning** — **%d** open file descriptors. System default limit is usually 1024.\n\n"+
					"```bash\n"+
					"# Check current FD limit\n"+
					"cat /proc/<PID>/limits | grep 'open files'\n\n"+
					"# List all open FDs\n"+
					"ls -la /proc/<PID>/fd | wc -l\n\n"+
					"# Find what types of FDs are open\n"+
					"ls -la /proc/<PID>/fd | grep -E 'socket|pipe|anon'\n\n"+
					"# Increase limit if needed (temporary)\n"+
					"ulimit -n 65535\n"+
					"```",
				snap.KPIs.OpenFDs)
		}
	}

	// Syscall errors
	if strings.Contains(q, "error") || strings.Contains(q, "syscall") || strings.Contains(q, "fail") {
		if snap != nil && snap.KPIs.ErrSyscallsPerSec > 10 {
			return fmt.Sprintf(
				"🚨 **Syscall Errors Detected** — **%.0f errors/s** is anomalous.\n\n"+
					"**Common causes:**\n"+
					"- `EACCES` / `EPERM`: Permission denied (check capabilities)\n"+
					"- `ENOENT`: Missing files (config files, temp dirs)\n"+
					"- `EAGAIN`: Resource temporarily unavailable (socket buffer full)\n"+
					"- `ECONNREFUSED`: Downstream service unavailable\n\n"+
					"```bash\n"+
					"# Trace syscall errors in real-time\n"+
					"strace -p <PID> -e trace=all -e fault=all 2>&1 | head -100\n\n"+
					"# eBPF trace for EACCES\n"+
					"bpftrace -e 'tracepoint:syscalls:sys_exit_* /args->ret == -13/ { printf(\"EACCES: %%s\\n\", probe); }'\n"+
					"```",
				snap.KPIs.ErrSyscallsPerSec)
		}
	}

	// Generic helpful response
	pid := 0
	comm := "process"
	if snap != nil {
		pid = snap.Meta.Target.PID
		comm = snap.Meta.Target.Comm
	}

	cpuPct := 0.0
	rssMiB := 0.0
	if snap != nil {
		cpuPct = snap.KPIs.CPUPct
		rssMiB = float64(snap.KPIs.RSSBytes) / (1024 * 1024)
	}

	_ = math.Pi // keep math import

	return fmt.Sprintf(
		"👁️ **DrishtiScope Agent Copilot** — Analyzing `%s` (PID %d)\n\n"+
			"**Current Vitals:**\n"+
			"- CPU: **%.1f%%** | RSS: **%.0f MiB**\n\n"+
			"**What would you like to investigate?** I can help with:\n"+
			"- 🔥 **CPU spikes** — profiling, flame graphs, hot functions\n"+
			"- 🧠 **Memory leaks** — RSS growth, mmap analysis\n"+
			"- 🌐 **Network issues** — TCP flows, connection leaks, DNS\n"+
			"- 🔒 **Security events** — EACCES, capability violations, file access\n"+
			"- 📊 **Syscall patterns** — top calls, error rates, latency\n\n"+
			"Try asking: *\"Why is CPU high?\"* or *\"Is there a memory leak?\"*",
		comm, pid, cpuPct, rssMiB)
}

// ─── LLM provider integrations ────────────────────────────────────────────────

type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []chatMsg `json:"messages"`
	MaxTokens int      `json:"max_tokens"`
}

type openAIResponse struct {
	Choices []struct {
		Message chatMsg `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func callOpenAI(ctx context.Context, apiKey string, messages []chatMsg, model string) (string, string, error) {
	if model == "" {
		model = "gpt-4o-mini"
	}
	body, _ := json.Marshal(openAIRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: 800,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var res openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}
	if res.Error != nil {
		return "", "", fmt.Errorf("openai: %s", res.Error.Message)
	}
	if len(res.Choices) == 0 {
		return "", "", fmt.Errorf("openai: empty choices")
	}
	return res.Choices[0].Message.Content, "openai/" + res.Model, nil
}

func callGroq(ctx context.Context, apiKey string, messages []chatMsg, model string) (string, string, error) {
	if model == "" {
		model = "llama-3.1-8b-instant"
	}
	body, _ := json.Marshal(openAIRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: 800,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var res openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}
	if res.Error != nil {
		return "", "", fmt.Errorf("groq: %s", res.Error.Message)
	}
	if len(res.Choices) == 0 {
		return "", "", fmt.Errorf("groq: empty choices")
	}
	return res.Choices[0].Message.Content, "groq/" + res.Model, nil
}

type anthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []chatMsg `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func callAnthropic(ctx context.Context, apiKey string, messages []chatMsg, model string) (string, string, error) {
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	// Extract system message
	systemMsg := ""
	userMessages := make([]chatMsg, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			userMessages = append(userMessages, m)
		}
	}

	body, _ := json.Marshal(anthropicRequest{
		Model:     model,
		MaxTokens: 800,
		System:    systemMsg,
		Messages:  userMessages,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var res anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}
	if res.Error != nil {
		return "", "", fmt.Errorf("anthropic: %s", res.Error.Message)
	}
	if len(res.Content) == 0 {
		return "", "", fmt.Errorf("anthropic: empty content")
	}
	return res.Content[0].Text, "anthropic/" + res.Model, nil
}

// ─── Gemini (Google AI Studio) ────────────────────────────────────────────────

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"` // "user" | "model"
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
	GenerationConfig  map[string]any  `json:"generation_config,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	ModelVersion string `json:"modelVersion"`
	Error        *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func callGemini(ctx context.Context, apiKey string, messages []chatMsg, model string) (string, string, error) {
	if model == "" {
		model = "gemini-3.6-flash"
	}

	// Extract system message and build Gemini content array
	var systemText string
	contents := make([]geminiContent, 0, len(messages))

	for _, m := range messages {
		if m.Role == "system" {
			systemText = m.Content
			continue
		}
		// Gemini uses "model" instead of "assistant"
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	payload := geminiRequest{
		Contents: contents,
		GenerationConfig: map[string]any{
			"maxOutputTokens": 2048,
			"temperature":     0.65,
		},
	}
	if systemText != "" {
		payload.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemText}},
		}
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, apiKey,
	)
	reqCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var res geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", err
	}
	if res.Error != nil {
		return "", "", fmt.Errorf("gemini: %s (%s)", res.Error.Message, res.Error.Status)
	}
	if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		return "", "", fmt.Errorf("gemini: empty candidates")
	}
	usedModel := res.ModelVersion
	if usedModel == "" {
		usedModel = model
	}
	return res.Candidates[0].Content.Parts[0].Text, "gemini/" + usedModel, nil
}
