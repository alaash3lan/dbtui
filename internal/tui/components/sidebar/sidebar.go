package sidebar

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbtui/internal/database"
	"github.com/alaa/dbtui/internal/stringutil"
)

// TableSelectedMsg is emitted when a table is selected.
type TableSelectedMsg struct {
	TableName string
}

// SchemaRequestMsg is emitted when the user presses 'i' to view schema info.
type SchemaRequestMsg struct {
	TableName string
}

// FavoriteToggledMsg is emitted when a table's favorite status is toggled.
type FavoriteToggledMsg struct {
	TableName  string
	IsFavorite bool
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
	Up          key.Binding
	Down        key.Binding
	Top         key.Binding
	Bottom      key.Binding
	Select      key.Binding
	Info        key.Binding
	Filter      key.Binding
	ClearFilter key.Binding
	Favorite    key.Binding
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
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("escape"),
			key.WithHelp("esc", "clear filter"),
		),
		Favorite: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "toggle favorite"),
		),
	}
}

// Model represents the sidebar state.
type Model struct {
	allTables      []database.TableInfo // unfiltered tables
	filteredTables []database.TableInfo // currently displayed (filtered or all)
	cursor         int
	dbName         string
	focused        bool
	width          int
	height         int
	keyMap         KeyMap
	schemaInfo     *database.SchemaInfo
	showSchema     bool
	colors         Colors

	// Filter
	filterInput  textinput.Model
	filterActive bool // true when typing in filter
	filterText   string

	// Favorites
	favorites    map[string]bool // set of favorited table names for current db
	favSeparator int            // index in filteredTables where non-favorites start (-1 if no separator)
}

// New creates a new sidebar model.
func New(dbName string) Model {
	fi := textinput.New()
	fi.Placeholder = "table name"
	fi.Prompt = "/ "
	fi.CharLimit = 256

	return Model{
		dbName:       dbName,
		keyMap:       DefaultKeyMap(),
		filterInput:  fi,
		favorites:    make(map[string]bool),
		favSeparator: -1,
	}
}

// SetTables updates the table list.
func (m *Model) SetTables(tables []database.TableInfo) {
	m.allTables = tables
	m.applyFilter()
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
	if !focused {
		m.filterActive = false
		m.filterInput.Blur()
	}
}

// SetSize sets the sidebar dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.filterInput.Width = width - 6 // account for prompt + border
}

// SetColors updates the theme colors.
func (m *Model) SetColors(c Colors) {
	m.colors = c
}

// SetFavorites sets the favorite tables for the current database.
func (m *Model) SetFavorites(favs []string) {
	m.favorites = make(map[string]bool, len(favs))
	for _, name := range favs {
		m.favorites[name] = true
	}
	m.applyFilter()
}

// SelectedTable returns the currently highlighted table name.
func (m Model) SelectedTable() string {
	if len(m.filteredTables) == 0 {
		return ""
	}
	return m.filteredTables[m.cursor].Name
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
		// When filter input is active, route keys there
		if m.filterActive {
			switch {
			case key.Matches(msg, m.keyMap.ClearFilter):
				m.filterActive = false
				m.filterInput.Blur()
				if m.filterText != "" {
					// Clear the filter
					m.filterInput.SetValue("")
					m.filterText = ""
					m.applyFilter()
				}
				return m, nil
			case msg.Type == tea.KeyEnter:
				// Confirm filter, exit filter mode
				m.filterText = m.filterInput.Value()
				m.filterActive = false
				m.filterInput.Blur()
				m.applyFilter()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				// Live filtering as user types
				m.filterText = m.filterInput.Value()
				m.applyFilter()
				return m, cmd
			}
		}

		// Navigation mode
		switch {
		case key.Matches(msg, m.keyMap.Filter):
			m.filterActive = true
			return m, m.filterInput.Focus()
		case key.Matches(msg, m.keyMap.ClearFilter):
			// Escape in navigation mode clears filter if active
			if m.filterText != "" {
				m.filterInput.SetValue("")
				m.filterText = ""
				m.applyFilter()
				return m, nil
			}
		case key.Matches(msg, m.keyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keyMap.Down):
			if m.cursor < len(m.filteredTables)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keyMap.Top):
			m.cursor = 0
		case key.Matches(msg, m.keyMap.Bottom):
			if len(m.filteredTables) > 0 {
				m.cursor = len(m.filteredTables) - 1
			}
		case key.Matches(msg, m.keyMap.Select):
			if len(m.filteredTables) > 0 {
				return m, func() tea.Msg {
					return TableSelectedMsg{TableName: m.filteredTables[m.cursor].Name}
				}
			}
		case key.Matches(msg, m.keyMap.Info):
			m.showSchema = !m.showSchema
			if m.showSchema && len(m.filteredTables) > 0 {
				tableName := m.filteredTables[m.cursor].Name
				return m, func() tea.Msg {
					return SchemaRequestMsg{TableName: tableName}
				}
			}
		case key.Matches(msg, m.keyMap.Favorite):
			if len(m.filteredTables) > 0 {
				tableName := m.filteredTables[m.cursor].Name
				isFav := !m.favorites[tableName]
				if isFav {
					m.favorites[tableName] = true
				} else {
					delete(m.favorites, tableName)
				}
				m.applyFilter()
				return m, func() tea.Msg {
					return FavoriteToggledMsg{TableName: tableName, IsFavorite: isFav}
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

	// Filter bar
	if m.filterActive {
		b.WriteString(m.filterInput.View())
	} else if m.filterText != "" {
		filterStyle := lipgloss.NewStyle().Foreground(c.Highlight)
		matchInfo := fmt.Sprintf(" %d/%d", len(m.filteredTables), len(m.allTables))
		b.WriteString(filterStyle.Render(fmt.Sprintf(" [%s]", m.filterText)))
		b.WriteString(dimmedStyle.Render(matchInfo))
	} else {
		b.WriteString(dimmedStyle.Render(" / to filter"))
	}
	b.WriteString("\n")

	tables := m.filteredTables

	if len(tables) == 0 {
		if m.filterText != "" {
			b.WriteString(dimmedStyle.Render(" No matching tables"))
		} else {
			b.WriteString(dimmedStyle.Render(" No tables found"))
		}
	} else {
		schemaHeight := 0
		if m.showSchema && m.schemaInfo != nil {
			schemaHeight = len(m.schemaInfo.Columns) + 8 // columns + header + metadata lines
			if len(m.schemaInfo.Indexes) > 0 {
				schemaHeight += len(m.schemaInfo.Indexes) + 1
			}
			if len(m.schemaInfo.ForeignKeys) > 0 {
				schemaHeight += len(m.schemaInfo.ForeignKeys) + 1
			}
		}
		visibleHeight := m.height - 5 - schemaHeight
		if visibleHeight < 1 {
			visibleHeight = 1
		}

		start := 0
		if m.cursor >= visibleHeight {
			start = m.cursor - visibleHeight + 1
		}
		end := start + visibleHeight
		if end > len(tables) {
			end = len(tables)
		}

		favStyle := lipgloss.NewStyle().Foreground(c.Highlight)

		for i := start; i < end; i++ {
			// Draw separator between favorites and rest
			if m.favSeparator > 0 && i == m.favSeparator {
				sep := strings.Repeat("─", contentWidth-2)
				b.WriteString(dimmedStyle.Render(" " + sep))
				b.WriteString("\n")
			}

			rowCount := formatRowCount(tables[i].Rows)
			rowCountSuffix := fmt.Sprintf(" (%s)", rowCount)
			isFav := m.favorites[tables[i].Name]
			// Reserve space for prefix ("> " or "* " or "  ") and row count suffix
			maxNameWidth := contentWidth - 2 - len(rowCountSuffix)
			if maxNameWidth < 1 {
				maxNameWidth = 1
			}
			name := stringutil.TruncateSimple(tables[i].Name, maxNameWidth)
			rowCountRendered := dimmedStyle.Render(rowCountSuffix)
			if i == m.cursor {
				b.WriteString(activeStyle.Width(contentWidth).Render(fmt.Sprintf("> %s", name) + rowCountRendered))
			} else if isFav {
				b.WriteString(itemStyle.Width(contentWidth).Render(favStyle.Render("* ")+name+rowCountRendered))
			} else {
				b.WriteString(itemStyle.Width(contentWidth).Render(fmt.Sprintf("  %s", name) + rowCountRendered))
			}
			if i < end-1 {
				b.WriteString("\n")
			}
		}
	}

	if m.showSchema && m.schemaInfo != nil {
		s := m.schemaInfo
		b.WriteString("\n\n")
		b.WriteString(headerStyle.Width(contentWidth).Render(" Schema Info"))
		b.WriteString("\n")

		// Table metadata
		b.WriteString(itemStyle.Render(fmt.Sprintf(" engine: %s", s.Engine)))
		b.WriteString("\n")
		b.WriteString(itemStyle.Render(fmt.Sprintf(" rows: %d", s.RowCount)))
		if s.DataSize != "" {
			b.WriteString("  ")
			b.WriteString(dimmedStyle.Render(s.DataSize))
		}
		b.WriteString("\n")
		b.WriteString(itemStyle.Render(fmt.Sprintf(" charset: %s", s.Charset)))
		if s.AutoIncr > 0 {
			b.WriteString("\n")
			b.WriteString(itemStyle.Render(fmt.Sprintf(" auto_incr: %d", s.AutoIncr)))
		}
		if s.CreateTime != "" {
			b.WriteString("\n")
			b.WriteString(dimmedStyle.Render(fmt.Sprintf("  created: %s", s.CreateTime)))
		}
		if s.UpdateTime != "" {
			b.WriteString("\n")
			b.WriteString(dimmedStyle.Render(fmt.Sprintf("  updated: %s", s.UpdateTime)))
		}
		if s.Comment != "" {
			b.WriteString("\n")
			b.WriteString(dimmedStyle.Render(fmt.Sprintf("  comment: %s", s.Comment)))
		}

		// Columns
		b.WriteString("\n")
		b.WriteString(dimmedStyle.Width(contentWidth).Render(" Columns"))
		for _, col := range s.Columns {
			marks := ""
			if col.Key == "PRI" {
				marks += " PK"
			} else if col.Key == "UNI" {
				marks += " UQ"
			} else if col.Key == "MUL" {
				marks += " IX"
			}
			if col.Extra != "" {
				marks += " " + col.Extra
			}
			nullable := ""
			if !col.Nullable {
				nullable = " NOT NULL"
			}
			def := ""
			if col.Default != nil {
				def = fmt.Sprintf(" =%s", *col.Default)
			}
			b.WriteString("\n")
			line := fmt.Sprintf("  %s %s%s%s%s", col.Name, col.Type, nullable, def, marks)
			b.WriteString(dimmedStyle.Render(stringutil.TruncateSimple(line, contentWidth-1)))
		}

		// Indexes
		if len(s.Indexes) > 0 {
			b.WriteString("\n")
			b.WriteString(dimmedStyle.Width(contentWidth).Render(" Indexes"))
			for _, idx := range s.Indexes {
				uniq := ""
				if idx.Unique && idx.Name != "PRIMARY" {
					uniq = "UNIQUE "
				}
				cols := strings.Join(idx.Columns, ", ")
				line := fmt.Sprintf("  %s%s (%s)", uniq, idx.Name, cols)
				b.WriteString("\n")
				b.WriteString(dimmedStyle.Render(stringutil.TruncateSimple(line, contentWidth-1)))
			}
		}

		// Foreign Keys
		if len(s.ForeignKeys) > 0 {
			b.WriteString("\n")
			b.WriteString(dimmedStyle.Width(contentWidth).Render(" Foreign Keys"))
			for _, fk := range s.ForeignKeys {
				cols := strings.Join(fk.Columns, ", ")
				refCols := strings.Join(fk.RefColumns, ", ")
				line := fmt.Sprintf("  %s (%s) -> %s(%s)", fk.Name, cols, fk.RefTable, refCols)
				b.WriteString("\n")
				b.WriteString(dimmedStyle.Render(stringutil.TruncateSimple(line, contentWidth-1)))
				if fk.OnDelete != "" && fk.OnDelete != "RESTRICT" {
					b.WriteString("\n")
					b.WriteString(dimmedStyle.Render(fmt.Sprintf("    ON DELETE %s ON UPDATE %s", fk.OnDelete, fk.OnUpdate)))
				}
			}
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

// formatRowCount returns a human-readable row count string.
// Numbers < 1000 are shown as-is, 1K-999K with one decimal, 1M+ with one decimal.
func formatRowCount(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func (m *Model) applyFilter() {
	var base []database.TableInfo
	if m.filterText == "" {
		base = m.allTables
	} else {
		needle := strings.ToLower(m.filterText)
		for _, t := range m.allTables {
			if strings.Contains(strings.ToLower(t.Name), needle) {
				base = append(base, t)
			}
		}
	}

	// Partition into favorites first, then the rest
	var favs, rest []database.TableInfo
	for _, t := range base {
		if m.favorites[t.Name] {
			favs = append(favs, t)
		} else {
			rest = append(rest, t)
		}
	}

	m.favSeparator = -1
	if len(favs) > 0 && len(rest) > 0 {
		m.favSeparator = len(favs)
	}
	m.filteredTables = append(favs, rest...)

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filteredTables) {
		if len(m.filteredTables) > 0 {
			m.cursor = len(m.filteredTables) - 1
		} else {
			m.cursor = 0
		}
	}
}

