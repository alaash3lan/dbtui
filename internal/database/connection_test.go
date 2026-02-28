package database

import (
	"testing"
)

func TestBuildDSN(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ConnectionConfig
		wantErr bool
	}{
		{
			name: "standard connection",
			cfg: ConnectionConfig{
				User:     "root",
				Password: "root",
				Host:     "127.0.0.1",
				Port:     3306,
				Database: "dbplus_test",
			},
			wantErr: false,
		},
		{
			name: "dsn override",
			cfg: ConnectionConfig{
				DSN: "root:root@tcp(127.0.0.1:3306)/dbplus_test",
			},
			wantErr: false,
		},
		{
			name: "missing user",
			cfg: ConnectionConfig{
				Host: "127.0.0.1",
			},
			wantErr: true,
		},
		{
			name:    "default host and port",
			cfg:     ConnectionConfig{User: "root", Password: "root"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn, err := buildDSN(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildDSN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && dsn == "" {
				t.Error("buildDSN() returned empty DSN")
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	cfg := ConnectionConfig{
		User:     "root",
		Password: "root",
		Host:     "127.0.0.1",
		Port:     3306,
		Database: "dbplus_test",
	}

	db, err := New(cfg)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer db.Close()

	// Test ListTables
	tables, err := db.ListTables()
	if err != nil {
		t.Fatalf("ListTables() error: %v", err)
	}
	if len(tables) != 4 {
		t.Errorf("ListTables() got %d tables, want 4", len(tables))
	}

	tableNames := make(map[string]bool)
	for _, tbl := range tables {
		tableNames[tbl.Name] = true
	}
	for _, expected := range []string{"customers", "products", "orders", "order_items"} {
		if !tableNames[expected] {
			t.Errorf("ListTables() missing table: %s", expected)
		}
	}

	// Test DescribeTable
	schema, err := db.DescribeTable("products")
	if err != nil {
		t.Fatalf("DescribeTable() error: %v", err)
	}
	if schema.Engine != "InnoDB" {
		t.Errorf("DescribeTable() engine = %s, want InnoDB", schema.Engine)
	}
	if len(schema.Columns) == 0 {
		t.Error("DescribeTable() returned no columns")
	}

	// Test FetchTableData
	result, err := db.FetchTableData("products", 100, 0)
	if err != nil {
		t.Fatalf("FetchTableData() error: %v", err)
	}
	if result.RowCount != 12 {
		t.Errorf("FetchTableData() got %d rows, want 12", result.RowCount)
	}
	if len(result.Columns) != 6 {
		t.Errorf("FetchTableData() got %d columns, want 6", len(result.Columns))
	}

	// Test Execute SELECT
	result, err = db.Execute("SELECT * FROM customers WHERE city = 'New York'")
	if err != nil {
		t.Fatalf("Execute() SELECT error: %v", err)
	}
	if !result.IsSelect {
		t.Error("Execute() SELECT should return IsSelect=true")
	}
	if result.RowCount != 1 {
		t.Errorf("Execute() SELECT got %d rows, want 1", result.RowCount)
	}

	// Test CountRows
	count, err := db.CountRows("customers")
	if err != nil {
		t.Fatalf("CountRows() error: %v", err)
	}
	if count != 10 {
		t.Errorf("CountRows() got %d, want 10", count)
	}

	// Test NULL handling
	result, err = db.Execute("SELECT email FROM customers WHERE name = 'Ivy Chen'")
	if err != nil {
		t.Fatalf("Execute() NULL test error: %v", err)
	}
	if result.RowCount != 1 || result.Rows[0][0] != "<NULL>" {
		t.Errorf("Execute() NULL handling: got %v, want <NULL>", result.Rows[0][0])
	}

	// Test Execute INSERT
	result, err = db.Execute("INSERT INTO customers (name, email, city) VALUES ('Test User', 'test@test.com', 'TestCity')")
	if err != nil {
		t.Fatalf("Execute() INSERT error: %v", err)
	}
	if result.IsSelect {
		t.Error("Execute() INSERT should return IsSelect=false")
	}
	if result.AffectedRows != 1 {
		t.Errorf("Execute() INSERT affected rows = %d, want 1", result.AffectedRows)
	}

	// Cleanup
	db.Execute("DELETE FROM customers WHERE name = 'Test User'")

	t.Logf("All integration tests passed. Tables: %d, Products: %d rows", len(tables), 12)
}
