package prs

import (
	"testing"

	"github.com/popplywop/azboard/internal/api"
)

func TestMergeStrategyAPIValue(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"squash", "squash"},
		{"merge", "noFastForward"},
		{"rebase", "rebase"},
		{"semilinear", "rebaseMerge"},
		{"", "squash"},
		{"invalid", "squash"},
	}
	for _, tt := range tests {
		got := mergeStrategyAPIValue(tt.input)
		if got != tt.want {
			t.Errorf("mergeStrategyAPIValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFilterThreads(t *testing.T) {
	m := DetailModel{
		threads: []api.Thread{
			{ID: 1, IsDeleted: false, Comments: []api.Comment{
				{ID: 1, CommentType: "text", IsDeleted: false, Content: "looks good"},
			}},
			{ID: 2, IsDeleted: true, Comments: []api.Comment{
				{ID: 1, CommentType: "text", IsDeleted: false, Content: "deleted thread"},
			}},
			{ID: 3, IsDeleted: false, Comments: []api.Comment{
				{ID: 1, CommentType: "system", IsDeleted: false, Content: "system only"},
			}},
			{ID: 4, IsDeleted: false, Comments: []api.Comment{
				{ID: 1, CommentType: "text", IsDeleted: true, Content: "deleted comment"},
			}},
			{ID: 5, IsDeleted: false, Comments: []api.Comment{
				{ID: 1, CommentType: "system", Content: "sys"},
				{ID: 2, CommentType: "text", IsDeleted: false, Content: "user reply"},
			}},
		},
	}

	visible := m.filterThreads()
	if len(visible) != 2 {
		t.Errorf("expected 2 visible threads, got %d", len(visible))
		for _, v := range visible {
			t.Logf("  thread ID=%d", v.ID)
		}
		return
	}
	if visible[0].ID != 1 || visible[1].ID != 5 {
		t.Errorf("expected thread IDs [1, 5], got [%d, %d]", visible[0].ID, visible[1].ID)
	}
}

func TestFilterThreadsEmpty(t *testing.T) {
	m := DetailModel{}
	visible := m.filterThreads()
	if len(visible) != 0 {
		t.Errorf("expected 0 threads, got %d", len(visible))
	}
}

func TestIsComposing(t *testing.T) {
	tests := []struct {
		mode int
		want bool
	}{
		{int(modeNormal), false},
		{int(modeCompose), true},
		{int(modeConfirm), true},
	}
	for _, tt := range tests {
		m := DetailModel{mode: detailMode(tt.mode)}
		got := m.IsComposing()
		if got != tt.want {
			t.Errorf("IsComposing() with mode %d = %v, want %v", tt.mode, got, tt.want)
		}
	}
}
