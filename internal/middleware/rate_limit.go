package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimitConfig holds rate limit configuration
type RateLimitConfig struct {
	// Max requests per window
	Max int
	// Window duration
	Window time.Duration
	// Key function to identify requesters
	KeyFunc func(c *fiber.Ctx) string
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(cfg RateLimitConfig) fiber.Handler {
	type visitor struct {
		count    int
		lastSeen time.Time
	}

	var (
		visitors = make(map[string]*visitor)
		mu       sync.Mutex
	)

	// Cleanup goroutine
	go func() {
		for {
			time.Sleep(cfg.Window)
			mu.Lock()
			for key, v := range visitors {
				if time.Since(v.lastSeen) > cfg.Window {
					delete(visitors, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *fiber.Ctx) error {
		key := cfg.KeyFunc(c)

		mu.Lock()
		v, exists := visitors[key]
		if !exists {
			visitors[key] = &visitor{count: 1, lastSeen: time.Now()}
			mu.Unlock()
			return c.Next()
		}

		// Reset if window expired
		if time.Since(v.lastSeen) > cfg.Window {
			v.count = 1
			v.lastSeen = time.Now()
			mu.Unlock()
			return c.Next()
		}

		v.count++
		if v.count > cfg.Max {
			mu.Unlock()
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests, please try again later",
			})
		}

		v.lastSeen = time.Now()
		mu.Unlock()

		return c.Next()
	}
}

// DefaultRateLimitKeyFunc returns user ID or IP as key
func DefaultRateLimitKeyFunc(c *fiber.Ctx) string {
	userID, ok := c.Locals("user_id").(string)
	if ok && userID != "" {
		return "user:" + userID
	}
	return "ip:" + c.IP()
}
