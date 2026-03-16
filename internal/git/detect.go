// Package git detects Bitbucket workspace and repository from git remote URLs.
package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// RepoInfo holds the detected Bitbucket repository metadata.
type RepoInfo struct {
	Workspace string
	Repo      string
	Branch    string
}

// DetectError is returned when remote URL parsing or detection fails.
type DetectError struct {
	Msg string
}

func (e *DetectError) Error() string {
	return e.Msg
}

// Separate patterns for SSH (colon separator) and HTTPS (slash separator).
// SSH:   git@bitbucket.org:workspace/repo.git
// HTTPS: https://[user@]bitbucket.org/workspace/repo.git
var (
	sshPattern   = regexp.MustCompile(`^git@bitbucket\.org:([^/]+)/([^/]+?)(?:\.git)?$`)
	httpsPattern = regexp.MustCompile(`^https?://(?:[^@]+@)?bitbucket\.org/([^/]+)/([^/]+?)(?:\.git)?$`)
)

// parseRemoteURL extracts workspace and repo from a Bitbucket remote URL.
// This is the testable core — no git subprocess calls.
func parseRemoteURL(url string) (*RepoInfo, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, &DetectError{Msg: "remote URL is empty"}
	}

	// Try SSH pattern first (git@bitbucket.org:workspace/repo)
	if m := sshPattern.FindStringSubmatch(url); m != nil {
		ws, repo := m[1], m[2]
		if ws == "" || repo == "" {
			return nil, &DetectError{Msg: fmt.Sprintf("could not extract workspace/repo from SSH URL: %s", url)}
		}
		return &RepoInfo{Workspace: ws, Repo: repo}, nil
	}

	// Try HTTPS pattern (https://[user@]bitbucket.org/workspace/repo)
	if m := httpsPattern.FindStringSubmatch(url); m != nil {
		ws, repo := m[1], m[2]
		if ws == "" || repo == "" {
			return nil, &DetectError{Msg: fmt.Sprintf("could not extract workspace/repo from HTTPS URL: %s", url)}
		}
		return &RepoInfo{Workspace: ws, Repo: repo}, nil
	}

	return nil, &DetectError{
		Msg: fmt.Sprintf("not a Bitbucket remote URL: %s (expected git@bitbucket.org:ws/repo or https://bitbucket.org/ws/repo)", url),
	}
}

// DetectRemote reads the git origin remote URL and current branch,
// then parses them into RepoInfo. This is the public API.
func DetectRemote() (*RepoInfo, error) {
	// Get the remote URL for origin.
	urlBytes, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return nil, &DetectError{
			Msg: fmt.Sprintf("failed to get git remote URL: %v (are you in a git repository with an 'origin' remote?)", err),
		}
	}

	url := strings.TrimSpace(string(urlBytes))
	info, err := parseRemoteURL(url)
	if err != nil {
		return nil, err
	}

	// Get the current branch.
	branchBytes, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		// Non-fatal: we still have workspace and repo.
		info.Branch = ""
	} else {
		info.Branch = strings.TrimSpace(string(branchBytes))
	}

	return info, nil
}
