package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Note represents a single saved note.
type Note struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// NoteStore is a thread-safe in-memory store for notes.
type NoteStore struct {
	mu    sync.RWMutex
	notes map[string]Note
}

func NewNoteStore() *NoteStore {
	return &NoteStore{notes: make(map[string]Note)}
}

func (s *NoteStore) Add(title, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	s.notes[title] = Note{
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *NoteStore) Get(title string) (Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[title]
	return n, ok
}

func (s *NoteStore) List() []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Note, 0, len(s.notes))
	for _, n := range s.notes {
		result = append(result, n)
	}
	return result
}

func (s *NoteStore) Delete(title string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notes[title]; !ok {
		return false
	}
	delete(s.notes, title)
	return true
}

func (s *NoteStore) Search(keyword string) []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	kw := strings.ToLower(keyword)
	var result []Note
	for _, n := range s.notes {
		if strings.Contains(strings.ToLower(n.Title), kw) ||
			strings.Contains(strings.ToLower(n.Content), kw) {
			result = append(result, n)
		}
	}
	return result
}

// Tool input types

type AddNoteInput struct {
	Title   string `json:"title" jsonschema:"title of the note"`
	Content string `json:"content" jsonschema:"content/body of the note"`
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

func registerTools(server *mcp.Server, store *NoteStore) {
	// add_note
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
		store.Add(input.Title, input.Content)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Saved note: %q", input.Title)}},
		}, nil, nil
	})

	// get_note
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

	// list_notes
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

	// delete_note
	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_note",
		Description: "Delete a note by its exact title.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DeleteNoteInput) (*mcp.CallToolResult, any, error) {
		if store.Delete(input.Title) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Deleted note: %q", input.Title)}},
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("No note found with title: %q", input.Title)}},
			IsError: true,
		}, nil, nil
	})

	// search_notes
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

func main() {
	store := NewNoteStore()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "karly-notes",
		Version: "v0.1.0",
	}, &mcp.ServerOptions{
		Instructions: "A personal second-brain notes server. Use add_note to save knowledge, get_note to retrieve it, list_notes to see everything, delete_note to remove notes, and search_notes to find notes by keyword.",
	})

	registerTools(server, store)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
