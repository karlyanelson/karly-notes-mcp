package common

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestNoteStoreAdd(t *testing.T) {
	store := NewNoteStore()
	if err := store.Add("test", "content", "repo"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	note, ok := store.Get("test")
	if !ok {
		t.Fatal("note not found after add")
	}
	if note.Content != "content" {
		t.Errorf("unexpected content: %s", note.Content)
	}
}

func TestNoteStoreSearch(t *testing.T) {
	store := NewNoteStore()
	store.Add("JWT gotcha", "normalize to UTC", "")
	store.Add("Docker tip", "use bridge mode", "")

	results := store.Search("jwt")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "JWT gotcha" {
		t.Errorf("unexpected title: %s", results[0].Title)
	}
}

func TestFileStorePersistence(t *testing.T) {
	dir := t.TempDir()

	fs1, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := fs1.Add("persist-test", "this should survive a restart", "github.com/test/my-app"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	fs2, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore (second instance): %v", err)
	}
	note, ok := fs2.Get("persist-test")
	if !ok {
		t.Fatal("note not found after reload")
	}
	if !strings.Contains(note.Content, "survive a restart") {
		t.Errorf("unexpected content after reload: %s", note.Content)
	}

	// Verify CreatedAt is preserved on update.
	originalCreatedAt := note.CreatedAt
	if err := fs2.Add("persist-test", "updated content", "github.com/test/my-app"); err != nil {
		t.Fatalf("Add (update): %v", err)
	}
	updated, _ := fs2.Get("persist-test")
	if updated.CreatedAt != originalCreatedAt {
		t.Errorf("CreatedAt changed on update: was %s, now %s", originalCreatedAt, updated.CreatedAt)
	}

	// Verify deletion is persisted.
	if _, err := fs2.Delete("persist-test"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	fs3, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore (third instance): %v", err)
	}
	if _, ok := fs3.Get("persist-test"); ok {
		t.Error("deleted note still present after reload")
	}

	// Verify the on-disk file is valid JSON with the expected structure.
	raw, err := os.ReadFile(dir + "/notes.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var fd FileData
	if err := json.Unmarshal(raw, &fd); err != nil {
		t.Fatalf("notes.json is not valid JSON: %v", err)
	}
	if fd.Version != 1 {
		t.Errorf("expected version 1, got %d", fd.Version)
	}
}
