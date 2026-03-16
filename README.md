# bbcli

The missing Bitbucket Cloud toolkit — for you and your AI assistant.

**bbcli** ships two tools from one Go codebase:

- `bbcli` — a CLI with structured JSON output, semantic exit codes, and zero interactive prompts
- `bbmcp` — an MCP server that gives Claude Code, Cursor, and Windsurf native Bitbucket tools

Both share one auth setup, one config, and identical behavior. What `gh` is to GitHub, bbcli is to Bitbucket — built for the age of AI-assisted development.

> **First Bitbucket MCP server.** No Bitbucket MCP server exists today. bbmcp fills that gap.

## Quick Start

```bash
# Install (gets you both bbcli and bbmcp)
brew install ashrocket/tap/bbcli

# Or build from source
go install github.com/ashrocket/bbcli/cmd/bbcli@latest
go install github.com/ashrocket/bbcli/cmd/bbmcp@latest

# Authenticate (one time, works for both)
bbcli auth login

# Use the CLI
bbcli pr create --title "Fix auth bug" --dest dev
bbcli pr list --json
bbcli pipeline watch 42

# Or give your AI assistant native Bitbucket tools
claude mcp add bitbucket -- bbmcp
```

## Why bbcli?

| Problem | bbcli Solution |
|---------|---------------|
| No Bitbucket CLI for AI agents | Every command has `--json` output and semantic exit codes |
| No Bitbucket MCP server | `bbmcp` exposes 38 curated tools for AI assistants |
| App passwords dying June 2026 | Built for API tokens from day one |
| Existing CLIs are unmaintained | Active development, open source (MIT) |

## For AI Agents (CLI)

```bash
# Set once in agent environment
export BBCLI_TOKEN="your-api-token"
export BBCLI_OUTPUT="json"

# Create a PR (idempotent — safe to retry)
PR_ID=$(bbcli pr create --title "Fix auth" --dest dev --output minimal)

# Check CI
PIPELINE=$(bbcli pipeline list --limit 1 --output minimal)
bbcli pipeline watch "$PIPELINE" --timeout 600

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

Add to your `.mcp.json` (Claude Code, Cursor, etc.):

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

38 tools available instantly — `bitbucket_pr_create`, `bitbucket_pipeline_status`, `bitbucket_source_view`, and more. No output parsing. No exit code checking. Native tool calls.

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

## Status

**Under active development.** Core PR and pipeline commands are being implemented first. The `bbcli api` escape hatch provides authenticated access to any Bitbucket endpoint in the meantime.

See [DESIGN.md](https://github.com/ashrocket/bbcli/issues/17) for the full specification.

## License

MIT
