//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/minisource/go-common/testing/e2e"
)

func TestComment_CRUDFlow(t *testing.T) {
	authURL := e2e.BaseURLFromEnv("AUTH_BASE_URL", "http://127.0.0.1:9001")
	token := e2e.LoginAuth(t, authURL, "admin@example.com", "AdminPass123!")
	h := e2e.Bearer(token)

	c := e2e.NewClient(e2e.BaseURLFromEnv("COMMENT_BASE_URL", "http://127.0.0.1:5010"), h)
	c.RequireUp(t, "/health")

	resourceID := fmt.Sprintf("e2e-crud-%d", time.Now().UnixNano())
	resp, body, err := c.Do(http.MethodPost, "/api/v1/comments", map[string]any{
		"tenantId": "default", "resourceType": "post", "resourceId": resourceID, "content": "e2e crud comment",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusTooManyRequests)
	if resp.StatusCode == http.StatusTooManyRequests {
		t.Skip("comment rate limit exceeded")
	}
	id := e2e.ExtractID(t, body)

	resp, body, err = c.Do(http.MethodGet, "/api/v1/comments/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK)

	resp, body, err = c.Do(http.MethodPut, "/api/v1/comments/"+id, map[string]any{
		"content": "updated e2e comment",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK)

	resp, body, err = c.Do(http.MethodPost, "/api/v1/comments/"+id+"/reactions", map[string]any{
		"type": "like",
	})
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusBadRequest)

	resp, body, err = c.Do(http.MethodGet, "/api/v1/comments/"+id+"/reactions/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusNotFound)

	resp, body, err = c.Do(http.MethodDelete, "/api/v1/comments/"+id, nil)
	if err != nil {
		t.Fatal(err)
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusNoContent)
}
