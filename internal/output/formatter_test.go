package output_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	clierrors "github.com/ashrocket/bbcli/internal/errors"
	"github.com/ashrocket/bbcli/internal/output"
)

// --- Mock Result implementation ---

// mockResult implements the Result interface for testing.
type mockResult struct {
	headers []string
	rows    [][]string
	minimal []string
}

func (m *mockResult) Headers() []string    { return m.headers }
func (m *mockResult) Rows() [][]string     { return m.rows }
func (m *mockResult) MinimalLines() []string { return m.minimal }

// newPRResult creates a mock result representing a list of pull requests.
func newPRResult() *mockResult {
	return &mockResult{
		headers: []string{"ID", "Title", "State"},
		rows: [][]string{
			{"1", "Add login page", "OPEN"},
			{"2", "Fix header alignment", "MERGED"},
			{"42", "Upgrade dependencies to latest", "DECLINED"},
		},
		minimal: []string{"1", "2", "42"},
	}
}

// newSingleResult creates a mock result with a single row.
func newSingleResult() *mockResult {
	return &mockResult{
		headers: []string{"ID", "URL"},
		rows:    [][]string{{"99", "https://bitbucket.org/repo/pull-requests/99"}},
		minimal: []string{"99"},
	}
}

// newEmptyResult creates a mock result with no rows.
func newEmptyResult() *mockResult {
	return &mockResult{
		headers: []string{"ID", "Title", "State"},
		rows:    [][]string{},
		minimal: []string{},
	}
}

// --- Result interface tests ---

func TestMockResultImplementsInterface(t *testing.T) {
	var r output.Result = &mockResult{}
	_ = r // compile-time check that mockResult satisfies Result
}

func TestResultHeaders(t *testing.T) {
	r := newPRResult()
	got := r.Headers()
	want := []string{"ID", "Title", "State"}
	if len(got) != len(want) {
		t.Fatalf("Headers() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Headers()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// --- JSON output tests ---

func TestFormatJSON(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "json")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()

	// Must be valid JSON
	if !json.Valid([]byte(out)) {
		t.Fatalf("output is not valid JSON:\n%s", out)
	}

	// Must be indented (multi-line)
	if !strings.Contains(out, "\n") {
		t.Error("JSON output should be indented (multi-line)")
	}

	// Parse and verify structure: array of objects with header keys
	var parsed []map[string]string
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("cannot unmarshal as []map[string]string: %v", err)
	}

	if len(parsed) != 3 {
		t.Fatalf("expected 3 items, got %d", len(parsed))
	}

	// Spot-check first row
	if parsed[0]["ID"] != "1" {
		t.Errorf("first row ID = %q, want %q", parsed[0]["ID"], "1")
	}
	if parsed[0]["Title"] != "Add login page" {
		t.Errorf("first row Title = %q, want %q", parsed[0]["Title"], "Add login page")
	}
	if parsed[0]["State"] != "OPEN" {
		t.Errorf("first row State = %q, want %q", parsed[0]["State"], "OPEN")
	}

	// Spot-check last row
	if parsed[2]["ID"] != "42" {
		t.Errorf("last row ID = %q, want %q", parsed[2]["ID"], "42")
	}
}

func TestFormatJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := newEmptyResult()

	err := output.Format(&buf, r, "json")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "[]" {
		t.Errorf("empty JSON output = %q, want %q", out, "[]")
	}
}

func TestFormatJSONSingleRow(t *testing.T) {
	var buf bytes.Buffer
	r := newSingleResult()

	err := output.Format(&buf, r, "json")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	var parsed []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("cannot unmarshal: %v", err)
	}

	if len(parsed) != 1 {
		t.Fatalf("expected 1 item, got %d", len(parsed))
	}
	if parsed[0]["URL"] != "https://bitbucket.org/repo/pull-requests/99" {
		t.Errorf("URL = %q, want full URL", parsed[0]["URL"])
	}
}

// --- Minimal output tests ---

func TestFormatMinimal(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "minimal")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	// Should be the minimal values, one per line, no headers
	if lines[0] != "1" {
		t.Errorf("line 0 = %q, want %q", lines[0], "1")
	}
	if lines[1] != "2" {
		t.Errorf("line 1 = %q, want %q", lines[1], "2")
	}
	if lines[2] != "42" {
		t.Errorf("line 2 = %q, want %q", lines[2], "42")
	}
}

func TestFormatMinimalNoHeaders(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "minimal")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	// Should NOT contain any header text
	if strings.Contains(out, "ID") || strings.Contains(out, "Title") || strings.Contains(out, "State") {
		t.Error("minimal output should not contain headers")
	}
}

func TestFormatMinimalEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := newEmptyResult()

	err := output.Format(&buf, r, "minimal")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	if out != "" {
		t.Errorf("empty minimal output = %q, want empty string", out)
	}
}

// --- Table output tests ---

func TestFormatTable(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// Should have header + 3 data rows
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (1 header + 3 rows), got %d:\n%s", len(lines), out)
	}

	// Header line must contain all column names
	header := lines[0]
	for _, h := range []string{"ID", "Title", "State"} {
		if !strings.Contains(header, h) {
			t.Errorf("header should contain %q, got: %q", h, header)
		}
	}

	// Data rows must contain their values
	if !strings.Contains(lines[1], "Add login page") {
		t.Errorf("row 1 should contain 'Add login page', got: %q", lines[1])
	}
	if !strings.Contains(lines[2], "MERGED") {
		t.Errorf("row 2 should contain 'MERGED', got: %q", lines[2])
	}
}

func TestFormatTableAlignment(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	// The "State" column values should start at the same position across rows,
	// which is the property tabwriter provides.
	// We verify by checking that all lines that contain a state value
	// have "State"/"OPEN"/"MERGED"/"DECLINED" starting at the same column.
	statePositions := make(map[int]bool)
	stateValues := []string{"State", "OPEN", "MERGED", "DECLINED"}
	for i, line := range lines {
		for _, sv := range stateValues {
			idx := strings.Index(line, sv)
			if idx >= 0 {
				statePositions[idx] = true
				_ = i
				break
			}
		}
	}

	if len(statePositions) != 1 {
		t.Errorf("state column values should be aligned to one position, found positions: %v\nfull output:\n%s",
			statePositions, out)
	}
}

func TestFormatTableNoANSICodes(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	// ANSI escape sequences start with ESC (0x1b). Table output must never
	// contain them because they break tabwriter alignment.
	if strings.Contains(out, "\x1b") {
		t.Errorf("table output contains ANSI escape codes, which break tabwriter alignment:\n%q", out)
	}
}

func TestFormatTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := newEmptyResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	// Empty table: no output at all (no orphan headers)
	if strings.TrimSpace(out) != "" {
		t.Errorf("empty table output should be empty, got: %q", out)
	}
}

// --- NO_COLOR env var tests ---

func TestNoColorEnvVarRespected(t *testing.T) {
	// Set NO_COLOR, format table output, verify no ANSI codes
	t.Setenv("NO_COLOR", "1")

	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Errorf("with NO_COLOR=1, output should not contain ANSI codes:\n%q", out)
	}
}

func TestNoColorEnvVarEmpty(t *testing.T) {
	// NO_COLOR="" means color is disabled (spec says presence matters, any value)
	t.Setenv("NO_COLOR", "")

	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	// Even with NO_COLOR="", we don't use ANSI in tabwriter anyway
	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Errorf("output should not contain ANSI codes:\n%q", out)
	}
}

func TestNoColorUnset(t *testing.T) {
	// Ensure NO_COLOR is not set
	os.Unsetenv("NO_COLOR")

	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	// Table output should still not have ANSI in tabwriter
	out := buf.String()
	if strings.Contains(out, "\x1b") {
		t.Errorf("table output should never contain ANSI codes inside tabwriter:\n%q", out)
	}
}

// --- Format dispatch tests ---

func TestFormatDispatchJSON(t *testing.T) {
	var buf bytes.Buffer
	r := newSingleResult()

	err := output.Format(&buf, r, "json")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Error("'json' mode should produce valid JSON")
	}
}

func TestFormatDispatchTable(t *testing.T) {
	var buf bytes.Buffer
	r := newSingleResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ID") {
		t.Error("'table' mode should include headers")
	}
}

func TestFormatDispatchMinimal(t *testing.T) {
	var buf bytes.Buffer
	r := newSingleResult()

	err := output.Format(&buf, r, "minimal")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "99" {
		t.Errorf("'minimal' mode output = %q, want %q", out, "99")
	}
}

// --- Unknown mode error tests ---

func TestFormatUnknownModeReturnsError(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "yaml")
	if err == nil {
		t.Fatal("Format() should return error for unknown mode 'yaml'")
	}
}

func TestFormatUnknownModeReturnsUsageError(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "xml")

	if err == nil {
		t.Fatal("Format() should return error for unknown mode 'xml'")
	}

	// Must be a *errors.CLIError with usage exit code
	var cliErr *clierrors.CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("error should be *errors.CLIError, got %T: %v", err, err)
	}
	if cliErr.ExitCode != clierrors.ExitUsage {
		t.Errorf("ExitCode = %d, want %d (ExitUsage)", cliErr.ExitCode, clierrors.ExitUsage)
	}
	if cliErr.Code != "USAGE_ERROR" {
		t.Errorf("Code = %q, want %q", cliErr.Code, "USAGE_ERROR")
	}
}

func TestFormatUnknownModeErrorMessage(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "csv")
	if err == nil {
		t.Fatal("expected error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "csv") {
		t.Errorf("error message should mention the unknown mode 'csv', got: %q", msg)
	}
}

func TestFormatEmptyModeString(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "")
	if err == nil {
		t.Fatal("Format() should return error for empty mode string")
	}
}

// --- Edge cases ---

func TestFormatJSONSpecialCharacters(t *testing.T) {
	r := &mockResult{
		headers: []string{"Title"},
		rows:    [][]string{{"Fix \"quotes\" & <angles>"}},
		minimal: []string{"Fix \"quotes\" & <angles>"},
	}

	var buf bytes.Buffer
	err := output.Format(&buf, r, "json")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	// Must be valid JSON even with special chars
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("output should be valid JSON:\n%s", buf.String())
	}

	var parsed []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed[0]["Title"] != "Fix \"quotes\" & <angles>" {
		t.Errorf("Title = %q, want original string with special chars", parsed[0]["Title"])
	}
}

func TestFormatMinimalTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "minimal")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	// Should end with exactly one newline
	if !strings.HasSuffix(out, "\n") {
		t.Error("minimal output should end with a newline")
	}
	if strings.HasSuffix(out, "\n\n") {
		t.Error("minimal output should not end with double newline")
	}
}

func TestFormatTableTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "table")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	// tabwriter output ends with newline
	if !strings.HasSuffix(out, "\n") {
		t.Error("table output should end with a newline")
	}
}

func TestFormatJSONTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	r := newPRResult()

	err := output.Format(&buf, r, "json")
	if err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Error("JSON output should end with a trailing newline for clean shell output")
	}
}
