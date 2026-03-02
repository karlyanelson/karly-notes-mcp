---
name: get-note
description: Retrieve a note by its exact title. Use when the user asks to see, read, or recall a specific note they saved previously.
allowed-tools: Bash(bash scripts/run.sh:*)
---

# Get Note

## When to use this skill

Use this skill when the user wants to:
- Retrieve a specific note by title
- Read the full content of a previously saved note
- Check what they wrote in a particular note

## How to retrieve a note

Run the CLI with the `get` command:

```bash
bash scripts/run.sh get --title "TITLE"
```

### Parameters

- `--title` (required): The exact title of the note to retrieve.

### Output

Returns the note as JSON to stdout:

```json
{"title": "JWT gotcha", "content": "Normalize to UTC...", "repo": "github.com/user/app", "created_at": "2025-03-15T10:30:00Z", "updated_at": "2025-03-15T10:30:00Z"}
```

If the note is not found, an error is printed to stderr with a non-zero exit code.

### Example

```bash
bash scripts/run.sh get --title "JWT timezone gotcha"
```
