package hub

import (
	"context"
	"log"
	"sync"
)

// Hub fans a single producer out to many WebSocket clients.
// All mutations happen on the Run goroutine except Broadcast, which is non-blocking.
type Hub struct {
	clients    map[*Client]struct{}
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	max        int

	mu    sync.RWMutex
	count int
	done  chan struct{}
}

func NewHub(maxClients int) *Hub {
	if maxClients <= 0 {
		maxClients = 128
	}
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		max:        maxClients,
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	defer h.closeAll()
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			if len(h.clients) >= h.max {
				close(client.send)
				continue
			}
			h.clients[client] = struct{}{}
			h.setCount(len(h.clients))
		case client := <-h.unregister:
			h.drop(client)
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					log.Printf("slow websocket client dropped")
					h.drop(client)
				}
			}
		}
	}
}

func (h *Hub) drop(client *Client) {
	if _, ok := h.clients[client]; !ok {
		return
	}
	delete(h.clients, client)
	close(client.send)
	h.setCount(len(h.clients))
}

func (h *Hub) closeAll() {
	for c := range h.clients {
		close(c.send)
		delete(h.clients, c)
	}
	h.setCount(0)
	select {
	case <-h.done:
	default:
		close(h.done)
	}
}

func (h *Hub) setCount(n int) {
	h.mu.Lock()
	h.count = n
	h.mu.Unlock()
}

func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		log.Printf("hub broadcast buffer full, dropping frame")
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

func (h *Hub) MaxClients() int { return h.max }
