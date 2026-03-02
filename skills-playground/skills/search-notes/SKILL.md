---
name: search-notes
description: Search notes by keyword. Matches against both title and content (case-insensitive). Use when the user wants to find notes about a topic or containing specific text.
allowed-tools: Bash(bash scripts/run.sh:*)
---

# Search Notes

## When to use this skill

Use this skill when the user wants to:
- Find notes about a specific topic
- Search for notes containing a keyword
- Look up something they saved before but don't remember the exact title

## How to search notes

Run the CLI with the `search` command:

```bash
bash scripts/run.sh search --keyword "KEYWORD"
```

### Parameters

- `--keyword` (required): The search term. Matches are case-insensitive and checked against both the title and content of every note.

### Output

Returns a JSON array of matching notes to stdout:

```json
[{"title": "JWT gotcha", "content": "Normalize to UTC...", "repo": "...", "created_at": "...", "updated_at": "..."}]
```

An empty array `[]` is returned if no notes match.

### Examples

```bash
bash scripts/run.sh search --keyword "JWT"
bash scripts/run.sh search --keyword "docker"
```
