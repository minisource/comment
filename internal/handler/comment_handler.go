package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/minisource/comment/internal/models"
	"github.com/minisource/comment/internal/usecase"
	"github.com/minisource/go-common/response"
)

// CommentHandler handles HTTP requests for comments
type CommentHandler struct {
	commentUsecase *usecase.CommentUsecase
}

// NewCommentHandler creates a new comment handler
func NewCommentHandler(commentUsecase *usecase.CommentUsecase) *CommentHandler {
	return &CommentHandler{
		commentUsecase: commentUsecase,
	}
}

// Create creates a new comment
// @Summary Create a new comment
// @Tags comments
// @Accept json
// @Produce json
// @Param request body models.CreateCommentRequest true "Comment data"
// @Success 201 {object} models.Comment
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/comments [post]
func (h *CommentHandler) Create(c *fiber.Ctx) error {
	var req models.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid_request", "Invalid request body")
	}

	// Get tenant and user from context
	tenantID, _ := c.Locals("tenant_id").(string)
	userID, _ := c.Locals("user_id").(string)
	userName, _ := c.Locals("user_name").(string)
	userEmail, _ := c.Locals("user_email").(string)

	// Set tenant from context if not in request
	if req.TenantID == "" {
		req.TenantID = tenantID
	}

	comment, err := h.commentUsecase.CreateComment(c.Context(), req, userID, userName, userEmail, c.IP(), c.Get("User-Agent"))
	if err != nil {
		return response.BadRequest(c, "create_failed", err.Error())
	}

	return response.Created(c, comment)
}

// Get gets a comment by ID
// @Summary Get a comment by ID
// @Tags comments
// @Produce json
// @Param id path string true "Comment ID"
// @Success 200 {object} models.Comment
// @Failure 404 {object} response.Response
// @Router /api/v1/comments/{id} [get]
func (h *CommentHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")

	comment, err := h.commentUsecase.GetComment(c.Context(), id)
	if err != nil {
		if err.Error() == "comment not found" {
			return response.NotFound(c, "Comment not found")
		}
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, comment)
}

// Update updates a comment
// @Summary Update a comment
// @Tags comments
// @Accept json
// @Produce json
// @Param id path string true "Comment ID"
// @Param request body models.UpdateCommentRequest true "Update data"
// @Success 200 {object} models.Comment
// @Failure 400 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/comments/{id} [put]
func (h *CommentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, _ := c.Locals("user_id").(string)

	var req models.UpdateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid_request", "Invalid request body")
	}

	comment, err := h.commentUsecase.UpdateComment(c.Context(), id, req, userID, false)
	if err != nil {
		if err.Error() == "comment not found" {
			return response.NotFound(c, err.Error())
		}
		if err.Error() == "you can only edit your own comments" {
			return response.Forbidden(c, err.Error())
		}
		return response.BadRequest(c, "update_failed", err.Error())
	}

	return response.OK(c, comment)
}

// Delete soft deletes a comment
// @Summary Delete a comment
// @Tags comments
// @Produce json
// @Param id path string true "Comment ID"
// @Success 204 "No Content"
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/comments/{id} [delete]
func (h *CommentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, _ := c.Locals("user_id").(string)

	if err := h.commentUsecase.DeleteComment(c.Context(), id, userID, false); err != nil {
		if err.Error() == "comment not found" {
			return response.NotFound(c, err.Error())
		}
		if err.Error() == "you can only delete your own comments" {
			return response.Forbidden(c, err.Error())
		}
		return response.InternalError(c, err.Error())
	}

	return response.NoContent(c)
}

// List lists comments with filters
// @Summary List comments
// @Tags comments
// @Produce json
// @Param resource_type query string true "Resource type"
// @Param resource_id query string true "Resource ID"
// @Param status query string false "Status filter"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param sort_by query string false "Sort field"
// @Param sort_order query string false "Sort order"
// @Success 200 {object} models.ListCommentsResponse
// @Router /api/v1/comments [get]
func (h *CommentHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	userID, _ := c.Locals("user_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	req := models.ListCommentsRequest{
		TenantID:     tenantID,
		ResourceType: c.Query("resource_type"),
		ResourceID:   c.Query("resource_id"),
		Status:       models.CommentStatus(c.Query("status")),
		ParentID:     c.Query("parent_id"),
		Page:         page,
		PageSize:     pageSize,
		SortBy:       c.Query("sort_by", "created_at"),
		SortOrder:    c.Query("sort_order", "desc"),
	}

	resp, err := h.commentUsecase.ListComments(c.Context(), req, userID, false)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, resp)
}

// GetReplies gets replies to a comment
// @Summary Get replies to a comment
// @Tags comments
// @Produce json
// @Param id path string true "Parent Comment ID"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {array} models.Comment
// @Router /api/v1/comments/{id}/replies [get]
func (h *CommentHandler) GetReplies(c *fiber.Ctx) error {
	id := c.Params("id")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	replies, total, err := h.commentUsecase.GetReplies(c.Context(), id, page, pageSize)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, fiber.Map{
		"replies": replies,
		"total":   total,
	})
}

// Search searches comments
// @Summary Search comments
// @Tags comments
// @Produce json
// @Param q query string true "Search query"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {array} models.Comment
// @Router /api/v1/comments/search [get]
func (h *CommentHandler) Search(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	query := c.Query("q")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	comments, total, err := h.commentUsecase.SearchComments(c.Context(), tenantID, query, page, pageSize)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, fiber.Map{
		"comments": comments,
		"total":    total,
	})
}

// GetStats gets comment statistics
// @Summary Get comment statistics
// @Tags comments
// @Produce json
// @Param resource_type query string true "Resource type"
// @Param resource_id query string true "Resource ID"
// @Success 200 {object} models.CommentStats
// @Router /api/v1/comments/stats [get]
func (h *CommentHandler) GetStats(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	resourceType := c.Query("resource_type")
	resourceID := c.Query("resource_id")

	stats, err := h.commentUsecase.GetCommentStats(c.Context(), tenantID, resourceType, resourceID)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, stats)
}
