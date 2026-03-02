// tools.go defines the MCP tools exposed by this server.
//
// In MCP, a "tool" is a function that Claude can call during a conversation.
// Each tool has:
//   - A name       (what Claude calls it, e.g. "add_note")
//   - A description (plain English explaining what it does — Claude reads this
//                    to decide when to call the tool)
//   - An input schema (what parameters it accepts — derived automatically from
//                      the Go struct and its struct tags)
//   - A handler    (the Go code that actually runs when Claude calls the tool)
package main

import (
	"context"
	"fmt"
	"strings"

	"karly-notes-mcp/common"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ── Tool input types ─────────────────────────────────────────────────────────

type AddNoteInput struct {
	Title   string `json:"title" jsonschema:"title of the note"`
	Content string `json:"content" jsonschema:"content/body of the note"`
	Repo    string `json:"repo" jsonschema:"git remote origin URL of the current repository (e.g. github.com/user/repo). Empty string if not in a git repo."`
}

type GetNoteInput struct {
	Title string `json:"title" jsonschema:"title of the note to retrieve"`
}

type ListNotesInput struct{}

type DeleteNoteInput struct {
	Title string `json:"title" jsonschema:"title of the note to delete"`
}

type SearchNotesInput struct {
	Keyword string `json:"keyword" jsonschema:"keyword to search for in note titles and content"`
}

type ListNotesForCurrentRepoInput struct {
	Repo string `json:"repo" jsonschema:"git remote origin URL of the current repository (e.g. github.com/user/repo)"`
}

// ── Tool registration ────────────────────────────────────────────────────────

func registerTools(server *mcp.Server, store common.Store) {

	// ── add_note ──────────────────────────────────────────────────────────────
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_note",
		Description: "Save a note with a title and content. If a note with the same title exists, it will be updated.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AddNoteInput) (*mcp.CallToolResult, any, error) {
		if input.Title == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: title is required"}},
				IsError: true,
			}, nil, nil
		}
		if err := store.Add(input.Title, input.Content, input.Repo); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error saving note: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Saved note: %q", input.Title)}},
		}, nil, nil
	})

	// ── get_note ──────────────────────────────────────────────────────────────
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_note",
		Description: "Retrieve a note by its exact title.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetNoteInput) (*mcp.CallToolResult, any, error) {
		note, ok := store.Get(input.Title)
		if !ok {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No note found with title: %q", input.Title)}},
				IsError: true,
			}, nil, nil
		}
		text := fmt.Sprintf("# %s\n\n%s\n\nRepo: %s\nCreated: %s\nUpdated: %s", note.Title, note.Content, note.Repo, note.CreatedAt, note.UpdatedAt)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil, nil
	})

	// ── list_notes ────────────────────────────────────────────────────────────
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_notes",
		Description: "List all saved notes. Returns titles and creation dates.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListNotesInput) (*mcp.CallToolResult, any, error) {
		notes := store.List()
		if len(notes) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "No notes saved yet."}},
			}, nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d note(s):\n\n", len(notes)))
		for _, n := range notes {
			sb.WriteString(fmt.Sprintf("- %s (created: %s)\n", n.Title, n.CreatedAt))
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		}, nil, nil
	})

	// ── delete_note ───────────────────────────────────────────────────────────
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_note",
		Description: "Delete a note by its exact title.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteNoteInput) (*mcp.CallToolResult, any, error) {
		found, err := store.Delete(input.Title)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error deleting note: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		if found {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Deleted note: %q", input.Title)}},
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No note found with title: %q", input.Title)}},
			IsError: true,
		}, nil, nil
	})

	// ── search_notes ──────────────────────────────────────────────────────────
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_notes",
		Description: "Search notes by keyword. Matches against both title and content (case-insensitive).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchNotesInput) (*mcp.CallToolResult, any, error) {
		if input.Keyword == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: keyword is required"}},
				IsError: true,
			}, nil, nil
		}
		results := store.Search(input.Keyword)
		if len(results) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No notes found matching: %q", input.Keyword)}},
			}, nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d note(s) matching %q:\n\n", len(results), input.Keyword))
		for _, n := range results {
			sb.WriteString(fmt.Sprintf("## %s\n%s\n\n", n.Title, n.Content))
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		}, nil, nil
	})

	// ── list_notes_for_current_repo ───────────────────────────────────────────
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_notes_for_current_repo",
		Description: "List all notes saved in a specific repository. Use this to surface relevant notes when starting work in a project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListNotesForCurrentRepoInput) (*mcp.CallToolResult, any, error) {
		if input.Repo == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: repo is required"}},
				IsError: true,
			}, nil, nil
		}
		notes := store.ListByRepo(input.Repo)
		if len(notes) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No notes found for repo: %s", input.Repo)}},
			}, nil, nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d note(s) for %s:\n\n", len(notes), input.Repo))
		for _, n := range notes {
			sb.WriteString(fmt.Sprintf("## %s\n%s\n\n", n.Title, n.Content))
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		}, nil, nil
	})
}
