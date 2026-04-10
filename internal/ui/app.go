package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/config"
	"github.com/popplywop/azboard/internal/ui/prs"
	"github.com/popplywop/azboard/internal/ui/repopicker"
	"github.com/popplywop/azboard/internal/ui/theme"
	"github.com/popplywop/azboard/internal/ui/workitems"
	"github.com/popplywop/azboard/internal/update"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type view int

const (
	viewList view = iota
	viewDetail
	viewCreatePR
	viewWorkItems
	viewWorkItemDetail
)

type tabID int

const (
	tabPRs       tabID = iota
	tabWorkItems       // Work Items tab
)

type AppModel struct {
	client               api.Clienter
	org                  string
	project              string
	orgURL               string
	repos                []string
	workItemTypes        []string
	defaultMergeStrategy string
	areaPath             string
	currentUserID        string
	jumpToPRID           int // if non-zero, fetch and open this PR on startup
	version              string
	autoRefreshInterval  time.Duration

	activeView view
	activeTab  tabID

	list        prs.ListModel
	detail      prs.DetailModel
	createPR    prs.CreatePRModel
	wiList      workitems.ListModel
	wiDetail    workitems.DetailModel
	wiListReady bool // lazy-init flag

	// Repo picker overlay
	picker       repopicker.PickerModel
	pickerActive bool

	width     int
	height    int
	showHelp  bool
	statusMsg string // transient status bar message (auto-clears)
}

func NewAppModel(
	client api.Clienter,
	org, project, orgURL string,
	repos, workItemTypes []string,
	defaultMergeStrategy, areaPath string,
	jumpToPRID int,
	version string,
	autoRefreshInterval time.Duration,
) AppModel {
	return AppModel{
		client:               client,
		org:                  org,
		project:              project,
		orgURL:               orgURL,
		repos:                repos,
		workItemTypes:        workItemTypes,
		defaultMergeStrategy: defaultMergeStrategy,
		areaPath:             areaPath,
		jumpToPRID:           jumpToPRID,
		version:              version,
		autoRefreshInterval:  autoRefreshInterval,
		activeView:           viewList,
		activeTab:            tabPRs,
		list:                 prs.NewListModel(client, repos),
	}
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.list.Init(), m.fetchUserID()}
	if m.jumpToPRID != 0 {
		cmds = append(cmds, m.fetchPRByIDCmd(m.jumpToPRID))
	}
	if m.version != "" && m.version != "dev" {
		cmds = append(cmds, m.checkForUpdate())
	}
	if m.autoRefreshInterval > 0 {
		cmds = append(cmds, tea.Tick(m.autoRefreshInterval, func(time.Time) tea.Msg {
			return autoRefreshTickMsg{}
		}))
	}
	return tea.Batch(cmds...)
}

type userIDLoadedMsg struct {
	id string
}

type appStatusClearMsg struct{}

type autoRefreshTickMsg struct{}

type updateAvailableMsg struct {
	latest string
}

type jumpToPRLoadedMsg struct {
	pr api.PullRequest
}

type jumpToPRErrorMsg struct {
	err error
}

func (m AppModel) fetchPRByIDCmd(prID int) tea.Cmd {
	return func() tea.Msg {
		pr, err := m.client.GetPullRequestByID(prID)
		if err != nil {
			return jumpToPRErrorMsg{err: err}
		}
		return jumpToPRLoadedMsg{pr: *pr}
	}
}

func (m AppModel) fetchUserID() tea.Cmd {
	return func() tea.Msg {
		id, _ := m.client.GetCurrentUserID()
		return userIDLoadedMsg{id: id}
	}
}

func (m AppModel) checkForUpdate() tea.Cmd {
	ver := m.version
	return func() tea.Msg {
		latest, hasUpdate := update.CheckLatestVersion(ver)
		if hasUpdate {
			return updateAvailableMsg{latest: latest}
		}
		return nil
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case userIDLoadedMsg:
		m.currentUserID = msg.id
		return m, nil

	case updateAvailableMsg:
		m.statusMsg = update.FormatUpdateNotice(msg.latest, m.version)
		return m, tea.Tick(10*time.Second, func(time.Time) tea.Msg { return appStatusClearMsg{} })

	case autoRefreshTickMsg:
		var refreshCmds []tea.Cmd
		// Re-schedule next tick
		refreshCmds = append(refreshCmds, tea.Tick(m.autoRefreshInterval, func(time.Time) tea.Msg {
			return autoRefreshTickMsg{}
		}))

		// Invalidate cache for active tab and silently re-fetch
		if inv, ok := m.client.(api.CacheInvalidator); ok {
			switch m.activeTab {
			case tabPRs:
				if m.activeView == viewList && !m.list.IsLoading() {
					inv.InvalidatePrefix("prs:")
					var cmd tea.Cmd
					m.list, cmd = m.list.SilentRefresh()
					refreshCmds = append(refreshCmds, cmd)
				}
			case tabWorkItems:
				if m.activeView == viewWorkItems && m.wiListReady && !m.wiList.IsLoading() {
					inv.InvalidatePrefix("wi:")
					var cmd tea.Cmd
					m.wiList, cmd = m.wiList.SilentRefresh()
					refreshCmds = append(refreshCmds, cmd)
				}
			}
		}
		return m, tea.Batch(refreshCmds...)

	case jumpToPRLoadedMsg:
		m.activeView = viewDetail
		m.detail = prs.NewDetailModel(m.client, msg.pr, m.defaultMergeStrategy)
		m.detail.SetContext(m.orgURL, m.project)
		cmd := m.detail.Init()
		var detailCmd tea.Cmd
		m.detail, detailCmd = m.detail.Update(tea.WindowSizeMsg{
			Width:  m.width,
			Height: m.height - 3,
		})
		return m, tea.Batch(cmd, detailCmd)

	case jumpToPRErrorMsg:
		m.statusMsg = fmt.Sprintf("Error loading PR: %s", msg.err)
		return m, tea.Tick(6*time.Second, func(time.Time) tea.Msg { return appStatusClearMsg{} })

	case appStatusClearMsg:
		m.statusMsg = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.pickerActive {
			var cmd tea.Cmd
			m.picker, cmd = m.picker.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		inner := tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - 3}
		switch m.activeView {
		case viewList:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(inner)
			cmds = append(cmds, cmd)
		case viewDetail:
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(inner)
			cmds = append(cmds, cmd)
		case viewWorkItems:
			var cmd tea.Cmd
			m.wiList, cmd = m.wiList.Update(inner)
			cmds = append(cmds, cmd)
		case viewWorkItemDetail:
			var cmd tea.Cmd
			m.wiDetail, cmd = m.wiDetail.Update(inner)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		// Don't intercept global keys when a sub-model owns input
		if m.pickerActive {
			break
		}
		if m.activeView == viewList && m.list.IsFiltering() {
			break
		}
		if m.activeView == viewDetail && m.detail.IsComposing() {
			break
		}
		if m.activeView == viewCreatePR {
			break
		}
		if m.activeView == viewWorkItemDetail && m.wiDetail.IsInSubMode() {
			break
		}

		// Dismiss help overlay with esc or q
		if m.showHelp && (key.Matches(msg, Keys.Back) || key.Matches(msg, Keys.Quit)) {
			m.showHelp = false
			return m, nil
		}

		switch {
		case key.Matches(msg, Keys.Quit):
			if m.activeView == viewDetail || m.activeView == viewCreatePR {
				m.activeView = viewList
				var cmd tea.Cmd
				m.list, cmd = m.list.RefreshWithRepos(m.repos)
				return m, cmd
			}
			if m.activeView == viewWorkItemDetail {
				m.activeView = viewWorkItems
				return m, nil
			}
			return m, tea.Quit

		case key.Matches(msg, Keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, Keys.Tab):
			return m.cycleTab()
		}

	// --- Repo picker messages ---
	// ReposLoadedMsg and ReposLoadErrorMsg are forwarded to the picker via
	// the picker delegation block below; no special handling needed here.

	case repopicker.RepoPickerDoneMsg:
		m.pickerActive = false
		m.repos = msg.Selected
		var cmd tea.Cmd
		m.list, cmd = m.list.RefreshWithRepos(msg.Selected)
		cmds = append(cmds, cmd)
		if msg.Save {
			// Write repos to config file and show confirmation
			cmds = append(cmds, saveReposCmd(msg.Selected))
			m.statusMsg = fmt.Sprintf("Repos saved to config (%d selected)", len(msg.Selected))
			cmds = append(cmds, tea.Tick(4*time.Second, func(time.Time) tea.Msg {
				return appStatusClearMsg{}
			}))
		}
		return m, tea.Batch(cmds...)

	case repopicker.RepoPickerCancelMsg:
		m.pickerActive = false
		return m, nil

	// --- PR list messages ---
	case prs.OpenRepoPickerMsg:
		m.picker = repopicker.NewPickerModel(m.repos)
		m.pickerActive = true
		cmd := m.picker.Init()
		// Size the picker
		var sizeCmd tea.Cmd
		m.picker, sizeCmd = m.picker.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		// Kick off the API fetch so the picker populates
		fetchCmd := m.listRepositoriesCmd()
		cmds = append(cmds, cmd, sizeCmd, fetchCmd)
		return m, tea.Batch(cmds...)

	case prs.OpenCreatePRMsg:
		m.createPR = prs.NewCreatePRModel(m.client, m.repos)
		m.activeView = viewCreatePR
		cmd := m.createPR.Init()
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case prs.SelectPRMsg:
		m.activeView = viewDetail
		m.detail = prs.NewDetailModel(m.client, msg.PR, m.defaultMergeStrategy)
		m.detail.SetContext(m.orgURL, m.project)
		cmd := m.detail.Init()
		var detailCmd tea.Cmd
		m.detail, detailCmd = m.detail.Update(tea.WindowSizeMsg{
			Width:  m.width,
			Height: m.height - 3,
		})
		cmds = append(cmds, cmd, detailCmd)
		return m, tea.Batch(cmds...)

	case prs.BackToListMsg:
		m.activeView = viewList
		// Refresh list after merge/abandon
		var cmd tea.Cmd
		m.list, cmd = m.list.RefreshWithRepos(m.repos)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case prs.PRsSkippedReposMsg:
		// One or more repos in the config weren't found in ADO — warn the user.
		m.statusMsg = fmt.Sprintf("Warning: repo(s) not found: %s", strings.Join(msg.Repos, ", "))
		cmds = append(cmds, tea.Tick(6*time.Second, func(time.Time) tea.Msg {
			return appStatusClearMsg{}
		}))
		return m, tea.Batch(cmds...)

	case prs.PRCreatedMsg:
		// Navigate to the new PR's detail view
		m.activeView = viewDetail
		m.detail = prs.NewDetailModel(m.client, msg.PR, m.defaultMergeStrategy)
		m.detail.SetContext(m.orgURL, m.project)
		var flashCmd tea.Cmd
		m.detail, flashCmd = m.detail.SetFlash("PR created!", theme.SuccessText)
		cmd := m.detail.Init()
		var detailCmd tea.Cmd
		m.detail, detailCmd = m.detail.Update(tea.WindowSizeMsg{
			Width:  m.width,
			Height: m.height - 3,
		})
		cmds = append(cmds, flashCmd, cmd, detailCmd)
		return m, tea.Batch(cmds...)

	case prs.CreatePRCancelMsg:
		m.activeView = viewList
		return m, nil

	// --- Work item messages ---
	case workitems.BackToWorkItemListMsg:
		m.activeView = viewWorkItems
		return m, nil

	case workitems.WorkItemSelectedMsg:
		m.activeView = viewWorkItemDetail
		m.wiDetail = workitems.NewDetailModel(m.client, msg.Item, m.orgURL, m.project)
		cmd := m.wiDetail.Init()
		var sizeCmd tea.Cmd
		m.wiDetail, sizeCmd = m.wiDetail.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height - 3})
		cmds = append(cmds, cmd, sizeCmd)
		return m, tea.Batch(cmds...)
	}

	// Delegate to active overlay / view
	if m.pickerActive {
		var cmd tea.Cmd
		m.picker, cmd = m.picker.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	switch m.activeView {
	case viewList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	case viewDetail:
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		cmds = append(cmds, cmd)
	case viewCreatePR:
		var cmd tea.Cmd
		m.createPR, cmd = m.createPR.Update(msg)
		cmds = append(cmds, cmd)
	case viewWorkItems:
		var cmd tea.Cmd
		m.wiList, cmd = m.wiList.Update(msg)
		cmds = append(cmds, cmd)
	case viewWorkItemDetail:
		var cmd tea.Cmd
		m.wiDetail, cmd = m.wiDetail.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) cycleTab() (tea.Model, tea.Cmd) {
	switch m.activeTab {
	case tabPRs:
		m.activeTab = tabWorkItems
		m.activeView = viewWorkItems
		if !m.wiListReady {
			m.wiListReady = true
			m.wiList = workitems.NewListModel(m.client, m.workItemTypes, m.currentUserID, m.areaPath)
			cmd := m.wiList.Init()
			var sizeCmd tea.Cmd
			m.wiList, sizeCmd = m.wiList.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height - 3})
			return m, tea.Batch(cmd, sizeCmd)
		}
	case tabWorkItems:
		m.activeTab = tabPRs
		m.activeView = viewList
	}
	return m, nil
}

func (m AppModel) View() tea.View {
	var full string

	if m.pickerActive {
		// The picker takes over the full screen — no overlay splicing needed.
		// Pass the available height minus tab bar and status bar so the picker
		// centres itself correctly.
		full = m.picker.View()
	} else {
		tabs := m.renderTabBar()

		var content string
		switch m.activeView {
		case viewList:
			content = m.list.View()
		case viewDetail:
			content = m.detail.View()
		case viewCreatePR:
			content = m.createPR.View()
		case viewWorkItems:
			content = m.wiList.View()
		case viewWorkItemDetail:
			content = m.wiDetail.View()
		}

		statusBar := m.renderStatusBar()

		if m.showHelp {
			content = m.renderHelp()
		}

		full = lipgloss.JoinVertical(lipgloss.Left, tabs, content, statusBar)
	}

	v := tea.NewView(full)
	v.AltScreen = true
	return v
}

func (m AppModel) renderTabBar() string {
	var tabs []string

	prActive := m.activeTab == tabPRs
	if prActive {
		tabs = append(tabs, theme.ActiveTab.Render("Pull Requests"))
	} else {
		tabs = append(tabs, theme.InactiveTab.Render("Pull Requests"))
	}

	wiActive := m.activeTab == tabWorkItems
	if wiActive {
		tabs = append(tabs, theme.ActiveTab.Render("Work Items"))
	} else {
		tabs = append(tabs, theme.InactiveTab.Render("Work Items"))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return theme.TabBar.Render(row)
}

func (m AppModel) renderStatusBar() string {
	left := theme.StatusBar.Render(fmt.Sprintf(" %s / %s", m.org, m.project))

	var right string
	if m.statusMsg != "" {
		right = theme.StatusBar.Render(theme.SuccessText.Render(" " + m.statusMsg + " "))
	} else {
		helpHint := theme.HelpKey.Render("?") + theme.HelpDesc.Render(" help")
		tabHint := theme.HelpKey.Render("tab") + theme.HelpDesc.Render(" switch tab")
		right = theme.StatusBar.Render(tabHint + "  " + helpHint)
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	spacer := lipgloss.NewStyle().Width(gap).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
}

func (m AppModel) renderHelp() string {
	listBindings := []struct{ key, desc string }{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"enter", "View PR details"},
		{"[ / ]", "Cycle scope"},
		{"/", "Filter PRs"},
		{"esc", "Clear filter / back"},
		{"r", "Refresh"},
		{"R", "Open repo picker"},
		{"n", "Create new PR"},
		{"tab", "Switch tab"},
		{"q", "Quit"},
		{"?", "Toggle help"},
	}

	detailBindings := []struct{ key, desc string }{
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
		{"m", "Merge PR"},
		{"X", "Abandon PR"},
		{"D", "Toggle draft/ready"},
		{"o", "Open in browser"},
		{"r", "Refresh"},
		{"esc", "Unfocus thread / back"},
		{"q", "Back to list"},
	}

	filesDiffBindings := []struct{ key, desc string }{
		{"enter", "View diff / toggle dir collapse"},
		{"r", "Refresh file list"},
		{"esc", "Back (diff → files, files → overview)"},
		{"↑/↓", "Navigate / scroll"},
		{"←/→", "Previous / next iteration"},
		{"i", "Enter cursor mode (diff)"},
		{"c", "Inline comment at cursor (cursor mode)"},
	}

	wiBindings := []struct{ key, desc string }{
		{"↑/k", "Move up"},
		{"↓/j", "Move down"},
		{"enter", "View work item"},
		{"[ / ]", "Cycle scope"},
		{"/", "Filter"},
		{"r", "Refresh"},
		{"esc", "Back"},
	}

	wiDetailBindings := []struct{ key, desc string }{
		{"s", "State transition"},
		{"c", "Add comment"},
		{"L", "Link to PR"},
		{"o", "Open in browser"},
		{"r", "Refresh"},
		{"esc", "Back to list"},
	}

	render := func(bindings []struct{ key, desc string }) []string {
		var lines []string
		for _, b := range bindings {
			line := fmt.Sprintf("  %s  %s",
				theme.HelpKey.Width(12).Render(b.key),
				theme.HelpDesc.Render(b.desc),
			)
			lines = append(lines, line)
		}
		return lines
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, theme.SectionHeader.Render("  PR List"))
	lines = append(lines, "")
	lines = append(lines, render(listBindings)...)

	lines = append(lines, "")
	lines = append(lines, theme.SectionHeader.Render("  PR Detail"))
	lines = append(lines, "")
	lines = append(lines, render(detailBindings)...)

	lines = append(lines, "")
	lines = append(lines, theme.SectionHeader.Render("  Files / Diff"))
	lines = append(lines, "")
	lines = append(lines, render(filesDiffBindings)...)

	lines = append(lines, "")
	lines = append(lines, theme.SectionHeader.Render("  Work Items"))
	lines = append(lines, "")
	lines = append(lines, render(wiBindings)...)

	lines = append(lines, "")
	lines = append(lines, theme.SectionHeader.Render("  Work Item Detail"))
	lines = append(lines, "")
	lines = append(lines, render(wiDetailBindings)...)

	lines = append(lines, "")
	lines = append(lines, theme.HelpDesc.Render("  Press ? to close"))

	return strings.Join(lines, "\n") + "\n"
}

// saveReposCmd persists the repo selection so it survives restarts.
func saveReposCmd(repos []string) tea.Cmd {
	return func() tea.Msg {
		_ = config.UpdateRepos(repos)
		return nil
	}
}

// listRepositoriesCmd fetches all repos from ADO and sends a ReposLoadedMsg to the picker.
func (m AppModel) listRepositoriesCmd() tea.Cmd {
	return func() tea.Msg {
		repos, err := m.client.ListRepositories()
		if err != nil {
			return repopicker.ReposLoadErrorMsg{Err: err}
		}
		return repopicker.ReposLoadedMsg{Repos: repos}
	}
}
