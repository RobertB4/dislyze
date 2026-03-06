package ratelimit

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAllow(t *testing.T) {
	dummyReq := httptest.NewRequest("POST", "/api/login", nil)
	dummyReq.RemoteAddr = "127.0.0.1:12345"

	t.Run("allows_requests_up_to_max", func(t *testing.T) {
		rl := NewRateLimiter("test", 1*time.Minute, 5)

		for i := 0; i < 5; i++ {
			assert.True(t, rl.Allow("user1", dummyReq), "attempt %d should be allowed", i+1)
		}
	})

	t.Run("blocks_at_exactly_max", func(t *testing.T) {
		rl := NewRateLimiter("test", 1*time.Minute, 3)

		for i := 0; i < 3; i++ {
			assert.True(t, rl.Allow("user1", dummyReq))
		}

		assert.False(t, rl.Allow("user1", dummyReq), "request beyond max should be blocked")
		assert.False(t, rl.Allow("user1", dummyReq), "subsequent requests should also be blocked")
	})

	t.Run("different_keys_are_independent", func(t *testing.T) {
		rl := NewRateLimiter("test", 1*time.Minute, 2)

		assert.True(t, rl.Allow("keyA", dummyReq))
		assert.True(t, rl.Allow("keyA", dummyReq))
		assert.False(t, rl.Allow("keyA", dummyReq), "keyA should be exhausted")

		assert.True(t, rl.Allow("keyB", dummyReq), "keyB should be independent of keyA")
		assert.True(t, rl.Allow("keyB", dummyReq))
		assert.False(t, rl.Allow("keyB", dummyReq), "keyB should now be exhausted")
	})

	t.Run("old_attempts_expire_after_window", func(t *testing.T) {
		window := 100 * time.Millisecond
		rl := NewRateLimiter("test", window, 2)

		assert.True(t, rl.Allow("user1", dummyReq))
		assert.True(t, rl.Allow("user1", dummyReq))
		assert.False(t, rl.Allow("user1", dummyReq), "should be blocked at max")

		// Wait for the window to fully pass
		time.Sleep(window + 50*time.Millisecond)

		assert.True(t, rl.Allow("user1", dummyReq), "should be allowed after window expires")
		assert.True(t, rl.Allow("user1", dummyReq), "second attempt after expiry should also be allowed")
	})

	t.Run("partial_window_expiry", func(t *testing.T) {
		window := 200 * time.Millisecond
		rl := NewRateLimiter("test", window, 3)

		// Make 2 attempts now
		assert.True(t, rl.Allow("user1", dummyReq))
		assert.True(t, rl.Allow("user1", dummyReq))

		// Wait for half the window, then make 1 more attempt
		time.Sleep(120 * time.Millisecond)
		assert.True(t, rl.Allow("user1", dummyReq), "third attempt within window")

		// Now at max — should be blocked
		assert.False(t, rl.Allow("user1", dummyReq), "fourth attempt should be blocked")

		// Wait for the first 2 attempts to expire (but the third is still within window)
		time.Sleep(100 * time.Millisecond)

		// The first 2 attempts should have expired, freeing 2 slots
		// The third attempt (made ~100ms ago) is still within the 200ms window
		assert.True(t, rl.Allow("user1", dummyReq), "should have capacity after partial expiry")
		assert.True(t, rl.Allow("user1", dummyReq), "should have second slot after partial expiry")
		assert.False(t, rl.Allow("user1", dummyReq), "should be blocked again at max")
	})

	t.Run("new_key_is_allowed", func(t *testing.T) {
		rl := NewRateLimiter("test", 1*time.Minute, 5)

		assert.True(t, rl.Allow("never-seen-before", dummyReq))
	})

	t.Run("zero_max_rejects_everything", func(t *testing.T) {
		rl := NewRateLimiter("test", 1*time.Minute, 0)

		assert.False(t, rl.Allow("user1", dummyReq), "zero max should reject immediately")
	})

	t.Run("max_of_one_allows_exactly_one", func(t *testing.T) {
		rl := NewRateLimiter("test", 1*time.Minute, 1)

		assert.True(t, rl.Allow("user1", dummyReq), "first attempt should be allowed")
		assert.False(t, rl.Allow("user1", dummyReq), "second attempt should be blocked")
	})
}
