// Command bbmcp runs the Bitbucket MCP server over stdio transport.
// It resolves authentication from the bbcli auth chain and exposes
// Bitbucket operations as MCP tools.
package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/ashrocket/bbcli/internal/api"
	"github.com/ashrocket/bbcli/internal/auth"
	"github.com/ashrocket/bbcli/internal/git"
	mcpserver "github.com/ashrocket/bbcli/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Best-effort git detection — don't fail if not in a repo.
	var workspace, repo string
	if info, err := git.DetectRemote(); err == nil {
		workspace = info.Workspace
		repo = info.Repo
	}

	// Resolve auth from the 4-level chain. Best-effort: if no token
	// is found, the server still starts — individual tool calls will
	// fail with a clear error when they need auth.
	var authenticator api.Authenticator
	if result, err := auth.Resolve("", repo); err == nil {
		authenticator = &authAdapter{result: result}
	} else {
		// No auth available — use a no-op authenticator that will
		// cause API calls to fail with 401 (which maps to a clear error).
		authenticator = &noopAuth{}
	}

	client := api.New(authenticator)
	srv := mcpserver.New(client, mcpserver.WithDefaults(workspace, repo))

	if err := server.ServeStdio(srv.MCPServer()); err != nil {
		fmt.Fprintf(os.Stderr, "bbmcp: %v\n", err)
		os.Exit(1)
	}
}

// authAdapter wraps auth.Result to satisfy api.Authenticator.
type authAdapter struct {
	result *auth.Result
}

func (a *authAdapter) Authenticate(req *http.Request) error {
	switch a.result.Kind {
	case auth.BasicAuth:
		// Token is "user:password" — encode as Basic auth.
		encoded := base64.StdEncoding.EncodeToString([]byte(a.result.Token))
		req.Header.Set("Authorization", "Basic "+encoded)
	default:
		req.Header.Set("Authorization", "Bearer "+a.result.Token)
	}
	return nil
}

func (a *authAdapter) Source() string {
	return a.result.Source.String()
}

func (a *authAdapter) Level() int {
	return int(a.result.Source)
}

// noopAuth is used when no token is available. API calls will fail
// with 401, which produces a clear error message.
type noopAuth struct{}

func (n *noopAuth) Authenticate(req *http.Request) error { return nil }
func (n *noopAuth) Source() string                        { return "none" }
func (n *noopAuth) Level() int                            { return 0 }
