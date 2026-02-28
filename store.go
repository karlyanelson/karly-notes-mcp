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
