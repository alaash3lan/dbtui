package database

import (
	"context"
	"os"
	"strconv"
	"testing"
)

func testConfig() ConnectionConfig {
	user := os.Getenv("DBTUI_TEST_USER")
	if user == "" {
		user = "root"
	}
	pass := os.Getenv("DBTUI_TEST_PASS")
	if pass == "" {
		pass = "root"
	}
	host := os.Getenv("DBTUI_TEST_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := 3306
	if p := os.Getenv("DBTUI_TEST_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	db := os.Getenv("DBTUI_TEST_DB")
	if db == "" {
		db = "dbtui_test"
	}
	return ConnectionConfig{
		User:     user,
		Password: pass,
		Host:     host,
		Port:     port,
		Database: db,
	}
}

func TestBuildDSN(t *testing.T) {
	cfg := testConfig()
	tests := []struct {
		name    string
		cfg     ConnectionConfig
		wantErr bool
	}{
		{
			name:    "standard connection",
			cfg:     cfg,
			wantErr: false,
		},
		{
			name: "dsn override",
			cfg: ConnectionConfig{
				DSN: cfg.User + ":" + cfg.Password + "@tcp(" + cfg.Host + ":" + strconv.Itoa(cfg.Port) + ")/" + cfg.Database,
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
			cfg:     ConnectionConfig{User: cfg.User, Password: cfg.Password},
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
	cfg := testConfig()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Test ListTables
	tables, err := db.ListTables(ctx)
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
	schema, err := db.DescribeTable(ctx, "products")
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
	result, err := db.FetchTableData(ctx, "products", 100, 0)
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
	result, err = db.Execute(ctx, "SELECT * FROM customers WHERE city = 'New York'")
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
	count, err := db.CountRows(ctx, "customers")
	if err != nil {
		t.Fatalf("CountRows() error: %v", err)
	}
	if count != 10 {
		t.Errorf("CountRows() got %d, want 10", count)
	}

	// Test NULL handling
	result, err = db.Execute(ctx, "SELECT email FROM customers WHERE name = 'Ivy Chen'")
	if err != nil {
		t.Fatalf("Execute() NULL test error: %v", err)
	}
	if result.RowCount != 1 || result.Rows[0][0] != "<NULL>" {
		t.Errorf("Execute() NULL handling: got %v, want <NULL>", result.Rows[0][0])
	}

	// Test Execute INSERT
	result, err = db.Execute(ctx, "INSERT INTO customers (name, email, city) VALUES ('Test User', 'test@test.com', 'TestCity')")
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
	db.Execute(ctx, "DELETE FROM customers WHERE name = 'Test User'")

	t.Logf("All integration tests passed. Tables: %d, Products: %d rows", len(tables), 12)
}
