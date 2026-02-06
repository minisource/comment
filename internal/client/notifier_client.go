package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// NotifierClient implements the NotifierClient interface
type NotifierClient struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// NewNotifierClient creates a new notifier client
func NewNotifierClient(baseURL string, enabled bool) *NotifierClient {
	return &NotifierClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: enabled,
	}
}

// NotificationRequest represents a notification request
type NotificationRequest struct {
	Type       string                 `json:"type"`
	Recipients []string               `json:"recipients"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Channels   []string               `json:"channels,omitempty"`
}

// SendNotification sends a notification
func (c *NotifierClient) SendNotification(ctx context.Context, notificationType, title, message string, data map[string]interface{}) error {
	if !c.enabled {
		return nil
	}

	req := NotificationRequest{
		Type:     notificationType,
		Title:    title,
		Message:  message,
		Data:     data,
		Channels: []string{"push", "email"},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/notifications", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notification service returned status %d", resp.StatusCode)
	}

	return nil
}

// SendNewCommentNotification sends notification for new comment
func (c *NotifierClient) SendNewCommentNotification(ctx context.Context, commentID, resourceType, resourceID, authorName string) error {
	return c.SendNotification(ctx, "new_comment",
		"New Comment",
		fmt.Sprintf("New comment by %s on %s", authorName, resourceType),
		map[string]interface{}{
			"comment_id":    commentID,
			"resource_type": resourceType,
			"resource_id":   resourceID,
			"author_name":   authorName,
		},
	)
}

// SendCommentApprovedNotification sends notification when comment is approved
func (c *NotifierClient) SendCommentApprovedNotification(ctx context.Context, commentID, userID string) error {
	return c.SendNotification(ctx, "comment_approved",
		"Comment Approved",
		"Your comment has been approved",
		map[string]interface{}{
			"comment_id": commentID,
			"user_id":    userID,
		},
	)
}

// SendCommentRejectedNotification sends notification when comment is rejected
func (c *NotifierClient) SendCommentRejectedNotification(ctx context.Context, commentID, userID, reason string) error {
	return c.SendNotification(ctx, "comment_rejected",
		"Comment Rejected",
		fmt.Sprintf("Your comment was rejected: %s", reason),
		map[string]interface{}{
			"comment_id": commentID,
			"user_id":    userID,
			"reason":     reason,
		},
	)
}

// SendReplyNotification sends notification when someone replies
func (c *NotifierClient) SendReplyNotification(ctx context.Context, commentID, parentAuthorID, replyAuthorName string) error {
	return c.SendNotification(ctx, "comment_reply",
		"New Reply",
		fmt.Sprintf("%s replied to your comment", replyAuthorName),
		map[string]interface{}{
			"comment_id":   commentID,
			"recipient_id": parentAuthorID,
			"author_name":  replyAuthorName,
		},
	)
}
