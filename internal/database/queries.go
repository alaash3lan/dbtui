package database

import (
	"context"
	"fmt"
)

// ListTables returns all tables in the current database with basic metadata.
func (db *DB) ListTables(ctx context.Context) ([]TableInfo, error) {
	rows, err := db.conn.QueryContext(ctx, "SHOW TABLE STATUS")
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to read columns: %w", err)
	}

	var tables []TableInfo
	for rows.Next() {
		// SHOW TABLE STATUS returns many columns; we only need Name, Engine, Rows
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		info := TableInfo{}
		for i, col := range columns {
			val := values[i]
			switch col {
			case "Name":
				if v, ok := val.([]byte); ok {
					info.Name = string(v)
				}
			case "Engine":
				if v, ok := val.([]byte); ok {
					info.Engine = string(v)
				}
			case "Rows":
				if v, ok := val.(int64); ok {
					info.Rows = v
				}
			}
		}

		if info.Name != "" {
			tables = append(tables, info)
		}
	}

	return tables, rows.Err()
}

// DescribeTable returns full schema information for a table.
func (db *DB) DescribeTable(ctx context.Context, name string) (*SchemaInfo, error) {
	if err := validIdentifier(name); err != nil {
		return nil, err
	}
	info := &SchemaInfo{TableName: name}

	// Get column details
	rows, err := db.conn.QueryContext(ctx, fmt.Sprintf("DESCRIBE `%s`", name))
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var nullable, key, extra string
		var defaultVal *string

		if err := rows.Scan(&col.Name, &col.Type, &nullable, &key, &defaultVal, &extra); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}

		col.Nullable = nullable == "YES"
		col.Key = key
		col.Default = defaultVal
		col.Extra = extra
		info.Columns = append(info.Columns, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get table status (engine, charset, row count)
	var tableStatus struct {
		engine    string
		collation string
		rows      int64
	}
	row := db.conn.QueryRowContext(ctx, "SELECT ENGINE, TABLE_COLLATION, TABLE_ROWS FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", name)
	if err := row.Scan(&tableStatus.engine, &tableStatus.collation, &tableStatus.rows); err == nil {
		info.Engine = tableStatus.engine
		info.Collation = tableStatus.collation
		info.RowCount = tableStatus.rows
		// Extract charset from collation (e.g. utf8mb4_general_ci -> utf8mb4)
		for i, c := range tableStatus.collation {
			if c == '_' {
				info.Charset = tableStatus.collation[:i]
				break
			}
		}
	}

	return info, nil
}
