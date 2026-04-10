package workitems

import (
	"testing"

	"github.com/popplywop/azboard/internal/api"
)

func TestMatchesItem(t *testing.T) {
	item := api.WorkItem{
		ID: 42,
		Fields: api.WorkItemFields{
			Title:        "Fix login bug",
			State:        "Active",
			WorkItemType: "Bug",
			AssignedTo:   api.IdentityRef{DisplayName: "Alice Chen"},
		},
	}

	m := ListModel{}

	tests := []struct {
		query string
		want  bool
	}{
		{"fix", true},
		{"login", true},
		{"42", true},
		{"active", true},
		{"bug", true},
		{"alice", true},
		{"nonexistent", false},
		{"fix alice", true},
		{"fix nobody", false},
	}
	for _, tt := range tests {
		got := m.matchesItem(item, tt.query)
		if got != tt.want {
			t.Errorf("matchesItem(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}
