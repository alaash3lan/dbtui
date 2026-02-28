package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alaa/dbplus/internal/database"
)

const version = "0.1.0"

// ParseFlags parses CLI arguments and returns a ConnectionConfig.
func ParseFlags() (database.ConnectionConfig, error) {
	var (
		user     string
		password string
		host     string
		port     int
		dsn      string
		showVer  bool
	)

	flag.StringVar(&user, "u", "", "MySQL user")
	flag.StringVar(&password, "p", "", "MySQL password")
	flag.StringVar(&host, "h", "127.0.0.1", "MySQL host")
	flag.IntVar(&port, "P", 3306, "MySQL port")
	flag.StringVar(&dsn, "dsn", "", "Full DSN connection string (overrides other flags)")
	flag.BoolVar(&showVer, "version", false, "Show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dbplus v%s - Modern MySQL/MariaDB CLI\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  dbplus [flags] [database]\n")
		fmt.Fprintf(os.Stderr, "  dbplus --dsn \"user:pass@tcp(host:port)/dbname\"\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVer {
		fmt.Printf("dbplus v%s\n", version)
		os.Exit(0)
	}

	if dsn != "" {
		dbName := extractDBFromDSN(dsn)
		return database.ConnectionConfig{
			DSN:      dsn,
			Database: dbName,
			User:     extractUserFromDSN(dsn),
			Host:     host,
		}, nil
	}

	if user == "" {
		return database.ConnectionConfig{}, fmt.Errorf("user is required (-u flag)")
	}

	// Positional argument is the database name
	dbName := ""
	if flag.NArg() > 0 {
		dbName = flag.Arg(0)
	}

	return database.ConnectionConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: dbName,
	}, nil
}

// Version returns the app version string.
func Version() string {
	return version
}

func extractDBFromDSN(dsn string) string {
	// DSN format: user:pass@tcp(host:port)/dbname?params
	idx := strings.LastIndex(dsn, "/")
	if idx == -1 {
		return ""
	}
	rest := dsn[idx+1:]
	if qIdx := strings.Index(rest, "?"); qIdx != -1 {
		return rest[:qIdx]
	}
	return rest
}

func extractUserFromDSN(dsn string) string {
	idx := strings.Index(dsn, ":")
	if idx == -1 {
		idx = strings.Index(dsn, "@")
		if idx == -1 {
			return ""
		}
	}
	return dsn[:idx]
}
