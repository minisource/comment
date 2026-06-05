package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/comment/config"
	"github.com/minisource/comment/internal/usecase"
	"github.com/minisource/go-sdk/auth"
)

// ServiceNotifier sends notifications via Notifier service routes (service auth).
type ServiceNotifier struct {
	baseURL     string
	enabled     bool
	adminUserID string
	authClient  *auth.Client
	httpClient  *http.Client
}

// NewServiceNotifier creates a notifier client for service-to-service calls.
func NewServiceNotifier(cfg *config.Config) usecase.NotifierClient {
	if !cfg.Notifier.Enabled {
		return &ServiceNotifier{enabled: false}
	}
	return &ServiceNotifier{
		baseURL:     cfg.Notifier.ServiceURL,
		enabled:     true,
		adminUserID: cfg.Notifier.AdminUserID,
		authClient: auth.NewClient(auth.ClientConfig{
			BaseURL:      cfg.Auth.ServiceURL,
			ClientID:     cfg.Notifier.ClientID,
			ClientSecret: cfg.Notifier.ClientSecret,
			AutoRefresh:  true,
		}),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *ServiceNotifier) SendNotification(ctx context.Context, n usecase.NotificationRequest) error {
	if !s.enabled {
		return nil
	}

	userID := s.adminUserID
	for _, r := range n.Recipients {
		if _, err := uuid.Parse(r); err == nil {
			userID = r
			break
		}
	}
	if userID == "" {
		return nil
	}

	token, err := s.authClient.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("notifier auth token: %w", err)
	}

	body, err := json.Marshal(map[string]any{
		"userId":   userID,
		"type":     "in_app",
		"body":     n.Body,
		"subject":  n.Title,
		"priority": "normal",
		"metadata": n.Data,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/v1/service/notifications", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notifier request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("notifier status %d", resp.StatusCode)
	}
	return nil
}
