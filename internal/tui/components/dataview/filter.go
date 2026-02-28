package dataview

import "strings"

// FilterResult holds the parsed filter parameters.
type FilterResult struct {
	Column string // empty means search all columns
	Value  string
}

// ParseFilter parses a filter string in the format "column | value" or just "value".
func ParseFilter(input string) FilterResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return FilterResult{}
	}

	parts := strings.SplitN(input, "|", 2)
	if len(parts) == 2 {
		col := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if col != "" && val != "" {
			return FilterResult{Column: col, Value: val}
		}
		if val != "" {
			return FilterResult{Value: val}
		}
		if col != "" {
			return FilterResult{Value: col}
		}
		return FilterResult{}
	}

	return FilterResult{Value: input}
}

// ApplyFilter filters rows based on the filter result. Returns filtered rows.
func ApplyFilter(columns []string, rows [][]string, f FilterResult) [][]string {
	if f.Value == "" {
		return rows
	}

	needle := strings.ToLower(f.Value)

	// Find column index if specified
	colIdx := -1
	if f.Column != "" {
		colLower := strings.ToLower(f.Column)
		for i, c := range columns {
			if strings.ToLower(c) == colLower {
				colIdx = i
				break
			}
		}
		// If specified column not found, search all
		if colIdx == -1 {
			colIdx = -1
		}
	}

	var filtered [][]string
	for _, row := range rows {
		if colIdx >= 0 {
			// Search specific column
			if colIdx < len(row) && strings.Contains(strings.ToLower(row[colIdx]), needle) {
				filtered = append(filtered, row)
			}
		} else {
			// Search all columns
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), needle) {
					filtered = append(filtered, row)
					break
				}
			}
		}
	}

	return filtered
}
