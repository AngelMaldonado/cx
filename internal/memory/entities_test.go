package memory

import (
	"testing"
	"time"
)

func TestSaveAndGetMemory(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	m := Memory{
		ID:         "test-obs-1",
		EntityType: "observation",
		Title:      "Test observation",
		Content:    "Found a bug in the auth module",
		Author:     "agent",
		Visibility: "project",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := SaveMemory(db, m); err != nil {
		t.Fatal(err)
	}

	got, err := GetMemory(db, "test-obs-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Test observation" {
		t.Errorf("expected title 'Test observation', got %q", got.Title)
	}
	if got.EntityType != "observation" {
		t.Errorf("expected entity_type 'observation', got %q", got.EntityType)
	}
}

func TestSaveMemoryValidation(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	err := SaveMemory(db, Memory{})
	if err == nil {
		t.Error("expected error for empty memory")
	}
}

func TestDeprecationChain(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	a := Memory{ID: "dec-a", EntityType: "decision", Title: "Decision A", Content: "First", Author: "agent", Status: "active", Visibility: "project", CreatedAt: now}
	b := Memory{ID: "dec-b", EntityType: "decision", Title: "Decision B", Content: "Replaces A", Author: "agent", Deprecates: "dec-a", Status: "active", Visibility: "project", CreatedAt: now}

	if err := SaveMemory(db, a); err != nil {
		t.Fatal(err)
	}
	if err := SaveMemory(db, b); err != nil {
		t.Fatal(err)
	}

	got, err := GetMemory(db, "dec-a")
	if err != nil {
		t.Fatal(err)
	}
	if got.Deprecated != 1 {
		t.Error("expected dec-a to be deprecated")
	}
	if got.Status != "superseded" {
		t.Errorf("expected status 'superseded', got %q", got.Status)
	}
}

func TestListMemoriesExcludesDeprecated(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	SaveMemory(db, Memory{ID: "a", EntityType: "observation", Title: "A", Content: "a", Author: "x", Visibility: "project", CreatedAt: now})
	SaveMemory(db, Memory{ID: "b", EntityType: "observation", Title: "B", Content: "b replaces a", Author: "x", Deprecates: "a", Visibility: "project", CreatedAt: now})

	results, err := ListMemories(db, ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (deprecated excluded), got %d", len(results))
	}

	// With include deprecated
	results, err = ListMemories(db, ListOpts{IncludeDeprecated: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results (deprecated included), got %d", len(results))
	}
}

func TestSessionCRUD(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	s := Session{ID: "sess-1", Mode: "build", Goal: "Implement feature X"}
	if err := SaveSession(db, s); err != nil {
		t.Fatal(err)
	}

	got, err := GetLatestSession(db)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "sess-1" || got.Mode != "build" {
		t.Errorf("unexpected session: %+v", got)
	}
}

func TestAgentRunCRUD(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	run := AgentRun{ID: "run-1", AgentType: "scout", ResultStatus: "success", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := SaveAgentRun(db, run); err != nil {
		t.Fatal(err)
	}

	runs, err := ListAgentRuns(db, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].AgentType != "scout" {
		t.Errorf("unexpected runs: %+v", runs)
	}
}

func TestDeprecateMemory(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	m := Memory{ID: "dep-me", EntityType: "observation", Title: "To be deprecated", Content: "c", Author: "agent", Visibility: "project", CreatedAt: now}
	if err := SaveMemory(db, m); err != nil {
		t.Fatal(err)
	}

	// Deprecate the memory
	if err := DeprecateMemory(db, "dep-me"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	got, err := GetMemory(db, "dep-me")
	if err != nil {
		t.Fatal(err)
	}
	if got.Deprecated != 1 {
		t.Error("expected memory to be deprecated")
	}

	// Deprecating again should return an error
	if err := DeprecateMemory(db, "dep-me"); err == nil {
		t.Error("expected error when deprecating already-deprecated memory")
	}

	// Non-existent ID should return an error
	if err := DeprecateMemory(db, "does-not-exist"); err == nil {
		t.Error("expected error for non-existent memory id")
	}
}

func TestMemoryLink(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	SaveMemory(db, Memory{ID: "m1", EntityType: "observation", Title: "M1", Content: "c1", Author: "a", Visibility: "project", CreatedAt: now})
	SaveMemory(db, Memory{ID: "m2", EntityType: "observation", Title: "M2", Content: "c2", Author: "a", Visibility: "project", CreatedAt: now})

	link := MemoryLink{FromID: "m1", ToID: "m2", RelationType: "related-to"}
	if err := SaveMemoryLink(db, link); err != nil {
		t.Fatal(err)
	}

	// Saving same link again should not error (INSERT OR IGNORE)
	if err := SaveMemoryLink(db, link); err != nil {
		t.Fatal(err)
	}
}
