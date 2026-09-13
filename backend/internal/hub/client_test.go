package hub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func TestNewHubDefault(t *testing.T) {
	h := NewHub(0)
	if h.MaxClients() != 128 {
		t.Fatalf("expected 128 max clients, got %d", h.MaxClients())
	}
}

func TestServeWS(t *testing.T) {
	h, cancel := startHub(t, 16)
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h, w, r, AcceptConfig{InsecureSkipVerify: true}, []byte(`{"kind":"hello"}`), []byte(`{"kind":"snap"}`))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ctx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket.Dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	// Wait for greet messages
	typ, msg1, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read greet 1 failed: %v", err)
	}
	if typ != websocket.MessageText || string(msg1) != `{"kind":"hello"}` {
		t.Fatalf("unexpected greet 1: %s", string(msg1))
	}

	_, msg2, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read greet 2 failed: %v", err)
	}
	if string(msg2) != `{"kind":"snap"}` {
		t.Fatalf("unexpected greet 2: %s", string(msg2))
	}

	// Verify client is registered in hub
	time.Sleep(50 * time.Millisecond)
	if h.ClientCount() != 1 {
		t.Fatalf("expected 1 client in hub, got %d", h.ClientCount())
	}

	// Send broadcast message and verify client receives it
	h.Broadcast([]byte(`{"kind":"event","id":"123"}`))

	_, msg3, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read broadcast failed: %v", err)
	}
	if string(msg3) != `{"kind":"event","id":"123"}` {
		t.Fatalf("unexpected broadcast message: %s", string(msg3))
	}

	// Close connection from client side
	_ = conn.Close(websocket.StatusNormalClosure, "client closed")

	// Verify client is unregistered from hub
	time.Sleep(100 * time.Millisecond)
	if h.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after close, got %d", h.ClientCount())
	}
}

func TestServeWSErrorHandshake(t *testing.T) {
	h, cancel := startHub(t, 16)
	defer cancel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ws", nil) // Not a valid websocket upgrade request

	// Should not panic, should return gracefully
	ServeWS(h, rec, req, AcceptConfig{InsecureSkipVerify: true})
}
