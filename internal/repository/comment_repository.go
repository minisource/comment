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

// CommentRepository handles comment data operations
type CommentRepository struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

// NewCommentRepository creates a new comment repository
func NewCommentRepository(db *database.MongoDB) *CommentRepository {
	return &CommentRepository{
		db:         db,
		collection: db.Collection("comments"),
	}
}

// Create inserts a new comment
func (r *CommentRepository) Create(ctx context.Context, comment *models.Comment) error {
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, comment)
	if err != nil {
		return err
	}

	comment.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID retrieves a comment by ID
func (r *CommentRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Comment, error) {
	var comment models.Comment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&comment)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &comment, nil
}

// Update updates a comment
func (r *CommentRepository) Update(ctx context.Context, comment *models.Comment) error {
	comment.UpdatedAt = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": comment.ID},
		bson.M{"$set": comment},
	)
	return err
}

// UpdateFields updates specific fields of a comment
func (r *CommentRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
	fields["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": fields},
	)
	return err
}

// SoftDelete marks a comment as deleted
func (r *CommentRepository) SoftDelete(ctx context.Context, id primitive.ObjectID, deletedBy string) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"is_deleted": true,
				"deleted_at": now,
				"deleted_by": deletedBy,
				"updated_at": now,
			},
		},
	)
	return err
}

// HardDelete permanently deletes a comment
func (r *CommentRepository) HardDelete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// List retrieves comments with filters
func (r *CommentRepository) List(ctx context.Context, req models.ListCommentsRequest) ([]*models.Comment, int64, error) {
	filter := bson.M{}

	if req.TenantID != "" {
		filter["tenant_id"] = req.TenantID
	}
	if req.ResourceType != "" {
		filter["resource_type"] = req.ResourceType
	}
	if req.ResourceID != "" {
		filter["resource_id"] = req.ResourceID
	}
	if req.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(req.ParentID)
		if err == nil {
			filter["parent_id"] = parentID
		}
	} else {
		// If no parent ID specified, get only root comments
		filter["parent_id"] = nil
	}
	if req.Status != "" {
		filter["status"] = req.Status
	}
	if req.AuthorID != "" {
		filter["author_id"] = req.AuthorID
	}
	if req.IsPinned != nil {
		filter["is_pinned"] = *req.IsPinned
	}
	if !req.IncludeDeleted {
		filter["is_deleted"] = false
	}

	// Count total
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Sort options
	sortField := "created_at"
	sortOrder := -1 // desc
	if req.SortBy != "" {
		switch req.SortBy {
		case "created_at", "like_count", "reply_count":
			sortField = req.SortBy
		}
	}
	if req.SortOrder == "asc" {
		sortOrder = 1
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "is_pinned", Value: -1}, {Key: sortField, Value: sortOrder}}).
		SetSkip(int64((req.Page - 1) * req.PageSize)).
		SetLimit(int64(req.PageSize))

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var comments []*models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// GetReplies retrieves replies for a comment
func (r *CommentRepository) GetReplies(ctx context.Context, parentID primitive.ObjectID, page, pageSize int) ([]*models.Comment, int64, error) {
	filter := bson.M{
		"parent_id":  parentID,
		"is_deleted": false,
		"status":     models.StatusApproved,
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var replies []*models.Comment
	if err := cursor.All(ctx, &replies); err != nil {
		return nil, 0, err
	}

	return replies, total, nil
}

// GetPending retrieves pending comments for moderation
func (r *CommentRepository) GetPending(ctx context.Context, tenantID string, page, pageSize int) ([]*models.Comment, int64, error) {
	filter := bson.M{
		"status":     models.StatusPending,
		"is_deleted": false,
	}
	if tenantID != "" {
		filter["tenant_id"] = tenantID
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var comments []*models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// GetStats retrieves statistics for a resource
func (r *CommentRepository) GetStats(ctx context.Context, tenantID, resourceType, resourceID string) (*models.CommentStats, error) {
	filter := bson.M{
		"tenant_id":     tenantID,
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"is_deleted":    false,
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.M{
			"_id":      nil,
			"total":    bson.M{"$sum": 1},
			"approved": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", models.StatusApproved}}, 1, 0}}},
			"pending":  bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", models.StatusPending}}, 1, 0}}},
			"rejected": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", models.StatusRejected}}, 1, 0}}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	stats := &models.CommentStats{}
	if len(results) > 0 {
		stats.TotalComments = int64(results[0]["total"].(int32))
		stats.ApprovedCount = int64(results[0]["approved"].(int32))
		stats.PendingCount = int64(results[0]["pending"].(int32))
		stats.RejectedCount = int64(results[0]["rejected"].(int32))
	}

	return stats, nil
}

// IncrementReplyCount increments the reply count of a comment
func (r *CommentRepository) IncrementReplyCount(ctx context.Context, id primitive.ObjectID, delta int) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$inc": bson.M{"reply_count": delta},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

// UpdateReactionCounts updates the reaction counts of a comment
func (r *CommentRepository) UpdateReactionCounts(ctx context.Context, id primitive.ObjectID, likeCount, dislikeCount int, reactionCounts map[string]int) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"like_count":      likeCount,
				"dislike_count":   dislikeCount,
				"reaction_counts": reactionCounts,
				"updated_at":      time.Now(),
			},
		},
	)
	return err
}

// IncrementReportCount increments the report count of a comment
func (r *CommentRepository) IncrementReportCount(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$inc": bson.M{"report_count": 1},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

// Search searches comments by content
func (r *CommentRepository) Search(ctx context.Context, tenantID, query string, page, pageSize int) ([]*models.Comment, int64, error) {
	filter := bson.M{
		"$text":      bson.M{"$search": query},
		"tenant_id":  tenantID,
		"is_deleted": false,
		"status":     models.StatusApproved,
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	findOptions := options.Find().
		SetSort(bson.M{"score": bson.M{"$meta": "textScore"}}).
		SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var comments []*models.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}
