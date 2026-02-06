package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/minisource/comment/internal/models"
	"github.com/minisource/comment/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ReactionUsecase handles reaction business logic
type ReactionUsecase struct {
	commentRepo  *repository.CommentRepository
	reactionRepo *repository.ReactionRepository
}

// NewReactionUsecase creates a new reaction usecase
func NewReactionUsecase(
	commentRepo *repository.CommentRepository,
	reactionRepo *repository.ReactionRepository,
) *ReactionUsecase {
	return &ReactionUsecase{
		commentRepo:  commentRepo,
		reactionRepo: reactionRepo,
	}
}

// AddReaction adds or updates a reaction to a comment
func (u *ReactionUsecase) AddReaction(ctx context.Context, commentID string, reactionType models.ReactionType, userID string) error {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return fmt.Errorf("invalid comment ID")
	}

	// Check if comment exists
	comment, err := u.commentRepo.GetByID(ctx, oid)
	if err != nil {
		return err
	}
	if comment == nil {
		return fmt.Errorf("comment not found")
	}

	// Cannot react to deleted comments
	if comment.IsDeleted {
		return fmt.Errorf("cannot react to deleted comment")
	}

	// Upsert reaction
	reaction := &models.Reaction{
		CommentID: oid,
		UserID:    userID,
		Type:      reactionType,
	}

	if err := u.reactionRepo.Upsert(ctx, reaction); err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	// Update reaction counts
	if err := u.updateReactionCounts(ctx, oid); err != nil {
		log.Printf("Failed to update reaction counts: %v", err)
	}

	return nil
}

// RemoveReaction removes a reaction from a comment
func (u *ReactionUsecase) RemoveReaction(ctx context.Context, commentID string, userID string) error {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return fmt.Errorf("invalid comment ID")
	}

	if err := u.reactionRepo.Delete(ctx, userID, oid); err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}

	// Update reaction counts
	if err := u.updateReactionCounts(ctx, oid); err != nil {
		log.Printf("Failed to update reaction counts: %v", err)
	}

	return nil
}

// GetUserReaction gets the current user's reaction to a comment
func (u *ReactionUsecase) GetUserReaction(ctx context.Context, commentID string, userID string) (*models.ReactionType, error) {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, fmt.Errorf("invalid comment ID")
	}

	reaction, err := u.reactionRepo.GetByUserAndComment(ctx, userID, oid)
	if err != nil {
		return nil, err
	}
	if reaction == nil {
		return nil, nil
	}

	return &reaction.Type, nil
}

// GetUserReactionsForComments gets user reactions for multiple comments
func (u *ReactionUsecase) GetUserReactionsForComments(ctx context.Context, commentIDs []string, userID string) (map[string]*models.ReactionType, error) {
	oids := make([]primitive.ObjectID, 0, len(commentIDs))
	for _, id := range commentIDs {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}
		oids = append(oids, oid)
	}

	reactions, err := u.reactionRepo.GetUserReactions(ctx, userID, oids)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*models.ReactionType)
	for oid, reaction := range reactions {
		result[oid.Hex()] = reaction
	}

	return result, nil
}

// updateReactionCounts updates the reaction counts on a comment
func (u *ReactionUsecase) updateReactionCounts(ctx context.Context, commentID primitive.ObjectID) error {
	counts, likeCount, dislikeCount, err := u.reactionRepo.GetReactionCounts(ctx, commentID)
	if err != nil {
		return err
	}

	return u.commentRepo.UpdateReactionCounts(ctx, commentID, likeCount, dislikeCount, counts)
}
