package api

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.1.2.3:9999"
	r.Header.Set("X-Forwarded-For", "8.8.8.8")
	if got := clientIP(r, false); got != "10.1.2.3" {
		t.Fatalf("got %s", got)
	}
	if got := clientIP(r, true); got != "8.8.8.8" {
		t.Fatalf("trusted xff %s", got)
	}
}

func TestIsLoopbackIP(t *testing.T) {
	if !isLoopbackIP("127.0.0.1") || !isLoopbackIP("::1") {
		t.Fatal("loopback")
	}
	if isLoopbackIP("8.8.8.8") || isLoopbackIP("") {
		t.Fatal("non-loopback")
	}
}
