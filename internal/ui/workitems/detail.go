package workitems

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// --- Messages ---

type BackToWorkItemListMsg struct{}

type WorkItemRefreshedMsg struct {
	Item api.WorkItem
}

type WorkItemRefreshErrorMsg struct {
	Err error
}

type WorkItemCommentsLoadedMsg struct {
	Comments []api.WorkItemComment
}

type WorkItemCommentsErrorMsg struct {
	Err error
}

type StateUpdatedMsg struct{}
type StateUpdateErrorMsg struct{ Err error }

type StatesLoadedMsg struct{ States []string }
type StatesLoadErrorMsg struct{ Err error }

type CommentPostedMsg struct{}
type CommentErrorMsg struct{ Err error }

type PRLinkedMsg struct{ PRID string }
type PRLinkErrorMsg struct{ Err error }

type wiFlashClearMsg struct{}

// --- Detail modes ---

type wiMode int

const (
	wiModeNormal      wiMode = iota
	wiModeStateSelect        // pick new state
	wiModeComment            // compose comment
	wiModeLinkPR             // enter PR ID
)

// --- DetailModel ---

type DetailModel struct {
	client  *api.Client
	item    api.WorkItem
	orgURL  string
	project string

	comments []api.WorkItemComment

	viewport viewport.Model
	spinner  spinner.Model
	textarea textarea.Model
	prInput  textinput.Model

	// State picker
	availableStates []string
	stateIndex      int

	mode            wiMode
	loading         bool
	statesLoading   bool
	commentsLoading bool
	err             error
	ready           bool
	width           int
	height          int
	flashMsg        string
	flashStyle      lipgloss.Style
}

func NewDetailModel(client *api.Client, item api.WorkItem, orgURL, project string) DetailModel {
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

	pi := textinput.New()
	pi.Prompt = "PR ID: "
	pi.Placeholder = "1234"
	pi.CharLimit = 10
	piStyles := pi.Styles()
	piStyles.Focused.Prompt = theme.FilterPrompt
	piStyles.Blurred.Prompt = theme.FilterPrompt
	piStyles.Focused.Text = theme.FilterText
	pi.SetStyles(piStyles)

	states := api.WorkItemStates[item.Fields.WorkItemType]
	if len(states) == 0 {
		states = []string{"New", "Active", "Closed"}
	}
	stateIdx := 0
	for i, st := range states {
		if st == item.Fields.State {
			stateIdx = i
			break
		}
	}

	return DetailModel{
		client:          client,
		item:            item,
		orgURL:          orgURL,
		project:         project,
		spinner:         s,
		textarea:        ta,
		prInput:         pi,
		availableStates: states,
		stateIndex:      stateIdx,
		loading:         true,
		statesLoading:   true,
	}
}

func (m DetailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchComments(), m.fetchStates())
}

// IsInSubMode returns true when the detail view owns all key input
// (state picker, comment compose, or link-PR input active).
func (m DetailModel) IsInSubMode() bool {
	return m.mode != wiModeNormal
}

func (m DetailModel) fetchStates() tea.Cmd {
	workItemType := m.item.Fields.WorkItemType
	return func() tea.Msg {
		states, err := m.client.GetWorkItemTypeStates(workItemType)
		if err != nil {
			return StatesLoadErrorMsg{Err: err}
		}
		return StatesLoadedMsg{States: states}
	}
}

func (m DetailModel) fetchItem() tea.Cmd {
	id := m.item.ID
	return func() tea.Msg {
		wi, err := m.client.GetWorkItem(id)
		if err != nil {
			return WorkItemRefreshErrorMsg{Err: err}
		}
		return WorkItemRefreshedMsg{Item: wi}
	}
}

func (m DetailModel) fetchComments() tea.Cmd {
	id := m.item.ID
	return func() tea.Msg {
		comments, err := m.client.GetWorkItemComments(id)
		if err != nil {
			return WorkItemCommentsErrorMsg{Err: err}
		}
		return WorkItemCommentsLoadedMsg{Comments: comments}
	}
}

func (m DetailModel) submitStateUpdate() tea.Cmd {
	id := m.item.ID
	state := m.availableStates[m.stateIndex]
	return func() tea.Msg {
		err := m.client.UpdateWorkItemState(id, state)
		if err != nil {
			return StateUpdateErrorMsg{Err: err}
		}
		return StateUpdatedMsg{}
	}
}

func (m DetailModel) submitComment() tea.Cmd {
	id := m.item.ID
	text := m.textarea.Value()
	return func() tea.Msg {
		err := m.client.AddWorkItemComment(id, text)
		if err != nil {
			return CommentErrorMsg{Err: err}
		}
		return CommentPostedMsg{}
	}
}

func (m DetailModel) submitLinkPR(prIDStr string) tea.Cmd {
	// Build a minimal artifact URL — we don't have the project GUID readily,
	// so use a placeholder format; the actual linking requires a full vstfs URL.
	// For MVP we construct the URL using the work item's project ID context.
	id := m.item.ID
	orgURL := m.orgURL
	project := m.project
	return func() tea.Msg {
		// Try to parse PR ID
		prID := strings.TrimSpace(prIDStr)
		if prID == "" {
			return PRLinkErrorMsg{Err: fmt.Errorf("PR ID cannot be empty")}
		}
		// ADO artifact URL: we need the project GUID and repo GUID which we
		// don't have here; this is a best-effort using the project name.
		// Actual format: vstfs:///Git/PullRequestId/{projectID}/{repoID}/{prID}
		// We'll use the simpler approach of constructing a hyperlink-style URL.
		_ = orgURL
		// Best-effort MVP: use project name in place of project GUID and omit
		// the repo GUID (use a zeroed placeholder). ADO commonly accepts the
		// project name here; the proper fix would require passing the repo GUID.
		artifactURL := fmt.Sprintf("vstfs:///Git/PullRequestId/%s/%s",
			project, prID)
		err := m.client.LinkWorkItemToPR(id, artifactURL)
		if err != nil {
			return PRLinkErrorMsg{Err: err}
		}
		return PRLinkedMsg{PRID: prID}
	}
}

func (m DetailModel) setFlash(msg string, style lipgloss.Style) (DetailModel, tea.Cmd) {
	m.flashMsg = msg
	m.flashStyle = style
	return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return wiFlashClearMsg{}
	})
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
		if m.ready {
			m.viewport.SetContent(m.renderContent())
		}

	case WorkItemCommentsLoadedMsg:
		m.loading = false
		m.comments = msg.Comments
		m.resizeViewport()
		if m.ready {
			m.viewport.SetContent(m.renderContent())
		}

	case WorkItemCommentsErrorMsg:
		m.loading = false
		m.err = msg.Err
		m.resizeViewport()

	case WorkItemRefreshedMsg:
		m.item = msg.Item
		// Re-align stateIndex to the refreshed item state (states already loaded)
		stateIdx := 0
		for i, st := range m.availableStates {
			if st == m.item.Fields.State {
				stateIdx = i
				break
			}
		}
		m.stateIndex = stateIdx
		if m.ready {
			m.viewport.SetContent(m.renderContent())
		}

	case WorkItemRefreshErrorMsg:
		// Non-fatal

	case StatesLoadedMsg:
		m.statesLoading = false
		m.availableStates = msg.States
		// Re-align stateIndex to the current item state
		m.stateIndex = 0
		for i, st := range m.availableStates {
			if st == m.item.Fields.State {
				m.stateIndex = i
				break
			}
		}

	case StatesLoadErrorMsg:
		m.statesLoading = false
		// Fall back to the hardcoded map; already set in NewDetailModel so no-op

	case StateUpdatedMsg:
		m.mode = wiModeNormal
		newState := m.availableStates[m.stateIndex]
		m.item.Fields.State = newState
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("State updated to %s", newState), theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchItem())

	case StateUpdateErrorMsg:
		m.mode = wiModeNormal
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("State update failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case CommentPostedMsg:
		m.mode = wiModeNormal
		m.textarea.Reset()
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash("Comment posted!", theme.SuccessText)
		cmds = append(cmds, flashCmd, m.fetchComments())
		m.resizeViewport()

	case CommentErrorMsg:
		m.mode = wiModeNormal
		m.textarea.Reset()
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("Comment failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)
		m.resizeViewport()

	case PRLinkedMsg:
		m.mode = wiModeNormal
		m.prInput.Reset()
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("Linked to PR #%s", msg.PRID), theme.SuccessText)
		cmds = append(cmds, flashCmd)

	case PRLinkErrorMsg:
		m.mode = wiModeNormal
		m.prInput.Reset()
		var flashCmd tea.Cmd
		m, flashCmd = m.setFlash(fmt.Sprintf("Link failed: %s", msg.Err), theme.ErrorText)
		cmds = append(cmds, flashCmd)

	case wiFlashClearMsg:
		m.flashMsg = ""

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		switch m.mode {
		case wiModeNormal:
			cmd := m.handleNormalKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case wiModeStateSelect:
			cmd := m.handleStateSelectKeys(msg)
			return m, cmd
		case wiModeComment:
			cmd := m.handleCommentKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		case wiModeLinkPR:
			cmd := m.handleLinkPRKeys(msg)
			if cmd != nil {
				return m, cmd
			}
		}
	}

	// Update sub-models
	switch m.mode {
	case wiModeComment:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	case wiModeLinkPR:
		var cmd tea.Cmd
		m.prInput, cmd = m.prInput.Update(msg)
		cmds = append(cmds, cmd)
	case wiModeStateSelect:
		// All keys are handled by handleStateSelectKeys above; don't forward to viewport.
	default:
		if m.ready && !m.loading {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *DetailModel) handleNormalKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		return func() tea.Msg { return BackToWorkItemListMsg{} }
	case "r":
		m.loading = true
		m.err = nil
		return tea.Batch(m.spinner.Tick, m.fetchItem(), m.fetchComments())
	case "s":
		m.mode = wiModeStateSelect
		m.resizeViewport()
		return nil
	case "c":
		m.mode = wiModeComment
		m.textarea.Reset()
		return m.textarea.Focus()
	case "L":
		m.mode = wiModeLinkPR
		m.prInput.Reset()
		return m.prInput.Focus()
	case "o":
		return m.openInBrowser()
	}
	return nil
}

func (m *DetailModel) handleStateSelectKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = wiModeNormal
		m.resizeViewport()
		return nil
	case "up", "k":
		if m.stateIndex > 0 {
			m.stateIndex--
		}
		return nil
	case "down", "j":
		if m.stateIndex < len(m.availableStates)-1 {
			m.stateIndex++
		}
		return nil
	case "enter":
		m.mode = wiModeNormal
		m.resizeViewport()
		return m.submitStateUpdate()
	}
	return nil
}

func (m *DetailModel) handleCommentKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = wiModeNormal
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
		return m.submitComment()
	}
	return nil
}

func (m *DetailModel) handleLinkPRKeys(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = wiModeNormal
		m.prInput.Reset()
		m.prInput.Blur()
		return nil
	case "enter":
		prID := strings.TrimSpace(m.prInput.Value())
		m.prInput.Blur()
		return m.submitLinkPR(prID)
	}
	return nil
}

func (m DetailModel) openInBrowser() tea.Cmd {
	orgURL := m.orgURL
	project := m.project
	id := m.item.ID
	return func() tea.Msg {
		url := fmt.Sprintf("%s/%s/_workitems/edit/%d",
			strings.TrimRight(orgURL, "/"),
			project,
			id,
		)
		var cmd string
		switch runtime.GOOS {
		case "darwin":
			cmd = "open"
		default:
			cmd = "xdg-open"
		}
		_ = exec.Command(cmd, url).Start()
		return nil
	}
}

func (m *DetailModel) resizeViewport() {
	headerH := 2
	footerH := 2
	modeH := 0
	switch m.mode {
	case wiModeComment:
		modeH = 9
	case wiModeStateSelect:
		modeH = len(m.availableStates) + 2
	case wiModeLinkPR:
		modeH = 3
	}
	vpH := m.height - headerH - footerH - modeH
	if vpH < 5 {
		vpH = 5
	}
	if !m.ready {
		m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(vpH))
		m.ready = true
	} else {
		m.viewport.SetWidth(m.width)
		m.viewport.SetHeight(vpH)
	}
}

func (m DetailModel) renderContent() string {
	var b strings.Builder
	wrapW := m.width - 6
	if wrapW < 40 {
		wrapW = 40
	}

	// Title
	b.WriteString(theme.Title.Render(wiTypeIcon(m.item.Fields.WorkItemType) + " " + fmt.Sprintf("#%d: %s", m.item.ID, m.item.Fields.Title)))
	b.WriteString("\n\n")

	// Metadata
	b.WriteString(theme.Label.Render("Type: "))
	b.WriteString(m.item.Fields.WorkItemType)
	b.WriteString("  ")
	b.WriteString(theme.Label.Render("State: "))
	b.WriteString(theme.WorkItemStateStyle(m.item.Fields.State).Render(m.item.Fields.State))
	b.WriteString("\n")
	b.WriteString(theme.Label.Render("Assigned To: "))
	if m.item.Fields.AssignedTo.DisplayName != "" {
		b.WriteString(m.item.Fields.AssignedTo.DisplayName)
	} else {
		b.WriteString("(unassigned)")
	}
	b.WriteString("\n")
	b.WriteString(theme.Label.Render("Area Path: "))
	b.WriteString(m.item.Fields.AreaPath)
	b.WriteString("\n")
	b.WriteString(theme.Label.Render("Created: "))
	b.WriteString(m.item.Fields.CreatedDate.Format("2006-01-02 15:04"))
	b.WriteString("  ")
	b.WriteString(theme.Label.Render("Updated: "))
	b.WriteString(m.item.Fields.ChangedDate.Format("2006-01-02 15:04"))
	b.WriteString("\n")

	// Description
	b.WriteString("\n")
	b.WriteString(theme.SectionHeader.Render("Description"))
	b.WriteString("\n")
	desc := stripHTML(strings.TrimSpace(m.item.Fields.Description))
	if desc == "" {
		desc = "(no description)"
	}
	b.WriteString(wordWrapWI(desc, wrapW))
	b.WriteString("\n")

	// Comments
	b.WriteString("\n")
	b.WriteString(theme.SectionHeader.Render("Comments"))
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(theme.ErrorText.Render(fmt.Sprintf("  Error loading comments: %s", m.err)))
		b.WriteString("\n")
	} else if len(m.comments) == 0 {
		b.WriteString("  No comments\n")
	} else {
		for _, c := range m.comments {
			b.WriteString(fmt.Sprintf("%s  %s\n",
				theme.CommentAuthor.Render(c.CreatedBy.DisplayName),
				theme.CommentDate.Render(c.CreatedDate.Format("2006-01-02 15:04")),
			))
			text := wordWrapWI(stripHTML(strings.TrimSpace(c.Text)), wrapW-4)
			for _, line := range strings.Split(text, "\n") {
				b.WriteString(theme.CommentBody.Render(line))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(theme.HelpDesc.Render("  s state · c comment · L link PR · o browser · r refresh · esc back"))
	b.WriteString("\n")

	return b.String()
}

func (m DetailModel) View() string {
	if m.loading && !m.ready {
		return fmt.Sprintf("\n  %s Loading work item...\n", m.spinner.View())
	}
	if !m.ready {
		return "\n  Initializing...\n"
	}

	return m.view()
}

func (m DetailModel) view() string {

	var sections []string
	sections = append(sections, m.viewport.View())

	if m.flashMsg != "" {
		sections = append(sections, m.flashStyle.Render("  "+m.flashMsg))
	}

	switch m.mode {
	case wiModeStateSelect:
		var sb strings.Builder
		sb.WriteString(theme.ConfirmPrompt.Render("  Select state:") + "\n")
		// Show a scrolling window of up to 10 items centered on the cursor.
		const maxVisible = 10
		total := len(m.availableStates)
		start := m.stateIndex - maxVisible/2
		if start < 0 {
			start = 0
		}
		end := start + maxVisible
		if end > total {
			end = total
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
		if start > 0 {
			sb.WriteString(theme.HelpDesc.Render(fmt.Sprintf("    (%d more above)", start)) + "\n")
		}
		for i := start; i < end; i++ {
			st := m.availableStates[i]
			if i == m.stateIndex {
				sb.WriteString(theme.StatePickerSelected.Render("  > " + st))
			} else {
				sb.WriteString(theme.HelpDesc.Render("    " + st))
			}
			sb.WriteString("\n")
		}
		if end < total {
			sb.WriteString(theme.HelpDesc.Render(fmt.Sprintf("    (%d more below)", total-end)) + "\n")
		}
		sb.WriteString(theme.ConfirmHint.Render("  enter confirm · esc cancel"))
		sections = append(sections, sb.String())

	case wiModeComment:
		sections = append(sections, theme.ComposeLabel.Render("  Add comment:"))
		sections = append(sections, m.textarea.View())
		sections = append(sections, theme.ComposeHint.Render("  ctrl+s submit · esc cancel"))

	case wiModeLinkPR:
		sections = append(sections, theme.ComposeLabel.Render("  "+m.prInput.View()))
		sections = append(sections, theme.ComposeHint.Render("  enter confirm · esc cancel"))
	}

	return strings.Join(sections, "\n")
}

// stripHTML removes basic HTML tags from ADO rich text fields.
func stripHTML(s string) string {
	// Very simple tag stripper — adequate for MVP
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	// Normalize whitespace
	result := strings.ReplaceAll(out.String(), "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", `"`)
	return strings.TrimSpace(result)
}

func wordWrapWI(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for i, paragraph := range strings.Split(text, "\n") {
		if i > 0 {
			result.WriteString("\n")
		}
		result.WriteString(wrapLineWI(paragraph, width))
	}
	return result.String()
}

func wrapLineWI(line string, width int) string {
	if utf8.RuneCountInString(line) <= width {
		return line
	}
	var result strings.Builder
	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}
	lineLen := 0
	for i, word := range words {
		wordLen := utf8.RuneCountInString(word)
		if i == 0 {
			result.WriteString(word)
			lineLen = wordLen
			continue
		}
		if lineLen+1+wordLen > width {
			result.WriteString("\n")
			result.WriteString(word)
			lineLen = wordLen
		} else {
			result.WriteString(" ")
			result.WriteString(word)
			lineLen += 1 + wordLen
		}
	}
	return result.String()
}
