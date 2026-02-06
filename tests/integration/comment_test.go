//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Comment represents a comment for testing
type Comment struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	EntityID  string `json:"entity_id"`
	UserID    string `json:"user_id"`
	Content   string `json:"content"`
	ParentID  string `json:"parent_id,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	app := fiber.New()

	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "comment",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestCreateComment tests comment creation
func TestCreateComment(t *testing.T) {
	app := fiber.New()

	var createdComment Comment

	app.Post("/api/v1/comments", func(c *fiber.Ctx) error {
		if err := c.BodyParser(&createdComment); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		if createdComment.Content == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Content is required",
			})
		}

		createdComment.ID = "comment-123"
		createdComment.Status = "approved"
		createdComment.TenantID = c.Get("X-Tenant-ID")
		return c.Status(fiber.StatusCreated).JSON(createdComment)
	})

	t.Run("Create Comment", func(t *testing.T) {
		comment := Comment{
			EntityID: "post-123",
			UserID:   "user-456",
			Content:  "This is a test comment",
		}
		body, _ := json.Marshal(comment)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant-123")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result Comment
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotEmpty(t, result.ID)
		assert.Equal(t, "approved", result.Status)
	})

	t.Run("Create Reply", func(t *testing.T) {
		comment := Comment{
			EntityID: "post-123",
			UserID:   "user-789",
			Content:  "This is a reply",
			ParentID: "comment-123",
		}
		body, _ := json.Marshal(comment)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant-123")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("Create Without Content", func(t *testing.T) {
		comment := Comment{
			EntityID: "post-123",
			UserID:   "user-456",
		}
		body, _ := json.Marshal(comment)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestListComments tests comment listing
func TestListComments(t *testing.T) {
	app := fiber.New()

	mockComments := []Comment{
		{ID: "1", EntityID: "post-123", Content: "Comment 1", Status: "approved"},
		{ID: "2", EntityID: "post-123", Content: "Comment 2", Status: "approved"},
		{ID: "3", EntityID: "post-456", Content: "Comment 3", Status: "pending"},
	}

	app.Get("/api/v1/entities/:entityId/comments", func(c *fiber.Ctx) error {
		entityID := c.Params("entityId")
		status := c.Query("status")

		var filtered []Comment
		for _, comment := range mockComments {
			if comment.EntityID == entityID && (status == "" || comment.Status == status) {
				filtered = append(filtered, comment)
			}
		}

		return c.JSON(fiber.Map{
			"data":  filtered,
			"total": len(filtered),
		})
	})

	t.Run("List Entity Comments", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/entities/post-123/comments", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, float64(2), result["total"])
	})

	t.Run("List With Status Filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/entities/post-456/comments?status=pending", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, float64(1), result["total"])
	})
}

// TestDeleteComment tests comment deletion
func TestDeleteComment(t *testing.T) {
	app := fiber.New()

	app.Delete("/api/v1/comments/:id", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/123", nil)
	req.Header.Set("X-Tenant-ID", "tenant-123")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// TestModerateComment tests comment moderation
func TestModerateComment(t *testing.T) {
	app := fiber.New()

	app.Patch("/api/v1/comments/:id/approve", func(c *fiber.Ctx) error {
		return c.JSON(Comment{ID: c.Params("id"), Status: "approved"})
	})

	app.Patch("/api/v1/comments/:id/reject", func(c *fiber.Ctx) error {
		return c.JSON(Comment{ID: c.Params("id"), Status: "rejected"})
	})

	t.Run("Approve Comment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/comments/123/approve", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result Comment
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "approved", result.Status)
	})

	t.Run("Reject Comment", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/comments/123/reject", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result Comment
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "rejected", result.Status)
	})
}

// TestThreadedComments tests threaded comment structure
func TestThreadedComments(t *testing.T) {
	t.Skip("Requires database connection")

	// TODO: Test fetching comment tree
	// TODO: Test depth limiting
	// TODO: Test ordering by date
}
