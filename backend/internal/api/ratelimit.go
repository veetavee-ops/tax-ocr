package api

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rlEntry
	limit   int
	window  time.Duration
}

type rlEntry struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		entries: make(map[string]*rlEntry),
		limit:   limit,
		window:  window,
	}
	go func() {
		for {
			time.Sleep(window)
			rl.mu.Lock()
			now := time.Now()
			for k, v := range rl.entries {
				if now.After(v.reset) {
					delete(rl.entries, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	e, ok := rl.entries[key]
	if !ok || now.After(e.reset) {
		rl.entries[key] = &rlEntry{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if e.count >= rl.limit {
		return false
	}
	e.count++
	return true
}

var loginLimiter = newRateLimiter(10, time.Minute)
