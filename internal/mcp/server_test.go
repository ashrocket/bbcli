package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ashrocket/bbcli/internal/api"
	bbmcp "github.com/ashrocket/bbcli/internal/mcp"
	"github.com/mark3labs/mcp-go/mcp"
)

// mockAuth implements api.Authenticator for testing.
type mockAuth struct{}

func (m *mockAuth) Authenticate(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer test-token")
	return nil
}
func (m *mockAuth) Source() string { return "test" }
func (m *mockAuth) Level() int    { return 0 }

// newTestServer creates a bbmcp Server backed by a mock HTTP server.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*bbmcp.Server, func()) {
	t.Helper()
	ts := httptest.NewServer(handler)
	client := api.NewWithBaseURL(&mockAuth{}, ts.URL,
		api.WithWorkspaceRepo("testws", "testrepo"),
	)
	srv := bbmcp.New(client, bbmcp.WithDefaults("testws", "testrepo"))
	return srv, ts.Close
}

// makePRListRequest builds a CallToolRequest for bitbucket_pr_list.
func makePRListRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "bitbucket_pr_list",
			Arguments: args,
		},
	}
}

func TestPRListReturnsResults(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path contains the correct workspace/repo.
		if !strings.Contains(r.URL.Path, "/repositories/testws/testrepo/pullrequests") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"id":    1,
					"title": "Fix login bug",
					"state": "OPEN",
					"author": map[string]string{
						"display_name": "Alice",
					},
					"source": map[string]any{
						"branch": map[string]string{"name": "fix/login"},
					},
					"destination": map[string]any{
						"branch": map[string]string{"name": "main"},
					},
					"created_on": "2026-03-15T10:00:00Z",
					"updated_on": "2026-03-15T11:00:00Z",
				},
				{
					"id":    2,
					"title": "Add dashboard widget",
					"state": "OPEN",
					"author": map[string]string{
						"display_name": "Bob",
					},
					"source": map[string]any{
						"branch": map[string]string{"name": "feature/widget"},
					},
					"destination": map[string]any{
						"branch": map[string]string{"name": "main"},
					},
					"created_on": "2026-03-14T10:00:00Z",
					"updated_on": "2026-03-14T12:00:00Z",
				},
			},
		})
	})

	srv, cleanup := newTestServer(t, handler)
	defer cleanup()

	req := makePRListRequest(map[string]any{
		"state": "OPEN",
		"limit": float64(10),
	})

	result, err := srv.HandlePRList(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	// Parse the text content to verify structure.
	text := mcp.GetTextFromContent(result.Content[0])
	var prs []map[string]any
	if err := json.Unmarshal([]byte(text), &prs); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}
	if len(prs) != 2 {
		t.Errorf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0]["title"] != "Fix login bug" {
		t.Errorf("first PR title = %v, want 'Fix login bug'", prs[0]["title"])
	}
}

func TestPRListMissingWorkspace(t *testing.T) {
	client := api.NewWithBaseURL(&mockAuth{}, "http://unused",
		api.WithWorkspaceRepo("", ""),
	)
	srv := bbmcp.New(client) // no defaults

	req := makePRListRequest(map[string]any{
		"repo": "somerepo",
	})

	result, err := srv.HandlePRList(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for missing workspace")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if !strings.Contains(text, "workspace is required") {
		t.Errorf("error text = %q, want it to mention 'workspace is required'", text)
	}
}

func TestPRListMissingRepo(t *testing.T) {
	client := api.NewWithBaseURL(&mockAuth{}, "http://unused")
	srv := bbmcp.New(client, bbmcp.WithDefaults("myws", ""))

	req := makePRListRequest(map[string]any{
		"workspace": "myws",
	})

	result, err := srv.HandlePRList(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for missing repo")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if !strings.Contains(text, "repo is required") {
		t.Errorf("error text = %q, want it to mention 'repo is required'", text)
	}
}

func TestPRListWithParamOverrides(t *testing.T) {
	var receivedPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{},
		})
	})

	srv, cleanup := newTestServer(t, handler)
	defer cleanup()

	// Override workspace and repo via tool params.
	req := makePRListRequest(map[string]any{
		"workspace": "override-ws",
		"repo":      "override-repo",
		"state":     "MERGED",
	})

	result, err := srv.HandlePRList(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	// The path should use the overridden workspace/repo.
	if !strings.Contains(receivedPath, "/repositories/override-ws/override-repo/") {
		t.Errorf("path = %q, expected overridden workspace/repo", receivedPath)
	}
}

func TestPRListAPIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error": "unauthorized"}`))
	})

	srv, cleanup := newTestServer(t, handler)
	defer cleanup()

	req := makePRListRequest(map[string]any{})

	result, err := srv.HandlePRList(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for 401 response")
	}
	text := mcp.GetTextFromContent(result.Content[0])
	if !strings.Contains(text, "Authentication failed") {
		t.Errorf("error text = %q, want mention of authentication failure", text)
	}
}

func TestPRListEmptyResults(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{},
		})
	})

	srv, cleanup := newTestServer(t, handler)
	defer cleanup()

	req := makePRListRequest(map[string]any{})

	result, err := srv.HandlePRList(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}

	text := mcp.GetTextFromContent(result.Content[0])
	// Empty array should still be valid JSON.
	if text != "null" && text != "[]" {
		var prs []any
		if err := json.Unmarshal([]byte(text), &prs); err != nil {
			t.Fatalf("result is not valid JSON: %v\ntext: %s", err, text)
		}
		if len(prs) != 0 {
			t.Errorf("expected 0 PRs, got %d", len(prs))
		}
	}
}
