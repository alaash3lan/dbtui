package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// validIdentifier rejects table/column names that could cause SQL injection.
func validIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if strings.ContainsAny(name, "`\x00") {
		return fmt.Errorf("identifier contains invalid characters: %q", name)
	}
	return nil
}

// FetchTableData runs SELECT * with pagination on a table.
func (db *DB) FetchTableData(ctx context.Context, table string, limit, offset int) (*QueryResult, error) {
	if err := validIdentifier(table); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d OFFSET %d", table, limit, offset)
	return db.executeSelect(ctx, query)
}

// Execute runs an arbitrary SQL statement and returns results.
func (db *DB) Execute(ctx context.Context, query string) (*QueryResult, error) {
	trimmed := strings.TrimSpace(strings.ToUpper(query))

	// Detect USE database
	if strings.HasPrefix(trimmed, "USE ") {
		// Extract db name from the trimmed query to handle leading whitespace
		trimmedQuery := strings.TrimSpace(query)
		dbName := trimmedQuery[4:]
		dbName = strings.TrimRight(dbName, ";")
		dbName = strings.TrimSpace(dbName)
		dbName = strings.Trim(dbName, "`")
		if err := db.SwitchDatabase(ctx, dbName); err != nil {
			return nil, err
		}
		return &QueryResult{
			IsSelect:     false,
			AffectedRows: 0,
			DatabaseChanged: dbName,
		}, nil
	}

	// Detect SELECT queries
	if strings.HasPrefix(trimmed, "SELECT") || strings.HasPrefix(trimmed, "SHOW") ||
		strings.HasPrefix(trimmed, "DESCRIBE") || strings.HasPrefix(trimmed, "EXPLAIN") {
		return db.executeSelect(ctx, query)
	}

	return db.executeExec(ctx, query)
}

// CountRows returns the total row count for a table.
func (db *DB) CountRows(ctx context.Context, table string) (int64, error) {
	if err := validIdentifier(table); err != nil {
		return 0, err
	}
	var count int64
	err := db.conn.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)).Scan(&count)
	return count, err
}

func (db *DB) executeSelect(ctx context.Context, query string) (*QueryResult, error) {
	start := time.Now()

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		columnTypes = nil
	}

	var result [][]string
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make([]string, len(columns))
		for i, val := range values {
			row[i] = formatValue(val, columnTypes, i)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &QueryResult{
		Columns:  columns,
		Rows:     result,
		RowCount: len(result),
		Duration: time.Since(start),
		IsSelect: true,
	}, nil
}

func (db *DB) executeExec(ctx context.Context, query string) (*QueryResult, error) {
	start := time.Now()

	res, err := db.conn.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}

	affected, _ := res.RowsAffected()

	return &QueryResult{
		AffectedRows: affected,
		Duration:     time.Since(start),
		IsSelect:     false,
	}, nil
}

func formatValue(val interface{}, colTypes []*sql.ColumnType, idx int) string {
	if val == nil {
		return "<NULL>"
	}

	switch v := val.(type) {
	case []byte:
		// Check if this is a binary/blob column
		if idx < len(colTypes) {
			typeName := strings.ToUpper(colTypes[idx].DatabaseTypeName())
			if strings.Contains(typeName, "BLOB") || strings.Contains(typeName, "BINARY") {
				if !isPrintable(v) {
					return fmt.Sprintf("<BINARY len=%d>", len(v))
				}
			}
		}
		return string(v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func isPrintable(b []byte) bool {
	for _, c := range b {
		if c < 32 && c != '\n' && c != '\r' && c != '\t' {
			return false
		}
	}
	return true
}
