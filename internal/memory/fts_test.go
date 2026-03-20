package memory

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFTSSearch(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	SaveMemory(db, Memory{ID: "obs-auth", EntityType: "observation", Title: "Auth bug found", Content: "The authentication module has a race condition", Author: "scout", Visibility: "project", CreatedAt: now})
	SaveMemory(db, Memory{ID: "obs-perf", EntityType: "observation", Title: "Performance issue", Content: "Database queries are slow under load", Author: "scout", Visibility: "project", CreatedAt: now})

	results, err := SearchMemories(db, "authentication", SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'authentication', got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != "obs-auth" {
		t.Errorf("expected obs-auth, got %s", results[0].ID)
	}
}

func TestFTSExcludesDeprecated(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	SaveMemory(db, Memory{ID: "old", EntityType: "observation", Title: "Old finding", Content: "outdated info", Author: "a", Visibility: "project", CreatedAt: now})
	SaveMemory(db, Memory{ID: "new", EntityType: "observation", Title: "New finding", Content: "replaces old finding", Author: "a", Deprecates: "old", Visibility: "project", CreatedAt: now})

	results, err := SearchMemories(db, "finding", SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (deprecated excluded), got %d", len(results))
	}

	results, err = SearchMemories(db, "finding", SearchOpts{IncludeDeprecated: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (deprecated included), got %d", len(results))
	}
}

func TestSearchAllProjects(t *testing.T) {
	// Create two temporary project directories with their own DBs
	projectA := t.TempDir()
	projectB := t.TempDir()

	// Create .cx directories
	os.MkdirAll(filepath.Join(projectA, ".cx"), 0o755)
	os.MkdirAll(filepath.Join(projectB, ".cx"), 0o755)

	// Open project DBs (creates and migrates them)
	dbA, err := OpenProjectDB(projectA)
	if err != nil {
		t.Fatal(err)
	}
	dbB, err := OpenProjectDB(projectB)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert distinct memories
	SaveMemory(dbA, Memory{ID: "alpha-1", EntityType: "observation", Title: "Alpha discovery", Content: "Found in project alpha", Author: "scout", Visibility: "project", CreatedAt: now})
	SaveMemory(dbB, Memory{ID: "beta-1", EntityType: "observation", Title: "Beta pattern", Content: "Found in project beta", Author: "scout", Visibility: "project", CreatedAt: now})

	dbA.Close()
	dbB.Close()

	// Create a test index DB
	indexDir := t.TempDir()
	os.MkdirAll(filepath.Join(indexDir, ".cx"), 0o755)
	indexPath := filepath.Join(indexDir, "index.db")
	indexDB, err := sql.Open("sqlite", indexPath)
	if err != nil {
		t.Fatal(err)
	}
	defer indexDB.Close()
	Migrate(indexDB, indexMigrations)

	// Register both projects
	indexDB.Exec("INSERT INTO projects (id, name, path) VALUES (?, ?, ?)", "proj-a", "Alpha Project", projectA)
	indexDB.Exec("INSERT INTO projects (id, name, path) VALUES (?, ?, ?)", "proj-b", "Beta Project", projectB)

	// Search for "discovery" — should find alpha only
	results, err := SearchAllProjects(indexDB, "discovery", SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'discovery', got %d", len(results))
	}
	if results[0].ProjectName != "Alpha Project" {
		t.Errorf("expected project 'Alpha Project', got %q", results[0].ProjectName)
	}
	if results[0].ID != "alpha-1" {
		t.Errorf("expected ID 'alpha-1', got %q", results[0].ID)
	}

	// Search for "beta" — should find beta only
	results, err = SearchAllProjects(indexDB, "beta", SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'beta', got %d", len(results))
	}
	if results[0].ProjectName != "Beta Project" {
		t.Errorf("expected project 'Beta Project', got %q", results[0].ProjectName)
	}

	// Search for nonexistent — should return empty
	results, err = SearchAllProjects(indexDB, "nonexistent", SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'nonexistent', got %d", len(results))
	}
}
