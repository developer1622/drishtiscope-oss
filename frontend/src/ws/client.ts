import { WsMessage, ConnectionStatus } from '../types/protocol';

type MessageHandler = (msg: WsMessage) => void;
type StatusHandler = (status: ConnectionStatus) => void;

class WebSocketClient {
  private url: string;
  private onMessage: MessageHandler;
  private onStatusChange: StatusHandler;
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectDelay = 5000;
  private intentionalDisconnect = false;

  constructor(url: string, onMessage: MessageHandler, onStatusChange: StatusHandler) {
    this.url = url;
    this.onMessage = onMessage;
    this.onStatusChange = onStatusChange;
  }

  connect() {
    this.intentionalDisconnect = false;
    this.onStatusChange(this.reconnectAttempts === 0 ? 'connecting' : 'reconnecting');
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      this.onStatusChange('connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const msg: WsMessage = JSON.parse(event.data);
        this.onMessage(msg);
      } catch (err) {
        console.error('Failed to parse WS message', err);
      }
    };

    this.ws.onclose = () => {
      if (this.intentionalDisconnect) {
        this.onStatusChange('offline');
        return;
      }
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  disconnect() {
    this.intentionalDisconnect = true;
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private scheduleReconnect() {
    let delay = 500;
    if (this.reconnectAttempts === 1) delay = 1000;
    else if (this.reconnectAttempts === 2) delay = 2000;
    else if (this.reconnectAttempts >= 3) delay = 5000;
    
    this.reconnectAttempts++;
    setTimeout(() => this.connect(), delay);
  }
}

export function createWSClient(url: string, onMessage: MessageHandler, onStatusChange: StatusHandler) {
  return new WebSocketClient(url, onMessage, onStatusChange);
}
