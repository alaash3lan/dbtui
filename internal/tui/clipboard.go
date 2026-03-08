package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/atotto/clipboard"
)

// copyToClipboardCmd returns a tea.Cmd that copies text to the system clipboard.
func (m Model) copyToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		err := clipboard.WriteAll(text)
		return clipboardResultMsg{Err: err}
	}
}
