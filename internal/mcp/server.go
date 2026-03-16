// Package mcp wraps the MCP server for bbcli, registering tools
// that delegate to the Bitbucket API client.
package mcp

import (
	"github.com/ashrocket/bbcli/internal/api"
	"github.com/mark3labs/mcp-go/server"
)

// Server holds the MCP server and the underlying Bitbucket API client.
type Server struct {
	mcp    *server.MCPServer
	client *api.Client

	// defaults used when tool params omit workspace/repo
	defaultWorkspace string
	defaultRepo      string
}

// Option configures the MCP server.
type Option func(*Server)

// WithDefaults sets the fallback workspace and repo used when tool
// parameters do not explicitly supply them.
func WithDefaults(workspace, repo string) Option {
	return func(s *Server) {
		s.defaultWorkspace = workspace
		s.defaultRepo = repo
	}
}

// New creates an MCP server with all tools registered.
// The api.Client handles auth, retries, and pagination.
func New(client *api.Client, opts ...Option) *Server {
	s := &Server{
		mcp:    server.NewMCPServer("bbmcp", "0.1.0"),
		client: client,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.registerTools()
	return s
}

// MCPServer returns the underlying mcp-go server (for ServeStdio).
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcp
}

// registerTools wires up every tool the MCP server exposes.
func (s *Server) registerTools() {
	s.registerPRTools()
	s.registerPipelineTools()
	s.registerRepoTools()
	s.registerMiscTools()
}
