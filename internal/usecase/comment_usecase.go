package usecase

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/minisource/comment/config"
	"github.com/minisource/comment/internal/models"
	"github.com/minisource/comment/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CommentUsecase handles comment business logic
type CommentUsecase struct {
	commentRepo   *repository.CommentRepository
	reactionRepo  *repository.ReactionRepository
	reportRepo    *repository.ReportRepository
	settingsRepo  *repository.SettingsRepository
	notifier      NotifierClient
	cfg           *config.Config
	badWordsRegex *regexp.Regexp
}

// NotifierClient interface for sending notifications
type NotifierClient interface {
	SendNotification(ctx context.Context, notification NotificationRequest) error
}

// NotificationRequest represents a notification to send
type NotificationRequest struct {
	Type       string            `json:"type"`
	Recipients []string          `json:"recipients"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Data       map[string]string `json:"data"`
}

// NewCommentUsecase creates a new comment usecase
func NewCommentUsecase(
	commentRepo *repository.CommentRepository,
	reactionRepo *repository.ReactionRepository,
	reportRepo *repository.ReportRepository,
	settingsRepo *repository.SettingsRepository,
	notifier NotifierClient,
	cfg *config.Config,
) *CommentUsecase {
	// Build bad words regex
	var badWordsRegex *regexp.Regexp
	if cfg.Moderation.BadWordsEnabled && len(cfg.Moderation.BadWordsList) > 0 {
		pattern := "(?i)\\b(" + strings.Join(cfg.Moderation.BadWordsList, "|") + ")\\b"
		badWordsRegex, _ = regexp.Compile(pattern)
	}

	return &CommentUsecase{
		commentRepo:   commentRepo,
		reactionRepo:  reactionRepo,
		reportRepo:    reportRepo,
		settingsRepo:  settingsRepo,
		notifier:      notifier,
		cfg:           cfg,
		badWordsRegex: badWordsRegex,
	}
}

// CreateComment creates a new comment
func (u *CommentUsecase) CreateComment(ctx context.Context, req models.CreateCommentRequest, authorID, authorName, authorEmail, ipAddress, userAgent string) (*models.Comment, error) {
	// Get settings
	settings, err := u.settingsRepo.GetOrCreate(ctx, req.TenantID, req.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Check if comments are enabled
	if !settings.CommentsEnabled {
		return nil, fmt.Errorf("comments are disabled for this resource type")
	}

	// Check anonymous permissions
	if req.IsAnonymous && !settings.AllowAnonymous {
		return nil, fmt.Errorf("anonymous comments are not allowed")
	}

	// Validate content length
	if len(req.Content) > settings.MaxCommentLength {
		return nil, fmt.Errorf("comment exceeds maximum length of %d characters", settings.MaxCommentLength)
	}

	// Check for parent comment (reply)
	var parentID *primitive.ObjectID
	var rootID *primitive.ObjectID
	depth := 0

	if req.ParentID != "" {
		pid, err := primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent ID")
		}

		parent, err := u.commentRepo.GetByID(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent comment: %w", err)
		}
		if parent == nil {
			return nil, fmt.Errorf("parent comment not found")
		}

		// Check if replies are allowed
		if !settings.AllowReplies {
			return nil, fmt.Errorf("replies are not allowed")
		}

		// Check max reply depth
		depth = parent.Depth + 1
		if depth > settings.MaxReplyDepth {
			return nil, fmt.Errorf("maximum reply depth exceeded")
		}

		parentID = &pid
		if parent.RootID != nil {
			rootID = parent.RootID
		} else {
			rootID = &pid
		}
	}

	// Check for bad words
	flaggedWords := u.checkBadWords(req.Content, settings.CustomBadWords)

	// Determine initial status
	status := models.StatusPending
	if !settings.RequireApproval {
		status = models.StatusApproved
	} else if len(flaggedWords) > 0 {
		status = models.StatusPending // Force pending if bad words detected
	}

	// Set author info
	displayName := authorName
	if req.AuthorName != "" {
		displayName = req.AuthorName
	}
	if req.IsAnonymous {
		displayName = "Anonymous"
		authorEmail = ""
	}

	comment := &models.Comment{
		TenantID:     req.TenantID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		ParentID:     parentID,
		RootID:       rootID,
		AuthorID:     authorID,
		AuthorName:   displayName,
		AuthorEmail:  authorEmail,
		IsAnonymous:  req.IsAnonymous,
		Content:      req.Content,
		Attachments:  req.Attachments,
		Status:       status,
		FlaggedWords: flaggedWords,
		IsPinned:     false,
		IsEdited:     false,
		ReplyCount:   0,
		LikeCount:    0,
		DislikeCount: 0,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Metadata:     req.Metadata,
		Depth:        depth,
		IsDeleted:    false,
	}

	if err := u.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// Increment parent reply count
	if parentID != nil {
		if err := u.commentRepo.IncrementReplyCount(ctx, *parentID, 1); err != nil {
			log.Printf("Failed to increment reply count: %v", err)
		}
	}

	// Send notifications
	go u.sendNewCommentNotification(comment, settings)

	return comment, nil
}

// GetComment retrieves a comment by ID
func (u *CommentUsecase) GetComment(ctx context.Context, id string) (*models.Comment, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid comment ID")
	}

	comment, err := u.commentRepo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, fmt.Errorf("comment not found")
	}

	return comment, nil
}

// UpdateComment updates a comment
func (u *CommentUsecase) UpdateComment(ctx context.Context, id string, req models.UpdateCommentRequest, userID string, isAdmin bool) (*models.Comment, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid comment ID")
	}

	comment, err := u.commentRepo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, fmt.Errorf("comment not found")
	}

	// Check ownership
	if comment.AuthorID != userID && !isAdmin {
		return nil, fmt.Errorf("you can only edit your own comments")
	}

	// Check if deleted
	if comment.IsDeleted {
		return nil, fmt.Errorf("cannot edit deleted comment")
	}

	// Get settings
	settings, err := u.settingsRepo.GetOrCreate(ctx, comment.TenantID, comment.ResourceType)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Validate content length
	if len(req.Content) > settings.MaxCommentLength {
		return nil, fmt.Errorf("comment exceeds maximum length of %d characters", settings.MaxCommentLength)
	}

	// Save edit history
	editRecord := models.EditRecord{
		Content:  comment.Content,
		EditedAt: time.Now(),
		EditedBy: userID,
	}
	comment.EditHistory = append(comment.EditHistory, editRecord)

	// Check for bad words in new content
	flaggedWords := u.checkBadWords(req.Content, settings.CustomBadWords)

	// Update fields
	comment.Content = req.Content
	comment.Attachments = req.Attachments
	comment.IsEdited = true
	comment.FlaggedWords = flaggedWords

	// If bad words found, set back to pending
	if len(flaggedWords) > 0 && settings.RequireApproval {
		comment.Status = models.StatusPending
	}

	if err := u.commentRepo.Update(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return comment, nil
}

// DeleteComment soft deletes a comment
func (u *CommentUsecase) DeleteComment(ctx context.Context, id string, userID string, isAdmin bool) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid comment ID")
	}

	comment, err := u.commentRepo.GetByID(ctx, oid)
	if err != nil {
		return err
	}
	if comment == nil {
		return fmt.Errorf("comment not found")
	}

	// Check ownership
	if comment.AuthorID != userID && !isAdmin {
		return fmt.Errorf("you can only delete your own comments")
	}

	if err := u.commentRepo.SoftDelete(ctx, oid, userID); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	// Decrement parent reply count
	if comment.ParentID != nil {
		if err := u.commentRepo.IncrementReplyCount(ctx, *comment.ParentID, -1); err != nil {
			log.Printf("Failed to decrement reply count: %v", err)
		}
	}

	return nil
}

// ListComments retrieves comments with filters
func (u *CommentUsecase) ListComments(ctx context.Context, req models.ListCommentsRequest, userID string, isAdmin bool) (*models.ListCommentsResponse, error) {
	// Non-admins can only see approved comments
	if !isAdmin && req.Status == "" {
		req.Status = models.StatusApproved
	}

	comments, total, err := u.commentRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &models.ListCommentsResponse{
		Comments:   comments,
		Total:      total,
		Page:       req.Page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// GetReplies retrieves replies for a comment
func (u *CommentUsecase) GetReplies(ctx context.Context, commentID string, page, pageSize int) ([]*models.Comment, int64, error) {
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid comment ID")
	}

	return u.commentRepo.GetReplies(ctx, oid, page, pageSize)
}

// ModerateComment approves or rejects a comment
func (u *CommentUsecase) ModerateComment(ctx context.Context, id string, req models.ModerateCommentRequest, moderatorID string) (*models.Comment, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid comment ID")
	}

	comment, err := u.commentRepo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, fmt.Errorf("comment not found")
	}

	now := time.Now()
	comment.Status = req.Status
	comment.ModeratedBy = moderatorID
	comment.ModeratedAt = &now

	if req.Status == models.StatusRejected {
		comment.RejectionReason = req.RejectionReason
	}

	if err := u.commentRepo.Update(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to moderate comment: %w", err)
	}

	// Send notification to author
	go u.sendModerationNotification(comment)

	return comment, nil
}

// PinComment pins or unpins a comment
func (u *CommentUsecase) PinComment(ctx context.Context, id string, isPinned bool, userID string) (*models.Comment, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid comment ID")
	}

	comment, err := u.commentRepo.GetByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, fmt.Errorf("comment not found")
	}

	now := time.Now()
	comment.IsPinned = isPinned
	if isPinned {
		comment.PinnedBy = userID
		comment.PinnedAt = &now
	} else {
		comment.PinnedBy = ""
		comment.PinnedAt = nil
	}

	if err := u.commentRepo.Update(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to pin comment: %w", err)
	}

	return comment, nil
}

// GetPendingComments retrieves comments pending moderation
func (u *CommentUsecase) GetPendingComments(ctx context.Context, tenantID string, page, pageSize int) ([]*models.Comment, int64, error) {
	return u.commentRepo.GetPending(ctx, tenantID, page, pageSize)
}

// GetCommentStats retrieves comment statistics
func (u *CommentUsecase) GetCommentStats(ctx context.Context, tenantID, resourceType, resourceID string) (*models.CommentStats, error) {
	return u.commentRepo.GetStats(ctx, tenantID, resourceType, resourceID)
}

// SearchComments searches comments
func (u *CommentUsecase) SearchComments(ctx context.Context, tenantID, query string, page, pageSize int) ([]*models.Comment, int64, error) {
	return u.commentRepo.Search(ctx, tenantID, query, page, pageSize)
}

// checkBadWords checks content for bad words
func (u *CommentUsecase) checkBadWords(content string, customBadWords []string) []string {
	var flagged []string

	// Check with default regex
	if u.badWordsRegex != nil {
		matches := u.badWordsRegex.FindAllString(content, -1)
		flagged = append(flagged, matches...)
	}

	// Check custom bad words
	if len(customBadWords) > 0 {
		pattern := "(?i)\\b(" + strings.Join(customBadWords, "|") + ")\\b"
		if customRegex, err := regexp.Compile(pattern); err == nil {
			matches := customRegex.FindAllString(content, -1)
			flagged = append(flagged, matches...)
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	unique := []string{}
	for _, word := range flagged {
		lower := strings.ToLower(word)
		if !seen[lower] {
			seen[lower] = true
			unique = append(unique, word)
		}
	}

	return unique
}

// sendNewCommentNotification sends notification for new comments
func (u *CommentUsecase) sendNewCommentNotification(comment *models.Comment, settings *models.CommentSettings) {
	if u.notifier == nil || !u.cfg.Notifier.Enabled {
		return
	}

	if !settings.NotifyOnNewComment && comment.ParentID == nil {
		return
	}
	if !settings.NotifyOnReply && comment.ParentID != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	notificationType := "comment.new"
	if comment.ParentID != nil {
		notificationType = "comment.reply"
	}

	title := "New Comment"
	if comment.ParentID != nil {
		title = "New Reply to Your Comment"
	}
	if comment.Status == models.StatusPending {
		title = "Comment Pending Approval"
		notificationType = "comment.pending"
	}

	notification := NotificationRequest{
		Type:       notificationType,
		Recipients: []string{"admin"}, // Will be replaced with actual admin IDs
		Title:      title,
		Body:       truncateString(comment.Content, 100),
		Data: map[string]string{
			"comment_id":    comment.ID.Hex(),
			"tenant_id":     comment.TenantID,
			"resource_type": comment.ResourceType,
			"resource_id":   comment.ResourceID,
			"author_id":     comment.AuthorID,
			"status":        string(comment.Status),
		},
	}

	if err := u.notifier.SendNotification(ctx, notification); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}

// sendModerationNotification sends notification when comment is moderated
func (u *CommentUsecase) sendModerationNotification(comment *models.Comment) {
	if u.notifier == nil || !u.cfg.Notifier.Enabled {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	title := "Your Comment Was Approved"
	body := "Your comment has been approved and is now visible."
	if comment.Status == models.StatusRejected {
		title = "Your Comment Was Rejected"
		body = "Your comment has been rejected."
		if comment.RejectionReason != "" {
			body += " Reason: " + comment.RejectionReason
		}
	}

	notification := NotificationRequest{
		Type:       "comment.moderated",
		Recipients: []string{comment.AuthorID},
		Title:      title,
		Body:       body,
		Data: map[string]string{
			"comment_id": comment.ID.Hex(),
			"status":     string(comment.Status),
		},
	}

	if err := u.notifier.SendNotification(ctx, notification); err != nil {
		log.Printf("Failed to send moderation notification: %v", err)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
