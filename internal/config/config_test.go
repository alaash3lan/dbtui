package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure no config file interferes by using a nonexistent XDG path.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := Load()

	if cfg.Display.Theme != DefaultTheme {
		t.Errorf("Theme = %q, want %q", cfg.Display.Theme, DefaultTheme)
	}
	if cfg.Display.PageSize != DefaultPageSize {
		t.Errorf("PageSize = %d, want %d", cfg.Display.PageSize, DefaultPageSize)
	}
	if cfg.Display.SidebarWidth != DefaultSidebarWidth {
		t.Errorf("SidebarWidth = %d, want %d", cfg.Display.SidebarWidth, DefaultSidebarWidth)
	}
	if cfg.Display.EditorHeight != DefaultEditorHeight {
		t.Errorf("EditorHeight = %d, want %d", cfg.Display.EditorHeight, DefaultEditorHeight)
	}
	if cfg.History.MaxEntries != DefaultHistoryMax {
		t.Errorf("MaxEntries = %d, want %d", cfg.History.MaxEntries, DefaultHistoryMax)
	}
	if !cfg.History.SaveToFile {
		t.Error("SaveToFile should default to true")
	}
	if cfg.Query.TimeoutSeconds != DefaultQueryTimeout {
		t.Errorf("TimeoutSeconds = %d, want %d", cfg.Query.TimeoutSeconds, DefaultQueryTimeout)
	}
}

func TestDefaultHistoryPath(t *testing.T) {
	path := defaultHistoryPath()

	if !strings.Contains(path, "dbtui") {
		t.Errorf("history path %q should contain 'dbtui'", path)
	}
	if strings.Contains(path, "dbplus") {
		t.Errorf("history path %q should not contain 'dbplus'", path)
	}
}
