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

### 2. Add the CLI to your PATH

Add the build output directory to your shell profile so the binary is always available after a rebuild:

```bash
# Add to ~/.zshrc (or ~/.bashrc)
export PATH="$HOME/path/to/karly-notes-mcp/skills-playground/karly-notes-cli:$PATH"
```

Then reload your shell (`source ~/.zshrc`) or open a new terminal.

### 3. Install the skills as a Claude Code plugin

This directory is a Claude Code plugin. Installing it once makes the skills available in every project, and they stay up to date automatically when you `git pull`.

Run these commands in a terminal after typing `claude` (You need claude code CLI installed)

```bash
# Add this repo as a local plugin marketplace
/plugin marketplace add /path/to/karly-notes-mcp/skills-playground

# Then open the plugin manager and install "karly-notes"
/plugin
```

After installation, skills are namespaced as `/karly-notes:<skill-name>` (e.g. `/karly-notes:add-note`).

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

## Updating the Plugin After Local Changes

When you edit skill files (e.g. `skills/add-note/SKILL.md`) or the CLI, the changes only exist in your local workspace. The installed plugin runs from a **cached copy** at `~/.claude/plugins/cache/karly-notes/`. To pick up your changes:

1. **Commit your changes** — the plugin cache is pinned to a git commit SHA, so uncommitted changes won't be picked up.

   ```bash
   git add -A && git commit -m "update skill instructions"
   ```

2. **Update the plugin** — in a Claude Code session, run:

   ```
   /plugin
   ```

   Then select the `karly-notes` plugin and choose **Update**. This re-caches the plugin from your latest commit.

3. **Restart Claude Code** — start a new session so the updated skill instructions are loaded.

If you've bumped the version in `marketplace.json` and `.claude-plugin/plugin.json`, Claude Code may prompt you to update automatically. For local development, keeping the version the same and manually updating is usually easiest.

## Further Reading

- [What are Agent Skills?](https://agentskills.io/what-are-skills)
- [Agent Skills Specification](https://agentskills.io/specification)
- [Using Scripts in Skills](https://agentskills.io/skill-creation/using-scripts)
