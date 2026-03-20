package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPushExportsProjectVisibility(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)

	// Project-visible observation — should be exported
	SaveMemory(db, Memory{ID: "pub-obs", EntityType: "observation", Title: "Public obs", Content: "shared content", Author: "agent", Visibility: "project", CreatedAt: now})
	// Personal observation — should NOT be exported
	SaveMemory(db, Memory{ID: "priv-obs", EntityType: "observation", Title: "Private obs", Content: "personal content", Author: "agent", Visibility: "personal", CreatedAt: now})
	// Session — should NEVER be exported
	SaveMemory(db, Memory{ID: "sess-mem", EntityType: "session", Title: "Session mem", Content: "session content", Author: "agent", Visibility: "project", CreatedAt: now})

	result, err := Push(db, tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	if result.Exported != 1 {
		t.Errorf("expected 1 exported, got %d", result.Exported)
	}

	// Verify file exists
	obsFile := filepath.Join(tmpDir, "memory", "observations", "pub-obs.md")
	if _, err := os.Stat(obsFile); os.IsNotExist(err) {
		t.Error("expected pub-obs.md to be created")
	}

	// Verify personal was NOT exported
	privFile := filepath.Join(tmpDir, "memory", "observations", "priv-obs.md")
	if _, err := os.Stat(privFile); !os.IsNotExist(err) {
		t.Error("personal observation should not be exported")
	}

	// Verify session was NOT exported
	sessFile := filepath.Join(tmpDir, "memory", "observations", "sess-mem.md")
	if _, err := os.Stat(sessFile); !os.IsNotExist(err) {
		t.Error("session memory should not be exported")
	}

	// Verify shared_at was updated
	got, _ := GetMemory(db, "pub-obs")
	if got.SharedAt == "" {
		t.Error("expected shared_at to be set after push")
	}
}

func TestPushIdempotent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)
	SaveMemory(db, Memory{ID: "idem-obs", EntityType: "observation", Title: "Idempotent", Content: "test", Author: "agent", Visibility: "project", CreatedAt: now})

	// First push
	Push(db, tmpDir, false)

	// Second push without --all should skip (shared_at is set)
	result, err := Push(db, tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Exported != 0 {
		t.Errorf("expected 0 exported on second push, got %d", result.Exported)
	}

	// Push with --all re-exports
	result, err = Push(db, tmpDir, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Exported != 1 {
		t.Errorf("expected 1 exported with --all, got %d", result.Exported)
	}
}

func TestPullImportsNew(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	obsDir := filepath.Join(tmpDir, "memory", "observations")
	os.MkdirAll(obsDir, 0o755)

	content := "---\nid: imported-1\nentity_type: observation\ntitle: Imported finding\nauthor: teammate\nvisibility: project\ncreated_at: 2026-03-20T00:00:00Z\n---\n\nSome discovery from a teammate"
	os.WriteFile(filepath.Join(obsDir, "imported-1.md"), []byte(content), 0o644)

	result, err := Pull(db, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result.Imported)
	}

	got, err := GetMemory(db, "imported-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Imported finding" {
		t.Errorf("expected title 'Imported finding', got %q", got.Title)
	}
}

func TestPullConflictDetection(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)

	// Save locally with one content
	SaveMemory(db, Memory{ID: "conflict-1", EntityType: "observation", Title: "Original", Content: "local version", Author: "agent", Visibility: "project", CreatedAt: now})

	// Write file with different content
	obsDir := filepath.Join(tmpDir, "memory", "observations")
	os.MkdirAll(obsDir, 0o755)
	content := "---\nid: conflict-1\nentity_type: observation\ntitle: Original\nauthor: teammate\nvisibility: project\ncreated_at: 2026-03-20T00:00:00Z\n---\n\nremote version"
	os.WriteFile(filepath.Join(obsDir, "conflict-1.md"), []byte(content), 0o644)

	result, err := Pull(db, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 {
		t.Errorf("expected 1 conflict, got %d", len(result.Conflicts))
	}
	if result.Conflicts[0].ID != "conflict-1" {
		t.Errorf("expected conflict on 'conflict-1', got %q", result.Conflicts[0].ID)
	}
}

func TestPullSkipsSameContent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)

	// Save and push
	SaveMemory(db, Memory{ID: "same-1", EntityType: "observation", Title: "Same title", Content: "same body", Author: "agent", Visibility: "project", CreatedAt: now})
	Push(db, tmpDir, false)

	// Pull — should skip since content matches
	result, err := Pull(db, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(result.Conflicts))
	}
}

func TestExportedFileHasValidFrontmatter(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)
	SaveMemory(db, Memory{ID: "fm-test", EntityType: "decision", Title: "Use SQLite", Content: "We decided to use SQLite", Author: "agent", Tags: "database,architecture", Visibility: "project", CreatedAt: now})

	Push(db, tmpDir, false)

	data, err := os.ReadFile(filepath.Join(tmpDir, "memory", "decisions", "fm-test.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		t.Error("expected file to start with frontmatter")
	}
	if !strings.Contains(content, "id: fm-test") {
		t.Error("expected frontmatter to contain id")
	}
	if !strings.Contains(content, "entity_type: decision") {
		t.Error("expected frontmatter to contain entity_type")
	}
	if !strings.Contains(content, "tags: database,architecture") {
		t.Error("expected frontmatter to contain tags")
	}
}
