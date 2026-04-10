package api

import (
	"net/http"
	"testing"
)

func TestListRepositories(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		respondJSON(w, 200, ListResponse[GitRepository]{
			Count: 2,
			Value: []GitRepository{
				{ID: "r1", Name: "inventory-api"},
				{ID: "r2", Name: "web-portal"},
			},
		})
	})
	_, c := testServer(t, mux)

	repos, err := c.ListRepositories()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2", len(repos))
	}
	if repos[0].Name != "inventory-api" {
		t.Errorf("got name %q, want %q", repos[0].Name, "inventory-api")
	}
}

func TestListBranches(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/my-repo/refs", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter"); got != "heads" {
			t.Errorf("expected filter=heads, got %s", got)
		}
		respondJSON(w, 200, ListResponse[GitBranch]{
			Count: 2,
			Value: []GitBranch{
				{Name: "refs/heads/main"},
				{Name: "refs/heads/feature"},
			},
		})
	})
	_, c := testServer(t, mux)

	branches, err := c.ListBranches("my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(branches) != 2 {
		t.Errorf("got %d branches, want 2", len(branches))
	}
	if branches[0].ShortName() != "main" {
		t.Errorf("got short name %q, want %q", branches[0].ShortName(), "main")
	}
}

func TestGetProjectID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/projects/TestProject", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, projectResponse{ID: "proj-123", Name: "TestProject"})
	})
	_, c := testServer(t, mux)

	id, err := c.GetProjectID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "proj-123" {
		t.Errorf("got %q, want %q", id, "proj-123")
	}
}
