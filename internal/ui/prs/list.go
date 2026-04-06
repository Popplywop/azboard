package prs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Messages
type PRsLoadedMsg struct {
	PRs          []api.PullRequest
	SkippedRepos []string // repos that returned 404
}

type PRsErrorMsg struct {
	Err error
}

// PRsSkippedReposMsg is emitted when one or more repos returned 404 during fetch.
// app.go handles this by flashing a warning to the user.
type PRsSkippedReposMsg struct {
	Repos []string
}

type SelectPRMsg struct {
	PR api.PullRequest
}

// OpenRepoPickerMsg tells AppModel to open the repo picker overlay.
type OpenRepoPickerMsg struct{}

// OpenCreatePRMsg tells AppModel to open the PR creation form.
type OpenCreatePRMsg struct{}

type prScope struct {
	Label     string
	APIStatus string
	DraftOnly bool
}

var listScopes = []prScope{
	{Label: "Active", APIStatus: "active"},
	{Label: "Draft", APIStatus: "active", DraftOnly: true},
	{Label: "Completed", APIStatus: "completed"},
	{Label: "Abandoned", APIStatus: "abandoned"},
	{Label: "All", APIStatus: "all"},
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
	scopeIndex  int
	repos       []string // selected repo names (empty = all projects)
}

func NewListModel(client *api.Client, repos []string) ListModel {
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
	fiStyles := fi.Styles()
	fiStyles.Focused.Prompt = theme.FilterPrompt
	fiStyles.Blurred.Prompt = theme.FilterPrompt
	fiStyles.Focused.Text = theme.FilterText
	fiStyles.Blurred.Text = theme.FilterText
	fi.SetStyles(fiStyles)
	fi.Placeholder = "type to filter by title, repo, author, status..."
	fi.CharLimit = 100

	loading := len(repos) > 0
	return ListModel{
		client:     client,
		table:      t,
		spinner:    s,
		filter:     fi,
		loading:    loading,
		scopeIndex: 0,
		repos:      repos,
	}
}

func (m ListModel) Init() tea.Cmd {
	if len(m.repos) == 0 {
		return nil
	}
	return tea.Batch(m.spinner.Tick, m.fetchPRs())
}

// IsFiltering returns true when the filter text input is focused.
func (m ListModel) IsFiltering() bool {
	return m.filtering
}

// SetRepos updates the repos and triggers a re-fetch.
func (m *ListModel) SetRepos(repos []string) {
	m.repos = repos
}

// is404 returns true when err is an ADO 404 (repo not found).
func is404(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "API returned 404")
}

func (m ListModel) fetchPRs() tea.Cmd {
	scope := m.currentScope()
	repos := m.repos
	return func() tea.Msg {
		if len(repos) == 0 {
			return PRsLoadedMsg{PRs: nil}
		}

		combined := make([]api.PullRequest, 0)
		seen := make(map[int]struct{})
		var skipped []string

		addPRs := func(prs []api.PullRequest) {
			for _, pr := range prs {
				if _, ok := seen[pr.PullRequestID]; ok {
					continue
				}
				seen[pr.PullRequestID] = struct{}{}
				combined = append(combined, pr)
			}
		}

		for _, repoName := range repos {
			if scope.DraftOnly {
				prs, err := m.client.ListDraftPullRequestsForRepo(repoName)
				if err != nil {
					if is404(err) {
						skipped = append(skipped, repoName)
						continue
					}
					return PRsErrorMsg{Err: err}
				}
				addPRs(prs)
				continue
			}

			statuses := []string{scope.APIStatus}
			if scope.APIStatus == "all" {
				statuses = []string{"active", "completed", "abandoned"}
			}

			repoSkipped := false
			for _, status := range statuses {
				prs, err := m.client.ListPullRequestsForRepo(repoName, status)
				if err != nil {
					if is404(err) {
						if !repoSkipped {
							skipped = append(skipped, repoName)
							repoSkipped = true
						}
						break
					}
					return PRsErrorMsg{Err: err}
				}
				addPRs(prs)
			}
		}

		sort.Slice(combined, func(i, j int) bool {
			return combined[i].CreationDate.After(combined[j].CreationDate)
		})
		return PRsLoadedMsg{PRs: combined, SkippedRepos: skipped}
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
		// Height breakdown: tab bar (2) + scope bar (2) + status bar (2) = -6;
		// -2 for the rounded border applied in View().
		m.table.SetHeight(m.height - 6 - filterHeight - 2)

	case PRsLoadedMsg:
		m.loading = false
		m.prs = msg.PRs
		m.applyFilter()
		if len(msg.SkippedRepos) > 0 {
			skipped := msg.SkippedRepos
			cmds = append(cmds, func() tea.Msg {
				return PRsSkippedReposMsg{Repos: skipped}
			})
		}

	case PRsErrorMsg:
		m.loading = false
		m.err = msg.Err

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
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
			cmd := m.filter.Focus()
			m.table.Blur()
			m.recalcTableHeight()
			return m, cmd

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if !m.loading && len(m.filteredPRs) > 0 {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.filteredPRs) {
					return m, func() tea.Msg {
						return SelectPRMsg{PR: m.filteredPRs[idx]}
					}
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("["))):
			m.cycleScope(-1)
			if len(m.repos) > 0 {
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.fetchPRs())
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("]"))):
			m.cycleScope(1)
			if len(m.repos) > 0 {
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.fetchPRs())
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
			if len(m.repos) > 0 {
				m.loading = true
				m.err = nil
				return m, tea.Batch(m.spinner.Tick, m.fetchPRs())
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("R"))):
			// Open repo picker
			return m, func() tea.Msg { return OpenRepoPickerMsg{} }

		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			// Open PR creation form
			return m, func() tea.Msg { return OpenCreatePRMsg{} }

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

// RefreshWithRepos updates repos selection and re-fetches PRs.
func (m ListModel) RefreshWithRepos(repos []string) (ListModel, tea.Cmd) {
	m.repos = repos
	m.prs = nil
	m.filteredPRs = nil
	m.err = nil
	if len(repos) == 0 {
		m.loading = false
		m.table.SetRows(nil)
		return m, nil
	}
	m.loading = true
	return m, tea.Batch(m.spinner.Tick, m.fetchPRs())
}

func (m *ListModel) recalcTableHeight() {
	filterHeight := 0
	if m.filtering || m.filter.Value() != "" {
		filterHeight = 2
	}
	// Height breakdown: tab bar (2) + scope bar (2) + status bar (2) = -6;
	// -2 for the rounded border applied in View().
	h := m.height - 6 - filterHeight - 2
	if h < 5 {
		h = 5
	}
	m.table.SetHeight(h)
}

func (m *ListModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.filter.Value()))

	if query == "" {
		m.filteredPRs = nil
		for _, pr := range m.prs {
			if m.matchesScope(pr) {
				m.filteredPRs = append(m.filteredPRs, pr)
			}
		}
	} else {
		m.filteredPRs = nil
		for _, pr := range m.prs {
			if m.matchesScope(pr) && m.matchesPR(pr, query) {
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

	// v2: must explicitly set viewport width or table renders empty.
	// Subtract 2 to account for the rounded border added in View().
	if m.width > 2 {
		m.table.SetWidth(m.width - 2)
	}
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
	// No repos selected — empty state
	if len(m.repos) == 0 {
		return "\n  No repositories selected.\n  Press R to open the repo picker, or set AZBOARD_REPOS in your config file.\n"
	}

	if m.loading {
		return fmt.Sprintf("\n  %s Loading pull requests...\n", m.spinner.View())
	}

	if m.err != nil {
		return theme.ErrorText.Render(fmt.Sprintf("\n  Error: %s\n\n  Press 'r' to retry", m.err))
	}

	if len(m.prs) == 0 {
		return fmt.Sprintf("\n  No %s pull requests found.\n\n  Press 'r' to refresh, 'n' to create a PR", strings.ToLower(m.currentScope().Label))
	}

	var sections []string

	// Scope bar
	sections = append(sections, m.renderScopeBar())

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
	sections = append(sections, theme.TableBorder.Render(m.table.View()))

	// Hint when no results match
	if len(m.filteredPRs) == 0 && m.filter.Value() != "" {
		sections = append(sections, theme.FilterCount.Render(
			fmt.Sprintf("\n  No PRs match \"%s\" — press esc to clear", m.filter.Value()),
		))
	}

	return strings.Join(sections, "\n")
}

func (m ListModel) renderScopeBar() string {
	parts := make([]string, 0, len(listScopes)+1)
	parts = append(parts, theme.HelpDesc.Render("  Scope:"))

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

func (m ListModel) currentScope() prScope {
	if m.scopeIndex < 0 || m.scopeIndex >= len(listScopes) {
		return listScopes[0]
	}
	return listScopes[m.scopeIndex]
}

func (m *ListModel) cycleScope(delta int) {
	n := len(listScopes)
	m.scopeIndex = (m.scopeIndex + delta + n) % n
}

func (m ListModel) matchesScope(pr api.PullRequest) bool {
	// When DraftOnly is set, the API already returned only draft PRs, so no
	// additional client-side filtering is needed. For all other scopes, every
	// PR in m.prs already matches the API-level status filter.
	_ = pr
	return true
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

// Ensure repopicker is imported (used by app.go indirectly)
