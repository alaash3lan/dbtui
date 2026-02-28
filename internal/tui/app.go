package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbplus/internal/database"
	"github.com/alaa/dbplus/internal/tui/components/sidebar"
	"github.com/alaa/dbplus/internal/tui/components/statusbar"
	"github.com/alaa/dbplus/internal/tui/components/titlebar"
)

// FocusedPane tracks which pane has keyboard focus.
type FocusedPane int

const (
	PaneSidebar FocusedPane = iota
	PaneDataView
	PaneQueryEditor
)

// Model is the root Bubble Tea model.
type Model struct {
	db       *database.DB
	keyMap   KeyMap
	styles   Styles
	version  string

	// Components
	sidebar   sidebar.Model
	titleBar  titlebar.Model
	statusBar statusbar.Model

	// Layout
	width        int
	height       int
	sidebarRatio float64
	focused      FocusedPane

	// State
	tables []database.TableInfo
	ready  bool
	err    error
}

// New creates the root model.
func New(db *database.DB, version string) Model {
	return Model{
		db:           db,
		keyMap:       DefaultKeyMap(),
		styles:       DefaultStyles(),
		version:      version,
		sidebar:      sidebar.New(db.DatabaseName()),
		titleBar:     titlebar.New(version),
		statusBar:    statusbar.New(db.DatabaseName(), db.User(), db.Host()),
		sidebarRatio: 0.20,
		focused:      PaneSidebar,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.fetchTableListCmd()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.ready = true

	case tea.KeyMsg:
		// Global keys handled first
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.FocusNext):
			m.cycleFocus(1)
			m.updateFocus()
			return m, nil
		case key.Matches(msg, m.keyMap.FocusPrev):
			m.cycleFocus(-1)
			m.updateFocus()
			return m, nil
		case key.Matches(msg, m.keyMap.GrowSidebar):
			m.sidebarRatio = min64(m.sidebarRatio+0.02, 0.40)
			m.updateLayout()
			return m, nil
		case key.Matches(msg, m.keyMap.ShrinkSidebar):
			m.sidebarRatio = max64(m.sidebarRatio-0.02, 0.10)
			m.updateLayout()
			return m, nil
		}

	case TableListMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.tables = msg.Tables
		m.sidebar.SetTables(msg.Tables)

	case sidebar.TableSelectedMsg:
		// Will be handled in Sprint 3 with DataView
		_ = msg.TableName

	case SchemaInfoMsg:
		if msg.Err == nil {
			m.sidebar.SetSchemaInfo(msg.Info)
		}
	}

	// Route to focused component
	switch m.focused {
	case PaneSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.err != nil {
		return m.styles.Error.Render("Error: " + m.err.Error())
	}

	// Title bar
	titleView := m.titleBar.View()

	// Status bar
	statusView := m.statusBar.View()

	// Available height for middle section
	availHeight := m.height - 2 // title + status bars

	// Sidebar
	sidebarWidth := int(float64(m.width) * m.sidebarRatio)
	mainWidth := m.width - sidebarWidth

	sidebarView := m.sidebar.View()

	// Placeholder for right pane (DataView + Editor will come in Sprint 3-4)
	rightContent := m.renderPlaceholder(mainWidth, availHeight)

	// Compose layout
	middleRow := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightContent)

	return lipgloss.JoinVertical(lipgloss.Left, titleView, middleRow, statusView)
}

func (m *Model) updateLayout() {
	sidebarWidth := int(float64(m.width) * m.sidebarRatio)
	availHeight := m.height - 2

	m.sidebar.SetSize(sidebarWidth, availHeight)
	m.sidebar.SetFocused(m.focused == PaneSidebar)
	m.titleBar.SetWidth(m.width)
	m.statusBar.SetWidth(m.width)
}

func (m *Model) updateFocus() {
	m.sidebar.SetFocused(m.focused == PaneSidebar)
}

func (m *Model) cycleFocus(dir int) {
	// For now only sidebar is focusable; more panes in Sprint 3-4
	total := 1 // will become 3
	m.focused = FocusedPane((int(m.focused) + dir + total) % total)
}

func (m Model) fetchTableListCmd() tea.Cmd {
	return func() tea.Msg {
		tables, err := m.db.ListTables()
		return TableListMsg{Tables: tables, Err: err}
	}
}

func (m Model) renderPlaceholder(width, height int) string {
	border := lipgloss.Color("#444444")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(width - 2).
		Height(height - 2)

	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Render("  Select a table to view data (Enter)")

	return style.Render(content)
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
