package editor

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// sqlKeywords is the set of SQL keywords to highlight, stored in upper-case for
// case-insensitive matching.
var sqlKeywords = map[string]struct{}{
	"SELECT": {}, "FROM": {}, "WHERE": {}, "JOIN": {}, "INSERT": {},
	"UPDATE": {}, "DELETE": {}, "CREATE": {}, "DROP": {}, "ALTER": {},
	"TABLE": {}, "INTO": {}, "VALUES": {}, "SET": {}, "ORDER": {},
	"BY": {}, "GROUP": {}, "HAVING": {}, "LIMIT": {}, "OFFSET": {},
	"AS": {}, "ON": {}, "AND": {}, "OR": {}, "NOT": {},
	"IN": {}, "IS": {}, "NULL": {}, "LIKE": {}, "BETWEEN": {},
	"EXISTS": {}, "DISTINCT": {}, "COUNT": {}, "SUM": {}, "AVG": {},
	"MIN": {}, "MAX": {}, "CASE": {}, "WHEN": {}, "THEN": {},
	"ELSE": {}, "END": {}, "UNION": {}, "ALL": {}, "DESC": {},
	"ASC": {}, "LEFT": {}, "RIGHT": {}, "INNER": {}, "OUTER": {},
	"CROSS": {}, "SHOW": {}, "DESCRIBE": {}, "EXPLAIN": {}, "USE": {},
	"INDEX": {}, "PRIMARY": {}, "KEY": {}, "FOREIGN": {}, "REFERENCES": {},
	"DEFAULT": {}, "AUTO_INCREMENT": {}, "IF": {}, "DATABASE": {},
	"DATABASES": {}, "TABLES": {}, "COLUMNS": {},
}

// highlightSQL applies syntax highlighting to SQL text. It colorizes keywords,
// string literals, and numeric literals using lipgloss styles.
func highlightSQL(input string, keywordColor, stringColor, numberColor lipgloss.Color) string {
	kwStyle := lipgloss.NewStyle().Foreground(keywordColor).Bold(true)
	strStyle := lipgloss.NewStyle().Foreground(stringColor)
	numStyle := lipgloss.NewStyle().Foreground(numberColor)

	var b strings.Builder
	b.Grow(len(input) * 2)

	runes := []rune(input)
	i := 0
	for i < len(runes) {
		ch := runes[i]

		// String literals: single or double quoted
		if ch == '\'' || ch == '"' {
			j := i + 1
			quote := ch
			for j < len(runes) {
				if runes[j] == '\\' && j+1 < len(runes) {
					j += 2
					continue
				}
				if runes[j] == quote {
					j++
					break
				}
				j++
			}
			b.WriteString(strStyle.Render(string(runes[i:j])))
			i = j
			continue
		}

		// Numbers: digits optionally followed by more digits/dots
		if unicode.IsDigit(ch) && (i == 0 || !isWordChar(runes[i-1])) {
			j := i + 1
			hasDot := false
			for j < len(runes) {
				if unicode.IsDigit(runes[j]) {
					j++
				} else if runes[j] == '.' && !hasDot {
					hasDot = true
					j++
				} else {
					break
				}
			}
			// Only highlight if the token ends at a non-word character
			if j >= len(runes) || !isWordChar(runes[j]) {
				b.WriteString(numStyle.Render(string(runes[i:j])))
				i = j
				continue
			}
		}

		// Words: potential keywords
		if isWordStartChar(ch) {
			j := i + 1
			for j < len(runes) && isWordChar(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			upper := strings.ToUpper(word)
			if _, ok := sqlKeywords[upper]; ok {
				b.WriteString(kwStyle.Render(word))
			} else {
				b.WriteString(word)
			}
			i = j
			continue
		}

		// Everything else: pass through
		b.WriteRune(ch)
		i++
	}

	return b.String()
}

// highlightRenderedSQL applies keyword highlighting to text that may contain
// ANSI escape sequences (e.g. from the textarea widget's cursor rendering).
// It skips over escape sequences and only colorizes bare words.
func highlightRenderedSQL(input string, keywordColor, stringColor, numberColor lipgloss.Color) string {
	kwStyle := lipgloss.NewStyle().Foreground(keywordColor).Bold(true)
	strStyle := lipgloss.NewStyle().Foreground(stringColor)
	numStyle := lipgloss.NewStyle().Foreground(numberColor)

	var b strings.Builder
	b.Grow(len(input) * 2)

	runes := []rune(input)
	i := 0
	for i < len(runes) {
		ch := runes[i]

		// Skip ANSI escape sequences: ESC[ ... final_byte
		if ch == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			j := i + 2
			for j < len(runes) {
				if (runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z') || runes[j] == '~' {
					j++
					break
				}
				j++
			}
			b.WriteString(string(runes[i:j]))
			i = j
			continue
		}

		// Also skip OSC sequences: ESC] ... ST
		if ch == '\x1b' && i+1 < len(runes) && runes[i+1] == ']' {
			j := i + 2
			for j < len(runes) {
				if runes[j] == '\x1b' || runes[j] == '\x07' {
					if runes[j] == '\x07' {
						j++
						break
					}
					if j+1 < len(runes) && runes[j+1] == '\\' {
						j += 2
						break
					}
				}
				j++
			}
			b.WriteString(string(runes[i:j]))
			i = j
			continue
		}

		// String literals
		if ch == '\'' || ch == '"' {
			j := i + 1
			quote := ch
			for j < len(runes) && runes[j] != '\x1b' {
				if runes[j] == '\\' && j+1 < len(runes) {
					j += 2
					continue
				}
				if runes[j] == quote {
					j++
					break
				}
				j++
			}
			b.WriteString(strStyle.Render(string(runes[i:j])))
			i = j
			continue
		}

		// Numbers
		if unicode.IsDigit(ch) && (i == 0 || !isWordChar(runes[i-1])) {
			j := i + 1
			hasDot := false
			for j < len(runes) && runes[j] != '\x1b' {
				if unicode.IsDigit(runes[j]) {
					j++
				} else if runes[j] == '.' && !hasDot {
					hasDot = true
					j++
				} else {
					break
				}
			}
			if j >= len(runes) || !isWordChar(runes[j]) {
				b.WriteString(numStyle.Render(string(runes[i:j])))
				i = j
				continue
			}
		}

		// Words: potential keywords
		if isWordStartChar(ch) {
			j := i + 1
			for j < len(runes) && isWordChar(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			upper := strings.ToUpper(word)
			if _, ok := sqlKeywords[upper]; ok {
				b.WriteString(kwStyle.Render(word))
			} else {
				b.WriteString(word)
			}
			i = j
			continue
		}

		b.WriteRune(ch)
		i++
	}

	return b.String()
}

// isWordChar returns true for characters that can appear inside identifiers.
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// isWordStartChar returns true for characters that can start an identifier.
func isWordStartChar(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
