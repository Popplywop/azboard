package api

import (
	"errors"
	"time"
)

// ErrDemoMode is returned by write operations in demo mode.
var ErrDemoMode = errors.New("write operations are disabled in demo mode")

// MockClient implements Clienter with hardcoded demo data.
type MockClient struct{}

// NewMockClient returns a new demo-mode client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

var demoRepos = []GitRepository{
	{ID: "repo-1", Name: "inventory-api"},
	{ID: "repo-2", Name: "web-portal"},
	{ID: "repo-3", Name: "auth-service"},
}

var demoUsers = []IdentityRef{
	{DisplayName: "Alice Chen", UniqueName: "achen@contoso.com", ID: "user-1"},
	{DisplayName: "Marcus Johnson", UniqueName: "mjohnson@contoso.com", ID: "user-2"},
	{DisplayName: "Sarah Kim", UniqueName: "skim@contoso.com", ID: "user-3"},
	{DisplayName: "David Park", UniqueName: "dpark@contoso.com", ID: "user-4"},
	{DisplayName: "Priya Patel", UniqueName: "ppatel@contoso.com", ID: "user-5"},
}

func demoTime(daysAgo int) time.Time {
	return time.Date(2026, 4, 9, 10, 30, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)
}

func ptr[T any](v T) *T { return &v }

var demoPRs = []PullRequest{
	{
		PullRequestID: 1847,
		Title:         "Add batch inventory sync endpoint",
		Description:   "Implements a new `/api/v2/inventory/sync` endpoint that accepts batch updates.\n\nThis replaces the existing single-item update flow and should reduce API calls by ~80% during nightly syncs.\n\n## Changes\n- New `BatchSyncHandler` with validation and conflict resolution\n- Rate limiting middleware (100 req/min per client)\n- Integration tests covering partial failure scenarios",
		Status:        "active",
		CreationDate:  demoTime(2),
		SourceRefName: "refs/heads/feature/batch-sync",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		CreatedBy:     demoUsers[0],
		Repository:    demoRepos[0],
		Reviewers: []Reviewer{
			{DisplayName: "Marcus Johnson", ID: "user-2", Vote: 10, IsRequired: true},
			{DisplayName: "Priya Patel", ID: "user-5", Vote: 5, IsRequired: false},
			{DisplayName: "David Park", ID: "user-4", Vote: 0, IsRequired: false},
		},
	},
	{
		PullRequestID: 1844,
		Title:         "Fix timezone handling in report scheduler",
		Description:   "Fixes the DST bug where scheduled reports would fire an hour late after spring-forward.",
		Status:        "active",
		CreationDate:  demoTime(3),
		SourceRefName: "refs/heads/fix/timezone-dst",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		CreatedBy:     demoUsers[1],
		Repository:    demoRepos[0],
		Reviewers: []Reviewer{
			{DisplayName: "Alice Chen", ID: "user-1", Vote: 10, IsRequired: true},
			{DisplayName: "Sarah Kim", ID: "user-3", Vote: 10, IsRequired: false},
		},
	},
	{
		PullRequestID: 1842,
		Title:         "Migrate dashboard to React 19",
		Description:   "Major upgrade of the dashboard SPA from React 17 to React 19.\n\nIncludes migration from class components to hooks and concurrent mode.",
		Status:        "active",
		CreationDate:  demoTime(5),
		SourceRefName: "refs/heads/feature/react-19",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "conflicts",
		CreatedBy:     demoUsers[2],
		Repository:    demoRepos[1],
		Reviewers: []Reviewer{
			{DisplayName: "Alice Chen", ID: "user-1", Vote: -5, IsRequired: true},
			{DisplayName: "David Park", ID: "user-4", Vote: 0, IsRequired: false},
		},
	},
	{
		PullRequestID: 1839,
		Title:         "Add OIDC provider support",
		Description:   "Adds OpenID Connect as an authentication provider alongside existing SAML.",
		Status:        "active",
		CreationDate:  demoTime(4),
		SourceRefName: "refs/heads/feature/oidc-auth",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		CreatedBy:     demoUsers[3],
		Repository:    demoRepos[2],
		Reviewers: []Reviewer{
			{DisplayName: "Marcus Johnson", ID: "user-2", Vote: -10, IsRequired: true},
			{DisplayName: "Priya Patel", ID: "user-5", Vote: 0, IsRequired: false},
		},
	},
	{
		PullRequestID: 1836,
		Title:         "[WIP] Prototype GraphQL gateway",
		Description:   "Early draft exploring a GraphQL layer over our REST APIs.",
		Status:        "active",
		IsDraft:       true,
		CreationDate:  demoTime(7),
		SourceRefName: "refs/heads/experiment/graphql",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		CreatedBy:     demoUsers[4],
		Repository:    demoRepos[0],
		Reviewers:     []Reviewer{},
	},
	{
		PullRequestID: 1835,
		Title:         "[WIP] Evaluate Tailwind v4 migration",
		Description:   "Spiking out what a Tailwind v4 migration looks like.",
		Status:        "active",
		IsDraft:       true,
		CreationDate:  demoTime(8),
		SourceRefName: "refs/heads/spike/tailwind-v4",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		CreatedBy:     demoUsers[2],
		Repository:    demoRepos[1],
		Reviewers:     []Reviewer{},
	},
	{
		PullRequestID: 1830,
		Title:         "Upgrade Go to 1.24 and update dependencies",
		Description:   "Routine dependency update. All tests passing.",
		Status:        "completed",
		CreationDate:  demoTime(10),
		ClosedDate:    ptr(demoTime(8)),
		SourceRefName: "refs/heads/chore/go-1.24",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		CreatedBy:     demoUsers[0],
		Repository:    demoRepos[0],
		Reviewers: []Reviewer{
			{DisplayName: "Marcus Johnson", ID: "user-2", Vote: 10, IsRequired: true},
		},
	},
	{
		PullRequestID: 1825,
		Title:         "Remove deprecated v1 auth endpoints",
		Description:   "Cleans up the v1 auth routes that have been deprecated since Q3.",
		Status:        "abandoned",
		CreationDate:  demoTime(14),
		ClosedDate:    ptr(demoTime(12)),
		SourceRefName: "refs/heads/cleanup/v1-auth",
		TargetRefName: "refs/heads/main",
		MergeStatus:   "succeeded",
		CreatedBy:     demoUsers[3],
		Repository:    demoRepos[2],
		Reviewers: []Reviewer{
			{DisplayName: "Alice Chen", ID: "user-1", Vote: -5, IsRequired: true},
		},
	},
}

var demoThreads = []Thread{
	{
		ID:     1,
		Status: "active",
		Comments: []Comment{
			{
				ID:            1,
				Author:        demoUsers[1],
				Content:       "Should we add a circuit breaker around the external inventory call? If their API goes down we could end up with a lot of failed syncs.",
				PublishedDate: demoTime(1),
				CommentType:   "text",
			},
			{
				ID:            2,
				Author:        demoUsers[0],
				Content:       "Good call. I added a retry with exponential backoff in the latest commit. Want me to add a full circuit breaker too?",
				PublishedDate: demoTime(1),
				CommentType:   "text",
			},
		},
		PublishedDate:   demoTime(1),
		LastUpdatedDate: demoTime(1),
	},
	{
		ID:     2,
		Status: "fixed",
		Comments: []Comment{
			{
				ID:            3,
				Author:        demoUsers[4],
				Content:       "The rate limit of 100/min might be too aggressive for the nightly bulk import. Our largest client pushes ~5k items.",
				PublishedDate: demoTime(2),
				CommentType:   "text",
			},
			{
				ID:            4,
				Author:        demoUsers[0],
				Content:       "Bumped to 500/min and added a configurable override via X-RateLimit-Override header for internal services.",
				PublishedDate: demoTime(1),
				CommentType:   "text",
			},
		},
		PublishedDate:   demoTime(2),
		LastUpdatedDate: demoTime(1),
	},
	{
		ID:     3,
		Status: "active",
		ThreadContext: &ThreadContext{
			FilePath:       "/src/handlers/sync.go",
			RightFileStart: &LineRange{Line: 42, Offset: 1},
			RightFileEnd:   &LineRange{Line: 42, Offset: 1},
		},
		Comments: []Comment{
			{
				ID:            5,
				Author:        demoUsers[3],
				Content:       "Nit: this error message could be more descriptive. Maybe include the item ID that failed?",
				PublishedDate: demoTime(1),
				CommentType:   "text",
			},
		},
		PublishedDate:   demoTime(1),
		LastUpdatedDate: demoTime(1),
	},
}

var demoIterations = []Iteration{
	{
		ID:              1,
		Description:     "Initial implementation",
		CreatedDate:     demoTime(2),
		SourceRefCommit: CommitRef{CommitID: "abc1234"},
		TargetRefCommit: CommitRef{CommitID: "def5678"},
	},
	{
		ID:              2,
		Description:     "Address review feedback",
		CreatedDate:     demoTime(1),
		SourceRefCommit: CommitRef{CommitID: "ghi9012"},
		TargetRefCommit: CommitRef{CommitID: "def5678"},
	},
}

var demoIterationChanges = []IterationChange{
	{ChangeID: 1, ChangeType: "add", Item: ChangeItem{Path: "/src/handlers/sync.go"}},
	{ChangeID: 2, ChangeType: "edit", Item: ChangeItem{Path: "/src/handlers/sync_test.go"}},
	{ChangeID: 3, ChangeType: "edit", Item: ChangeItem{Path: "/src/middleware/ratelimit.go"}},
	{ChangeID: 4, ChangeType: "edit", Item: ChangeItem{Path: "/src/middleware/ratelimit_test.go"}},
	{ChangeID: 5, ChangeType: "edit", Item: ChangeItem{Path: "/src/models/inventory.go"}},
	{ChangeID: 6, ChangeType: "add", Item: ChangeItem{Path: "/src/models/batch.go"}},
	{ChangeID: 7, ChangeType: "edit", Item: ChangeItem{Path: "/docs/api.md"}},
}

const demoDiff = `--- a/src/handlers/sync.go
+++ b/src/handlers/sync.go
@@ -1,6 +1,8 @@
 package handlers

 import (
+	"context"
+	"fmt"
 	"net/http"
 	"time"

@@ -35,10 +37,24 @@
 }

 // BatchSync handles bulk inventory update requests.
-func (h *InventoryHandler) BatchSync(w http.ResponseWriter, r *http.Request) {
+func (h *InventoryHandler) BatchSync(ctx context.Context, w http.ResponseWriter, r *http.Request) {
 	var req BatchSyncRequest
 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
-		http.Error(w, "invalid request body", http.StatusBadRequest)
+		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
+		return
+	}
+
+	if len(req.Items) == 0 {
+		http.Error(w, "items array must not be empty", http.StatusBadRequest)
+		return
+	}
+
+	results, err := h.service.ProcessBatch(ctx, req.Items)
+	if err != nil {
+		log.Error("batch sync failed",
+			"error", err,
+			"item_count", len(req.Items),
+		)
+		http.Error(w, "internal error processing batch", http.StatusInternalServerError)
 		return
 	}
 `

var demoWorkItems = []WorkItem{
	{
		ID: 4521,
		Fields: WorkItemFields{
			Title:        "Batch sync endpoint returns 500 on duplicate SKUs",
			State:        "Active",
			WorkItemType: "Bug",
			AssignedTo:   demoUsers[0],
			Description:  "When the batch sync payload contains duplicate SKUs, the endpoint returns a 500 instead of a 409 Conflict with details about which items collided.\n\nRepro steps:\n1. POST to /api/v2/inventory/sync with two items sharing SKU 'ABC-123'\n2. Observe 500 Internal Server Error\n\nExpected: 409 with conflict details",
			CreatedDate:  demoTime(3),
			ChangedDate:  demoTime(1),
			AreaPath:     "Contoso\\Backend",
			TeamProject:  "Contoso",
		},
	},
	{
		ID: 4518,
		Fields: WorkItemFields{
			Title:        "Add export to CSV for inventory reports",
			State:        "Active",
			WorkItemType: "User Story",
			AssignedTo:   demoUsers[0],
			Description:  "As a warehouse manager, I want to export my inventory report to CSV so I can share it with suppliers who don't have portal access.",
			CreatedDate:  demoTime(5),
			ChangedDate:  demoTime(2),
			AreaPath:     "Contoso\\Frontend",
			TeamProject:  "Contoso",
		},
	},
	{
		ID: 4515,
		Fields: WorkItemFields{
			Title:        "Write integration tests for OIDC flow",
			State:        "New",
			WorkItemType: "Task",
			AssignedTo:   demoUsers[3],
			Description:  "Cover the full OIDC authorization code flow including token refresh and logout.",
			CreatedDate:  demoTime(4),
			ChangedDate:  demoTime(4),
			AreaPath:     "Contoso\\Auth",
			TeamProject:  "Contoso",
		},
	},
	{
		ID: 4510,
		Fields: WorkItemFields{
			Title:        "Multi-warehouse inventory tracking",
			State:        "Active",
			WorkItemType: "Feature",
			AssignedTo:   demoUsers[4],
			Description:  "Support tracking inventory quantities across multiple warehouse locations with transfer tracking.",
			CreatedDate:  demoTime(14),
			ChangedDate:  demoTime(2),
			AreaPath:     "Contoso\\Backend",
			TeamProject:  "Contoso",
		},
	},
	{
		ID: 4505,
		Fields: WorkItemFields{
			Title:        "Q2 Platform Modernization",
			State:        "Active",
			WorkItemType: "Epic",
			AssignedTo:   demoUsers[1],
			Description:  "Umbrella epic for all Q2 modernization work including React 19 migration, Go 1.24 upgrade, and OIDC integration.",
			CreatedDate:  demoTime(30),
			ChangedDate:  demoTime(1),
			AreaPath:     "Contoso",
			TeamProject:  "Contoso",
		},
	},
	{
		ID: 4498,
		Fields: WorkItemFields{
			Title:        "Dashboard load time exceeds 3s on slow connections",
			State:        "Resolved",
			WorkItemType: "Bug",
			AssignedTo:   demoUsers[2],
			Description:  "Lighthouse performance score dropped below 60. Main culprit is unoptimized bundle size.",
			CreatedDate:  demoTime(10),
			ChangedDate:  demoTime(3),
			AreaPath:     "Contoso\\Frontend",
			TeamProject:  "Contoso",
		},
	},
}

var demoWorkItemComments = []WorkItemComment{
	{
		ID:          1,
		Text:        "Confirmed the 500 on duplicate SKUs. The unique constraint on the DB is throwing an unhandled exception.",
		CreatedBy:   demoUsers[1],
		CreatedDate: demoTime(2),
	},
	{
		ID:          2,
		Text:        "I'll add a pre-check for duplicates before hitting the DB. Should be a quick fix in the BatchSync handler.",
		CreatedBy:   demoUsers[0],
		CreatedDate: demoTime(1),
	},
}

// --- Clienter implementation ---

func (m *MockClient) GetProjectID() (string, error) {
	return "project-1", nil
}

func (m *MockClient) ListRepositories() ([]GitRepository, error) {
	return demoRepos, nil
}

func (m *MockClient) ListBranches(repoName string) ([]GitBranch, error) {
	return []GitBranch{
		{Name: "refs/heads/main"},
		{Name: "refs/heads/develop"},
		{Name: "refs/heads/feature/batch-sync"},
		{Name: "refs/heads/feature/react-19"},
		{Name: "refs/heads/feature/oidc-auth"},
	}, nil
}

func (m *MockClient) ListPullRequests(status string) ([]PullRequest, error) {
	var result []PullRequest
	for _, pr := range demoPRs {
		if status == "all" || pr.Status == status {
			result = append(result, pr)
		}
	}
	return result, nil
}

func (m *MockClient) ListPullRequestsForRepo(repoName, status string) ([]PullRequest, error) {
	var result []PullRequest
	for _, pr := range demoPRs {
		if pr.Repository.Name == repoName && (status == "all" || pr.Status == status) {
			result = append(result, pr)
		}
	}
	return result, nil
}

func (m *MockClient) ListDraftPullRequests() ([]PullRequest, error) {
	var result []PullRequest
	for _, pr := range demoPRs {
		if pr.IsDraft {
			result = append(result, pr)
		}
	}
	return result, nil
}

func (m *MockClient) ListDraftPullRequestsForRepo(repoName string) ([]PullRequest, error) {
	var result []PullRequest
	for _, pr := range demoPRs {
		if pr.IsDraft && pr.Repository.Name == repoName {
			result = append(result, pr)
		}
	}
	return result, nil
}

func (m *MockClient) GetPullRequestByID(prID int) (*PullRequest, error) {
	for _, pr := range demoPRs {
		if pr.PullRequestID == prID {
			return &pr, nil
		}
	}
	return nil, errors.New("pull request not found")
}

func (m *MockClient) GetPullRequest(repoID string, prID int) (*PullRequest, error) {
	return m.GetPullRequestByID(prID)
}

func (m *MockClient) GetPullRequestThreads(repoID string, prID int) ([]Thread, error) {
	if prID == 1847 {
		return demoThreads, nil
	}
	return []Thread{}, nil
}

func (m *MockClient) CreatePullRequest(repoID, title, sourceBranch, targetBranch, description string, isDraft bool) (PullRequest, error) {
	return PullRequest{}, ErrDemoMode
}

func (m *MockClient) MergePullRequest(repoID string, prID int, strategy, commitMsg string, deleteSourceBranch bool) error {
	return ErrDemoMode
}

func (m *MockClient) AbandonPullRequest(repoID string, prID int) error {
	return ErrDemoMode
}

func (m *MockClient) ToggleDraft(repoID string, prID int, isDraft bool) error {
	return ErrDemoMode
}

func (m *MockClient) CreateThread(repoID string, prID int, content string, threadCtx *ThreadContext) error {
	return ErrDemoMode
}

func (m *MockClient) ReplyToThread(repoID string, prID, threadID int, content string) error {
	return ErrDemoMode
}

func (m *MockClient) UpdateThreadStatus(repoID string, prID, threadID int, status string) error {
	return ErrDemoMode
}

func (m *MockClient) SetVote(repoID string, prID int, reviewerID string, vote int) error {
	return ErrDemoMode
}

func (m *MockClient) GetCurrentUserID() (string, error) {
	return "user-1", nil
}

func (m *MockClient) GetPullRequestIterations(repoID string, prID int) ([]Iteration, error) {
	if prID == 1847 {
		return demoIterations, nil
	}
	return []Iteration{{ID: 1, CreatedDate: demoTime(1), SourceRefCommit: CommitRef{CommitID: "abc"}, TargetRefCommit: CommitRef{CommitID: "def"}}}, nil
}

func (m *MockClient) GetPullRequestIterationChanges(repoID string, prID, iterationID int) ([]IterationChange, error) {
	if prID == 1847 {
		return demoIterationChanges, nil
	}
	return []IterationChange{}, nil
}

func (m *MockClient) GetFileContentAtCommit(repoID, filePath, commitID string) (string, error) {
	return "// demo file content", nil
}

func (m *MockClient) BuildUnifiedDiff(repoID string, change IterationChange, oldCommitID, newCommitID string) (string, error) {
	return demoDiff, nil
}

func (m *MockClient) ListWorkItems(types []string, assignedTo, areaPath string, activeOnly bool) ([]WorkItem, error) {
	var result []WorkItem
	for _, wi := range demoWorkItems {
		if activeOnly && (wi.Fields.State == "Closed" || wi.Fields.State == "Done" || wi.Fields.State == "Resolved") {
			continue
		}
		if assignedTo == "@me" {
			if wi.Fields.AssignedTo.ID != "user-1" {
				continue
			}
		} else if assignedTo != "" && wi.Fields.AssignedTo.ID != assignedTo {
			continue
		}
		result = append(result, wi)
	}
	return result, nil
}

func (m *MockClient) GetWorkItem(id int) (WorkItem, error) {
	for _, wi := range demoWorkItems {
		if wi.ID == id {
			return wi, nil
		}
	}
	return WorkItem{}, errors.New("work item not found")
}

func (m *MockClient) GetWorkItemComments(id int) ([]WorkItemComment, error) {
	if id == 4521 {
		return demoWorkItemComments, nil
	}
	return []WorkItemComment{}, nil
}

func (m *MockClient) UpdateWorkItemState(id int, state string) error {
	return ErrDemoMode
}

func (m *MockClient) AddWorkItemComment(id int, text string) error {
	return ErrDemoMode
}

func (m *MockClient) GetWorkItemTypeStates(workItemType string) ([]string, error) {
	states, ok := WorkItemStates[workItemType]
	if !ok {
		return []string{"New", "Active", "Closed"}, nil
	}
	return states, nil
}

func (m *MockClient) LinkWorkItemToPR(workItemID int, prArtifactURL string) error {
	return ErrDemoMode
}
