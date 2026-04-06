package prs

import (
	"fmt"
	"strings"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// --- Create PR messages ---

type PRCreatedMsg struct {
	PR api.PullRequest
}

type PRCreateErrorMsg struct {
	Err error
}

// branchesLoadedMsg carries fetched branches back to the create form.
type branchesLoadedMsg struct {
	branches []string
}

type branchesErrorMsg struct {
	err error
}

// createStep identifies the current step in the PR creation form.
type createStep int

const (
	stepRepo createStep = iota
	stepSourceBranch
	stepTargetBranch
	stepTitle
	stepDescription
	stepDraft
)

// branchPicker is a filterable, scrollable inline branch selector.
type branchPicker struct {
	all      []string
	filtered []string
	cursor   int
	filter   textinput.Model
}

func newBranchPicker() branchPicker {
	fi := textinput.New()
	fi.Prompt = "/ "
	fi.Placeholder = "filter branches..."
	fi.CharLimit = 100
	s := fi.Styles()
	s.Focused.Prompt = theme.FilterPrompt
	s.Blurred.Prompt = theme.FilterPrompt
	s.Focused.Text = theme.FilterText
	fi.SetStyles(s)
	return branchPicker{filter: fi}
}

func (p *branchPicker) setAll(branches []string) {
	p.all = branches
	p.filtered = make([]string, len(branches))
	copy(p.filtered, branches)
	p.cursor = 0
}

func (p *branchPicker) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(p.filter.Value()))
	if q == "" {
		p.filtered = make([]string, len(p.all))
		copy(p.filtered, p.all)
	} else {
		p.filtered = p.filtered[:0]
		for _, b := range p.all {
			if strings.Contains(strings.ToLower(b), q) {
				p.filtered = append(p.filtered, b)
			}
		}
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = len(p.filtered) - 1
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *branchPicker) selected() string {
	if len(p.filtered) == 0 {
		return ""
	}
	return p.filtered[p.cursor]
}

func (p *branchPicker) update(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if p.cursor < len(p.filtered)-1 {
			p.cursor++
		}
		return nil
	case "k", "up":
		if p.cursor > 0 {
			p.cursor--
		}
		return nil
	default:
		var cmd tea.Cmd
		p.filter, cmd = p.filter.Update(msg)
		p.applyFilter()
		return cmd
	}
}

func (p *branchPicker) focus() tea.Cmd {
	return p.filter.Focus()
}

func (p *branchPicker) blur() {
	p.filter.Blur()
}

func (p *branchPicker) view(maxVisible int) string {
	var b strings.Builder
	b.WriteString("  " + p.filter.View() + "\n")

	if len(p.filtered) == 0 {
		b.WriteString(theme.HelpDesc.Render("  (no matches)") + "\n")
		return b.String()
	}

	start := p.cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > len(p.filtered) {
		start = len(p.filtered) - maxVisible
		if start < 0 {
			start = 0
		}
	}
	end := start + maxVisible
	if end > len(p.filtered) {
		end = len(p.filtered)
	}

	for i := start; i < end; i++ {
		name := p.filtered[i]
		if i == p.cursor {
			b.WriteString("  " + theme.ActiveTab.Render(" "+name+" ") + "\n")
		} else {
			b.WriteString("  " + theme.InactiveTab.Render(" "+name+" ") + "\n")
		}
	}

	if len(p.filtered) < len(p.all) {
		b.WriteString(theme.HelpDesc.Render(fmt.Sprintf("  %d/%d · j/k move", len(p.filtered), len(p.all))) + "\n")
	} else {
		b.WriteString(theme.HelpDesc.Render(fmt.Sprintf("  %d branches · j/k move", len(p.all))) + "\n")
	}

	return b.String()
}

// CreatePRModel is a multi-step PR creation form.
type CreatePRModel struct {
	client          *api.Client
	repos           []string
	step            createStep
	repoCursor      int
	repoChosen      string
	branchesLoading bool
	branchesErr     error
	srcPicker       branchPicker
	tgtPicker       branchPicker
	srcChosen       string
	tgtChosen       string
	titleInput      textinput.Model
	descArea        textarea.Model
	isDraft         bool
	submitting      bool
	flashMsg        string
	flashIsErr      bool
	width           int
	height          int
}

func NewCreatePRModel(client *api.Client, repos []string) CreatePRModel {
	titleInput := textinput.New()
	titleInput.Prompt = "Title: "
	titleInput.Placeholder = "PR title"
	titleInput.CharLimit = 200
	s := titleInput.Styles()
	s.Focused.Prompt = theme.FilterPrompt
	s.Blurred.Prompt = theme.FilterPrompt
	s.Focused.Text = theme.FilterText
	titleInput.SetStyles(s)

	descArea := textarea.New()
	descArea.Placeholder = "Optional description... (ctrl+s to skip/submit)"
	descArea.ShowLineNumbers = false
	descArea.SetHeight(5)
	descArea.CharLimit = 4000

	return CreatePRModel{
		client:     client,
		repos:      repos,
		step:       stepRepo,
		srcPicker:  newBranchPicker(),
		tgtPicker:  newBranchPicker(),
		titleInput: titleInput,
		descArea:   descArea,
	}
}

func (m CreatePRModel) Init() tea.Cmd { return nil }

func (m CreatePRModel) IsActive() bool { return true }

func (m CreatePRModel) fetchBranchesCmd() tea.Cmd {
	client := m.client
	repo := m.repoChosen
	return func() tea.Msg {
		branches, err := client.ListBranches(repo)
		if err != nil {
			return branchesErrorMsg{err: err}
		}
		names := make([]string, len(branches))
		for i, b := range branches {
			names[i] = b.ShortName()
		}
		return branchesLoadedMsg{branches: names}
	}
}

func (m CreatePRModel) Update(msg tea.Msg) (CreatePRModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case branchesLoadedMsg:
		m.branchesLoading = false
		m.srcPicker.setAll(msg.branches)
		m.tgtPicker.setAll(msg.branches)
		m.step = stepSourceBranch
		cmd := m.srcPicker.focus()
		return m, cmd

	case branchesErrorMsg:
		m.branchesLoading = false
		m.branchesErr = msg.err
		return m, nil

	case PRCreateErrorMsg:
		m.submitting = false
		m.flashMsg = fmt.Sprintf("Error: %s", msg.Err)
		m.flashIsErr = true
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.step > stepRepo {
				m.step--
				// If stepping back to repo from source, clear branch state
				if m.step == stepRepo {
					m.branchesErr = nil
				}
				cmd := m.focusCurrent()
				return m, cmd
			}
			return m, func() tea.Msg { return CreatePRCancelMsg{} }

		case "ctrl+s":
			return m.trySubmit()

		case "enter", "tab":
			if m.step == stepDescription || m.step == stepDraft {
				return m.trySubmit()
			}
			cmd := m.advanceStep()
			return m, cmd

		default:
			return m.updateActiveInput(msg)
		}
	}

	return m.updateActiveInputMsg(msg)
}

func (m *CreatePRModel) advanceStep() tea.Cmd {
	switch m.step {
	case stepRepo:
		if len(m.repos) == 0 {
			m.flashMsg = "No repos configured — press esc and open the repo picker (R)"
			m.flashIsErr = true
			return nil
		}
		m.repoChosen = m.repos[m.repoCursor]
		m.flashMsg = ""
		m.branchesLoading = true
		m.branchesErr = nil
		return m.fetchBranchesCmd()

	case stepSourceBranch:
		chosen := m.srcPicker.selected()
		if chosen == "" {
			m.flashMsg = "Select a source branch"
			m.flashIsErr = true
			return nil
		}
		m.srcChosen = chosen
		m.srcPicker.blur()
		m.flashMsg = ""
		m.step = stepTargetBranch
		return m.tgtPicker.focus()

	case stepTargetBranch:
		chosen := m.tgtPicker.selected()
		if chosen == "" {
			m.flashMsg = "Select a target branch"
			m.flashIsErr = true
			return nil
		}
		m.tgtChosen = chosen
		m.tgtPicker.blur()
		m.flashMsg = ""
		m.step = stepTitle
		return m.titleInput.Focus()

	case stepTitle:
		if strings.TrimSpace(m.titleInput.Value()) == "" {
			m.flashMsg = "Title is required"
			m.flashIsErr = true
			return nil
		}
		m.flashMsg = ""
		m.step = stepDescription
		return m.descArea.Focus()

	case stepDescription:
		m.flashMsg = ""
		m.step = stepDraft
		return nil
	}
	return nil
}

func (m *CreatePRModel) focusCurrent() tea.Cmd {
	m.srcPicker.blur()
	m.tgtPicker.blur()
	m.titleInput.Blur()
	m.descArea.Blur()

	switch m.step {
	case stepSourceBranch:
		return m.srcPicker.focus()
	case stepTargetBranch:
		return m.tgtPicker.focus()
	case stepTitle:
		return m.titleInput.Focus()
	case stepDescription:
		return m.descArea.Focus()
	}
	return nil
}

func (m CreatePRModel) trySubmit() (CreatePRModel, tea.Cmd) {
	if m.repoChosen == "" || m.srcChosen == "" || m.tgtChosen == "" ||
		strings.TrimSpace(m.titleInput.Value()) == "" {
		m.flashMsg = "Please fill in all required fields"
		m.flashIsErr = true
		return m, nil
	}

	m.submitting = true
	m.flashMsg = ""

	client := m.client
	repoName := m.repoChosen
	title := strings.TrimSpace(m.titleInput.Value())
	src := "refs/heads/" + m.srcChosen
	tgt := "refs/heads/" + m.tgtChosen
	desc := strings.TrimSpace(m.descArea.Value())
	isDraft := m.isDraft

	return m, func() tea.Msg {
		pr, err := client.CreatePullRequest(repoName, title, src, tgt, desc, isDraft)
		if err != nil {
			return PRCreateErrorMsg{Err: err}
		}
		return PRCreatedMsg{PR: pr}
	}
}

func (m CreatePRModel) updateActiveInput(msg tea.KeyPressMsg) (CreatePRModel, tea.Cmd) {
	switch m.step {
	case stepRepo:
		switch msg.String() {
		case "j", "down":
			if m.repoCursor < len(m.repos)-1 {
				m.repoCursor++
			}
		case "k", "up":
			if m.repoCursor > 0 {
				m.repoCursor--
			}
		}
		return m, nil

	case stepSourceBranch:
		cmd := m.srcPicker.update(msg)
		return m, cmd

	case stepTargetBranch:
		cmd := m.tgtPicker.update(msg)
		return m, cmd

	case stepTitle:
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd

	case stepDescription:
		var cmd tea.Cmd
		m.descArea, cmd = m.descArea.Update(msg)
		return m, cmd

	case stepDraft:
		switch msg.String() {
		case "y", "Y":
			m.isDraft = true
			return m.trySubmit()
		case "n", "N":
			m.isDraft = false
			return m.trySubmit()
		case "space", " ":
			m.isDraft = !m.isDraft
		}
	}
	return m, nil
}

func (m CreatePRModel) updateActiveInputMsg(msg tea.Msg) (CreatePRModel, tea.Cmd) {
	var cmd tea.Cmd
	switch m.step {
	case stepSourceBranch:
		m.srcPicker.filter, cmd = m.srcPicker.filter.Update(msg)
	case stepTargetBranch:
		m.tgtPicker.filter, cmd = m.tgtPicker.filter.Update(msg)
	case stepTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case stepDescription:
		m.descArea, cmd = m.descArea.Update(msg)
	}
	return m, cmd
}

func (m CreatePRModel) View() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(theme.Title.Render("  Create Pull Request"))
	b.WriteString("\n\n")

	stepLabel := func(s createStep, label string) string {
		if m.step == s {
			return theme.Label.Render("▶ " + label)
		}
		if s < m.step {
			return theme.HelpDesc.Render("  ✓ " + label)
		}
		return theme.HelpDesc.Render("  · " + label)
	}

	branchVisible := m.height - 18
	if branchVisible < 4 {
		branchVisible = 4
	}
	if branchVisible > 15 {
		branchVisible = 15
	}

	// --- Repo ---
	b.WriteString(stepLabel(stepRepo, "Repository") + "\n")
	if m.step == stepRepo {
		if len(m.repos) == 0 {
			b.WriteString(theme.ErrorText.Render("  No repos configured. Press esc and open the repo picker (R).") + "\n")
		} else {
			maxVis := branchVisible
			start := m.repoCursor - maxVis/2
			if start < 0 {
				start = 0
			}
			if start+maxVis > len(m.repos) {
				start = len(m.repos) - maxVis
				if start < 0 {
					start = 0
				}
			}
			end := start + maxVis
			if end > len(m.repos) {
				end = len(m.repos)
			}
			for i := start; i < end; i++ {
				r := m.repos[i]
				if i == m.repoCursor {
					b.WriteString("  " + theme.ActiveTab.Render(" "+r+" ") + "\n")
				} else {
					b.WriteString("  " + theme.InactiveTab.Render(" "+r+" ") + "\n")
				}
			}
			b.WriteString(theme.HelpDesc.Render("  j/k move · enter select") + "\n")
		}
	} else {
		b.WriteString(theme.HelpDesc.Render("    "+m.repoChosen) + "\n")
	}
	b.WriteString("\n")

	// Loading / error state blocks the rest of the form
	if m.branchesLoading {
		b.WriteString(theme.Spinner.Render("  Loading branches...") + "\n")
		return b.String()
	}
	if m.branchesErr != nil && m.step > stepRepo {
		b.WriteString(theme.ErrorText.Render(fmt.Sprintf("  Error loading branches: %s", m.branchesErr)) + "\n")
		b.WriteString(theme.HelpDesc.Render("  Press esc to go back") + "\n")
		return b.String()
	}

	// --- Source branch ---
	b.WriteString(stepLabel(stepSourceBranch, "Source branch") + "\n")
	if m.step == stepSourceBranch {
		b.WriteString(m.srcPicker.view(branchVisible))
	} else if m.step > stepSourceBranch {
		b.WriteString(theme.HelpDesc.Render("    "+m.srcChosen) + "\n")
	}
	b.WriteString("\n")

	// --- Target branch ---
	b.WriteString(stepLabel(stepTargetBranch, "Target branch") + "\n")
	if m.step == stepTargetBranch {
		b.WriteString(m.tgtPicker.view(branchVisible))
	} else if m.step > stepTargetBranch {
		b.WriteString(theme.HelpDesc.Render("    "+m.tgtChosen) + "\n")
	}
	b.WriteString("\n")

	// --- Title ---
	b.WriteString(stepLabel(stepTitle, "Title") + "\n")
	if m.step == stepTitle {
		b.WriteString("  " + m.titleInput.View() + "\n")
	} else if m.step > stepTitle {
		b.WriteString(theme.HelpDesc.Render("    "+m.titleInput.Value()) + "\n")
	}
	b.WriteString("\n")

	// --- Description ---
	b.WriteString(stepLabel(stepDescription, "Description (optional)") + "\n")
	if m.step == stepDescription {
		b.WriteString(m.descArea.View() + "\n")
	} else if m.step > stepDescription {
		desc := strings.TrimSpace(m.descArea.Value())
		if desc == "" {
			desc = "(none)"
		}
		b.WriteString(theme.HelpDesc.Render("    "+desc) + "\n")
	}
	b.WriteString("\n")

	// --- Draft ---
	b.WriteString(stepLabel(stepDraft, "Draft?") + "\n")
	if m.step == stepDraft {
		draftVal := "No"
		if m.isDraft {
			draftVal = "Yes"
		}
		b.WriteString(theme.Label.Render("  Draft: "+draftVal) + "\n")
		b.WriteString(theme.HelpDesc.Render("  y/n · space toggle · ctrl+s submit · esc back") + "\n")
	}
	b.WriteString("\n")

	// Flash / status
	if m.submitting {
		b.WriteString(theme.Spinner.Render("  Creating PR...") + "\n")
	} else if m.flashMsg != "" {
		if m.flashIsErr {
			b.WriteString(theme.ErrorText.Render("  "+m.flashMsg) + "\n")
		} else {
			b.WriteString(theme.SuccessText.Render("  "+m.flashMsg) + "\n")
		}
	}

	switch m.step {
	case stepSourceBranch, stepTargetBranch:
		b.WriteString(theme.HelpDesc.Render("  enter select · esc back") + "\n")
	case stepDraft, stepRepo:
		// hint already shown inline
	default:
		b.WriteString(theme.HelpDesc.Render("  enter advance · esc back · ctrl+s submit") + "\n")
	}

	return b.String()
}

// CreatePRCancelMsg is emitted when the user cancels PR creation.
type CreatePRCancelMsg struct{}
