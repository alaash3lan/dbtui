package dataview

import "testing"

func TestParseFilter(t *testing.T) {
	tests := []struct {
		input      string
		wantCol    string
		wantVal    string
	}{
		{"", "", ""},
		{"quantum", "", "quantum"},
		{"name | Quantum", "name", "Quantum"},
		{"  name  |  Quantum  ", "name", "Quantum"},
		{"| value", "", "value"},
		{"col |", "", "col"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			f := ParseFilter(tt.input)
			if f.Column != tt.wantCol {
				t.Errorf("ParseFilter(%q).Column = %q, want %q", tt.input, f.Column, tt.wantCol)
			}
			if f.Value != tt.wantVal {
				t.Errorf("ParseFilter(%q).Value = %q, want %q", tt.input, f.Value, tt.wantVal)
			}
		})
	}
}

func TestApplyFilter(t *testing.T) {
	columns := []string{"name", "city"}
	rows := [][]string{
		{"Alice", "New York"},
		{"Bob", "London"},
		{"Charlie", "New York"},
		{"Diana", "Tokyo"},
	}

	// All-column search
	result := ApplyFilter(columns, rows, FilterResult{Value: "new york"})
	if len(result) != 2 {
		t.Errorf("all-column filter: got %d, want 2", len(result))
	}

	// Column-specific search
	result = ApplyFilter(columns, rows, FilterResult{Column: "name", Value: "ali"})
	if len(result) != 1 {
		t.Errorf("column filter: got %d, want 1", len(result))
	}

	// No match
	result = ApplyFilter(columns, rows, FilterResult{Value: "zzzzz"})
	if len(result) != 0 {
		t.Errorf("no match filter: got %d, want 0", len(result))
	}

	// Empty filter returns all
	result = ApplyFilter(columns, rows, FilterResult{})
	if len(result) != 4 {
		t.Errorf("empty filter: got %d, want 4", len(result))
	}
}
