package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbplus/internal/database"
)

// TableSelectedMsg is emitted when a table is selected.
type TableSelectedMsg struct {
	TableName string
}

// KeyMap defines sidebar-specific keybindings.
type KeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Top    key.Binding
	Bottom key.Binding
	Select key.Binding
	Info   key.Binding
}

// DefaultKeyMap returns sidebar key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "first"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "last"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select table"),
		),
		Info: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "schema info"),
		),
	}
}

// Model represents the sidebar state.
type Model struct {
	tables     []database.TableInfo
	cursor     int
	dbName     string
	focused    bool
	width      int
	height     int
	keyMap     KeyMap
	schemaInfo *database.SchemaInfo
	showSchema bool
	styles     modelStyles
}

type modelStyles struct {
	header  lipgloss.Style
	item    lipgloss.Style
	active  lipgloss.Style
	dimmed  lipgloss.Style
	border  lipgloss.Style
	focused lipgloss.Style
}

// New creates a new sidebar model.
func New(dbName string) Model {
	highlight := lipgloss.Color("#7DC4E4")
	subtle := lipgloss.Color("#626262")
	border := lipgloss.Color("#444444")
	focusBorder := lipgloss.Color("#7DC4E4")
	activeBg := lipgloss.Color("#313244")

	return Model{
		dbName: dbName,
		keyMap: DefaultKeyMap(),
		styles: modelStyles{
			header: lipgloss.NewStyle().
				Foreground(highlight).
				Bold(true),
			item: lipgloss.NewStyle().
				PaddingLeft(1),
			active: lipgloss.NewStyle().
				Background(activeBg).
				Foreground(highlight).
				Bold(true).
				PaddingLeft(1),
			dimmed: lipgloss.NewStyle().
				Foreground(subtle),
			border: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(border),
			focused: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(focusBorder),
		},
	}
}

// SetTables updates the table list.
func (m *Model) SetTables(tables []database.TableInfo) {
	m.tables = tables
	if m.cursor >= len(tables) {
		m.cursor = max(0, len(tables)-1)
	}
}

// SetSchemaInfo updates the schema info display.
func (m *Model) SetSchemaInfo(info *database.SchemaInfo) {
	m.schemaInfo = info
}

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the sidebar dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SelectedTable returns the currently highlighted table name.
func (m Model) SelectedTable() string {
	if len(m.tables) == 0 {
		return ""
	}
	return m.tables[m.cursor].Name
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keyMap.Down):
			if m.cursor < len(m.tables)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keyMap.Top):
			m.cursor = 0
		case key.Matches(msg, m.keyMap.Bottom):
			m.cursor = len(m.tables) - 1
		case key.Matches(msg, m.keyMap.Select):
			if len(m.tables) > 0 {
				return m, func() tea.Msg {
					return TableSelectedMsg{TableName: m.tables[m.cursor].Name}
				}
			}
		case key.Matches(msg, m.keyMap.Info):
			m.showSchema = !m.showSchema
		}
	}

	return m, nil
}

// View renders the sidebar.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	contentWidth := m.width - 2 // account for border
	if contentWidth < 1 {
		contentWidth = 1
	}

	var b strings.Builder

	// Database name header
	header := m.styles.header.Width(contentWidth).Render(fmt.Sprintf(" %s", m.dbName))
	b.WriteString(header)
	b.WriteString("\n")

	// Tables section header
	b.WriteString(m.styles.dimmed.Width(contentWidth).Render(" Tables"))
	b.WriteString("\n")

	if len(m.tables) == 0 {
		b.WriteString(m.styles.dimmed.Render(" No tables found"))
	} else {
		// Calculate visible area for table list
		schemaHeight := 0
		if m.showSchema && m.schemaInfo != nil {
			schemaHeight = len(m.schemaInfo.Columns) + 5
		}
		visibleHeight := m.height - 4 - schemaHeight // header + section header + borders + padding
		if visibleHeight < 1 {
			visibleHeight = 1
		}

		// Scrolling window
		start := 0
		if m.cursor >= visibleHeight {
			start = m.cursor - visibleHeight + 1
		}
		end := start + visibleHeight
		if end > len(m.tables) {
			end = len(m.tables)
		}

		for i := start; i < end; i++ {
			name := truncate(m.tables[i].Name, contentWidth-2)
			if i == m.cursor {
				b.WriteString(m.styles.active.Width(contentWidth).Render(fmt.Sprintf("> %s", name)))
			} else {
				b.WriteString(m.styles.item.Width(contentWidth).Render(fmt.Sprintf("  %s", name)))
			}
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	// Schema info section
	if m.showSchema && m.schemaInfo != nil {
		b.WriteString("\n\n")
		b.WriteString(m.styles.dimmed.Width(contentWidth).Render(" Schema Info"))
		b.WriteString("\n")
		b.WriteString(m.styles.item.Render(fmt.Sprintf(" engine: %s", m.schemaInfo.Engine)))
		b.WriteString("\n")
		b.WriteString(m.styles.item.Render(fmt.Sprintf(" rows: %d", m.schemaInfo.RowCount)))
		b.WriteString("\n")
		b.WriteString(m.styles.item.Render(fmt.Sprintf(" charset: %s", m.schemaInfo.Charset)))
		for _, col := range m.schemaInfo.Columns {
			keyMark := ""
			if col.Key == "PRI" {
				keyMark = " PK"
			}
			b.WriteString("\n")
			b.WriteString(m.styles.dimmed.Render(fmt.Sprintf("  %s %s%s", col.Name, col.Type, keyMark)))
		}
	}

	content := b.String()

	// Apply border style
	style := m.styles.border
	if m.focused {
		style = m.styles.focused
	}

	return style.
		Width(contentWidth).
		Height(m.height - 2). // account for border
		Render(content)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
