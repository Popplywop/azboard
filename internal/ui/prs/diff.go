package prs

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

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
	viewMode       diffViewMode
	cursorMode     bool
	cursorLine     int   // 0-based index into the rendered lines
	lineCount      int   // total rendered lines
	lineNewNumbers []int // lineNewNumbers[i] = new-file line number for rendered line i (0 if not applicable)
	hunkLines      []int // rendered line indexes where hunk headers are shown
	currentHunk    int
}

type diffViewMode int

const (
	diffViewInline diffViewMode = iota
	diffViewModified
	diffViewSplit
)

const (
	inlineGutterWidth   = 10 // "1234 1234│" — 4 + 1 + 4 + 1
	modifiedGutterWidth = 5  // "1234│"
)

func NewDiffModel() DiffModel {
	return DiffModel{viewMode: diffViewInline}
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

	content, lineCount, lineNums, hunkLines := colorizeDiff(m.rawDiff, m.viewport.Width(), m.cursorLine, m.cursorMode, m.viewMode)
	m.lineCount = lineCount
	m.lineNewNumbers = lineNums
	m.hunkLines = hunkLines
	m.currentHunk = clampHunkIndex(m.currentHunk, len(hunkLines))
	m.viewport.SetContent(content)
}

func (m *DiffModel) SetDiff(path, diff string) {
	m.path = path
	m.rawDiff = diff
	m.cursorLine = 0
	m.cursorMode = false
	m.currentHunk = 0
	if m.ready {
		content, lineCount, lineNums, hunkLines := colorizeDiff(diff, m.viewport.Width(), 0, false, m.viewMode)
		m.lineCount = lineCount
		m.lineNewNumbers = lineNums
		m.hunkLines = hunkLines
		m.currentHunk = clampHunkIndex(0, len(hunkLines))
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
				m.cursorLine = m.visibleMidpointLine()
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
		case "v":
			m.viewMode = (m.viewMode + 1) % 3
			m.cursorLine = 0
			m.currentHunk = 0
			m.refreshContent()
			if m.ready {
				m.viewport.GotoTop()
			}
			return m, nil
		case "1":
			m.viewMode = diffViewInline
			m.cursorLine = 0
			m.currentHunk = 0
			m.refreshContent()
			if m.ready {
				m.viewport.GotoTop()
			}
			return m, nil
		case "2":
			m.viewMode = diffViewModified
			m.cursorLine = 0
			m.currentHunk = 0
			m.refreshContent()
			if m.ready {
				m.viewport.GotoTop()
			}
			return m, nil
		case "3":
			m.viewMode = diffViewSplit
			m.cursorLine = 0
			m.currentHunk = 0
			m.refreshContent()
			if m.ready {
				m.viewport.GotoTop()
			}
			return m, nil
		case "]":
			m.jumpNextHunk()
			m.refreshContent()
			return m, nil
		case "[":
			m.jumpPrevHunk()
			m.refreshContent()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiffModel) visibleMidpointLine() int {
	if m.lineCount <= 0 {
		return 0
	}
	line := m.viewport.YOffset() + m.viewport.Height()/2
	if line < 0 {
		return 0
	}
	if line >= m.lineCount {
		return m.lineCount - 1
	}
	return line
}

func (m *DiffModel) refreshContent() {
	if !m.ready {
		return
	}
	content, lineCount, lineNums, hunkLines := colorizeDiff(m.rawDiff, m.viewport.Width(), m.cursorLine, m.cursorMode, m.viewMode)
	m.lineCount = lineCount
	m.lineNewNumbers = lineNums
	m.hunkLines = hunkLines
	m.currentHunk = clampHunkIndex(m.currentHunk, len(hunkLines))
	m.viewport.SetContent(content)
}

func (m DiffModel) View() string {
	if !m.ready || m.path == "" {
		return theme.HelpDesc.Render("  No diff loaded.")
	}
	hint := "  ↑/↓ scroll · [/ ] hunk jump · v mode · 1/2/3 set mode · i cursor mode · esc back"
	if m.cursorMode {
		hint = "  j/k move cursor · c comment · [/ ] hunk jump · v mode · 1/2/3 set mode · esc exit cursor mode"
	}
	return theme.SectionHeader.Render("Diff ["+m.viewMode.Label()+"]: "+m.path) + "\n" + m.viewport.View() + "\n" +
		theme.HelpDesc.Render(hint)
}

func (m diffViewMode) Label() string {
	switch m {
	case diffViewModified:
		return "modified"
	case diffViewSplit:
		return "split"
	default:
		return "inline"
	}
}

func colorizeDiff(diff string, totalWidth int, cursorLine int, cursorMode bool, mode diffViewMode) (string, int, []int, []int) {
	if totalWidth < 40 {
		totalWidth = 40
	}

	type parsedLine struct {
		kind   string // "ctx", "add", "del", "hunk", "meta"
		raw    string
		oldNo  int
		newNo  int
		oldTxt string
		newTxt string
	}

	parsed := make([]parsedLine, 0, 256)
	oldLine, newLine := 0, 0

	for _, rawLine := range strings.Split(diff, "\n") {
		rawLine = strings.ReplaceAll(rawLine, "\t", "    ")
		switch {
		case strings.HasPrefix(rawLine, "@@"):
			if mm := hunkHeaderRe.FindStringSubmatch(rawLine); len(mm) == 3 {
				if v, err := strconv.Atoi(mm[1]); err == nil {
					oldLine = v
				}
				if v, err := strconv.Atoi(mm[2]); err == nil {
					newLine = v
				}
			}
			parsed = append(parsed, parsedLine{kind: "hunk", raw: rawLine})
		case strings.HasPrefix(rawLine, "+++") || strings.HasPrefix(rawLine, "---"):
			parsed = append(parsed, parsedLine{kind: "meta", raw: rawLine})
		case strings.HasPrefix(rawLine, "+"):
			parsed = append(parsed, parsedLine{kind: "add", raw: rawLine, newNo: newLine, newTxt: stripDiffPrefix(rawLine)})
			newLine++
		case strings.HasPrefix(rawLine, "-"):
			parsed = append(parsed, parsedLine{kind: "del", raw: rawLine, oldNo: oldLine, oldTxt: stripDiffPrefix(rawLine)})
			oldLine++
		default:
			parsed = append(parsed, parsedLine{kind: "ctx", raw: rawLine, oldNo: oldLine, newNo: newLine, oldTxt: stripDiffPrefix(rawLine), newTxt: stripDiffPrefix(rawLine)})
			oldLine++
			newLine++
		}
	}

	type renderedLine struct {
		line      string
		newFileNo int
	}

	rendered := make([]renderedLine, 0, len(parsed))
	hunkLines := make([]int, 0, 16)

	switch mode {
	case diffViewModified:
		wrapWidth := totalWidth - modifiedGutterWidth
		if wrapWidth < 20 {
			wrapWidth = 20
		}
		for _, p := range parsed {
			if p.kind == "hunk" {
				hunkLines = append(hunkLines, len(rendered))
				rendered = append(rendered, renderedLine{line: theme.DiffHunk.Render(p.raw), newFileNo: 0})
				continue
			}
			if p.kind != "ctx" && p.kind != "add" {
				continue
			}
			gutter := theme.DiffGutter.Render(fmt.Sprintf("%4d", p.newNo)) + theme.DiffGutterSep.Render("│")
			parts := wrapRunes(p.newTxt, wrapWidth)
			for i, part := range parts {
				g := gutter
				if i > 0 {
					g = theme.DiffGutter.Render("    ") + theme.DiffGutterSep.Render("│")
				}
				content := part
				if p.kind == "add" {
					content = theme.DiffAdd.Render(part)
				}
				lineNo := 0
				if i == 0 {
					lineNo = p.newNo
				}
				rendered = append(rendered, renderedLine{line: g + content, newFileNo: lineNo})
			}
		}

	case diffViewSplit:
		leftGutterW := 5
		rightGutterW := 5
		sep := " │ "
		contentW := (totalWidth - leftGutterW - rightGutterW - utf8.RuneCountInString(sep)) / 2
		if contentW < 12 {
			contentW = 12
		}
		for _, p := range parsed {
			if p.kind == "meta" || p.kind == "hunk" {
				if p.kind == "hunk" {
					hunkLines = append(hunkLines, len(rendered))
				}
				styled := p.raw
				if p.kind == "meta" {
					styled = theme.DiffMeta.Render(styled)
				} else {
					styled = theme.DiffHunk.Render(styled)
				}
				rendered = append(rendered, renderedLine{line: styled})
				continue
			}

			leftText := ""
			rightText := ""
			leftNo := 0
			rightNo := 0
			leftKind := ""
			rightKind := ""

			switch p.kind {
			case "ctx":
				leftText, rightText = p.oldTxt, p.newTxt
				leftNo, rightNo = p.oldNo, p.newNo
			case "add":
				rightText, rightNo = p.newTxt, p.newNo
				rightKind = "add"
			case "del":
				leftText, leftNo = p.oldTxt, p.oldNo
				leftKind = "del"
			}

			leftParts := wrapRunes(leftText, contentW)
			rightParts := wrapRunes(rightText, contentW)
			rows := len(leftParts)
			if len(rightParts) > rows {
				rows = len(rightParts)
			}
			if rows == 0 {
				rows = 1
				leftParts = []string{""}
				rightParts = []string{""}
			}

			for i := 0; i < rows; i++ {
				leftPart := ""
				rightPart := ""
				if i < len(leftParts) {
					leftPart = leftParts[i]
				}
				if i < len(rightParts) {
					rightPart = rightParts[i]
				}

				leftG := "    "
				rightG := "    "
				if i == 0 && leftNo > 0 {
					leftG = fmt.Sprintf("%4d", leftNo)
				}
				if i == 0 && rightNo > 0 {
					rightG = fmt.Sprintf("%4d", rightNo)
				}

				leftStyled := padRunes(leftPart, contentW)
				rightStyled := padRunes(rightPart, contentW)
				if leftKind == "del" {
					leftStyled = theme.DiffDelete.Render(leftStyled)
				}
				if rightKind == "add" {
					rightStyled = theme.DiffAdd.Render(rightStyled)
				}

				leftCol := theme.DiffGutter.Render(leftG) + theme.DiffGutterSep.Render("│") + leftStyled
				rightCol := theme.DiffGutter.Render(rightG) + theme.DiffGutterSep.Render("│") + rightStyled

				lineNo := 0
				if i == 0 {
					lineNo = rightNo
				}
				rendered = append(rendered, renderedLine{line: leftCol + sep + rightCol, newFileNo: lineNo})
			}
		}

	default: // inline
		wrapWidth := totalWidth - inlineGutterWidth
		if wrapWidth < 20 {
			wrapWidth = 20
		}
		for _, p := range parsed {
			var gutterOld, gutterNew string
			content := p.raw
			diffType := p.kind
			thisNewLine := 0

			switch p.kind {
			case "hunk", "meta":
				gutterOld, gutterNew = "    ", "    "
			case "add":
				gutterOld, gutterNew = "    ", fmt.Sprintf("%4d", p.newNo)
				thisNewLine = p.newNo
			case "del":
				gutterOld, gutterNew = fmt.Sprintf("%4d", p.oldNo), "    "
			default:
				gutterOld, gutterNew = fmt.Sprintf("%4d", p.oldNo), fmt.Sprintf("%4d", p.newNo)
				thisNewLine = p.newNo
			}

			gutter := theme.DiffGutter.Render(gutterOld+" "+gutterNew) + theme.DiffGutterSep.Render("│")
			parts := wrapRunes(content, wrapWidth)
			if diffType == "hunk" {
				hunkLines = append(hunkLines, len(rendered))
			}
			for i, part := range parts {
				g := gutter
				if i > 0 {
					g = theme.DiffGutter.Render("         ") + theme.DiffGutterSep.Render("│")
				}
				r := part
				switch diffType {
				case "hunk":
					r = theme.DiffHunk.Render(part)
				case "meta":
					r = theme.DiffMeta.Render(part)
				case "add":
					r = theme.DiffAdd.Render(part)
				case "del":
					r = theme.DiffDelete.Render(part)
				}
				lineNo := 0
				if i == 0 {
					lineNo = thisNewLine
				}
				rendered = append(rendered, renderedLine{line: g + r, newFileNo: lineNo})
			}
		}
	}
	var b strings.Builder
	lineNums := make([]int, len(rendered))
	for i, l := range rendered {
		lineNums[i] = l.newFileNo
		line := l.line
		if cursorMode && i == cursorLine {
			line = lipgloss.NewStyle().
				Background(theme.Subtle).
				Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String(), len(rendered), lineNums, hunkLines
}

func clampHunkIndex(idx, count int) int {
	if count == 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= count {
		return count - 1
	}
	return idx
}

func (m *DiffModel) jumpNextHunk() {
	if len(m.hunkLines) == 0 {
		return
	}
	if m.currentHunk < len(m.hunkLines)-1 {
		m.currentHunk++
	}
	target := m.hunkLines[m.currentHunk]
	m.viewport.SetYOffset(target)
	if m.cursorMode {
		m.cursorLine = target
	}
}

func (m *DiffModel) jumpPrevHunk() {
	if len(m.hunkLines) == 0 {
		return
	}
	if m.currentHunk > 0 {
		m.currentHunk--
	}
	target := m.hunkLines[m.currentHunk]
	m.viewport.SetYOffset(target)
	if m.cursorMode {
		m.cursorLine = target
	}
}

func stripDiffPrefix(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	if r[0] == '+' || r[0] == '-' || r[0] == ' ' {
		return string(r[1:])
	}
	return s
}

func padRunes(s string, width int) string {
	w := len([]rune(s))
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
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
