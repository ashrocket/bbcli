package errors_test

import (
	"encoding/json"
	"testing"

	"github.com/ashrocket/bbcli/internal/errors"
)

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      *errors.CLIError
		wantCode int
		wantStr  string
	}{
		{"auth missing", errors.NewAuthMissing(), errors.ExitAuth, "AUTH_MISSING"},
		{"auth failed", errors.NewAuthError("expired", "refresh"), errors.ExitAuth, "AUTH_FAILED"},
		{"not found", errors.NewNotFoundError("PR", 42), errors.ExitNotFound, "NOT_FOUND"},
		{"state error", errors.NewStateError("ALREADY_MERGED", "PR already merged", "check state"), errors.ExitState, "ALREADY_MERGED"},
		{"rate limited", errors.NewRateLimitError(30), errors.ExitRateLimit, "RATE_LIMITED"},
		{"network error", errors.NewNetworkError(nil), errors.ExitNetworkAPI, "NETWORK_ERROR"},
		{"api error", errors.NewAPIError(500, "internal"), errors.ExitNetworkAPI, "API_ERROR"},
		{"usage error", errors.NewUsageError("missing --title"), errors.ExitUsage, "USAGE_ERROR"},
		{"general error", errors.NewGeneralError("unexpected"), errors.ExitGeneral, "GENERAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.ExitCode != tt.wantCode {
				t.Errorf("ExitCode = %d, want %d", tt.err.ExitCode, tt.wantCode)
			}
			if tt.err.Code != tt.wantStr {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantStr)
			}
		})
	}
}

func TestRetryableFlag(t *testing.T) {
	retryable := []struct {
		name string
		err  *errors.CLIError
	}{
		{"rate limited", errors.NewRateLimitError(10)},
		{"network error", errors.NewNetworkError(nil)},
		{"api error", errors.NewAPIError(503, "unavailable")},
	}
	for _, tt := range retryable {
		t.Run(tt.name+" should be retryable", func(t *testing.T) {
			if !tt.err.Retryable {
				t.Errorf("%s: Retryable = false, want true", tt.name)
			}
		})
	}

	notRetryable := []struct {
		name string
		err  *errors.CLIError
	}{
		{"auth", errors.NewAuthError("bad token", "fix it")},
		{"not found", errors.NewNotFoundError("PR", 99)},
		{"state", errors.NewStateError("X", "msg", "fix")},
		{"usage", errors.NewUsageError("bad input")},
	}
	for _, tt := range notRetryable {
		t.Run(tt.name+" should not be retryable", func(t *testing.T) {
			if tt.err.Retryable {
				t.Errorf("%s: Retryable = true, want false", tt.name)
			}
		})
	}
}

func TestErrorInterface(t *testing.T) {
	err := errors.NewAuthMissing()
	var e error = err // must satisfy error interface
	if e.Error() == "" {
		t.Error("Error() should not be empty")
	}
}

func TestErrorEnvelopeJSON(t *testing.T) {
	cliErr := errors.NewRateLimitError(30)
	envelope := errors.ErrorEnvelope{Error: cliErr}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed struct {
		Error struct {
			Code       string `json:"code"`
			Message    string `json:"message"`
			Suggestion string `json:"suggestion"`
			Retryable  bool   `json:"retryable"`
			Details    any    `json:"details"`
		} `json:"error"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed.Error.Code != "RATE_LIMITED" {
		t.Errorf("Code = %q, want RATE_LIMITED", parsed.Error.Code)
	}
	if !parsed.Error.Retryable {
		t.Error("Retryable should be true")
	}
	if parsed.Error.Suggestion == "" {
		t.Error("Suggestion should not be empty")
	}
}

func TestExitCodeIsNotInJSON(t *testing.T) {
	err := errors.NewAuthMissing()
	data, _ := json.Marshal(err)
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if _, exists := raw["exit_code"]; exists {
		t.Error("exit_code should be excluded from JSON (json:\"-\")")
	}
}

func TestRateLimitErrorDetails(t *testing.T) {
	err := errors.NewRateLimitError(120)
	data, _ := json.Marshal(err)
	var raw map[string]any
	json.Unmarshal(data, &raw)
	details, ok := raw["details"].(map[string]any)
	if !ok {
		t.Fatal("details should be a map")
	}
	if details["retry_after_seconds"] != float64(120) {
		t.Errorf("retry_after_seconds = %v, want 120", details["retry_after_seconds"])
	}
}
