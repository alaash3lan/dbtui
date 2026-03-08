package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryRing(t *testing.T) {
	h := NewHistoryRing(3)

	// Empty history
	_, ok := h.Previous()
	if ok {
		t.Error("Previous() on empty history should return false")
	}

	// Push and navigate
	h.Push("SELECT 1")
	h.Push("SELECT 2")
	h.Push("SELECT 3")

	entry, ok := h.Previous()
	if !ok || entry != "SELECT 3" {
		t.Errorf("Previous() = %q, want SELECT 3", entry)
	}

	entry, ok = h.Previous()
	if !ok || entry != "SELECT 2" {
		t.Errorf("Previous() = %q, want SELECT 2", entry)
	}

	entry, ok = h.Next()
	if !ok || entry != "SELECT 3" {
		t.Errorf("Next() = %q, want SELECT 3", entry)
	}

	// Max size enforcement
	h.Push("SELECT 4")
	if h.Len() != 3 {
		t.Errorf("Len() = %d after overflow, want 3", h.Len())
	}

	// Duplicate skip
	h.Push("SELECT 4")
	if h.Len() != 3 {
		t.Errorf("Len() = %d after duplicate, want 3", h.Len())
	}

	// Next past end returns empty
	h.Reset()
	h.Previous()
	h.Previous()
	h.Previous()
	h.Next()
	h.Next()
	entry, ok = h.Next()
	if !ok || entry != "" {
		t.Errorf("Next() past end = %q, want empty", entry)
	}
}

func TestHistoryLoadSaveFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")

	h := NewHistoryRing(100)
	h.Push("SELECT 1")
	h.Push("SELECT 2")
	h.Push("SELECT 3")

	if err := h.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile() error: %v", err)
	}

	h2 := NewHistoryRing(100)
	if err := h2.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	if h2.Len() != 3 {
		t.Fatalf("LoadFromFile() loaded %d entries, want 3", h2.Len())
	}

	entries := h2.Entries()
	want := []string{"SELECT 1", "SELECT 2", "SELECT 3"}
	for i, w := range want {
		if entries[i] != w {
			t.Errorf("entry[%d] = %q, want %q", i, entries[i], w)
		}
	}
}

func TestHistoryLoadSaveWithNewlines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")

	h := NewHistoryRing(100)
	h.Push("SELECT *\nFROM customers\nWHERE id = 1")
	h.Push("INSERT INTO t\nVALUES (1, 'a')")

	if err := h.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile() error: %v", err)
	}

	h2 := NewHistoryRing(100)
	if err := h2.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}

	entries := h2.Entries()
	if len(entries) != 2 {
		t.Fatalf("loaded %d entries, want 2", len(entries))
	}
	if entries[0] != "SELECT *\nFROM customers\nWHERE id = 1" {
		t.Errorf("entry[0] = %q, want multiline SELECT", entries[0])
	}
	if entries[1] != "INSERT INTO t\nVALUES (1, 'a')" {
		t.Errorf("entry[1] = %q, want multiline INSERT", entries[1])
	}
}

func TestHistoryLoadNonexistent(t *testing.T) {
	h := NewHistoryRing(100)
	err := h.LoadFromFile(filepath.Join(t.TempDir(), "does_not_exist"))
	if err == nil {
		t.Error("LoadFromFile() should return error for nonexistent file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("LoadFromFile() error should be 'not exist', got: %v", err)
	}
}

func TestHistoryEntries(t *testing.T) {
	h := NewHistoryRing(10)
	h.Push("query1")
	h.Push("query2")

	entries := h.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries() returned %d entries, want 2", len(entries))
	}

	// Mutating the returned slice should not affect the history.
	entries[0] = "modified"
	original := h.Entries()
	if original[0] != "query1" {
		t.Errorf("Entries() did not return a copy; internal state was modified")
	}
}
