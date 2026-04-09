package workitems

import (
	"fmt"
	"strings"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"
	"github.com/popplywop/azboard/internal/ui/uiutil"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// --- Messages ---

type WorkItemsLoadedMsg struct {
	Items []api.WorkItem
}

type WorkItemsErrorMsg struct {
	Err error
}

type WorkItemSelectedMsg struct {
	Item api.WorkItem
}

// --- Scopes ---

type wiScope struct {
	Label      string
	AssignedTo string // "" = all, "@me" = current user
	ActiveOnly bool   // true = filter to non-terminal states server-side
}

var listScopes = []wiScope{
	{Label: "My Work", AssignedTo: "@me", ActiveOnly: true},
	{Label: "Active", ActiveOnly: true},
	{Label: "All"},
}

// --- ListModel ---

type ListModel struct {
	client        api.Clienter
	workItemTypes []string
	currentUserID string
	areaPath      string

	table      table.Model
	spinner    spinner.Model
	filter     textinput.Model
	items      []api.WorkItem
	filtered   []api.WorkItem
	filtering  bool
	loading    bool
	err        error
	width      int
	height     int
	scopeIndex int
}

func NewListModel(client api.Clienter, workItemTypes []string, currentUserID, areaPath string) ListModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.Spinner

	columns := []table.Column{
		{Title: "T", Width: 3},
		{Title: "ID", Width: 6},
		{Title: "Title", Width: 40},
		{Title: "State", Width: 12},
		{Title: "Assigned To", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	ts := table.DefaultStyles()
	ts.Header = ts.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.Primary)
	ts.Selected = ts.Selected.
		Foreground(theme.White).
		Background(theme.Primary).
		Bold(false)
	t.SetStyles(ts)

	fi := textinput.New()
	fi.Prompt = "/ "
	fiStyles := fi.Styles()
	fiStyles.Focused.Prompt = theme.FilterPrompt
	fiStyles.Blurred.Prompt = theme.FilterPrompt
	fiStyles.Focused.Text = theme.FilterText
	fiStyles.Blurred.Text = theme.FilterText
	fi.SetStyles(fiStyles)
	fi.Placeholder = "filter by title, id, state, type, assignee..."
	fi.CharLimit = 100

	return ListModel{
		client:        client,
		workItemTypes: workItemTypes,
		currentUserID: currentUserID,
		areaPath:      areaPath,
		table:         t,
		spinner:       s,
		filter:        fi,
		loading:       true,
		scopeIndex:    0,
	}
}

func (m ListModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchItems())
}

func (m ListModel) IsFiltering() bool {
	return m.filtering
}

func (m ListModel) currentScope() wiScope {
	if m.scopeIndex < 0 || m.scopeIndex >= len(listScopes) {
		return listScopes[0]
	}
	return listScopes[m.scopeIndex]
}

func (m ListModel) fetchItems() tea.Cmd {
	scope := m.currentScope()
	types := m.workItemTypes
	assignedTo := scope.AssignedTo
	activeOnly := scope.ActiveOnly
	areaPath := m.areaPath
	return func() tea.Msg {
		items, err := m.client.ListWorkItems(types, assignedTo, areaPath, activeOnly)
		if err != nil {
			return WorkItemsErrorMsg{Err: err}
		}
		return WorkItemsLoadedMsg{Items: items}
	}
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcColumns()
		m.recalcTableHeight()

	case WorkItemsLoadedMsg:
		m.loading = false
		m.items = msg.Items
		m.applyFilter()

	case WorkItemsErrorMsg:
		m.loading = false
		m.err = msg.Err

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		if m.filtering {
			switch msg.String() {
			case "esc":
				if m.filter.Value() != "" {
					m.filter.SetValue("")
					m.applyFilter()
				} else {
					m.filtering = false
					m.filter.Blur()
					m.table.Focus()
					m.recalcTableHeight()
				}
				return m, nil
			case "enter":
				m.filtering = false
				m.filter.Blur()
				m.table.Focus()
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				cmds = append(cmds, cmd)
				m.applyFilter()
				return m, tea.Batch(cmds...)
			}
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("/"))):
			m.filtering = true
			cmd := m.filter.Focus()
			m.table.Blur()
			m.recalcTableHeight()
			return m, cmd

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if !m.loading && len(m.filtered) > 0 {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.filtered) {
					item := m.filtered[idx]
					return m, func() tea.Msg { return WorkItemSelectedMsg{Item: item} }
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("["))):
			n := len(listScopes)
			m.scopeIndex = (m.scopeIndex - 1 + n) % n
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.fetchItems())

		case key.Matches(msg, key.NewBinding(key.WithKeys("]"))):
			n := len(listScopes)
			m.scopeIndex = (m.scopeIndex + 1) % n
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.fetchItems())

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.fetchItems())

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.filter.Value() != "" {
				m.filter.SetValue("")
				m.applyFilter()
				m.recalcTableHeight()
				return m, nil
			}
		}
	}

	if !m.filtering {
		prevCursor := m.table.Cursor()
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		// Rebuild rows if cursor moved so per-row colors stay correct.
		if m.table.Cursor() != prevCursor {
			m.table.SetRows(m.buildRows(m.filtered))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ListModel) recalcTableHeight() {
	filterH := 0
	if m.filtering || m.filter.Value() != "" {
		filterH = 2
	}
	h := max(m.height-8-filterH-2, 5)
	m.table.SetHeight(h)
}

func (m *ListModel) recalcColumns() {
	available := max(m.width-4, 80)
	typeW := 3
	idW := 6
	stateW := 12
	assigneeW := max(14, available/5)
	titleW := max(20, available-typeW-idW-stateW-assigneeW-4)

	m.table.SetColumns([]table.Column{
		{Title: "T", Width: typeW},
		{Title: "ID", Width: idW},
		{Title: "Title", Width: titleW},
		{Title: "State", Width: stateW},
		{Title: "Assigned To", Width: assigneeW},
	})
	if m.width > 2 {
		m.table.SetWidth(m.width - 2)
	}
}

func (m *ListModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	m.filtered = nil
	for _, item := range m.items {
		if query == "" || m.matchesItem(item, query) {
			m.filtered = append(m.filtered, item)
		}
	}
	m.table.SetRows(m.buildRows(m.filtered))
	if m.table.Cursor() >= len(m.filtered) && len(m.filtered) > 0 {
		m.table.SetCursor(0)
	}
}

func (m ListModel) matchesItem(item api.WorkItem, query string) bool {
	terms := strings.FieldsSeq(query)
	for term := range terms {
		found := strings.Contains(strings.ToLower(item.Fields.Title), term) ||
			strings.Contains(fmt.Sprintf("%d", item.ID), term) ||
			strings.Contains(strings.ToLower(item.Fields.State), term) ||
			strings.Contains(strings.ToLower(item.Fields.WorkItemType), term) ||
			strings.Contains(strings.ToLower(item.Fields.AssignedTo.DisplayName), term)
		if !found {
			return false
		}
	}
	return true
}

func (m ListModel) buildRows(items []api.WorkItem) []table.Row {
	sel := m.table.Cursor()
	rows := make([]table.Row, len(items))
	for i, item := range items {
		t := item.Fields.WorkItemType
		state := item.Fields.State
		var typeCell, stateCell string
		if i == sel {
			// Selected row: plain strings so the table's Selected background
			// covers the entire row without fighting pre-baked ANSI codes.
			typeCell = theme.WorkItemTypeIcon(t)
			stateCell = state
		} else {
			typeCell = theme.WorkItemTypeStyle(t).Render(theme.WorkItemTypeIcon(t))
			stateCell = theme.WorkItemStateStyle(state).Render(state)
		}
		rows[i] = table.Row{
			typeCell,
			fmt.Sprintf("%d", item.ID),
			uiutil.Truncate(item.Fields.Title, 60),
			stateCell,
			uiutil.Truncate(item.Fields.AssignedTo.DisplayName, 20),
		}
	}
	return rows
}

func (m ListModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading work items...\n", m.spinner.View())
	}

	if m.err != nil {
		return theme.ErrorText.Render(fmt.Sprintf("\n  Error: %s\n\n  Press 'r' to retry", m.err))
	}

	if len(m.items) == 0 {
		return "\n  No work items found.\n\n  Press 'r' to refresh, [ / ] to change scope"
	}

	var sections []string
	sections = append(sections, m.renderScopeBar())

	if m.filtering || m.filter.Value() != "" {
		filterLine := m.filter.View()
		countText := theme.FilterCount.Render(fmt.Sprintf("  %d/%d", len(m.filtered), len(m.items)))
		bar := theme.FilterBar.Render(filterLine + countText)
		sections = append(sections, bar)
	}

	sections = append(sections, theme.TableBorder.Render(m.table.View()))

	if len(m.filtered) == 0 && m.filter.Value() != "" {
		sections = append(sections, theme.FilterCount.Render(
			fmt.Sprintf("\n  No items match \"%s\" — press esc to clear", m.filter.Value()),
		))
	}

	return strings.Join(sections, "\n")
}

func (m ListModel) renderScopeBar() string {
	parts := []string{theme.HelpDesc.Render("  Scope:")}
	for i, s := range listScopes {
		label := " " + s.Label + " "
		if i == m.scopeIndex {
			parts = append(parts, theme.ActiveTab.Render(label))
		} else {
			parts = append(parts, theme.InactiveTab.Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return theme.FilterBar.Render(bar + theme.HelpDesc.Render("   [ / ] cycle"))
}
