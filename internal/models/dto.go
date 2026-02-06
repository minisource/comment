package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// CreateCommentRequest represents the request to create a new comment
type CreateCommentRequest struct {
	TenantID     string         `json:"tenantId" validate:"required"`
	ResourceType string         `json:"resourceType" validate:"required"`
	ResourceID   string         `json:"resourceId" validate:"required"`
	ParentID     string         `json:"parentId,omitempty"`
	Content      string         `json:"content" validate:"required,min=1,max=5000"`
	AuthorName   string         `json:"authorName,omitempty"`
	IsAnonymous  bool           `json:"isAnonymous,omitempty"`
	Attachments  []Attachment   `json:"attachments,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// UpdateCommentRequest represents the request to update a comment
type UpdateCommentRequest struct {
	Content     string       `json:"content" validate:"required,min=1,max=5000"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// ModerateCommentRequest represents the request to moderate a comment
type ModerateCommentRequest struct {
	Status          CommentStatus `json:"status" validate:"required,oneof=approved rejected spam"`
	RejectionReason string        `json:"rejectionReason,omitempty"`
}

// PinCommentRequest represents the request to pin/unpin a comment
type PinCommentRequest struct {
	IsPinned bool `json:"isPinned"`
}

// ReactionRequest represents the request to add/update a reaction
type ReactionRequest struct {
	Type ReactionType `json:"type" validate:"required,oneof=like dislike love haha wow sad angry"`
}

// ReportRequest represents the request to report a comment
type ReportRequest struct {
	Reason      string `json:"reason" validate:"required,oneof=spam inappropriate harassment hate_speech misinformation other"`
	Description string `json:"description,omitempty" validate:"max=500"`
}

// ListCommentsRequest represents query parameters for listing comments
type ListCommentsRequest struct {
	TenantID       string        `query:"tenantId"`
	ResourceType   string        `query:"resourceType"`
	ResourceID     string        `query:"resourceId"`
	ParentID       string        `query:"parentId"`
	Status         CommentStatus `query:"status"`
	AuthorID       string        `query:"authorId"`
	IsPinned       *bool         `query:"isPinned"`
	SortBy         string        `query:"sortBy"`    // created_at, like_count, reply_count
	SortOrder      string        `query:"sortOrder"` // asc, desc
	Page           int           `query:"page"`
	PageSize       int           `query:"pageSize"`
	IncludeDeleted bool          `query:"includeDeleted"`
}

// ListCommentsResponse represents paginated comments response
type ListCommentsResponse struct {
	Comments   []*Comment `json:"comments"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalPages int        `json:"totalPages"`
}

// CommentWithReplies represents a comment with its replies
type CommentWithReplies struct {
	Comment *Comment              `json:"comment"`
	Replies []*CommentWithReplies `json:"replies,omitempty"`
}

// CommentStats represents statistics for a resource
type CommentStats struct {
	TotalComments     int64            `json:"totalComments"`
	ApprovedCount     int64            `json:"approvedCount"`
	PendingCount      int64            `json:"pendingCount"`
	RejectedCount     int64            `json:"rejectedCount"`
	TotalReactions    int64            `json:"totalReactions"`
	AverageRating     float64          `json:"averageRating,omitempty"`
	ReactionBreakdown map[string]int64 `json:"reactionBreakdown,omitempty"`
}

// PendingModeration represents comments pending moderation
type PendingModeration struct {
	Comments []*Comment `json:"comments"`
	Total    int64      `json:"total"`
}

// UserReaction represents the current user's reaction to a comment
type UserReaction struct {
	CommentID primitive.ObjectID `json:"commentId"`
	Type      *ReactionType      `json:"type"` // nil if no reaction
}

// SettingsRequest represents request to update tenant settings
type SettingsRequest struct {
	RequireApproval     *bool          `json:"requireApproval,omitempty"`
	AllowAnonymous      *bool          `json:"allowAnonymous,omitempty"`
	AllowReplies        *bool          `json:"allowReplies,omitempty"`
	MaxReplyDepth       *int           `json:"maxReplyDepth,omitempty"`
	AllowReactions      *bool          `json:"allowReactions,omitempty"`
	AllowedReactions    []ReactionType `json:"allowedReactions,omitempty"`
	AllowAttachments    *bool          `json:"allowAttachments,omitempty"`
	MaxAttachments      *int           `json:"maxAttachments,omitempty"`
	MaxCommentLength    *int           `json:"maxCommentLength,omitempty"`
	CommentsEnabled     *bool          `json:"commentsEnabled,omitempty"`
	NotifyOnNewComment  *bool          `json:"notifyOnNewComment,omitempty"`
	NotifyOnReply       *bool          `json:"notifyOnReply,omitempty"`
	AutoApproveVerified *bool          `json:"autoApproveVerified,omitempty"`
	BadWordsFilter      *bool          `json:"badWordsFilter,omitempty"`
	CustomBadWords      []string       `json:"customBadWords,omitempty"`
}
