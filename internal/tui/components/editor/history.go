package editor

import (
	"os"
	"path/filepath"
	"strings"
)

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

// Entries returns a copy of all history entries.
func (h *HistoryRing) Entries() []string {
	out := make([]string, len(h.entries))
	copy(out, h.entries)
	return out
}

// LoadFromFile reads history from a file. Each line is one query;
// literal newlines within queries are escaped as \n.
func (h *HistoryRing) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		query := unescapeLine(line)
		h.Push(query)
	}
	return nil
}

// SaveToFile writes history entries to a file, creating parent
// directories if needed. Newlines within queries are escaped as \n.
func (h *HistoryRing) SaveToFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	var b strings.Builder
	for _, entry := range h.entries {
		line := strings.ReplaceAll(entry, `\`, `\\`)
		line = strings.ReplaceAll(line, "\n", `\n`)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

// unescapeLine reverses the escaping applied by SaveToFile.
// It handles \n → newline and \\ → backslash without double-processing.
func unescapeLine(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
			case '\\':
				b.WriteByte('\\')
				i++
			default:
				b.WriteByte(s[i])
			}
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
