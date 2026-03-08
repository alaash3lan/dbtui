package database

import (
	"database/sql"
	"testing"
	"time"
)

func TestValidIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple name", "customers", false},
		{"valid with underscore", "order_items", false},
		{"valid with numbers", "table123", false},
		{"empty string", "", true},
		{"backtick injection", "users`; DROP TABLE--", true},
		{"null byte", "users\x00evil", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validIdentifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validIdentifier(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	// Build a minimal colTypes slice for testing byte handling.
	// We pass nil colTypes for most tests, and test binary detection separately.
	now := time.Date(2024, 6, 15, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name     string
		val      any
		colTypes []*sql.ColumnType
		idx      int
		want     string
	}{
		{"nil value", nil, nil, 0, "<NULL>"},
		{"string value", "hello", nil, 0, "hello"},
		{"int64 value", int64(42), nil, 0, "42"},
		{"float64 value", float64(3.14), nil, 0, "3.14"},
		{"bool true", true, nil, 0, "1"},
		{"bool false", false, nil, 0, "0"},
		{"time value", now, nil, 0, "2024-06-15 10:30:45"},
		{"printable bytes", []byte("hello world"), nil, 0, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.val, tt.colTypes, tt.idx)
			if got != tt.want {
				t.Errorf("formatValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsPrintable(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"printable ASCII", []byte("Hello, World!"), true},
		{"with newline", []byte("line1\nline2"), true},
		{"with tab", []byte("col1\tcol2"), true},
		{"with carriage return", []byte("line1\r\nline2"), true},
		{"binary data", []byte{0x00, 0x01, 0x02}, false},
		{"control char BEL", []byte{0x07}, false},
		{"mixed printable and binary", []byte("hello\x00world"), false},
		{"empty bytes", []byte{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPrintable(tt.input)
			if got != tt.want {
				t.Errorf("isPrintable(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
