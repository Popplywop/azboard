package api

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// --- httptest-based integration tests ---

func TestListWorkItems(t *testing.T) {
	mux := http.NewServeMux()
	// WIQL endpoint returns work item refs
	mux.HandleFunc("/wit/wiql", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body WIQLRequest
		json.NewDecoder(r.Body).Decode(&body)
		if !strings.Contains(body.Query, "System.TeamProject") {
			t.Errorf("query missing project clause: %s", body.Query)
		}
		respondJSON(w, 200, WIQLResult{
			WorkItems: []WIQLRef{
				{ID: 1, URL: "http://example.com/1"},
				{ID: 2, URL: "http://example.com/2"},
			},
		})
	})
	// Work item details endpoint
	mux.HandleFunc("/wit/workitems", func(w http.ResponseWriter, r *http.Request) {
		ids := r.URL.Query().Get("ids")
		if ids != "1,2" {
			t.Errorf("expected ids=1,2, got %s", ids)
		}
		respondJSON(w, 200, ListResponse[WorkItem]{
			Count: 2,
			Value: []WorkItem{
				{ID: 1, Fields: WorkItemFields{Title: "Bug 1", WorkItemType: "Bug"}},
				{ID: 2, Fields: WorkItemFields{Title: "Story 1", WorkItemType: "User Story"}},
			},
		})
	})
	_, c := testServer(t, mux)

	items, err := c.ListWorkItems([]string{"Bug", "User Story"}, "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
}

func TestGetWorkItem(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wit/workitems/42", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("$expand"); got != "all" {
			t.Errorf("expected $expand=all, got %s", got)
		}
		respondJSON(w, 200, WorkItem{ID: 42, Fields: WorkItemFields{Title: "Test Item"}})
	})
	_, c := testServer(t, mux)

	wi, err := c.GetWorkItem(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wi.ID != 42 || wi.Fields.Title != "Test Item" {
		t.Errorf("got %+v, want ID=42 Title=Test Item", wi)
	}
}

func TestGetWorkItemComments(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wit/workitems/42/comments", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, WorkItemCommentsResult{
			Count:    1,
			Comments: []WorkItemComment{{ID: 1, Text: "hello"}},
		})
	})
	_, c := testServer(t, mux)

	comments, err := c.GetWorkItemComments(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 1 || comments[0].Text != "hello" {
		t.Errorf("got %+v, want 1 comment with text 'hello'", comments)
	}
}

func TestUpdateWorkItemState(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wit/workitems/42", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json-patch+json" {
			t.Errorf("expected json-patch content type, got %s", ct)
		}
		var ops []WorkItemPatchOp
		json.NewDecoder(r.Body).Decode(&ops)
		if len(ops) != 1 || ops[0].Path != "/fields/System.State" {
			t.Errorf("unexpected patch ops: %+v", ops)
		}
		if ops[0].Value != "Active" {
			t.Errorf("expected state Active, got %v", ops[0].Value)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.UpdateWorkItemState(42, "Active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddWorkItemComment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wit/workitems/42/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body AddWorkItemCommentRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Text != "new comment" {
			t.Errorf("expected text 'new comment', got %q", body.Text)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.AddWorkItemComment(42, "new comment")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetWorkItemTypeStates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wit/workitemtypes/", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, ListResponse[WorkItemTypeState]{
			Count: 3,
			Value: []WorkItemTypeState{
				{Name: "New"},
				{Name: "Active"},
				{Name: "Closed"},
			},
		})
	})
	_, c := testServer(t, mux)

	states, err := c.GetWorkItemTypeStates("Bug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 3 || states[0] != "New" {
		t.Errorf("got %v, want [New Active Closed]", states)
	}
}

func TestLinkWorkItemToPR(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/wit/workitems/42", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var ops []map[string]interface{}
		json.NewDecoder(r.Body).Decode(&ops)
		if len(ops) != 1 || ops[0]["path"] != "/relations/-" {
			t.Errorf("unexpected patch ops: %+v", ops)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.LinkWorkItemToPR(42, fmt.Sprintf("vstfs:///Git/PullRequestId/proj/repo/%d", 5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
