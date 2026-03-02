// main_test.go tests the MCP tools end-to-end by simulating what Claude does
// at runtime.
//
// In production, Claude is the "client" — it sends tool call requests to this
// server over stdin/stdout. In tests, we play the role of Claude ourselves:
// we create a test client, connect it to the server over an in-memory
// transport, and call tools the same way Claude would. This means the tests
// exercise the full request/response path, not just isolated Go functions.
package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"karly-notes-mcp/common"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// setupTestServer creates a fresh MCP server backed by an in-memory store.
func setupTestServer() (*mcp.Server, *common.NoteStore) {
	store := common.NewNoteStore()
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "karly-notes",
		Version: "v0.1.0",
	}, nil)

	registerTools(server, store)
	return server, store
}

// connectTestClient wires a test MCP client to the server using an in-memory
// transport.
func connectTestClient(t *testing.T, ctx context.Context, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	st, ct := mcp.NewInMemoryTransports()
	go server.Run(ctx, st)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return session
}

// callTool sends a tool call request to the server, exactly as Claude would.
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

// getText extracts the text string from a tool result's first content item.
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

	result := callTool(t, ctx, session, "add_note", map[string]string{
		"title":   "test note",
		"content": "hello world",
		"repo":    "github.com/test/my-app",
	})
	text := getText(t, result)
	if text != `Saved note: "test note"` {
		t.Errorf("unexpected add result: %s", text)
	}

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

	result := callTool(t, ctx, session, "list_notes", map[string]string{})
	text := getText(t, result)
	if text != "No notes saved yet." {
		t.Errorf("expected empty message, got: %s", text)
	}

	store.Add("note1", "content1", "")
	store.Add("note2", "content2", "")

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

	store.Add("to-delete", "bye", "")

	result := callTool(t, ctx, session, "delete_note", map[string]string{"title": "to-delete"})
	text := getText(t, result)
	if !strings.Contains(text, "Deleted") {
		t.Errorf("expected deleted message, got: %s", text)
	}

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

	store.Add("JWT timezone gotcha", "normalize to UTC before comparing expiry", "github.com/test/auth-service")
	store.Add("Docker networking", "use bridge mode for local dev", "github.com/test/infra")

	result := callTool(t, ctx, session, "search_notes", map[string]string{"keyword": "JWT"})
	text := getText(t, result)
	if !strings.Contains(text, "JWT timezone gotcha") {
		t.Errorf("expected JWT note in results, got: %s", text)
	}
	if strings.Contains(text, "Docker") {
		t.Errorf("did not expect Docker note in JWT search results")
	}

	result = callTool(t, ctx, session, "search_notes", map[string]string{"keyword": "kubernetes"})
	text = getText(t, result)
	if !strings.Contains(text, "No notes found") {
		t.Errorf("expected no results, got: %s", text)
	}

	result = callTool(t, ctx, session, "search_notes", map[string]string{"keyword": "utc"})
	text = getText(t, result)
	if !strings.Contains(text, "JWT timezone gotcha") {
		t.Errorf("expected case-insensitive match, got: %s", text)
	}
}

func TestListNotesForCurrentRepo(t *testing.T) {
	server, store := setupTestServer()
	ctx := context.Background()
	session := connectTestClient(t, ctx, server)
	defer session.Close()

	store.Add("JWT gotcha", "normalize to UTC", "github.com/test/auth-service")
	store.Add("Docker tip", "use bridge mode", "github.com/test/infra")
	store.Add("Go modules", "always use go mod tidy", "github.com/test/auth-service")
	store.Add("General note", "not in any repo", "")

	result := callTool(t, ctx, session, "list_notes_for_current_repo", map[string]string{
		"repo": "github.com/test/auth-service",
	})
	text := getText(t, result)
	if !strings.Contains(text, "2 note(s)") {
		t.Errorf("expected 2 notes for auth-service, got: %s", text)
	}
	if !strings.Contains(text, "JWT gotcha") {
		t.Errorf("expected JWT gotcha in results, got: %s", text)
	}
	if !strings.Contains(text, "Go modules") {
		t.Errorf("expected Go modules in results, got: %s", text)
	}
	if strings.Contains(text, "Docker") {
		t.Errorf("did not expect Docker note in auth-service results")
	}

	result = callTool(t, ctx, session, "list_notes_for_current_repo", map[string]string{
		"repo": "github.com/test/unknown",
	})
	text = getText(t, result)
	if !strings.Contains(text, "No notes found") {
		t.Errorf("expected no results for unknown repo, got: %s", text)
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
