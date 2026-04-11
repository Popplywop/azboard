package prs

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestWrapRunesNoWrap(t *testing.T) {
	got := wrapRunes("short", 80)
	if len(got) != 1 || got[0] != "short" {
		t.Errorf("wrapRunes(%q, 80) = %v, want [\"short\"]", "short", got)
	}
}

func TestWrapRunesWrap(t *testing.T) {
	got := wrapRunes("abcdef", 3)
	want := []string{"abc", "def"}
	if len(got) != len(want) {
		t.Fatalf("wrapRunes(%q, 3) = %v (len %d), want %v", "abcdef", got, len(got), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("wrapRunes[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWrapRunesExact(t *testing.T) {
	got := wrapRunes("abc", 3)
	if len(got) != 1 || got[0] != "abc" {
		t.Errorf("wrapRunes(%q, 3) = %v, want [\"abc\"]", "abc", got)
	}
}

func TestWrapRunesZeroWidth(t *testing.T) {
	got := wrapRunes("abc", 0)
	if len(got) != 1 || got[0] != "abc" {
		t.Errorf("wrapRunes(%q, 0) = %v, want [\"abc\"]", "abc", got)
	}
}

func TestWrapRunesEmpty(t *testing.T) {
	got := wrapRunes("", 80)
	if len(got) != 1 || got[0] != "" {
		t.Errorf("wrapRunes(\"\", 80) = %v, want [\"\"]", got)
	}
}

func TestColorizeDiffLineNumbers(t *testing.T) {
	diff := "@@ -1,3 +1,4 @@\n context\n+added\n context2\n-removed\n"
	_, lineCount, lineNums := colorizeDiff(diff, 80, 0, false)
	if lineCount < 5 {
		t.Errorf("lineCount = %d, want >= 5", lineCount)
	}
	// lineNums should have non-zero entries for context and added lines
	hasNonZero := false
	for _, n := range lineNums {
		if n > 0 {
			hasNonZero = true
			break
		}
	}
	if !hasNonZero {
		t.Error("expected some non-zero line numbers in lineNums")
	}
}

func TestDiffCursorModeStartsAtVisibleMidpoint(t *testing.T) {
	var lines []string
	lines = append(lines, "@@ -1,80 +1,80 @@")
	for i := 0; i < 80; i++ {
		lines = append(lines, " context")
	}
	diff := strings.Join(lines, "\n")

	m := NewDiffModel()
	m.SetSize(120, 20)
	m.SetDiff("file.txt", diff)
	m.viewport.SetYOffset(12)

	want := m.viewport.YOffset() + m.viewport.Height()/2
	if want >= m.lineCount {
		want = m.lineCount - 1
	}

	gotModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "i", Code: 'i'}))
	if !gotModel.InCursorMode() {
		t.Fatal("expected cursor mode to be enabled after pressing i")
	}
	if gotModel.CursorLine() != want {
		t.Fatalf("cursor line = %d, want %d", gotModel.CursorLine(), want)
	}
}

func TestDiffCursorModeMidpointClampsToLastLine(t *testing.T) {
	diff := "@@ -1,2 +1,2 @@\n a\n b\n"

	m := NewDiffModel()
	m.SetSize(120, 30)
	m.SetDiff("small.txt", diff)
	m.viewport.SetYOffset(999)

	gotModel, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "i", Code: 'i'}))
	if gotModel.CursorLine() != m.lineCount-1 {
		t.Fatalf("cursor line = %d, want %d", gotModel.CursorLine(), m.lineCount-1)
	}
}
