package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/minisource/comment/internal/database"
	"github.com/minisource/go-common/response"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *database.MongoDB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *database.MongoDB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

// HealthCheck returns service health status
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} response.HealthResponse
// @Failure 503 {object} response.HealthResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	// Check MongoDB connection
	mongoStatus := "healthy"
	if err := h.db.Ping(ctx); err != nil {
		mongoStatus = "unhealthy: " + err.Error()
	}

	resp := response.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services: map[string]string{
			"mongodb": mongoStatus,
		},
	}

	// If any service is unhealthy, set overall status
	if mongoStatus != "healthy" {
		resp.Status = "unhealthy"
		return c.Status(fiber.StatusServiceUnavailable).JSON(resp)
	}

	return c.JSON(resp)
}

// Readiness checks if service is ready to accept traffic
// @Summary Readiness check
// @Tags health
// @Produce json
// @Success 200 {object} response.ReadinessResponse
// @Failure 503 {object} response.ReadinessResponse
// @Router /ready [get]
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	// Check MongoDB connection
	if err := h.db.Ping(ctx); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(response.ReadinessResponse{
			Ready:   false,
			Message: "MongoDB not ready: " + err.Error(),
		})
	}

	return c.JSON(response.ReadinessResponse{
		Ready:   true,
		Message: "Service is ready",
	})
}

// Liveness checks if service is alive
// @Summary Liveness check
// @Tags health
// @Produce json
// @Success 200 {object} response.LivenessResponse
// @Router /live [get]
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(response.LivenessResponse{
		Alive: true,
	})
}
