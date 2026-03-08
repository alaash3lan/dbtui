package editor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ExecuteQueryMsg is emitted when the user submits SQL.
type ExecuteQueryMsg struct {
	SQL string
}

// KeyMap defines editor-specific keybindings.
type KeyMap struct {
	Execute      key.Binding
	ForceExecute key.Binding
	HistoryPrev  key.Binding
	HistoryNext  key.Binding
	Clear        key.Binding
}

// DefaultKeyMap returns editor key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Execute: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "execute"),
		),
		ForceExecute: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "force execute"),
		),
		HistoryPrev: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "prev history"),
		),
		HistoryNext: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next history"),
		),
		Clear: key.NewBinding(
			key.WithKeys("escape"),
			key.WithHelp("esc", "clear"),
		),
	}
}

// Colors holds the theme colors used by the editor.
type Colors struct {
	Highlight    lipgloss.Color
	Subtle       lipgloss.Color
	Border       lipgloss.Color
	FocusBorder  lipgloss.Color
	ErrorColor   lipgloss.Color
	SuccessColor lipgloss.Color
	WarningColor lipgloss.Color
}

// Model represents the query editor state.
type Model struct {
	textarea    textarea.Model
	history     *HistoryRing
	running     bool
	lastError   string
	lastResult  string
	focused     bool
	width       int
	height      int
	keyMap      KeyMap
	savedInput  string // saved when navigating history
	colors      Colors
	saveToFile  bool
	historyFile string
}

// New creates a new editor model.
func New(maxEntries int) Model {
	ta := textarea.New()
	ta.Placeholder = "SELECT * FROM ..."
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.SetHeight(3)
	ta.CharLimit = 4096
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()

	if maxEntries <= 0 {
		maxEntries = 100
	}

	return Model{
		textarea: ta,
		history:  NewHistoryRing(maxEntries),
		keyMap:   DefaultKeyMap(),
	}
}

// SetHistoryConfig configures file-based history persistence.
func (m *Model) SetHistoryConfig(saveToFile bool, filePath string) {
	m.saveToFile = saveToFile
	m.historyFile = filePath
}

// LoadHistory loads history from file if persistence is enabled.
func (m *Model) LoadHistory() {
	if m.saveToFile && m.historyFile != "" {
		_ = m.history.LoadFromFile(m.historyFile)
	}
}

// SaveHistory writes history to file if persistence is enabled.
func (m *Model) SaveHistory() {
	if m.saveToFile && m.historyFile != "" {
		_ = m.history.SaveToFile(m.historyFile)
	}
}

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
	}
}

// SetSize sets the editor dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Textarea height = total height minus border(2) minus prompt line(1) minus status line(1)
	taHeight := height - 5
	if taHeight < 1 {
		taHeight = 1
	}
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(taHeight)
}

// SetRunning sets the running indicator.
func (m *Model) SetRunning(running bool) {
	m.running = running
}

// SetError sets the last error message.
func (m *Model) SetError(err string) {
	m.lastError = err
	m.lastResult = ""
}

// SetResult sets the last result message (for non-SELECT queries).
func (m *Model) SetResult(msg string) {
	m.lastResult = msg
	m.lastError = ""
}

// SetColors updates the theme colors.
func (m *Model) SetColors(c Colors) {
	m.colors = c
}

// ClearStatus clears error and result messages.
func (m *Model) ClearStatus() {
	m.lastError = ""
	m.lastResult = ""
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Clear):
			m.textarea.SetValue("")
			m.history.Reset()
			m.lastError = ""
			m.lastResult = ""
			return m, nil

		case key.Matches(msg, m.keyMap.ForceExecute):
			return m.execute()

		case key.Matches(msg, m.keyMap.Execute):
			val := strings.TrimSpace(m.textarea.Value())
			if val != "" && strings.HasSuffix(val, ";") {
				return m.execute()
			}
			// No semicolon — insert newline
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd

		case key.Matches(msg, m.keyMap.HistoryPrev):
			// Only navigate history when cursor is on the first line
			if m.textarea.Line() == 0 {
				if entry, ok := m.history.Previous(); ok {
					if m.savedInput == "" {
						m.savedInput = m.textarea.Value()
					}
					m.textarea.SetValue(entry)
					m.textarea.CursorEnd()
					return m, nil
				}
				return m, nil
			}

		case key.Matches(msg, m.keyMap.HistoryNext):
			// Only navigate when cursor is on the last line
			lines := strings.Count(m.textarea.Value(), "\n")
			if m.textarea.Line() >= lines {
				if entry, ok := m.history.Next(); ok {
					if entry == "" {
						m.textarea.SetValue(m.savedInput)
						m.savedInput = ""
					} else {
						m.textarea.SetValue(entry)
					}
					m.textarea.CursorEnd()
					return m, nil
				}
				return m, nil
			}
		}
	}

	// Pass to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) execute() (Model, tea.Cmd) {
	val := strings.TrimSpace(m.textarea.Value())
	if val == "" {
		return m, nil
	}

	// Strip trailing semicolon for execution
	sql := strings.TrimSuffix(val, ";")
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return m, nil
	}

	m.history.Push(val)
	m.running = true
	m.lastError = ""
	m.lastResult = ""
	m.savedInput = ""

	return m, func() tea.Msg {
		return ExecuteQueryMsg{SQL: sql}
	}
}

// View renders the editor.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	c := m.colors
	highlight := c.Highlight
	subtle := c.Subtle
	border := c.Border
	focusBorder := c.FocusBorder
	errColor := c.ErrorColor
	successColor := c.SuccessColor

	var b strings.Builder

	// Prompt line
	promptStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true)
	b.WriteString(promptStyle.Render("mysql>> "))

	if m.running {
		runStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF")).Bold(true)
		b.WriteString(runStyle.Render("[Running...]"))
	}
	b.WriteString("\n")

	// Textarea
	b.WriteString(m.textarea.View())

	// Status line (error or result)
	if m.lastError != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(errColor).Render(m.lastError))
	} else if m.lastResult != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(successColor).Render(m.lastResult))
	}

	// History indicator
	if m.history.Len() > 0 {
		histInfo := fmt.Sprintf("history: %d", m.history.Len())
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(subtle).Render(histInfo))
	}

	// Apply border
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(contentWidth).
		Height(m.height - 2)

	if m.focused {
		style = style.BorderForeground(focusBorder)
	}

	return style.Render(b.String())
}
