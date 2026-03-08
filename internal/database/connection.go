package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// DB wraps a sql.DB connection with dbplus-specific operations.
type DB struct {
	conn   *sql.DB
	config ConnectionConfig
}

// New creates a new DB instance from the given config.
func New(cfg ConnectionConfig) (*DB, error) {
	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid connection config: %w", err)
	}

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	conn.SetConnMaxLifetime(5 * time.Minute)
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	return &DB{conn: conn, config: cfg}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Ping verifies the connection is still alive.
func (db *DB) Ping() error {
	return db.conn.Ping()
}

// Conn returns the underlying sql.DB for direct use.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// DatabaseName returns the connected database name.
func (db *DB) DatabaseName() string {
	return db.config.Database
}

// User returns the connected user.
func (db *DB) User() string {
	return db.config.User
}

// Host returns the connected host.
func (db *DB) Host() string {
	return db.config.Host
}

// SwitchDatabase changes the active database.
func (db *DB) SwitchDatabase(ctx context.Context, name string) error {
	if err := validIdentifier(name); err != nil {
		return err
	}
	_, err := db.conn.ExecContext(ctx, fmt.Sprintf("USE `%s`", name))
	if err != nil {
		return err
	}
	db.config.Database = name
	return nil
}

// ListDatabases returns all accessible databases.
func (db *DB) ListDatabases(ctx context.Context) ([]string, error) {
	rows, err := db.conn.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		databases = append(databases, name)
	}
	return databases, rows.Err()
}

func buildDSN(cfg ConnectionConfig) (string, error) {
	if cfg.DSN != "" {
		return cfg.DSN, nil
	}

	if cfg.User == "" {
		return "", fmt.Errorf("user is required")
	}

	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}

	port := cfg.Port
	if port == 0 {
		port = 3306
	}

	mysqlCfg := mysql.Config{
		User:                 cfg.User,
		Passwd:               cfg.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", host, port),
		DBName:               cfg.Database,
		AllowNativePasswords: true,
		ParseTime:            true,
		InterpolateParams:    true,
	}

	return mysqlCfg.FormatDSN(), nil
}
