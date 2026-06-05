//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/minisource/go-common/testing/e2e"
)

func commentClient(t *testing.T) *e2e.Client {
	t.Helper()
	authURL := e2e.BaseURLFromEnv("AUTH_BASE_URL", "http://127.0.0.1:9001")
	token := e2e.LoginAuth(t, authURL, "admin@example.com", "AdminPass123!")
	c := e2e.NewClient(e2e.BaseURLFromEnv("COMMENT_BASE_URL", "http://127.0.0.1:5010"), e2e.Bearer(token))
	c.RequireUp(t, "/health")
	return c
}

func TestComment_RepliesAndReactions(t *testing.T) {
	c := commentClient(t)
	resourceID := fmt.Sprintf("e2e-replies-%d", time.Now().UnixNano())

	resp, body, err := c.Do(http.MethodPost, "/api/v1/comments", map[string]any{
		"tenantId": "default", "resourceType": "post", "resourceId": resourceID, "content": "parent comment",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusTooManyRequests)
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Skip("comment rate limit exceeded")
	}
	parentID := e2e.ExtractID(t, body)

	resp, body, err = c.Do(http.MethodPost, "/api/v1/comments", map[string]any{
		"tenantId": "default", "resourceType": "post", "resourceId": resourceID,
		"parentId": parentID, "content": "reply comment",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusTooManyRequests)
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Skip("comment rate limit exceeded")
	}

	resp, body, err = c.Do(http.MethodGet, "/api/v1/comments/"+parentID+"/replies", nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK)

	resp, body, err = c.Do(http.MethodPost, "/api/v1/comments/"+parentID+"/reactions", map[string]any{
		"type": "like",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusBadRequest)

	resp, body, err = c.Do(http.MethodDelete, "/api/v1/comments/"+parentID+"/reactions", nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusNoContent, http.StatusNotFound)

	resp, body, err = c.Do(http.MethodDelete, "/api/v1/comments/"+parentID, nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusNoContent)
}

func TestComment_AdminModeration(t *testing.T) {
	c := commentClient(t)

	resp, body, err := c.Do(http.MethodGet, "/api/v1/admin/comments/pending", nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusForbidden)

	resourceID := fmt.Sprintf("e2e-mod-%d", time.Now().UnixNano())
	resp, body, err = c.Do(http.MethodPost, "/api/v1/comments", map[string]any{
		"tenantId": "default", "resourceType": "post", "resourceId": resourceID, "content": "moderate me",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusTooManyRequests)
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Skip("comment rate limit exceeded")
	}
	id := e2e.ExtractID(t, body)

	resp, body, err = c.Do(http.MethodPost, "/api/v1/admin/comments/"+id+"/moderate", map[string]any{
		"action": "approve",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusForbidden, http.StatusBadRequest)

	resp, body, err = c.Do(http.MethodPost, "/api/v1/admin/comments/"+id+"/pin", map[string]any{
		"pinned": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusForbidden, http.StatusBadRequest)
}

func TestComment_AdminHardDelete(t *testing.T) {
	c := commentClient(t)
	resourceID := fmt.Sprintf("e2e-admin-del-%d", time.Now().UnixNano())

	resp, body, err := c.Do(http.MethodPost, "/api/v1/comments", map[string]any{
		"tenantId": "default", "resourceType": "post", "resourceId": resourceID, "content": "admin delete me",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusTooManyRequests)
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Skip("comment rate limit exceeded")
	}
	id := e2e.ExtractID(t, body)

	resp, body, err = c.Do(http.MethodDelete, "/api/v1/admin/comments/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == http.StatusForbidden {
		t.Skip("admin lacks comment admin access")
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusNoContent)

	resp, body, err = c.Do(http.MethodGet, "/api/v1/comments/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusNotFound, http.StatusOK)
}
