package prs

import (
	"testing"

	"github.com/popplywop/azboard/internal/api"
)

func TestShortName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Alice Chen", "Alice C."},
		{"Marcus", "Marcus"},
		{"", ""},
		{"Jean-Paul DuBois", "Jean-Paul D."},
		{"A B C", "A C."},
	}
	for _, tt := range tests {
		got := shortName(tt.input)
		if got != tt.want {
			t.Errorf("shortName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMatchesPR(t *testing.T) {
	pr := api.PullRequest{
		PullRequestID: 42,
		Title:         "Fix authentication bug",
		Status:        "active",
		IsDraft:       false,
		CreatedBy:     api.IdentityRef{DisplayName: "Alice Chen"},
		Repository:    api.GitRepository{Name: "auth-service"},
		Reviewers:     []api.Reviewer{{DisplayName: "Bob Smith", Vote: 10}},
	}

	m := ListModel{}

	tests := []struct {
		query string
		want  bool
	}{
		{"fix", true},
		{"authentication", true},
		{"auth-service", true},
		{"alice", true},
		{"active", true},
		{"42", true},
		{"bob", true},
		{"nonexistent", false},
		{"fix alice", true},
		{"fix nobody", false},
	}
	for _, tt := range tests {
		got := m.matchesPR(pr, tt.query)
		if got != tt.want {
			t.Errorf("matchesPR(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestMatchesPRDraft(t *testing.T) {
	pr := api.PullRequest{IsDraft: true}
	m := ListModel{}
	if !m.matchesPR(pr, "draft") {
		t.Error("expected draft PR to match 'draft' query")
	}
}

func TestCycleScope(t *testing.T) {
	m := ListModel{scopeIndex: 0}

	m.cycleScope(1)
	if m.scopeIndex != 1 {
		t.Errorf("after +1: got %d, want 1", m.scopeIndex)
	}

	m.cycleScope(-1)
	if m.scopeIndex != 0 {
		t.Errorf("after -1: got %d, want 0", m.scopeIndex)
	}

	m.cycleScope(-1)
	if m.scopeIndex != len(listScopes)-1 {
		t.Errorf("after wrap: got %d, want %d", m.scopeIndex, len(listScopes)-1)
	}
}

func TestBuildRows(t *testing.T) {
	prs := []api.PullRequest{
		{
			PullRequestID: 1,
			Title:         "Test PR",
			Status:        "active",
			CreatedBy:     api.IdentityRef{DisplayName: "Alice"},
			Repository:    api.GitRepository{Name: "repo"},
		},
	}
	m := ListModel{}
	rows := m.buildRows(prs)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0][0] != "1" {
		t.Errorf("got ID %q, want %q", rows[0][0], "1")
	}
}

func TestIs404WithAPIError(t *testing.T) {
	err := &api.APIError{StatusCode: 404, Body: "not found"}
	if !is404(err) {
		t.Error("expected is404 to return true for 404 APIError")
	}
}

func TestIs404WithNilError(t *testing.T) {
	if is404(nil) {
		t.Error("expected is404 to return false for nil")
	}
}
