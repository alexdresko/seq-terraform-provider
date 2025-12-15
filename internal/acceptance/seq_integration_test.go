//go:build integration
// +build integration

package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const defaultServerURL = "http://seq:80"

type apiKeyResponse struct {
	ID          string   `json:"Id"`
	Title       string   `json:"Title"`
	Token       string   `json:"Token"`
	OwnerID     string   `json:"OwnerId"`
	Permissions []string `json:"Permissions"`
}

func TestSeqContainer_HealthAndAPIKeyCRUD(t *testing.T) {
	serverURL := firstNonEmpty(os.Getenv("SEQ_SERVER_URL"), defaultServerURL)
	apiKey := strings.TrimSpace(os.Getenv("SEQ_API_KEY"))
	if apiKey == "" {
		t.Skip("SEQ_API_KEY not set; export a Seq API key token to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}

	// 1) Health check (no auth expected).
	if err := doRequest(ctx, client, "", http.MethodGet, serverURL+"/health", nil, nil); err != nil {
		t.Fatalf("Seq /health check failed (is the devcontainer Seq running?): %v", err)
	}

	// Preflight: API keys require sufficient permission (typically "System").
	if err := doRequest(ctx, client, apiKey, http.MethodGet, serverURL+"/api/apikeys", nil, nil); err != nil {
		if strings.Contains(err.Error(), " 403 ") || strings.Contains(err.Error(), ": 403:") {
			t.Skipf("SEQ_API_KEY is not authorized for /api/apikeys (need System permission). Error: %v", err)
		}
		t.Fatalf("preflight /api/apikeys check failed: %v", err)
	}

	// 2) Create an API key.
	title := fmt.Sprintf("terraform-acceptance-%d", time.Now().UnixNano())
	createBody := map[string]any{
		"Title":       title,
		"Permissions": []string{"Read"},
	}

	var created apiKeyResponse
	if err := doRequest(ctx, client, apiKey, http.MethodPost, serverURL+"/api/apikeys", createBody, &created); err != nil {
		t.Fatalf("create api key failed: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("create api key: expected Id in response")
	}

	// Best-effort cleanup if anything fails after creation.
	defer func() {
		_ = doRequest(context.Background(), client, apiKey, http.MethodDelete, serverURL+"/api/apikeys/"+created.ID, nil, nil)
	}()

	// 3) Read it back.
	var got apiKeyResponse
	if err := doRequest(ctx, client, apiKey, http.MethodGet, serverURL+"/api/apikeys/"+created.ID, nil, &got); err != nil {
		t.Fatalf("read api key failed: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("read api key: expected Id %q, got %q", created.ID, got.ID)
	}
	if got.Title != title {
		t.Fatalf("read api key: expected Title %q, got %q", title, got.Title)
	}

	// 4) Delete it.
	if err := doRequest(ctx, client, apiKey, http.MethodDelete, serverURL+"/api/apikeys/"+created.ID, nil, nil); err != nil {
		t.Fatalf("delete api key failed: %v", err)
	}

	// 5) Confirm it's gone.
	err := doRequest(ctx, client, apiKey, http.MethodGet, serverURL+"/api/apikeys/"+created.ID, nil, &got)
	if err == nil {
		t.Fatalf("expected GET after delete to fail, but it succeeded")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected 404 after delete, got: %v", err)
	}
}

func doRequest(ctx context.Context, client *http.Client, apiKey, method, url string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("X-Seq-ApiKey", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("%s %s: %d: %s", method, url, resp.StatusCode, msg)
	}

	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
