package prs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// hunkHeaderRe matches @@ -old[,count] +new[,count] @@ ...
var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

type DiffModel struct {
	viewport       viewport.Model
	ready          bool
	width          int
	height         int
	path           string
	rawDiff        string
	cursorMode     bool
	cursorLine     int   // 0-based index into the rendered lines
	lineCount      int   // total rendered lines
	lineNewNumbers []int // lineNewNumbers[i] = new-file line number for rendered line i (0 if not applicable)
}

const gutterWidth = 10 // "1234 1234│" — 4 + 1 + 4 + 1

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

	content, lineCount, lineNums := colorizeDiff(m.rawDiff, m.viewport.Width()-gutterWidth, m.cursorLine, m.cursorMode)
	m.lineCount = lineCount
	m.lineNewNumbers = lineNums
	m.viewport.SetContent(content)
}

func (m *DiffModel) SetDiff(path, diff string) {
	m.path = path
	m.rawDiff = diff
	m.cursorLine = 0
	m.cursorMode = false
	if m.ready {
		content, lineCount, lineNums := colorizeDiff(diff, m.viewport.Width()-gutterWidth, 0, false)
		m.lineCount = lineCount
		m.lineNewNumbers = lineNums
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
	}
}

// CursorLine returns the current cursor line (0-based) when in cursor mode.
func (m DiffModel) CursorLine() int { return m.cursorLine }

// InCursorMode reports whether cursor mode is active.
func (m DiffModel) InCursorMode() bool { return m.cursorMode }

// CursorNewFileLine returns the new-file line number at the current cursor position,
// or 0 if the cursor is on a line that has no new-side line number.
func (m DiffModel) CursorNewFileLine() int {
	if m.cursorLine < 0 || m.cursorLine >= len(m.lineNewNumbers) {
		return 0
	}
	return m.lineNewNumbers[m.cursorLine]
}

// Path returns the file path this diff is showing.
func (m DiffModel) Path() string { return m.path }

func (m DiffModel) Update(msg tea.Msg) (DiffModel, tea.Cmd) {
	if !m.ready {
		return m, nil
	}

	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "i":
			if !m.cursorMode {
				m.cursorMode = true
				m.refreshContent()
				return m, nil
			}
		case "esc":
			if m.cursorMode {
				m.cursorMode = false
				m.refreshContent()
				return m, nil
			}
		case "j", "down":
			if m.cursorMode {
				if m.cursorLine < m.lineCount-1 {
					m.cursorLine++
				}
				m.refreshContent()
				return m, nil
			}
		case "k", "up":
			if m.cursorMode {
				if m.cursorLine > 0 {
					m.cursorLine--
				}
				m.refreshContent()
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *DiffModel) refreshContent() {
	if !m.ready {
		return
	}
	content, lineCount, lineNums := colorizeDiff(m.rawDiff, m.viewport.Width()-gutterWidth, m.cursorLine, m.cursorMode)
	m.lineCount = lineCount
	m.lineNewNumbers = lineNums
	m.viewport.SetContent(content)
}

func (m DiffModel) View() string {
	if !m.ready || m.path == "" {
		return theme.HelpDesc.Render("  No diff loaded.")
	}
	hint := "  ↑/↓ scroll · i cursor mode · esc back"
	if m.cursorMode {
		hint = "  j/k move cursor · c comment · esc exit cursor mode"
	}
	return theme.SectionHeader.Render("Diff: "+m.path) + "\n" + m.viewport.View() + "\n" +
		theme.HelpDesc.Render(hint)
}

func colorizeDiff(diff string, wrapWidth int, cursorLine int, cursorMode bool) (string, int, []int) {
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	// First pass: build all rendered lines with gutter
	type renderedLine struct {
		gutter    string // pre-rendered gutter (already has ANSI)
		content   string // raw content text (no ANSI yet)
		diffType  string // "+", "-", "@@", "meta", ""
		newFileNo int    // new-file line number for this rendered line (0 = N/A)
	}

	var lines []renderedLine

	oldLine, newLine := 0, 0

	for _, rawLine := range strings.Split(diff, "\n") {
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")

		var diffType string
		var thisNewLine int
		switch {
		case strings.HasPrefix(rawLine, "@@"):
			diffType = "@@"
			// Parse hunk header to reset counters
			if m := hunkHeaderRe.FindStringSubmatch(rawLine); len(m) == 3 {
				if v, err := strconv.Atoi(m[1]); err == nil {
					oldLine = v
				}
				if v, err := strconv.Atoi(m[2]); err == nil {
					newLine = v
				}
			}
		case strings.HasPrefix(rawLine, "+++") || strings.HasPrefix(rawLine, "---"):
			diffType = "meta"
		case strings.HasPrefix(rawLine, "+"):
			diffType = "+"
			thisNewLine = newLine
		case strings.HasPrefix(rawLine, "-"):
			diffType = "-"
		default:
			diffType = ""
			thisNewLine = newLine
		}

		// Build gutter for this line
		var gutterOld, gutterNew string
		if diffType == "@@" || diffType == "meta" {
			gutterOld = "    "
			gutterNew = "    "
		} else if diffType == "+" {
			gutterOld = "    "
			gutterNew = fmt.Sprintf("%4d", newLine)
			newLine++
		} else if diffType == "-" {
			gutterOld = fmt.Sprintf("%4d", oldLine)
			gutterNew = "    "
			oldLine++
		} else {
			gutterOld = fmt.Sprintf("%4d", oldLine)
			gutterNew = fmt.Sprintf("%4d", newLine)
			oldLine++
			newLine++
		}
		gutter := theme.DiffGutter.Render(gutterOld+" "+gutterNew) + theme.DiffGutterSep.Render("│")

		// Wrap and emit lines
		wrapped := wrapRunes(rawLine, wrapWidth)
		for i, part := range wrapped {
			var g string
			if i == 0 {
				g = gutter
			} else {
				// continuation lines get blank gutter
				g = theme.DiffGutter.Render("         ") + theme.DiffGutterSep.Render("│")
			}
			lineNum := 0
			if i == 0 {
				lineNum = thisNewLine
			}
			lines = append(lines, renderedLine{gutter: g, content: part, diffType: diffType, newFileNo: lineNum})
		}
	}

	// Second pass: apply diff colors (and cursor highlight if in cursor mode)
	var b strings.Builder
	lineNums := make([]int, len(lines))
	for i, l := range lines {
		lineNums[i] = l.newFileNo
		var rendered string
		if cursorMode && i == cursorLine {
			// Highlight entire line with subtle background — color content, then join
			rendered = l.gutter + lipgloss.NewStyle().
				Background(theme.Subtle).
				Render(l.content)
		} else {
			switch l.diffType {
			case "@@":
				rendered = l.gutter + theme.DiffHunk.Render(l.content)
			case "meta":
				rendered = l.gutter + theme.DiffMeta.Render(l.content)
			case "+":
				rendered = l.gutter + theme.DiffAdd.Render(l.content)
			case "-":
				rendered = l.gutter + theme.DiffDelete.Render(l.content)
			default:
				rendered = l.gutter + l.content
			}
		}
		b.WriteString(rendered)
		b.WriteString("\n")
	}

	return b.String(), len(lines), lineNums
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
