package git

import (
	"testing"
)

func TestParseRemoteURL_SSH(t *testing.T) {
	url := "git@bitbucket.org:myworkspace/myrepo.git"
	info, err := parseRemoteURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want %q", info.Workspace, "myworkspace")
	}
	if info.Repo != "myrepo" {
		t.Errorf("Repo = %q, want %q", info.Repo, "myrepo")
	}
}

func TestParseRemoteURL_HTTPS(t *testing.T) {
	url := "https://bitbucket.org/myworkspace/myrepo.git"
	info, err := parseRemoteURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want %q", info.Workspace, "myworkspace")
	}
	if info.Repo != "myrepo" {
		t.Errorf("Repo = %q, want %q", info.Repo, "myrepo")
	}
}

func TestParseRemoteURL_HTTPSWithUsername(t *testing.T) {
	url := "https://user@bitbucket.org/myworkspace/myrepo.git"
	info, err := parseRemoteURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want %q", info.Workspace, "myworkspace")
	}
	if info.Repo != "myrepo" {
		t.Errorf("Repo = %q, want %q", info.Repo, "myrepo")
	}
}

func TestParseRemoteURL_SSHWithoutGitSuffix(t *testing.T) {
	url := "git@bitbucket.org:myworkspace/myrepo"
	info, err := parseRemoteURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want %q", info.Workspace, "myworkspace")
	}
	if info.Repo != "myrepo" {
		t.Errorf("Repo = %q, want %q", info.Repo, "myrepo")
	}
}

func TestParseRemoteURL_HTTPSWithoutGitSuffix(t *testing.T) {
	url := "https://bitbucket.org/myworkspace/myrepo"
	info, err := parseRemoteURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Workspace != "myworkspace" {
		t.Errorf("Workspace = %q, want %q", info.Workspace, "myworkspace")
	}
	if info.Repo != "myrepo" {
		t.Errorf("Repo = %q, want %q", info.Repo, "myrepo")
	}
}

func TestParseRemoteURL_NonBitbucketSSH(t *testing.T) {
	url := "git@github.com:user/repo.git"
	_, err := parseRemoteURL(url)
	if err == nil {
		t.Fatal("expected error for non-Bitbucket SSH URL, got nil")
	}
	de, ok := err.(*DetectError)
	if !ok {
		t.Fatalf("expected *DetectError, got %T", err)
	}
	if de.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestParseRemoteURL_NonBitbucketHTTPS(t *testing.T) {
	url := "https://github.com/user/repo.git"
	_, err := parseRemoteURL(url)
	if err == nil {
		t.Fatal("expected error for non-Bitbucket HTTPS URL, got nil")
	}
	if _, ok := err.(*DetectError); !ok {
		t.Fatalf("expected *DetectError, got %T", err)
	}
}

func TestParseRemoteURL_EmptyURL(t *testing.T) {
	_, err := parseRemoteURL("")
	if err == nil {
		t.Fatal("expected error for empty URL, got nil")
	}
	de, ok := err.(*DetectError)
	if !ok {
		t.Fatalf("expected *DetectError, got %T", err)
	}
	if de.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestParseRemoteURL_MalformedURL(t *testing.T) {
	urls := []string{
		"bitbucket.org/workspace/repo",       // no scheme or git@ prefix
		"git@bitbucket.org:",                  // missing path after colon
		"git@bitbucket.org:workspace",         // missing repo
		"https://bitbucket.org/",              // missing workspace and repo
		"https://bitbucket.org/workspace",     // missing repo
		"https://bitbucket.org/workspace/",    // trailing slash, no repo
	}
	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			_, err := parseRemoteURL(url)
			if err == nil {
				t.Errorf("expected error for malformed URL %q, got nil", url)
			}
		})
	}
}

func TestParseRemoteURL_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantWS    string
		wantRepo  string
		wantError bool
	}{
		{
			name:     "ssh standard",
			url:      "git@bitbucket.org:kureapp/kure-agents.git",
			wantWS:   "kureapp",
			wantRepo: "kure-agents",
		},
		{
			name:     "https standard",
			url:      "https://bitbucket.org/kureapp/kure-agents.git",
			wantWS:   "kureapp",
			wantRepo: "kure-agents",
		},
		{
			name:     "https with user",
			url:      "https://ashley@bitbucket.org/kureapp/kureapp-dashboard.git",
			wantWS:   "kureapp",
			wantRepo: "kureapp-dashboard",
		},
		{
			name:     "ssh no .git suffix",
			url:      "git@bitbucket.org:kureapp/kuredevtools",
			wantWS:   "kureapp",
			wantRepo: "kuredevtools",
		},
		{
			name:     "repo with hyphens and numbers",
			url:      "git@bitbucket.org:my-org/my-repo-2.git",
			wantWS:   "my-org",
			wantRepo: "my-repo-2",
		},
		{
			name:      "github ssh",
			url:       "git@github.com:user/repo.git",
			wantError: true,
		},
		{
			name:      "gitlab https",
			url:       "https://gitlab.com/user/repo.git",
			wantError: true,
		},
		{
			name:      "empty",
			url:       "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseRemoteURL(tt.url)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if _, ok := err.(*DetectError); !ok {
					t.Errorf("expected *DetectError, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if info.Workspace != tt.wantWS {
				t.Errorf("Workspace = %q, want %q", info.Workspace, tt.wantWS)
			}
			if info.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", info.Repo, tt.wantRepo)
			}
		})
	}
}

func TestDetectError_ErrorInterface(t *testing.T) {
	de := &DetectError{Msg: "something went wrong"}
	var e error = de // must satisfy error interface
	if e.Error() != "something went wrong" {
		t.Errorf("Error() = %q, want %q", e.Error(), "something went wrong")
	}
}

func TestRepoInfo_Fields(t *testing.T) {
	info := &RepoInfo{
		Workspace: "ws",
		Repo:      "rp",
		Branch:    "main",
	}
	if info.Workspace != "ws" {
		t.Errorf("Workspace = %q, want %q", info.Workspace, "ws")
	}
	if info.Repo != "rp" {
		t.Errorf("Repo = %q, want %q", info.Repo, "rp")
	}
	if info.Branch != "main" {
		t.Errorf("Branch = %q, want %q", info.Branch, "main")
	}
}
