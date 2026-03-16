package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashrocket/bbcli/internal/api"
	"github.com/ashrocket/bbcli/internal/errors"
)

// mockAuth implements auth.Authenticator for testing.
type mockAuth struct{}

func (m *mockAuth) Authenticate(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer test-token")
	return nil
}
func (m *mockAuth) Source() string { return "test" }
func (m *mockAuth) Level() int    { return 0 }

func TestRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "42"})
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL)
	result, err := client.Request("GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]string
	json.Unmarshal(result, &parsed)
	if parsed["id"] != "42" {
		t.Errorf("id = %q, want 42", parsed["id"])
	}
}

func TestRequest401ReturnsAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL)
	_, err := client.Request("GET", "/test", nil)

	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.ExitCode != errors.ExitAuth {
		t.Errorf("ExitCode = %d, want %d (auth)", cliErr.ExitCode, errors.ExitAuth)
	}
}

func TestRequest404ReturnsNotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL)
	_, err := client.Request("GET", "/test", nil)

	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.ExitCode != errors.ExitNotFound {
		t.Errorf("ExitCode = %d, want %d (not found)", cliErr.ExitCode, errors.ExitNotFound)
	}
}

func TestRequest429RetriesAndReturnsRateLimitError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0") // don't actually wait in tests
		w.WriteHeader(429)
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL, api.WithNoRetryBackoff(true))
	_, err := client.Request("GET", "/test", nil)

	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.ExitCode != errors.ExitRateLimit {
		t.Errorf("ExitCode = %d, want %d (rate limit)", cliErr.ExitCode, errors.ExitRateLimit)
	}
	// Should have retried (1 original + 3 retries = 4 attempts)
	if attempts != 4 {
		t.Errorf("attempts = %d, want 4 (1 + 3 retries)", attempts)
	}
}

func TestRequest429WithNoRetryFlag(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(429)
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL, api.WithNoRetry(true))
	_, err := client.Request("GET", "/test", nil)

	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.ExitCode != errors.ExitRateLimit {
		t.Errorf("ExitCode = %d, want %d", cliErr.ExitCode, errors.ExitRateLimit)
	}
	if attempts != 1 {
		t.Errorf("with --no-retry, attempts = %d, want 1", attempts)
	}
}

func TestRequest500Retries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(500)
			w.Write([]byte("internal server error"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL, api.WithNoRetryBackoff(true))
	result, err := client.Request("GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}

	var parsed map[string]string
	json.Unmarshal(result, &parsed)
	if parsed["ok"] != "true" {
		t.Error("expected success after retry")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRequestBodyPreservedOnRetry(t *testing.T) {
	attempts := 0
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		bodies = append(bodies, string(body[:n]))
		if attempts == 1 {
			w.WriteHeader(429)
			w.Header().Set("Retry-After", "0")
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL, api.WithNoRetryBackoff(true))
	body := []byte(`{"title":"Fix bug"}`)
	_, err := client.Request("POST", "/test", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bodies) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(bodies))
	}
	// CRITICAL: body must be identical on retry (the bug we fixed)
	if bodies[0] != bodies[1] {
		t.Errorf("body changed on retry!\nattempt 1: %q\nattempt 2: %q", bodies[0], bodies[1])
	}
}

func TestPlaceholderSubstitution(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL, api.WithWorkspaceRepo("kureapp", "agents"))
	_, err := client.Request("GET", "/repositories/{workspace}/{repo_slug}/pullrequests", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/repositories/kureapp/agents/pullrequests"
	if receivedPath != want {
		t.Errorf("path = %q, want %q", receivedPath, want)
	}
}

func TestAuthHeaderInjected(t *testing.T) {
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := api.NewWithBaseURL(&mockAuth{}, server.URL)
	client.Request("GET", "/test", nil)

	if authHeader != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", authHeader, "Bearer test-token")
	}
}
