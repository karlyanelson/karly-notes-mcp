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
claude mcp add --transport stdio karly-notes -- /absolute/path/to/binary/karly-notes-mcp/karly-notes-mcp
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
      "command": "/absolute/path/to/binary/karly-notes-mcp/karly-notes-mcp"
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
        "command": "/absolute/path/to/binary/karly-notes-mcp/karly-notes-mcp"
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

> Go has testing built in — no need to install Jest, Vitest, or any test library. The `go test` command is part of the language itself, the same way `node` is. Test files live right next to the source files they test (not in a separate `__tests__` folder), and any file ending in `_test.go` is automatically picked up.

### Run all tests

```bash
go test -v .
```

The flags:
- `-v` — "verbose": print each test name and whether it passed or failed (without this you only see a summary, like running Jest without `--verbose`)
- `.` — "this package": run tests in the current directory

### What the output looks like

```
=== RUN   TestAddAndGetNote
--- PASS: TestAddAndGetNote (0.00s)
=== RUN   TestSearchNotes
--- PASS: TestSearchNotes (0.00s)
=== RUN   TestFileStorePersistence
--- PASS: TestFileStorePersistence (0.00s)
PASS
ok  	karly-notes-mcp	0.17s
```

`PASS` at the bottom means everything passed. If a test fails you'll see `FAIL` with the file name and line number of the failing assertion.

### Run a single test by name

```bash
go test -v -run TestFileStorePersistence .
```

`-run` takes a pattern matched against test function names — handy when you're working on one thing and don't want to wait for the full suite (same idea as `jest -t "my test name"`).

### What's being tested

| Test | What it covers |
|------|---------------|
| `TestAddAndGetNote` | Saving a note and reading it back via the MCP tools |
| `TestListNotes` | Empty list response, then listing after notes are added |
| `TestDeleteNote` | Deleting an existing note; error on deleting one that doesn't exist |
| `TestSearchNotes` | Keyword match, no-match, and case-insensitive search |
| `TestGetNoteNotFound` | Error response when a note title doesn't exist |
| `TestFileStorePersistence` | Notes survive a "restart" (write → new instance → read), `created_at` is preserved on update, deletions persist to disk, JSON file is valid |

The first five tests use an in-memory store so they run instantly with no disk I/O. `TestFileStorePersistence` uses Go's `t.TempDir()` — a temporary folder that's automatically cleaned up after the test, so it never leaves anything on your machine.

## Current Limitations

- No tags or categories (search by keyword only)
- No import/export

## Persistence

Notes are stored in a JSON file on your local machine. The file is created automatically on first run.

### Default location

```
~/.karly-notes/notes.json
```

### Changing the location

Pass `--notes-dir` when registering the server:

```bash
claude mcp add --transport stdio karly-notes -- /path/to/karly-notes-mcp --notes-dir /path/to/your/notes
```

Or set the `KARLY_NOTES_DIR` environment variable:

```bash
claude mcp add --transport stdio karly-notes \
  --env KARLY_NOTES_DIR=/path/to/your/notes \
  -- /path/to/karly-notes-mcp
```

### File format

```json
{
  "version": 1,
  "notes": {
    "JWT timezone gotcha": {
      "title": "JWT timezone gotcha",
      "content": "When validating tokens, always normalize to UTC before comparing expiry.",
      "created_at": "2025-03-15T10:30:00Z",
      "updated_at": "2025-03-15T10:30:00Z"
    }
  }
}
```

The file is plain JSON — you can read, edit, back up, or sync it with any tool you like (iCloud Drive, Dropbox, git, etc.). The `version` field is reserved for future format migrations.

### How writes work

Every mutation (`add_note`, `delete_note`) writes to disk atomically:

```
write → ~/.karly-notes/notes.json.tmp
rename → ~/.karly-notes/notes.json
```

The rename is atomic on all major filesystems, so the file is never left in a half-written state even if the process is killed mid-write.
