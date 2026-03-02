---
name: delete-note
description: Delete a note by its exact title. Use when the user wants to remove a note they no longer need.
allowed-tools: Bash(karly-notes-cli:*)
---

# Delete Note

## When to use this skill

Use this skill when the user wants to:
- Remove a specific note
- Clean up old or outdated notes
- Delete a note they no longer need

## How to delete a note

Run the CLI with the `delete` command:

```bash
karly-notes-cli delete --title "TITLE"
```

### Parameters

- `--title` (required): The exact title of the note to delete.

### Output

Returns JSON to stdout on success:

```json
{"status": "deleted", "title": "My Note"}
```

If the note is not found, an error is printed to stderr with a non-zero exit code.

### Example

```bash
karly-notes-cli delete --title "old todo list"
```
