package theme

import "testing"

func TestStatusStyle(t *testing.T) {
	tests := []struct {
		status  string
		isDraft bool
	}{
		{"active", false},
		{"completed", false},
		{"abandoned", false},
		{"active", true},
		{"unknown", false},
	}
	for _, tt := range tests {
		// Should not panic
		s := StatusStyle(tt.status, tt.isDraft)
		_ = s.Render("test")
	}
}

func TestWorkItemTypeIcon(t *testing.T) {
	tests := []struct {
		itemType string
		want     string
	}{
		{"Bug", "\u25c9"},
		{"User Story", "\u25c8"},
		{"Task", "\u25fb"},
		{"Epic", "\u25c6"},
		{"Feature", "\u25c7"},
		{"Unknown", "\u00b7"},
	}
	for _, tt := range tests {
		got := WorkItemTypeIcon(tt.itemType)
		if got != tt.want {
			t.Errorf("WorkItemTypeIcon(%q) = %q, want %q", tt.itemType, got, tt.want)
		}
	}
}

func TestWorkItemTypeStyle(t *testing.T) {
	types := []string{"Bug", "User Story", "Task", "Feature", "Epic", "Unknown"}
	for _, typ := range types {
		s := WorkItemTypeStyle(typ)
		_ = s.Render("test")
	}
}

func TestWorkItemStateStyle(t *testing.T) {
	states := []string{"Active", "In Progress", "New", "To Do", "Resolved", "Closed", "Done", "Unknown"}
	for _, state := range states {
		s := WorkItemStateStyle(state)
		_ = s.Render("test")
	}
}

func TestVoteStyle(t *testing.T) {
	votes := []int{10, 5, 0, -5, -10, 99}
	for _, vote := range votes {
		s := VoteStyle(vote)
		_ = s.Render("test")
	}
}
