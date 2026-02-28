package editor

import "testing"

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
