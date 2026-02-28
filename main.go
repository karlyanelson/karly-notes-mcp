// main.go is the entry point for the karly-notes MCP server.
//
// An MCP server is a program that exposes "tools" — functions that Claude can
// call during a conversation. When you ask Claude to "save a note" or "find
// my notes about Docker", Claude calls the appropriate tool here, this program
// handles it, and returns a response that Claude passes back to you.
//
// This file sets up the server in three steps:
//  1. Configure where notes are stored on disk
//  2. Create an MCP server and register the note tools on it
//  3. Start the server and listen for requests from Claude over stdin/stdout
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	// Create the MCP server. Think of this like creating an Express app before
	// you add routes — the server exists but has no tools yet.
	//
	// The Instructions string is plain English that Claude reads at startup to
	// understand what this server does and when to use its tools. Think of it
	// as a system prompt for the server itself.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "karly-notes",
		Version: "v0.1.0",
	}, &mcp.ServerOptions{
		Instructions: "A personal second-brain notes server. Use add_note to save knowledge, get_note to retrieve it, list_notes to see everything, delete_note to remove notes, and search_notes to find notes by keyword.",
	})

	// Register all five note tools on the server (defined in tools.go).
	// This is like adding routes to an Express app — until tools are registered,
	// the server exists but Claude has nothing it can call.
	registerTools(server, store)

	// Start the server and block until it exits. StdioTransport means Claude
	// communicates with this process by writing to its stdin and reading from
	// its stdout — no network port needed. Claude launches this binary as a
	// subprocess when a session starts, and this line is what keeps it alive
	// and listening for incoming tool calls.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
