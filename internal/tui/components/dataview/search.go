package dataview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Search operators
var searchOperators = []string{
	"contains",   // LIKE '%val%'
	"equals",     // = 'val'
	"starts with", // LIKE 'val%'
	"ends with",  // LIKE '%val'
}

// searchFormField tracks which field in the form is focused.
type searchFormField int

const (
	fieldColumn   searchFormField = iota
	fieldOperator
	fieldValue
)

// searchFormState holds the state for the server-side search form.
type searchFormState struct {
	active      bool
	field       searchFormField // which field is focused
	columnIdx   int             // selected column index (0 = first column)
	operatorIdx int             // selected operator index
	valueInput  textinput.Model
	columns     []string // copy of available columns
}

func newSearchForm() searchFormState {
	vi := textinput.New()
	vi.Placeholder = "search value..."
	vi.Prompt = ""
	vi.CharLimit = 256
	return searchFormState{
		valueInput: vi,
	}
}

// open initializes the search form with the available columns.
func (sf *searchFormState) open(columns []string) {
	sf.active = true
	sf.field = fieldColumn
	sf.columnIdx = 0
	sf.operatorIdx = 0
	sf.columns = columns
	sf.valueInput.SetValue("")
	sf.valueInput.Blur()
}

// close hides the search form.
func (sf *searchFormState) close() {
	sf.active = false
	sf.valueInput.Blur()
}

// selectedColumn returns the currently selected column name.
func (sf *searchFormState) selectedColumn() string {
	if len(sf.columns) == 0 {
		return ""
	}
	return sf.columns[sf.columnIdx]
}

// selectedOperator returns the currently selected operator display name.
func (sf *searchFormState) selectedOperator() string {
	return searchOperators[sf.operatorIdx]
}

// buildQuery returns the SQL operator and parameterized value for the search.
func (sf *searchFormState) buildQuery() (column, operator, param string) {
	column = sf.selectedColumn()
	value := strings.TrimSpace(sf.valueInput.Value())

	switch sf.operatorIdx {
	case 0: // contains
		operator = "LIKE"
		param = "%" + value + "%"
	case 1: // equals
		operator = "="
		param = value
	case 2: // starts with
		operator = "LIKE"
		param = value + "%"
	case 3: // ends with
		operator = "LIKE"
		param = "%" + value
	}
	return
}

// displayText returns a human-readable description of the current search.
func (sf *searchFormState) displayText() string {
	col := sf.selectedColumn()
	op := sf.selectedOperator()
	val := strings.TrimSpace(sf.valueInput.Value())
	return fmt.Sprintf("%s %s '%s'", col, op, val)
}

// render draws the search form.
func (sf *searchFormState) render(width int, highlight, subtle, focusBorder, bg lipgloss.Color) string {
	if !sf.active {
		return ""
	}

	labelStyle := lipgloss.NewStyle().Foreground(subtle)
	activeLabel := lipgloss.NewStyle().Foreground(highlight).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(highlight)
	selectedStyle := lipgloss.NewStyle().
		Background(focusBorder).
		Foreground(lipgloss.Color("#1E1E2E")).
		Bold(true).
		Padding(0, 1)
	unselectedStyle := lipgloss.NewStyle().
		Foreground(subtle).
		Padding(0, 1)

	var b strings.Builder

	// Title
	b.WriteString(activeLabel.Render(" Search"))
	b.WriteString("\n")

	// Column selector
	if sf.field == fieldColumn {
		b.WriteString(activeLabel.Render(" Column: "))
	} else {
		b.WriteString(labelStyle.Render(" Column: "))
	}
	colName := sf.selectedColumn()
	if sf.field == fieldColumn {
		b.WriteString(valueStyle.Render(fmt.Sprintf("◀ %s ▶", colName)))
	} else {
		b.WriteString(valueStyle.Render(colName))
	}
	b.WriteString("\n")

	// Operator selector
	if sf.field == fieldOperator {
		b.WriteString(activeLabel.Render(" Match:  "))
	} else {
		b.WriteString(labelStyle.Render(" Match:  "))
	}
	for i, op := range searchOperators {
		if i == sf.operatorIdx {
			b.WriteString(selectedStyle.Render(op))
		} else {
			b.WriteString(unselectedStyle.Render(op))
		}
	}
	b.WriteString("\n")

	// Value input
	if sf.field == fieldValue {
		b.WriteString(activeLabel.Render(" Value:  "))
	} else {
		b.WriteString(labelStyle.Render(" Value:  "))
	}
	b.WriteString(sf.valueInput.View())
	b.WriteString("\n")

	// Help
	b.WriteString(labelStyle.Render(" Tab/↑↓ switch field  ←→ change  Enter search  Esc cancel"))

	return b.String()
}
