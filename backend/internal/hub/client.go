package hub

import (
	"context"
	"log"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

// Client is one browser WebSocket.
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

type AcceptConfig struct {
	OriginPatterns     []string
	InsecureSkipVerify bool
}

// ServeWS upgrades the HTTP connection, registers the client, sends greet
// frames, then blocks until the socket closes.
func ServeWS(h *Hub, w http.ResponseWriter, r *http.Request, ac AcceptConfig, greetMsgs ...[]byte) {
	opts := &websocket.AcceptOptions{
		OriginPatterns:     ac.OriginPatterns,
		InsecureSkipVerify: ac.InsecureSkipVerify,
	}
	c, err := websocket.Accept(w, r, opts)
	if err != nil {
		log.Printf("WS accept error: %v", err)
		return
	}
	// Bound inbound payload size (browsers shouldn't send large frames).
	c.SetReadLimit(32 << 10)

	client := &Client{
		conn: c,
		send: make(chan []byte, 64),
	}

	select {
	case h.register <- client:
	case <-time.After(2 * time.Second):
		_ = c.Close(websocket.StatusTryAgainLater, "hub unavailable")
		return
	}

	go client.writePump()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	for _, msg := range greetMsgs {
		if msg != nil {
			_ = c.Write(ctx, websocket.MessageText, msg)
		}
	}
	cancel()

	readCtx := r.Context()
	for {
		_, _, err := c.Read(readCtx)
		if err != nil {
			break
		}
	}

	select {
	case h.unregister <- client:
	case <-time.After(2 * time.Second):
	}
}

func (c *Client) writePump() {
	defer func() {
		if c.conn != nil {
			_ = c.conn.Close(websocket.StatusNormalClosure, "bye")
		}
	}()
	for msg := range c.send {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		err := c.conn.Write(ctx, websocket.MessageText, msg)
		cancel()
		if err != nil {
			log.Printf("WS write error: %v", err)
			return
		}
	}
}
