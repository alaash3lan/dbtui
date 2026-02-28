package editor

// HistoryRing is a fixed-size ring buffer for query history.
type HistoryRing struct {
	entries []string
	maxSize int
	cursor  int // navigation cursor, -1 means "not navigating"
}

// NewHistoryRing creates a history buffer with the given capacity.
func NewHistoryRing(maxSize int) *HistoryRing {
	return &HistoryRing{
		maxSize: maxSize,
		cursor:  -1,
	}
}

// Push adds a query to the history. Duplicates of the last entry are skipped.
func (h *HistoryRing) Push(query string) {
	if query == "" {
		return
	}
	// Skip duplicate of the most recent entry
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == query {
		h.Reset()
		return
	}

	h.entries = append(h.entries, query)
	if len(h.entries) > h.maxSize {
		h.entries = h.entries[1:]
	}
	h.Reset()
}

// Previous moves the cursor back and returns the entry.
func (h *HistoryRing) Previous() (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}

	if h.cursor == -1 {
		h.cursor = len(h.entries) - 1
	} else if h.cursor > 0 {
		h.cursor--
	} else {
		return h.entries[0], true
	}

	return h.entries[h.cursor], true
}

// Next moves the cursor forward and returns the entry.
// Returns empty string when past the end (back to current input).
func (h *HistoryRing) Next() (string, bool) {
	if h.cursor == -1 {
		return "", false
	}

	h.cursor++
	if h.cursor >= len(h.entries) {
		h.cursor = -1
		return "", true // signal to restore original input
	}

	return h.entries[h.cursor], true
}

// Reset puts the cursor back to the default (not navigating) state.
func (h *HistoryRing) Reset() {
	h.cursor = -1
}

// Len returns the number of entries.
func (h *HistoryRing) Len() int {
	return len(h.entries)
}
