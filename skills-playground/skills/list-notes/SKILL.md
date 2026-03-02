---
name: list-notes
description: List all saved notes with their titles and timestamps. Use when the user wants to see what notes they have or browse their note collection.
allowed-tools: Bash(bash scripts/run.sh:*)
---

# List Notes

## When to use this skill

Use this skill when the user wants to:
- See all their saved notes
- Browse what notes exist
- Check if they have any notes saved

## How to list notes

Run the CLI with the `list` command:

```bash
bash scripts/run.sh list
```

### Parameters

None.

### Output

Returns a JSON array of all notes to stdout:

```json
[{"title": "JWT gotcha", "content": "...", "repo": "...", "created_at": "...", "updated_at": "..."}, ...]
```

An empty array `[]` is returned if no notes exist.

### Example

```bash
bash scripts/run.sh list
```
