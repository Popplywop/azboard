package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestListPullRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("searchCriteria.status"); got != "active" {
			t.Errorf("expected status=active, got %s", got)
		}
		respondJSON(w, 200, ListResponse[PullRequest]{
			Count: 1,
			Value: []PullRequest{{PullRequestID: 42, Title: "Test PR", Status: "active"}},
		})
	})
	_, c := testServer(t, mux)

	prs, err := c.ListPullRequests("active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 || prs[0].PullRequestID != 42 {
		t.Errorf("got %+v, want 1 PR with ID 42", prs)
	}
}

func TestListPullRequestsForRepo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/my-repo/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, ListResponse[PullRequest]{
			Count: 1,
			Value: []PullRequest{{PullRequestID: 10, Title: "Repo PR"}},
		})
	})
	_, c := testServer(t, mux)

	prs, err := c.ListPullRequestsForRepo("my-repo", "active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 || prs[0].PullRequestID != 10 {
		t.Errorf("got %+v, want 1 PR with ID 10", prs)
	}
}

func TestListDraftPullRequests(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("searchCriteria.isDraft"); got != "true" {
			t.Errorf("expected isDraft=true, got %s", got)
		}
		respondJSON(w, 200, ListResponse[PullRequest]{
			Count: 1,
			Value: []PullRequest{{PullRequestID: 99, IsDraft: true}},
		})
	})
	_, c := testServer(t, mux)

	prs, err := c.ListDraftPullRequests()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 || !prs[0].IsDraft {
		t.Errorf("got %+v, want 1 draft PR", prs)
	}
}

func TestListDraftPullRequestsForRepo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/my-repo/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("searchCriteria.isDraft"); got != "true" {
			t.Errorf("expected isDraft=true, got %s", got)
		}
		respondJSON(w, 200, ListResponse[PullRequest]{
			Count: 1,
			Value: []PullRequest{{PullRequestID: 77, IsDraft: true}},
		})
	})
	_, c := testServer(t, mux)

	prs, err := c.ListDraftPullRequestsForRepo("my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 {
		t.Errorf("got %d PRs, want 1", len(prs))
	}
}

func TestGetPullRequestByID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, ListResponse[PullRequest]{
			Count: 2,
			Value: []PullRequest{
				{PullRequestID: 1, Title: "Wrong PR"},
				{PullRequestID: 42, Title: "Right PR"},
			},
		})
	})
	_, c := testServer(t, mux)

	pr, err := c.GetPullRequestByID(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.Title != "Right PR" {
		t.Errorf("got title %q, want %q", pr.Title, "Right PR")
	}
}

func TestGetPullRequestByIDNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, ListResponse[PullRequest]{Count: 0, Value: nil})
	})
	_, c := testServer(t, mux)

	_, err := c.GetPullRequestByID(999)
	if err == nil {
		t.Fatal("expected error for missing PR")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestGetPullRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, PullRequest{PullRequestID: 5, Title: "Single PR"})
	})
	_, c := testServer(t, mux)

	pr, err := c.GetPullRequest("repo-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.PullRequestID != 5 {
		t.Errorf("got ID %d, want 5", pr.PullRequestID)
	}
}

func TestGetPullRequestThreads(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5/threads", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, ListResponse[Thread]{
			Count: 1,
			Value: []Thread{{ID: 1, Status: "active"}},
		})
	})
	_, c := testServer(t, mux)

	threads, err := c.GetPullRequestThreads("repo-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(threads) != 1 || threads[0].ID != 1 {
		t.Errorf("got %+v, want 1 thread with ID 1", threads)
	}
}

func TestCreatePullRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body CreatePullRequestRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Title != "New PR" || body.SourceRefName != "refs/heads/feature" {
			t.Errorf("unexpected body: %+v", body)
		}
		respondJSON(w, 201, PullRequest{PullRequestID: 100, Title: "New PR"})
	})
	_, c := testServer(t, mux)

	pr, err := c.CreatePullRequest("repo-1", "New PR", "refs/heads/feature", "refs/heads/main", "desc", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr.PullRequestID != 100 {
		t.Errorf("got ID %d, want 100", pr.PullRequestID)
	}
}

func TestMergePullRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var body CompletePullRequestRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Status != "completed" || body.CompletionOptions.MergeStrategy != "squash" {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.MergePullRequest("repo-1", 5, "squash", "merge commit", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAbandonPullRequest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5", func(w http.ResponseWriter, r *http.Request) {
		var body StatusUpdateRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Status != "abandoned" {
			t.Errorf("expected status=abandoned, got %s", body.Status)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.AbandonPullRequest("repo-1", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToggleDraft(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"isDraft":true`) {
			t.Errorf("expected isDraft:true in body, got %s", body)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.ToggleDraft("repo-1", 5, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateThread(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5/threads", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body CreateThreadRequest
		json.NewDecoder(r.Body).Decode(&body)
		if len(body.Comments) != 1 || body.Comments[0].Content != "looks good" {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.CreateThread("repo-1", 5, "looks good", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReplyToThread(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5/threads/1/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.ReplyToThread("repo-1", 5, 1, "thanks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateThreadStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5/threads/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var body UpdateThreadRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Status != "fixed" {
			t.Errorf("expected status=fixed, got %s", body.Status)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.UpdateThreadStatus("repo-1", 5, 1, "fixed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetVote(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/repositories/repo-1/pullrequests/5/reviewers/user-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		var body SetVoteRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Vote != 10 {
			t.Errorf("expected vote=10, got %d", body.Vote)
		}
		w.WriteHeader(200)
	})
	_, c := testServer(t, mux)

	err := c.SetVote("repo-1", 5, "user-1", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCurrentUserID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/connectionData", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 200, ConnectionData{
			AuthenticatedUser: struct {
				ID          string `json:"id"`
				DisplayName string `json:"providerDisplayName"`
			}{ID: "user-abc"},
		})
	})
	_, c := testServer(t, mux)

	id, err := c.GetCurrentUserID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "user-abc" {
		t.Errorf("got %q, want %q", id, "user-abc")
	}
}

func TestListPullRequests404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 404, map[string]string{"message": "not found"})
	})
	_, c := testServer(t, mux)

	_, err := c.ListPullRequests("active")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got false for: %v", err)
	}
}

func TestListPullRequests500(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, 500, map[string]string{"message": "internal error"})
	})
	_, c := testServer(t, mux)

	_, err := c.ListPullRequests("active")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !IsServerError(err) {
		t.Errorf("expected IsServerError to be true, got false for: %v", err)
	}
}

func TestPATAuth401NoRetry(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/git/pullrequests", func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(401)
	})
	_, c := testServer(t, mux)

	_, err := c.ListPullRequests("active")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if calls != 1 {
		t.Errorf("PAT auth should not retry on 401, got %d calls", calls)
	}
}
