package database

import "time"

// ConnectionConfig holds all parameters needed to connect to a database.
type ConnectionConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	DSN      string // If provided, overrides individual fields
	TLS      string // "", "true", "skip-verify", or path to CA cert file
	TLSCert  string // Path to client certificate (optional, used with CA cert)
	TLSKey   string // Path to client key (optional, used with CA cert)
}

// TableInfo represents basic metadata about a database table.
type TableInfo struct {
	Name   string
	Engine string
	Rows   int64
}

// ColumnInfo represents a single column's metadata.
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Key      string  // PRI, UNI, MUL, or empty
	Default  *string
	Extra    string  // e.g. auto_increment
}

// SchemaInfo holds full schema details for a table.
type SchemaInfo struct {
	TableName string
	Columns   []ColumnInfo
	Engine    string
	Charset   string
	Collation string
	RowCount  int64
}

// QueryResult holds the output of an executed query.
type QueryResult struct {
	Columns         []string
	Rows            [][]string
	RowCount        int
	AffectedRows    int64
	Duration        time.Duration
	IsSelect        bool
	DatabaseChanged string // non-empty if a USE command changed the database
}
