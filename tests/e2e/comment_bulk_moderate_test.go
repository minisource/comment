//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/minisource/go-common/testing/e2e"
)

func TestComment_BulkModerate(t *testing.T) {
	c := commentClient(t)
	resourceID := fmt.Sprintf("e2e-bulk-mod-%d", time.Now().UnixNano())

	var ids []string
	for i := 0; i < 2; i++ {
		resp, body, err := c.Do(http.MethodPost, "/api/v1/comments", map[string]any{
			"tenantId": "default", "resourceType": "post", "resourceId": resourceID,
			"content": fmt.Sprintf("bulk moderate comment %d", i),
		})
		if err != nil {
			t.Fatal(err)
		}
		e2e.ExpectStatus(t, resp, body, http.StatusOK, http.StatusCreated, http.StatusTooManyRequests)
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Skip("comment rate limit exceeded")
		}
		ids = append(ids, e2e.ExtractID(t, body))
	}

	resp, body, err := c.Do(http.MethodPost, "/api/v1/admin/comments/bulk-moderate", map[string]any{
		"comment_ids": ids, "status": "approved",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == http.StatusForbidden {
		t.Skip("admin lacks comment moderation access")
	}
	e2e.ExpectStatus(t, resp, body, http.StatusOK)

	var parsed map[string]any
	e2e.ParseJSON(t, body, &parsed)
	data, _ := parsed["data"].(map[string]any)
	if data == nil {
		data = parsed
	}
	sc, _ := data["success_count"].(float64)
	if sc == 0 {
		sc, _ = data["successCount"].(float64)
	}
	if int(sc) < 1 {
		t.Fatalf("expected bulk moderation success, body: %s", string(body))
	}
}
