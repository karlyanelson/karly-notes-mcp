---
name: list-notes-for-repo
description: List all notes saved in a specific repository. Use when the user starts working in a project and wants to surface relevant notes, or when they ask for notes related to a particular repo.
allowed-tools: Bash(karly-notes-cli:*)
---

# List Notes for Repository

## When to use this skill

Use this skill when the user wants to:
- See all notes associated with a specific repository
- Surface relevant context when starting work in a project
- Check what notes exist for the current codebase

## How to list notes by repo

Run the CLI with the `list-by-repo` command:

```bash
karly-notes-cli list-by-repo --repo "REPO"
```

### Parameters

- `--repo` (required): The git remote origin URL of the repository (e.g. `github.com/user/repo`). Must be an exact match.

### Output

Returns a JSON array of notes for that repository to stdout:

```json
[{"title": "JWT gotcha", "content": "...", "repo": "github.com/user/repo", "created_at": "...", "updated_at": "..."}]
```

An empty array `[]` is returned if no notes exist for the given repo.

### Example

```bash
karly-notes-cli list-by-repo --repo "github.com/user/my-app"
```
