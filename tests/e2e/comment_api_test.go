//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/minisource/go-common/testing/e2e"
)

func TestComment_API(t *testing.T) {
	c := e2e.NewClient(e2e.BaseURLFromEnv("COMMENT_BASE_URL", "http://127.0.0.1:5010"), nil)
	c.RequireUp(t, "/health")

	authURL := e2e.BaseURLFromEnv("AUTH_BASE_URL", "http://127.0.0.1:9001")
	token := e2e.LoginAuth(t, authURL, "admin@example.com", "AdminPass123!")
	h := e2e.Bearer(token)

	c.RunCases(t, []e2e.Case{
		{Name: "health", Method: http.MethodGet, Path: "/health", WantCode: []int{http.StatusOK}},
		{Name: "ready", Method: http.MethodGet, Path: "/ready", WantCode: []int{http.StatusOK}},
		{Name: "live", Method: http.MethodGet, Path: "/live", WantCode: []int{http.StatusOK}},
		{Name: "list", Method: http.MethodGet, Path: "/api/v1/comments?limit=5", Headers: h, WantCode: []int{http.StatusOK}},
		{Name: "stats", Method: http.MethodGet, Path: "/api/v1/comments/stats", Headers: h, WantCode: []int{http.StatusOK}},
		{Name: "search", Method: http.MethodGet, Path: "/api/v1/comments/search?q=e2e", Headers: h, WantCode: []int{http.StatusOK, http.StatusBadRequest}},
		{Name: "create", Method: http.MethodPost, Path: "/api/v1/comments", Headers: h, Body: map[string]any{
			"resourceType": "post", "resourceId": "e2e-entity-1", "tenantId": "default", "content": "e2e comment",
		}, WantCode: []int{http.StatusOK, http.StatusCreated, http.StatusBadRequest, http.StatusTooManyRequests}},
		{Name: "admin_pending", Method: http.MethodGet, Path: "/api/v1/admin/comments/pending", Headers: h, WantCode: []int{http.StatusOK, http.StatusForbidden}},
	})
}
