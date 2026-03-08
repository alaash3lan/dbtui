package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbplus/internal/database"
	"github.com/alaa/dbplus/internal/stringutil"
)

// TableSelectedMsg is emitted when a table is selected.
type TableSelectedMsg struct {
	TableName string
}

// SchemaRequestMsg is emitted when the user presses 'i' to view schema info.
type SchemaRequestMsg struct {
	TableName string
}

// Colors holds the theme colors used by the sidebar.
type Colors struct {
	Highlight   lipgloss.Color
	Subtle      lipgloss.Color
	Border      lipgloss.Color
	FocusBorder lipgloss.Color
	ActiveBg    lipgloss.Color
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
	colors     Colors
}

// New creates a new sidebar model.
func New(dbName string) Model {
	return Model{
		dbName: dbName,
		keyMap: DefaultKeyMap(),
	}
}

// SetTables updates the table list.
func (m *Model) SetTables(tables []database.TableInfo) {
	m.tables = tables
	if m.cursor >= len(tables) {
		m.cursor = maxInt(0, len(tables)-1)
	}
}

// SetSchemaInfo updates the schema info display.
func (m *Model) SetSchemaInfo(info *database.SchemaInfo) {
	m.schemaInfo = info
}

// SetDBName updates the database name header.
func (m *Model) SetDBName(name string) {
	m.dbName = name
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

// SetColors updates the theme colors.
func (m *Model) SetColors(c Colors) {
	m.colors = c
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
			if m.showSchema && len(m.tables) > 0 {
				tableName := m.tables[m.cursor].Name
				return m, func() tea.Msg {
					return SchemaRequestMsg{TableName: tableName}
				}
			}
		}
	}

	return m, nil
}

// View renders the sidebar.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	c := m.colors
	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	headerStyle := lipgloss.NewStyle().Foreground(c.Highlight).Bold(true)
	dimmedStyle := lipgloss.NewStyle().Foreground(c.Subtle)
	itemStyle := lipgloss.NewStyle().PaddingLeft(1)
	activeStyle := lipgloss.NewStyle().Background(c.ActiveBg).Foreground(c.Highlight).Bold(true).PaddingLeft(1)

	var b strings.Builder

	b.WriteString(headerStyle.Width(contentWidth).Render(fmt.Sprintf(" %s", m.dbName)))
	b.WriteString("\n")
	b.WriteString(dimmedStyle.Width(contentWidth).Render(" Tables"))
	b.WriteString("\n")

	if len(m.tables) == 0 {
		b.WriteString(dimmedStyle.Render(" No tables found"))
	} else {
		schemaHeight := 0
		if m.showSchema && m.schemaInfo != nil {
			schemaHeight = len(m.schemaInfo.Columns) + 5
		}
		visibleHeight := m.height - 4 - schemaHeight
		if visibleHeight < 1 {
			visibleHeight = 1
		}

		start := 0
		if m.cursor >= visibleHeight {
			start = m.cursor - visibleHeight + 1
		}
		end := start + visibleHeight
		if end > len(m.tables) {
			end = len(m.tables)
		}

		for i := start; i < end; i++ {
			name := stringutil.TruncateSimple(m.tables[i].Name, contentWidth-2)
			if i == m.cursor {
				b.WriteString(activeStyle.Width(contentWidth).Render(fmt.Sprintf("> %s", name)))
			} else {
				b.WriteString(itemStyle.Width(contentWidth).Render(fmt.Sprintf("  %s", name)))
			}
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	if m.showSchema && m.schemaInfo != nil {
		b.WriteString("\n\n")
		b.WriteString(dimmedStyle.Width(contentWidth).Render(" Schema Info"))
		b.WriteString("\n")
		b.WriteString(itemStyle.Render(fmt.Sprintf(" engine: %s", m.schemaInfo.Engine)))
		b.WriteString("\n")
		b.WriteString(itemStyle.Render(fmt.Sprintf(" rows: %d", m.schemaInfo.RowCount)))
		b.WriteString("\n")
		b.WriteString(itemStyle.Render(fmt.Sprintf(" charset: %s", m.schemaInfo.Charset)))
		for _, col := range m.schemaInfo.Columns {
			keyMark := ""
			if col.Key == "PRI" {
				keyMark = " PK"
			}
			b.WriteString("\n")
			b.WriteString(dimmedStyle.Render(fmt.Sprintf("  %s %s%s", col.Name, col.Type, keyMark)))
		}
	}

	content := b.String()

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.Border)
	if m.focused {
		borderStyle = borderStyle.BorderForeground(c.FocusBorder)
	}

	return borderStyle.
		Width(contentWidth).
		Height(m.height - 2).
		Render(content)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
