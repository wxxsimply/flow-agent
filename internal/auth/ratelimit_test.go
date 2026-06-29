package auth

import (
	"testing"
	"time"
)

func TestRateLimiter_blocksAfterMax(t *testing.T) {
	l := NewRateLimiter(2, time.Minute)
	if !l.Allow("ip-1") || !l.Allow("ip-1") {
		t.Fatal("first two should pass")
	}
	if l.Allow("ip-1") {
		t.Fatal("third should be blocked")
	}
}

func TestRateLimiter_separateKeys(t *testing.T) {
	l := NewRateLimiter(1, time.Minute)
	if !l.Allow("a") {
		t.Fatal("a should pass")
	}
	if !l.Allow("b") {
		t.Fatal("b should pass")
	}
}
