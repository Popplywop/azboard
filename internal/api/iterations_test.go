package api

import (
	"net/http"
	"testing"
)

func TestGetPullRequestIterations(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5/iterations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		respondJSON(w, 200, ListResponse[Iteration]{
			Count: 2,
			Value: []Iteration{
				{ID: 1, SourceRefCommit: CommitRef{CommitID: "abc"}, TargetRefCommit: CommitRef{CommitID: "def"}},
				{ID: 2, SourceRefCommit: CommitRef{CommitID: "ghi"}, TargetRefCommit: CommitRef{CommitID: "jkl"}},
			},
		})
	})
	_, c := testServer(t, mux)

	iters, err := c.GetPullRequestIterations("repo-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(iters) != 2 {
		t.Errorf("got %d iterations, want 2", len(iters))
	}
	if iters[0].SourceRefCommit.CommitID != "abc" {
		t.Errorf("got commit %q, want %q", iters[0].SourceRefCommit.CommitID, "abc")
	}
}

func TestGetPullRequestIterationChanges(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5/iterations/1/changes", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("$top"); got != "2000" {
			t.Errorf("expected $top=2000, got %s", got)
		}
		respondJSON(w, 200, iterationChangesResponse{
			ChangeEntries: []struct {
				ChangeTrackingID int            `json:"changeTrackingId"`
				ChangeType       string         `json:"changeType"`
				Item             ChangeItem     `json:"item"`
				OriginalPath     string         `json:"originalPath"`
				AdditionalProps  map[string]any `json:"additionalProperties"`
			}{
				{ChangeTrackingID: 1, ChangeType: "edit", Item: ChangeItem{Path: "/src/main.go"}},
				{ChangeTrackingID: 2, ChangeType: "add", Item: ChangeItem{Path: "/src/new.go"}},
			},
		})
	})
	_, c := testServer(t, mux)

	changes, err := c.GetPullRequestIterationChanges("repo-1", 5, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("got %d changes, want 2", len(changes))
	}
	if changes[0].ChangeType != "edit" {
		t.Errorf("got change type %q, want %q", changes[0].ChangeType, "edit")
	}
	if changes[1].Item.Path != "/src/new.go" {
		t.Errorf("got path %q, want %q", changes[1].Item.Path, "/src/new.go")
	}
}
