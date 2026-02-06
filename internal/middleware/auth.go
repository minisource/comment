package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/minisource/go-sdk/auth"
)

// AuthConfig holds auth middleware configuration
type AuthConfig struct {
	AuthClient   *auth.Client
	SkipPaths    []string
	RequireAdmin []string
}

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(cfg AuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip auth for certain paths
		path := c.Path()
		for _, skipPath := range cfg.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Get authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Missing authorization header",
			})
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Invalid authorization header format",
			})
		}

		// Validate token
		result, err := cfg.AuthClient.ValidateToken(c.Context(), token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Failed to validate token",
			})
		}

		if !result.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Token is not valid",
			})
		}

		// Check admin requirement for certain paths
		for _, adminPath := range cfg.RequireAdmin {
			if strings.HasPrefix(path, adminPath) {
				if !hasAdminScope(result.Scopes) {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error":   "forbidden",
						"message": "Admin access required",
					})
				}
				break
			}
		}

		// Set user info in context
		c.Locals("user_id", result.ClientID)
		c.Locals("user_name", result.ServiceName)
		c.Locals("client_id", result.ClientID)

		return c.Next()
	}
}

// hasAdminScope checks if user has admin scope
func hasAdminScope(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "admin" || scope == "comments:moderate" {
			return true
		}
	}
	return false
}
