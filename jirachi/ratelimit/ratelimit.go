package ratelimit

import (
	"dislyze/jirachi/logger"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	serviceName string // e.g. lugia, giratina
	attempts    map[string][]time.Time
	mu          sync.RWMutex
	window      time.Duration
	max         int
}

func NewRateLimiter(serviceName string, window time.Duration, max int) *RateLimiter {
	return &RateLimiter{
		serviceName: serviceName,
		attempts:    make(map[string][]time.Time),
		window:      window,
		max:         max,
	}
}

func (rl *RateLimiter) Allow(key string, r *http.Request) bool {
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
		logger.LogRateLimitViolation(logger.RateLimitEvent{
			EventType: "rate_limit_violation",
			Service:   rl.serviceName,
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Endpoint:  r.URL.Path,
			Timestamp: time.Now(),
			Limit:     fmt.Sprintf("%d attempts per %d minutes", rl.max, int(rl.window.Minutes())),
		})
		return false
	}

	rl.attempts[key] = append(rl.attempts[key], now)
	return true
}
