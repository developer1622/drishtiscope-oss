import React, { useEffect } from 'react';
import { Dashboard } from './pages/Dashboard';
import { createWSClient } from './ws/client';
import { useScopeStore } from './store/useScopeStore';
import { HelloPayload, Snapshot, EventRow } from './types/protocol';
import { apiHeaders, authedWSURL } from './utils/api';

function wsURL(): string {
  if (import.meta.env.VITE_WS_URL) {
    return import.meta.env.VITE_WS_URL as string;
  }
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/ws`;
}

function App() {
  const { setSnapshot, addEvent, setConnection, setMode, setHello, refreshRate } = useScopeStore();
  const lastUpdateRef = React.useRef<number>(0);
  const refreshRateRef = React.useRef(refreshRate);

  useEffect(() => {
    refreshRateRef.current = refreshRate;
  }, [refreshRate]);

  useEffect(() => {
    fetch('/api/snapshot', { headers: apiHeaders() })
      .then(res => (res.ok ? res.json() : null))
      .then(data => {
        if (data && data.kind === 'snapshot' && data.payload) {
          setSnapshot(data.payload as Snapshot);
          if (data.mode) setMode(data.mode);
        }
      })
      .catch(() => {
        /* backend may not be up yet; WS reconnect will fill in */
      });

    const client = createWSClient(
      authedWSURL(wsURL()),
      (msg) => {
        if (msg.mode) setMode(msg.mode);

        switch (msg.kind) {
          case 'hello':
            setHello(msg.payload as HelloPayload);
            break;
          case 'snapshot': {
            const rate = refreshRateRef.current;
            if (rate === 'manual') {
              // Ignore automated live pushes in manual mode
              break;
            }
            const minInterval = parseInt(rate, 10) || 2000;
            const now = Date.now();
            if (now - lastUpdateRef.current >= minInterval) {
              lastUpdateRef.current = now;
              setSnapshot(msg.payload as Snapshot);
            }
            break;
          }
          case 'event':
            addEvent(msg.payload as EventRow);
            break;
          case 'heartbeat':
            break;
        }
      },
      (status) => setConnection(status)
    );

    client.connect();
    return () => client.disconnect();
  }, [setSnapshot, addEvent, setConnection, setMode, setHello]);

  return <Dashboard />;
}

export default App;
