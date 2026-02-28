package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestServer() (*mcp.Server, *NoteStore) {
	store := NewNoteStore()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "karly-notes",
		Version: "v0.1.0",
	}, nil)

	registerTools(server, store)
	return server, store
}

func connectTestClient(t *testing.T, ctx context.Context, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	st, ct := mcp.NewInMemoryTransports()
	// Start server FIRST so it's ready to handle the initialize handshake
	go server.Run(ctx, st)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return session
}

func callTool(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: json.RawMessage(argsJSON),
	})
	if err != nil {
		t.Fatalf("call tool %s: %v", name, err)
	}
	return result
}

func getText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("no content in result")
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func TestAddAndGetNote(t *testing.T) {
	server, _ := setupTestServer()
	ctx := context.Background()
	session := connectTestClient(t, ctx, server)
	defer session.Close()

	// Add a note
	result := callTool(t, ctx, session, "add_note", map[string]string{
		"title":   "test note",
		"content": "hello world",
	})
	text := getText(t, result)
	if text != `Saved note: "test note"` {
		t.Errorf("unexpected add result: %s", text)
	}

	// Get the note
	result = callTool(t, ctx, session, "get_note", map[string]string{
		"title": "test note",
	})
	text = getText(t, result)
	if !strings.Contains(text, "hello world") {
		t.Errorf("expected content 'hello world', got: %s", text)
	}
	if !strings.Contains(text, "# test note") {
		t.Errorf("expected title header, got: %s", text)
	}
}

func TestListNotes(t *testing.T) {
	server, store := setupTestServer()
	ctx := context.Background()
	session := connectTestClient(t, ctx, server)
	defer session.Close()

	// Empty list
	result := callTool(t, ctx, session, "list_notes", map[string]string{})
	text := getText(t, result)
	if text != "No notes saved yet." {
		t.Errorf("expected empty message, got: %s", text)
	}

	// Add some notes
	store.Add("note1", "content1")
	store.Add("note2", "content2")

	result = callTool(t, ctx, session, "list_notes", map[string]string{})
	text = getText(t, result)
	if !strings.Contains(text, "2 note(s)") {
		t.Errorf("expected 2 notes, got: %s", text)
	}
}

func TestDeleteNote(t *testing.T) {
	server, store := setupTestServer()
	ctx := context.Background()
	session := connectTestClient(t, ctx, server)
	defer session.Close()

	store.Add("to-delete", "bye")

	// Delete existing
	result := callTool(t, ctx, session, "delete_note", map[string]string{"title": "to-delete"})
	text := getText(t, result)
	if !strings.Contains(text, "Deleted") {
		t.Errorf("expected deleted message, got: %s", text)
	}

	// Delete non-existent
	result = callTool(t, ctx, session, "delete_note", map[string]string{"title": "nope"})
	if !result.IsError {
		t.Error("expected error for non-existent note")
	}
}

func TestSearchNotes(t *testing.T) {
	server, store := setupTestServer()
	ctx := context.Background()
	session := connectTestClient(t, ctx, server)
	defer session.Close()

	store.Add("JWT timezone gotcha", "normalize to UTC before comparing expiry")
	store.Add("Docker networking", "use bridge mode for local dev")

	// Search matching
	result := callTool(t, ctx, session, "search_notes", map[string]string{"keyword": "JWT"})
	text := getText(t, result)
	if !strings.Contains(text, "JWT timezone gotcha") {
		t.Errorf("expected JWT note in results, got: %s", text)
	}
	if strings.Contains(text, "Docker") {
		t.Errorf("did not expect Docker note in JWT search results")
	}

	// Search no match
	result = callTool(t, ctx, session, "search_notes", map[string]string{"keyword": "kubernetes"})
	text = getText(t, result)
	if !strings.Contains(text, "No notes found") {
		t.Errorf("expected no results, got: %s", text)
	}

	// Case-insensitive search
	result = callTool(t, ctx, session, "search_notes", map[string]string{"keyword": "utc"})
	text = getText(t, result)
	if !strings.Contains(text, "JWT timezone gotcha") {
		t.Errorf("expected case-insensitive match, got: %s", text)
	}
}

func TestGetNoteNotFound(t *testing.T) {
	server, _ := setupTestServer()
	ctx := context.Background()
	session := connectTestClient(t, ctx, server)
	defer session.Close()

	result := callTool(t, ctx, session, "get_note", map[string]string{"title": "nonexistent"})
	if !result.IsError {
		t.Error("expected error for non-existent note")
	}
}

func TestFileStorePersistence(t *testing.T) {
	dir := t.TempDir()

	// Write a note with the first store instance.
	fs1, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := fs1.Add("persist-test", "this should survive a restart"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Open a second instance pointing to the same directory — simulates a restart.
	fs2, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore (second instance): %v", err)
	}
	note, ok := fs2.Get("persist-test")
	if !ok {
		t.Fatal("note not found after reload — persistence is broken")
	}
	if !strings.Contains(note.Content, "survive a restart") {
		t.Errorf("unexpected content after reload: %s", note.Content)
	}

	// Verify CreatedAt is preserved on update (not reset to now).
	originalCreatedAt := note.CreatedAt
	if err := fs2.Add("persist-test", "updated content"); err != nil {
		t.Fatalf("Add (update): %v", err)
	}
	updated, _ := fs2.Get("persist-test")
	if updated.CreatedAt != originalCreatedAt {
		t.Errorf("CreatedAt changed on update: was %s, now %s", originalCreatedAt, updated.CreatedAt)
	}
	if !strings.Contains(updated.Content, "updated content") {
		t.Errorf("content was not updated: %s", updated.Content)
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
	var fd fileData
	if err := json.Unmarshal(raw, &fd); err != nil {
		t.Fatalf("notes.json is not valid JSON: %v", err)
	}
	if fd.Version != 1 {
		t.Errorf("expected version 1, got %d", fd.Version)
	}
}
