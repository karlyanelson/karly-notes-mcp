package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// Store is the interface implemented by both NoteStore (in-memory, used in
// tests) and FileStore (persistent, used in production).
type Store interface {
	Add(title, content string) error
	Get(title string) (Note, bool)
	List() []Note
	Delete(title string) (bool, error)
	Search(keyword string) []Note
}

// ── In-memory store (used by tests) ─────────────────────────────────────────

// NoteStore is a thread-safe in-memory store for notes.
type NoteStore struct {
	mu    sync.RWMutex
	notes map[string]Note
}

func NewNoteStore() *NoteStore {
	return &NoteStore{notes: make(map[string]Note)}
}

func (s *NoteStore) Add(title, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	existing, exists := s.notes[title]
	createdAt := now
	if exists {
		createdAt = existing.CreatedAt
	}
	s.notes[title] = Note{
		Title:     title,
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}
	return nil
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

func (s *NoteStore) Delete(title string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notes[title]; !ok {
		return false, nil
	}
	delete(s.notes, title)
	return true, nil
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

// ── File-backed store ────────────────────────────────────────────────────────

// fileData is the on-disk JSON structure.
type fileData struct {
	Version int             `json:"version"`
	Notes   map[string]Note `json:"notes"`
}

// FileStore is a thread-safe, file-backed note store. Notes are kept in memory
// and flushed to disk atomically after every mutation.
type FileStore struct {
	mu   sync.RWMutex
	path string
	data fileData
}

// NewFileStore creates (or loads) a note store rooted at dir/notes.json.
// The directory is created if it does not exist.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create notes directory: %w", err)
	}
	path := filepath.Join(dir, "notes.json")
	fs := &FileStore{
		path: path,
		data: fileData{Version: 1, Notes: make(map[string]Note)},
	}
	if err := fs.load(); err != nil {
		return nil, fmt.Errorf("load notes from %s: %w", path, err)
	}
	log.Printf("notes store: %s (%d note(s))", path, len(fs.data.Notes))
	return fs, nil
}

// load reads notes.json from disk into memory. If the file does not exist yet
// it is treated as an empty store.
func (fs *FileStore) load() error {
	raw, err := os.ReadFile(fs.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &fs.data)
}

// save writes the current in-memory state to disk atomically.
// Write to a tmp file first, then rename so the file is never half-written.
// Must be called with fs.mu held for writing.
func (fs *FileStore) save() error {
	raw, err := json.MarshalIndent(fs.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := fs.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, fs.path)
}

func (fs *FileStore) Add(title, content string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	existing, exists := fs.data.Notes[title]
	createdAt := now
	if exists {
		createdAt = existing.CreatedAt
	}
	fs.data.Notes[title] = Note{
		Title:     title,
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}
	return fs.save()
}

func (fs *FileStore) Get(title string) (Note, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	n, ok := fs.data.Notes[title]
	return n, ok
}

func (fs *FileStore) List() []Note {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	result := make([]Note, 0, len(fs.data.Notes))
	for _, n := range fs.data.Notes {
		result = append(result, n)
	}
	return result
}

func (fs *FileStore) Delete(title string) (bool, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if _, ok := fs.data.Notes[title]; !ok {
		return false, nil
	}
	delete(fs.data.Notes, title)
	return true, fs.save()
}

func (fs *FileStore) Search(keyword string) []Note {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	kw := strings.ToLower(keyword)
	var result []Note
	for _, n := range fs.data.Notes {
		if strings.Contains(strings.ToLower(n.Title), kw) ||
			strings.Contains(strings.ToLower(n.Content), kw) {
			result = append(result, n)
		}
	}
	return result
}

// ── Tool input types ─────────────────────────────────────────────────────────

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

// ── Tool registration ────────────────────────────────────────────────────────

func registerTools(server *mcp.Server, store Store) {
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
		if err := store.Add(input.Title, input.Content); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error saving note: %v", err)}},
				IsError: true,
			}, nil, nil
		}
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

// ── Entry point ──────────────────────────────────────────────────────────────

func main() {
	// Resolve notes directory: flag > env var > default (~/.karly-notes)
	defaultDir := os.Getenv("KARLY_NOTES_DIR")
	if defaultDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home directory: %v", err)
		}
		defaultDir = filepath.Join(home, ".karly-notes")
	}

	var notesDir string
	flag.StringVar(&notesDir, "notes-dir", defaultDir, "directory where notes.json is stored")
	flag.Parse()

	store, err := NewFileStore(notesDir)
	if err != nil {
		log.Fatalf("failed to initialize note store at %s: %v", notesDir, err)
	}

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
