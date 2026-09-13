import React, { useState, useRef, useEffect, useCallback } from 'react';
import { useScopeStore } from '../store/useScopeStore';
import { apiHeaders } from '../utils/api';
import {
  Bot,
  X,
  Send,
  Loader2,
  Copy,
  Check,
  ChevronDown,
  Zap,
  AlertTriangle,
  Activity,
  Cpu,
  Network,
  HardDrive,
  Shield,
  Minimize2,
  Sparkles,
} from 'lucide-react';

// ─── types ────────────────────────────────────────────────────────────────────

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  model?: string;
  source?: string;
  tookMs?: number;
  ts: number;
}

// ─── quick prompt chips ───────────────────────────────────────────────────────

const QUICK_PROMPTS = [
  { icon: <Cpu size={12} />, label: 'CPU Spike?', query: 'Why is CPU so high? Diagnose and suggest fixes.' },
  { icon: <Activity size={12} />, label: 'Memory Leak?', query: 'Is there a memory leak? Analyze RSS growth and suggest how to diagnose.' },
  { icon: <Sparkles size={12} />, label: '📈 Graph Thread vs Mem', query: 'Draw a graph of thread concurrency vs memory RSS density and explain what it reveals.' },
  { icon: <Sparkles size={12} />, label: '📉 Plot Runqueue Wait', query: 'Plot the CFS scheduler runqueue latency curve to detect whether this agent is starving for CPU.' },
  { icon: <Network size={12} />, label: 'Network Issues?', query: 'Analyze active network connections and identify any anomalies or connection leaks.' },
  { icon: <AlertTriangle size={12} />, label: 'Syscall Errors?', query: 'What syscall errors are occurring? Why and how to fix them?' },
  { icon: <HardDrive size={12} />, label: 'Disk I/O?', query: 'Analyze disk I/O patterns. Is read/write usage abnormal for an AI agent?' },
  { icon: <Shield size={12} />, label: 'Security Check?', query: 'Are there any security anomalies, privilege escalation attempts, or suspicious file accesses?' },
];

// ─── markdown-lite renderer ───────────────────────────────────────────────────

function renderMarkdown(text: string): React.ReactNode {
  const blocks = text.split('\n');
  const out: React.ReactNode[] = [];
  let codeBlock: string[] = [];
  let inCode = false;
  let codeLang = '';

  for (let i = 0; i < blocks.length; i++) {
    const line = blocks[i];

    if (line.startsWith('```')) {
      if (!inCode) {
        inCode = true;
        codeLang = line.slice(3).trim();
        codeBlock = [];
      } else {
        out.push(<CodeBlock key={i} code={codeBlock.join('\n')} lang={codeLang} />);
        inCode = false;
        codeBlock = [];
        codeLang = '';
      }
      continue;
    }

    if (inCode) {
      codeBlock.push(line);
      continue;
    }

    if (line.startsWith('### ')) {
      out.push(<h4 key={i} className="font-bold text-txt text-xs mt-2 mb-0.5 flex items-center gap-1"><span className="text-cyan">▸</span> {line.slice(4)}</h4>);
    } else if (line.startsWith('## ')) {
      out.push(<h3 key={i} className="font-bold text-txt text-sm mt-2 mb-1">{line.slice(3)}</h3>);
    } else if (line.startsWith('# ')) {
      out.push(<h2 key={i} className="font-bold text-txt text-base mt-2 mb-1">{line.slice(2)}</h2>);
    } else if (line.startsWith('- ') || line.startsWith('* ')) {
      out.push(<li key={i} className="ml-3 text-xs text-txt/90 list-none flex gap-1.5 items-start"><span className="text-cyan shrink-0 mt-0.5">•</span><span>{inlineMarkdown(line.slice(2))}</span></li>);
    } else if (line.match(/^\d+\. /)) {
      const num = line.match(/^(\d+)\. (.*)/);
      if (num) {
        out.push(<li key={i} className="ml-3 text-xs text-txt/90 list-none flex gap-1.5 items-start"><span className="text-amber shrink-0 font-bold">{num[1]}.</span><span>{inlineMarkdown(num[2])}</span></li>);
      }
    } else if (line.trim() === '') {
      out.push(<div key={i} className="h-1" />);
    } else {
      out.push(<p key={i} className="text-xs text-txt/90 leading-relaxed">{inlineMarkdown(line)}</p>);
    }
  }

  if (inCode && codeBlock.length > 0) {
    out.push(<CodeBlock key="trailing-code" code={codeBlock.join('\n')} lang={codeLang} />);
  }

  return <>{out}</>;
}

function inlineMarkdown(text: string): React.ReactNode {
  // Bold **text** and inline `code`
  const parts = text.split(/(\*\*[^*]+\*\*|`[^`]+`)/g);
  return (
    <>
      {parts.map((part, i) => {
        if (part.startsWith('**') && part.endsWith('**')) {
          return <strong key={i} className="font-semibold text-txt">{part.slice(2, -2)}</strong>;
        }
        if (part.startsWith('`') && part.endsWith('`')) {
          return <code key={i} className="bg-panel2 text-cyan px-1 py-0.5 rounded text-[11px] font-mono border border-border/60">{part.slice(1, -1)}</code>;
        }
        return <span key={i}>{part}</span>;
      })}
    </>
  );
}

function CodeBlock({ code, lang }: { code: string; lang: string }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="my-1.5 rounded-lg bg-[#0a0c12] border border-border/60 overflow-hidden">
      <div className="flex items-center justify-between px-3 py-1 bg-panel2/50 border-b border-border/40">
        <span className="text-[10px] font-mono text-muted">{lang || 'shell'}</span>
        <button
          onClick={copy}
          className="flex items-center gap-1 text-[10px] font-mono text-muted hover:text-cyan transition-colors"
        >
          {copied ? <Check size={11} className="text-green" /> : <Copy size={11} />}
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
      <pre className="p-3 text-[11px] font-mono text-cyan overflow-x-auto leading-relaxed whitespace-pre-wrap">
        {code}
      </pre>
    </div>
  );
}

// ─── main ChatDrawer component ────────────────────────────────────────────────

export function ChatDrawer() {
  const { isChatOpen, toggleChat, setChatOpen, snapshot, addScratchGraph } = useScopeStore();
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [includeSnapshot, setIncludeSnapshot] = useState(true);
  const [unreadCount, setUnreadCount] = useState(0);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Scroll to bottom on new messages
  useEffect(() => {
    if (isChatOpen) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
      setUnreadCount(0);
    } else if (messages.length > 0 && messages[messages.length - 1].role === 'assistant') {
      setUnreadCount((n) => n + 1);
    }
  }, [messages, isChatOpen]);

  // Auto-resize textarea
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.style.height = 'auto';
      inputRef.current.style.height = Math.min(inputRef.current.scrollHeight, 120) + 'px';
    }
  }, [inputValue]);

  // Focus input when drawer opens
  useEffect(() => {
    if (isChatOpen && inputRef.current) {
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [isChatOpen]);

  const sendMessage = useCallback(async (content: string) => {
    if (!content.trim() || loading) return;

    // Check if user requested a graph to be drawn
    const lower = content.toLowerCase();
    let graphCreated = false;
    let graphTitle = '';

    if (lower.includes('graph') || lower.includes('plot') || lower.includes('draw') || lower.includes('chart')) {
      if (lower.includes('thread') || lower.includes('mem') || lower.includes('rss')) {
        addScratchGraph({
          id: `sg-thread-mem-${Date.now()}`,
          title: 'Thread Concurrency vs Memory RSS Density',
          type: 'composed',
          description: 'Correlating worker thread count with physical RAM consumption to detect memory leaks per worker thread.',
          helpMetricId: 'thread_memory_density',
          yAxisLabel: 'MiB / Threads',
          series: [
            { key: 'rssMiB', name: 'RSS Physical RAM (MiB)', color: '#3ce0cf', type: 'bar' },
            { key: 'threads', name: 'Thread Count', color: '#f5b942', type: 'line' },
          ],
          dataPreset: 'thread_mem',
        });
        graphCreated = true;
        graphTitle = 'Thread Concurrency vs Memory RSS Density';
      } else if (lower.includes('runqueue') || lower.includes('scheduler') || lower.includes('wait')) {
        addScratchGraph({
          id: `sg-runqueue-${Date.now()}`,
          title: 'CFS Scheduler Runqueue & Thread Contention',
          type: 'composed',
          description: 'Correlating CPU scheduler wait line (runqueue) against active concurrency switches.',
          helpMetricId: 'scheduler_runqueue',
          yAxisLabel: 'Latency (µs) / Count',
          series: [
            { key: 'runqueue', name: 'Runqueue Wait (µs)', color: '#4285F4', type: 'area' },
            { key: 'threads', name: 'Active Threads', color: '#FBBC04', type: 'line' },
          ],
          dataPreset: 'runqueue_switches',
        });
        graphCreated = true;
        graphTitle = 'CFS Scheduler Runqueue & Thread Contention';
      } else if (lower.includes('net') || lower.includes('stream') || lower.includes('ingress') || lower.includes('egress')) {
        addScratchGraph({
          id: `sg-net-bidi-${Date.now()}`,
          title: 'Bi-Directional Network Ingress vs Egress Streaming',
          type: 'area',
          description: 'Dual stream comparing outbound prompt payload (TX) against incoming LLM token stream (RX).',
          helpMetricId: 'net_stream_bidi',
          yAxisLabel: 'Throughput (KB/s)',
          series: [
            { key: 'rxKb', name: 'Ingress RX (KB/s)', color: '#34A853', type: 'area' },
            { key: 'txKb', name: 'Egress TX (KB/s)', color: '#EA4335', type: 'area' },
          ],
          dataPreset: 'net_bidi',
        });
        graphCreated = true;
        graphTitle = 'Bi-Directional Network Ingress vs Egress Streaming';
      } else if (lower.includes('fault') || lower.includes('page')) {
        addScratchGraph({
          id: `sg-page-faults-${Date.now()}`,
          title: 'Linux Page Fault Rates (Minor vs Major)',
          type: 'area',
          description: 'Tracking memory page allocations. Minor faults resolve in RAM; major faults require disk page-in.',
          helpMetricId: 'memory_subsystem',
          yAxisLabel: 'Faults / sec',
          series: [
            { key: 'minorFaults', name: 'Minor Faults (RAM)', color: '#4285F4', type: 'area' },
            { key: 'majorFaults', name: 'Major Faults (Disk)', color: '#ff5d73', type: 'line' },
          ],
          dataPreset: 'page_faults',
        });
        graphCreated = true;
        graphTitle = 'Linux Page Fault Rates (Minor vs Major)';
      } else {
        addScratchGraph({
          id: `sg-quantile-${Date.now()}`,
          title: 'Syscall Latency Quantile Distribution (P50 – P99.9)',
          type: 'line',
          description: 'Kernel latency step function across statistical percentiles to detect severe tail-latency drag.',
          helpMetricId: 'latency_quantile_curve',
          yAxisLabel: 'Latency (µs)',
          series: [
            { key: 'latency', name: 'Latency (µs)', color: '#a855f7', type: 'line' },
          ],
          dataPreset: 'quantile_curve',
        });
        graphCreated = true;
        graphTitle = 'Syscall Latency Quantile Distribution';
      }
    }

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: content.trim(),
      ts: Date.now(),
    };

    setMessages((prev) => [...prev, userMsg]);
    setInputValue('');
    setLoading(true);

    try {
      const history = messages.slice(-10).map((m) => ({ role: m.role, content: m.content }));
      history.push({ role: 'user', content: content.trim() });

      const res = await fetch('/api/v1/chat', {
        method: 'POST',
        headers: apiHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify({
          messages: history,
          include_snapshot: includeSnapshot,
        }),
      });

      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();

      let replyContent = data.reply;
      if (graphCreated) {
        replyContent += `\n\n> 💡 **In Linux, there is always something to learn!** I have dynamically mounted the **"${graphTitle}"** visualization directly into the **Dynamic Graph Scratchpad** at the bottom of the dashboard for you to interact with!`;
      }

      const assistantMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: replyContent,
        model: data.model,
        source: data.source,
        tookMs: data.took_ms,
        ts: Date.now(),
      };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: '⚠️ **Connection Error** — Could not reach the DrishtiScope backend.\n\nMake sure the backend is running on `:8080` and try again.\n\n```bash\n# Check backend status\ncurl http://localhost:8080/api/health\n```',
          ts: Date.now(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  }, [loading, messages, includeSnapshot, addScratchGraph]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage(inputValue);
    }
  };

  const handleQuickPrompt = (query: string) => {
    sendMessage(query);
  };

  const comm = snapshot?.meta?.target?.comm || 'agy';

  return (
    <>
      {/* Floating Action Button */}
      <button
        type="button"
        onClick={toggleChat}
        title="DrishtiScope Agent Copilot"
        className={`fixed bottom-5 right-5 z-40 w-14 h-14 rounded-full shadow-2xl flex items-center justify-center transition-all duration-300 ${
          isChatOpen
            ? 'bg-rose/90 text-white scale-95 rotate-45'
            : 'bg-cyan text-[#0a0c12] hover:scale-110 hover:shadow-cyan/25'
        }`}
        style={{ boxShadow: isChatOpen ? undefined : '0 8px 32px rgba(60,224,207,0.35)' }}
      >
        {isChatOpen ? <X size={22} /> : <Bot size={22} />}
        {!isChatOpen && unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 w-5 h-5 bg-rose text-white text-[10px] font-bold rounded-full flex items-center justify-center animate-pulse">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {/* Drawer */}
      <div
        className={`fixed bottom-24 right-5 z-40 w-[380px] sm:w-[440px] max-h-[80vh] flex flex-col bg-panel border border-border rounded-2xl shadow-2xl overflow-hidden transition-all duration-300 ${
          isChatOpen
            ? 'opacity-100 scale-100 translate-y-0'
            : 'opacity-0 scale-95 translate-y-4 pointer-events-none'
        }`}
        style={{ maxHeight: 'min(600px, calc(100vh - 140px))' }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 bg-panel2 border-b border-border shrink-0">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-full bg-cyan/15 border border-cyan/30 flex items-center justify-center">
              <Bot size={16} className="text-cyan" />
            </div>
            <div>
              <div className="flex items-center gap-1.5">
                <span className="text-sm font-bold text-txt">Agent Copilot</span>
                <span className="w-1.5 h-1.5 rounded-full bg-green animate-pulse" />
              </div>
              <p className="text-[10px] text-muted font-mono">Observing `{comm}` · AI telemetry</p>
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            {/* Snapshot toggle */}
            <button
              type="button"
              onClick={() => setIncludeSnapshot((v) => !v)}
              title={includeSnapshot ? 'Live context ON — toggle to disable' : 'Live context OFF — click to enable'}
              className={`flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] font-mono border transition-all ${
                includeSnapshot
                  ? 'bg-cyan/15 text-cyan border-cyan/30'
                  : 'bg-panel text-muted border-border'
              }`}
            >
              <Zap size={10} />
              <span>Live</span>
            </button>
            <button
              type="button"
              onClick={() => setChatOpen(false)}
              className="p-1 rounded-lg text-muted hover:text-txt hover:bg-border/60 transition-colors"
              title="Minimize"
            >
              <Minimize2 size={14} />
            </button>
          </div>
        </div>

        {/* Messages area */}
        <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-3 min-h-0">
          {messages.length === 0 && (
            <div className="flex flex-col gap-3 pt-1">
              <div className="bg-panel2/50 border border-border/60 rounded-xl p-3 text-center">
                <Bot size={28} className="text-cyan mx-auto mb-1.5" />
                <p className="text-xs font-semibold text-txt">DrishtiScope Agent Copilot</p>
                <p className="text-[11px] text-muted mt-1 leading-relaxed">
                  Ask me anything about your observed process — LLM activities, CPU, memory, network, tools, or syscalls.
                </p>
              </div>
              {/* Quick prompts */}
              <div className="grid grid-cols-2 gap-1.5">
                {QUICK_PROMPTS.map((p) => (
                  <button
                    key={p.label}
                    type="button"
                    onClick={() => handleQuickPrompt(p.query)}
                    className="flex items-center gap-1.5 px-2.5 py-2 rounded-lg bg-panel2 border border-border hover:border-cyan/40 hover:bg-cyan/5 text-[11px] text-muted hover:text-txt transition-all text-left"
                  >
                    <span className="text-cyan shrink-0">{p.icon}</span>
                    <span className="font-medium">{p.label}</span>
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex gap-2 ${msg.role === 'user' ? 'flex-row-reverse' : 'flex-row'} items-start`}
            >
              {/* Avatar */}
              <div
                className={`w-7 h-7 rounded-full shrink-0 flex items-center justify-center text-[11px] font-bold ${
                  msg.role === 'user'
                    ? 'bg-cyan/15 text-cyan border border-cyan/30'
                    : 'bg-amber/15 text-amber border border-amber/30'
                }`}
              >
                {msg.role === 'user' ? 'You' : <Bot size={14} />}
              </div>

              {/* Bubble */}
              <div
                className={`max-w-[85%] rounded-xl px-3 py-2 text-xs ${
                  msg.role === 'user'
                    ? 'bg-cyan/15 border border-cyan/25 text-txt'
                    : 'bg-panel2 border border-border text-txt'
                }`}
              >
                {msg.role === 'assistant' ? renderMarkdown(msg.content) : <p className="text-xs leading-relaxed">{msg.content}</p>}
                {msg.model && (
                  <div className="mt-1.5 flex items-center gap-2 text-[10px] text-muted/60 font-mono border-t border-border/40 pt-1">
                    <span>{msg.model}</span>
                    {msg.tookMs !== undefined && <span>{msg.tookMs}ms</span>}
                    {msg.source === 'rule' && (
                      <span className="text-amber/70">rule-engine</span>
                    )}
                  </div>
                )}
              </div>
            </div>
          ))}

          {loading && (
            <div className="flex gap-2 items-center">
              <div className="w-7 h-7 rounded-full bg-amber/15 text-amber border border-amber/30 flex items-center justify-center">
                <Bot size={14} />
              </div>
              <div className="bg-panel2 border border-border rounded-xl px-3 py-2 flex items-center gap-2">
                <Loader2 size={13} className="animate-spin text-cyan" />
                <span className="text-xs text-muted font-mono">Analyzing telemetry…</span>
              </div>
            </div>
          )}

          <div ref={bottomRef} />
        </div>

        {/* Quick prompts (when chat has messages) */}
        {messages.length > 0 && !loading && (
          <div className="px-3 py-1.5 flex gap-1.5 overflow-x-auto shrink-0 border-t border-border/40">
            {QUICK_PROMPTS.slice(0, 4).map((p) => (
              <button
                key={p.label}
                type="button"
                onClick={() => handleQuickPrompt(p.query)}
                className="shrink-0 flex items-center gap-1 px-2 py-1 rounded-lg bg-panel2 border border-border hover:border-cyan/40 text-[10px] text-muted hover:text-cyan transition-all"
              >
                <span className="text-cyan">{p.icon}</span>
                <span>{p.label}</span>
              </button>
            ))}
          </div>
        )}

        {/* Input area */}
        <div className="px-3 pb-3 pt-2 border-t border-border shrink-0">
          <div className="flex items-end gap-2 bg-panel2 border border-border rounded-xl px-3 py-2 focus-within:border-cyan/50 transition-colors">
            <textarea
              ref={inputRef}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask about CPU, memory, syscalls, security…"
              rows={1}
              disabled={loading}
              className="flex-1 bg-transparent text-xs text-txt placeholder:text-muted/60 resize-none focus:outline-none font-mono leading-relaxed min-h-[20px]"
              style={{ maxHeight: '100px' }}
            />
            <button
              type="button"
              onClick={() => sendMessage(inputValue)}
              disabled={loading || !inputValue.trim()}
              className="shrink-0 w-7 h-7 rounded-lg bg-cyan text-[#0a0c12] flex items-center justify-center disabled:opacity-40 disabled:cursor-not-allowed hover:bg-cyan/80 transition-colors"
              title="Send (Enter)"
            >
              {loading ? <Loader2 size={13} className="animate-spin" /> : <Send size={13} />}
            </button>
          </div>
          <div className="flex items-center justify-between mt-1 px-1 text-[10px] font-mono text-muted/60">
            <span>Enter to send · Shift+Enter newline</span>
            <span className="text-cyan/80">💡 In Linux, there is always something to learn!</span>
          </div>
        </div>
      </div>

      {/* Backdrop for mobile */}
      {isChatOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/20 backdrop-blur-[1px] sm:hidden"
          onClick={() => setChatOpen(false)}
        />
      )}
    </>
  );
}
