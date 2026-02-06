package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/minisource/comment/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB holds the MongoDB client and database
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(cfg config.MongoDBConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set client options
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)

	log.Printf("Connected to MongoDB database: %s", cfg.Database)

	return &MongoDB{
		Client:   client,
		Database: database,
	}, nil
}

// Close disconnects from MongoDB
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

// Ping checks the MongoDB connection
func (m *MongoDB) Ping(ctx context.Context) error {
	return m.Client.Ping(ctx, nil)
}

// Collection returns a MongoDB collection
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// CreateIndexes creates necessary indexes for the comment collections
func (m *MongoDB) CreateIndexes(ctx context.Context) error {
	// Comments collection indexes
	commentsCollection := m.Collection("comments")

	commentIndexes := []mongo.IndexModel{
		// Compound index for listing comments by resource
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "resource_type", Value: 1},
				{Key: "resource_id", Value: 1},
				{Key: "is_deleted", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("idx_resource_comments"),
		},
		// Index for replies
		{
			Keys: bson.D{
				{Key: "parent_id", Value: 1},
				{Key: "is_deleted", Value: 1},
			},
			Options: options.Index().SetName("idx_parent_comments"),
		},
		// Index for author's comments
		{
			Keys: bson.D{
				{Key: "author_id", Value: 1},
				{Key: "is_deleted", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_author_comments"),
		},
		// Index for moderation queue
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "status", Value: 1},
				{Key: "created_at", Value: 1},
			},
			Options: options.Index().SetName("idx_moderation_queue"),
		},
		// Index for pinned comments
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "resource_type", Value: 1},
				{Key: "resource_id", Value: 1},
				{Key: "is_pinned", Value: 1},
			},
			Options: options.Index().SetName("idx_pinned_comments"),
		},
		// Text index for content search
		{
			Keys: bson.D{
				{Key: "content", Value: "text"},
				{Key: "author_name", Value: "text"},
			},
			Options: options.Index().SetName("idx_content_search"),
		},
		// Index for sorting by popularity
		{
			Keys: bson.D{
				{Key: "like_count", Value: -1},
			},
			Options: options.Index().SetName("idx_like_count"),
		},
		// TTL index for soft-deleted comments (auto-delete after 30 days)
		{
			Keys: bson.D{
				{Key: "deleted_at", Value: 1},
			},
			Options: options.Index().
				SetName("idx_deleted_ttl").
				SetExpireAfterSeconds(30 * 24 * 60 * 60), // 30 days
		},
	}

	if _, err := commentsCollection.Indexes().CreateMany(ctx, commentIndexes); err != nil {
		return fmt.Errorf("failed to create comment indexes: %w", err)
	}

	// Reactions collection indexes
	reactionsCollection := m.Collection("reactions")

	reactionIndexes := []mongo.IndexModel{
		// Unique index for user reaction per comment
		{
			Keys: bson.D{
				{Key: "comment_id", Value: 1},
				{Key: "user_id", Value: 1},
			},
			Options: options.Index().
				SetName("idx_user_reaction").
				SetUnique(true),
		},
		// Index for counting reactions by type
		{
			Keys: bson.D{
				{Key: "comment_id", Value: 1},
				{Key: "type", Value: 1},
			},
			Options: options.Index().SetName("idx_comment_reaction_type"),
		},
	}

	if _, err := reactionsCollection.Indexes().CreateMany(ctx, reactionIndexes); err != nil {
		return fmt.Errorf("failed to create reaction indexes: %w", err)
	}

	// Reports collection indexes
	reportsCollection := m.Collection("reports")

	reportIndexes := []mongo.IndexModel{
		// Index for comment reports
		{
			Keys: bson.D{
				{Key: "comment_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("idx_comment_reports"),
		},
		// Prevent duplicate reports from same user
		{
			Keys: bson.D{
				{Key: "comment_id", Value: 1},
				{Key: "reporter_id", Value: 1},
			},
			Options: options.Index().
				SetName("idx_unique_report").
				SetUnique(true),
		},
	}

	if _, err := reportsCollection.Indexes().CreateMany(ctx, reportIndexes); err != nil {
		return fmt.Errorf("failed to create report indexes: %w", err)
	}

	// Settings collection indexes
	settingsCollection := m.Collection("settings")

	settingsIndexes := []mongo.IndexModel{
		// Unique index for tenant + resource type settings
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "resource_type", Value: 1},
			},
			Options: options.Index().
				SetName("idx_tenant_settings").
				SetUnique(true),
		},
	}

	if _, err := settingsCollection.Indexes().CreateMany(ctx, settingsIndexes); err != nil {
		return fmt.Errorf("failed to create settings indexes: %w", err)
	}

	log.Println("MongoDB indexes created successfully")
	return nil
}
