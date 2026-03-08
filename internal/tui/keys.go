package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all global keybindings.
type KeyMap struct {
	Quit          key.Binding
	FocusNext     key.Binding
	FocusPrev     key.Binding
	Help          key.Binding
	GrowSidebar   key.Binding
	ShrinkSidebar key.Binding
	ToggleTheme   key.Binding
	Refresh       key.Binding
	ExportCSV     key.Binding
	ExportJSON    key.Binding
	ExplainQuery  key.Binding
	SwitchDB      key.Binding
}

// DefaultKeyMap returns the default set of keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next pane"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Help: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("F1", "help"),
		),
		GrowSidebar: key.NewBinding(
			key.WithKeys("ctrl+right"),
			key.WithHelp("ctrl+→", "grow sidebar"),
		),
		ShrinkSidebar: key.NewBinding(
			key.WithKeys("ctrl+left"),
			key.WithHelp("ctrl+←", "shrink sidebar"),
		),
		ToggleTheme: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "toggle theme"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		ExportCSV: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "export CSV"),
		),
		ExportJSON: key.NewBinding(
			key.WithKeys("ctrl+j"),
			key.WithHelp("ctrl+j", "export JSON"),
		),
		ExplainQuery: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "explain query"),
		),
		SwitchDB: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "switch database"),
		),
	}
}
