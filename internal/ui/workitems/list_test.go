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

func TestMatchesItemEmptyQuery(t *testing.T) {
	item := api.WorkItem{ID: 1, Fields: api.WorkItemFields{Title: "anything"}}
	m := ListModel{}
	// Empty query after Fields splitting should match everything
	if !m.matchesItem(item, "") {
		t.Error("empty query should match all items")
	}
}

func TestCurrentScope(t *testing.T) {
	m := ListModel{scopeIndex: 0}
	scope := m.currentScope()
	if scope.Label != "My Work" {
		t.Errorf("expected 'My Work', got %q", scope.Label)
	}
	if scope.AssignedTo != "@me" {
		t.Errorf("expected '@me', got %q", scope.AssignedTo)
	}
}

func TestCurrentScopeOutOfBounds(t *testing.T) {
	m := ListModel{scopeIndex: -1}
	scope := m.currentScope()
	if scope.Label != "My Work" {
		t.Errorf("out-of-bounds should return first scope, got %q", scope.Label)
	}

	m.scopeIndex = 999
	scope = m.currentScope()
	if scope.Label != "My Work" {
		t.Errorf("out-of-bounds should return first scope, got %q", scope.Label)
	}
}

func TestIsLoading(t *testing.T) {
	m := ListModel{loading: true}
	if !m.IsLoading() {
		t.Error("expected true")
	}
	m.loading = false
	if m.IsLoading() {
		t.Error("expected false")
	}
}

func TestIsFiltering(t *testing.T) {
	m := ListModel{filtering: true}
	if !m.IsFiltering() {
		t.Error("expected true")
	}
	m.filtering = false
	if m.IsFiltering() {
		t.Error("expected false")
	}
}
