---
name: add-note
description: Save a note with a title and content. Use when the user wants to save, remember, or jot down information for later. If a note with the same title exists, it will be updated.
allowed-tools: Bash(karly-notes-cli:*)
---

# Add Note

## When to use this skill

Use this skill when the user wants to:
- Save a note, reminder, or piece of knowledge
- Update an existing note with new content
- Record a debugging insight, decision, or tip

## How to save a note

Run the CLI with the `add` command:

```bash
karly-notes-cli add --title "TITLE" --content "CONTENT" [--repo "REPO"]
```

### Parameters

- `--title` (required): A short, descriptive title for the note.
- `--content` (required): The body of the note. Can be multi-line.
- `--repo` (optional): The git remote origin URL if this note is associated with a specific repository (e.g. `github.com/user/repo`).

### Output

Returns JSON to stdout on success:

```json
{"status": "saved", "title": "My Note"}
```

Errors are printed to stderr with a non-zero exit code.

### Examples

Save a simple note:

```bash
karly-notes-cli add --title "JWT timezone gotcha" --content "Normalize to UTC before comparing expiry"
```

Save a note tied to a repo:

```bash
karly-notes-cli add --title "Docker networking" --content "Use bridge mode for local dev" --repo "github.com/user/infra"
```
