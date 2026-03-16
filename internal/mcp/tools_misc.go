package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ashrocket/bbcli/internal/api"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerMiscTools registers compare, search, commit, and escape-hatch tools.
func (s *Server) registerMiscTools() {
	// --- bitbucket_compare_diff ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_compare_diff",
		mcp.WithDescription("Compare two refs and return a diff. "+
			"The spec is formatted as 'base..head' (e.g. 'main..feature')."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("spec",
			mcp.Description("Diff spec in 'base..head' format (e.g. 'main..feature')."),
			mcp.Required(),
		),
	), s.HandleCompareDiff)

	// --- bitbucket_search_code ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_search_code",
		mcp.WithDescription("Search code in a Bitbucket workspace."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("query",
			mcp.Description("Search query string."),
			mcp.Required(),
		),
	), s.HandleSearchCode)

	// --- bitbucket_commit_list ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_commit_list",
		mcp.WithDescription("List commits in a repository, optionally filtered by branch."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("branch",
			mcp.Description("Filter commits by branch name."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of commits to return (default 25, max 100)."),
			mcp.DefaultNumber(25),
			mcp.Min(1),
			mcp.Max(100),
		),
	), s.HandleCommitList)

	// --- bitbucket_commit_statuses ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_commit_statuses",
		mcp.WithDescription("Get build statuses for a specific commit."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("commit",
			mcp.Description("Full commit hash."),
			mcp.Required(),
		),
	), s.HandleCommitStatuses)

	// --- bitbucket_api (escape hatch) ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_api",
		mcp.WithDescription("Make a raw Bitbucket API request. "+
			"Use {workspace} and {repo_slug} placeholders in path for auto-substitution."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("path",
			mcp.Description("API path (e.g. '/repositories/{workspace}/{repo_slug}/issues')."),
			mcp.Required(),
		),
		mcp.WithString("method",
			mcp.Description("HTTP method."),
			mcp.DefaultString("GET"),
			mcp.Enum("GET", "POST", "PUT", "DELETE", "PATCH"),
		),
		mcp.WithString("body",
			mcp.Description("Request body as a JSON string (for POST/PUT/PATCH)."),
		),
	), s.HandleAPI)
}

// HandleCompareDiff implements the bitbucket_compare_diff tool.
func (s *Server) HandleCompareDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	spec := req.GetString("spec", "")
	if spec == "" {
		return mcp.NewToolResultError("spec is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/diff/%s", spec)
	raw, err := client.Request("GET", path, nil)
	if err != nil {
		return mapError(err), nil
	}

	return mcp.NewToolResultText(string(raw)), nil
}

// HandleSearchCode implements the bitbucket_search_code tool.
func (s *Server) HandleSearchCode(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace := req.GetString("workspace", s.defaultWorkspace)
	if workspace == "" {
		return mcp.NewToolResultError("workspace is required: pass it as a parameter or run from a git repo with a Bitbucket remote"), nil
	}

	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// Code search uses a workspace-level endpoint, no repo needed.
	client := api.NewWithBaseURL(s.client.Auth(), s.client.BaseURL(),
		api.WithWorkspaceRepo(workspace, ""),
	)

	path := fmt.Sprintf("/workspaces/{workspace}/search/code?search_query=%s", query)
	values, err := client.ListAll(path, 25)
	if err != nil {
		return mapError(err), nil
	}

	return jsonResult(values)
}

// HandleCommitList implements the bitbucket_commit_list tool.
func (s *Server) HandleCommitList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	branch := req.GetString("branch", "")
	limit := req.GetInt("limit", 25)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	client := s.scopedClient(workspace, repo)

	path := "/repositories/{workspace}/{repo_slug}/commits"
	if branch != "" {
		path += "/" + branch
	}

	values, err := client.ListAll(path, limit)
	if err != nil {
		return mapError(err), nil
	}

	return jsonResult(values)
}

// HandleCommitStatuses implements the bitbucket_commit_statuses tool.
func (s *Server) HandleCommitStatuses(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	commit := req.GetString("commit", "")
	if commit == "" {
		return mcp.NewToolResultError("commit is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/commit/%s/statuses", commit)
	values, err := client.ListAll(path, 25)
	if err != nil {
		return mapError(err), nil
	}

	return jsonResult(values)
}

// HandleAPI implements the bitbucket_api escape hatch tool.
func (s *Server) HandleAPI(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace := req.GetString("workspace", s.defaultWorkspace)
	repo := req.GetString("repo", s.defaultRepo)

	apiPath := req.GetString("path", "")
	if apiPath == "" {
		return mcp.NewToolResultError("path is required"), nil
	}

	method := req.GetString("method", "GET")
	bodyStr := req.GetString("body", "")

	var payload []byte
	if bodyStr != "" {
		// Validate that it's valid JSON.
		if !json.Valid([]byte(bodyStr)) {
			return mcp.NewToolResultError("body must be valid JSON"), nil
		}
		payload = []byte(bodyStr)
	}

	client := api.NewWithBaseURL(s.client.Auth(), s.client.BaseURL(),
		api.WithWorkspaceRepo(workspace, repo),
	)

	raw, err := client.Request(method, apiPath, payload)
	if err != nil {
		return mapError(err), nil
	}

	// Try to pretty-print JSON; if it fails, return raw.
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err == nil {
		out, err := json.MarshalIndent(parsed, "", "  ")
		if err == nil {
			return mcp.NewToolResultText(string(out)), nil
		}
	}

	return mcp.NewToolResultText(string(raw)), nil
}
