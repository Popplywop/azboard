package repopicker

import (
	"fmt"
	"strings"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ReposLoadedMsg is sent when the repository list has been fetched.
type ReposLoadedMsg struct {
	Repos []api.GitRepository
}

// ReposLoadErrorMsg is sent when repository list fetch fails.
type ReposLoadErrorMsg struct {
	Err error
}

// RepoPickerDoneMsg is sent when the user confirms the selection.
type RepoPickerDoneMsg struct {
	Selected []string // repo names
	Save     bool     // true if user pressed ctrl+s (write to config)
}

// RepoPickerCancelMsg is sent when the user presses esc.
type RepoPickerCancelMsg struct{}

// PickerModel is the floating repo selection modal.
type PickerModel struct {
	repos    []api.GitRepository
	filtered []api.GitRepository
	selected map[string]bool // repo name -> selected
	cursor   int
	filter   textinput.Model
	loading  bool
	err      error
	width    int
	height   int
}

func NewPickerModel(initialSelected []string) PickerModel {
	fi := textinput.New()
	fi.Prompt = "Search: "
	fiStyles := fi.Styles()
	fiStyles.Focused.Prompt = theme.FilterPrompt
	fiStyles.Blurred.Prompt = theme.FilterPrompt
	fiStyles.Focused.Text = theme.FilterText
	fi.SetStyles(fiStyles)
	fi.Placeholder = "filter repositories..."
	fi.CharLimit = 100

	sel := make(map[string]bool)
	for _, r := range initialSelected {
		sel[r] = true
	}

	m := PickerModel{
		filter:   fi,
		selected: sel,
		loading:  true,
	}
	return m
}

func (m PickerModel) Init() tea.Cmd {
	return m.filter.Focus()
}

func (m PickerModel) applyFilter() []api.GitRepository {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		return m.repos
	}
	var out []api.GitRepository
	for _, r := range m.repos {
		if strings.Contains(strings.ToLower(r.Name), q) {
			out = append(out, r)
		}
	}
	return out
}

func (m PickerModel) Update(msg tea.Msg) (PickerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case ReposLoadedMsg:
		m.loading = false
		m.repos = msg.Repos
		m.filtered = m.applyFilter()
		m.cursor = 0

	case ReposLoadErrorMsg:
		m.loading = false
		m.err = msg.Err

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return RepoPickerCancelMsg{} }

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case "space", " ":
			// Toggle selection for item under cursor
			if m.cursor < len(m.filtered) {
				name := m.filtered[m.cursor].Name
				m.selected[name] = !m.selected[name]
			}
			return m, nil

		case "enter":
			return m, func() tea.Msg {
				return RepoPickerDoneMsg{Selected: m.selectedNames(), Save: false}
			}

		case "ctrl+s":
			return m, func() tea.Msg {
				return RepoPickerDoneMsg{Selected: m.selectedNames(), Save: true}
			}

		default:
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			cmds = append(cmds, cmd)
			m.filtered = m.applyFilter()
			// Clamp cursor
			if m.cursor >= len(m.filtered) && len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
			}
			return m, tea.Batch(cmds...)
		}
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m PickerModel) selectedNames() []string {
	var names []string
	for name, ok := range m.selected {
		if ok {
			names = append(names, name)
		}
	}
	return names
}

func (m PickerModel) View() string {
	// Box dimensions: ~60% of terminal width, ~60% height
	boxW := m.width * 60 / 100
	if boxW < 50 {
		boxW = 50
	}
	if boxW > m.width-4 {
		boxW = m.width - 4
	}
	boxH := m.height * 60 / 100
	if boxH < 10 {
		boxH = 10
	}

	inner := boxW - 4 // account for border + padding

	var lines []string

	// Title
	lines = append(lines, theme.Title.Render("Select Repositories"))
	lines = append(lines, "")

	// Filter input
	filterLine := m.filter.View()
	lines = append(lines, filterLine)
	lines = append(lines, "")

	if m.loading {
		lines = append(lines, "  Loading repositories...")
	} else if m.err != nil {
		lines = append(lines, theme.ErrorText.Render(fmt.Sprintf("  Error: %s", m.err)))
	} else if len(m.filtered) == 0 {
		lines = append(lines, theme.HelpDesc.Render("  No repositories match."))
	} else {
		// How many rows available for repo list
		listH := boxH - 6 // title + blank + filter + blank + hint + border
		if listH < 1 {
			listH = 1
		}

		// Scroll window
		start := 0
		if m.cursor >= listH {
			start = m.cursor - listH + 1
		}
		end := start + listH
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := start; i < end; i++ {
			repo := m.filtered[i]
			checkbox := "[ ]"
			if m.selected[repo.Name] {
				checkbox = "[x]"
			}
			line := fmt.Sprintf("%s %s", checkbox, repo.Name)
			if len(line) > inner {
				line = line[:inner]
			}
			if i == m.cursor {
				line = theme.ActiveTab.Render(line)
			} else {
				line = "  " + line
			}
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")
	lines = append(lines, theme.HelpDesc.Render("  space toggle · enter confirm · ctrl+s save & confirm · esc cancel"))

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2).
		Width(boxW)

	rendered := box.Render(content)

	// Center the box
	leftPad := (m.width - lipgloss.Width(rendered)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (m.height - lipgloss.Height(rendered)) / 2
	if topPad < 0 {
		topPad = 0
	}

	var finalLines []string
	for i := 0; i < topPad; i++ {
		finalLines = append(finalLines, "")
	}

	for _, line := range strings.Split(rendered, "\n") {
		finalLines = append(finalLines, strings.Repeat(" ", leftPad)+line)
	}

	return strings.Join(finalLines, "\n")
}
