package main

import (
	"fmt"
	"os"

	"github.com/alaa/dbplus/cmd"
	"github.com/alaa/dbplus/internal/database"
)

func main() {
	cfg, err := cmd.ParseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'dbplus --help' for usage.\n")
		os.Exit(1)
	}

	db, err := database.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connection error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("Connected to %s@%s", cfg.User, cfg.Host)
	if cfg.Database != "" {
		fmt.Printf("/%s", cfg.Database)
	}
	fmt.Println()

	// TUI will be launched here in Sprint 2
	_ = db
}
