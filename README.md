# bbcli

The missing Bitbucket Cloud toolkit — for you and your AI assistant.

**bbcli** ships two tools from one Go codebase:

- `bbcli` — a CLI with structured JSON output, semantic exit codes, and zero interactive prompts
- `bbmcp` — an MCP server that gives Claude Code, Cursor, and Windsurf native Bitbucket tools

Both share one auth setup, one config, and identical behavior. What `gh` is to GitHub, bbcli is to Bitbucket — built for the age of AI-assisted development.

> **First Bitbucket MCP server.** No Bitbucket MCP server exists. bbmcp fills that gap.

## Install

```bash
# Requires Go 1.21+
go install github.com/ashrocket/bbcli/cmd/bbcli@latest
go install github.com/ashrocket/bbcli/cmd/bbmcp@latest
```

Make sure `~/go/bin` is in your PATH:
```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Quick Start

```bash
# Authenticate (one time, works for both CLI and MCP)
bbcli auth login

# Set your default workspace
bbcli config set defaults.workspace myworkspace

# List PRs
bbcli pr list -r my-repo

# Create a PR
bbcli pr create --title "Fix auth bug" --dest dev -r my-repo

# Check pipelines
bbcli pipeline list -r my-repo
bbcli pipeline status 42 -r my-repo

# Watch a pipeline until it finishes
bbcli pipeline watch -r my-repo

# Hit any Bitbucket API endpoint directly
bbcli api /user
```

## Commands

```
bbcli pr list              List pull requests
bbcli pr create            Create a pull request
bbcli pr view <ID>         View PR details
bbcli pr approve <ID>      Approve a PR
bbcli pr decline <ID>      Decline a PR
bbcli pr merge <ID>        Merge a PR

bbcli pipeline list        List recent pipelines
bbcli pipeline status <N>  Show pipeline steps
bbcli pipeline watch [N]   Watch until completion

bbcli branch delete <name> Delete a remote branch

bbcli auth login           Store credentials
bbcli auth status          Show auth chain

bbcli config set <k> <v>   Set config value
bbcli config get <k>       Get config value
bbcli config list          Show all config

bbcli api <path>           Authenticated API request
```

Every command supports `--json`, `--output minimal`, and `--output table` (default).

## Authentication

bbcli resolves auth from (in priority order):

1. `BBCLI_TOKEN` environment variable
2. `--token` flag
3. `~/.config/bbcli/credentials` (stored by `bbcli auth login`)
4. Legacy `~/.bb-cli-token-*` files

```bash
# Option A: Environment variable (recommended for CI/agents)
export BBCLI_TOKEN="your-bitbucket-token"

# Option B: Persistent login
bbcli auth login --token your-bitbucket-token

# Option C: Interactive login (prompts for token)
bbcli auth login
```

Supports both Bearer tokens (workspace/repo access tokens) and Basic auth (email:app_password format).

## For AI Agents (CLI)

```bash
# Set once in agent environment
export BBCLI_TOKEN="your-api-token"
export BBCLI_OUTPUT="json"

# Create a PR (idempotent — safe to retry)
PR_ID=$(bbcli pr create --title "Fix auth" --dest dev --output minimal)

# Check CI
bbcli pipeline watch --timeout 600

# Merge
bbcli pr merge "$PR_ID"
```

Every error includes a machine-readable code and `retryable` flag:
```json
{
  "error": {
    "code": "RATE_LIMITED",
    "message": "API rate limit exceeded. Retry after 30 seconds.",
    "suggestion": "Wait and retry, or reduce request frequency",
    "retryable": true
  }
}
```

## For AI Agents (MCP)

Give your AI coding assistant native Bitbucket tools:

```bash
# Claude Code — one command setup
claude mcp add bitbucket -- bbmcp
```

Or add to `.mcp.json`:
```json
{
  "mcpServers": {
    "bitbucket": {
      "command": "bbmcp",
      "env": { "BBCLI_TOKEN": "your-api-token" }
    }
  }
}
```

22 tools available: `bitbucket_pr_create`, `bitbucket_pr_list`, `bitbucket_pipeline_status`, `bitbucket_source_view`, `bitbucket_api`, and more.

## Exit Codes

| Code | Name | Agent Action |
|------|------|-------------|
| 0 | Success | Proceed |
| 1 | General error | Log and escalate |
| 2 | Usage error | Fix invocation |
| 3 | Auth error | Refresh credentials |
| 4 | Not found | Check resource ID |
| 5 | State error | Check current state |
| 6 | Rate limited | Back off and retry |
| 7 | Network/API error | Retry or escalate |

## Auto-Detection

bbcli auto-detects your workspace and repo from the git remote in your current directory. No need to pass `-w` and `-r` flags when working inside a Bitbucket repo.

```bash
cd ~/code/my-bitbucket-repo
bbcli pr list    # just works — workspace and repo detected from git remote
```

Override with flags: `bbcli pr list -w other-workspace -r other-repo`

## License

MIT
