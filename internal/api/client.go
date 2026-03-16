// Package api provides the HTTP client for Bitbucket Cloud REST API v2.0.
// All requests flow through this client, which handles auth injection,
// rate limiting, retries, pagination, and debug logging.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ashrocket/bbcli/internal/errors"
)

const (
	defaultBaseURL = "https://api.bitbucket.org/2.0"
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	defaultPageLen = 50
)

// Authenticator injects auth headers into HTTP requests.
type Authenticator interface {
	Authenticate(req *http.Request) error
	Source() string
	Level() int
}

// Client is the Bitbucket API client.
type Client struct {
	http           *http.Client
	auth           Authenticator
	baseURL        string
	workspace      string
	repo           string
	debug          bool
	noRetry        bool
	noRetryBackoff bool // skip sleep in tests
}

// ClientOption configures the client.
type ClientOption func(*Client)

func WithDebug(debug bool) ClientOption {
	return func(c *Client) { c.debug = debug }
}

func WithNoRetry(noRetry bool) ClientOption {
	return func(c *Client) { c.noRetry = noRetry }
}

func WithNoRetryBackoff(skip bool) ClientOption {
	return func(c *Client) { c.noRetryBackoff = skip }
}

func WithWorkspaceRepo(workspace, repo string) ClientOption {
	return func(c *Client) {
		c.workspace = workspace
		c.repo = repo
	}
}

// New creates a new API client targeting the production Bitbucket API.
func New(authenticator Authenticator, opts ...ClientOption) *Client {
	return newClient(authenticator, defaultBaseURL, opts...)
}

// NewWithBaseURL creates a client with a custom base URL (for testing).
func NewWithBaseURL(authenticator Authenticator, baseURL string, opts ...ClientOption) *Client {
	return newClient(authenticator, baseURL, opts...)
}

func newClient(authenticator Authenticator, baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		http:    &http.Client{Timeout: defaultTimeout},
		auth:    authenticator,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Request makes a single API request with auth and retry handling.
// Body is []byte (not io.Reader) so retries can re-send the same payload.
func (c *Client) Request(method, path string, body []byte) (json.RawMessage, error) {
	path = c.substitutePlaceholders(path)

	url := c.baseURL + path
	if !strings.HasPrefix(path, "/") {
		url = c.baseURL + "/" + path
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 && c.noRetry {
			break
		}

		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequest(method, url, bodyReader)
		if err != nil {
			return nil, errors.NewNetworkError(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		if err := c.auth.Authenticate(req); err != nil {
			return nil, errors.NewAuthError(err.Error(), "Check your token configuration")
		}

		if c.debug {
			fmt.Fprintf(DebugWriter, "[DEBUG] %s %s (attempt %d)\n", method, url, attempt+1)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = errors.NewNetworkError(err)
			c.backoff(attempt)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close immediately, not deferred inside loop

		if err != nil {
			return nil, errors.NewNetworkError(err)
		}

		if c.debug {
			fmt.Fprintf(DebugWriter, "[DEBUG] %d %s\n", resp.StatusCode, url)
		}

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return json.RawMessage(respBody), nil

		case resp.StatusCode == 401, resp.StatusCode == 403:
			return nil, errors.NewAuthError(
				fmt.Sprintf("Authentication failed (HTTP %d). Token source: %s", resp.StatusCode, c.auth.Source()),
				"Check your token has the required scopes, or run 'bbcli auth login'",
			)

		case resp.StatusCode == 404:
			return nil, errors.NewNotFoundError("resource", path)

		case resp.StatusCode == 429:
			if c.noRetry {
				return nil, errors.NewRateLimitError(0)
			}
			waitSeconds := getRetryAfter(resp, attempt)
			if !c.noRetryBackoff {
				time.Sleep(time.Duration(waitSeconds) * time.Second)
			}
			lastErr = errors.NewRateLimitError(waitSeconds)
			continue

		case resp.StatusCode >= 500:
			lastErr = errors.NewAPIError(resp.StatusCode, string(respBody))
			c.backoff(attempt)
			continue // Retry 5xx with backoff

		default:
			return nil, errors.NewGeneralError(
				fmt.Sprintf("Unexpected HTTP %d: %s", resp.StatusCode, string(respBody)),
			)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.NewNetworkError(fmt.Errorf("request failed after %d attempts", maxRetries+1))
}

// ListAll auto-paginates a list endpoint, following "next" links.
func (c *Client) ListAll(path string, limit int) ([]json.RawMessage, error) {
	if limit == 0 {
		limit = 1<<31 - 1
	}

	var allValues []json.RawMessage
	currentPath := path
	if !strings.Contains(currentPath, "pagelen=") {
		separator := "?"
		if strings.Contains(currentPath, "?") {
			separator = "&"
		}
		currentPath += fmt.Sprintf("%spagelen=%d", separator, defaultPageLen)
	}

	const maxPages = 200
	for page := 0; len(allValues) < limit && page < maxPages; page++ {
		raw, err := c.Request("GET", currentPath, nil)
		if err != nil {
			return allValues, err
		}

		var pageData struct {
			Values []json.RawMessage `json:"values"`
			Next   string            `json:"next"`
		}
		if err := json.Unmarshal(raw, &pageData); err != nil {
			return nil, errors.NewGeneralError(fmt.Sprintf("Failed to parse paginated response: %v", err))
		}

		allValues = append(allValues, pageData.Values...)

		if pageData.Next == "" {
			break
		}

		next := strings.TrimPrefix(pageData.Next, c.baseURL)
		if next == pageData.Next {
			break // next URL doesn't match our base — stop
		}
		currentPath = next
	}

	if len(allValues) > limit {
		allValues = allValues[:limit]
	}

	return allValues, nil
}

func (c *Client) substitutePlaceholders(path string) string {
	path = strings.ReplaceAll(path, "{workspace}", c.workspace)
	path = strings.ReplaceAll(path, "{repo_slug}", c.repo)
	path = strings.ReplaceAll(path, "{repo}", c.repo)
	return path
}

func (c *Client) backoff(attempt int) {
	if !c.noRetryBackoff {
		time.Sleep(time.Duration(1<<attempt) * time.Second)
	}
}

func getRetryAfter(resp *http.Response, attempt int) int {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			return secs
		}
	}
	return 1 << attempt
}

// Auth returns the client's authenticator.
func (c *Client) Auth() Authenticator {
	return c.auth
}

// BaseURL returns the client's base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Workspace returns the resolved workspace.
func (c *Client) Workspace() string {
	return c.workspace
}

// Repo returns the resolved repository slug.
func (c *Client) Repo() string {
	return c.repo
}

// DebugWriter is where debug output goes (stderr in production).
var DebugWriter io.Writer = io.Discard
