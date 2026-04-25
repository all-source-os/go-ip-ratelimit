package ratelimit

import (
	"testing"
	"time"
)

func TestAllow_UpToLimit(t *testing.T) {
	rl := New(5, time.Hour)
	defer rl.Stop()

	for i := 0; i < 5; i++ {
		if !rl.Allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestAllow_BlocksOverLimit(t *testing.T) {
	rl := New(3, time.Hour)
	defer rl.Stop()

	for i := 0; i < 3; i++ {
		rl.Allow("1.2.3.4")
	}
	if rl.Allow("1.2.3.4") {
		t.Error("4th request should be blocked")
	}
}

func TestAllow_DifferentKeysIndependent(t *testing.T) {
	rl := New(2, time.Hour)
	defer rl.Stop()

	rl.Allow("1.1.1.1")
	rl.Allow("1.1.1.1")

	if rl.Allow("1.1.1.1") {
		t.Error("1.1.1.1 should be blocked")
	}
	if !rl.Allow("2.2.2.2") {
		t.Error("2.2.2.2 should be allowed")
	}
}

func TestAllow_WindowExpiry(t *testing.T) {
	rl := New(2, 50*time.Millisecond)
	defer rl.Stop()

	rl.Allow("x")
	rl.Allow("x")
	if rl.Allow("x") {
		t.Error("should be blocked before expiry")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("x") {
		t.Error("should be allowed after expiry")
	}
}

func TestRetryAfter(t *testing.T) {
	rl := New(1, time.Hour)
	defer rl.Stop()

	rl.Allow("x")
	ra := rl.RetryAfter("x")
	if ra < 3500 || ra > 3601 {
		t.Errorf("RetryAfter = %d, want ~3600", ra)
	}
}

func TestRemaining(t *testing.T) {
	rl := New(5, time.Hour)
	defer rl.Stop()

	if r := rl.Remaining("x"); r != 5 {
		t.Errorf("Remaining = %d, want 5", r)
	}

	rl.Allow("x")
	rl.Allow("x")

	if r := rl.Remaining("x"); r != 3 {
		t.Errorf("Remaining = %d, want 3", r)
	}
}

func TestReset(t *testing.T) {
	rl := New(2, time.Hour)
	defer rl.Stop()

	rl.Allow("x")
	rl.Allow("x")
	if rl.Allow("x") {
		t.Error("should be blocked")
	}

	rl.Reset("x")

	if !rl.Allow("x") {
		t.Error("should be allowed after reset")
	}
}
