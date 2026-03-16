// Package errors defines typed errors with exit codes for bbcli.
// Every error carries an exit code, machine-readable code string,
// human message, suggestion, and retryable flag.
package errors

import "fmt"

// Exit codes — each maps to a distinct agent recovery action.
const (
	ExitSuccess    = 0 // Operation completed
	ExitGeneral    = 1 // Unexpected failure
	ExitUsage      = 2 // Bad flags, missing required args
	ExitAuth       = 3 // Token missing, expired, invalid
	ExitNotFound   = 4 // Resource does not exist
	ExitState      = 5 // State prevents action (declined PR, protected branch)
	ExitRateLimit  = 6 // API rate limit exceeded after retries
	ExitNetworkAPI = 7 // Cannot reach Bitbucket or 5xx
)

// CLIError is the base error type. All errors returned by internal packages
// should be this type so the root command can extract the exit code.
type CLIError struct {
	ExitCode   int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Retryable  bool   `json:"retryable"`
	Details    any    `json:"details,omitempty"`
}

func (e *CLIError) Error() string {
	return e.Message
}

// ErrorEnvelope is the JSON structure written to stderr on error.
type ErrorEnvelope struct {
	Error *CLIError `json:"error"`
}

// Constructors for each error type.

func NewAuthError(message, suggestion string) *CLIError {
	return &CLIError{
		ExitCode:   ExitAuth,
		Code:       "AUTH_FAILED",
		Message:    message,
		Suggestion: suggestion,
		Retryable:  false,
	}
}

func NewAuthMissing() *CLIError {
	return &CLIError{
		ExitCode:   ExitAuth,
		Code:       "AUTH_MISSING",
		Message:    "No authentication token found",
		Suggestion: "Run 'bbcli auth login' or set BBCLI_TOKEN environment variable",
		Retryable:  false,
	}
}

func NewNotFoundError(resource string, id any) *CLIError {
	return &CLIError{
		ExitCode:   ExitNotFound,
		Code:       "NOT_FOUND",
		Message:    fmt.Sprintf("%s not found: %v", resource, id),
		Suggestion: fmt.Sprintf("Check that the %s exists and you have access", resource),
		Retryable:  false,
	}
}

func NewStateError(code, message, suggestion string) *CLIError {
	return &CLIError{
		ExitCode:   ExitState,
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		Retryable:  false,
	}
}

func NewRateLimitError(retryAfterSecs int) *CLIError {
	return &CLIError{
		ExitCode:   ExitRateLimit,
		Code:       "RATE_LIMITED",
		Message:    fmt.Sprintf("API rate limit exceeded. Retry after %d seconds.", retryAfterSecs),
		Suggestion: "Wait and retry, or reduce request frequency",
		Retryable:  true,
		Details:    map[string]int{"retry_after_seconds": retryAfterSecs},
	}
}

func NewNetworkError(err error) *CLIError {
	return &CLIError{
		ExitCode:   ExitNetworkAPI,
		Code:       "NETWORK_ERROR",
		Message:    fmt.Sprintf("Cannot reach Bitbucket API: %v", err),
		Suggestion: "Check your network connection and try again",
		Retryable:  true,
	}
}

func NewAPIError(statusCode int, body string) *CLIError {
	return &CLIError{
		ExitCode:   ExitNetworkAPI,
		Code:       "API_ERROR",
		Message:    fmt.Sprintf("Bitbucket API returned %d: %s", statusCode, body),
		Suggestion: "This is a Bitbucket server error. Try again later.",
		Retryable:  true,
	}
}

func NewUsageError(message string) *CLIError {
	return &CLIError{
		ExitCode:   ExitUsage,
		Code:       "USAGE_ERROR",
		Message:    message,
		Retryable:  false,
	}
}

func NewGeneralError(message string) *CLIError {
	return &CLIError{
		ExitCode: ExitGeneral,
		Code:     "GENERAL_ERROR",
		Message:  message,
	}
}
