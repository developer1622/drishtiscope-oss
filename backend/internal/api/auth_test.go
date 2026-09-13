package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenOK(t *testing.T) {
	if !tokenOK("", "") {
		t.Fatal("empty")
	}
	if tokenOK("a", "b") {
		t.Fatal("mismatch")
	}
	if !tokenOK("same", "same") {
		t.Fatal("match")
	}
	if tokenOK("sam", "same") {
		t.Fatal("length")
	}
}

func TestBearerToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/snapshot", nil)
	r.Header.Set("Authorization", "Bearer abc")
	if bearerToken(r) != "abc" {
		t.Fatal(bearerToken(r))
	}
	r = httptest.NewRequest(http.MethodGet, "/ws?access_token=xyz", nil)
	if bearerToken(r) != "xyz" {
		t.Fatal(bearerToken(r))
	}
}

func TestIsPublicPath(t *testing.T) {
	if !isPublicPath("/api/health") || isPublicPath("/api/snapshot") {
		t.Fatal("public")
	}
}
