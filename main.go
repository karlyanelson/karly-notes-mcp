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
