package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// TenantMiddleware extracts tenant from request
func TenantMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get tenant from header
		tenantID := c.Get("X-Tenant-ID")

		// If not in header, try query parameter
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}

		// If still empty, use default
		if tenantID == "" {
			tenantID = "default"
		}

		// Store in context
		c.Locals("tenant_id", tenantID)

		return c.Next()
	}
}
