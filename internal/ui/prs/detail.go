package prs

import (
	"fmt"
	"strings"
	"time"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"
	"github.com/popplywop/azboard/internal/ui/uiutil"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// --- Detail mode state machine ---

type detailMode int

const (
	modeNormal      detailMode = iota
	modeCompose                // textarea focused for writing a comment
	modeConfirm                // vote confirmation prompt
	modeMerge                  // merge dialog
	modeAbandon                // abandon confirm
	modeDraftToggle            // draft/ready toggle confirm
	modeDiffComment            // inline diff comment compose
)

type detailPane int

const (
	paneOverview detailPane = iota
	paneFiles
	paneDiff
)

// --- Messages ---

type ThreadsLoadedMsg struct {
	Threads []api.Thread
}

type ThreadsErrorMsg struct {
	Err error
}

type BackToListMsg struct{}

type UserIDLoadedMsg struct {
	UserID string
}

type UserIDErrorMsg struct {
	Err error
}

type VoteSubmittedMsg struct{}

type VoteErrorMsg struct {
	Err error
}

type CommentPostedMsg struct{}

type CommentErrorMsg struct {
	Err error
}

type ThreadStatusUpdatedMsg struct{}

type ThreadStatusErrorMsg struct {
	Err error
}

type flashClearMsg struct{}

type PRRefreshedMsg struct {
	PR api.PullRequest
}

type PRRefreshErrorMsg struct {
	Err error
}

type FilesLoadedMsg struct {
	BaseCommit string
	HeadCommit string
	Changes    []api.IterationChange
	Iterations []api.Iteration
}

type FilesErrorMsg struct {
	Err error
}

// FilesChangesLoadedMsg is used when only the changed files list is refreshed
// (e.g. after switching iteration) without reloading the full iteration list.
type FilesChangesLoadedMsg struct {
	BaseCommit string
	HeadCommit string
	Changes    []api.IterationChange
}

type DiffLoadedMsg struct {
	Path string
	Diff string
}

type DiffErrorMsg struct {
	Err error
}

// PR lifecycle result messages
type PRMergedMsg struct {
	PRID int
}

type PRMergeErrorMsg struct {
	Err error
}

type PRAbandonedMsg struct {
	PRID int
}

type PRAbandonErrorMsg struct {
	Err error
}

type PRDraftToggledMsg struct {
	PRID    int
	IsDraft bool
}

type PRDraftToggleErrorMsg struct {
	Err error
}

// --- DetailModel ---

type DetailModel struct {
	client  api.Clienter
	pr      api.PullRequest
	threads []api.Thread

	// Context for building browser URLs
	orgURL  string
	project string

	// UI components
	viewport viewport.Model
	spinner  spinner.Model
	textarea textarea.Model
	files    FilesModel
	diff     DiffModel

	// State
	mode          detailMode
	loading       bool
	err           error
	ready         bool
	width         int
	height        int
	focusedThread int    // index into filtered threads, -1 = none
	replyThreadID int    // thread ID being replied to, -1 = new thread
	confirmAction string // human-readable action name
	confirmVote   int    // vote value to submit
	userID        string // current user's ID
	flashMsg      string // temporary success/error message
	flashStyle    lipgloss.Style
	pane          detailPane
	filesLoading  bool
	filesLoaded   bool
	diffLoading   bool
	filesErr      error
	diffErr       error
	baseCommitID  string
	headCommitID  string

	// Thread positions for scrolling (line number where each thread starts)
	threadPositions []int

	// Merge dialog state
	mergeDeleteBranch bool
	mergeOptions      []mergeOption
	mergeOptionIndex  int

	// Inline diff comment
	pendingDiffFile      string
	pendingDiffLine      int
	defaultMergeStrategy string
}

type mergeOption struct {
	Label    string
	APIValue string
}

func NewDetailModel(client api.Clienter, pr api.PullRequest, defaultMergeStrategy string) DetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.Spinner

	ta := textarea.New()
	ta.Placeholder = "Write your comment..."
	ta.ShowLineNumbers = false
	ta.SetHeight(5)
	styles := ta.Styles()
	styles.Focused.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(styles)
	ta.CharLimit = 4000

	options := []mergeOption{
		{Label: "Squash merge", APIValue: "squash"},
		{Label: "Merge commit", APIValue: "noFastForward"},
		{Label: "Rebase", APIValue: "rebase"},
		{Label: "Semi-linear (rebase merge)", APIValue: "rebaseMerge"},
	}

	// Find default strategy index
	optIdx := 0
	for i, o := range options {
		if o.APIValue == mergeStrategyAPIValue(defaultMergeStrategy) {
			optIdx = i
			break
		}
	}

	return DetailModel{
		client:               client,
		pr:                   pr,
		spinner:              s,
		textarea:             ta,
		loading:              true,
		focusedThread:        -1,
		replyThreadID:        -1,
		pane:                 paneOverview,
		files:                NewFilesModel(),
		diff:                 NewDiffModel(),
		mergeOptions:         options,
		mergeOptionIndex:     optIdx,
		mergeDeleteBranch:    true,
		defaultMergeStrategy: defaultMergeStrategy,
	}
}

// SetContext sets the orgURL and project fields used to build browser URLs.
func (m *DetailModel) SetContext(orgURL, project string) {
	m.orgURL = orgURL
	m.project = project
}

// mergeStrategyAPIValue converts a config string to the ADO API value.
func mergeStrategyAPIValue(s string) string {
	switch s {
	case "merge":
		return "noFastForward"
	case "rebase":
		return "rebase"
	case "semilinear":
		return "rebaseMerge"
	default:
		return "squash"
	}
}

func (m DetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchThreads(), m.fetchUserID())
}

// IsComposing returns true when the detail view is in compose or confirm mode.
func (m DetailModel) IsComposing() bool {
	return m.mode == modeCompose || m.mode == modeConfirm
}

func (m DetailModel) fetchThreads() tea.Cmd {
	pr := m.pr
	return func() tea.Msg {
		threads, err := m.client.GetPullRequestThreads(pr.Repository.ID, pr.PullRequestID)
		if err != nil {
			return ThreadsErrorMsg{Err: err}
		}
		return ThreadsLoadedMsg{Threads: threads}
	}
}

func (m DetailModel) fetchUserID() tea.Cmd {
	return func() tea.Msg {
		id, err := m.client.GetCurrentUserID()
		if err != nil {
			return UserIDErrorMsg{Err: err}
		}
		return UserIDLoadedMsg{UserID: id}
	}
}

func (m DetailModel) fetchPR() tea.Cmd {
	pr := m.pr
	return func() tea.Msg {
		updated, err := m.client.GetPullRequest(pr.Repository.ID, pr.PullRequestID)
		if err != nil {
			return PRRefreshErrorMsg{Err: err}
		}
		return PRRefreshedMsg{PR: *updated}
	}
}

func (m DetailModel) fetchFiles() tea.Cmd {
	pr := m.pr
	return func() tea.Msg {
		iterations, err := m.client.GetPullRequestIterations(pr.Repository.ID, pr.PullRequestID)
		if err != nil {
			return FilesErrorMsg{Err: err}
		}
		if len(iterations) == 0 {
			return FilesLoadedMsg{}
		}

		latest := iterations[len(iterations)-1]
		changes, err := m.client.GetPullRequestIterationChanges(pr.Repository.ID, pr.PullRequestID, latest.ID)
		if err != nil {
			return FilesErrorMsg{Err: err}
		}

		return FilesLoadedMsg{
			BaseCommit: latest.TargetRefCommit.CommitID,
			HeadCommit: latest.SourceRefCommit.CommitID,
			Changes:    changes,
			Iterations: iterations,
		}
	}
}

// fetchChangesForIteration re-fetches changed files for the given iteration
// without re-fetching the full iteration list.
func (m DetailModel) fetchChangesForIteration(iter api.Iteration) tea.Cmd {
	pr := m.pr
	return func() tea.Msg {
		changes, err := m.client.GetPullRequestIterationChanges(pr.Repository.ID, pr.PullRequestID, iter.ID)
		if err != nil {
			return FilesErrorMsg{Err: err}
		}
		return FilesChangesLoadedMsg{
			BaseCommit: iter.TargetRefCommit.CommitID,
			HeadCommit: iter.SourceRefCommit.CommitID,
			Changes:    changes,
		}
	}
}

func (m DetailModel) fetchDiff(ch api.IterationChange) tea.Cmd {
	pr := m.pr
	base := m.baseCommitID
	head := m.headCommitID
	return func() tea.Msg {
		d, err := m.client.BuildUnifiedDiff(pr.Repository.ID, ch, base, head)
		if err != nil {
			return DiffErrorMsg{Err: err}
		}
		return DiffLoadedMsg{Path: ch.Item.Path, Diff: d}
	}
}

func (m DetailModel) submitVote() tea.Cmd {
	pr := m.pr
	userID := m.userID
	vote := m.confirmVote
	return func() tea.Msg {
		err := m.client.SetVote(pr.Repository.ID, pr.PullRequestID, userID, vote)
		if err != nil {
			return VoteErrorMsg{Err: err}
		}
		return VoteSubmittedMsg{}
	}
}

func (m DetailModel) submitComment() tea.Cmd {
	pr := m.pr
	content := m.textarea.Value()
	threadID := m.replyThreadID
	return func() tea.Msg {
		var err error
		if threadID == -1 {
			err = m.client.CreateThread(pr.Repository.ID, pr.PullRequestID, content, nil)
		} else {
			err = m.client.ReplyToThread(pr.Repository.ID, pr.PullRequestID, threadID, content)
		}
		if err != nil {
			return CommentErrorMsg{Err: err}
		}
		return CommentPostedMsg{}
	}
}

func (m DetailModel) toggleThreadStatus(thread api.Thread) tea.Cmd {
	pr := m.pr
	newStatus := "fixed"
	if thread.Status == "fixed" || thread.Status == "closed" {
		newStatus = "active"
	}
	return func() tea.Msg {
		err := m.client.UpdateThreadStatus(pr.Repository.ID, pr.PullRequestID, thread.ID, newStatus)
		if err != nil {
			return ThreadStatusErrorMsg{Err: err}
		}
		return ThreadStatusUpdatedMsg{}
	}
}

func (m DetailModel) submitMerge() tea.Cmd {
	pr := m.pr
	strategy := m.mergeOptions[m.mergeOptionIndex].APIValue
	deleteBranch := m.mergeDeleteBranch
	return func() tea.Msg {
		err := m.client.MergePullRequest(pr.Repository.ID, pr.PullRequestID, strategy, "", deleteBranch)
		if err != nil {
			return PRMergeErrorMsg{Err: err}
		}
		return PRMergedMsg{PRID: pr.PullRequestID}
	}
}

func (m DetailModel) submitAbandon() tea.Cmd {
	pr := m.pr
	return func() tea.Msg {
		err := m.client.AbandonPullRequest(pr.Repository.ID, pr.PullRequestID)
		if err != nil {
			return PRAbandonErrorMsg{Err: err}
		}
		return PRAbandonedMsg{PRID: pr.PullRequestID}
	}
}

func (m DetailModel) submitDraftToggle() tea.Cmd {
	pr := m.pr
	newDraft := !pr.IsDraft
	return func() tea.Msg {
		err := m.client.ToggleDraft(pr.Repository.ID, pr.PullRequestID, newDraft)
		if err != nil {
			return PRDraftToggleErrorMsg{Err: err}
		}
		return PRDraftToggledMsg{PRID: pr.PullRequestID, IsDraft: newDraft}
	}
}

func (m DetailModel) openInBrowser() tea.Cmd {
	pr := m.pr
	orgURL := m.orgURL
	project := m.project
	return func() tea.Msg {
		url := fmt.Sprintf("%s/%s/_git/%s/pullrequest/%d",
			strings.TrimRight(orgURL, "/"),
			project,
			pr.Repository.Name,
			pr.PullRequestID,
		)
		uiutil.OpenBrowserURL(url)
		return nil
	}
}

func (m DetailModel) SetFlash(msg string, style lipgloss.Style) (DetailModel, tea.Cmd) {
	m.flashMsg = msg
	m.flashStyle = style
	return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return flashClearMsg{}
	})
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		m.files.SetSize(m.width, m.height-4)
		m.diff.SetSize(m.width, m.height-4)
		if !m.loading && m.ready {
			m.viewport.SetContent(m.renderContent())
		}

	case ThreadsLoadedMsg:
		m.loading = false
		m.threads = msg.Threads
		if m.ready {
			m.viewport.SetContent(m.renderContent())
			if m.focusedThread == -1 {
				m.viewport.GotoTop()
			}
		}

	case ThreadsErrorMsg:
		m.loading = false
		m.err = msg.Err
		if m.ready {
			m.viewport.SetContent(m.renderContent())
		}

	case UserIDLoadedMsg:
		m.userID = msg.UserID

	case UserIDErrorMsg:
		// Non-fatal — voting just won't work
		m.userID = ""

	case PRRefreshedMsg:
		m.pr = msg.PR
		if m.ready {
			m.viewport.SetContent(m.renderContent())
		}

	case PRRefreshErrorMsg:
		// Non-fatal, the stale data is still shown

	case FilesLoadedMsg:
		m.filesLoading = false
		m.filesLoaded = true
		m.filesErr = nil
		m.baseCommitID = msg.BaseCommit
		m.headCommitID = msg.HeadCommit
		m.files.SetChanges(msg.Changes)
		m.files.SetIterations(msg.Iterations)

	case FilesChangesLoadedMsg:
		m.filesLoading = false
		m.filesLoaded = true
		m.filesErr = nil
		m.baseCommitID = msg.BaseCommit
		m.headCommitID = msg.HeadCommit
		m.files.SetChanges(msg.Changes)

	case FilesErrorMsg:
		m.filesLoading = false
		m.filesLoaded = true
		m.filesErr = msg.Err

	case DiffLoadedMsg:
		m.diffLoading = false
		m.diffErr = nil
		m.diff.SetDiff(msg.Path, msg.Diff)
		m.pane = paneDiff

	case DiffErrorMsg:
		m.diffLoading = false
		m.diffErr = msg.Err

	case IterationChangedMsg:
		// User switched iteration — re-fetch changes and reset diff
		m.filesLoading = true
		m.filesErr = nil
		m.baseCommitID = msg.Iteration.TargetRefCommit.CommitID
		m.headCommitID = msg.Iteration.SourceRefCommit.CommitID
		m.pane = paneFiles
		return m, m.fetchChangesForIteration(msg.Iteration)

	case VoteSubmittedMsg:
		m.mode = modeNormal
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash("Vote submitted!", theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchPR(), m.fetchThreads())

	case VoteErrorMsg:
		m.mode = modeNormal
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash(fmt.Sprintf("Vote failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case CommentPostedMsg:
		m.mode = modeNormal
		m.textarea.Reset()
		m.pendingDiffFile = ""
		m.pendingDiffLine = 0
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash("Comment posted!", theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchThreads())
		m.resizeViewport()

	case CommentErrorMsg:
		m.mode = modeNormal
		m.textarea.Reset()
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash(fmt.Sprintf("Comment failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)
		m.resizeViewport()

	case ThreadStatusUpdatedMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash("Thread status updated!", theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchThreads())

	case ThreadStatusErrorMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash(fmt.Sprintf("Status update failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case PRMergedMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash("PR merged!", theme.SuccessText)
		cmds = append(cmds, flashCmd, func() tea.Msg { return BackToListMsg{} })

	case PRMergeErrorMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash(fmt.Sprintf("Merge failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case PRAbandonedMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash("PR abandoned.", theme.SuccessText)
		cmds = append(cmds, flashCmd, func() tea.Msg { return BackToListMsg{} })

	case PRAbandonErrorMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash(fmt.Sprintf("Abandon failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case PRDraftToggledMsg:
		m.pr.IsDraft = msg.IsDraft
		var label string
		if msg.IsDraft {
			label = "Converted to draft."
		} else {
			label = "Marked as ready for review."
		}
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash(label, theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchPR())

	case PRDraftToggleErrorMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.SetFlash(fmt.Sprintf("Draft toggle failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case flashClearMsg:
		m.flashMsg = ""

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		switch m.mode {
		case modeNormal:
			cmd := m.handleNormalKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case modeCompose:
			cmd := m.handleComposeKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case modeConfirm:
			cmd := m.handleConfirmKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case modeMerge:
			cmd := m.handleMergeKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case modeAbandon:
			cmd := m.handleAbandonKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case modeDraftToggle:
			cmd := m.handleDraftToggleKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case modeDiffComment:
			cmd := m.handleDiffCommentKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		}
	}

	// Update sub-components
	if m.mode == modeCompose || m.mode == modeDiffComment {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.pane == paneFiles {
		var cmd tea.Cmd
		m.files, cmd = m.files.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.pane == paneDiff {
		var cmd tea.Cmd
		m.diff, cmd = m.diff.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.ready && !m.loading {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *DetailModel) handleNormalKeys(msg tea.KeyPressMsg) tea.Cmd {
	if m.pane == paneFiles {
		switch msg.String() {
		case "esc":
			m.pane = paneOverview
			return nil
		case "r":
			m.filesLoading = true
			m.filesErr = nil
			m.filesLoaded = false
			return m.fetchFiles()
		case "enter":
			if m.files.IsOnDir() {
				m.files.toggleCollapse()
				return nil
			}
			ch, ok := m.files.SelectedChange()
			if !ok {
				return nil
			}
			m.diffLoading = true
			m.diffErr = nil
			return m.fetchDiff(ch)
		default:
			return nil
		}
	}

	if m.pane == paneDiff {
		switch msg.String() {
		case "esc":
			if m.diff.InCursorMode() {
				// esc in cursor mode exits cursor mode, handled by DiffModel.Update
				return nil
			}
			m.pane = paneFiles
			return nil
		case "c":
			if m.diff.InCursorMode() {
				lineNo := m.diff.CursorNewFileLine()
				if lineNo > 0 {
					m.pendingDiffFile = m.diff.Path()
					m.pendingDiffLine = lineNo
					m.mode = modeDiffComment
					m.textarea.Reset()
					cmd := m.textarea.Focus()
					m.resizeViewport()
					return cmd
				}
			}
			return nil
		default:
			return nil
		}
	}

	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		if m.focusedThread >= 0 {
			// First esc unfocuses thread
			m.focusedThread = -1
			m.viewport.SetContent(m.renderContent())
			return nil
		}
		return func() tea.Msg { return BackToListMsg{} }

	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		m.loading = true
		m.err = nil
		m.filesLoaded = false
		return tea.Batch(m.spinner.Tick, m.fetchPR(), m.fetchThreads())

	case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
		m.nextThread()
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("N"))):
		m.prevThread()
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		filtered := m.filterThreads()
		if m.focusedThread >= 0 && m.focusedThread < len(filtered) {
			m.replyThreadID = filtered[m.focusedThread].ID
			m.mode = modeCompose
			m.textarea.Reset()
			cmd := m.textarea.Focus()
			m.resizeViewport()
			return cmd
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("C"))):
		m.replyThreadID = -1
		m.mode = modeCompose
		m.textarea.Reset()
		cmd := m.textarea.Focus()
		m.resizeViewport()
		return cmd

	case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
		filtered := m.filterThreads()
		if m.focusedThread >= 0 && m.focusedThread < len(filtered) {
			return m.toggleThreadStatus(filtered[m.focusedThread])
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		return m.startVoteConfirm("Approve", 10)

	case key.Matches(msg, key.NewBinding(key.WithKeys("A"))):
		return m.startVoteConfirm("Approve with suggestions", 5)

	case key.Matches(msg, key.NewBinding(key.WithKeys("x"))):
		return m.startVoteConfirm("Reject", -10)

	case key.Matches(msg, key.NewBinding(key.WithKeys("w"))):
		return m.startVoteConfirm("Wait for author", -5)

	case key.Matches(msg, key.NewBinding(key.WithKeys("0"))):
		return m.startVoteConfirm("Reset vote", 0)

	case key.Matches(msg, key.NewBinding(key.WithKeys("f"))):
		m.pane = paneFiles
		if !m.filesLoaded && !m.filesLoading {
			m.filesLoading = true
			m.filesErr = nil
			return m.fetchFiles()
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("m"))):
		if m.pr.Status == "active" {
			m.mode = modeMerge
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("X"))):
		if m.pr.Status == "active" {
			m.mode = modeAbandon
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("D"))):
		if m.pr.Status == "active" {
			m.mode = modeDraftToggle
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("o"))):
		return m.openInBrowser()
	}

	return nil
}

func (m *DetailModel) startVoteConfirm(action string, vote int) tea.Cmd {
	if m.userID == "" {
		m.flashMsg = "Cannot vote: user ID not loaded (check auth)"
		m.flashStyle = theme.ErrorText
		return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return flashClearMsg{} })
	}
	m.mode = modeConfirm
	m.confirmAction = action
	m.confirmVote = vote
	return nil
}

func (m *DetailModel) handleComposeKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.textarea.Reset()
		m.textarea.Blur()
		m.resizeViewport()
		return nil
	case "ctrl+s":
		content := strings.TrimSpace(m.textarea.Value())
		if content == "" {
			return nil
		}
		m.textarea.Blur()
		cmd := m.submitComment()
		return cmd
	}
	return nil
}

func (m *DetailModel) handleConfirmKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		return m.submitVote()
	case "n", "N", "esc":
		m.mode = modeNormal
		return nil
	}
	return nil
}

func (m *DetailModel) handleMergeKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		return nil
	case "up", "k":
		if m.mergeOptionIndex > 0 {
			m.mergeOptionIndex--
		}
		return nil
	case "down", "j":
		if m.mergeOptionIndex < len(m.mergeOptions)-1 {
			m.mergeOptionIndex++
		}
		return nil
	case "d", "D":
		m.mergeDeleteBranch = !m.mergeDeleteBranch
		return nil
	case "enter", "ctrl+s":
		m.mode = modeNormal
		return m.submitMerge()
	}
	return nil
}

func (m *DetailModel) handleAbandonKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		return m.submitAbandon()
	case "n", "N", "esc":
		m.mode = modeNormal
		return nil
	}
	return nil
}

func (m *DetailModel) handleDraftToggleKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		return m.submitDraftToggle()
	case "n", "N", "esc":
		m.mode = modeNormal
		return nil
	}
	return nil
}

func (m *DetailModel) handleDiffCommentKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.textarea.Reset()
		m.textarea.Blur()
		m.pendingDiffFile = ""
		m.pendingDiffLine = 0
		m.resizeViewport()
		return nil
	case "ctrl+s":
		content := strings.TrimSpace(m.textarea.Value())
		if content == "" {
			return nil
		}
		m.textarea.Blur()
		return m.submitDiffComment(content)
	}
	return nil
}

func (m DetailModel) submitDiffComment(content string) tea.Cmd {
	pr := m.pr
	filePath := m.pendingDiffFile
	lineNo := m.pendingDiffLine
	return func() tea.Msg {
		ctx := &api.ThreadContext{
			FilePath:       filePath,
			RightFileStart: &api.LineRange{Line: lineNo, Offset: 1},
			RightFileEnd:   &api.LineRange{Line: lineNo, Offset: 1},
		}
		err := m.client.CreateThread(pr.Repository.ID, pr.PullRequestID, content, ctx)
		if err != nil {
			return CommentErrorMsg{Err: err}
		}
		return CommentPostedMsg{}
	}
}

func (m *DetailModel) nextThread() {
	filtered := m.filterThreads()
	if len(filtered) == 0 {
		return
	}
	m.focusedThread++
	if m.focusedThread >= len(filtered) {
		m.focusedThread = 0
	}
	m.viewport.SetContent(m.renderContent())
	m.scrollToFocusedThread()
}

func (m *DetailModel) prevThread() {
	filtered := m.filterThreads()
	if len(filtered) == 0 {
		return
	}
	m.focusedThread--
	if m.focusedThread < 0 {
		m.focusedThread = len(filtered) - 1
	}
	m.viewport.SetContent(m.renderContent())
	m.scrollToFocusedThread()
}

func (m *DetailModel) scrollToFocusedThread() {
	if m.focusedThread >= 0 && m.focusedThread < len(m.threadPositions) {
		targetLine := m.threadPositions[m.focusedThread]
		m.viewport.SetYOffset(targetLine)
	}
}

func (m *DetailModel) resizeViewport() {
	headerHeight := 2
	footerHeight := 2

	composeHeight := 0
	if m.mode == modeCompose || m.mode == modeDiffComment {
		composeHeight = 9 // label + textarea + hint
	} else if m.mode == modeConfirm || m.mode == modeAbandon || m.mode == modeDraftToggle {
		composeHeight = 2
	} else if m.mode == modeMerge {
		composeHeight = len(m.mergeOptions) + 4 // options + delete toggle + hint
	}

	vpHeight := m.height - headerHeight - footerHeight - composeHeight
	if vpHeight < 5 {
		vpHeight = 5
	}

	if !m.ready {
		m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(vpHeight))
		m.ready = true
	} else {
		m.viewport.SetWidth(m.width)
		m.viewport.SetHeight(vpHeight)
	}
}

func (m *DetailModel) renderContent() string {
	var b strings.Builder
	wrapWidth := m.width - 6
	if wrapWidth < 40 {
		wrapWidth = 40
	}

	// Title
	b.WriteString(theme.Title.Render(uiutil.WordWrap(fmt.Sprintf("PR #%d: %s", m.pr.PullRequestID, m.pr.Title), wrapWidth)))
	b.WriteString("\n\n")

	// Branch info
	b.WriteString(theme.Label.Render("Branch: "))
	b.WriteString(fmt.Sprintf("%s → %s", m.pr.SourceBranch(), m.pr.TargetBranch()))
	b.WriteString("\n")

	// Status
	status := m.pr.Status
	if m.pr.IsDraft {
		status = "draft"
	}
	b.WriteString(theme.Label.Render("Status: "))
	b.WriteString(theme.StatusStyle(m.pr.Status, m.pr.IsDraft).Render(status))
	b.WriteString("\n")

	// Author
	b.WriteString(theme.Label.Render("Author: "))
	b.WriteString(m.pr.CreatedBy.DisplayName)
	b.WriteString("\n")

	// Created
	b.WriteString(theme.Label.Render("Created: "))
	b.WriteString(m.pr.CreationDate.Format("2006-01-02 15:04"))
	b.WriteString("\n")

	// Reviewers
	b.WriteString("\n")
	b.WriteString(theme.SectionHeader.Render("Reviewers"))
	b.WriteString("\n")

	if len(m.pr.Reviewers) == 0 {
		b.WriteString("  No reviewers assigned\n")
	} else {
		for _, r := range m.pr.Reviewers {
			icon := r.VoteIcon()
			style := theme.VoteStyle(r.Vote)
			required := ""
			if r.IsRequired {
				required = " (required)"
			}
			b.WriteString(fmt.Sprintf("  %s %s — %s%s\n",
				style.Render(icon),
				r.DisplayName,
				style.Render(r.VoteString()),
				required,
			))
		}
	}

	// Description
	b.WriteString("\n")
	b.WriteString(theme.SectionHeader.Render("Description"))
	b.WriteString("\n")

	desc := strings.TrimSpace(m.pr.Description)
	if desc == "" {
		desc = "(no description)"
	}
	b.WriteString(uiutil.WordWrap(desc, wrapWidth))
	b.WriteString("\n")

	// Comment threads
	b.WriteString("\n")
	b.WriteString(theme.SectionHeader.Render("Comments"))
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(theme.ErrorText.Render(fmt.Sprintf("  Error loading threads: %s", m.err)))
		b.WriteString("\n")
	} else {
		visibleThreads := m.filterThreads()
		if len(visibleThreads) == 0 {
			b.WriteString("  No comments\n")
		} else {
			m.threadPositions = make([]int, len(visibleThreads))
			for i, thread := range visibleThreads {
				// Record the line position before rendering this thread
				m.threadPositions[i] = strings.Count(b.String(), "\n")
				isFocused := i == m.focusedThread
				m.renderThread(&b, thread, i+1, isFocused, wrapWidth)
			}
		}
	}

	// Keybinding hints at bottom
	b.WriteString("\n")
	hints := theme.HelpDesc.Render("  f files · n/N threads · c reply · C new comment · s resolve · a approve · x reject · ? help")
	b.WriteString(uiutil.WordWrap(hints, wrapWidth))
	b.WriteString("\n")

	return b.String()
}

func (m DetailModel) filterThreads() []api.Thread {
	var visible []api.Thread
	for _, t := range m.threads {
		if t.IsDeleted {
			continue
		}
		hasUserComment := false
		for _, c := range t.Comments {
			if !c.IsDeleted && c.CommentType != "system" {
				hasUserComment = true
				break
			}
		}
		if hasUserComment {
			visible = append(visible, t)
		}
	}
	return visible
}

func (m *DetailModel) renderThread(b *strings.Builder, thread api.Thread, num int, focused bool, wrapWidth int) {
	var threadBuf strings.Builder

	// Thread header
	statusStr := ""
	if thread.Status != "" && thread.Status != "unknown" {
		statusStr = fmt.Sprintf(" [%s]", thread.Status)
	}

	filePath := ""
	if thread.ThreadContext != nil && thread.ThreadContext.FilePath != "" {
		filePath = " " + theme.FilePath.Render(thread.ThreadContext.FilePath)
	}

	threadBuf.WriteString(fmt.Sprintf("── Thread %d%s%s ──\n",
		num,
		theme.ThreadStatus.Render(statusStr),
		filePath,
	))

	// Comments in thread
	contentWrap := wrapWidth - 4 // account for indent
	if contentWrap < 30 {
		contentWrap = 30
	}

	for _, comment := range thread.Comments {
		if comment.IsDeleted || comment.CommentType == "system" {
			continue
		}

		threadBuf.WriteString(fmt.Sprintf("%s  %s\n",
			theme.CommentAuthor.Render(comment.Author.DisplayName),
			theme.CommentDate.Render(comment.PublishedDate.Format("2006-01-02 15:04")),
		))

		content := strings.TrimSpace(comment.Content)
		wrapped := uiutil.WordWrap(content, contentWrap)
		for _, line := range strings.Split(wrapped, "\n") {
			threadBuf.WriteString(theme.CommentBody.Render(line))
			threadBuf.WriteString("\n")
		}
		threadBuf.WriteString("\n")
	}

	// Apply focused/unfocused styling to the whole thread block
	threadContent := threadBuf.String()
	if focused {
		b.WriteString("\n")
		// Apply the focused style line by line to get the left border
		for _, line := range strings.Split(strings.TrimRight(threadContent, "\n"), "\n") {
			b.WriteString(theme.FocusedThread.Render(line))
			b.WriteString("\n")
		}
	} else {
		b.WriteString("\n")
		for _, line := range strings.Split(strings.TrimRight(threadContent, "\n"), "\n") {
			b.WriteString(theme.UnfocusedThread.Render(line))
			b.WriteString("\n")
		}
	}
}

func (m DetailModel) View() string {
	if m.loading && !m.ready {
		return fmt.Sprintf("\n  %s Loading PR details...\n", m.spinner.View())
	}

	if !m.ready {
		return "\n  Initializing...\n"
	}

	var sections []string

	if m.pane == paneFiles {
		if m.filesLoading {
			sections = append(sections, fmt.Sprintf("\n  %s Loading changed files...\n", m.spinner.View()))
		} else if m.filesErr != nil {
			sections = append(sections, theme.ErrorText.Render(fmt.Sprintf("\n  Error loading files: %s\n", m.filesErr)))
		} else {
			sections = append(sections, m.files.View())
		}
	} else if m.pane == paneDiff {
		if m.diffLoading {
			sections = append(sections, fmt.Sprintf("\n  %s Loading diff...\n", m.spinner.View()))
		} else if m.diffErr != nil {
			sections = append(sections, theme.ErrorText.Render(fmt.Sprintf("\n  Error loading diff: %s\n", m.diffErr)))
		} else {
			sections = append(sections, m.diff.View())
		}
	} else {
		sections = append(sections, m.viewport.View())
	}

	// Flash message
	if m.flashMsg != "" {
		sections = append(sections, m.flashStyle.Render("  "+m.flashMsg))
	}

	// Compose area
	if m.mode == modeCompose {
		label := "New comment:"
		if m.replyThreadID != -1 {
			label = fmt.Sprintf("Replying to thread #%d:", m.replyThreadID)
		}
		sections = append(sections, theme.ComposeLabel.Render("  "+label))
		sections = append(sections, m.textarea.View())
		sections = append(sections, theme.ComposeHint.Render("  ctrl+s submit · esc cancel"))
	}

	// Confirm prompt (vote)
	if m.mode == modeConfirm {
		prompt := fmt.Sprintf("  %s this PR? (y/n)", m.confirmAction)
		sections = append(sections, theme.ConfirmPrompt.Render(prompt))
		sections = append(sections, theme.ConfirmHint.Render("  y confirm · n/esc cancel"))
	}

	// Merge dialog
	if m.mode == modeMerge {
		var mb strings.Builder
		mb.WriteString(theme.ConfirmPrompt.Render("  Merge strategy:") + "\n")
		for i, opt := range m.mergeOptions {
			cursor := "  "
			if i == m.mergeOptionIndex {
				cursor = "> "
			}
			row := fmt.Sprintf("  %s%s", cursor, opt.Label)
			if i == m.mergeOptionIndex {
				mb.WriteString(theme.MergeStrategyActive.Render(row))
			} else {
				mb.WriteString(theme.MergeStrategyInactive.Render(row))
			}
			mb.WriteString("\n")
		}
		deleteStr := "n"
		if m.mergeDeleteBranch {
			deleteStr = "Y"
		}
		mb.WriteString(theme.ConfirmPrompt.Render(fmt.Sprintf("  Delete source branch? [%s]  (d to toggle)", deleteStr)) + "\n")
		mb.WriteString(theme.ConfirmHint.Render("  enter/ctrl+s confirm · esc cancel"))
		sections = append(sections, mb.String())
	}

	// Abandon confirm
	if m.mode == modeAbandon {
		sections = append(sections, theme.ConfirmPrompt.Render("  Abandon this PR? [y/N]"))
		sections = append(sections, theme.ConfirmHint.Render("  y confirm · n/esc cancel"))
	}

	// Draft toggle confirm
	if m.mode == modeDraftToggle {
		var prompt string
		if m.pr.IsDraft {
			prompt = "  Mark as ready for review? [y/N]"
		} else {
			prompt = "  Convert to draft? [y/N]"
		}
		sections = append(sections, theme.ConfirmPrompt.Render(prompt))
		sections = append(sections, theme.ConfirmHint.Render("  y confirm · n/esc cancel"))
	}

	// Inline diff comment compose
	if m.mode == modeDiffComment {
		label := fmt.Sprintf("  Comment on %s line %d:", m.pendingDiffFile, m.pendingDiffLine)
		sections = append(sections, theme.ComposeLabel.Render(label))
		sections = append(sections, m.textarea.View())
		sections = append(sections, theme.ComposeHint.Render("  ctrl+s submit · esc cancel"))
	}

	return strings.Join(sections, "\n")
}
