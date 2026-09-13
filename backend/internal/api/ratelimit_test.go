package api

import "testing"

func TestLimiterAllowsBurstThenBlocks(t *testing.T) {
	l := newLimiter(1, 3)
	if !l.Allow("a") || !l.Allow("a") || !l.Allow("a") {
		t.Fatal("burst")
	}
	if l.Allow("a") {
		t.Fatal("should block")
	}
	if !l.Allow("b") {
		t.Fatal("other key")
	}
}

func TestLimiterIndependentKeys(t *testing.T) {
	l := newLimiter(100, 1)
	if !l.Allow("x") {
		t.Fatal("x")
	}
	if !l.Allow("y") {
		t.Fatal("y")
	}
}
