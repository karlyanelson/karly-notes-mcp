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
//
// When you say "save a note about JWT tokens" in the chat, Claude reads the
// tool descriptions, decides add_note is the right fit, infers the title and
// content from your message, and calls the handler with those values. You never
// type the tool name yourself — Claude figures it out from context.
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ── Tool input types ─────────────────────────────────────────────────────────
//
// Each tool has a dedicated input struct that describes its parameters.
// The struct tags serve two purposes:
//
//	json:"..."       — the JSON key name Claude uses when calling the tool.
//	                   Claude never sees the Go field name (e.g. "Title");
//	                   it only sees the JSON key ("title").
//
//	jsonschema:"..." — becomes the parameter description in the tool's schema.
//	                   This is what Claude reads to understand what each field
//	                   means and what value to put in it.

type AddNoteInput struct {
	Title   string `json:"title" jsonschema:"title of the note"`
	Content string `json:"content" jsonschema:"content/body of the note"`
}

type GetNoteInput struct {
	Title string `json:"title" jsonschema:"title of the note to retrieve"`
}

type ListNotesInput struct{} // list_notes takes no parameters

type DeleteNoteInput struct {
	Title string `json:"title" jsonschema:"title of the note to delete"`
}

type SearchNotesInput struct {
	Keyword string `json:"keyword" jsonschema:"keyword to search for in note titles and content"`
}

// ── Tool registration ────────────────────────────────────────────────────────

// registerTools attaches all five note tools to the MCP server.
// It is called once at startup in main.go. After this point, Claude knows the
// tools exist and can call any of them at any point during a conversation.
func registerTools(server *mcp.Server, store Store) {

	// mcp.AddTool registers one tool with the server. It takes three arguments:
	//
	//   1. server     — the MCP server to register on
	//   2. *mcp.Tool  — the tool's name and description (what Claude reads)
	//   3. handler    — the Go function that runs when Claude calls the tool
	//
	// The handler function signature always looks like this:
	//
	//   func(ctx, req, input InputType) (*mcp.CallToolResult, any, error)
	//
	// Parameters:
	//   ctx   — a Go context, used for cancellation/timeouts (safe to ignore here)
	//   req   — the raw request from Claude (rarely needed directly)
	//   input — the typed input struct, already parsed from the JSON Claude sent
	//
	// Return values:
	//   *mcp.CallToolResult — the response Claude will read
	//   any                 — structured output (an advanced MCP feature; always nil here)
	//   error               — a Go-level transport error; for expected failures like
	//                         "note not found", use IsError:true in the result instead

	// ── add_note ──────────────────────────────────────────────────────────────
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_note",
		Description: "Save a note with a title and content. If a note with the same title exists, it will be updated.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AddNoteInput) (*mcp.CallToolResult, any, error) {
		if input.Title == "" {
			// IsError: true tells Claude the tool call failed at the application
			// level (like a 4xx HTTP status). Claude will read the Content text
			// and use it to explain the problem to the user.
			//
			// This is different from returning a Go error (the third return value),
			// which signals a transport or infrastructure failure — something Claude
			// can't recover from gracefully.
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: title is required"}},
				IsError: true,
			}, nil, nil
		}
		if err := store.Add(input.Title, input.Content); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error saving note: %v", err)}},
				IsError: true,
			}, nil, nil
		}
		// mcp.TextContent wraps the text Claude will read as the tool's response.
		// MCP also supports image and embedded resource content types, but plain
		// text is all this server needs.
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
		text := fmt.Sprintf("# %s\n\n%s\n\nCreated: %s\nUpdated: %s", note.Title, note.Content, note.CreatedAt, note.UpdatedAt)
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
}
