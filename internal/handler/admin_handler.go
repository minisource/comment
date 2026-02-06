package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/minisource/comment/internal/models"
	"github.com/minisource/comment/internal/usecase"
	"github.com/minisource/go-common/response"
)

// AdminHandler handles admin HTTP requests
type AdminHandler struct {
	commentUsecase *usecase.CommentUsecase
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(commentUsecase *usecase.CommentUsecase) *AdminHandler {
	return &AdminHandler{
		commentUsecase: commentUsecase,
	}
}

// GetPendingComments gets pending comments for moderation
// @Summary Get pending comments for moderation
// @Tags admin
// @Produce json
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {array} models.Comment
// @Router /api/v1/admin/comments/pending [get]
func (h *AdminHandler) GetPendingComments(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	comments, total, err := h.commentUsecase.GetPendingComments(c.Context(), tenantID, page, pageSize)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, fiber.Map{
		"comments": comments,
		"total":    total,
	})
}

// ModerateComment approves or rejects a comment
// @Summary Moderate a comment (approve/reject)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Comment ID"
// @Param request body models.ModerateCommentRequest true "Moderation data"
// @Success 200 {object} models.Comment
// @Failure 400 {object} response.Response
// @Router /api/v1/admin/comments/{id}/moderate [post]
func (h *AdminHandler) ModerateComment(c *fiber.Ctx) error {
	id := c.Params("id")
	moderatorID, _ := c.Locals("user_id").(string)

	var req models.ModerateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid_request", "Invalid request body")
	}

	// Validate status
	if req.Status != models.StatusApproved && req.Status != models.StatusRejected && req.Status != models.StatusSpam {
		return response.BadRequest(c, "invalid_status", "Status must be 'approved', 'rejected', or 'spam'")
	}

	comment, err := h.commentUsecase.ModerateComment(c.Context(), id, req, moderatorID)
	if err != nil {
		return response.BadRequest(c, "moderate_failed", err.Error())
	}

	return response.OK(c, comment)
}

// PinComment pins or unpins a comment
// @Summary Pin or unpin a comment
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Comment ID"
// @Param request body models.PinCommentRequest true "Pin data"
// @Success 200 {object} models.Comment
// @Failure 400 {object} response.Response
// @Router /api/v1/admin/comments/{id}/pin [post]
func (h *AdminHandler) PinComment(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, _ := c.Locals("user_id").(string)

	var req models.PinCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid_request", "Invalid request body")
	}

	comment, err := h.commentUsecase.PinComment(c.Context(), id, req.IsPinned, userID)
	if err != nil {
		return response.BadRequest(c, "pin_failed", err.Error())
	}

	return response.OK(c, comment)
}

// HardDelete permanently deletes a comment
// @Summary Permanently delete a comment
// @Tags admin
// @Produce json
// @Param id path string true "Comment ID"
// @Success 204 "No Content"
// @Failure 400 {object} response.Response
// @Router /api/v1/admin/comments/{id} [delete]
func (h *AdminHandler) HardDelete(c *fiber.Ctx) error {
	id := c.Params("id")
	userID, _ := c.Locals("user_id").(string)

	// Use DeleteComment with isAdmin=true
	if err := h.commentUsecase.DeleteComment(c.Context(), id, userID, true); err != nil {
		return response.BadRequest(c, "delete_failed", err.Error())
	}

	return response.NoContent(c)
}

// BulkModerate moderates multiple comments at once
// @Summary Bulk moderate comments
// @Tags admin
// @Accept json
// @Produce json
// @Param request body BulkModerateRequest true "Bulk moderation data"
// @Success 200 {object} BulkModerateResponse
// @Failure 400 {object} response.Response
// @Router /api/v1/admin/comments/bulk-moderate [post]
func (h *AdminHandler) BulkModerate(c *fiber.Ctx) error {
	moderatorID, _ := c.Locals("user_id").(string)

	var req BulkModerateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid_request", "Invalid request body")
	}

	if len(req.CommentIDs) == 0 {
		return response.BadRequest(c, "invalid_request", "No comment IDs provided")
	}

	successCount := 0
	failedIDs := []string{}

	for _, commentID := range req.CommentIDs {
		_, err := h.commentUsecase.ModerateComment(c.Context(), commentID, models.ModerateCommentRequest{
			Status:          req.Status,
			RejectionReason: req.RejectionReason,
		}, moderatorID)
		if err != nil {
			failedIDs = append(failedIDs, commentID)
		} else {
			successCount++
		}
	}

	return response.OK(c, BulkModerateResponse{
		SuccessCount: successCount,
		FailedCount:  len(failedIDs),
		FailedIDs:    failedIDs,
	})
}

// BulkModerateRequest represents bulk moderation request
type BulkModerateRequest struct {
	CommentIDs      []string             `json:"comment_ids"`
	Status          models.CommentStatus `json:"status"`
	RejectionReason string               `json:"rejection_reason,omitempty"`
}

// BulkModerateResponse represents bulk moderation response
type BulkModerateResponse struct {
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	FailedIDs    []string `json:"failed_ids,omitempty"`
}
