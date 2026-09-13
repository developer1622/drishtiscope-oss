import React, { useState, useEffect } from 'react';
import { METRIC_DOCS, MetricDoc } from '../data/metricDocs';
import { useScopeStore } from '../store/useScopeStore';
import {
  HelpCircle,
  X,
  Terminal,
  Copy,
  Check,
  Activity,
  ShieldCheck,
  AlertTriangle,
  Cpu,
  Flame,
  CheckCircle2,
  Database,
  ExternalLink,
  Sparkles,
} from 'lucide-react';

export function MetricHelpButton({
  metricId,
  color = 'cyan',
  size = 14,
  className = '',
  title,
}: {
  metricId: string;
  color?: 'cyan' | 'blue' | 'emerald' | 'amber' | 'rose' | 'purple';
  size?: number;
  className?: string;
  title?: string;
}) {
  const { openMetricHelp } = useScopeStore();
  const doc = METRIC_DOCS[metricId];
  const tooltipText = title || (doc ? `What is ${doc.title}? (Click to view guide)` : 'Click to learn what this metric means');

  const colorStyles = {
    cyan: 'text-cyan/80 hover:text-cyan hover:bg-cyan/15 border-cyan/30 hover:border-cyan/60',
    blue: 'text-[#4285F4]/90 hover:text-[#4285F4] hover:bg-blue-500/15 border-blue-500/30 hover:border-blue-500/60',
    emerald: 'text-[#34A853]/90 hover:text-[#34A853] hover:bg-emerald-500/15 border-emerald-500/30 hover:border-emerald-500/60',
    amber: 'text-amber/90 hover:text-amber hover:bg-amber/15 border-amber/30 hover:border-amber/60',
    rose: 'text-[#EA4335]/90 hover:text-[#EA4335] hover:bg-rose-500/15 border-rose-500/30 hover:border-rose-500/60',
    purple: 'text-purple-400/90 hover:text-purple-400 hover:bg-purple-500/15 border-purple-500/30 hover:border-purple-500/60',
  }[color];

  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation();
        openMetricHelp(metricId);
      }}
      title={tooltipText}
      aria-label={tooltipText}
      className={`inline-flex items-center justify-center rounded-full p-0.5 border transition-all duration-150 cursor-pointer shadow-sm hover:scale-110 active:scale-95 ${colorStyles} ${className}`}
    >
      <HelpCircle size={size} className="stroke-[2.2]" />
    </button>
  );
}

export function MetricHelpModal() {
  const { activeMetricHelpId, closeMetricHelp } = useScopeStore();
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        closeMetricHelp();
      }
    };
    if (activeMetricHelpId) {
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [activeMetricHelpId, closeMetricHelp]);

  if (!activeMetricHelpId) return null;

  const doc: MetricDoc | undefined = METRIC_DOCS[activeMetricHelpId];
  if (!doc) return null;

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const badgeColorClasses = {
    cyan: 'bg-cyan/15 text-cyan border-cyan/40',
    blue: 'bg-[#4285F4]/15 text-[#4285F4] border-blue-500/40',
    emerald: 'bg-[#34A853]/15 text-[#34A853] border-emerald-500/40',
    amber: 'bg-amber/15 text-amber border-amber/40',
    rose: 'bg-rose-500/15 text-rose border-rose-500/40',
    purple: 'bg-purple-500/15 text-purple-400 border-purple-500/40',
  }[doc.themeColor];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/75 backdrop-blur-sm animate-in fade-in duration-200"
      onClick={closeMetricHelp}
    >
      <div
        className="bg-panel border border-border rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] flex flex-col overflow-hidden text-txt transition-all scale-100"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between p-4 sm:p-5 border-b border-border bg-panel2/50">
          <div className="flex items-center gap-3">
            <div
              className={`w-10 h-10 rounded-xl border flex items-center justify-center shrink-0 ${badgeColorClasses}`}
            >
              <HelpCircle size={20} className="stroke-[2.5]" />
            </div>
            <div>
              <div className="flex items-center gap-2 flex-wrap">
                <h3 className="text-lg font-bold text-txt">{doc.title}</h3>
                <span
                  className={`text-[11px] px-2 py-0.5 rounded-full border font-mono font-medium ${badgeColorClasses}`}
                >
                  {doc.category}
                </span>
              </div>
              <p className="text-xs text-muted mt-0.5">{doc.shortDefinition}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={closeMetricHelp}
            className="w-8 h-8 rounded-lg flex items-center justify-center text-muted hover:text-txt hover:bg-border/60 transition-colors"
            title="Close (ESC)"
          >
            <X size={18} />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-4 sm:p-5 overflow-y-auto space-y-4 text-sm">
          {/* Plain English Explanation */}
          <div className="bg-panel2 border border-border/80 rounded-xl p-3.5 sm:p-4">
            <div className="text-xs font-mono uppercase text-muted tracking-wider flex items-center gap-1.5 mb-1.5">
              <Sparkles size={14} className="text-cyan" />
              <span>What Is This? (Simple Terms)</span>
            </div>
            <p className="text-txt/90 leading-relaxed text-sm">{doc.juniorAdminExplanation}</p>
          </div>

          {/* How to Read the Numbers: Healthy vs Warning vs Critical */}
          <div>
            <div className="text-xs font-mono uppercase text-muted tracking-wider mb-2 flex items-center justify-between">
              <span>How To Understand The Number</span>
              <span className="text-muted/70 font-mono text-[11px]">Unit: {doc.howToRead.unit}</span>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              {/* Healthy */}
              <div className="bg-[#34A853]/10 border border-[#34A853]/30 rounded-xl p-3 flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-bold text-[#34A853]">
                  <CheckCircle2 size={14} />
                  <span>NORMAL / HEALTHY</span>
                </div>
                <div className="text-xs text-txt/90 mt-1.5 font-medium">{doc.howToRead.healthy}</div>
              </div>

              {/* Warning */}
              <div className="bg-amber/10 border border-amber/30 rounded-xl p-3 flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-bold text-amber">
                  <AlertTriangle size={14} />
                  <span>WARNING / HIGH</span>
                </div>
                <div className="text-xs text-txt/90 mt-1.5 font-medium">{doc.howToRead.warning}</div>
              </div>

              {/* Critical */}
              <div className="bg-rose/10 border border-rose/30 rounded-xl p-3 flex flex-col justify-between">
                <div className="flex items-center gap-1.5 text-xs font-bold text-rose">
                  <Flame size={14} />
                  <span>CRITICAL / DANGER</span>
                </div>
                <div className="text-xs text-txt/90 mt-1.5 font-medium">{doc.howToRead.critical}</div>
              </div>
            </div>
          </div>

          {/* Why Autonomous AI Agents Care */}
          <div className="bg-blue-500/10 border border-blue-500/25 rounded-xl p-3.5 sm:p-4">
            <div className="text-xs font-mono uppercase text-[#4285F4] tracking-wider flex items-center gap-1.5 mb-1.5 font-bold">
              <Activity size={14} />
              <span>Why Autonomous AI Agents (e.g. agy, grok) Care</span>
            </div>
            <p className="text-txt/90 text-xs sm:text-sm leading-relaxed">{doc.whyAiAgentsCare}</p>
          </div>

          {/* Junior Admin Linux Command Line Cheat Sheet */}
          <div className="bg-black/40 border border-border rounded-xl p-3.5 sm:p-4">
            <div className="flex items-center justify-between text-xs font-mono text-muted mb-2">
              <span className="flex items-center gap-1.5">
                <Terminal size={14} className="text-green" />
                <span>Junior Linux Admin Terminal Command</span>
              </span>
              <button
                type="button"
                onClick={() => handleCopy(doc.linuxAdminCommand)}
                className="flex items-center gap-1 text-[11px] font-mono px-2 py-0.5 rounded bg-panel2 border border-border hover:border-cyan text-muted hover:text-cyan transition-colors"
                title="Copy shell command"
              >
                {copied ? (
                  <>
                    <Check size={12} className="text-green" />
                    <span className="text-green">Copied!</span>
                  </>
                ) : (
                  <>
                    <Copy size={12} />
                    <span>Copy Command</span>
                  </>
                )}
              </button>
            </div>
            <div className="bg-[#0b0e14] p-2.5 rounded-lg border border-border/60 font-mono text-xs text-cyan selection:bg-cyan/30 overflow-x-auto select-all">
              {doc.linuxAdminCommand}
            </div>
            <div className="text-[11px] text-muted/70 font-mono mt-2 flex items-center gap-1">
              <span className="text-muted font-bold">Kernel Source:</span>
              <span>{doc.kernelDataSource}</span>
            </div>
          </div>
        </div>

        {/* Modal Footer */}
        <div className="p-3 sm:p-4 border-t border-border bg-panel2/40 flex items-center justify-between text-xs text-muted">
          <span className="font-mono text-[11px]">DrishtiScope SRE Metric Encyclopedia</span>
          <button
            type="button"
            onClick={closeMetricHelp}
            className="px-3.5 py-1.5 rounded-lg bg-border hover:bg-border/80 text-txt font-medium transition-colors"
          >
            Got It
          </button>
        </div>
      </div>
    </div>
  );
}
