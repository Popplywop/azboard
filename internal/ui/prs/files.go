package prs

import (
	"fmt"
	"sort"
	"strings"

	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/ui/theme"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// treeEntry is one visible row in the flattened file tree.
type treeEntry struct {
	change      api.IterationChange
	displayPath string // tree-decorated string shown in the Path column
	isDir       bool   // true → directory label row, not diffable
	dirPath     string // canonical dir path used as collapse key (dirs only)
}

type FilesModel struct {
	table     table.Model
	changes   []api.IterationChange
	entries   []treeEntry     // currently visible rows (respects collapse state)
	root      *pathNode       // full trie built from changes; rebuilt on SetChanges
	collapsed map[string]bool // set of dirPath strings that are collapsed
	width     int
	height    int
}

func NewFilesModel() FilesModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "T", Width: 4},
			{Title: "Path", Width: 70},
		}),
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

	return FilesModel{
		table:     t,
		collapsed: make(map[string]bool),
	}
}

func (m *FilesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.recalcColumns()
	tableHeight := height - 6 // section header + border + hint line
	if tableHeight < 5 {
		tableHeight = 5
	}
	m.table.SetHeight(tableHeight)
}

func (m *FilesModel) SetChanges(changes []api.IterationChange) {
	m.changes = changes
	m.collapsed = make(map[string]bool) // reset collapse state on new data
	m.root = buildTrie(changes)
	m.rebuildRows()
}

// rebuildRows re-flattens the trie respecting the current collapsed set and
// pushes the result into the table.
func (m *FilesModel) rebuildRows() {
	m.entries = nil
	flattenNode(m.root, "", true, m.collapsed, &m.entries)

	rows := make([]table.Row, 0, len(m.entries))
	for _, e := range m.entries {
		icon := ""
		if e.isDir {
			if m.collapsed[e.dirPath] {
				icon = "▶"
			} else {
				icon = "▼"
			}
		} else {
			icon = changeTypeIcon(e.change.ChangeType)
		}
		rows = append(rows, table.Row{icon, e.displayPath})
	}
	m.table.SetRows(rows)

	// Keep cursor in bounds after a collapse may have shrunk the list.
	if m.table.Cursor() >= len(m.entries) && len(m.entries) > 0 {
		m.table.SetCursor(len(m.entries) - 1)
	}
}

// toggleCollapse toggles the collapsed state of the directory at the current
// cursor position, if it is a directory row.
func (m *FilesModel) toggleCollapse() {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.entries) {
		return
	}
	e := m.entries[idx]
	if !e.isDir {
		return
	}
	m.collapsed[e.dirPath] = !m.collapsed[e.dirPath]
	m.rebuildRows()
}

func (m FilesModel) SelectedChange() (api.IterationChange, bool) {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.entries) {
		return api.IterationChange{}, false
	}
	if m.entries[idx].isDir {
		return api.IterationChange{}, false
	}
	return m.entries[idx].change, true
}

func (m FilesModel) IsOnDir() bool {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.entries) {
		return false
	}
	return m.entries[idx].isDir
}

func (m FilesModel) HasChanges() bool {
	return len(m.changes) > 0
}

func (m FilesModel) Update(msg tea.Msg) (FilesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		_ = msg
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m FilesModel) View() string {
	var b strings.Builder
	b.WriteString(theme.SectionHeader.Render(fmt.Sprintf("  Files (%d changed)", len(m.changes))))
	b.WriteString("\n")
	b.WriteString(theme.TableBorder.Render(m.table.View()))
	b.WriteString("\n")
	b.WriteString(theme.HelpDesc.Render("  ↑/↓ navigate · enter toggle dir / view diff · esc back"))
	return b.String()
}

func (m *FilesModel) recalcColumns() {
	available := m.width - 6 // -2 border, -4 padding
	if available < 40 {
		available = 40
	}
	typeW := 4
	pathW := available - typeW - 2
	if pathW < 20 {
		pathW = 20
	}
	m.table.SetColumns([]table.Column{
		{Title: "T", Width: typeW},
		{Title: "Path", Width: pathW},
	})
	if m.width > 2 {
		m.table.SetWidth(m.width - 2)
	}
}

// --- Trie building ---

type pathNode struct {
	name     string
	children map[string]*pathNode
	change   *api.IterationChange // non-nil for leaf (file) nodes
	isDir    bool
}

func newPathNode(name string) *pathNode {
	return &pathNode{
		name:     name,
		children: make(map[string]*pathNode),
		isDir:    true,
	}
}

func buildTrie(changes []api.IterationChange) *pathNode {
	root := newPathNode("")
	for i := range changes {
		ch := &changes[i]
		p := strings.TrimPrefix(ch.Item.Path, "/")
		parts := strings.Split(p, "/")
		node := root
		for j, part := range parts {
			if _, ok := node.children[part]; !ok {
				node.children[part] = newPathNode(part)
			}
			child := node.children[part]
			if j == len(parts)-1 {
				child.isDir = false
				child.change = ch
			}
			node = child
		}
	}
	return root
}

// flattenNode recursively walks the trie and appends visible treeEntry rows.
// collapsed is the set of dirPaths that should be collapsed (children hidden).
// isRoot suppresses emitting a row for the synthetic root node.
func flattenNode(node *pathNode, prefix string, isRoot bool, collapsed map[string]bool, entries *[]treeEntry) {
	keys := make([]string, 0, len(node.children))
	for k := range node.children {
		keys = append(keys, k)
	}
	// Dirs before files, then lexicographic within each group.
	sort.Slice(keys, func(i, j int) bool {
		ci := node.children[keys[i]]
		cj := node.children[keys[j]]
		if ci.isDir != cj.isDir {
			return ci.isDir
		}
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	for idx, key := range keys {
		child := node.children[key]
		isLast := idx == len(keys)-1

		var connector, childPrefix string
		if isRoot {
			connector = ""
			childPrefix = ""
		} else if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		} else {
			connector = "├── "
			childPrefix = prefix + "│   "
		}

		displayPath := prefix + connector + key

		if child.isDir {
			// Build a canonical path for this directory by walking up from
			// prefix indentation — we use the display path stripped of connectors.
			dirPath := canonicalDirPath(displayPath)
			*entries = append(*entries, treeEntry{
				change:      api.IterationChange{Item: api.ChangeItem{Path: dirPath}},
				displayPath: displayPath + "/",
				isDir:       true,
				dirPath:     dirPath,
			})
			if !collapsed[dirPath] {
				flattenNode(child, childPrefix, false, collapsed, entries)
			}
		} else {
			*entries = append(*entries, treeEntry{
				change:      *child.change,
				displayPath: displayPath,
				isDir:       false,
			})
		}
	}
}

// canonicalDirPath strips tree drawing characters from a displayPath to
// produce a stable key for collapse tracking.
func canonicalDirPath(displayPath string) string {
	// Strip all leading tree-drawing prefix characters iteratively.
	s := displayPath
	for {
		trimmed := s
		for _, tok := range []string{"└── ", "├── ", "│   ", "    "} {
			trimmed = strings.TrimPrefix(trimmed, tok)
		}
		if trimmed == s {
			break
		}
		s = trimmed
	}
	return s
}

func changeTypeIcon(changeType string) string {
	switch strings.ToLower(changeType) {
	case "add":
		return "A"
	case "delete":
		return "D"
	case "rename":
		return "R"
	default:
		return "M"
	}
}
