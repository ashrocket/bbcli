// Package auth resolves authentication tokens from a 5-level priority chain:
//
//  1. BBCLI_TOKEN environment variable (highest priority)
//  2. --token CLI flag
//  3. Credentials file (~/.config/bbcli/credentials)
//  4. OS keychain (stub — not yet implemented)
//  5. Legacy files (~/.bb-cli-token-{repo}, ~/.bb-cli-personal-token, ~/.bb-cli-token)
//
// Present-but-invalid = hard error (no fallthrough).
// Absent = continue to next level.
// Tokens containing ":" are Basic auth; all others are Bearer auth.
package auth

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ashrocket/bbcli/internal/errors"
)

// Level identifies where a token was (or wasn't) found.
type Level int

const (
	LevelEnvVar      Level = 1
	LevelFlag        Level = 2
	LevelCredFile    Level = 3
	LevelKeychain    Level = 4
	LevelLegacyFile  Level = 5
)

// String returns a human-readable label for display in `auth status`.
func (l Level) String() string {
	switch l {
	case LevelEnvVar:
		return "env"
	case LevelFlag:
		return "flag"
	case LevelCredFile:
		return "credentials-file"
	case LevelKeychain:
		return "keychain"
	case LevelLegacyFile:
		return "legacy-file"
	default:
		return "unknown"
	}
}

// AuthKind distinguishes Basic (email:token) from Bearer (OAuth/PAT) auth.
type AuthKind int

const (
	BearerAuth AuthKind = iota
	BasicAuth
)

// TraceEntry records whether a specific level was checked and what happened.
type TraceEntry struct {
	Level Level
	Label string
	Found bool
	Token string // populated only if Found
	Err   error  // non-nil if the level returned an error
}

// Result is the resolved authentication token with metadata.
type Result struct {
	Token  string
	Kind   AuthKind
	Source Level
	Trace  []TraceEntry
}

// lastTrace stores the trace from the most recent Resolve call,
// so callers can inspect it even when Resolve returns an error.
var (
	lastTraceMu sync.Mutex
	lastTraceVal []TraceEntry
)

// LastTrace returns the trace from the most recent call to Resolve.
// This is useful for `auth status` display when Resolve returns an error.
func LastTrace() []TraceEntry {
	lastTraceMu.Lock()
	defer lastTraceMu.Unlock()
	cp := make([]TraceEntry, len(lastTraceVal))
	copy(cp, lastTraceVal)
	return cp
}

// TryLevel runs fn and wraps the result as a TraceEntry.
// This is the shared helper that extracts the repeated try-and-record pattern.
func TryLevel(level Level, label string, fn func() (string, error)) TraceEntry {
	token, err := fn()
	if err != nil {
		return TraceEntry{Level: level, Label: label, Found: false, Err: err}
	}
	if token == "" {
		return TraceEntry{Level: level, Label: label, Found: false}
	}
	return TraceEntry{Level: level, Label: label, Found: true, Token: token}
}

// Resolve walks the 5-level auth chain and returns the first token found.
// tokenFlag is the value of --token (empty string if not provided).
// repoName is the current repository slug (used for legacy file lookup).
func Resolve(tokenFlag, repoName string) (*Result, error) {
	var trace []TraceEntry

	levels := []struct {
		level Level
		label string
		fn    func() (string, error)
	}{
		{LevelEnvVar, "env", readEnvVar},
		{LevelFlag, "flag", func() (string, error) { return tokenFlag, nil }},
		{LevelCredFile, "credentials-file", readCredentialsFile},
		{LevelKeychain, "keychain", readKeychain},
		{LevelLegacyFile, "legacy-file", func() (string, error) { return readLegacyFiles(repoName) }},
	}

	for _, l := range levels {
		entry := TryLevel(l.level, l.label, l.fn)
		trace = append(trace, entry)

		if entry.Found {
			// Save trace and return success.
			saveTrace(trace)
			return &Result{
				Token:  entry.Token,
				Kind:   classifyToken(entry.Token),
				Source: entry.Level,
				Trace:  trace,
			}, nil
		}
		// Absent (not found, no error) → continue to next level.
		// An error from keychain is not fatal — it means "not configured",
		// so we fall through.
	}

	// Nothing found anywhere.
	saveTrace(trace)
	return nil, errors.NewAuthMissing()
}

func saveTrace(trace []TraceEntry) {
	lastTraceMu.Lock()
	defer lastTraceMu.Unlock()
	lastTraceVal = make([]TraceEntry, len(trace))
	copy(lastTraceVal, trace)
}

// classifyToken determines if a token is Basic (email:password) or Bearer.
func classifyToken(token string) AuthKind {
	if strings.Contains(token, ":") {
		return BasicAuth
	}
	return BearerAuth
}

// readEnvVar reads BBCLI_TOKEN. Empty string = absent.
func readEnvVar() (string, error) {
	return os.Getenv("BBCLI_TOKEN"), nil
}

// readCredentialsFile reads the token from ~/.config/bbcli/credentials.
// Respects BBCLI_CONFIG_DIR env var for the config directory.
func readCredentialsFile() (string, error) {
	dir := os.Getenv("BBCLI_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil // can't determine home — absent
		}
		dir = filepath.Join(home, ".config", "bbcli")
	}
	credPath := filepath.Join(dir, "credentials")
	data, err := os.ReadFile(credPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // absent
		}
		return "", err
	}
	token := strings.TrimSpace(string(data))
	return token, nil
}

// readKeychain is a stub that always returns an error.
// This will be replaced with real OS keychain integration later.
func readKeychain() (string, error) {
	return "", os.ErrNotExist
}

// readLegacyFiles checks legacy token files in priority order:
//  1. ~/.bb-cli-token-{repo}  (if repoName is non-empty)
//  2. ~/.bb-cli-personal-token
//  3. ~/.bb-cli-token
func readLegacyFiles(repoName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var candidates []string
	if repoName != "" {
		candidates = append(candidates, filepath.Join(home, ".bb-cli-token-"+repoName))
	}
	candidates = append(candidates,
		filepath.Join(home, ".bb-cli-personal-token"),
		filepath.Join(home, ".bb-cli-token"),
	)

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err // real I/O error → propagate
		}
		token := strings.TrimSpace(string(data))
		if token != "" {
			return token, nil
		}
	}

	return "", nil // no file found → absent
}
