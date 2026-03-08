package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alaa/dbtui/internal/config"
	"github.com/alaa/dbtui/internal/database"
)

const version = "0.1.0"

// ParseFlags parses CLI arguments and returns a ConnectionConfig.
func ParseFlags() (database.ConnectionConfig, error) {
	var (
		user    string
		password string
		host    string
		port    int
		dsn     string
		profile string
		tlsMode string
		tlsCert string
		tlsKey  string
		showVer bool
	)

	flag.StringVar(&user, "u", "", "MySQL user")
	flag.StringVar(&password, "p", "", "MySQL password")
	flag.StringVar(&host, "h", "127.0.0.1", "MySQL host")
	flag.IntVar(&port, "P", 3306, "MySQL port")
	flag.StringVar(&dsn, "dsn", "", "Full DSN connection string (overrides other flags)")
	flag.StringVar(&profile, "c", "", "Connection profile name from config file")
	flag.StringVar(&tlsMode, "tls", "", "TLS mode: \"true\", \"skip-verify\", or path to CA cert file")
	flag.StringVar(&tlsCert, "tls-cert", "", "Path to client certificate for mutual TLS")
	flag.StringVar(&tlsKey, "tls-key", "", "Path to client key for mutual TLS")
	flag.BoolVar(&showVer, "version", false, "Show version")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dbtui v%s - Modern MySQL/MariaDB CLI\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  dbtui [flags] [database]\n")
		fmt.Fprintf(os.Stderr, "  dbtui --dsn \"user:pass@tcp(host:port)/dbname\"\n")
		fmt.Fprintf(os.Stderr, "  dbtui -c <profile>            # use a saved connection profile\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVer {
		fmt.Printf("dbtui v%s\n", version)
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

	// If a profile is specified, load it from the config file.
	if profile != "" {
		cfg := config.Load()
		conn := cfg.FindConnection(profile)
		if conn == nil {
			available := cfg.ConnectionNames()
			if len(available) == 0 {
				return database.ConnectionConfig{}, fmt.Errorf("profile %q not found (no profiles configured)", profile)
			}
			return database.ConnectionConfig{}, fmt.Errorf("profile %q not found, available profiles: %s", profile, strings.Join(available, ", "))
		}

		// Start from profile values, then apply explicit flag overrides.
		connCfg := database.ConnectionConfig{
			Host:     conn.Host,
			Port:     conn.Port,
			User:     conn.User,
			Password: conn.Password,
			Database: conn.Database,
			TLS:      conn.TLS,
			TLSCert:  conn.TLSCert,
			TLSKey:   conn.TLSKey,
		}

		// Override with any flags that were explicitly set.
		flagSet := make(map[string]bool)
		flag.Visit(func(f *flag.Flag) { flagSet[f.Name] = true })

		if flagSet["u"] {
			connCfg.User = user
		}
		if flagSet["p"] {
			connCfg.Password = password
		}
		if flagSet["h"] {
			connCfg.Host = host
		}
		if flagSet["P"] {
			connCfg.Port = port
		}
		if flagSet["tls"] {
			connCfg.TLS = tlsMode
		}
		if flagSet["tls-cert"] {
			connCfg.TLSCert = tlsCert
		}
		if flagSet["tls-key"] {
			connCfg.TLSKey = tlsKey
		}

		// Positional argument overrides profile database.
		if flag.NArg() > 0 {
			connCfg.Database = flag.Arg(0)
		}

		return connCfg, nil
	}

	if user == "" {
		return database.ConnectionConfig{}, fmt.Errorf("user is required (-u flag or -c profile)")
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
		TLS:      tlsMode,
		TLSCert:  tlsCert,
		TLSKey:   tlsKey,
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
	atIdx := strings.Index(dsn, "@")
	if atIdx == -1 {
		return ""
	}
	userPass := dsn[:atIdx]
	if colonIdx := strings.Index(userPass, ":"); colonIdx != -1 {
		return userPass[:colonIdx]
	}
	return userPass
}
