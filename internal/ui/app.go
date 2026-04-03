package ui

import (
	"fmt"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/prs"
	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type view int

const (
	viewList view = iota
	viewDetail
)

type AppModel struct {
	client     *api.Client
	org        string
	project    string
	activeView view
	list       prs.ListModel
	detail     prs.DetailModel
	width      int
	height     int
	showHelp   bool
}

func NewAppModel(client *api.Client, org, project string) AppModel {
	return AppModel{
		client:     client,
		org:        org,
		project:    project,
		activeView: viewList,
		list:       prs.NewListModel(client),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.list.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		// Don't intercept keys when the filter input or compose/confirm is focused
		if m.activeView == viewList && m.list.IsFiltering() {
			break
		}
		if m.activeView == viewDetail && m.detail.IsComposing() {
			break
		}

		switch {
		case key.Matches(msg, Keys.Quit):
			if m.activeView == viewDetail {
				// In detail view, q goes back to list
				m.activeView = viewList
				return m, nil
			}
			return m, tea.Quit
		case key.Matches(msg, Keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

	case prs.SelectPRMsg:
		m.activeView = viewDetail
		m.detail = prs.NewDetailModel(m.client, msg.PR)
		cmd := m.detail.Init()
		// Forward the current window size to the detail view
		var detailCmd tea.Cmd
		m.detail, detailCmd = m.detail.Update(tea.WindowSizeMsg{
			Width:  m.width,
			Height: m.height - 3, // Account for tab bar and status bar
		})
		cmds = append(cmds, cmd, detailCmd)
		return m, tea.Batch(cmds...)

	case prs.BackToListMsg:
		m.activeView = viewList
		return m, nil
	}

	// Delegate to active view
	switch m.activeView {
	case viewList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	case viewDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() tea.View {
	// Tab bar
	tabs := m.renderTabBar()

	// Main content
	var content string
	switch m.activeView {
	case viewList:
		content = m.list.View()
	case viewDetail:
		content = m.detail.View()
	}

	// Status bar
	statusBar := m.renderStatusBar()

	// Help overlay
	if m.showHelp {
		content = m.renderHelp()
	}

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, tabs, content, statusBar))
	v.AltScreen = true
	return v
}

func (m AppModel) renderTabBar() string {
	var tabs []string

	prTab := "Pull Requests"
	if m.activeView == viewList || m.activeView == viewDetail {
		tabs = append(tabs, theme.ActiveTab.Render(prTab))
	} else {
		tabs = append(tabs, theme.InactiveTab.Render(prTab))
	}

	// Placeholder for future tabs
	tabs = append(tabs, theme.InactiveTab.Render("Sprint Board"))
	tabs = append(tabs, theme.InactiveTab.Render("Work Items"))

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return theme.TabBar.Render(row)
}

func (m AppModel) renderStatusBar() string {
	left := theme.StatusBar.Render(fmt.Sprintf(" %s / %s", m.org, m.project))

	helpHint := theme.HelpKey.Render("?") + theme.HelpDesc.Render(" help")
	right := theme.StatusBar.Render(helpHint)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	spacer := lipgloss.NewStyle().Width(gap).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
}

func (m AppModel) renderHelp() string {
	listBindings := []struct {
		key  string
		desc string
	}{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"enter", "View PR details"},
		{"[ / ]", "Cycle scope"},
		{"/", "Filter PRs"},
		{"esc", "Clear filter / back"},
		{"r", "Refresh"},
		{"q", "Quit"},
		{"?", "Toggle help"},
	}

	detailBindings := []struct {
		key  string
		desc string
	}{
		{"f", "Open files view"},
		{"n/N", "Next / prev thread"},
		{"c", "Reply to focused thread"},
		{"C", "New comment thread"},
		{"s", "Resolve / reactivate thread"},
		{"a", "Approve PR"},
		{"A", "Approve with suggestions"},
		{"x", "Reject PR"},
		{"w", "Wait for author"},
		{"0", "Reset vote"},
		{"r", "Refresh"},
		{"esc", "Unfocus thread / back"},
		{"q", "Back to list"},
	}

	var lines []string
	lines = append(lines, "")

	lines = append(lines, theme.SectionHeader.Render("  PR List"))
	lines = append(lines, "")
	for _, b := range listBindings {
		line := fmt.Sprintf("  %s  %s",
			theme.HelpKey.Width(12).Render(b.key),
			theme.HelpDesc.Render(b.desc),
		)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, theme.SectionHeader.Render("  PR Detail"))
	lines = append(lines, "")
	for _, b := range detailBindings {
		line := fmt.Sprintf("  %s  %s",
			theme.HelpKey.Width(12).Render(b.key),
			theme.HelpDesc.Render(b.desc),
		)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, theme.HelpDesc.Render("  Press ? to close"))

	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}
