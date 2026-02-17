package history

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
}

func TestRecordAndQueryProjectHistory(t *testing.T) {
	store := openTestStore(t)

	repoID, err := store.EnsureRepo("/home/user/myproject")
	if err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}

	err = store.Record(repoID, "src", "run tests", "go test ./...", OutcomeAccepted)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	err = store.Record(repoID, "src", "run linter", "golangci-lint run", OutcomeAccepted)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	entries, err := store.ProjectHistory(repoID, "src", 20)
	if err != nil {
		t.Fatalf("ProjectHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Command != "golangci-lint run" {
		t.Errorf("expected most recent first, got %q", entries[0].Command)
	}
}

func TestRecentGlobalHistory(t *testing.T) {
	store := openTestStore(t)

	repo1, _ := store.EnsureRepo("/project1")
	repo2, _ := store.EnsureRepo("/project2")

	store.Record(repo1, ".", "build", "make build", OutcomeAccepted)
	store.Record(repo2, ".", "search", "rg pattern", OutcomeAccepted)
	store.Record(repo1, ".", "test", "go test", OutcomeRejected)

	entries, err := store.RecentGlobal(10)
	if err != nil {
		t.Fatalf("RecentGlobal: %v", err)
	}
	// Only accepted commands in global history
	if len(entries) != 2 {
		t.Fatalf("expected 2 accepted entries, got %d", len(entries))
	}
}

func TestEnsureRepoIsIdempotent(t *testing.T) {
	store := openTestStore(t)

	id1, _ := store.EnsureRepo("/same/path")
	id2, _ := store.EnsureRepo("/same/path")
	if id1 != id2 {
		t.Errorf("expected same id, got %d and %d", id1, id2)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}
