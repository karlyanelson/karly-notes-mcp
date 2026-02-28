// store.go handles storing and retrieving notes.
//
// There are two implementations of the Store interface:
//   - NoteStore  — in-memory only, used in tests (fast, no disk I/O)
//   - FileStore  — persists notes to a JSON file on disk, used in production
//
// Both implement the same Store interface, so the tool handlers in tools.go
// don't need to know or care which one they're talking to.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Note represents a single saved note.
//
// The `json:"..."` struct tags control how fields are named in the JSON file.
// Without them, Go would use the capitalized field names (Title, Content, etc.)
// as the JSON keys. The tags make the on-disk format use lowercase snake_case
// keys instead (title, content, created_at, updated_at), which is conventional
// for JSON.
type Note struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Repo      string `json:"repo"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Store is the interface implemented by both NoteStore (in-memory, used in
// tests) and FileStore (persistent, used in production).
type Store interface {
	Add(title, content, repo string) error
	Get(title string) (Note, bool)
	List() []Note
	ListByRepo(repo string) []Note
	Delete(title string) (bool, error)
	Search(keyword string) []Note
}

// ── In-memory store (used by tests) ─────────────────────────────────────────

// NoteStore is a thread-safe in-memory store for notes.
//
// Thread safety matters here because an MCP server can receive multiple tool
// calls concurrently — for example, Claude might call list_notes and
// search_notes at the same time. sync.RWMutex prevents data races by allowing
// many simultaneous reads but only one write at a time.
type NoteStore struct {
	mu    sync.RWMutex
	notes map[string]Note
}

func NewNoteStore() *NoteStore {
	return &NoteStore{notes: make(map[string]Note)}
}

func (s *NoteStore) Add(title, content, repo string) error {
	s.mu.Lock()
	// defer schedules s.mu.Unlock() to run when this function returns,
	// regardless of how it exits. This ensures the mutex is always released
	// even if the function returns early or panics.
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	existing, exists := s.notes[title]
	createdAt := now
	if exists {
		// Preserve the original creation time if the note already exists.
		createdAt = existing.CreatedAt
	}
	s.notes[title] = Note{
		Title:     title,
		Content:   content,
		Repo:      repo,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}
	return nil
}

func (s *NoteStore) Get(title string) (Note, bool) {
	s.mu.RLock() // RLock allows concurrent reads; no write is happening
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

func (s *NoteStore) ListByRepo(repo string) []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Note
	for _, n := range s.notes {
		if n.Repo == repo {
			result = append(result, n)
		}
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
// Wrapping notes in a versioned envelope (rather than storing a bare array)
// makes it possible to migrate the format later without losing existing data.
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

func (fs *FileStore) Add(title, content, repo string) error {
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
		Repo:      repo,
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

func (fs *FileStore) ListByRepo(repo string) []Note {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var result []Note
	for _, n := range fs.data.Notes {
		if n.Repo == repo {
			result = append(result, n)
		}
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
