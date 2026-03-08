package database

import (
	"errors"
	"testing"
)

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"connection refused", errors.New("dial tcp: connection refused"), true},
		{"broken pipe", errors.New("write: broken pipe"), true},
		{"unexpected EOF", errors.New("unexpected EOF"), true},
		{"bad connection", errors.New("driver: bad connection"), true},
		{"invalid connection", errors.New("invalid connection"), true},
		{"connection reset", errors.New("read: connection reset by peer"), true},
		{"connection was aborted", errors.New("connection was aborted"), true},
		{"case insensitive", errors.New("Connection Refused"), true},
		{"syntax error", errors.New("Error 1064: You have an error in your SQL syntax"), false},
		{"table not found", errors.New("Error 1146: Table 'test.foo' doesn't exist"), false},
		{"generic error", errors.New("something went wrong"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConnectionError(tt.err)
			if got != tt.want {
				t.Errorf("IsConnectionError(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsConnectionErrorNil(t *testing.T) {
	if IsConnectionError(nil) {
		t.Error("IsConnectionError(nil) should return false")
	}
}
