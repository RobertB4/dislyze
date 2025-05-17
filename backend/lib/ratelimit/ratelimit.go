package ratelimit

import (
	"sync"
	"time"
)

type RateLimiter struct {
	attempts map[string][]time.Time
	mu       sync.RWMutex
	window   time.Duration
	max      int
}

func NewRateLimiter(window time.Duration, max int) *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		max:      max,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	if attempts, exists := rl.attempts[key]; exists {
		valid := attempts[:0]
		for _, t := range attempts {
			if t.After(windowStart) {
				valid = append(valid, t)
			}
		}
		rl.attempts[key] = valid
	}

	if len(rl.attempts[key]) >= rl.max {
		return false
	}

	rl.attempts[key] = append(rl.attempts[key], now)
	return true
}
