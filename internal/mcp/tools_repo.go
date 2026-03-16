package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerRepoTools registers all repository and branch related tools.
func (s *Server) registerRepoTools() {
	// --- bitbucket_repo_view ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_repo_view",
		mcp.WithDescription("View repository details including default branch, language, and permissions."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
	), s.HandleRepoView)

	// --- bitbucket_branch_list ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_branch_list",
		mcp.WithDescription("List branches in a repository."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of branches to return (default 25, max 100)."),
			mcp.DefaultNumber(25),
			mcp.Min(1),
			mcp.Max(100),
		),
	), s.HandleBranchList)

	// --- bitbucket_branch_delete ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_branch_delete",
		mcp.WithDescription("Delete a branch from a repository."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("branch_name",
			mcp.Description("Name of the branch to delete."),
			mcp.Required(),
		),
	), s.HandleBranchDelete)

	// --- bitbucket_source_list ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_source_list",
		mcp.WithDescription("List files and directories at a path in the repository source."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("path",
			mcp.Description("Path within the repository (default is root)."),
		),
		mcp.WithString("ref",
			mcp.Description("Branch, tag, or commit hash (defaults to repository main branch)."),
		),
	), s.HandleSourceList)

	// --- bitbucket_source_view ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_source_view",
		mcp.WithDescription("View the contents of a file in the repository source."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("path",
			mcp.Description("Path to the file within the repository."),
			mcp.Required(),
		),
		mcp.WithString("ref",
			mcp.Description("Branch, tag, or commit hash (defaults to repository main branch)."),
		),
	), s.HandleSourceView)
}

// HandleRepoView implements the bitbucket_repo_view tool.
func (s *Server) HandleRepoView(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	client := s.scopedClient(workspace, repo)
	raw, err := client.Request("GET", "/repositories/{workspace}/{repo_slug}", nil)
	if err != nil {
		return mapError(err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	return jsonResult(result)
}

// HandleBranchList implements the bitbucket_branch_list tool.
func (s *Server) HandleBranchList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	limit := req.GetInt("limit", 25)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	client := s.scopedClient(workspace, repo)
	values, err := client.ListAll("/repositories/{workspace}/{repo_slug}/refs/branches", limit)
	if err != nil {
		return mapError(err), nil
	}

	return jsonResult(values)
}

// HandleBranchDelete implements the bitbucket_branch_delete tool.
func (s *Server) HandleBranchDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	branchName := req.GetString("branch_name", "")
	if branchName == "" {
		return mcp.NewToolResultError("branch_name is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/refs/branches/%s", branchName)
	_, err := client.Request("DELETE", path, nil)
	if err != nil {
		return mapError(err), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Branch %q deleted successfully.", branchName)), nil
}

// HandleSourceList implements the bitbucket_source_list tool.
func (s *Server) HandleSourceList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	pathParam := req.GetString("path", "")
	ref := req.GetString("ref", "")

	client := s.scopedClient(workspace, repo)

	apiPath := "/repositories/{workspace}/{repo_slug}/src"
	if ref != "" {
		apiPath += "/" + ref
	}
	if pathParam != "" {
		apiPath += "/" + pathParam
	}

	values, err := client.ListAll(apiPath, 100)
	if err != nil {
		return mapError(err), nil
	}

	return jsonResult(values)
}

// HandleSourceView implements the bitbucket_source_view tool.
func (s *Server) HandleSourceView(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	pathParam := req.GetString("path", "")
	if pathParam == "" {
		return mcp.NewToolResultError("path is required"), nil
	}

	ref := req.GetString("ref", "")

	client := s.scopedClient(workspace, repo)

	apiPath := "/repositories/{workspace}/{repo_slug}/src"
	if ref != "" {
		apiPath += "/" + ref
	}
	apiPath += "/" + pathParam

	raw, err := client.Request("GET", apiPath, nil)
	if err != nil {
		return mapError(err), nil
	}

	// Source view returns file content; return as-is.
	return mcp.NewToolResultText(string(raw)), nil
}
