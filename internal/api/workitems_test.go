package api

import (
	"strings"
	"testing"
)

func TestBuildWIQLQueryBasic(t *testing.T) {
	q := buildWIQLQuery("MyProject", nil, "", "", false)
	if !strings.Contains(q, "[System.TeamProject] = 'MyProject'") {
		t.Errorf("query missing project clause: %s", q)
	}
	if !strings.HasPrefix(q, "SELECT [System.Id] FROM WorkItems WHERE") {
		t.Errorf("query has wrong prefix: %s", q)
	}
}

func TestBuildWIQLQueryWithTypes(t *testing.T) {
	q := buildWIQLQuery("P", []string{"Bug", "Task"}, "", "", false)
	if !strings.Contains(q, "[System.WorkItemType] IN ('Bug', 'Task')") {
		t.Errorf("query missing type clause: %s", q)
	}
}

func TestBuildWIQLQueryWithAssignedTo(t *testing.T) {
	q := buildWIQLQuery("P", nil, "@me", "", false)
	if !strings.Contains(q, "[System.AssignedTo] = @me") {
		t.Errorf("query missing assignedTo clause: %s", q)
	}
}

func TestBuildWIQLQueryWithAreaPath(t *testing.T) {
	q := buildWIQLQuery("P", nil, "", "PDI\\Team", false)
	if !strings.Contains(q, "[System.AreaPath] UNDER 'PDI\\Team'") {
		t.Errorf("query missing area path clause: %s", q)
	}
}

func TestBuildWIQLQueryActiveOnly(t *testing.T) {
	q := buildWIQLQuery("P", nil, "", "", true)
	if !strings.Contains(q, "[System.State] NOT IN") {
		t.Errorf("query missing active-only clause: %s", q)
	}
}

func TestBuildWIQLQueryEscaping(t *testing.T) {
	q := buildWIQLQuery("My'Project", []string{"User's Story"}, "", "Path'Here", false)
	if !strings.Contains(q, "My''Project") {
		t.Errorf("project not escaped: %s", q)
	}
	if !strings.Contains(q, "User''s Story") {
		t.Errorf("type not escaped: %s", q)
	}
	if !strings.Contains(q, "Path''Here") {
		t.Errorf("area path not escaped: %s", q)
	}
}

func TestBuildWIQLQueryAllClauses(t *testing.T) {
	q := buildWIQLQuery("P", []string{"Bug"}, "@me", "Area", true)
	// Should have 4 AND-separated clauses + the project clause
	parts := strings.Split(q[strings.Index(q, "WHERE ")+6:strings.Index(q, " ORDER")], " AND ")
	if len(parts) != 5 {
		t.Errorf("expected 5 clauses, got %d: %v", len(parts), parts)
	}
}

func TestEscapeWIQL(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"normal", "normal"},
		{"it's", "it''s"},
		{"'quoted'", "''quoted''"},
		{"", ""},
	}
	for _, tt := range tests {
		got := escapeWIQL(tt.input)
		if got != tt.want {
			t.Errorf("escapeWIQL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
