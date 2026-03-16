package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ashrocket/bbcli/internal/api"
	bbErrors "github.com/ashrocket/bbcli/internal/errors"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerPRTools registers all pull-request related tools.
func (s *Server) registerPRTools() {
	// --- bitbucket_pr_list ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_list",
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
	), s.HandlePRList)

	// --- bitbucket_pr_create ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_create",
		mcp.WithDescription("Create a new pull request in a Bitbucket repository."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("title",
			mcp.Description("Pull request title."),
			mcp.Required(),
		),
		mcp.WithString("source",
			mcp.Description("Source branch name. Defaults to current branch if omitted."),
		),
		mcp.WithString("destination",
			mcp.Description("Destination branch name. Defaults to repository main branch if omitted."),
		),
		mcp.WithString("description",
			mcp.Description("Pull request description body."),
		),
		mcp.WithString("reviewers",
			mcp.Description("Comma-separated list of reviewer UUIDs or usernames."),
		),
		mcp.WithBoolean("close_source_branch",
			mcp.Description("Whether to close the source branch after merge."),
			mcp.DefaultBool(false),
		),
	), s.HandlePRCreate)

	// --- bitbucket_pr_view ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_view",
		mcp.WithDescription("View details of a specific pull request."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID."),
			mcp.Required(),
		),
	), s.HandlePRView)

	// --- bitbucket_pr_approve ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_approve",
		mcp.WithDescription("Approve a pull request."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID."),
			mcp.Required(),
		),
	), s.HandlePRApprove)

	// --- bitbucket_pr_decline ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_decline",
		mcp.WithDescription("Decline a pull request."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID."),
			mcp.Required(),
		),
		mcp.WithString("reason",
			mcp.Description("Reason for declining the PR."),
		),
	), s.HandlePRDecline)

	// --- bitbucket_pr_merge ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_merge",
		mcp.WithDescription("Merge a pull request."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID."),
			mcp.Required(),
		),
		mcp.WithString("strategy",
			mcp.Description("Merge strategy."),
			mcp.Enum("merge_commit", "squash", "fast_forward"),
		),
		mcp.WithString("message",
			mcp.Description("Merge commit message."),
		),
		mcp.WithBoolean("close_source_branch",
			mcp.Description("Whether to close the source branch after merge."),
			mcp.DefaultBool(false),
		),
	), s.HandlePRMerge)

	// --- bitbucket_pr_diff ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_diff",
		mcp.WithDescription("Get the diff of a pull request."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID."),
			mcp.Required(),
		),
	), s.HandlePRDiff)

	// --- bitbucket_pr_comments ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_comments",
		mcp.WithDescription("List comments on a pull request."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID."),
			mcp.Required(),
		),
	), s.HandlePRComments)

	// --- bitbucket_pr_comment ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pr_comment",
		mcp.WithDescription("Add a comment to a pull request."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID."),
			mcp.Required(),
		),
		mcp.WithString("body",
			mcp.Description("Comment body text."),
			mcp.Required(),
		),
	), s.HandlePRComment)
}

// scopedClient builds a scoped API client for the given workspace/repo,
// preserving the parent client's base URL and auth.
func (s *Server) scopedClient(workspace, repo string) *api.Client {
	return api.NewWithBaseURL(s.client.Auth(), s.client.BaseURL(),
		api.WithWorkspaceRepo(workspace, repo),
	)
}

// resolveWorkspaceRepo extracts workspace/repo from the request, falling
// back to server defaults. Returns an error result if either is missing.
func (s *Server) resolveWorkspaceRepo(req mcp.CallToolRequest) (string, string, *mcp.CallToolResult) {
	workspace := req.GetString("workspace", s.defaultWorkspace)
	repo := req.GetString("repo", s.defaultRepo)
	if workspace == "" {
		return "", "", mcp.NewToolResultError("workspace is required: pass it as a parameter or run from a git repo with a Bitbucket remote")
	}
	if repo == "" {
		return "", "", mcp.NewToolResultError("repo is required: pass it as a parameter or run from a git repo with a Bitbucket remote")
	}
	return workspace, repo, nil
}

// jsonResult marshals v as indented JSON and returns it as a tool result text.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// HandlePRList implements the bitbucket_pr_list tool.
// Exported so tests can call it directly.
func (s *Server) HandlePRList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}
	state := req.GetString("state", "OPEN")
	limit := req.GetInt("limit", 25)

	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	client := s.scopedClient(workspace, repo)

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

	return jsonResult(prs)
}

// HandlePRCreate implements the bitbucket_pr_create tool.
func (s *Server) HandlePRCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	title := req.GetString("title", "")
	if title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	source := req.GetString("source", "")
	destination := req.GetString("destination", "")
	description := req.GetString("description", "")
	reviewersStr := req.GetString("reviewers", "")
	closeSource := req.GetBool("close_source_branch", false)

	body := map[string]any{
		"title":               title,
		"close_source_branch": closeSource,
	}
	if description != "" {
		body["description"] = description
	}
	if source != "" {
		body["source"] = map[string]any{
			"branch": map[string]string{"name": source},
		}
	}
	if destination != "" {
		body["destination"] = map[string]any{
			"branch": map[string]string{"name": destination},
		}
	}
	if reviewersStr != "" {
		var reviewers []map[string]string
		for _, r := range strings.Split(reviewersStr, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				reviewers = append(reviewers, map[string]string{"username": r})
			}
		}
		if len(reviewers) > 0 {
			body["reviewers"] = reviewers
		}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build request body: %v", err)), nil
	}

	client := s.scopedClient(workspace, repo)
	raw, err := client.Request("POST", "/repositories/{workspace}/{repo_slug}/pullrequests", payload)
	if err != nil {
		return mapError(err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	return jsonResult(result)
}

// HandlePRView implements the bitbucket_pr_view tool.
func (s *Server) HandlePRView(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID == 0 {
		return mcp.NewToolResultError("pr_id is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d", prID)
	raw, err := client.Request("GET", path, nil)
	if err != nil {
		return mapError(err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	return jsonResult(result)
}

// HandlePRApprove implements the bitbucket_pr_approve tool.
func (s *Server) HandlePRApprove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID == 0 {
		return mcp.NewToolResultError("pr_id is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/approve", prID)
	raw, err := client.Request("POST", path, nil)
	if err != nil {
		return mapError(err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	return jsonResult(result)
}

// HandlePRDecline implements the bitbucket_pr_decline tool.
func (s *Server) HandlePRDecline(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID == 0 {
		return mcp.NewToolResultError("pr_id is required"), nil
	}

	reason := req.GetString("reason", "")
	var payload []byte
	if reason != "" {
		body := map[string]string{"message": reason}
		payload, _ = json.Marshal(body)
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/decline", prID)
	raw, err := client.Request("POST", path, payload)
	if err != nil {
		return mapError(err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	return jsonResult(result)
}

// HandlePRMerge implements the bitbucket_pr_merge tool.
func (s *Server) HandlePRMerge(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID == 0 {
		return mcp.NewToolResultError("pr_id is required"), nil
	}

	strategy := req.GetString("strategy", "")
	message := req.GetString("message", "")
	closeSource := req.GetBool("close_source_branch", false)

	body := map[string]any{
		"close_source_branch": closeSource,
	}
	if strategy != "" {
		body["merge_strategy"] = strategy
	}
	if message != "" {
		body["message"] = message
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build request body: %v", err)), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/merge", prID)
	raw, err := client.Request("POST", path, payload)
	if err != nil {
		return mapError(err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	return jsonResult(result)
}

// HandlePRDiff implements the bitbucket_pr_diff tool.
func (s *Server) HandlePRDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID == 0 {
		return mcp.NewToolResultError("pr_id is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/diff", prID)
	raw, err := client.Request("GET", path, nil)
	if err != nil {
		return mapError(err), nil
	}

	// Diff comes back as plain text in the JSON raw response; return as-is.
	return mcp.NewToolResultText(string(raw)), nil
}

// HandlePRComments implements the bitbucket_pr_comments tool.
func (s *Server) HandlePRComments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID == 0 {
		return mcp.NewToolResultError("pr_id is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/comments", prID)
	values, err := client.ListAll(path, 100)
	if err != nil {
		return mapError(err), nil
	}

	return jsonResult(values)
}

// HandlePRComment implements the bitbucket_pr_comment tool.
func (s *Server) HandlePRComment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	prID := req.GetInt("pr_id", 0)
	if prID == 0 {
		return mcp.NewToolResultError("pr_id is required"), nil
	}

	body := req.GetString("body", "")
	if body == "" {
		return mcp.NewToolResultError("body is required"), nil
	}

	payload, err := json.Marshal(map[string]any{
		"content": map[string]string{"raw": body},
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build request body: %v", err)), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pullrequests/%d/comments", prID)
	raw, err := client.Request("POST", path, payload)
	if err != nil {
		return mapError(err), nil
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
	}

	return jsonResult(result)
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
