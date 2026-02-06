package repository

import (
	"context"
	"errors"
	"time"

	"github.com/minisource/comment/internal/database"
	"github.com/minisource/comment/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SettingsRepository handles settings data operations
type SettingsRepository struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *database.MongoDB) *SettingsRepository {
	return &SettingsRepository{
		db:         db,
		collection: db.Collection("settings"),
	}
}

// GetOrCreate retrieves settings or creates default ones
func (r *SettingsRepository) GetOrCreate(ctx context.Context, tenantID, resourceType string) (*models.CommentSettings, error) {
	var settings models.CommentSettings
	err := r.collection.FindOne(ctx, bson.M{
		"tenant_id":     tenantID,
		"resource_type": resourceType,
	}).Decode(&settings)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Create default settings
			settings = models.CommentSettings{
				TenantID:            tenantID,
				ResourceType:        resourceType,
				RequireApproval:     true,
				AllowAnonymous:      false,
				AllowReplies:        true,
				MaxReplyDepth:       5,
				AllowReactions:      true,
				AllowedReactions:    []models.ReactionType{models.ReactionLike, models.ReactionDislike, models.ReactionLove, models.ReactionHaha, models.ReactionWow, models.ReactionSad, models.ReactionAngry},
				AllowAttachments:    false,
				MaxAttachments:      3,
				MaxCommentLength:    5000,
				CommentsEnabled:     true,
				NotifyOnNewComment:  true,
				NotifyOnReply:       true,
				AutoApproveVerified: false,
				BadWordsFilter:      true,
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			}

			result, err := r.collection.InsertOne(ctx, settings)
			if err != nil {
				if mongo.IsDuplicateKeyError(err) {
					// Race condition - another request created it, fetch again
					return r.GetOrCreate(ctx, tenantID, resourceType)
				}
				return nil, err
			}
			settings.ID = result.InsertedID.(primitive.ObjectID)
			return &settings, nil
		}
		return nil, err
	}

	return &settings, nil
}

// Update updates settings
func (r *SettingsRepository) Update(ctx context.Context, tenantID, resourceType string, req models.SettingsRequest) (*models.CommentSettings, error) {
	filter := bson.M{
		"tenant_id":     tenantID,
		"resource_type": resourceType,
	}

	update := bson.M{"updated_at": time.Now()}

	if req.RequireApproval != nil {
		update["require_approval"] = *req.RequireApproval
	}
	if req.AllowAnonymous != nil {
		update["allow_anonymous"] = *req.AllowAnonymous
	}
	if req.AllowReplies != nil {
		update["allow_replies"] = *req.AllowReplies
	}
	if req.MaxReplyDepth != nil {
		update["max_reply_depth"] = *req.MaxReplyDepth
	}
	if req.AllowReactions != nil {
		update["allow_reactions"] = *req.AllowReactions
	}
	if req.AllowedReactions != nil {
		update["allowed_reactions"] = req.AllowedReactions
	}
	if req.AllowAttachments != nil {
		update["allow_attachments"] = *req.AllowAttachments
	}
	if req.MaxAttachments != nil {
		update["max_attachments"] = *req.MaxAttachments
	}
	if req.MaxCommentLength != nil {
		update["max_comment_length"] = *req.MaxCommentLength
	}
	if req.CommentsEnabled != nil {
		update["comments_enabled"] = *req.CommentsEnabled
	}
	if req.NotifyOnNewComment != nil {
		update["notify_on_new_comment"] = *req.NotifyOnNewComment
	}
	if req.NotifyOnReply != nil {
		update["notify_on_reply"] = *req.NotifyOnReply
	}
	if req.AutoApproveVerified != nil {
		update["auto_approve_verified"] = *req.AutoApproveVerified
	}
	if req.BadWordsFilter != nil {
		update["bad_words_filter"] = *req.BadWordsFilter
	}
	if req.CustomBadWords != nil {
		update["custom_bad_words"] = req.CustomBadWords
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)

	var settings models.CommentSettings
	err := r.collection.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, opts).Decode(&settings)
	if err != nil {
		return nil, err
	}

	return &settings, nil
}

// GetByTenant retrieves all settings for a tenant
func (r *SettingsRepository) GetByTenant(ctx context.Context, tenantID string) ([]*models.CommentSettings, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var settings []*models.CommentSettings
	if err := cursor.All(ctx, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}
