package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CommentStatus represents the moderation status of a comment
type CommentStatus string

const (
	StatusPending  CommentStatus = "pending"
	StatusApproved CommentStatus = "approved"
	StatusRejected CommentStatus = "rejected"
	StatusSpam     CommentStatus = "spam"
)

// ReactionType represents the type of reaction
type ReactionType string

const (
	ReactionLike    ReactionType = "like"
	ReactionDislike ReactionType = "dislike"
	ReactionLove    ReactionType = "love"
	ReactionHaha    ReactionType = "haha"
	ReactionWow     ReactionType = "wow"
	ReactionSad     ReactionType = "sad"
	ReactionAngry   ReactionType = "angry"
)

// Comment represents a comment in the system
type Comment struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	TenantID     string              `bson:"tenant_id" json:"tenantId"`                     // Multi-tenant support (e.g., "shop", "ticket", "blog")
	ResourceType string              `bson:"resource_type" json:"resourceType"`             // e.g., "product", "ticket", "article"
	ResourceID   string              `bson:"resource_id" json:"resourceId"`                 // ID of the resource being commented on
	ParentID     *primitive.ObjectID `bson:"parent_id,omitempty" json:"parentId,omitempty"` // For replies
	RootID       *primitive.ObjectID `bson:"root_id,omitempty" json:"rootId,omitempty"`     // Root comment ID for nested replies

	// Author info
	AuthorID     string `bson:"author_id" json:"authorId"`
	AuthorName   string `bson:"author_name" json:"authorName"`
	AuthorEmail  string `bson:"author_email,omitempty" json:"authorEmail,omitempty"`
	AuthorAvatar string `bson:"author_avatar,omitempty" json:"authorAvatar,omitempty"`
	IsAnonymous  bool   `bson:"is_anonymous" json:"isAnonymous"`

	// Content
	Content     string       `bson:"content" json:"content"`
	ContentHTML string       `bson:"content_html,omitempty" json:"contentHtml,omitempty"` // Sanitized HTML
	Attachments []Attachment `bson:"attachments,omitempty" json:"attachments,omitempty"`

	// Moderation
	Status          CommentStatus `bson:"status" json:"status"`
	ModeratedBy     string        `bson:"moderated_by,omitempty" json:"moderatedBy,omitempty"`
	ModeratedAt     *time.Time    `bson:"moderated_at,omitempty" json:"moderatedAt,omitempty"`
	RejectionReason string        `bson:"rejection_reason,omitempty" json:"rejectionReason,omitempty"`
	FlaggedWords    []string      `bson:"flagged_words,omitempty" json:"flaggedWords,omitempty"`
	ReportCount     int           `bson:"report_count" json:"reportCount"`

	// Features
	IsPinned    bool         `bson:"is_pinned" json:"isPinned"`
	PinnedBy    string       `bson:"pinned_by,omitempty" json:"pinnedBy,omitempty"`
	PinnedAt    *time.Time   `bson:"pinned_at,omitempty" json:"pinnedAt,omitempty"`
	IsEdited    bool         `bson:"is_edited" json:"isEdited"`
	EditHistory []EditRecord `bson:"edit_history,omitempty" json:"editHistory,omitempty"`

	// Stats
	ReplyCount     int            `bson:"reply_count" json:"replyCount"`
	LikeCount      int            `bson:"like_count" json:"likeCount"`
	DislikeCount   int            `bson:"dislike_count" json:"dislikeCount"`
	ReactionCounts map[string]int `bson:"reaction_counts,omitempty" json:"reactionCounts,omitempty"`

	// Metadata
	IPAddress string         `bson:"ip_address,omitempty" json:"-"` // Hidden from API
	UserAgent string         `bson:"user_agent,omitempty" json:"-"` // Hidden from API
	Metadata  map[string]any `bson:"metadata,omitempty" json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time  `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deletedAt,omitempty"`
	IsDeleted bool       `bson:"is_deleted" json:"isDeleted"`
	DeletedBy string     `bson:"deleted_by,omitempty" json:"deletedBy,omitempty"`

	// Depth for nested replies
	Depth int `bson:"depth" json:"depth"`
}

// Attachment represents a file attached to a comment
type Attachment struct {
	ID         string    `bson:"id" json:"id"`
	Type       string    `bson:"type" json:"type"` // image, video, file
	URL        string    `bson:"url" json:"url"`
	Filename   string    `bson:"filename" json:"filename"`
	Size       int64     `bson:"size" json:"size"`
	MimeType   string    `bson:"mime_type" json:"mimeType"`
	UploadedAt time.Time `bson:"uploaded_at" json:"uploadedAt"`
}

// EditRecord tracks edit history
type EditRecord struct {
	Content  string    `bson:"content" json:"content"`
	EditedAt time.Time `bson:"edited_at" json:"editedAt"`
	EditedBy string    `bson:"edited_by" json:"editedBy"`
}

// Reaction represents a user reaction to a comment
type Reaction struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID primitive.ObjectID `bson:"comment_id" json:"commentId"`
	UserID    string             `bson:"user_id" json:"userId"`
	Type      ReactionType       `bson:"type" json:"type"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
}

// Report represents a user report on a comment
type Report struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID   primitive.ObjectID `bson:"comment_id" json:"commentId"`
	ReporterID  string             `bson:"reporter_id" json:"reporterId"`
	Reason      string             `bson:"reason" json:"reason"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Status      string             `bson:"status" json:"status"` // pending, reviewed, dismissed
	ReviewedBy  string             `bson:"reviewed_by,omitempty" json:"reviewedBy,omitempty"`
	ReviewedAt  *time.Time         `bson:"reviewed_at,omitempty" json:"reviewedAt,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"createdAt"`
}

// CommentSettings represents tenant-specific comment settings
type CommentSettings struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TenantID            string             `bson:"tenant_id" json:"tenantId"`
	ResourceType        string             `bson:"resource_type" json:"resourceType"`
	RequireApproval     bool               `bson:"require_approval" json:"requireApproval"`
	AllowAnonymous      bool               `bson:"allow_anonymous" json:"allowAnonymous"`
	AllowReplies        bool               `bson:"allow_replies" json:"allowReplies"`
	MaxReplyDepth       int                `bson:"max_reply_depth" json:"maxReplyDepth"`
	AllowReactions      bool               `bson:"allow_reactions" json:"allowReactions"`
	AllowedReactions    []ReactionType     `bson:"allowed_reactions" json:"allowedReactions"`
	AllowAttachments    bool               `bson:"allow_attachments" json:"allowAttachments"`
	MaxAttachments      int                `bson:"max_attachments" json:"maxAttachments"`
	MaxCommentLength    int                `bson:"max_comment_length" json:"maxCommentLength"`
	CommentsEnabled     bool               `bson:"comments_enabled" json:"commentsEnabled"`
	NotifyOnNewComment  bool               `bson:"notify_on_new_comment" json:"notifyOnNewComment"`
	NotifyOnReply       bool               `bson:"notify_on_reply" json:"notifyOnReply"`
	AutoApproveVerified bool               `bson:"auto_approve_verified" json:"autoApproveVerified"`
	BadWordsFilter      bool               `bson:"bad_words_filter" json:"badWordsFilter"`
	CustomBadWords      []string           `bson:"custom_bad_words,omitempty" json:"customBadWords,omitempty"`
	CreatedAt           time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt           time.Time          `bson:"updated_at" json:"updatedAt"`
}
