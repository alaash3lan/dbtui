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

// TableSelectedMsg is sent when the user selects a table in the sidebar.
type TableSelectedMsg struct {
	TableName string
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

// ConnectionErrorMsg signals a connection problem.
type ConnectionErrorMsg struct {
	Err error
}

// FilterChangedMsg is sent when the data filter changes.
type FilterChangedMsg struct {
	Column string
	Value  string
}
