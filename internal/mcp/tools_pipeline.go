package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerPipelineTools registers all pipeline-related tools.
func (s *Server) registerPipelineTools() {
	// --- bitbucket_pipeline_list ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pipeline_list",
		mcp.WithDescription("List recent pipelines for a Bitbucket repository. "+
			"Optionally filter by branch."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("branch",
			mcp.Description("Filter pipelines by branch name."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of pipelines to return (default 25, max 100)."),
			mcp.DefaultNumber(25),
			mcp.Min(1),
			mcp.Max(100),
		),
	), s.HandlePipelineList)

	// --- bitbucket_pipeline_status ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pipeline_status",
		mcp.WithDescription("Get the status and details of a specific pipeline build."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("build_number",
			mcp.Description("Pipeline build number."),
			mcp.Required(),
		),
	), s.HandlePipelineStatus)

	// --- bitbucket_pipeline_watch ---
	s.mcp.AddTool(mcp.NewTool("bitbucket_pipeline_watch",
		mcp.WithDescription("Poll a pipeline until it completes or times out. "+
			"Returns final status. Max timeout 120 seconds for MCP."),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug. Uses auto-detected default if omitted."),
		),
		mcp.WithString("repo",
			mcp.Description("Repository slug. Uses auto-detected default if omitted."),
		),
		mcp.WithNumber("build_number",
			mcp.Description("Pipeline build number."),
			mcp.Required(),
		),
		mcp.WithNumber("timeout_seconds",
			mcp.Description("Maximum seconds to wait (default 60, max 120)."),
			mcp.DefaultNumber(60),
			mcp.Min(5),
			mcp.Max(120),
		),
	), s.HandlePipelineWatch)
}

// HandlePipelineList implements the bitbucket_pipeline_list tool.
func (s *Server) HandlePipelineList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	path := "/repositories/{workspace}/{repo_slug}/pipelines/?sort=-created_on"
	if branch != "" {
		path += fmt.Sprintf("&target.branch=%s", branch)
	}

	values, err := client.ListAll(path, limit)
	if err != nil {
		return mapError(err), nil
	}

	return jsonResult(values)
}

// HandlePipelineStatus implements the bitbucket_pipeline_status tool.
func (s *Server) HandlePipelineStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	buildNumber := req.GetInt("build_number", 0)
	if buildNumber == 0 {
		return mcp.NewToolResultError("build_number is required"), nil
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pipelines/%d", buildNumber)
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

// HandlePipelineWatch implements the bitbucket_pipeline_watch tool.
func (s *Server) HandlePipelineWatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workspace, repo, errResult := s.resolveWorkspaceRepo(req)
	if errResult != nil {
		return errResult, nil
	}

	buildNumber := req.GetInt("build_number", 0)
	if buildNumber == 0 {
		return mcp.NewToolResultError("build_number is required"), nil
	}

	timeout := req.GetInt("timeout_seconds", 60)
	if timeout < 5 {
		timeout = 5
	}
	if timeout > 120 {
		timeout = 120
	}

	client := s.scopedClient(workspace, repo)
	path := fmt.Sprintf("/repositories/{workspace}/{repo_slug}/pipelines/%d", buildNumber)

	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	pollInterval := 5 * time.Second

	for {
		raw, err := client.Request("GET", path, nil)
		if err != nil {
			return mapError(err), nil
		}

		var result map[string]any
		if err := json.Unmarshal(raw, &result); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		// Check if pipeline has completed.
		if state, ok := result["state"].(map[string]any); ok {
			if name, _ := state["name"].(string); name == "COMPLETED" {
				return jsonResult(result)
			}
		}

		if time.Now().After(deadline) {
			// Return current state with a note about timeout.
			result["_watch_status"] = "TIMED_OUT"
			result["_watch_message"] = fmt.Sprintf("Pipeline still running after %d seconds", timeout)
			return jsonResult(result)
		}

		select {
		case <-ctx.Done():
			result["_watch_status"] = "CANCELLED"
			return jsonResult(result)
		case <-time.After(pollInterval):
			// Continue polling.
		}
	}
}
