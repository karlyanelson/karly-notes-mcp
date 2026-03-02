# skills-playground

An alternative to the MCP server approach — same note-taking capabilities, but delivered through [Agent Skills](https://agentskills.io) instead of MCP tools.

## How it works

Instead of an MCP server that Claude discovers and calls tools on, this approach uses:

1. **A CLI** (`karly-notes-cli/`) — A Go command-line tool with subcommands (`add`, `get`, `list`, `delete`, `search`, `list-by-repo`) that read/write the same `~/.karly-notes/notes.json` file as the MCP server.

2. **Skill files** (`skills/`) — Markdown files following the [Agent Skills spec](https://agentskills.io/specification) that tell an agent *when* and *how* to use the CLI commands.

### Agent Skills in a nutshell

Agent Skills are an open format for giving AI agents new capabilities. Each skill is a folder with a `SKILL.md` file containing:

- **YAML frontmatter** (`name`, `description`) — loaded at startup so the agent knows the skill exists
- **Markdown body** — full instructions loaded on-demand when the agent decides to use the skill

This is a "progressive disclosure" pattern: the agent sees short descriptions for all skills upfront, then loads full instructions only when relevant to the current task.

### MCP vs Skills — what's different?

| | MCP Server | Agent Skills |
|---|---|---|
| **How the agent calls it** | Built-in MCP tool protocol | Runs a shell command via Bash |
| **Discovery** | Agent sees tool schemas at session start | Agent sees skill name + description at session start |
| **Execution** | Typed RPC over stdin/stdout | Agent runs `karly-notes-cli <command>` in a shell |
| **Output format** | MCP CallToolResult | JSON on stdout, errors on stderr |
| **Setup** | `claude mcp add` registration | Drop skill folders where the agent looks for them |
| **Portability** | Works with MCP-compatible agents | Works with any agent that supports the Agent Skills format |

Both approaches share the same underlying storage via the `common/` package.

## Setup

### 1. Build the CLI

From the repo root:

```bash
go build -o skills-playground/karly-notes-cli/karly-notes-cli ./skills-playground/karly-notes-cli/
```

### 2. Make the CLI available on your PATH

Either add the build output to your PATH or copy it somewhere that's already on your PATH:

```bash
cp skills-playground/karly-notes-cli/karly-notes-cli /usr/local/bin/
```

### 3. Point your agent at the skills

How you do this depends on your agent. For Claude Code, you can place skill folders in `.claude/skills/` in your project root, or reference them from your agent's skill discovery configuration.

## CLI Usage

```bash
karly-notes-cli add --title "JWT gotcha" --content "Normalize to UTC before comparing expiry"
karly-notes-cli get --title "JWT gotcha"
karly-notes-cli list
karly-notes-cli search --keyword "JWT"
karly-notes-cli delete --title "old note"
karly-notes-cli list-by-repo --repo "github.com/user/repo"
```

All commands output JSON to stdout. Errors go to stderr with a non-zero exit code.

## Skill Files

Each skill maps to one CLI command:

| Skill | CLI Command |
|-------|-------------|
| `skills/add-note/` | `karly-notes-cli add` |
| `skills/get-note/` | `karly-notes-cli get` |
| `skills/list-notes/` | `karly-notes-cli list` |
| `skills/delete-note/` | `karly-notes-cli delete` |
| `skills/search-notes/` | `karly-notes-cli search` |
| `skills/list-notes-for-repo/` | `karly-notes-cli list-by-repo` |

## Further Reading

- [What are Agent Skills?](https://agentskills.io/what-are-skills)
- [Agent Skills Specification](https://agentskills.io/specification)
- [Using Scripts in Skills](https://agentskills.io/skill-creation/using-scripts)
