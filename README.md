# karly-notes-mcp

A personal "second brain" MCP server that lets an AI agent save, retrieve, search, and manage notes on your behalf. Built with the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk).

Think of it as a scratchpad for things too small for a wiki but too important to lose.

## Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `add_note` | Save a note (or update an existing one by title) | `title` (string), `content` (string) |
| `get_note` | Retrieve a note by its exact title | `title` (string) |
| `list_notes` | List all saved notes with titles and creation dates | _(none)_ |
| `delete_note` | Remove a note by title | `title` (string) |
| `search_notes` | Find notes matching a keyword (case-insensitive, searches title + content) | `keyword` (string) |

## Build & Run

Requires Go 1.24+.

```bash
go build -o karly-notes-mcp .
```

The server communicates over stdio using the MCP JSON-RPC protocol. You don't run it directly — your MCP client (Claude Code, VS Code, etc.) launches it.

## Configuration

### Claude Code (CLI — recommended)

Claude Code has a built-in command for registering MCP servers — no JSON editing required:

```bash
claude mcp add --transport stdio karly-notes -- /Users/karlynelson/workspace/karly-notes-mcp/karly-notes-mcp
```

Verify it was added:

```bash
claude mcp list
```

Check live status from inside a Claude Code session:

```
/mcp
```

Remove it if needed:

```bash
claude mcp remove karly-notes
```

**Scope options** (pass `--scope <value>` to `mcp add`):

| Scope | Where it's saved | Use when |
|-------|-----------------|----------|
| `local` (default) | `~/.claude.json` | Private to you in the current project |
| `user` | `~/.claude.json` | Available across all your projects |
| `project` | `.mcp.json` in repo | Shared with teammates via version control |

### Claude Code (manual JSON)

If you prefer to edit config directly, add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "karly-notes": {
      "command": "/Users/karlynelson/workspace/karly-notes-mcp/karly-notes-mcp"
    }
  }
}
```

### VS Code (Claude extension)

Add to your MCP settings:

```json
{
  "mcp": {
    "servers": {
      "karly-notes": {
        "command": "/Users/karlynelson/workspace/karly-notes-mcp/karly-notes-mcp"
      }
    }
  }
}
```

## Usage Examples

### Capturing a debugging insight

You're debugging a tricky authentication flow. You finally figure out why JWT tokens were expiring unexpectedly — it was a timezone issue in the token validation logic. You don't want to forget this.

You just say to Claude:

> "Save a note called 'JWT timezone gotcha' — when validating tokens, always normalize to UTC before comparing expiry. The `iat` claim comes in as local time on Windows but UTC on Linux, which causes flaky auth failures in dev vs prod."

Claude calls `add_note` behind the scenes. Done.

### Recalling it later

Two weeks later, you're onboarding a new project with auth and you vaguely remember hitting this before:

> "Do I have any notes about JWT or authentication?"

Claude calls `search_notes` with "JWT", finds it, and surfaces exactly what you wrote.

### Other real engineering uses

- **Tribal knowledge**: Store snippets your team hasn't documented ("the staging DB needs a VPN + this specific SSH key")
- **Decision log**: Save "why did I do it this way" decisions next to nothing in the codebase
- **Quick scratchpad**: Things too small for a wiki but too important to lose
- **Error solutions**: Save error messages + their solutions as you encounter them
- **Meeting notes**: Quick capture of action items or decisions from standups
- **Code recipes**: Save snippets you keep looking up ("Go context timeout pattern", "React useEffect cleanup")

### More example interactions

> "Save a note called 'Docker compose gotcha' — if you get `bind: address already in use`, run `lsof -i :PORT` to find the zombie process"

> "What notes do I have?"

> "Find my notes about Docker"

> "Delete the note called 'old todo list'"

## Running Tests

```bash
go test -v ./...
```

## Current Limitations

- Notes are stored in memory and lost when the server restarts
- No tags or categories (search by keyword only)
- No import/export

## Future Design: File-Based Persistence

The current in-memory store is great for getting started, but notes disappear when the server restarts. The next step is persisting notes to disk.

### Approach: Single JSON File

Store all notes in one file at a well-known location:

```
~/.karly-notes/notes.json
```

**Why a single file instead of one file per note:**

- Simpler to back up, sync, or version control (one file to track)
- Atomic operations are easier — write the whole file in one go
- No filename sanitization issues (note titles can contain anything)
- For the expected volume (tens to hundreds of notes), a single file is perfectly fast
- Easier to inspect and manually edit if needed

**When to consider splitting into multiple files:**

If the note count grows into thousands, or if individual notes contain very large content (embedded images, long logs), a directory-per-note structure would avoid rewriting the entire file on every change. But that's a bridge to cross later.

### File Format

```json
{
  "version": 1,
  "notes": {
    "JWT timezone gotcha": {
      "title": "JWT timezone gotcha",
      "content": "When validating tokens, always normalize to UTC...",
      "created_at": "2025-03-15T10:30:00Z",
      "updated_at": "2025-03-15T10:30:00Z"
    },
    "Docker compose gotcha": {
      "title": "Docker compose gotcha",
      "content": "If you get bind: address already in use...",
      "created_at": "2025-03-16T14:20:00Z",
      "updated_at": "2025-03-16T14:20:00Z"
    }
  }
}
```

The `version` field allows migrating the format later without breaking existing files.

### Implementation Plan

1. **`FileStore` struct** that implements the same interface as the current in-memory `NoteStore`
2. **Load on startup**: read `~/.karly-notes/notes.json` into memory (create with empty notes if missing)
3. **Write on mutation**: after every `add_note` or `delete_note`, write the full file back to disk
4. **Atomic writes**: write to a temp file first, then rename — prevents corruption if the process is killed mid-write
5. **Configurable path**: allow overriding the notes directory via a `--notes-dir` flag or `KARLY_NOTES_DIR` environment variable

### Atomic Write Strategy

```
write to ~/.karly-notes/notes.json.tmp
fsync
rename ~/.karly-notes/notes.json.tmp -> ~/.karly-notes/notes.json
```

This ensures the file is never in a half-written state. The rename is atomic on all major filesystems.

### Migration Path

The switch from in-memory to file-based is a one-line change in `main()` — swap `NewNoteStore()` for `NewFileStore(path)`. Both implement the same methods, so no tool handler changes are needed.
