package prs

import (
	"strings"

	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type DiffModel struct {
	viewport viewport.Model
	ready    bool
	width    int
	height   int
	path     string
	rawDiff  string
}

func NewDiffModel() DiffModel {
	return DiffModel{}
}

func (m *DiffModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	vh := height - 3
	if vh < 5 {
		vh = 5
	}

	if !m.ready {
		m.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(vh))
		m.ready = true
	} else {
		m.viewport.SetWidth(width)
		m.viewport.SetHeight(vh)
	}

	m.viewport.SetContent(colorizeDiff(m.rawDiff, m.viewport.Width()))
}

func (m *DiffModel) SetDiff(path, diff string) {
	m.path = path
	m.rawDiff = diff
	if m.ready {
		m.viewport.SetContent(colorizeDiff(diff, m.viewport.Width()))
		m.viewport.GotoTop()
	}
}

func (m DiffModel) Update(msg tea.Msg) (DiffModel, tea.Cmd) {
	if !m.ready {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiffModel) View() string {
	if !m.ready {
		return "Loading diff..."
	}
	return theme.SectionHeader.Render("Diff: "+m.path) + "\n" + m.viewport.View() + "\n" +
		theme.HelpDesc.Render("  ↑/↓ scroll · esc back")
}

func colorizeDiff(diff string, width int) string {
	var b strings.Builder
	wrapWidth := width - 2
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	for _, line := range strings.Split(diff, "\n") {
		line = strings.ReplaceAll(line, "\t", "    ")
		wrapped := wrapRunes(line, wrapWidth)
		for _, part := range wrapped {
			switch {
			case strings.HasPrefix(part, "@@"):
				b.WriteString(theme.DiffHunk.Render(part))
			case strings.HasPrefix(part, "+++") || strings.HasPrefix(part, "---"):
				b.WriteString(theme.DiffMeta.Render(part))
			case strings.HasPrefix(part, "+"):
				b.WriteString(theme.DiffAdd.Render(part))
			case strings.HasPrefix(part, "-"):
				b.WriteString(theme.DiffDelete.Render(part))
			default:
				b.WriteString(part)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

func wrapRunes(line string, width int) []string {
	if width <= 0 {
		return []string{line}
	}
	r := []rune(line)
	if len(r) <= width {
		return []string{line}
	}
	parts := make([]string, 0, len(r)/width+1)
	for len(r) > width {
		parts = append(parts, string(r[:width]))
		r = r[width:]
	}
	parts = append(parts, string(r))
	return parts
}
