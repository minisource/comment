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

// ReportRepository handles report data operations
type ReportRepository struct {
	db         *database.MongoDB
	collection *mongo.Collection
}

// NewReportRepository creates a new report repository
func NewReportRepository(db *database.MongoDB) *ReportRepository {
	return &ReportRepository{
		db:         db,
		collection: db.Collection("reports"),
	}
}

// Create inserts a new report
func (r *ReportRepository) Create(ctx context.Context, report *models.Report) error {
	report.CreatedAt = time.Now()
	report.Status = "pending"

	result, err := r.collection.InsertOne(ctx, report)
	if err != nil {
		// Check for duplicate report
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("you have already reported this comment")
		}
		return err
	}

	report.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByCommentID retrieves reports for a comment
func (r *ReportRepository) GetByCommentID(ctx context.Context, commentID primitive.ObjectID) ([]*models.Report, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"comment_id": commentID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var reports []*models.Report
	if err := cursor.All(ctx, &reports); err != nil {
		return nil, err
	}

	return reports, nil
}

// GetPending retrieves pending reports
func (r *ReportRepository) GetPending(ctx context.Context, page, pageSize int) ([]*models.Report, int64, error) {
	filter := bson.M{"status": "pending"}

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

	var reports []*models.Report
	if err := cursor.All(ctx, &reports); err != nil {
		return nil, 0, err
	}

	return reports, total, nil
}

// UpdateStatus updates the status of a report
func (r *ReportRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status, reviewedBy string) error {
	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":      status,
				"reviewed_by": reviewedBy,
				"reviewed_at": now,
			},
		},
	)
	return err
}

// CountByCommentID counts reports for a comment
func (r *ReportRepository) CountByCommentID(ctx context.Context, commentID primitive.ObjectID) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"comment_id": commentID})
}
