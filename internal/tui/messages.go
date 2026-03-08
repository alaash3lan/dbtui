package tui

import (
	"time"

	"github.com/alaa/dbtui/internal/database"
)

// TableListMsg is sent when the table list has been fetched.
type TableListMsg struct {
	Tables []database.TableInfo
	Err    error
}

// QueryResultMsg is sent when a query finishes executing.
type QueryResultMsg struct {
	Columns         []string
	Rows            [][]string
	RowCount        int
	AffectedRows    int64
	Duration        time.Duration
	IsSelect        bool
	Err             error
	DatabaseChanged string
}

// SchemaInfoMsg is sent when schema info has been fetched.
type SchemaInfoMsg struct {
	Info *database.SchemaInfo
	Err  error
}

// exportRequestMsg is sent when the user triggers an export.
type exportRequestMsg struct {
	Format string // "csv" or "json"
}

// exportResultMsg carries the result of an export operation.
type exportResultMsg struct {
	Path string
	Err  error
}

// clipboardResultMsg carries the result of a clipboard operation.
type clipboardResultMsg struct {
	Err error
}

// databaseListMsg is sent when the database list has been fetched.
type databaseListMsg struct {
	Databases []string
	Err       error
}

// switchDatabaseResultMsg carries the result of a database switch.
type switchDatabaseResultMsg struct {
	Name string
	Err  error
}

// reconnectMsg triggers a reconnect attempt.
type reconnectMsg struct{}

// reconnectResultMsg carries the result of a reconnect attempt.
type reconnectResultMsg struct {
	Err error
}

// deleteRowResultMsg carries the result of a row deletion.
type deleteRowResultMsg struct {
	AffectedRows int64
	Err          error
}

// bookmarkSelectedMsg is sent when the user picks a saved query bookmark.
type bookmarkSelectedMsg struct {
	SQL string
}

// bookmarkSavedMsg is sent after saving a query bookmark.
type bookmarkSavedMsg struct {
	Name string
	Err  error
}

// favoriteSavedMsg is sent after persisting table favorites to disk.
type favoriteSavedMsg struct {
	Err error
}

