package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all dbplus configuration.
type Config struct {
	Display DisplayConfig `toml:"display"`
	History HistoryConfig `toml:"history"`
	Query   QueryConfig   `toml:"query"`
}

// QueryConfig holds query execution settings.
type QueryConfig struct {
	TimeoutSeconds int `toml:"timeout_seconds"`
}

// DisplayConfig holds UI-related settings.
type DisplayConfig struct {
	Theme        string `toml:"theme"`
	PageSize     int    `toml:"page_size"`
	SidebarWidth int    `toml:"sidebar_width"`
	EditorHeight int    `toml:"editor_height"`
}

// HistoryConfig holds query history settings.
type HistoryConfig struct {
	MaxEntries int    `toml:"max_entries"`
	SaveToFile bool   `toml:"save_to_file"`
	File       string `toml:"file"`
}

// Load reads the config file and returns a Config with defaults applied.
func Load() *Config {
	cfg := &Config{
		Display: DisplayConfig{
			Theme:        DefaultTheme,
			PageSize:     DefaultPageSize,
			SidebarWidth: DefaultSidebarWidth,
			EditorHeight: DefaultEditorHeight,
		},
		History: HistoryConfig{
			MaxEntries: DefaultHistoryMax,
			SaveToFile: true,
			File:       defaultHistoryPath(),
		},
		Query: QueryConfig{
			TimeoutSeconds: DefaultQueryTimeout,
		},
	}

	configPath := findConfigFile()
	if configPath == "" {
		return cfg
	}

	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse config %s: %v\n", configPath, err)
	}
	return cfg
}

func findConfigFile() string {
	// Check XDG config dir first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		p := filepath.Join(xdg, "dbplus", "config.toml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Check ~/.config/dbplus/
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	p := filepath.Join(home, ".config", "dbplus", "config.toml")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	// Check ~/.dbplus.toml
	p = filepath.Join(home, ".dbplus.toml")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	return ""
}

func defaultHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "dbplus", "history")
}
