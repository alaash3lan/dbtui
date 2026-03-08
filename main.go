package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alaa/dbplus/cmd"
	"github.com/alaa/dbplus/internal/config"
	"github.com/alaa/dbplus/internal/database"
	"github.com/alaa/dbplus/internal/tui"
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

	appCfg := config.Load()
	queryTimeout := time.Duration(appCfg.Query.TimeoutSeconds) * time.Second
	model := tui.New(db, cmd.Version(), queryTimeout)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
