package prs

import (
	"fmt"
	"strings"
	"time"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Detail mode state machine ---

type detailMode int

const (
	modeNormal  detailMode = iota
	modeCompose            // textarea focused for writing a comment
	modeConfirm            // vote confirmation prompt
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

// --- DetailModel ---

type DetailModel struct {
	client  *api.Client
	pr      api.PullRequest
	threads []api.Thread

	// UI components
	viewport viewport.Model
	spinner  spinner.Model
	textarea textarea.Model

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

	// Thread positions for scrolling (line number where each thread starts)
	threadPositions []int
}

func NewDetailModel(client *api.Client, pr api.PullRequest) DetailModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = theme.Spinner

	ta := textarea.New()
	ta.Placeholder = "Write your comment..."
	ta.ShowLineNumbers = false
	ta.SetHeight(5)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.CharLimit = 4000

	return DetailModel{
		client:        client,
		pr:            pr,
		spinner:       s,
		textarea:      ta,
		loading:       true,
		focusedThread: -1,
		replyThreadID: -1,
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
			err = m.client.CreateThread(pr.Repository.ID, pr.PullRequestID, content)
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

func (m DetailModel) setFlash(msg string, style lipgloss.Style) (DetailModel, tea.Cmd) {
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

	case VoteSubmittedMsg:
		m.mode = modeNormal
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash("Vote submitted!", theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchPR(), m.fetchThreads())

	case VoteErrorMsg:
		m.mode = modeNormal
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("Vote failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case CommentPostedMsg:
		m.mode = modeNormal
		m.textarea.Reset()
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash("Comment posted!", theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchThreads())
		m.resizeViewport()

	case CommentErrorMsg:
		m.mode = modeNormal
		m.textarea.Reset()
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("Comment failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)
		m.resizeViewport()

	case ThreadStatusUpdatedMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash("Thread status updated!", theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchThreads())

	case ThreadStatusErrorMsg:
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("Status update failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case flashClearMsg:
		m.flashMsg = ""

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
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
		}
	}

	// Update sub-components
	if m.mode == modeCompose {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.ready && !m.loading {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *DetailModel) handleNormalKeys(msg tea.KeyMsg) tea.Cmd {
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
			m.textarea.Focus()
			m.resizeViewport()
			return m.textarea.Cursor.BlinkCmd()
		}
		return nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("C"))):
		m.replyThreadID = -1
		m.mode = modeCompose
		m.textarea.Reset()
		m.textarea.Focus()
		m.resizeViewport()
		return m.textarea.Cursor.BlinkCmd()

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

func (m *DetailModel) handleComposeKeys(msg tea.KeyMsg) tea.Cmd {
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

func (m *DetailModel) handleConfirmKeys(msg tea.KeyMsg) tea.Cmd {
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
	if m.mode == modeCompose {
		composeHeight = 9 // label + textarea + hint
	} else if m.mode == modeConfirm {
		composeHeight = 2
	}

	vpHeight := m.height - headerHeight - footerHeight - composeHeight
	if vpHeight < 5 {
		vpHeight = 5
	}

	if !m.ready {
		m.viewport = viewport.New(m.width, vpHeight)
		m.viewport.HighPerformanceRendering = false
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
	}
}

func (m *DetailModel) renderContent() string {
	var b strings.Builder
	wrapWidth := m.width - 6
	if wrapWidth < 40 {
		wrapWidth = 40
	}

	// Title
	b.WriteString(theme.Title.Render(wordWrap(fmt.Sprintf("PR #%d: %s", m.pr.PullRequestID, m.pr.Title), wrapWidth)))
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
	b.WriteString(wordWrap(desc, wrapWidth))
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
	hints := theme.HelpDesc.Render("  n/N threads · c reply · C new comment · s resolve · a approve · x reject · ? help")
	b.WriteString(wordWrap(hints, wrapWidth))
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
		wrapped := wordWrap(content, contentWrap)
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

	// Main viewport
	sections = append(sections, m.viewport.View())

	// Flash message
	if m.flashMsg != "" {
		sections = append(sections, m.flashStyle.Render("  "+m.flashMsg))
	}

	// Compose area
	if m.mode == modeCompose {
		label := "New comment:"
		if m.replyThreadID != -1 {
			label = fmt.Sprintf("Replying to thread:")
		}
		sections = append(sections, theme.ComposeLabel.Render("  "+label))
		sections = append(sections, m.textarea.View())
		sections = append(sections, theme.ComposeHint.Render("  ctrl+s submit · esc cancel"))
	}

	// Confirm prompt
	if m.mode == modeConfirm {
		prompt := fmt.Sprintf("  %s this PR? (y/n)", m.confirmAction)
		sections = append(sections, theme.ConfirmPrompt.Render(prompt))
	}

	return strings.Join(sections, "\n")
}

// wordWrap wraps text to the given width at word boundaries.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for i, paragraph := range strings.Split(text, "\n") {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(wrapLine(paragraph, width))
	}
	return result.String()
}

func wrapLine(line string, width int) string {
	if len(line) <= width {
		return line
	}

	var result strings.Builder
	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}

	lineLen := 0
	for i, word := range words {
		wordLen := len(word)

		if i == 0 {
			// Handle words longer than width
			if wordLen > width {
				for len(word) > width {
					result.WriteString(word[:width])
					result.WriteString("\n")
					word = word[width:]
				}
				result.WriteString(word)
				lineLen = len(word)
			} else {
				result.WriteString(word)
				lineLen = wordLen
			}
			continue
		}

		if lineLen+1+wordLen > width {
			result.WriteString("\n")
			// Handle words longer than width
			if wordLen > width {
				for len(word) > width {
					result.WriteString(word[:width])
					result.WriteString("\n")
					word = word[width:]
				}
				result.WriteString(word)
				lineLen = len(word)
			} else {
				result.WriteString(word)
				lineLen = wordLen
			}
		} else {
			result.WriteString(" ")
			result.WriteString(word)
			lineLen += 1 + wordLen
		}
	}

	return result.String()
}
