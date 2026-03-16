package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ashrocket/bbcli/internal/api"
	bbErrors "github.com/ashrocket/bbcli/internal/errors"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerPRTools registers all pull-request related tools.
func (s *Server) registerPRTools() {
	tool := mcp.NewTool("bitbucket_pr_list",
		mcp.WithDescription("List pull requests for a Bitbucket repository. "+
			"Returns PR id, title, state, author, and source/destination branches."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("state",
			mcp.Description("Filter by PR state."),
			mcp.Enum("OPEN", "MERGED", "DECLINED", "SUPERSEDED"),
			mcp.DefaultString("OPEN"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of PRs to return (default 25, max 100)."),
			mcp.DefaultNumber(25),
			mcp.Min(1),
			mcp.Max(100),
		),
	)

	s.mcp.AddTool(tool, s.handlePRList)
}

// handlePRList implements the bitbucket_pr_list tool.
func (s *Server) handlePRList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace := req.GetString("workspace", s.defaultWorkspace)
	repo := req.GetString("repo", s.defaultRepo)
	state := req.GetString("state", "OPEN")
	limit := req.GetInt("limit", 25)

	if workspace == "" {
		return mcp.NewToolResultError("workspace is required: pass it as a parameter or run from a git repo with a Bitbucket remote"), nil
	}
	if repo == "" {
		return mcp.NewToolResultError("repo is required: pass it as a parameter or run from a git repo with a Bitbucket remote"), nil
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	// Build a scoped client with the requested workspace/repo,
	// preserving the parent client's base URL and auth.
	client := api.NewWithBaseURL(s.client.Auth(), s.client.BaseURL(),
		api.WithWorkspaceRepo(workspace, repo),
	)

	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests?state=%s", state)
	values, err := client.ListAll(path, limit)
	if err != nil {
		return mapError(err), nil
	}

	// Build a compact JSON array of the results.
	type prSummary struct {
		ID          json.RawMessage `json:"id"`
		Title       json.RawMessage `json:"title"`
		State       json.RawMessage `json:"state"`
		Author      json.RawMessage `json:"author"`
		Source      json.RawMessage `json:"source"`
		Destination json.RawMessage `json:"destination"`
		CreatedOn   json.RawMessage `json:"created_on"`
		UpdatedOn   json.RawMessage `json:"updated_on"`
	}

	var prs []prSummary
	for _, raw := range values {
		var full map[string]json.RawMessage
		if err := json.Unmarshal(raw, &full); err != nil {
			continue
		}
		pr := prSummary{
			ID:          full["id"],
			Title:       full["title"],
			State:       full["state"],
			Author:      full["author"],
			Source:      full["source"],
			Destination: full["destination"],
			CreatedOn:   full["created_on"],
			UpdatedOn:   full["updated_on"],
		}
		prs = append(prs, pr)
	}

	out, err := json.MarshalIndent(prs, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(out)), nil
}

// mapError converts a bbcli CLIError into an MCP tool error result.
// Tool-level errors are returned as isError results, not protocol errors.
func mapError(err error) *mcp.CallToolResult {
	if cliErr, ok := err.(*bbErrors.CLIError); ok {
		msg := cliErr.Message
		if cliErr.Suggestion != "" {
			msg += "\n\nSuggestion: " + cliErr.Suggestion
		}
		return mcp.NewToolResultError(msg)
	}
	return mcp.NewToolResultError(err.Error())
}
