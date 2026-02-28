# karly-notes-mcp

A personal "second brain" MCP server that lets an AI agent save, retrieve, search, and manage notes on your behalf. Built with the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk).

Think of it as a scratchpad for things too small for a wiki but too important to lose.

## Table of Contents

- [How it works](#how-it-works)
- [Setup](#setup)
- [Tools](#tools)
- [Build & Run](#build--run)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Running Tests](#running-tests)
- [Current Limitations](#current-limitations)
- [Persistence](#persistence)
- [Making Changes](#making-changes-development-workflow)
- [About MCP Servers](#about-mcp-servers)

---

## Quick Start

```bash
git clone https://github.com/your-username/karly-notes-mcp
cd karly-notes-mcp
go build -o karly-notes-mcp .
claude mcp add --scope user --transport stdio karly-notes -- $(pwd)/karly-notes-mcp
```

## How it works

If you've used Claude in VS Code, you've noticed it only knows what you tell it in the conversation — it can't save anything or remember notes between chats. It's powerful, but stateless.

**MCP (Model Context Protocol)** is a standard that lets you give Claude new capabilities by connecting it to external programs. Those programs are called **MCP servers**, and the capabilities they expose are called **tools**.

A tool is a function Claude is allowed to call. You don't call tools yourself — Claude decides when to use them based on what you're asking. If you say "save a note about Docker networking", Claude reads the available tools, picks `add_note`, infers the title and content from your message, and calls it. You never type the tool name.

Here's the full flow for a message like *"save a note about JWT tokens"*:

```
You type a message
       ↓
Claude reads it and decides which tool to call (add_note)
       ↓
Claude calls add_note with { title: "...", content: "..." }
       ↓
This Go program receives the call, saves the note to disk, returns "Saved!"
       ↓
Claude reads the result and replies to you
```

This Go program is the MCP server. It runs silently in the background whenever you have a Claude session open. Claude launches it automatically as a subprocess and communicates with it over stdin/stdout. You never interact with it directly — you just talk to Claude normally.

For more on how the subprocess communication works, see [MCP Transports](#mcp-transports).

---

## Setup

### 1. Install Go

Check if you already have it:

```bash
go version
```

If not, install it via [brew](https://brew.sh) (macOS):

```bash
brew install go
```

Or download it from [go.dev/dl](https://go.dev/dl). You need **Go 1.24 or newer**.

### 2. Clone the repo

```bash
git clone https://github.com/your-username/karly-notes-mcp
cd karly-notes-mcp
```

### 3. Build the binary

> **First time with Go?** Go is a compiled language — unlike JavaScript where you run source files directly with `node`, Go source code has to be compiled into a binary first. Think of it like a build step that produces a standalone executable, similar to `npm run build` producing a `dist/` folder. You only need to do this once (or again after pulling changes).

```bash
go build -o karly-notes-mcp .
```

This compiles the Go source files and produces a single executable file called `karly-notes-mcp` in the current directory. This is the file Claude will actually run.

### 4. Register it with Claude Code

Run this in your **terminal** (not inside Claude), from inside the `karly-notes-mcp/` directory you just cloned:

```bash
claude mcp add --scope user --transport stdio karly-notes -- $(pwd)/karly-notes-mcp
```

A few things to unpack:

- **Run it from the `karly-notes-mcp/` directory.** The `$(pwd)` part is a shell shortcut that expands to the full path of wherever you currently are. If you ran `go build` in that directory and you're still there, `$(pwd)/karly-notes-mcp` becomes the exact path to your binary automatically. If you run it from a different directory, `$(pwd)` will be wrong — just replace it with the absolute path to the binary instead.

- **`--transport stdio` tells Claude how to communicate with the server.** See [MCP Transports](#mcp-transports) for a full explanation.

- **`--scope user` makes it available in every project on your machine.** This is almost certainly what you want for a personal notes server. Your notes are yours, not tied to any one project. Without `--scope user`, the default scope is `local`, which only makes the server available in whatever directory you're currently working in — not very useful for a global second brain.

- **You run this once, ever.** Claude Code remembers it. From that point on, whenever you open any project and start a session, the notes server is available — Claude launches it automatically in the background when it needs it.

See [MCP Scopes](#mcp-scopes) for a full breakdown of scope options and the `--scope project` gotcha with compiled binaries.

That's it — Claude Code will now launch and manage the server automatically whenever you use it.

---

## Tools

Each tool is a function Claude can call. Claude reads the description to decide when to use it, and fills in the parameters based on context from your message.

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

This produces a compiled binary called `karly-notes-mcp`. You never run this binary yourself — Claude Code launches it automatically in the background whenever you start a session that uses it. The binary is what Claude actually executes each time it needs to call one of the note tools.

The binary is listed in `.gitignore` and not committed to the repo, so anyone who clones this project needs to run `go build` once to produce it on their own machine before registering it with Claude. See [Setup](#setup) above.

## Configuration

### Claude Code (CLI — recommended)

Claude Code has a built-in command for registering MCP servers — no JSON editing required:

```bash
claude mcp add --scope user --transport stdio karly-notes -- /absolute/path/to/binary/karly-notes-mcp/karly-notes-mcp
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

See [MCP Scopes](#mcp-scopes) for a full breakdown of scope options.

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
claude mcp add --scope user --transport stdio karly-notes -- /path/to/karly-notes-mcp --notes-dir /path/to/your/notes
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

---

## Making Changes (Development Workflow)

If you fork this repo and make changes to the Go source files, here's what you need to do:

### Step 1 — Rebuild the binary

```bash
go build -o karly-notes-mcp .
```

This overwrites the existing binary in place at the same path. You do not need to re-run `claude mcp add` — that was a one-time registration. Claude Code stored the path to the binary; as long as you rebuild to the same location, the registered path stays valid forever.

### Step 2 — Start a new Claude session

Claude launches the MCP server binary fresh at the start of each session. Your current session won't pick up the new binary — you need to open a new one.

**In the terminal (Claude Code CLI):**

Exit your current session if one is open, then start a new one:

```bash
claude
```

**In VS Code (Claude extension):**

Open the Command Palette (`Cmd+Shift+P` on Mac, `Ctrl+Shift+P` on Windows/Linux), search for **"Claude: New Chat"**, and open a new chat window. The extension starts a fresh session — and a fresh MCP server process — for each new chat.

### Step 3 — Verify the server connected

Once you're in a new session, type this into the chat:

```
/mcp
```

This is a built-in Claude Code command (not a message sent to the AI — think of it like a slash command in Slack). It shows you the status of all registered MCP servers. You should see something like:

```
karly-notes: connected
  Tools: add_note, get_note, list_notes, delete_note, search_notes
```

If you see `connected` and your tools listed, the new binary is live and working. If you see `failed` or `disconnected`, double-check that the build succeeded and printed no errors.

### Don't want to restart your session?

You can reconnect the server from inside your current Claude session without restarting anything. It works the same way in both the terminal and VS Code:

1. Type `/mcp` in the chat
2. Select **"Manage MCP servers"**
3. Select **karly-notes**
4. Select **"Reconnect"**

Claude restarts just the MCP server process using the new binary. Your conversation context stays intact.

---

## About MCP Servers

### MCP Transports

The `--transport` flag in `claude mcp add` tells Claude *how* to communicate with the MCP server — what channel to use to send and receive messages.

There are two transports you'll encounter:

#### `stdio` (Standard I/O)

Claude launches the binary as a **child process** and talks to it by writing to its `stdin` and reading from its `stdout`. It's pure inter-process communication — no network, no ports, no URLs. The process starts when your Claude session starts and exits when it ends.

This is why `claude mcp add` takes a **path to a binary** with this transport — Claude needs something to execute.

This also matches what's inside `main.go`:

```go
server.Run(context.Background(), &mcp.StdioTransport{})
```

Both sides have to agree on the transport. The binary says "I speak stdio" and `--transport stdio` tells Claude to use stdio to reach it. If they didn't match, they'd have no way to communicate.

**Use `stdio` for:** local binaries running on your machine — like this server.

#### `http`

Claude connects to a **remote server** over the network via HTTP. The server is already running somewhere (a hosted URL), and Claude just points at it.

This transport takes a URL instead of a binary path:

```bash
claude mcp add --transport http my-server https://my-server.example.com/mcp
```

**Use `http` for:** hosted or shared MCP servers running in the cloud.

For this server, `stdio` is always the right choice — it's a local binary that lives on your machine.

---

### MCP Scopes

The `--scope` flag controls where the registration is saved and which projects can see the server.

| Scope | Command flag | Where to run the command | Good for |
|-------|-------------|--------------------------|----------|
| `user` | `--scope user` | Anywhere — it's global | Personal tools you want everywhere (like this server) |
| `project` | `--scope project` | Inside the target project's root directory | Team tools shared via the repo — saves a `.mcp.json` file that teammates get when they clone |
| `local` | *(default, no flag needed)* | Inside the target project's root directory | Project tools you don't want committed to git |

> **`--scope project` tip:** Run the command from inside the project you want it scoped to, not from the `karly-notes-mcp/` directory. Also use an absolute path to the binary (not `$(pwd)/...`) since `$(pwd)` will resolve to that project's folder, not where the binary lives.
>
> **`--scope project` and compiled binaries don't mix well.** The `.mcp.json` file stores the absolute path to the binary on *your* machine — something like `/Users/yourname/workspace/karly-notes-mcp/karly-notes-mcp`. When a teammate clones the repo, that path won't exist on their machine, so the server will fail to launch for them even if they've built the binary themselves. `--scope project` works best for servers that use a universal command like `npx some-package` (same on every machine) or a remote URL. For a compiled binary, every developer needs to register it themselves with `--scope user` or `--scope local` using their own path.
