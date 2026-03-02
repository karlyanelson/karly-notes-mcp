// karly-notes-cli is a command-line interface for managing notes.
//
// It provides the same note operations as the MCP server, but as CLI commands
// that can be called from shell scripts — including agent skill scripts.
//
// Usage:
//
//	karly-notes-cli add --title "My Note" --content "Note body" [--repo "github.com/user/repo"]
//	karly-notes-cli get --title "My Note"
//	karly-notes-cli list
//	karly-notes-cli list-by-repo --repo "github.com/user/repo"
//	karly-notes-cli delete --title "My Note"
//	karly-notes-cli search --keyword "search term"
//	karly-notes-cli --help
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"karly-notes-mcp/common"
)

func main() {
	log.SetFlags(0) // no timestamp prefix for CLI output

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	if command == "--help" || command == "-h" {
		printUsage()
		return
	}

	store, err := initStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse flags for the subcommand (skip program name and command).
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	title := fs.String("title", "", "title of the note")
	content := fs.String("content", "", "content/body of the note")
	repo := fs.String("repo", "", "git remote origin URL of the current repository")
	keyword := fs.String("keyword", "", "keyword to search for in note titles and content")
	fs.Parse(os.Args[2:])

	switch command {
	case "add":
		runAdd(store, *title, *content, *repo)
	case "get":
		runGet(store, *title)
	case "list":
		runList(store)
	case "list-by-repo":
		runListByRepo(store, *repo)
	case "delete":
		runDelete(store, *title)
	case "search":
		runSearch(store, *keyword)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func initStore() (common.Store, error) {
	dir := os.Getenv("KARLY_NOTES_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".karly-notes")
	}
	// Suppress the log.Printf in NewFileStore for CLI usage.
	log.SetOutput(os.Stderr)
	return common.NewFileStore(dir)
}

func runAdd(store common.Store, title, content, repo string) {
	if title == "" {
		fmt.Fprintln(os.Stderr, "Error: --title is required")
		os.Exit(1)
	}
	if err := store.Add(title, content, repo); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving note: %v\n", err)
		os.Exit(1)
	}
	result := map[string]string{"status": "saved", "title": title}
	json.NewEncoder(os.Stdout).Encode(result)
}

func runGet(store common.Store, title string) {
	if title == "" {
		fmt.Fprintln(os.Stderr, "Error: --title is required")
		os.Exit(1)
	}
	note, ok := store.Get(title)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: no note found with title %q\n", title)
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(note)
}

func runList(store common.Store) {
	notes := store.List()
	json.NewEncoder(os.Stdout).Encode(notes)
}

func runListByRepo(store common.Store, repo string) {
	if repo == "" {
		fmt.Fprintln(os.Stderr, "Error: --repo is required")
		os.Exit(1)
	}
	notes := store.ListByRepo(repo)
	json.NewEncoder(os.Stdout).Encode(notes)
}

func runDelete(store common.Store, title string) {
	if title == "" {
		fmt.Fprintln(os.Stderr, "Error: --title is required")
		os.Exit(1)
	}
	found, err := store.Delete(title)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting note: %v\n", err)
		os.Exit(1)
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Error: no note found with title %q\n", title)
		os.Exit(1)
	}
	result := map[string]string{"status": "deleted", "title": title}
	json.NewEncoder(os.Stdout).Encode(result)
}

func runSearch(store common.Store, keyword string) {
	if keyword == "" {
		fmt.Fprintln(os.Stderr, "Error: --keyword is required")
		os.Exit(1)
	}
	results := store.Search(keyword)
	json.NewEncoder(os.Stdout).Encode(results)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: karly-notes-cli <command> [flags]

Commands:
  add            Save a note (or update an existing one by title)
  get            Retrieve a note by its exact title
  list           List all saved notes
  list-by-repo   List all notes saved in a specific repository
  delete         Remove a note by title
  search         Find notes matching a keyword (case-insensitive)

Flags:
  --title     Title of the note (required for add, get, delete)
  --content   Content/body of the note (used with add)
  --repo      Git remote origin URL (used with add, list-by-repo)
  --keyword   Search keyword (required for search)

Environment:
  KARLY_NOTES_DIR   Directory where notes.json is stored (default: ~/.karly-notes)

Examples:
  karly-notes-cli add --title "JWT gotcha" --content "Normalize to UTC before comparing expiry"
  karly-notes-cli get --title "JWT gotcha"
  karly-notes-cli list
  karly-notes-cli search --keyword "JWT"
  karly-notes-cli delete --title "old note"
  karly-notes-cli list-by-repo --repo "github.com/user/repo"
`)
}
