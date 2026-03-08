package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConnectionProfile holds a named connection bookmark.
type ConnectionProfile struct {
	Name     string `toml:"name"`
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Database string `toml:"database"`
	TLS      string `toml:"tls"`      // "true", "skip-verify", or path to CA cert
	TLSCert  string `toml:"tls_cert"` // Path to client certificate (optional)
	TLSKey   string `toml:"tls_key"`  // Path to client key (optional)
}

// Config holds all dbtui configuration.
type Config struct {
	Display     DisplayConfig       `toml:"display"`
	History     HistoryConfig       `toml:"history"`
	Query       QueryConfig         `toml:"query"`
	Connections []ConnectionProfile `toml:"connections"`
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
		p := filepath.Join(xdg, "dbtui", "config.toml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Check ~/.config/dbtui/
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	p := filepath.Join(home, ".config", "dbtui", "config.toml")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	// Check ~/.dbtui.toml
	p = filepath.Join(home, ".dbtui.toml")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	return ""
}

// FindConnection returns the ConnectionProfile with the given name, or nil if not found.
func (c *Config) FindConnection(name string) *ConnectionProfile {
	for i := range c.Connections {
		if c.Connections[i].Name == name {
			return &c.Connections[i]
		}
	}
	return nil
}

// ConnectionNames returns the names of all configured connection profiles.
func (c *Config) ConnectionNames() []string {
	names := make([]string, len(c.Connections))
	for i, conn := range c.Connections {
		names[i] = conn.Name
	}
	return names
}

func defaultHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "dbtui", "history")
}

// SavedQuery holds a named SQL bookmark.
type SavedQuery struct {
	Name string `toml:"name"`
	SQL  string `toml:"sql"`
}

// BookmarksConfig holds all saved query bookmarks.
type BookmarksConfig struct {
	Bookmarks []SavedQuery `toml:"bookmarks"`
}

// LoadBookmarks reads bookmarks from ~/.config/dbtui/bookmarks.toml.
func LoadBookmarks() *BookmarksConfig {
	cfg := &BookmarksConfig{}

	path := bookmarksPath()
	if path == "" {
		return cfg
	}

	if _, err := os.Stat(path); err != nil {
		return cfg
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse bookmarks %s: %v\n", path, err)
	}
	return cfg
}

// SaveBookmarks writes bookmarks back to ~/.config/dbtui/bookmarks.toml.
func SaveBookmarks(cfg *BookmarksConfig) error {
	path := bookmarksPath()
	if path == "" {
		return fmt.Errorf("cannot determine bookmarks path")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create bookmarks file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}

func bookmarksPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dbtui", "bookmarks.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "dbtui", "bookmarks.toml")
}

// FavoritesConfig holds per-database table favorites.
type FavoritesConfig struct {
	Favorites map[string][]string `toml:"favorites"`
}

// LoadFavorites reads favorites from ~/.config/dbtui/favorites.toml.
func LoadFavorites() *FavoritesConfig {
	cfg := &FavoritesConfig{
		Favorites: make(map[string][]string),
	}

	path := favoritesPath()
	if path == "" {
		return cfg
	}

	if _, err := os.Stat(path); err != nil {
		return cfg
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to parse favorites %s: %v\n", path, err)
	}
	if cfg.Favorites == nil {
		cfg.Favorites = make(map[string][]string)
	}
	return cfg
}

// SaveFavorites writes favorites to ~/.config/dbtui/favorites.toml.
func SaveFavorites(cfg *FavoritesConfig) error {
	path := favoritesPath()
	if path == "" {
		return fmt.Errorf("unable to determine favorites path")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create favorites file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	return enc.Encode(cfg)
}

func favoritesPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dbtui", "favorites.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "dbtui", "favorites.toml")
}
