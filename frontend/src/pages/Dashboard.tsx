import React, { useEffect } from 'react';
import { Header } from '../components/Header';
import { StatusBar } from '../components/StatusBar';
import { useScopeStore } from '../store/useScopeStore';
import { BasicTab } from '../components/tabs/BasicTab';
import { MediumTab } from '../components/tabs/MediumTab';
import { AdvancedTab } from '../components/tabs/AdvancedTab';
import { CompletePictureTab } from '../components/tabs/CompletePictureTab';
import { MetricHelpModal } from '../components/MetricHelpModal';
import { apiHeaders } from '../utils/api';

export function Dashboard() {
  const { snapshot, events, connection, mode, hello, selectedPid, setSelectedPid, activeTab } =
    useScopeStore();

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && document.activeElement?.tagName !== 'INPUT') {
        e.preventDefault();
        const input = document.querySelector('input[type="text"]') as HTMLInputElement;
        if (input) input.focus();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleTargetChange = async (comm: string, pid?: number) => {
    const body = pid !== undefined && pid > 0 ? { comm, pid } : { comm, pid: 0 };
    try {
      await fetch('/api/target', {
        method: 'POST',
        headers: apiHeaders({ 'Content-Type': 'application/json' }),
        body: JSON.stringify(body),
      });
    } catch (e) {
      console.error('Failed to change target:', e);
    }
  };

  if (!snapshot && (connection === 'connecting' || connection === 'reconnecting')) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-bg text-muted">
        <div className="flex flex-col items-center gap-4">
          <div className="w-8 h-8 rounded-full border-2 border-cyan border-t-transparent animate-spin" />
          <div className="text-sm font-mono tracking-wide">Connecting to DrishtiScope...</div>
        </div>
      </div>
    );
  }

  const target = snapshot?.meta?.target;
  const targetLabel = target
    ? `${target.comm || '—'}${target.pid ? ` [${target.pid}]` : ''}`
    : undefined;

  const targetProcess = (snapshot?.processes || []).find(
    (p) =>
      (target?.pid && p.pid === target.pid) ||
      (target?.comm && p.comm.toLowerCase() === target.comm.toLowerCase())
  );

  return (
    <div className="min-h-screen flex flex-col bg-bg h-screen overflow-hidden transition-colors">
      <Header
        mode={mode}
        hello={hello}
        status={connection}
        targetLabel={targetLabel}
        processes={snapshot?.processes || []}
        onTargetSelect={handleTargetChange}
      />

      {mode === 'mock' && (
        <div className="bg-amber/15 text-amber text-center text-xs py-1 px-4 font-mono uppercase tracking-wider font-semibold border-b border-amber/20 flex items-center justify-center gap-2">
          <span>⚡ MOCK ENGINE ACTIVE</span>
          <span className="opacity-40">|</span>
          <span>
            {hello?.ebpf_fail_reason
              ? `eBPF notice: ${hello.ebpf_fail_reason}`
              : `Observing ${targetLabel || 'agy'} — Select process or click row to switch focus`}
          </span>
        </div>
      )}

      {/* Responsive Main Container */}
      <main className="flex-1 overflow-y-auto p-3 sm:p-4 md:p-5 flex flex-col gap-4 min-h-0 w-full max-w-full">
        {activeTab === 'basic' && (
          <BasicTab snapshot={snapshot} targetProcess={targetProcess} />
        )}

        {activeTab === 'medium' && (
          <MediumTab
            snapshot={snapshot}
            events={events}
            selectedPid={selectedPid}
            activeTargetComm={target?.comm}
            onSelectPid={setSelectedPid}
            onTargetChange={handleTargetChange}
          />
        )}

        {activeTab === 'advanced' && (
          <AdvancedTab snapshot={snapshot} />
        )}

        {activeTab === 'complete' && (
          <CompletePictureTab snapshot={snapshot} events={events} />
        )}
      </main>

      <StatusBar meta={snapshot?.meta} schema={hello?.schema} />
      <MetricHelpModal />
    </div>
  );
}
