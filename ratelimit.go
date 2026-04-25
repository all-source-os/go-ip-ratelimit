// Package ratelimit provides a simple in-memory IP-based rate limiter
// using a sliding window algorithm. Thread-safe, zero dependencies beyond
// the standard library.
//
// Extracted from the AllSource Control Plane (https://github.com/all-source-os/all-source).
//
// Usage:
//
//	limiter := ratelimit.New(5, time.Hour)  // 5 requests per IP per hour
//	defer limiter.Stop()
//
//	// In your HTTP handler:
//	ip := r.RemoteAddr
//	if !limiter.Allow(ip) {
//	    http.Error(w, "rate limited", http.StatusTooManyRequests)
//	    return
//	}
package ratelimit

import (
	"sync"
	"time"
)

// Limiter enforces a per-key sliding window rate limit. Keys are typically
// IP addresses but can be any string (user ID, API key, etc.).
//
// Thread-safe via sync.RWMutex. Stale entries are cleaned up periodically
// by a background goroutine.
type Limiter struct {
	mu       sync.RWMutex
	windows  map[string][]time.Time
	limit    int
	window   time.Duration
	stopChan chan struct{}
}

// New creates a rate limiter that allows `limit` requests per `window`
// duration per key. Starts a background goroutine that cleans up stale
// entries every 10 minutes. Call Stop() when done to release resources.
func New(limit int, window time.Duration) *Limiter {
	rl := &Limiter{
		windows:  make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		stopChan: make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

// Allow checks whether the given key is within the rate limit. If allowed,
// records the request and returns true. If rate-limited, returns false.
func (rl *Limiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	timestamps := rl.windows[key]
	valid := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	if len(valid) >= rl.limit {
		rl.windows[key] = valid
		return false
	}

	rl.windows[key] = append(valid, now)
	return true
}

// RetryAfter returns the number of seconds until the oldest request in the
// window expires. Useful for setting the Retry-After HTTP header.
// Returns 0 if the key has no requests in the window.
func (rl *Limiter) RetryAfter(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	timestamps := rl.windows[key]
	if len(timestamps) == 0 {
		return 0
	}

	oldest := timestamps[0]
	expiresAt := oldest.Add(rl.window)
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Seconds()) + 1
}

// Remaining returns how many requests the key has left in the current window.
func (rl *Limiter) Remaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	count := 0
	for _, ts := range rl.windows[key] {
		if ts.After(cutoff) {
			count++
		}
	}

	remaining := rl.limit - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset clears all recorded requests for a key.
func (rl *Limiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.windows, key)
}

// Stop halts the background cleanup goroutine.
func (rl *Limiter) Stop() {
	close(rl.stopChan)
}

func (rl *Limiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-rl.window)
			for key, timestamps := range rl.windows {
				valid := timestamps[:0]
				for _, ts := range timestamps {
					if ts.After(cutoff) {
						valid = append(valid, ts)
					}
				}
				if len(valid) == 0 {
					delete(rl.windows, key)
				} else {
					rl.windows[key] = valid
				}
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}
