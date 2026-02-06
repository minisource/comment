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

// ReactionRepository handles reaction data operations
type ReactionRepository struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

// NewReactionRepository creates a new reaction repository
func NewReactionRepository(db *database.MongoDB) *ReactionRepository {
	return &ReactionRepository{
		db:         db,
		collection: db.Collection("reactions"),
	}
}

// Upsert creates or updates a reaction
func (r *ReactionRepository) Upsert(ctx context.Context, reaction *models.Reaction) error {
	filter := bson.M{
		"comment_id": reaction.CommentID,
		"user_id":    reaction.UserID,
	}

	update := bson.M{
		"$set": bson.M{
			"type":       reaction.Type,
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	if result.UpsertedID != nil {
		reaction.ID = result.UpsertedID.(primitive.ObjectID)
	}

	return nil
}

// GetByUserAndComment retrieves a user's reaction to a comment
func (r *ReactionRepository) GetByUserAndComment(ctx context.Context, userID string, commentID primitive.ObjectID) (*models.Reaction, error) {
	var reaction models.Reaction
	err := r.collection.FindOne(ctx, bson.M{
		"comment_id": commentID,
		"user_id":    userID,
	}).Decode(&reaction)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &reaction, nil
}

// Delete removes a reaction
func (r *ReactionRepository) Delete(ctx context.Context, userID string, commentID primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{
		"comment_id": commentID,
		"user_id":    userID,
	})
	return err
}

// GetReactionCounts retrieves reaction counts for a comment
func (r *ReactionRepository) GetReactionCounts(ctx context.Context, commentID primitive.ObjectID) (map[string]int, int, int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"comment_id": commentID}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, 0, err
	}

	counts := make(map[string]int)
	likeCount := 0
	dislikeCount := 0

	for _, result := range results {
		reactionType := result["_id"].(string)
		count := int(result["count"].(int32))
		counts[reactionType] = count

		if reactionType == string(models.ReactionLike) {
			likeCount = count
		} else if reactionType == string(models.ReactionDislike) {
			dislikeCount = count
		}
	}

	return counts, likeCount, dislikeCount, nil
}

// GetUserReactions retrieves all reactions by a user for a list of comments
func (r *ReactionRepository) GetUserReactions(ctx context.Context, userID string, commentIDs []primitive.ObjectID) (map[primitive.ObjectID]*models.ReactionType, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"user_id":    userID,
		"comment_id": bson.M{"$in": commentIDs},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reactions []models.Reaction
	if err := cursor.All(ctx, &reactions); err != nil {
		return nil, err
	}

	result := make(map[primitive.ObjectID]*models.ReactionType)
	for _, reaction := range reactions {
		rt := reaction.Type
		result[reaction.CommentID] = &rt
	}

	return result, nil
}
