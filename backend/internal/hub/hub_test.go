package hub

import (
	"context"
	"testing"
	"time"
)

func startHub(t *testing.T, max int) (*Hub, context.CancelFunc) {
	t.Helper()
	h := NewHub(max)
	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)
	t.Cleanup(cancel)
	return h, cancel
}

func TestHubBroadcast(t *testing.T) {
	h, _ := startHub(t, 8)
	c := &Client{send: make(chan []byte, 4)}
	h.register <- c
	time.Sleep(20 * time.Millisecond)

	h.Broadcast([]byte("hello"))
	select {
	case got := <-c.send:
		if string(got) != "hello" {
			t.Fatalf("got %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	if h.ClientCount() != 1 {
		t.Fatalf("count=%d", h.ClientCount())
	}
}

func TestHubDropsSlowClient(t *testing.T) {
	h, _ := startHub(t, 8)
	c := &Client{send: make(chan []byte, 1)}
	h.register <- c
	time.Sleep(20 * time.Millisecond)

	h.Broadcast([]byte("one"))
	h.Broadcast([]byte("two")) // buffer full → drop
	time.Sleep(20 * time.Millisecond)
	if h.ClientCount() != 0 {
		t.Fatalf("slow client should be dropped, count=%d", h.ClientCount())
	}
}

func TestHubMaxClients(t *testing.T) {
	h, _ := startHub(t, 1)
	a := &Client{send: make(chan []byte, 1)}
	b := &Client{send: make(chan []byte, 1)}
	h.register <- a
	time.Sleep(20 * time.Millisecond)
	h.register <- b
	time.Sleep(20 * time.Millisecond)
	if h.ClientCount() != 1 {
		t.Fatalf("count=%d want 1", h.ClientCount())
	}
	// overflow client channel is closed
	select {
	case _, ok := <-b.send:
		if ok {
			t.Fatal("overflow client should have closed send")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for close")
	}
}

func TestHubUnregister(t *testing.T) {
	h, _ := startHub(t, 8)
	c := &Client{send: make(chan []byte, 1)}
	h.register <- c
	time.Sleep(20 * time.Millisecond)
	h.unregister <- c
	time.Sleep(20 * time.Millisecond)
	if h.ClientCount() != 0 {
		t.Fatalf("count=%d", h.ClientCount())
	}
}

func TestHubShutdownClosesClients(t *testing.T) {
	h, cancel := startHub(t, 8)
	c := &Client{send: make(chan []byte, 1)}
	h.register <- c
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case _, ok := <-c.send:
		if ok {
			t.Fatal("expected closed send on shutdown")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestBroadcastDoesNotBlock(t *testing.T) {
	h := NewHub(1)
	// Run is not started, so broadcast buffer fills then drops.
	for i := 0; i < 300; i++ {
		h.Broadcast([]byte("x"))
	}
}
