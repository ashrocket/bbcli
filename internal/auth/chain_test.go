package auth_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ashrocket/bbcli/internal/auth"
	"github.com/ashrocket/bbcli/internal/errors"
)

// --- Resolve: priority and fallthrough ---

func TestEnvVarResolvesToLevel1(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "env-token-value")

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "env-token-value" {
		t.Errorf("Token = %q, want %q", result.Token, "env-token-value")
	}
	if result.Source != auth.LevelEnvVar {
		t.Errorf("Source = %d, want %d (LevelEnvVar)", result.Source, auth.LevelEnvVar)
	}
}

func TestFlagResolvesToLevel2(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "") // ensure env var is absent
	os.Unsetenv("BBCLI_TOKEN")

	result, err := auth.Resolve("flag-token-value", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "flag-token-value" {
		t.Errorf("Token = %q, want %q", result.Token, "flag-token-value")
	}
	if result.Source != auth.LevelFlag {
		t.Errorf("Source = %d, want %d (LevelFlag)", result.Source, auth.LevelFlag)
	}
}

func TestEnvWinsOverFlag(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "env-wins")

	result, err := auth.Resolve("flag-loses", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "env-wins" {
		t.Errorf("Token = %q, want %q (env should beat flag)", result.Token, "env-wins")
	}
	if result.Source != auth.LevelEnvVar {
		t.Errorf("Source = %d, want %d (LevelEnvVar)", result.Source, auth.LevelEnvVar)
	}
}

func TestNeitherEnvNorFlagFallsThrough(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	// With no keychain and no legacy files, should get AuthMissing.
	_, err := auth.Resolve("", "")
	if err == nil {
		t.Fatal("expected error when no token found anywhere")
	}
	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.Code != "AUTH_MISSING" {
		t.Errorf("Code = %q, want AUTH_MISSING", cliErr.Code)
	}
}

// --- AuthKind: Basic vs Bearer ---

func TestTokenWithColonIsBasicAuth(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "user@example.com:app-password")

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != auth.BasicAuth {
		t.Errorf("Kind = %d, want %d (BasicAuth)", result.Kind, auth.BasicAuth)
	}
}

func TestTokenWithoutColonIsBearerAuth(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "some-oauth-token-no-colon")

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != auth.BearerAuth {
		t.Errorf("Kind = %d, want %d (BearerAuth)", result.Kind, auth.BearerAuth)
	}
}

// --- Chain trace: records all levels checked ---

func TestChainTraceRecordsLevelsChecked(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "found-at-env")

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When env var is set, only level 1 is checked.
	if len(result.Trace) != 1 {
		t.Fatalf("Trace length = %d, want 1", len(result.Trace))
	}
	if result.Trace[0].Level != auth.LevelEnvVar {
		t.Errorf("Trace[0].Level = %d, want %d", result.Trace[0].Level, auth.LevelEnvVar)
	}
	if !result.Trace[0].Found {
		t.Error("Trace[0].Found = false, want true")
	}
}

func TestChainTraceRecordsMultipleLevels(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	// No token anywhere — should check all 4 levels.
	_, err := auth.Resolve("", "")
	if err == nil {
		t.Fatal("expected error when no token found")
	}

	// We need the trace even on error. Use ResolveWithTrace.
	trace := auth.LastTrace()
	if len(trace) < 4 {
		t.Fatalf("Trace length = %d, want >= 4 (all levels checked)", len(trace))
	}
	for _, entry := range trace {
		if entry.Found {
			t.Errorf("Level %d should not be found", entry.Level)
		}
	}
}

// --- Legacy file resolution ---

func TestLegacyFileRepoSpecific(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create repo-specific legacy file.
	repoFile := filepath.Join(tmpDir, ".bb-cli-token-myrepo")
	if err := os.WriteFile(repoFile, []byte("repo-specific-token"), 0600); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	result, err := auth.Resolve("", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "repo-specific-token" {
		t.Errorf("Token = %q, want %q", result.Token, "repo-specific-token")
	}
	if result.Source != auth.LevelLegacyFile {
		t.Errorf("Source = %d, want %d (LevelLegacyFile)", result.Source, auth.LevelLegacyFile)
	}
}

func TestLegacyFilePersonalFallback(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Only create the personal token file (no repo-specific).
	personalFile := filepath.Join(tmpDir, ".bb-cli-personal-token")
	if err := os.WriteFile(personalFile, []byte("personal-token"), 0600); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	result, err := auth.Resolve("", "somerepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "personal-token" {
		t.Errorf("Token = %q, want %q", result.Token, "personal-token")
	}
}

func TestLegacyFileGenericFallback(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Only create the generic token file.
	genericFile := filepath.Join(tmpDir, ".bb-cli-token")
	if err := os.WriteFile(genericFile, []byte("generic-token"), 0600); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "generic-token" {
		t.Errorf("Token = %q, want %q", result.Token, "generic-token")
	}
}

func TestLegacyFileRepoSpecificBeatsPersonal(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create both repo-specific and personal files.
	repoFile := filepath.Join(tmpDir, ".bb-cli-token-myrepo")
	personalFile := filepath.Join(tmpDir, ".bb-cli-personal-token")
	os.WriteFile(repoFile, []byte("repo-token"), 0600)
	os.WriteFile(personalFile, []byte("personal-token"), 0600)

	result, err := auth.Resolve("", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "repo-token" {
		t.Errorf("Token = %q, want %q (repo-specific should win)", result.Token, "repo-token")
	}
}

func TestLegacyFileTrimsWhitespace(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	genericFile := filepath.Join(tmpDir, ".bb-cli-token")
	if err := os.WriteFile(genericFile, []byte("  token-with-spaces\n"), 0600); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "token-with-spaces" {
		t.Errorf("Token = %q, want %q (should be trimmed)", result.Token, "token-with-spaces")
	}
}

// --- No token anywhere → AuthMissing ---

func TestNoTokenAnywhereReturnsAuthMissing(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// tmpDir has no legacy files

	_, err := auth.Resolve("", "")
	if err == nil {
		t.Fatal("expected AuthMissing error")
	}
	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected *errors.CLIError, got %T", err)
	}
	if cliErr.Code != "AUTH_MISSING" {
		t.Errorf("Code = %q, want AUTH_MISSING", cliErr.Code)
	}
	if cliErr.ExitCode != errors.ExitAuth {
		t.Errorf("ExitCode = %d, want %d", cliErr.ExitCode, errors.ExitAuth)
	}
}

// --- tryLevel helper ---

func TestTryLevelFound(t *testing.T) {
	entry := auth.TryLevel(auth.LevelEnvVar, "env", func() (string, error) {
		return "found-token", nil
	})
	if !entry.Found {
		t.Error("Found = false, want true")
	}
	if entry.Token != "found-token" {
		t.Errorf("Token = %q, want %q", entry.Token, "found-token")
	}
	if entry.Level != auth.LevelEnvVar {
		t.Errorf("Level = %d, want %d", entry.Level, auth.LevelEnvVar)
	}
}

func TestTryLevelNotFound(t *testing.T) {
	entry := auth.TryLevel(auth.LevelFlag, "flag", func() (string, error) {
		return "", nil
	})
	if entry.Found {
		t.Error("Found = true, want false")
	}
	if entry.Token != "" {
		t.Errorf("Token = %q, want empty", entry.Token)
	}
}

func TestTryLevelError(t *testing.T) {
	entry := auth.TryLevel(auth.LevelKeychain, "keychain", func() (string, error) {
		return "", os.ErrNotExist
	})
	if entry.Found {
		t.Error("Found = true, want false when error")
	}
	if entry.Err == nil {
		t.Error("Err = nil, want error propagated")
	}
}

// --- Edge cases ---

func TestEmptyEnvVarIsAbsent(t *testing.T) {
	// An empty BBCLI_TOKEN should be treated as absent, not invalid.
	t.Setenv("BBCLI_TOKEN", "")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	genericFile := filepath.Join(tmpDir, ".bb-cli-token")
	os.WriteFile(genericFile, []byte("fallback-token"), 0600)

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "fallback-token" {
		t.Errorf("Token = %q, want %q (empty env should fall through)", result.Token, "fallback-token")
	}
}

func TestEmptyFlagIsAbsent(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	genericFile := filepath.Join(tmpDir, ".bb-cli-token")
	os.WriteFile(genericFile, []byte("fallback-token"), 0600)

	// Empty string flag should be treated as absent.
	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "fallback-token" {
		t.Errorf("Token = %q, want %q (empty flag should fall through)", result.Token, "fallback-token")
	}
}

func TestBasicAuthColonInLegacyFile(t *testing.T) {
	t.Setenv("BBCLI_TOKEN", "")
	os.Unsetenv("BBCLI_TOKEN")

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	genericFile := filepath.Join(tmpDir, ".bb-cli-token")
	os.WriteFile(genericFile, []byte("user@example.com:app-password"), 0600)

	result, err := auth.Resolve("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Kind != auth.BasicAuth {
		t.Errorf("Kind = %d, want %d (BasicAuth for colon token)", result.Kind, auth.BasicAuth)
	}
}

// --- Level label display ---

func TestLevelLabels(t *testing.T) {
	tests := []struct {
		level auth.Level
		want  string
	}{
		{auth.LevelEnvVar, "env"},
		{auth.LevelFlag, "flag"},
		{auth.LevelKeychain, "keychain"},
		{auth.LevelLegacyFile, "legacy-file"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if !strings.Contains(strings.ToLower(got), tt.want) {
				t.Errorf("Level(%d).String() = %q, want it to contain %q", tt.level, got, tt.want)
			}
		})
	}
}
