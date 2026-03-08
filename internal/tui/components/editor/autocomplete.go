package editor

import (
	"sort"
	"strings"
)

// sqlCompletionKeywords are SQL keywords offered for autocompletion.
var sqlCompletionKeywords = []string{
	"SELECT", "FROM", "WHERE", "JOIN", "INSERT",
	"UPDATE", "DELETE", "CREATE", "DROP", "ALTER",
	"TABLE", "INTO", "VALUES", "SET", "ORDER",
	"BY", "GROUP", "HAVING", "LIMIT", "OFFSET",
	"AS", "ON", "AND", "OR", "NOT",
	"IN", "IS", "NULL", "LIKE", "BETWEEN",
	"EXISTS", "DISTINCT", "COUNT", "SUM", "AVG",
	"MIN", "MAX", "CASE", "WHEN", "THEN",
	"ELSE", "END", "UNION", "ALL", "DESC",
	"ASC", "LEFT", "RIGHT", "INNER", "OUTER",
	"CROSS", "SHOW", "DESCRIBE", "EXPLAIN", "USE",
	"INDEX", "PRIMARY", "KEY", "FOREIGN", "REFERENCES",
	"DEFAULT", "AUTO_INCREMENT", "IF", "DATABASE",
	"DATABASES", "TABLES", "COLUMNS",
}

// completionState holds the state for Tab-cycling through completions.
type completionState struct {
	active      bool     // currently cycling through completions
	prefix      string   // the partial word being completed
	prefixStart int      // byte offset in textarea where prefix starts
	matches     []string // matching candidates
	index       int      // current match index
}

// buildCompletions finds matching completions for the word under the cursor.
func buildCompletions(text string, cursorPos int, tableNames []string) completionState {
	// Find the start of the current word (walk back from cursor)
	start := cursorPos
	for start > 0 {
		ch := text[start-1]
		if isWordByte(ch) {
			start--
		} else {
			break
		}
	}

	if start == cursorPos {
		return completionState{} // no word to complete
	}

	prefix := text[start:cursorPos]
	upper := strings.ToUpper(prefix)

	var matches []string

	// Match table names first (higher priority)
	for _, t := range tableNames {
		if strings.HasPrefix(strings.ToUpper(t), upper) && !strings.EqualFold(t, prefix) {
			matches = append(matches, t)
		}
	}

	// Then SQL keywords
	for _, kw := range sqlCompletionKeywords {
		if strings.HasPrefix(kw, upper) && kw != upper {
			matches = append(matches, kw)
		}
	}

	sort.Strings(matches)

	if len(matches) == 0 {
		return completionState{}
	}

	return completionState{
		active:      true,
		prefix:      prefix,
		prefixStart: start,
		matches:     matches,
		index:       0,
	}
}

// current returns the currently selected completion.
func (cs *completionState) current() string {
	if !cs.active || len(cs.matches) == 0 {
		return ""
	}
	return cs.matches[cs.index]
}

// next cycles to the next completion.
func (cs *completionState) next() {
	if !cs.active || len(cs.matches) == 0 {
		return
	}
	cs.index = (cs.index + 1) % len(cs.matches)
}

// reset clears the completion state.
func (cs *completionState) reset() {
	cs.active = false
	cs.prefix = ""
	cs.prefixStart = 0
	cs.matches = nil
	cs.index = 0
}

func isWordByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}
