package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/minisource/comment/internal/models"
	"github.com/minisource/comment/internal/usecase"
	"github.com/minisource/go-common/response"
)

// ReactionHandler handles HTTP requests for reactions
type ReactionHandler struct {
	reactionUsecase *usecase.ReactionUsecase
}

// NewReactionHandler creates a new reaction handler
func NewReactionHandler(reactionUsecase *usecase.ReactionUsecase) *ReactionHandler {
	return &ReactionHandler{
		reactionUsecase: reactionUsecase,
	}
}

// AddReaction adds or updates a reaction
// @Summary Add or update a reaction to a comment
// @Tags reactions
// @Accept json
// @Produce json
// @Param id path string true "Comment ID"
// @Param request body models.ReactionRequest true "Reaction data"
// @Success 200 {object} response.SuccessMessage
// @Failure 400 {object} response.Response
// @Router /api/v1/comments/{id}/reactions [post]
func (h *ReactionHandler) AddReaction(c *fiber.Ctx) error {
	commentID := c.Params("id")
	userID := c.Locals("user_id").(string)

	var req models.ReactionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid_request", "Invalid request body")
	}

	// Validate reaction type
	if !isValidReactionType(req.Type) {
		return response.BadRequest(c, "invalid_reaction_type", "Invalid reaction type. Valid types: like, dislike, love, haha, wow, sad, angry")
	}

	if err := h.reactionUsecase.AddReaction(c.Context(), commentID, req.Type, userID); err != nil {
		return response.BadRequest(c, "reaction_failed", err.Error())
	}

	return response.OKMessage(c, "Reaction added successfully")
}

// RemoveReaction removes a reaction
// @Summary Remove a reaction from a comment
// @Tags reactions
// @Produce json
// @Param id path string true "Comment ID"
// @Success 204 "No Content"
// @Failure 400 {object} response.Response
// @Router /api/v1/comments/{id}/reactions [delete]
func (h *ReactionHandler) RemoveReaction(c *fiber.Ctx) error {
	commentID := c.Params("id")
	userID := c.Locals("user_id").(string)

	if err := h.reactionUsecase.RemoveReaction(c.Context(), commentID, userID); err != nil {
		return response.BadRequest(c, "remove_reaction_failed", err.Error())
	}

	return response.NoContent(c)
}

// GetUserReaction gets the current user's reaction to a comment
// @Summary Get current user's reaction to a comment
// @Tags reactions
// @Produce json
// @Param id path string true "Comment ID"
// @Success 200 {object} UserReactionResponse
// @Router /api/v1/comments/{id}/reactions/me [get]
func (h *ReactionHandler) GetUserReaction(c *fiber.Ctx) error {
	commentID := c.Params("id")
	userID := c.Locals("user_id").(string)

	reaction, err := h.reactionUsecase.GetUserReaction(c.Context(), commentID, userID)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	resp := UserReactionResponse{
		CommentID:  commentID,
		HasReacted: reaction != nil,
	}
	if reaction != nil {
		resp.ReactionType = string(*reaction)
	}

	return response.OK(c, resp)
}

// UserReactionResponse represents user reaction response
type UserReactionResponse struct {
	CommentID    string `json:"comment_id"`
	HasReacted   bool   `json:"has_reacted"`
	ReactionType string `json:"reaction_type,omitempty"`
}

// isValidReactionType checks if a reaction type is valid
func isValidReactionType(rt models.ReactionType) bool {
	validTypes := []models.ReactionType{
		models.ReactionLike,
		models.ReactionDislike,
		models.ReactionLove,
		models.ReactionHaha,
		models.ReactionWow,
		models.ReactionSad,
		models.ReactionAngry,
	}
	for _, t := range validTypes {
		if rt == t {
			return true
		}
	}
	return false
}
