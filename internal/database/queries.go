package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
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
				} else if v, ok := val.(uint64); ok {
					info.Rows = int64(v)
				} else if v, ok := val.([]byte); ok {
					if n, err := strconv.ParseInt(string(v), 10, 64); err == nil {
						info.Rows = n
					}
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

	// Get table status (engine, charset, row count, sizes, timestamps)
	var (
		engine, collation, createTime, updateTime, comment sql.NullString
		tableRows, autoIncr                                sql.NullInt64
		dataLength, indexLength                             sql.NullInt64
	)
	row := db.conn.QueryRowContext(ctx, `
		SELECT ENGINE, TABLE_COLLATION, TABLE_ROWS, AUTO_INCREMENT,
		       DATA_LENGTH, INDEX_LENGTH, CREATE_TIME, UPDATE_TIME, TABLE_COMMENT
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`, name)
	if err := row.Scan(&engine, &collation, &tableRows, &autoIncr,
		&dataLength, &indexLength, &createTime, &updateTime, &comment); err == nil {
		info.Engine = engine.String
		info.Collation = collation.String
		info.RowCount = tableRows.Int64
		info.AutoIncr = autoIncr.Int64
		info.CreateTime = createTime.String
		info.UpdateTime = updateTime.String
		info.Comment = comment.String
		if dataLength.Valid {
			info.DataSize = formatBytes(dataLength.Int64 + indexLength.Int64)
		}
		// Extract charset from collation (e.g. utf8mb4_general_ci -> utf8mb4)
		for i, c := range collation.String {
			if c == '_' {
				info.Charset = collation.String[:i]
				break
			}
		}
	}

	// Get indexes
	idxRows, err := db.conn.QueryContext(ctx, fmt.Sprintf("SHOW INDEX FROM `%s`", name))
	if err == nil {
		defer idxRows.Close()
		idxCols, _ := idxRows.Columns()
		idxMap := make(map[string]*IndexInfo)
		var idxOrder []string
		for idxRows.Next() {
			vals := make([]interface{}, len(idxCols))
			ptrs := make([]interface{}, len(idxCols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := idxRows.Scan(ptrs...); err != nil {
				continue
			}
			colMap := make(map[string]string, len(idxCols))
			for i, col := range idxCols {
				switch v := vals[i].(type) {
				case []byte:
					colMap[col] = string(v)
				case int64:
					colMap[col] = strconv.FormatInt(v, 10)
				case uint64:
					colMap[col] = strconv.FormatUint(v, 10)
				default:
					colMap[col] = fmt.Sprintf("%v", v)
				}
			}
			keyName := colMap["Key_name"]
			colName := colMap["Column_name"]
			nonUnique := colMap["Non_unique"]
			idxType := colMap["Index_type"]

			if idx, ok := idxMap[keyName]; ok {
				idx.Columns = append(idx.Columns, colName)
			} else {
				idx := &IndexInfo{
					Name:    keyName,
					Columns: []string{colName},
					Unique:  nonUnique == "0",
					Type:    idxType,
				}
				idxMap[keyName] = idx
				idxOrder = append(idxOrder, keyName)
			}
		}
		for _, k := range idxOrder {
			info.Indexes = append(info.Indexes, *idxMap[k])
		}
	}

	// Get foreign keys
	fkRows, err := db.conn.QueryContext(ctx, `
		SELECT CONSTRAINT_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME,
		       REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		  AND REFERENCED_TABLE_NAME IS NOT NULL
		ORDER BY CONSTRAINT_NAME, ORDINAL_POSITION`, name)
	if err == nil {
		defer fkRows.Close()
		fkMap := make(map[string]*ForeignKeyInfo)
		var fkOrder []string
		for fkRows.Next() {
			var cName, colName, refTable, refCol string
			if err := fkRows.Scan(&cName, &colName, &refTable, &refCol); err != nil {
				continue
			}
			if fk, ok := fkMap[cName]; ok {
				fk.Columns = append(fk.Columns, colName)
				fk.RefColumns = append(fk.RefColumns, refCol)
			} else {
				fk := &ForeignKeyInfo{
					Name:       cName,
					Columns:    []string{colName},
					RefTable:   refTable,
					RefColumns: []string{refCol},
				}
				fkMap[cName] = fk
				fkOrder = append(fkOrder, cName)
			}
		}
		// Get ON DELETE / ON UPDATE rules
		for _, cName := range fkOrder {
			fk := fkMap[cName]
			var onDel, onUpd string
			ruleRow := db.conn.QueryRowContext(ctx, `
				SELECT DELETE_RULE, UPDATE_RULE
				FROM information_schema.REFERENTIAL_CONSTRAINTS
				WHERE CONSTRAINT_SCHEMA = DATABASE() AND TABLE_NAME = ?
				  AND CONSTRAINT_NAME = ?`, name, cName)
			if err := ruleRow.Scan(&onDel, &onUpd); err == nil {
				fk.OnDelete = onDel
				fk.OnUpdate = onUpd
			}
			info.ForeignKeys = append(info.ForeignKeys, *fk)
		}
	}

	return info, nil
}

// formatBytes returns a human-readable byte size.
func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
