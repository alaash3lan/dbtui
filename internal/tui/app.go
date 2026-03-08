package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbtui/internal/config"
	"github.com/alaa/dbtui/internal/database"
	"github.com/alaa/dbtui/internal/tui/components/dataview"
	"github.com/alaa/dbtui/internal/tui/components/editor"
	"github.com/alaa/dbtui/internal/tui/components/sidebar"
	"github.com/alaa/dbtui/internal/tui/components/statusbar"
	"github.com/alaa/dbtui/internal/tui/components/titlebar"
)

// FocusedPane tracks which pane has keyboard focus.
type FocusedPane int

const (
	PaneSidebar FocusedPane = iota
	PaneDataView
	PaneQueryEditor
)

const totalPanes = 3

// editorHeightRatio is the fraction of the right pane given to the editor.
const editorHeightRatio = 0.30

// Model is the root Bubble Tea model.
type Model struct {
	db           *database.DB
	keyMap       KeyMap
	styles       Styles
	theme        Theme
	version      string
	queryTimeout time.Duration

	// Components
	sidebar   sidebar.Model
	dataView  dataview.Model
	editor    editor.Model
	titleBar  titlebar.Model
	statusBar statusbar.Model

	// Layout
	width        int
	height       int
	sidebarRatio float64
	focused      FocusedPane

	// State
	tables       []database.TableInfo
	currentTable string
	ready        bool
	err          error
	showHelp     bool
	cancelQuery  context.CancelFunc
	queryRunning bool

	// Database switcher
	showDBSwitcher bool
	databases      []string
	dbCursor       int

	// Query bookmarks
	showBookmarks     bool
	bookmarks         []config.SavedQuery
	bookmarkCursor    int
	savingBookmark    bool
	bookmarkNameInput textinput.Model

	// Delete row confirmation
	showDeleteConfirm bool
	deleteRowInfo     *deleteRowInfo

	// Table favorites
	favoritesCfg *config.FavoritesConfig
}

// deleteRowInfo holds info needed to confirm and execute a row deletion.
type deleteRowInfo struct {
	table      string
	pkColumns  []string
	pkValues   []string
	rowPreview string
}

// New creates the root model.
func New(db *database.DB, version string, queryTimeout time.Duration, pageSize int, historyCfg config.HistoryConfig) Model {
	ed := editor.New(historyCfg.MaxEntries)
	ed.SetHistoryConfig(historyCfg.SaveToFile, historyCfg.File)
	ed.LoadHistory()

	ti := textinput.New()
	ti.Placeholder = "Bookmark name..."
	ti.CharLimit = 100
	ti.Width = 30

	bookmarksCfg := config.LoadBookmarks()
	favoritesCfg := config.LoadFavorites()

	sb := sidebar.New(db.DatabaseName())
	sb.SetFavorites(favoritesCfg.Favorites[db.DatabaseName()])

	m := Model{
		db:                db,
		keyMap:            DefaultKeyMap(),
		styles:            DefaultStyles(),
		theme:             DarkTheme(),
		version:           version,
		queryTimeout:      queryTimeout,
		sidebar:           sb,
		dataView:          dataview.New(pageSize),
		editor:            ed,
		titleBar:          titlebar.New(version),
		statusBar:         statusbar.New(db.DatabaseName(), db.User(), db.Host()),
		sidebarRatio:      0.20,
		focused:           PaneSidebar,
		bookmarks:         bookmarksCfg.Bookmarks,
		bookmarkNameInput: ti,
		favoritesCfg:      favoritesCfg,
	}
	m.applyTheme()
	return m
}

// SaveHistory persists the editor's query history to disk.
func (m *Model) SaveHistory() {
	m.editor.SaveHistory()
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
		// Database switcher overlay intercepts all keys when active
		if m.showDBSwitcher {
			switch msg.String() {
			case "esc":
				m.showDBSwitcher = false
				return m, nil
			case "j", "down":
				if m.dbCursor < len(m.databases)-1 {
					m.dbCursor++
				}
				return m, nil
			case "k", "up":
				if m.dbCursor > 0 {
					m.dbCursor--
				}
				return m, nil
			case "enter":
				if len(m.databases) > 0 && m.dbCursor < len(m.databases) {
					selected := m.databases[m.dbCursor]
					m.showDBSwitcher = false
					return m, m.switchDatabaseCmd(selected)
				}
				return m, nil
			}
			return m, nil
		}

		// Bookmark name input overlay intercepts all keys when active
		if m.savingBookmark {
			switch msg.String() {
			case "esc":
				m.savingBookmark = false
				m.bookmarkNameInput.SetValue("")
				m.bookmarkNameInput.Blur()
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.bookmarkNameInput.Value())
				if name == "" {
					return m, nil
				}
				sql := strings.TrimSpace(m.editor.Value())
				m.savingBookmark = false
				m.bookmarkNameInput.SetValue("")
				m.bookmarkNameInput.Blur()
				return m, m.saveBookmarkCmd(name, sql)
			default:
				var cmd tea.Cmd
				m.bookmarkNameInput, cmd = m.bookmarkNameInput.Update(msg)
				return m, cmd
			}
		}

		// Bookmark selector overlay intercepts all keys when active
		if m.showBookmarks {
			switch msg.String() {
			case "esc":
				m.showBookmarks = false
				return m, nil
			case "j", "down":
				if m.bookmarkCursor < len(m.bookmarks)-1 {
					m.bookmarkCursor++
				}
				return m, nil
			case "k", "up":
				if m.bookmarkCursor > 0 {
					m.bookmarkCursor--
				}
				return m, nil
			case "enter":
				if len(m.bookmarks) > 0 && m.bookmarkCursor < len(m.bookmarks) {
					selected := m.bookmarks[m.bookmarkCursor]
					m.showBookmarks = false
					return m, func() tea.Msg {
						return bookmarkSelectedMsg{SQL: selected.SQL}
					}
				}
				return m, nil
			case "x":
				if len(m.bookmarks) > 0 && m.bookmarkCursor < len(m.bookmarks) {
					m.bookmarks = append(m.bookmarks[:m.bookmarkCursor], m.bookmarks[m.bookmarkCursor+1:]...)
					if m.bookmarkCursor >= len(m.bookmarks) && m.bookmarkCursor > 0 {
						m.bookmarkCursor--
					}
					return m, m.persistBookmarksCmd()
				}
				return m, nil
			}
			return m, nil
		}

		// Delete row confirmation overlay intercepts all keys when active
		if m.showDeleteConfirm {
			switch msg.String() {
			case "y", "Y":
				m.showDeleteConfirm = false
				info := m.deleteRowInfo
				m.deleteRowInfo = nil
				return m, m.deleteRowCmd(info.table, info.pkColumns, info.pkValues)
			case "n", "N", "esc":
				m.showDeleteConfirm = false
				m.deleteRowInfo = nil
				return m, nil
			}
			return m, nil
		}

		// Global keys handled first
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			if m.queryRunning && m.cancelQuery != nil {
				m.cancelQuery()
				m.cancelQuery = nil
				return m, nil
			}
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
			m.sidebarRatio = min(m.sidebarRatio+0.02, 0.40)
			m.updateLayout()
			return m, nil
		case key.Matches(msg, m.keyMap.ShrinkSidebar):
			m.sidebarRatio = max(m.sidebarRatio-0.02, 0.10)
			m.updateLayout()
			return m, nil
		case key.Matches(msg, m.keyMap.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keyMap.ToggleTheme):
			if m.theme.Name == "dark" {
				m.theme = LightTheme()
			} else {
				m.theme = DarkTheme()
			}
			m.applyTheme()
			return m, nil
		case key.Matches(msg, m.keyMap.Refresh):
			batch := []tea.Cmd{m.fetchTableListCmd()}
			if m.currentTable != "" {
				batch = append(batch, m.fetchTableDataCmd(m.currentTable), m.fetchCountCmd(m.currentTable))
			}
			return m, tea.Batch(batch...)
		case key.Matches(msg, m.keyMap.ExportCSV):
			return m, func() tea.Msg { return exportRequestMsg{Format: "csv"} }
		case key.Matches(msg, m.keyMap.ExportJSON):
			return m, func() tea.Msg { return exportRequestMsg{Format: "json"} }
		case key.Matches(msg, m.keyMap.ExplainQuery):
			val := strings.TrimSpace(m.editor.Value())
			val = strings.TrimRight(val, "; \t\n\r")
			if val == "" {
				return m, nil
			}
			sql := "EXPLAIN " + val
			m.editor.SetRunning(true)
			m.queryRunning = true
			m.err = nil
			return m, m.executeQueryCmd(sql)
		case key.Matches(msg, m.keyMap.SwitchDB):
			m.showDBSwitcher = true
			m.dbCursor = 0
			return m, m.fetchDatabaseListCmd()
		case key.Matches(msg, m.keyMap.Bookmarks):
			m.showBookmarks = true
			m.bookmarkCursor = 0
			return m, nil
		case key.Matches(msg, m.keyMap.SaveBookmark):
			val := strings.TrimSpace(m.editor.Value())
			if val == "" {
				return m, nil
			}
			m.savingBookmark = true
			m.bookmarkNameInput.SetValue("")
			m.bookmarkNameInput.Focus()
			return m, m.bookmarkNameInput.Cursor.BlinkCmd()
		}

	case reconnectMsg:
		m.statusBar.SetConnectionStatus("Reconnecting...")
		return m, m.reconnectCmd()

	case reconnectResultMsg:
		if msg.Err != nil {
			m.statusBar.SetConnectionStatus("Disconnected")
			m.err = fmt.Errorf("reconnect failed: %w", msg.Err)
			return m, nil
		}
		m.statusBar.SetConnectionStatus("")
		m.err = nil
		return m, m.fetchTableListCmd()

	case switchDatabaseResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.sidebar.SetDBName(msg.Name)
		m.sidebar.SetFavorites(m.favoritesCfg.Favorites[msg.Name])
		m.statusBar.SetDBName(msg.Name)
		m.currentTable = ""
		m.dataView.SetData("", nil, nil)
		m.editor.SetTableNames(nil) // clear stale autocomplete until new list arrives
		m.editor.SetResult(fmt.Sprintf("Database changed to %s", msg.Name))
		return m, m.fetchTableListCmd()

	case databaseListMsg:
		if msg.Err != nil {
			m.showDBSwitcher = false
			m.err = msg.Err
			return m, nil
		}
		m.databases = msg.Databases
		// Position cursor on current database
		currentDB := m.db.DatabaseName()
		for i, db := range m.databases {
			if db == currentDB {
				m.dbCursor = i
				break
			}
		}

	case TableListMsg:
		if msg.Err != nil {
			if database.IsConnectionError(msg.Err) {
				return m, func() tea.Msg { return reconnectMsg{} }
			}
			m.err = msg.Err
			return m, nil
		}
		m.tables = msg.Tables
		m.sidebar.SetTables(msg.Tables)
		// Feed table names to editor for autocomplete
		names := make([]string, len(msg.Tables))
		for i, t := range msg.Tables {
			names[i] = t.Name
		}
		m.editor.SetTableNames(names)

	case sidebar.TableSelectedMsg:
		m.currentTable = msg.TableName
		m.err = nil
		m.editor.ClearStatus()
		return m, tea.Batch(
			m.fetchTableDataCmd(msg.TableName),
			m.fetchCountCmd(msg.TableName),
			m.fetchSchemaCmd(msg.TableName),
		)

	case sidebar.SchemaRequestMsg:
		return m, m.fetchSchemaCmd(msg.TableName)

	case sidebar.FavoriteToggledMsg:
		return m, m.saveFavoriteCmd(msg.TableName, msg.IsFavorite)

	case editor.ExecuteQueryMsg:
		m.editor.SetRunning(true)
		m.queryRunning = true
		m.err = nil
		return m, m.executeQueryCmd(msg.SQL)

	case QueryResultMsg:
		m.editor.SetRunning(false)
		m.queryRunning = false
		m.cancelQuery = nil
		if msg.Err != nil {
			if database.IsConnectionError(msg.Err) {
				m.editor.SetError("Connection lost. Reconnecting...")
				return m, func() tea.Msg { return reconnectMsg{} }
			}
			m.editor.SetError(msg.Err.Error())
			return m, nil
		}
		m.err = nil
		if msg.IsSelect {
			tableName := m.currentTable
			if tableName == "" {
				tableName = "query"
			}
			m.dataView.SetData(tableName, msg.Columns, msg.Rows)
			m.titleBar.SetRowCount(msg.RowCount)
			m.statusBar.SetQueryInfo(msg.Duration, msg.RowCount)
			m.editor.SetResult(fmt.Sprintf("%d rows in %s", msg.RowCount, msg.Duration))
		} else {
			// Handle database switch
			if msg.DatabaseChanged != "" {
				m.sidebar.SetDBName(msg.DatabaseChanged)
				m.sidebar.SetFavorites(m.favoritesCfg.Favorites[msg.DatabaseChanged])
				m.statusBar.SetDBName(msg.DatabaseChanged)
				m.currentTable = ""
				m.dataView.SetData("", nil, nil)
				m.editor.SetTableNames(nil) // clear stale autocomplete
				m.editor.SetResult(fmt.Sprintf("Database changed to %s", msg.DatabaseChanged))
				return m, m.fetchTableListCmd()
			}
			m.editor.SetResult(fmt.Sprintf("%d rows affected in %s", msg.AffectedRows, msg.Duration))
			m.statusBar.SetQueryInfo(msg.Duration, int(msg.AffectedRows))
			// Refresh table list (handles CREATE/DROP TABLE) and current table data
			batch := []tea.Cmd{m.fetchTableListCmd()}
			if m.currentTable != "" {
				batch = append(batch, m.fetchTableDataCmd(m.currentTable), m.fetchCountCmd(m.currentTable))
			}
			return m, tea.Batch(batch...)
		}

	case dataview.PageRequestMsg:
		return m, m.fetchPageCmd(msg.Table, msg.Page, msg.Offset, msg.Limit)

	case pageDataMsg:
		if msg.err != nil {
			if database.IsConnectionError(msg.err) {
				return m, func() tea.Msg { return reconnectMsg{} }
			}
			m.err = msg.err
			return m, nil
		}
		m.dataView.SetData(msg.table, msg.columns, msg.rows)
		m.dataView.SetPageDirect(msg.page)
		m.titleBar.SetRowCount(len(msg.rows))
		return m, nil

	case tableCountMsg:
		m.dataView.SetTotalRows(msg.count)

	case SchemaInfoMsg:
		if msg.Err == nil {
			m.sidebar.SetSchemaInfo(msg.Info)
			m.dataView.SetSchemaInfo(msg.Info)
		}

	case exportRequestMsg:
		columns := m.dataView.Columns()
		rows := m.dataView.Rows()
		if len(columns) == 0 || len(rows) == 0 {
			m.editor.SetError("No data to export")
			return m, nil
		}
		format := msg.Format
		return m, m.exportCmd(columns, rows, format)

	case exportResultMsg:
		if msg.Err != nil {
			m.editor.SetError(fmt.Sprintf("Export failed: %s", msg.Err))
			return m, nil
		}
		rowCount := len(m.dataView.Rows())
		m.editor.SetResult(fmt.Sprintf("Exported %d rows to %s", rowCount, filepath.Base(msg.Path)))

	case dataview.DeleteRowRequestMsg:
		m.showDeleteConfirm = true
		m.deleteRowInfo = &deleteRowInfo{
			table:      msg.Table,
			pkColumns:  msg.PKColumns,
			pkValues:   msg.PKValues,
			rowPreview: msg.RowPreview,
		}
		return m, nil

	case deleteRowResultMsg:
		if msg.Err != nil {
			if database.IsConnectionError(msg.Err) {
				m.editor.SetError("Connection lost. Reconnecting...")
				return m, func() tea.Msg { return reconnectMsg{} }
			}
			m.editor.SetError(fmt.Sprintf("Delete failed: %s", msg.Err))
			return m, nil
		}
		m.editor.SetResult(fmt.Sprintf("Deleted %d row(s)", msg.AffectedRows))
		// Refresh table data and count
		if m.currentTable != "" {
			return m, tea.Batch(
				m.fetchTableDataCmd(m.currentTable),
				m.fetchCountCmd(m.currentTable),
			)
		}
		return m, nil

	case dataview.CopyToClipboardMsg:
		return m, m.copyToClipboardCmd(msg.Text)

	case clipboardResultMsg:
		if msg.Err != nil {
			m.editor.SetError(fmt.Sprintf("Copy failed: %s", msg.Err))
		} else {
			m.editor.SetResult("Copied to clipboard")
		}
		return m, nil

	case bookmarkSelectedMsg:
		m.editor.SetValue(msg.SQL)
		m.editor.ClearStatus()
		return m, nil

	case bookmarkSavedMsg:
		if msg.Err != nil {
			m.editor.SetError(fmt.Sprintf("Bookmark save failed: %s", msg.Err))
		} else {
			// Reload bookmarks from disk to stay in sync
			bookmarksCfg := config.LoadBookmarks()
			m.bookmarks = bookmarksCfg.Bookmarks
			if msg.Name != "" {
				m.editor.SetResult(fmt.Sprintf("Bookmark saved: %s", msg.Name))
			}
		}
		return m, nil

	case favoriteSavedMsg:
		if msg.Err != nil {
			m.editor.SetError(fmt.Sprintf("Favorite save failed: %s", msg.Err))
		}
		return m, nil
	}

	// Route cursor blink and other non-key messages to textinput when saving bookmark
	if m.savingBookmark {
		var cmd tea.Cmd
		m.bookmarkNameInput, cmd = m.bookmarkNameInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	// Route to focused component
	switch m.focused {
	case PaneSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case PaneDataView:
		var cmd tea.Cmd
		m.dataView, cmd = m.dataView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case PaneQueryEditor:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
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

	// Minimum terminal size
	if m.width < 60 || m.height < 16 {
		return lipgloss.NewStyle().
			Foreground(m.theme.WarningColor).
			Bold(true).
			Render(fmt.Sprintf("Terminal too small (%dx%d). Minimum: 60x16", m.width, m.height))
	}

	// Help overlay
	if m.showHelp {
		return m.renderHelp()
	}

	// Database switcher overlay
	if m.showDBSwitcher {
		return m.renderDBSwitcher()
	}

	// Bookmark selector overlay
	if m.showBookmarks {
		return m.renderBookmarks()
	}

	// Bookmark name input overlay
	if m.savingBookmark {
		return m.renderBookmarkSave()
	}

	// Delete row confirmation overlay
	if m.showDeleteConfirm {
		return m.renderDeleteConfirm()
	}

	// Title bar
	titleView := m.titleBar.View()

	// Status bar
	statusView := m.statusBar.View()

	// Available height for middle section
	availHeight := m.height - 2

	// Sidebar
	sidebarWidth := int(float64(m.width) * m.sidebarRatio)
	mainWidth := m.width - sidebarWidth

	sidebarView := m.sidebar.View()

	// Right pane: DataView (top) + Editor (bottom)
	editorHeight := int(float64(availHeight) * editorHeightRatio)
	if editorHeight < 6 {
		editorHeight = 6
	}
	dataViewHeight := availHeight - editorHeight

	dataViewView := m.dataView.View()
	editorView := m.editor.View()

	// Error overlay in data area
	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.ErrorColor).
			Width(mainWidth - 2).
			Height(dataViewHeight - 2).
			Padding(1)
		dataViewView = errStyle.Render(
			m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)),
		)
	}

	rightPane := lipgloss.JoinVertical(lipgloss.Left, dataViewView, editorView)

	// Compose layout
	middleRow := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, titleView, middleRow, statusView)
}

func (m *Model) updateLayout() {
	sidebarWidth := int(float64(m.width) * m.sidebarRatio)
	mainWidth := m.width - sidebarWidth
	availHeight := m.height - 2

	editorHeight := int(float64(availHeight) * editorHeightRatio)
	if editorHeight < 6 {
		editorHeight = 6
	}
	dataViewHeight := availHeight - editorHeight

	m.sidebar.SetSize(sidebarWidth, availHeight)
	m.dataView.SetSize(mainWidth, dataViewHeight)
	m.editor.SetSize(mainWidth, editorHeight)
	m.updateFocus()
	m.titleBar.SetWidth(m.width)
	m.statusBar.SetWidth(m.width)
}

func (m *Model) updateFocus() {
	m.sidebar.SetFocused(m.focused == PaneSidebar)
	m.dataView.SetFocused(m.focused == PaneDataView)
	m.editor.SetFocused(m.focused == PaneQueryEditor)
}

func (m *Model) cycleFocus(dir int) {
	next := (int(m.focused) + dir + totalPanes) % totalPanes
	m.focused = FocusedPane(next)
}

// tea.Cmd factories

func (m Model) fetchTableListCmd() tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		tables, err := db.ListTables(ctx)
		return TableListMsg{Tables: tables, Err: err}
	}
}

func (m Model) fetchTableDataCmd(table string) tea.Cmd {
	db := m.db
	pageSize := m.dataView.PageSize()
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		result, err := db.FetchTableData(ctx, table, pageSize, 0)
		if err != nil {
			return QueryResultMsg{Err: err}
		}
		return QueryResultMsg{
			Columns:  result.Columns,
			Rows:     result.Rows,
			RowCount: result.RowCount,
			Duration: result.Duration,
			IsSelect: true,
		}
	}
}

func (m *Model) executeQueryCmd(sql string) tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.cancelQuery = cancel
	return func() tea.Msg {
		defer cancel()
		result, err := db.Execute(ctx, sql)
		if err != nil {
			return QueryResultMsg{Err: err}
		}
		return QueryResultMsg{
			Columns:         result.Columns,
			Rows:            result.Rows,
			RowCount:        result.RowCount,
			AffectedRows:    result.AffectedRows,
			Duration:        result.Duration,
			IsSelect:        result.IsSelect,
			DatabaseChanged: result.DatabaseChanged,
		}
	}
}

type pageDataMsg struct {
	table   string
	page    int
	columns []string
	rows    [][]string
	err     error
}

type tableCountMsg struct {
	count int64
}

func (m Model) fetchPageCmd(table string, page, offset, limit int) tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		result, err := db.FetchTableData(ctx, table, limit, offset)
		if err != nil {
			return pageDataMsg{table: table, page: page, err: err}
		}
		return pageDataMsg{
			table:   table,
			page:    page,
			columns: result.Columns,
			rows:    result.Rows,
		}
	}
}

func (m Model) fetchCountCmd(table string) tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		count, err := db.CountRows(ctx, table)
		if err != nil {
			return tableCountMsg{count: 0}
		}
		return tableCountMsg{count: count}
	}
}

func (m Model) reconnectCmd() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		err := db.EnsureConnected()
		return reconnectResultMsg{Err: err}
	}
}

func (m Model) fetchSchemaCmd(table string) tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		info, err := db.DescribeTable(ctx, table)
		return SchemaInfoMsg{Info: info, Err: err}
	}
}

func (m Model) fetchDatabaseListCmd() tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		databases, err := db.ListDatabases(ctx)
		return databaseListMsg{Databases: databases, Err: err}
	}
}

func (m Model) switchDatabaseCmd(name string) tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := db.SwitchDatabase(ctx, name)
		return switchDatabaseResultMsg{Name: name, Err: err}
	}
}

func (m Model) exportCmd(columns []string, rows [][]string, format string) tea.Cmd {
	return func() tea.Msg {
		cwd, err := os.Getwd()
		if err != nil {
			return exportResultMsg{Err: err}
		}
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("dbtui_export_%s.%s", timestamp, format)
		path := filepath.Join(cwd, filename)

		switch format {
		case "json":
			err = exportJSON(columns, rows, path)
		default:
			err = exportCSV(columns, rows, path)
		}
		return exportResultMsg{Path: path, Err: err}
	}
}

func (m Model) deleteRowCmd(table string, pkColumns []string, pkValues []string) tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		affected, err := db.DeleteRow(ctx, table, pkColumns, pkValues)
		return deleteRowResultMsg{AffectedRows: affected, Err: err}
	}
}

func (m Model) renderDeleteConfirm() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.ErrorColor).Bold(true)
	desc := lipgloss.NewStyle().Foreground(t.Text)
	dim := lipgloss.NewStyle().Foreground(t.Subtle)
	warn := lipgloss.NewStyle().Foreground(t.WarningColor).Bold(true)

	content := title.Render("Delete Row") + "\n\n"
	content += warn.Render("Are you sure you want to delete this row?") + "\n\n"

	if m.deleteRowInfo != nil {
		// Show a preview of the row data, limit lines
		lines := strings.Split(m.deleteRowInfo.rowPreview, "\n")
		maxLines := m.height - 14
		if maxLines < 5 {
			maxLines = 5
		}
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			lines = append(lines, fmt.Sprintf("... %d more columns", len(strings.Split(m.deleteRowInfo.rowPreview, "\n"))-maxLines))
		}
		for _, line := range lines {
			content += desc.Render("  "+line) + "\n"
		}
	}

	content += "\n" + dim.Render("y: confirm delete  n/Esc: cancel")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.ErrorColor).
		Padding(1, 2).
		Width(50)

	rendered := box.Render(content)
	hPad := (m.width - lipgloss.Width(rendered)) / 2
	vPad := (m.height - lipgloss.Height(rendered)) / 2
	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	padded := strings.Repeat("\n", vPad)
	hPadding := strings.Repeat(" ", hPad)
	renderedLines := strings.Split(rendered, "\n")
	for _, line := range renderedLines {
		padded += hPadding + line + "\n"
	}

	return padded
}

func (m Model) renderHelp() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Highlight).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(t.WarningColor).Bold(true)
	desc := lipgloss.NewStyle().Foreground(t.Text)
	dim := lipgloss.NewStyle().Foreground(t.Subtle)

	help := title.Render("dbtui Keyboard Shortcuts") + "\n\n"

	help += title.Render("Global") + "\n"
	help += keyStyle.Render("  Ctrl+C      ") + desc.Render("Quit") + "\n"
	help += keyStyle.Render("  Tab         ") + desc.Render("Next pane") + "\n"
	help += keyStyle.Render("  Shift+Tab   ") + desc.Render("Previous pane") + "\n"
	help += keyStyle.Render("  Ctrl+Left   ") + desc.Render("Shrink sidebar") + "\n"
	help += keyStyle.Render("  Ctrl+Right  ") + desc.Render("Grow sidebar") + "\n"
	help += keyStyle.Render("  Ctrl+T      ") + desc.Render("Toggle dark/light theme") + "\n"
	help += keyStyle.Render("  Ctrl+R      ") + desc.Render("Refresh tables & data") + "\n"
	help += keyStyle.Render("  Ctrl+S      ") + desc.Render("Export data as CSV") + "\n"
	help += keyStyle.Render("  Ctrl+J      ") + desc.Render("Export data as JSON") + "\n"
	help += keyStyle.Render("  Ctrl+X      ") + desc.Render("Explain current query") + "\n"
	help += keyStyle.Render("  Ctrl+D      ") + desc.Render("Switch database") + "\n"
	help += keyStyle.Render("  Ctrl+B      ") + desc.Render("Open query bookmarks") + "\n"
	help += keyStyle.Render("  Ctrl+K      ") + desc.Render("Save query as bookmark") + "\n"
	help += keyStyle.Render("  F1          ") + desc.Render("Toggle this help") + "\n\n"

	help += title.Render("Sidebar") + "\n"
	help += keyStyle.Render("  j/k arrows  ") + desc.Render("Navigate tables") + "\n"
	help += keyStyle.Render("  Enter       ") + desc.Render("Select table, load data") + "\n"
	help += keyStyle.Render("  i           ") + desc.Render("Toggle schema info") + "\n"
	help += keyStyle.Render("  g/G         ") + desc.Render("First/last table") + "\n"
	help += keyStyle.Render("  /           ") + desc.Render("Filter tables") + "\n"
	help += keyStyle.Render("  f           ") + desc.Render("Toggle table favorite") + "\n"
	help += keyStyle.Render("  Escape      ") + desc.Render("Clear filter") + "\n\n"

	help += title.Render("Data View") + "\n"
	help += keyStyle.Render("  arrows/hjkl ") + desc.Render("Scroll grid") + "\n"
	help += keyStyle.Render("  / Ctrl+F    ") + desc.Render("Activate filter") + "\n"
	help += keyStyle.Render("  Escape      ") + desc.Render("Clear filter") + "\n"
	help += keyStyle.Render("  PgUp/PgDn   ") + desc.Render("Scroll viewport up/down") + "\n"
	help += keyStyle.Render("  n/p         ") + desc.Render("Next/prev server page") + "\n"
	help += keyStyle.Render("  Home/End    ") + desc.Render("First/last row") + "\n"
	help += keyStyle.Render("  c           ") + desc.Render("Copy cell to clipboard") + "\n"
	help += keyStyle.Render("  y           ") + desc.Render("Copy row to clipboard") + "\n"
	help += keyStyle.Render("  d           ") + desc.Render("Toggle row detail view") + "\n"
	help += keyStyle.Render("  s           ") + desc.Render("Sort by current column") + "\n"
	help += keyStyle.Render("  Ctrl+N      ") + desc.Render("Toggle row numbers") + "\n"
	help += keyStyle.Render("  x           ") + desc.Render("Delete row (with confirmation)") + "\n\n"

	help += title.Render("Query Editor") + "\n"
	help += keyStyle.Render("  Enter       ") + desc.Render("Execute (requires ;) or newline") + "\n"
	help += keyStyle.Render("  Ctrl+E      ") + desc.Render("Force execute") + "\n"
	help += keyStyle.Render("  Up/Down     ") + desc.Render("Navigate history") + "\n"
	help += keyStyle.Render("  Escape      ") + desc.Render("Clear input") + "\n\n"

	help += title.Render("Tips") + "\n"
	help += desc.Render("  Type USE dbname; to switch databases") + "\n"
	help += desc.Render("  Ctrl+C cancels running query, press again to quit") + "\n\n"

	help += dim.Render("Press F1 to close")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Highlight).
		Padding(1, 2).
		Width(50)

	// Center in terminal
	rendered := box.Render(help)
	hPad := (m.width - lipgloss.Width(rendered)) / 2
	vPad := (m.height - lipgloss.Height(rendered)) / 2
	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	padded := strings.Repeat("\n", vPad)
	hPadding := strings.Repeat(" ", hPad)
	lines := strings.Split(rendered, "\n")
	for _, line := range lines {
		padded += hPadding + line + "\n"
	}

	return padded
}

func (m Model) renderDBSwitcher() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Highlight).Bold(true)
	itemStyle := lipgloss.NewStyle().Foreground(t.Text)
	cursorStyle := lipgloss.NewStyle().Foreground(t.Highlight).Bold(true)
	currentStyle := lipgloss.NewStyle().Foreground(t.SuccessColor)
	dim := lipgloss.NewStyle().Foreground(t.Subtle)

	content := title.Render("Switch Database") + "\n\n"

	if len(m.databases) == 0 {
		content += dim.Render("Loading...")
	} else {
		currentDB := m.db.DatabaseName()
		// Calculate visible window for scrolling
		maxVisible := m.height - 10
		if maxVisible < 5 {
			maxVisible = 5
		}
		start := 0
		if m.dbCursor >= maxVisible {
			start = m.dbCursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(m.databases) {
			end = len(m.databases)
		}

		for i := start; i < end; i++ {
			db := m.databases[i]
			prefix := "  "
			if i == m.dbCursor {
				prefix = "> "
			}

			line := prefix + db
			if db == currentDB {
				line += " (current)"
			}

			if i == m.dbCursor {
				content += cursorStyle.Render(line)
			} else if db == currentDB {
				content += currentStyle.Render(line)
			} else {
				content += itemStyle.Render(line)
			}
			if i < end-1 {
				content += "\n"
			}
		}

		if end < len(m.databases) {
			content += "\n" + dim.Render(fmt.Sprintf("  ... %d more", len(m.databases)-end))
		}
	}

	content += "\n\n" + dim.Render("j/k: navigate  Enter: select  Esc: close")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Highlight).
		Padding(1, 2).
		Width(40)

	// Center in terminal
	rendered := box.Render(content)
	hPad := (m.width - lipgloss.Width(rendered)) / 2
	vPad := (m.height - lipgloss.Height(rendered)) / 2
	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	padded := strings.Repeat("\n", vPad)
	hPadding := strings.Repeat(" ", hPad)
	lines := strings.Split(rendered, "\n")
	for _, line := range lines {
		padded += hPadding + line + "\n"
	}

	return padded
}

func (m Model) renderBookmarks() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Highlight).Bold(true)
	itemStyle := lipgloss.NewStyle().Foreground(t.Text)
	cursorStyle := lipgloss.NewStyle().Foreground(t.Highlight).Bold(true)
	dim := lipgloss.NewStyle().Foreground(t.Subtle)

	content := title.Render("Query Bookmarks") + "\n\n"

	if len(m.bookmarks) == 0 {
		content += dim.Render("No bookmarks saved yet.")
		content += "\n" + dim.Render("Press Ctrl+K to save the current query.")
	} else {
		// Calculate visible window for scrolling
		maxVisible := m.height - 10
		if maxVisible < 5 {
			maxVisible = 5
		}
		start := 0
		if m.bookmarkCursor >= maxVisible {
			start = m.bookmarkCursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(m.bookmarks) {
			end = len(m.bookmarks)
		}

		for i := start; i < end; i++ {
			bm := m.bookmarks[i]
			prefix := "  "
			if i == m.bookmarkCursor {
				prefix = "> "
			}

			line := prefix + bm.Name
			if i == m.bookmarkCursor {
				content += cursorStyle.Render(line)
			} else {
				content += itemStyle.Render(line)
			}
			if i < end-1 {
				content += "\n"
			}
		}

		if end < len(m.bookmarks) {
			content += "\n" + dim.Render(fmt.Sprintf("  ... %d more", len(m.bookmarks)-end))
		}
	}

	content += "\n\n" + dim.Render("j/k: navigate  Enter: load  x: delete  Esc: close")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Highlight).
		Padding(1, 2).
		Width(50)

	// Center in terminal
	rendered := box.Render(content)
	hPad := (m.width - lipgloss.Width(rendered)) / 2
	vPad := (m.height - lipgloss.Height(rendered)) / 2
	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	padded := strings.Repeat("\n", vPad)
	hPadding := strings.Repeat(" ", hPad)
	lines := strings.Split(rendered, "\n")
	for _, line := range lines {
		padded += hPadding + line + "\n"
	}

	return padded
}

func (m Model) renderBookmarkSave() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Highlight).Bold(true)
	dim := lipgloss.NewStyle().Foreground(t.Subtle)

	content := title.Render("Save Bookmark") + "\n\n"
	content += m.bookmarkNameInput.View() + "\n\n"
	content += dim.Render("Enter: save  Esc: cancel")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Highlight).
		Padding(1, 2).
		Width(40)

	// Center in terminal
	rendered := box.Render(content)
	hPad := (m.width - lipgloss.Width(rendered)) / 2
	vPad := (m.height - lipgloss.Height(rendered)) / 2
	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	padded := strings.Repeat("\n", vPad)
	hPadding := strings.Repeat(" ", hPad)
	lines := strings.Split(rendered, "\n")
	for _, line := range lines {
		padded += hPadding + line + "\n"
	}

	return padded
}

func (m *Model) saveBookmarkCmd(name, sql string) tea.Cmd {
	return func() tea.Msg {
		bookmarksCfg := config.LoadBookmarks()
		bookmarksCfg.Bookmarks = append(bookmarksCfg.Bookmarks, config.SavedQuery{
			Name: name,
			SQL:  sql,
		})
		err := config.SaveBookmarks(bookmarksCfg)
		return bookmarkSavedMsg{Name: name, Err: err}
	}
}

func (m *Model) persistBookmarksCmd() tea.Cmd {
	bookmarks := make([]config.SavedQuery, len(m.bookmarks))
	copy(bookmarks, m.bookmarks)
	return func() tea.Msg {
		cfg := &config.BookmarksConfig{Bookmarks: bookmarks}
		err := config.SaveBookmarks(cfg)
		return bookmarkSavedMsg{Name: "", Err: err}
	}
}

func (m *Model) saveFavoriteCmd(tableName string, isFav bool) tea.Cmd {
	dbName := m.db.DatabaseName()

	// Update in-memory config
	favs := m.favoritesCfg.Favorites[dbName]
	if isFav {
		favs = append(favs, tableName)
	} else {
		var filtered []string
		for _, f := range favs {
			if f != tableName {
				filtered = append(filtered, f)
			}
		}
		favs = filtered
	}
	if len(favs) == 0 {
		delete(m.favoritesCfg.Favorites, dbName)
	} else {
		m.favoritesCfg.Favorites[dbName] = favs
	}

	cfg := m.favoritesCfg
	return func() tea.Msg {
		err := config.SaveFavorites(cfg)
		if err != nil {
			return favoriteSavedMsg{Err: err}
		}
		return favoriteSavedMsg{}
	}
}

func (m *Model) applyTheme() {
	t := m.theme
	m.sidebar.SetColors(sidebar.Colors{
		Highlight:   t.Highlight,
		Subtle:      t.Subtle,
		Border:      t.Border,
		FocusBorder: t.FocusBorder,
		ActiveBg:    t.ActiveBg,
	})
	m.dataView.SetColors(dataview.Colors{
		Highlight:    t.Highlight,
		Subtle:       t.Subtle,
		Border:       t.Border,
		FocusBorder:  t.FocusBorder,
		SelectedBg:   t.SelectedBg,
		WarningColor: t.WarningColor,
	})
	m.editor.SetColors(editor.Colors{
		Highlight:    t.Highlight,
		Subtle:       t.Subtle,
		Border:       t.Border,
		FocusBorder:  t.FocusBorder,
		ErrorColor:   t.ErrorColor,
		SuccessColor: t.SuccessColor,
		WarningColor: t.WarningColor,
		KeywordColor: t.KeywordColor,
		StringColor:  t.StringColor,
		NumberColor:  t.NumberColor,
	})
	m.titleBar.SetColors(titlebar.Colors{
		Highlight:  t.Highlight,
		Text:       t.Text,
		Background: t.HeaderBg,
	})
	m.statusBar.SetColors(statusbar.Colors{
		Highlight:  t.Highlight,
		Text:       t.Text,
		Background: t.HeaderBg,
	})
	m.styles.Error = lipgloss.NewStyle().Foreground(t.ErrorColor).Bold(true)
}
