package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/minisource/go-common/logging"
)

// LoggingMiddleware creates a logging middleware
func LoggingMiddleware(logger logging.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Log request
		duration := time.Since(start)
		status := c.Response().StatusCode()

		logger.Info(
			logging.RequestResponse,
			logging.Api,
			"HTTP Request",
			map[logging.ExtraKey]interface{}{
				"status":      status,
				"method":      c.Method(),
				"path":        c.Path(),
				"duration_ms": duration.Milliseconds(),
				"ip":          c.IP(),
			},
		)

		return err
	}
}
