package prs

import (
	"fmt"
	"strings"

	"azboard/internal/api"
	"azboard/internal/ui/theme"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type PRsLoadedMsg struct {
	PRs []api.PullRequest
}

type PRsErrorMsg struct {
	Err error
}

type SelectPRMsg struct {
	PR api.PullRequest
}

// ListModel is the PR list view.
type ListModel struct {
	client      *api.Client
	table       table.Model
	spinner     spinner.Model
	filter      textinput.Model
	prs         []api.PullRequest
	filteredPRs []api.PullRequest
	filtering   bool
	loading     bool
	err         error
	width       int
	height      int
}

func NewListModel(client *api.Client) ListModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.Spinner

	columns := []table.Column{
		{Title: "#", Width: 6},
		{Title: "Title", Width: 40},
		{Title: "Repository", Width: 20},
		{Title: "Author", Width: 18},
		{Title: "Status", Width: 10},
		{Title: "Reviewers", Width: 24},
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
	fi.PromptStyle = theme.FilterPrompt
	fi.TextStyle = theme.FilterText
	fi.Cursor.Style = theme.FilterCursor
	fi.Placeholder = "type to filter by title, repo, author, status..."
	fi.CharLimit = 100

	return ListModel{
		client:  client,
		table:   t,
		spinner: s,
		filter:  fi,
		loading: true,
	}
}

func (m ListModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchPRs())
}

// IsFiltering returns true when the filter text input is focused.
func (m ListModel) IsFiltering() bool {
	return m.filtering
}

func (m ListModel) fetchPRs() tea.Cmd {
	return func() tea.Msg {
		prs, err := m.client.ListPullRequests("active")
		if err != nil {
			return PRsErrorMsg{Err: err}
		}
		return PRsLoadedMsg{PRs: prs}
	}
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcColumns()
		filterHeight := 0
		if m.filtering || m.filter.Value() != "" {
			filterHeight = 2
		}
		m.table.SetHeight(m.height - 6 - filterHeight)

	case PRsLoadedMsg:
		m.loading = false
		m.prs = msg.PRs
		m.applyFilter()

	case PRsErrorMsg:
		m.loading = false
		m.err = msg.Err

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		// When filter input is focused, handle its keys first
		if m.filtering {
			switch msg.String() {
			case "esc":
				if m.filter.Value() != "" {
					// First esc clears the filter text
					m.filter.SetValue("")
					m.applyFilter()
				} else {
					// Second esc (or esc on empty) closes the filter bar
					m.filtering = false
					m.filter.Blur()
					m.table.Focus()
					m.recalcTableHeight()
				}
				return m, nil
			case "enter":
				// Confirm filter and return focus to table
				m.filtering = false
				m.filter.Blur()
				m.table.Focus()
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				// Pass keystrokes to the text input
				var cmd tea.Cmd
				m.filter, cmd = m.filter.Update(msg)
				cmds = append(cmds, cmd)
				m.applyFilter()
				return m, tea.Batch(cmds...)
			}
		}

		// Normal mode (filter not focused)
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("/"))):
			m.filtering = true
			m.filter.Focus()
			m.table.Blur()
			m.recalcTableHeight()
			return m, m.filter.Cursor.BlinkCmd()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if !m.loading && len(m.filteredPRs) > 0 {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.filteredPRs) {
					return m, func() tea.Msg {
						return SelectPRMsg{PR: m.filteredPRs[idx]}
					}
				}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.fetchPRs())
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// If there's an active filter, clear it
			if m.filter.Value() != "" {
				m.filter.SetValue("")
				m.applyFilter()
				m.recalcTableHeight()
				return m, nil
			}
		}
	}

	if !m.filtering {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ListModel) recalcTableHeight() {
	filterHeight := 0
	if m.filtering || m.filter.Value() != "" {
		filterHeight = 2
	}
	m.table.SetHeight(m.height - 6 - filterHeight)
}

func (m *ListModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))

	if query == "" {
		m.filteredPRs = m.prs
	} else {
		m.filteredPRs = nil
		for _, pr := range m.prs {
			if m.matchesPR(pr, query) {
				m.filteredPRs = append(m.filteredPRs, pr)
			}
		}
	}

	m.table.SetRows(m.buildRows(m.filteredPRs))

	// Reset cursor if it's out of bounds
	if m.table.Cursor() >= len(m.filteredPRs) && len(m.filteredPRs) > 0 {
		m.table.SetCursor(0)
	}
}

func (m ListModel) matchesPR(pr api.PullRequest, query string) bool {
	// Split query into terms for AND matching
	terms := strings.Fields(query)

	for _, term := range terms {
		found := false

		// Check each field
		if strings.Contains(strings.ToLower(pr.Title), term) {
			found = true
		} else if strings.Contains(strings.ToLower(pr.Repository.Name), term) {
			found = true
		} else if strings.Contains(strings.ToLower(pr.CreatedBy.DisplayName), term) {
			found = true
		} else if strings.Contains(strings.ToLower(pr.Status), term) {
			found = true
		} else if pr.IsDraft && strings.Contains("draft", term) {
			found = true
		} else if strings.Contains(fmt.Sprintf("%d", pr.PullRequestID), term) {
			found = true
		} else {
			// Check reviewer names
			for _, r := range pr.Reviewers {
				if strings.Contains(strings.ToLower(r.DisplayName), term) {
					found = true
					break
				}
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func (m *ListModel) recalcColumns() {
	available := m.width - 4 // Some padding
	if available < 80 {
		available = 80
	}

	// Proportional column widths
	idW := 6
	statusW := 10
	repoW := max(14, available/7)
	authorW := max(14, available/7)
	reviewW := max(16, available/5)
	titleW := max(20, available-idW-statusW-repoW-authorW-reviewW-5)

	m.table.SetColumns([]table.Column{
		{Title: "#", Width: idW},
		{Title: "Title", Width: titleW},
		{Title: "Repository", Width: repoW},
		{Title: "Author", Width: authorW},
		{Title: "Status", Width: statusW},
		{Title: "Reviewers", Width: reviewW},
	})
}

func (m ListModel) buildRows(prs []api.PullRequest) []table.Row {
	rows := make([]table.Row, len(prs))
	for i, pr := range prs {
		status := pr.Status
		if pr.IsDraft {
			status = "draft"
		}

		reviewers := m.formatReviewers(pr.Reviewers)

		rows[i] = table.Row{
			fmt.Sprintf("%d", pr.PullRequestID),
			truncate(pr.Title, 50),
			pr.Repository.Name,
			pr.CreatedBy.DisplayName,
			status,
			reviewers,
		}
	}
	return rows
}

func (m ListModel) formatReviewers(reviewers []api.Reviewer) string {
	if len(reviewers) == 0 {
		return "-"
	}

	parts := make([]string, 0, len(reviewers))
	for _, r := range reviewers {
		icon := r.VoteIcon()
		name := shortName(r.DisplayName)
		parts = append(parts, fmt.Sprintf("%s%s", icon, name))
	}
	return strings.Join(parts, " ")
}

func (m ListModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading pull requests...\n", m.spinner.View())
	}

	if m.err != nil {
		return theme.ErrorText.Render(fmt.Sprintf("\n  Error: %s\n\n  Press 'r' to retry", m.err))
	}

	if len(m.prs) == 0 {
		return "\n  No active pull requests found.\n\n  Press 'r' to refresh"
	}

	var sections []string

	// Filter bar (shown when filtering or when there's an active filter)
	if m.filtering || m.filter.Value() != "" {
		filterLine := m.filter.View()
		countText := theme.FilterCount.Render(
			fmt.Sprintf("  %d/%d", len(m.filteredPRs), len(m.prs)),
		)
		bar := theme.FilterBar.Render(filterLine + countText)
		sections = append(sections, bar)
	}

	// Table
	sections = append(sections, m.table.View())

	// Hint when no results match
	if len(m.filteredPRs) == 0 && m.filter.Value() != "" {
		sections = append(sections, theme.FilterCount.Render(
			fmt.Sprintf("\n  No PRs match \"%s\" — press esc to clear", m.filter.Value()),
		))
	}

	return strings.Join(sections, "\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func shortName(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	// First name + last initial
	return fmt.Sprintf("%s %s.", parts[0], string([]rune(parts[len(parts)-1])[0:1]))
}
