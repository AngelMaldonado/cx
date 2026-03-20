package memory

import (
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
