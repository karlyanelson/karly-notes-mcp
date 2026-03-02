# karly-notes

An AI sandbox repo for exploring different ways to give AI agents persistent note-taking capabilities.

The same core note storage (add, get, list, search, delete) is exposed through two different approaches:

1. **MCP Server** (`mcp/`) — A Model Context Protocol server that Claude and other MCP-compatible agents can call directly as tools.
2. **Agent Skills + CLI** (`skills-playground/`) — A Go CLI with the same commands, invoked by agent skill markdown files following the [Agent Skills](https://agentskills.io) open format.

Both share a common storage library (`common/`) so the note logic is written once.

## Project Structure

```
.
├── common/                          # Shared Go package (Note, Store, FileStore)
│   ├── store.go
│   ├── store_test.go
│   └── go.mod
├── mcp/                             # MCP server (tools for Claude)
│   ├── main.go
│   ├── tools.go
│   ├── main_test.go
│   ├── README.md
│   └── go.mod
├── skills-playground/               # Agent Skills approach
│   ├── cli/                         # Go CLI with subcommands
│   │   ├── main.go
│   │   └── go.mod
│   ├── scripts/                     # Wrapper script for CLI invocation
│   │   └── run.sh
│   ├── skills/                      # SKILL.md files for each operation
│   │   ├── add-note/
│   │   ├── get-note/
│   │   ├── list-notes/
│   │   ├── delete-note/
│   │   ├── search-notes/
│   │   └── list-notes-for-repo/
│   └── README.md
└── go.work                          # Go workspace (ties modules together)
```

## Getting Started

Requires Go 1.24+.

```bash
# Build everything from the repo root
go build -o mcp/karly-notes-mcp ./mcp/
go build -o skills-playground/cli/karly-notes-cli ./skills-playground/cli/

# Run all tests
go test ./...
```

See each subdirectory's README for setup details:

- [`mcp/README.md`](mcp/README.md) — How to register the MCP server with Claude Code
- [`skills-playground/README.md`](skills-playground/README.md) — How the CLI + agent skills approach works
